package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Families: giving a young child Friendship Points from its parents,
// passing a dead pet's belongings and points on to a child, and pop-ups
// that tell the player how long a child stays young.

// popup is a message the page shows in front of everything until it is
// dismissed
type popup struct {
	ID    string `json:"id"`
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// adultWarning is how long before a young pet grows up the player is
// warned, in real time
const adultWarning = 5 * time.Minute

// addPopupLocked queues a pop-up. Caller must hold s.mu.
func (s *Server) addPopupLocked(p popup) {
	for _, q := range s.popups {
		if q.ID == p.ID {
			return
		}
	}
	s.popups = append(s.popups, p)
}

// realDuration converts game days to real time at the game's speed
func (s *Server) realDuration(gameDays float64) time.Duration {
	scale := 1.0
	if s.clock != nil {
		scale = max(s.clock.GetTimeScaleMultiplier(), 1e-9)
	}
	return time.Duration(gameDays * secondsPerGameDay / scale * float64(time.Second))
}

// describeDuration says roughly how long a duration is
func describeDuration(d time.Duration) string {
	switch minutes := int(d.Round(time.Minute).Minutes()); {
	case minutes < 1:
		return "less than a minute"
	case minutes == 1:
		return "about a minute"
	case minutes < 120:
		return fmt.Sprintf("about %d minutes", minutes)
	default:
		return fmt.Sprintf("about %d hours", minutes/60)
	}
}

// youngLimitText explains the time limit for giving a young pet points
func youngLimitText(name string, left time.Duration) string {
	return fmt.Sprintf("While %s is young (%s more of play while %s is the active pet), its parents can give it "+
		"Friendship Points: open a parent's Body panel and choose Family. After %s grows up, points can pass to it "+
		"only when a parent dies.", name, describeDuration(left), name, name)
}

// familyMember is a parent or child as the page shows it
type familyMember struct {
	Slot        string       `json:"slot"`
	Name        string       `json:"name"`
	Species     core.Species `json:"species"`
	Alive       bool         `json:"alive"`
	Young       bool         `json:"young"`
	YoungLeft   int          `json:"young_seconds_left"` // Real seconds of play until it grows up (0 if grown)
	Points      int          `json:"points"`
	CanTransfer bool         `json:"can_transfer"` // Can be given points now
}

// familyStatus is the active pet's family
type familyStatus struct {
	Name       string         `json:"name"`
	Alive      bool           `json:"alive"`
	Points     int            `json:"points"`
	Young      bool           `json:"young"`
	YoungLeft  int            `json:"young_seconds_left"`
	Parents    []familyMember `json:"parents"`
	Children   []familyMember `json:"children"`
	CanPassOn  bool           `json:"can_pass_on"` // Dead, with belongings not yet passed on
	PassedOnTo string         `json:"passed_on_to,omitempty"`
}

// familyOf returns a pet's living parents and its children among the
// saved pets other than the one in slot
func (s *Server) familyOf(pet *core.DigitalPet, slot string) (parents, children []familyMember) {
	parents, children = []familyMember{}, []familyMember{}
	if s.store == nil {
		return
	}
	saved, err := s.store.List()
	if err != nil {
		return
	}
	for _, entry := range saved {
		if entry.Slot == slot || entry.Unreadable {
			continue
		}
		other, _, _, err := s.store.Load(entry.Slot)
		if err != nil {
			continue
		}
		member := familyMember{Slot: entry.Slot, Name: other.Name, Species: other.GetSpecies(), Alive: other.IsAlive(),
			Young: other.IsYoung(), Points: other.FriendshipPoints}
		if member.Young {
			member.YoungLeft = int(s.realDuration(other.YoungDaysLeft()).Seconds())
		}
		switch {
		case other.IsChildOf(pet.ID):
			member.CanTransfer = member.Alive && (member.Young || !pet.IsAlive())
			children = append(children, member)
		case pet.IsChildOf(other.ID) && member.Alive:
			parents = append(parents, member)
		}
	}
	return
}

// handleFamily handles GET /api/family - the active pet's parents and
// children, and what can be given
func (s *Server) handleFamily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	pet, slot, ok := s.activeCopy(w)
	if !ok {
		return
	}
	st := familyStatus{Name: pet.Name, Alive: pet.IsAlive(), Points: pet.FriendshipPoints, Young: pet.IsYoung()}
	if st.Young {
		st.YoungLeft = int(s.realDuration(pet.YoungDaysLeft()).Seconds())
	}
	st.Parents, st.Children = s.familyOf(pet, slot)
	st.CanPassOn = !st.Alive && pet.PassedOnTo == "" && len(st.Children) > 0
	if pet.PassedOnTo != "" {
		st.PassedOnTo = string(pet.PassedOnTo)
		for _, c := range st.Children {
			if c.Slot == string(pet.PassedOnTo) {
				st.PassedOnTo = c.Name
			}
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: st})
}

// handleFamilyTransfer handles POST /api/family/transfer with
// {"child_slot", "amount", "parent_pin"}: gives the active pet's points to
// its child (young, unless the active pet has died)
func (s *Server) handleFamilyTransfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChildSlot string `json:"child_slot"`
		Amount    int    `json:"amount"`
		ParentPIN string `json:"parent_pin"`
	}
	if !decodePost(w, r, &req) || !s.parentAllows(w, req.ParentPIN) {
		return
	}
	s.withChild(w, req.ChildSlot, func(parent, child *core.DigitalPet) (string, error) {
		if err := core.TransferPoints(parent, child, req.Amount); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s gave %s %d Friendship Points. ❤️", parent.Name, child.Name, req.Amount), nil
	})
}

// handleFamilyPassOn handles POST /api/family/pass_on with {"child_slot",
// "points", "parent_pin"}: passes the active pet's belongings, after it
// died, to its child, with points of its Friendship Points
func (s *Server) handleFamilyPassOn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChildSlot string `json:"child_slot"`
		Points    int    `json:"points"`
		ParentPIN string `json:"parent_pin"`
	}
	if !decodePost(w, r, &req) || !s.parentAllows(w, req.ParentPIN) {
		return
	}
	s.withChild(w, req.ChildSlot, func(parent, child *core.DigitalPet) (string, error) {
		if req.Points < 0 || req.Points > parent.FriendshipPoints {
			return "", core.ErrNothingToGive
		}
		got, err := core.PassOn(parent, child)
		if err != nil {
			return "", err
		}
		if req.Points > 0 {
			if err := core.TransferPoints(parent, child, req.Points); err != nil {
				return "", err
			}
		}
		parts := []string{}
		if n := len(got.Items) + len(got.Cosmetics); n > 0 {
			parts = append(parts, fmt.Sprintf("%d belongings", n))
		}
		if got.Supplies > 0 {
			parts = append(parts, fmt.Sprintf("%d supplies", got.Supplies))
		}
		if req.Points > 0 {
			parts = append(parts, fmt.Sprintf("%d Friendship Points", req.Points))
		}
		if len(parts) == 0 {
			return fmt.Sprintf("%s had nothing left to pass on to %s.", parent.Name, child.Name), nil
		}
		return fmt.Sprintf("%s's %s now belong to %s. 💐", parent.Name, strings.Join(parts, ", "), child.Name), nil
	})
}

// withChild runs change on the active pet and one of its children, saved
// in childSlot, then saves both
func (s *Server) withChild(w http.ResponseWriter, childSlot string, change func(parent, child *core.DigitalPet) (string, error)) {
	if !ValidSlot(childSlot) || s.store == nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid pet"})
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pet == nil || childSlot == s.slot {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Choose one of this pet's children"})
		return
	}
	child, _, _, err := s.store.Load(childSlot)
	if err != nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No saved pet in that slot"})
		return
	}
	s.markActiveLocked(time.Now())
	text, err := change(s.pet, child)
	if err != nil {
		writeFamilyError(w, s.pet.Name, child.Name, err)
		return
	}
	if err := s.store.SavePet(childSlot, child); err != nil {
		log.Printf("Failed to save %s: %v", child.Name, err)
	}
	if err := s.saveActiveLocked(); err != nil {
		log.Printf("Failed to save %s: %v", s.pet.Name, err)
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"text": text}})
}

// writeFamilyError explains why points or belongings couldn't be given
func writeFamilyError(w http.ResponseWriter, parent, child string, err error) {
	message := "Something went wrong"
	switch {
	case errors.Is(err, core.ErrNotChild):
		message = child + " isn't " + parent + "'s child"
	case errors.Is(err, core.ErrChildGrownUp):
		message = child + " has grown up: points can pass to it only when " + parent + " dies"
	case errors.Is(err, core.ErrNothingToGive):
		message = "Choose how many points to give, up to what " + parent + " has"
	case errors.Is(err, core.ErrStillAlive):
		message = parent + " is still alive"
	case errors.Is(err, core.ErrAlreadyPassedOn):
		message = parent + "'s belongings have already been passed on"
	case errors.Is(err, core.ErrPetNotAvailable):
		message = child + " has passed away"
	}
	writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: message})
}

// handlePopupDismiss handles POST /api/popup/dismiss with {"id"}
func (s *Server) handlePopupDismiss(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	s.mu.Lock()
	kept := s.popups[:0]
	for _, p := range s.popups {
		if p.ID != req.ID {
			kept = append(kept, p)
		}
	}
	s.popups = kept
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// checkGrowingUpLocked warns the player when the active pet, a young pet
// with a living parent, has adultWarning of play left before it grows up.
// Caller must hold s.mu.
func (s *Server) checkGrowingUpLocked() {
	pet := s.pet
	if pet == nil || !pet.IsAlive() || !pet.IsYoung() ||
		s.realDuration(pet.YoungDaysLeft()) > adultWarning {
		return
	}
	if pet.MarkAdultWarned() {
		return
	}
	parents, _ := s.familyOf(pet, s.slot)
	if len(parents) == 0 {
		return
	}
	var names []string
	for _, p := range parents {
		names = append(names, fmt.Sprintf("%s (%d points)", p.Name, p.Points))
	}
	s.addPopupLocked(popup{ID: "grown_" + string(pet.ID), Icon: "⏰", Title: pet.Name + " is almost grown up",
		Text: fmt.Sprintf("%s grows up in %s. Until then, %s can give %s Friendship Points: switch to a parent, "+
			"open its Body panel and choose Family. After %s grows up, points can pass to it only when a parent dies.",
			pet.Name, describeDuration(s.realDuration(pet.YoungDaysLeft())), strings.Join(names, " and "), pet.Name, pet.Name)})
}

// birthPopupLocked tells the player how long a newborn stays young.
// Caller must hold s.mu.
func (s *Server) birthPopupLocked(baby *core.DigitalPet) {
	s.addPopupLocked(popup{ID: "born_" + string(baby.ID), Icon: "🍼", Title: baby.Name + " was born!",
		Text: youngLimitText(baby.Name, s.realDuration(biology.AdultAge))})
}
