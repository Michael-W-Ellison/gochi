package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/hub"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// AR playdates: players connected to the same cloud hub bring their active
// pets to one place. One player starts a playdate and gets a short code;
// the others join with it. In AR, each player marks the same real spot as
// the playdate's center, and every pet is shown around it. The page sends
// its pet's position relative to the center about once a second; the
// server passes it to the hub with the pet's card and gets back the other
// pets and what happened. When a player has their pet play with another,
// this server works out how it goes and the other player's server records
// it for their pet.

// playdateTimeout is how long a playdate request to the hub may take
const playdateTimeout = 10 * time.Second

// maxPlaydateLog is how many things that happened the page is shown
const maxPlaydateLog = 8

// playdateCard is what the hub passes on to the other players
type playdateCard struct {
	core.PlaymateCard
	Sprite string `json:"sprite"` // How the pet looks now, as the player's page draws it
}

// playEvent is the data of a "play" playdate event
type playEvent struct {
	From    core.PlaymateCard `json:"from"`   // The pet that started playing
	ToPet   types.PetID       `json:"to_pet"` // The pet it played with
	Quality float64           `json:"quality"`
	Bond    float64           `json:"bond"`
}

// Playmate is a pet at the playdate, as the page sees it
type Playmate struct {
	Owner    string            `json:"owner"`
	Slot     int               `json:"slot"`
	You      bool              `json:"you"`
	Pet      core.PlaymateCard `json:"pet"`
	Sprite   string            `json:"sprite"`
	Position hub.Offset        `json:"position"`
	Relation string            `json:"relation"` // How your pet gets along with it, or ""
}

// playdateSession is the playdate this server's player is at
type playdateSession struct {
	code    string
	host    string
	you     string
	petID   types.PetID // The pet the player brought
	last    int64       // The latest event seen
	members []hub.PlaydateMember
	log     []string // What happened, oldest first
}

// PlaydateStatus is what the page is told about the playdate
type PlaydateStatus struct {
	Active  bool       `json:"active"`
	Code    string     `json:"code,omitempty"`
	Host    string     `json:"host,omitempty"`
	You     string     `json:"you,omitempty"`
	Members []Playmate `json:"members"`
	Log     []string   `json:"log"`
}

// playdateState holds the server's playdate
type playdateState struct {
	mu      sync.Mutex // Held for whole playdate requests; acquire before Server.mu and cloud.mu
	session *playdateSession
}

// hubClient returns a client for the connected hub
func (s *Server) hubClient() (*hub.Client, error) {
	c := s.cloud
	if c == nil {
		return nil, errNotConnected
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connectedLocked() {
		return nil, errNotConnected
	}
	return c.clientLocked(), nil
}

// activeCard returns the active pet's card, with how it looks
func (s *Server) activeCard(sprite string) (json.RawMessage, types.PetID, error) {
	s.mu.RLock()
	pet := s.pet
	s.mu.RUnlock()
	if pet == nil {
		return nil, "", errors.New("you need a pet to bring to a playdate")
	}
	card := playdateCard{PlaymateCard: pet.PlaymateCard(), Sprite: clipText(sprite, 16)}
	data, err := json.Marshal(card)
	return data, card.ID, err
}

// clipText shortens text to at most n characters
func clipText(text string, n int) string {
	if r := []rune(text); len(r) > n {
		return string(r[:n])
	}
	return text
}

// playdateStatusLocked describes the playdate for the page. Caller must hold
// s.playdates.mu.
func (s *Server) playdateStatusLocked() PlaydateStatus {
	pd := s.playdates.session
	if pd == nil {
		return PlaydateStatus{Members: []Playmate{}, Log: []string{}}
	}
	status := PlaydateStatus{Active: true, Code: pd.code, Host: pd.host, You: pd.you, Members: []Playmate{},
		Log: append([]string{}, pd.log...)}
	s.mu.RLock()
	pet := s.pet
	s.mu.RUnlock()
	for _, m := range pd.members {
		var card playdateCard
		if err := json.Unmarshal(m.Card, &card); err != nil || !card.Valid() {
			continue
		}
		mate := Playmate{Owner: m.Owner, Slot: m.Slot, You: strings.EqualFold(m.Owner, pd.you), Pet: card.PlaymateCard,
			Sprite: card.Sprite, Position: m.Position}
		if pet != nil && !mate.You {
			mate.Relation = pet.RelationTo(card.ID)
		}
		status.Members = append(status.Members, mate)
	}
	return status
}

// applyPlaydateLocked takes in the hub's playdate: its members, and events
// since the last seen, recording plays with this player's pet. Caller must
// hold s.playdates.mu.
func (s *Server) applyPlaydateLocked(state hub.PlaydateState) {
	pd := s.playdates.session
	pd.host, pd.you, pd.members = state.Host, state.You, state.Members
	for _, e := range state.Events {
		if e.Seq <= pd.last {
			continue
		}
		pd.last = e.Seq
		if e.Text != "" {
			pd.log = append(pd.log, clipText(e.Text, 200))
		}
		if e.Kind != "play" || !strings.EqualFold(e.To, pd.you) || strings.EqualFold(e.From, pd.you) {
			continue
		}
		var play playEvent
		if err := json.Unmarshal(e.Data, &play); err != nil {
			continue
		}
		s.mu.Lock()
		if s.pet != nil && s.pet.ID == play.ToPet && play.ToPet == pd.petID {
			if relation := s.pet.PlayedWithPlaymate(play.From, play.Quality, play.Bond); relation != "" {
				pd.log = append(pd.log, s.pet.Name+" and "+play.From.Name+" are "+relation+".")
			}
		}
		s.mu.Unlock()
	}
	pd.last = max(pd.last, state.Last)
	if over := len(pd.log) - maxPlaydateLog; over > 0 {
		pd.log = append(pd.log[:0:0], pd.log[over:]...)
	}
}

// writePlaydateError writes an error from a playdate request
func writePlaydateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, hub.ErrNoPlaydate), errors.Is(err, hub.ErrPlaydateFull), errors.Is(err, hub.ErrNotInPlaydate):
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: strings.ToUpper(err.Error()[:1]) + err.Error()[1:] + "."})
	default:
		writeCloudError(w, err)
	}
}

// handlePlaydate handles GET /api/playdate (the playdate, as last
// updated) and POST /api/playdate with {"action": "start", "join" (with
// "code") or "leave", "sprite"}
func (s *Server) handlePlaydate(w http.ResponseWriter, r *http.Request) {
	s.playdates.mu.Lock()
	defer s.playdates.mu.Unlock()
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.playdateStatusLocked()})
		return
	}
	var req struct {
		Action string `json:"action"`
		Code   string `json:"code"`
		Sprite string `json:"sprite"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	client, err := s.hubClient()
	if err != nil {
		writeCloudError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), playdateTimeout)
	defer cancel()

	if req.Action == "leave" {
		if pd := s.playdates.session; pd != nil {
			if err := client.LeavePlaydate(ctx, pd.code); err != nil && !errors.Is(err, hub.ErrNoPlaydate) &&
				!errors.Is(err, hub.ErrNotInPlaydate) {
				log.Printf("Leaving the playdate: %v", err)
			}
		}
		s.playdates.session = nil
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.playdateStatusLocked()})
		return
	}

	card, petID, err := s.activeCard(req.Sprite)
	if err != nil {
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: err.Error()})
		return
	}
	var state hub.PlaydateState
	switch req.Action {
	case "start":
		state, err = client.StartPlaydate(ctx, card)
	case "join":
		code := hub.NormalizePlaydateCode(req.Code)
		if code == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Enter the playdate's code"})
			return
		}
		state, err = client.JoinPlaydate(ctx, code, card)
	default:
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown action"})
		return
	}
	if err != nil {
		writePlaydateError(w, err)
		return
	}
	// Events from before joining don't concern this player
	s.playdates.session = &playdateSession{code: state.Code, petID: petID, last: state.Last}
	s.applyPlaydateLocked(state)
	s.mu.Lock()
	s.markActiveLocked(time.Now())
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.playdateStatusLocked()})
}

// handlePlaydateUpdate handles POST /api/playdate/update with
// {"position", "sprite"}: sends the pet's position relative to the
// playdate's center and how it looks, and returns the playdate
func (s *Server) handlePlaydateUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Position hub.Offset `json:"position"`
		Sprite   string     `json:"sprite"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	s.playdates.mu.Lock()
	defer s.playdates.mu.Unlock()
	pd := s.playdates.session
	if pd == nil {
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "You aren't at a playdate"})
		return
	}
	client, err := s.hubClient()
	if err != nil {
		writeCloudError(w, err)
		return
	}
	card, petID, err := s.activeCard(req.Sprite)
	if err != nil || petID != pd.petID {
		// The player switched pets: the playdate was for the other one
		client.LeavePlaydate(r.Context(), pd.code)
		s.playdates.session = nil
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "You left the playdate when you changed pets"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), playdateTimeout)
	defer cancel()
	state, err := client.UpdatePlaydate(ctx, pd.code, card, req.Position, pd.last)
	if err != nil {
		if errors.Is(err, hub.ErrNoPlaydate) || errors.Is(err, hub.ErrNotInPlaydate) {
			s.playdates.session = nil
		}
		writePlaydateError(w, err)
		return
	}
	s.applyPlaydateLocked(state)
	s.mu.Lock()
	s.markActiveLocked(time.Now())
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.playdateStatusLocked()})
}

// handlePlaydatePlay handles POST /api/playdate/play with {"owner"}: the
// player's pet plays with that player's pet
func (s *Server) handlePlaydatePlay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Owner string `json:"owner"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	s.playdates.mu.Lock()
	defer s.playdates.mu.Unlock()
	pd := s.playdates.session
	if pd == nil {
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "You aren't at a playdate"})
		return
	}
	var other *playdateCard
	for _, m := range pd.members {
		var card playdateCard
		if strings.EqualFold(m.Owner, req.Owner) && !strings.EqualFold(m.Owner, pd.you) && json.Unmarshal(m.Card, &card) == nil {
			other = &card
		}
	}
	if other == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "That player isn't at the playdate"})
		return
	}
	client, err := s.hubClient()
	if err != nil {
		writeCloudError(w, err)
		return
	}

	s.mu.Lock()
	if s.pet == nil || s.pet.ID != pd.petID {
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "The pet you brought isn't your active pet"})
		return
	}
	s.markActiveLocked(time.Now())
	mine := s.pet.PlaymateCard()
	meeting, bond, err := s.pet.PlayWithPlaymate(other.PlaymateCard)
	if err != nil {
		s.mu.Unlock()
		writeActivityError(w, mine.Name, err)
		return
	}
	data := s.interactionData(meeting.InteractionResult)
	s.mu.Unlock()
	data["text"] = meeting.Text
	data["quality"] = meeting.Quality
	data["relationship"] = meeting.Relationship

	// Tell the other player's game, so their pet remembers it too
	event, _ := json.Marshal(playEvent{From: mine, ToPet: other.ID, Quality: meeting.Quality, Bond: bond})
	ctx, cancel := context.WithTimeout(r.Context(), playdateTimeout)
	defer cancel()
	if _, err := client.PostPlaydateEvent(ctx, pd.code, hub.PlaydateEvent{Kind: "play", To: req.Owner,
		Text: meeting.Text, Data: event}); err != nil {
		log.Printf("Telling the playdate about a meeting: %v", err)
		data["note"] = strings.TrimSpace(meeting.Note + " The other player's game couldn't be told about it.")
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}
