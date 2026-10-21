package hub

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Playdates: shared AR sessions. One player starts a playdate and gets a
// short code; others join with it. Each player's game regularly sends what
// its pet is like (a card of fixed fields) and the pet's position relative
// to the playdate's center, a real spot every player marks in their own AR
// view, and gets back everyone else's. When one pet plays with another,
// the player's game reports how it went as numbers, and the hub writes
// what every member reads.
//
// Nothing a player types reaches another player: pets are shown as their
// owner's handle and their species ("Blue Otter 42's dog"), cards hold
// only values from fixed lists and numbers, and the hub writes every
// event's text. Playdates are kept in memory only.

// Playdate limits
const (
	MaxPlaydateMembers = 6
	maxPlaydates       = 500
	maxPlaydateEvents  = 50
	maxPlaydateOffset  = 5.0                    // Meters from the center
	positionStep       = 0.1                    // Positions are rounded to 10 cm
	maxPlaydateMove    = 0.5                    // Meters a pet moves at most per change
	changeInterval     = 900 * time.Millisecond // A member's pet and position change at most about once a second
	levelStep          = 0.1                    // Sociability, play quality and bond are rounded to tenths
	memberTimeout      = 30 * time.Second       // Members who stop updating leave
	playdateCodeLength = 6
)

// playdateAlphabet has no letters or digits that look alike
const playdateAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// Playdate errors
var (
	ErrNoPlaydate      = errors.New("no playdate with that code")
	ErrPlaydateFull    = errors.New("that playdate is full")
	ErrNotInPlaydate   = errors.New("you aren't in that playdate")
	ErrTooManyPlaydate = errors.New("the hub has too many playdates right now; try again later")
	ErrBadCard         = errors.New("invalid pet card")
)

// Offset is a position relative to the playdate's center, in meters
type Offset struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// clamp keeps an offset on the floor within reach of the center and
// within maxPlaydateMove of from, rounded to 10 cm, and replaces values
// that aren't numbers with 0. Positions carry little more than where the
// pet is, so they can't be used to pass messages quickly.
func (o Offset) clamp(from Offset) Offset {
	c := func(v, from float64) float64 {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			v = 0
		}
		v = min(max(v, from-maxPlaydateMove, -maxPlaydateOffset), from+maxPlaydateMove, maxPlaydateOffset)
		return math.Round(v/positionStep) * positionStep
	}
	return Offset{X: c(o.X, from.X), Z: c(o.Z, from.Z)}
}

// roundLevel rounds a level to tenths, within lowest and 1
func roundLevel(v, lowest float64) float64 {
	return math.Round(min(max(v, lowest), 1)/levelStep) * levelStep
}

// CardRequest is what a player's game says about its pet
type CardRequest struct {
	PetKey      string            `json:"pet_key"` // The pet's hub key (PetKey)
	Species     string            `json:"species"`
	Young       bool              `json:"young"`
	Sociability float64           `json:"sociability"`
	Unavailable string            `json:"unavailable,omitempty"` // A core Unavailable code, or ""
	Look        string            `json:"look"`                  // One of core.SpriteStates
	Outfit      map[string]string `json:"outfit,omitempty"`      // Cosmetic IDs from the game's list, by slot
	Breed       string            `json:"breed,omitempty"`       // A breed ID from the game's list
}

// Card is a pet at a playdate as every member sees it. The hub fills in
// its ID (stable for the pet, but meaningless elsewhere) and its name.
type Card struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"` // "<handle>'s <species>"
	Species     string  `json:"species"`
	Young       bool    `json:"young"`
	Sociability float64 `json:"sociability"`
	Unavailable string  `json:"unavailable,omitempty"`
	Look        string  `json:"look"`
	// What the pet wears: a cosmetic ID from the game's list per slot
	Outfit map[string]string `json:"outfit,omitempty"`
	// Its breed: an ID from the game's list
	Breed string `json:"breed,omitempty"`
}

// unavailableCodes are the reasons a card can give
var unavailableCodes = []string{"", core.UnavailableGone, core.UnavailableHibernating, core.UnavailableExploring,
	core.UnavailableAsleep}

// card checks a card request and makes the card for an account's pet
func makeCard(accountID, handle string, req CardRequest) (Card, error) {
	species := core.Species(req.Species)
	if !keyPattern.MatchString(req.PetKey) || !species.Known() || !contains(core.SpriteStates, req.Look) ||
		!contains(unavailableCodes, req.Unavailable) || math.IsNaN(req.Sociability) ||
		len(req.Outfit) > len(core.CosmeticSlots) || !core.ValidOutfit(req.Outfit) {
		return Card{}, ErrBadCard
	}
	if b, ok := core.FindBreed(req.Breed); req.Breed != "" && (!ok || b.Species != species) {
		return Card{}, ErrBadCard
	}
	sum := sha256.Sum256([]byte("playmate:" + accountID + ":" + req.PetKey))
	return Card{ID: "playmate_" + hex.EncodeToString(sum[:12]), Name: handle + "'s " + strings.ToLower(species.Def().Name),
		Species: req.Species, Young: req.Young, Sociability: roundLevel(req.Sociability, 0),
		Unavailable: req.Unavailable, Look: req.Look, Outfit: req.Outfit, Breed: req.Breed}, nil
}

// PlaydateMember is a player at a playdate
type PlaydateMember struct {
	Handle   string    `json:"handle"`
	Slot     int       `json:"slot"` // Place around the center, from 0
	Card     Card      `json:"card"`
	Position Offset    `json:"position"`
	Updated  time.Time `json:"updated"`
	changed  time.Time // When its card or position last changed
}

// PlayReport is a play between two pets, as numbers only
type PlayReport struct {
	To      string  `json:"to"`      // The other player's handle
	Quality float64 `json:"quality"` // 0.0 to 1.0
	Bond    float64 `json:"bond"`    // -1.0 to 1.0
}

// PlaydateEvent is something that happened at a playdate, written by the
// hub
type PlaydateEvent struct {
	Seq      int64     `json:"seq"`
	Kind     string    `json:"kind"` // "play"
	From     string    `json:"from"` // Handles
	To       string    `json:"to"`
	FromCard Card      `json:"from_card"` // The pet that started playing
	ToPet    string    `json:"to_pet"`    // The card ID of the pet it played with
	Quality  float64   `json:"quality"`
	Bond     float64   `json:"bond"`
	Text     string    `json:"text"`
	Created  time.Time `json:"created"`
}

// PlaydateState is a playdate as one member sees it
type PlaydateState struct {
	Code    string           `json:"code"`
	Host    string           `json:"host"`
	You     string           `json:"you"`
	Members []PlaydateMember `json:"members"`
	Events  []PlaydateEvent  `json:"events"` // Those after the sequence number asked for
	Last    int64            `json:"last"`   // Sequence number of the latest event
}

// playdate is a playdate on the hub
type playdate struct {
	code    string
	host    string                     // Handle
	members map[string]*PlaydateMember // By account ID
	events  []PlaydateEvent
	seq     int64
}

// newPlaydateCode returns a random playdate code
func newPlaydateCode() (string, error) {
	var b strings.Builder
	for i := 0; i < playdateCodeLength; i++ {
		n, err := randomIndex(len(playdateAlphabet))
		if err != nil {
			return "", err
		}
		b.WriteByte(playdateAlphabet[n])
	}
	return b.String(), nil
}

// NormalizePlaydateCode returns a code as the hub writes it
func NormalizePlaydateCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// expirePlaydatesLocked drops members who stopped updating and playdates
// with no one left. Caller must hold h.mu.
func (h *Hub) expirePlaydatesLocked() {
	now := h.now()
	for code, pd := range h.playdates {
		for id, m := range pd.members {
			if now.Sub(m.Updated) > memberTimeout {
				delete(pd.members, id)
			}
		}
		if len(pd.members) == 0 {
			delete(h.playdates, code)
		}
	}
}

// leaveAllLocked takes a player out of every playdate. Caller must hold
// h.mu.
func (h *Hub) leaveAllLocked(id string) {
	for code, pd := range h.playdates {
		delete(pd.members, id)
		if len(pd.members) == 0 {
			delete(h.playdates, code)
		}
	}
}

// joinLocked adds a player to a playdate in the first free slot. Caller
// must hold h.mu.
func (pd *playdate) joinLocked(id, handle string, card Card, now time.Time) {
	used := make(map[int]bool)
	for _, m := range pd.members {
		used[m.Slot] = true
	}
	slot := 0
	for used[slot] {
		slot++
	}
	pd.members[id] = &PlaydateMember{Handle: handle, Slot: slot, Card: card, Updated: now}
}

// stateLocked returns the playdate as a member sees it, with the events
// after seq. Caller must hold h.mu.
func (pd *playdate) stateLocked(handle string, after int64) PlaydateState {
	state := PlaydateState{Code: pd.code, Host: pd.host, You: handle, Members: []PlaydateMember{},
		Events: []PlaydateEvent{}, Last: pd.seq}
	for _, m := range pd.members {
		state.Members = append(state.Members, *m)
	}
	sort.Slice(state.Members, func(i, j int) bool { return state.Members[i].Slot < state.Members[j].Slot })
	for _, e := range pd.events {
		if e.Seq > after {
			state.Events = append(state.Events, e)
		}
	}
	return state
}

// StartPlaydate starts a playdate with the player's pet and returns it.
// The player leaves any other playdate.
func (h *Hub) StartPlaydate(id string, req CardRequest) (PlaydateState, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	me, err := h.accountLocked(id)
	if err != nil {
		return PlaydateState{}, err
	}
	card, err := makeCard(id, me.Handle, req)
	if err != nil {
		return PlaydateState{}, err
	}
	h.expirePlaydatesLocked()
	h.leaveAllLocked(id)
	if len(h.playdates) >= maxPlaydates {
		return PlaydateState{}, ErrTooManyPlaydate
	}
	var code string
	for code == "" || h.playdates[code] != nil {
		if code, err = newPlaydateCode(); err != nil {
			return PlaydateState{}, err
		}
	}
	pd := &playdate{code: code, host: me.Handle, members: make(map[string]*PlaydateMember)}
	pd.joinLocked(id, me.Handle, card, h.now())
	h.playdates[code] = pd
	return pd.stateLocked(me.Handle, 0), nil
}

// JoinPlaydate adds the player's pet to a playdate. The player leaves any
// other playdate.
func (h *Hub) JoinPlaydate(id, code string, req CardRequest) (PlaydateState, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	me, err := h.accountLocked(id)
	if err != nil {
		return PlaydateState{}, err
	}
	card, err := makeCard(id, me.Handle, req)
	if err != nil {
		return PlaydateState{}, err
	}
	h.expirePlaydatesLocked()
	pd := h.playdates[NormalizePlaydateCode(code)]
	if pd == nil {
		return PlaydateState{}, ErrNoPlaydate
	}
	if pd.members[id] == nil {
		if len(pd.members) >= MaxPlaydateMembers {
			return PlaydateState{}, ErrPlaydateFull
		}
		h.leaveAllLocked(id)
		pd.joinLocked(id, me.Handle, card, h.now())
	}
	return pd.stateLocked(me.Handle, 0), nil
}

// memberLocked returns a playdate and the player's place in it. Caller
// must hold h.mu.
func (h *Hub) memberLocked(id, code string) (*playdate, *PlaydateMember, error) {
	if _, err := h.accountLocked(id); err != nil {
		return nil, nil, err
	}
	h.expirePlaydatesLocked()
	pd := h.playdates[NormalizePlaydateCode(code)]
	if pd == nil {
		return nil, nil, ErrNoPlaydate
	}
	m := pd.members[id]
	if m == nil {
		return nil, nil, ErrNotInPlaydate
	}
	return pd, m, nil
}

// UpdatePlaydate records the player's pet (if req isn't nil) and its
// position, and returns the playdate with the events after seq
func (h *Hub) UpdatePlaydate(id, code string, req *CardRequest, position Offset, after int64) (PlaydateState, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	pd, m, err := h.memberLocked(id, code)
	if err != nil {
		return PlaydateState{}, err
	}
	var card *Card
	if req != nil {
		c, err := makeCard(id, m.Handle, *req)
		if err != nil {
			return PlaydateState{}, err
		}
		card = &c
	}
	if now := h.now(); now.Sub(m.changed) >= changeInterval {
		if card != nil {
			m.Card = *card
		}
		m.Position, m.changed = position.clamp(m.Position), now
	}
	m.Updated = h.now()
	return pd.stateLocked(m.Handle, after), nil
}

// ReportPlay records that the player's pet played with another member's
// pet. The hub writes the event's text from the two pets' names and how
// it went.
func (h *Hub) ReportPlay(id, code string, report PlayReport) (PlaydateEvent, error) {
	if math.IsNaN(report.Quality) || math.IsNaN(report.Bond) {
		return PlaydateEvent{}, errors.New("invalid play")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	pd, m, err := h.memberLocked(id, code)
	if err != nil {
		return PlaydateEvent{}, err
	}
	var to *PlaydateMember
	for other, member := range pd.members {
		if member.Handle == report.To && other != id {
			to = member
		}
	}
	if to == nil {
		return PlaydateEvent{}, errors.New("that player isn't at the playdate")
	}
	quality := roundLevel(report.Quality, 0)
	pd.seq++
	e := PlaydateEvent{Seq: pd.seq, Kind: "play", From: m.Handle, To: to.Handle, FromCard: m.Card, ToPet: to.Card.ID,
		Quality: quality, Bond: roundLevel(report.Bond, -1), Created: h.now(),
		Text: core.MeetingText(m.Card.Name, core.Species(m.Card.Species), to.Card.Name, quality)}
	pd.events = append(pd.events, e)
	if over := len(pd.events) - maxPlaydateEvents; over > 0 {
		pd.events = append(pd.events[:0:0], pd.events[over:]...)
	}
	m.Updated = h.now()
	return e, nil
}

// LeavePlaydate takes the player out of a playdate
func (h *Hub) LeavePlaydate(id, code string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	pd, _, err := h.memberLocked(id, code)
	if err != nil {
		return err
	}
	delete(pd.members, id)
	if len(pd.members) == 0 {
		delete(h.playdates, pd.code)
	}
	return nil
}
