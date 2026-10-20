package hub

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"
)

// Playdates: shared AR sessions. One player starts a playdate and gets a
// short code; others join with it. Each player's game regularly sends its
// pet's card (which the hub passes on without reading) and the pet's
// position relative to the playdate's center, a real spot every player
// marks in their own AR view, and gets back everyone else's. Players post
// events, such as one pet playing with another, which every member sees.
//
// Playdates are kept in memory only: they last as long as players keep
// taking part.

// Playdate limits
const (
	MaxPlaydateMembers = 6
	maxPlaydates       = 500
	maxCardBytes       = 2048
	maxEventText       = 200
	maxPlaydateEvents  = 50
	maxPlaydateOffset  = 10.0             // Meters from the center
	memberTimeout      = 30 * time.Second // Members who stop updating leave
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
)

// Offset is a position relative to the playdate's center, in meters
type Offset struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// clamp keeps an offset within reach of the center, and replaces values
// that aren't numbers with 0
func (o Offset) clamp() Offset {
	c := func(v float64) float64 {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return min(max(v, -maxPlaydateOffset), maxPlaydateOffset)
	}
	return Offset{c(o.X), c(o.Y), c(o.Z)}
}

// PlaydateMember is a player at a playdate
type PlaydateMember struct {
	Owner    string          `json:"owner"`
	Slot     int             `json:"slot"` // Place around the center, from 0
	Card     json.RawMessage `json:"card"` // The pet, as the player's game describes it
	Position Offset          `json:"position"`
	Updated  time.Time       `json:"updated"`
}

// PlaydateEvent is something that happened at a playdate
type PlaydateEvent struct {
	Seq     int64           `json:"seq"`
	Kind    string          `json:"kind"` // Such as "play"
	From    string          `json:"from"` // The player who posted it
	To      string          `json:"to,omitempty"`
	Text    string          `json:"text"`
	Data    json.RawMessage `json:"data,omitempty"` // Passed on as posted
	Created time.Time       `json:"created"`
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
	host    string
	members map[string]*PlaydateMember // By lowercase username
	events  []PlaydateEvent
	seq     int64
}

// newPlaydateCode returns a random playdate code
func newPlaydateCode() (string, error) {
	var b strings.Builder
	for i := 0; i < playdateCodeLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(playdateAlphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(playdateAlphabet[n.Int64()])
	}
	return b.String(), nil
}

// NormalizePlaydateCode returns a code as the hub writes it
func NormalizePlaydateCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// checkCard checks a pet card's size and that it is JSON
func checkCard(card json.RawMessage) error {
	if len(card) == 0 || len(card) > maxCardBytes || !json.Valid(card) {
		return errors.New("invalid pet card")
	}
	return nil
}

// expirePlaydatesLocked drops members who stopped updating and playdates
// with no one left. Caller must hold h.mu.
func (h *Hub) expirePlaydatesLocked() {
	now := h.now()
	for code, pd := range h.playdates {
		for lower, m := range pd.members {
			if now.Sub(m.Updated) > memberTimeout {
				delete(pd.members, lower)
			}
		}
		if len(pd.members) == 0 {
			delete(h.playdates, code)
		}
	}
}

// leaveAllLocked takes a player out of every playdate. Caller must hold
// h.mu.
func (h *Hub) leaveAllLocked(lower string) {
	for code, pd := range h.playdates {
		delete(pd.members, lower)
		if len(pd.members) == 0 {
			delete(h.playdates, code)
		}
	}
}

// joinLocked adds a player to a playdate in the first free slot. Caller
// must hold h.mu.
func (pd *playdate) joinLocked(username string, card json.RawMessage, now time.Time) {
	used := make(map[int]bool)
	for _, m := range pd.members {
		used[m.Slot] = true
	}
	slot := 0
	for used[slot] {
		slot++
	}
	pd.members[strings.ToLower(username)] = &PlaydateMember{Owner: username, Slot: slot, Card: card, Updated: now}
}

// stateLocked returns the playdate as a member sees it, with the events
// after seq. Caller must hold h.mu.
func (pd *playdate) stateLocked(username string, after int64) PlaydateState {
	state := PlaydateState{Code: pd.code, Host: pd.host, You: username, Members: []PlaydateMember{},
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

// accountLocked returns a player's account, or ErrUnauthorized. Caller must hold
// h.mu.
func (h *Hub) accountLocked(username string) (*account, error) {
	me, ok := h.accounts[strings.ToLower(username)]
	if !ok {
		return nil, ErrUnauthorized
	}
	return me, nil
}

// StartPlaydate starts a playdate with the player's pet (its card) and
// returns it. The player leaves any other playdate.
func (h *Hub) StartPlaydate(username string, card json.RawMessage) (PlaydateState, error) {
	if err := checkCard(card); err != nil {
		return PlaydateState{}, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	me, err := h.accountLocked(username)
	if err != nil {
		return PlaydateState{}, err
	}
	h.expirePlaydatesLocked()
	h.leaveAllLocked(strings.ToLower(username))
	if len(h.playdates) >= maxPlaydates {
		return PlaydateState{}, ErrTooManyPlaydate
	}
	var code string
	for code == "" || h.playdates[code] != nil {
		if code, err = newPlaydateCode(); err != nil {
			return PlaydateState{}, err
		}
	}
	pd := &playdate{code: code, host: me.Username, members: make(map[string]*PlaydateMember)}
	pd.joinLocked(me.Username, card, h.now())
	h.playdates[code] = pd
	return pd.stateLocked(me.Username, 0), nil
}

// JoinPlaydate adds the player's pet to a playdate. The player leaves any
// other playdate.
func (h *Hub) JoinPlaydate(username, code string, card json.RawMessage) (PlaydateState, error) {
	if err := checkCard(card); err != nil {
		return PlaydateState{}, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	me, err := h.accountLocked(username)
	if err != nil {
		return PlaydateState{}, err
	}
	h.expirePlaydatesLocked()
	pd := h.playdates[NormalizePlaydateCode(code)]
	if pd == nil {
		return PlaydateState{}, ErrNoPlaydate
	}
	lower := strings.ToLower(me.Username)
	if pd.members[lower] == nil {
		if len(pd.members) >= MaxPlaydateMembers {
			return PlaydateState{}, ErrPlaydateFull
		}
		h.leaveAllLocked(lower)
		pd.joinLocked(me.Username, card, h.now())
	}
	return pd.stateLocked(me.Username, 0), nil
}

// memberLocked returns a playdate and the player's place in it. Caller
// must hold h.mu.
func (h *Hub) memberLocked(username, code string) (*playdate, *PlaydateMember, error) {
	if _, err := h.accountLocked(username); err != nil {
		return nil, nil, err
	}
	h.expirePlaydatesLocked()
	pd := h.playdates[NormalizePlaydateCode(code)]
	if pd == nil {
		return nil, nil, ErrNoPlaydate
	}
	m := pd.members[strings.ToLower(username)]
	if m == nil {
		return nil, nil, ErrNotInPlaydate
	}
	return pd, m, nil
}

// UpdatePlaydate records the player's pet (its card, if given) and its
// position, and returns the playdate with the events after seq
func (h *Hub) UpdatePlaydate(username, code string, card json.RawMessage, position Offset, after int64) (PlaydateState, error) {
	if card != nil {
		if err := checkCard(card); err != nil {
			return PlaydateState{}, err
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	pd, m, err := h.memberLocked(username, code)
	if err != nil {
		return PlaydateState{}, err
	}
	if card != nil {
		m.Card = card
	}
	m.Position, m.Updated = position.clamp(), h.now()
	return pd.stateLocked(m.Owner, after), nil
}

// PostPlaydateEvent adds an event to a playdate. To, if given, must be a
// member.
func (h *Hub) PostPlaydateEvent(username, code string, e PlaydateEvent) (PlaydateEvent, error) {
	if len(e.Data) > maxCardBytes || (len(e.Data) > 0 && !json.Valid(e.Data)) {
		return PlaydateEvent{}, errors.New("invalid event data")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	pd, m, err := h.memberLocked(username, code)
	if err != nil {
		return PlaydateEvent{}, err
	}
	if e.To != "" {
		to := pd.members[strings.ToLower(e.To)]
		if to == nil {
			return PlaydateEvent{}, errors.New("that player isn't at the playdate")
		}
		e.To = to.Owner
	}
	pd.seq++
	e.Seq, e.From, e.Created = pd.seq, m.Owner, h.now()
	e.Kind, e.Text = clip(e.Kind, 20), clip(e.Text, maxEventText)
	pd.events = append(pd.events, e)
	if over := len(pd.events) - maxPlaydateEvents; over > 0 {
		pd.events = append(pd.events[:0:0], pd.events[over:]...)
	}
	m.Updated = h.now()
	return e, nil
}

// LeavePlaydate takes the player out of a playdate
func (h *Hub) LeavePlaydate(username, code string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	pd, _, err := h.memberLocked(username, code)
	if err != nil {
		return err
	}
	delete(pd.members, strings.ToLower(username))
	if len(pd.members) == 0 {
		delete(h.playdates, pd.code)
	}
	return nil
}
