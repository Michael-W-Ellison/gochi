package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
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
// server passes it to the hub with what the pet is like and gets back the
// other pets and what happened. When a player has their pet play with
// another, this server works out how it goes and reports it to the hub as
// numbers; the other player's server records it for their pet.
//
// Nothing about the pet that a player typed goes to the hub: other players
// see "<handle>'s <species>", and the hub writes every event's text.

// playdateTimeout is how long a playdate request to the hub may take
const playdateTimeout = 10 * time.Second

// maxPlaydateLog is how many things that happened the page is shown
const maxPlaydateLog = 8

// Playmate is a pet at the playdate, as the page sees it
type Playmate struct {
	Handle   string                   `json:"handle"` // Its owner, as the hub calls them
	Slot     int                      `json:"slot"`
	You      bool                     `json:"you"`
	Name     string                   `json:"name"` // As the hub names it, such as "Blue Otter 42's dog"
	Species  string                   `json:"species"`
	Young    bool                     `json:"young"`
	Look     string                   `json:"look"`   // One of core.SpriteStates
	Outfit   map[string]core.Cosmetic `json:"outfit"` // What it wears, by slot
	Breed    string                   `json:"breed"`  // Its breed ID; drawn in the breed's typical look
	Typical  core.PetLook             `json:"typical"`
	Position hub.Offset               `json:"position"`
	Relation string                   `json:"relation"` // How your pet gets along with it, or ""
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

// activeCard returns what the hub is told about the active pet: its hub
// key, species, whether it is young, how sociable it is, why it can't
// play (as a code) and how it looks (one of core.SpriteStates)
func (s *Server) activeCard(look string) (hub.CardRequest, types.PetID, error) {
	s.mu.RLock()
	pet := s.pet
	s.mu.RUnlock()
	if pet == nil {
		return hub.CardRequest{}, "", errors.New("you need a pet to bring to a playdate")
	}
	if !slices.Contains(core.SpriteStates, look) {
		look = "idle"
	}
	card := pet.PlaymateCard()
	breed, _ := pet.GetLook()
	return hub.CardRequest{PetKey: hub.PetKey(string(card.ID)), Species: string(card.Species), Young: card.Young,
			Sociability: card.Sociability, Unavailable: card.Unavailable, Look: look, Outfit: pet.OutfitIDs(), Breed: breed},
		card.ID, nil
}

// playmateCard turns a hub card into the game's
func playmateCard(c hub.Card) core.PlaymateCard {
	return core.PlaymateCard{ID: types.PetID(c.ID), Name: c.Name, Species: core.Species(c.Species), Young: c.Young,
		Sociability: c.Sociability, Unavailable: c.Unavailable}
}

// playdateStatusLocked describes the playdate for the page. Caller must
// hold s.playdates.mu.
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
		mate := Playmate{Handle: m.Handle, Slot: m.Slot, You: m.Handle == pd.you, Name: m.Card.Name,
			Species: m.Card.Species, Young: m.Card.Young, Look: m.Card.Look, Position: m.Position,
			Outfit: map[string]core.Cosmetic{}, Breed: m.Card.Breed, Typical: core.TypicalLook(m.Card.Breed)}
		for slot, id := range m.Card.Outfit {
			if c, ok := core.FindCosmetic(id); ok && c.Slot == slot && (c.Holiday == "" || core.HolidayOn(c.Holiday)) {
				mate.Outfit[slot] = c
			}
		}
		if pet != nil && !mate.You {
			mate.Relation = pet.RelationTo(types.PetID(m.Card.ID))
		}
		status.Members = append(status.Members, mate)
	}
	return status
}

// myCardID returns the hub's ID for this player's pet
func (pd *playdateSession) myCardID() string {
	for _, m := range pd.members {
		if m.Handle == pd.you {
			return m.Card.ID
		}
	}
	return ""
}

// applyPlaydateLocked takes in the hub's playdate: its members, and events
// since the last seen, recording plays with this player's pet. Caller must
// hold s.playdates.mu.
func (s *Server) applyPlaydateLocked(state hub.PlaydateState) {
	pd := s.playdates.session
	pd.host, pd.you, pd.members = state.Host, state.You, state.Members
	mine := pd.myCardID()
	for _, e := range state.Events {
		if e.Seq <= pd.last {
			continue
		}
		pd.last = e.Seq
		if e.Text != "" {
			pd.log = append(pd.log, e.Text)
		}
		if e.Kind != "play" || e.To != pd.you || e.From == pd.you || e.ToPet != mine {
			continue
		}
		from := playmateCard(e.FromCard)
		s.mu.Lock()
		if s.pet != nil && s.pet.ID == pd.petID {
			if relation := s.pet.PlayedWithPlaymate(from, e.Quality, e.Bond); relation != "" {
				pd.log = append(pd.log, s.pet.Name+" and "+from.Name+" are "+relation+".")
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
// "code") or "leave", "look"}
func (s *Server) handlePlaydate(w http.ResponseWriter, r *http.Request) {
	s.playdates.mu.Lock()
	defer s.playdates.mu.Unlock()
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.playdateStatusLocked()})
		return
	}
	var req struct {
		Action    string `json:"action"`
		Code      string `json:"code"`
		Look      string `json:"look"`
		ParentPIN string `json:"parent_pin"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if req.Action != "leave" && !s.parentAllows(w, req.ParentPIN) {
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

	card, petID, err := s.activeCard(req.Look)
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
// {"position", "look"}: sends the pet's position relative to the
// playdate's center and how it looks, and returns the playdate
func (s *Server) handlePlaydateUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Position hub.Offset `json:"position"`
		Look     string     `json:"look"`
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
	card, petID, err := s.activeCard(req.Look)
	if err != nil || petID != pd.petID {
		// The player switched pets: the playdate was for the other one
		client.LeavePlaydate(r.Context(), pd.code)
		s.playdates.session = nil
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "You left the playdate when you changed pets"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), playdateTimeout)
	defer cancel()
	state, err := client.UpdatePlaydate(ctx, pd.code, &card, req.Position, pd.last)
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

// handlePlaydatePlay handles POST /api/playdate/play with {"handle"}: the
// player's pet plays with that player's pet
func (s *Server) handlePlaydatePlay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Handle string `json:"handle"`
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
	var other *core.PlaymateCard
	for _, m := range pd.members {
		if m.Handle == req.Handle && m.Handle != pd.you {
			card := playmateCard(m.Card)
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
	meeting, bond, err := s.pet.PlayWithPlaymate(*other)
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

	// Tell the other player's game, so their pet remembers it too: only
	// how it went, as numbers
	ctx, cancel := context.WithTimeout(r.Context(), playdateTimeout)
	defer cancel()
	if _, err := client.ReportPlay(ctx, pd.code, hub.PlayReport{To: req.Handle, Quality: meeting.Quality,
		Bond: bond}); err != nil {
		log.Printf("Telling the playdate about a meeting: %v", err)
		data["note"] = strings.TrimSpace(meeting.Note + " The other player's game couldn't be told about it.")
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}
