package main

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// The household's souvenir collection: every souvenir any pet has found.
// Completing a place's four common souvenirs earns Home Stars, and so does
// each rare one; completing every set earns more. Up to five souvenirs go
// on the house's shelf; a whole place's set there cheers every pet up a
// little. Pets can also give each other souvenirs.

// Home Stars for the collection
const (
	starsSet     = 10
	starsRare    = 10
	starsCurator = 50
	shelfSize    = 5
)

// errBadShelf is returned for a shelf of souvenirs not collected
var errBadShelf = errors.New("the shelf can hold up to 5 different souvenirs your pets have found")

// validShelf keeps the shelf to collected souvenirs, each once, at most
// shelfSize
func validShelf(shelf, collected []string) []string {
	var kept []string
	for _, id := range shelf {
		if slices.Contains(collected, id) && !slices.Contains(kept, id) && len(kept) < shelfSize {
			kept = append(kept, id)
		}
	}
	return kept
}

// shelfSetPlace returns the place whose whole set is on the shelf, or ""
func shelfSetPlace(shelf []string) string {
	if len(shelf) != shelfSize {
		return ""
	}
	first, ok := core.FindSouvenir(shelf[0])
	if !ok {
		return ""
	}
	for _, id := range shelf {
		if s, ok := core.FindSouvenir(id); !ok || s.Place != first.Place {
			return ""
		}
	}
	return first.Place
}

// collectionNews is what adding souvenirs to the collection completed
type collectionNews struct {
	Sets    []string // Places whose common souvenirs are now all found
	Rare    []string // Rare souvenirs found for the first time
	AllSets bool     // Every set complete
}

// collect adds souvenirs to the household's collection
func (h *household) collect(ids []string) collectionNews {
	h.mu.Lock()
	defer h.mu.Unlock()
	var news collectionNews
	before := setsComplete(h.data.Collected)
	added := false
	for _, id := range ids {
		s, ok := core.FindSouvenir(id)
		if !ok || slices.Contains(h.data.Collected, id) {
			continue
		}
		h.data.Collected = append(h.data.Collected, id)
		added = true
		if s.Rare {
			news.Rare = append(news.Rare, id)
		}
	}
	if !added {
		return news
	}
	after := setsComplete(h.data.Collected)
	for _, place := range after {
		if !slices.Contains(before, place) {
			news.Sets = append(news.Sets, place)
		}
	}
	news.AllSets = len(after) == len(core.SouvenirSets()) && len(before) < len(after)
	h.saveOrLogLocked()
	return news
}

// setsComplete returns the places whose four common souvenirs are all in
// the collection
func setsComplete(collected []string) []string {
	var done []string
	for place, set := range core.SouvenirSets() {
		complete := true
		for _, s := range set {
			if !s.Rare && !slices.Contains(collected, s.ID) {
				complete = false
			}
		}
		if complete {
			done = append(done, place)
		}
	}
	sort.Strings(done)
	return done
}

// setShelf puts souvenirs on the shelf
func (h *household) setShelf(ids []string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	shelf := validShelf(ids, h.data.Collected)
	if len(shelf) != len(ids) {
		return errBadShelf
	}
	h.data.Shelf, h.data.SettingsTime = shelf, time.Now()
	h.apply()
	return h.saveLocked()
}

// collectLocked adds souvenirs to the household's collection and pays its
// rewards. Caller must hold s.mu.
func (s *Server) collectLocked(pet *core.DigitalPet, ids []string) {
	if s.house == nil || len(ids) == 0 {
		return
	}
	news := s.house.collect(ids)
	for _, place := range news.Sets {
		s.awardStarsLocked(starsSet, "set:"+place, true, "completing a set of souvenirs")
		if pet != nil {
			pet.Award("set_collector")
			pet.KeepSetCompleted(place)
		}
	}
	for _, id := range news.Rare {
		s.awardStarsLocked(starsRare, "rare:"+id, true, "a rare souvenir")
	}
	if news.AllSets {
		s.awardStarsLocked(starsCurator, "curator", true, "completing every set of souvenirs")
		if pet != nil {
			pet.Award("curator")
		}
	}
	if len(news.Sets) > 0 && pet != nil {
		s.events.Emit(core.GameEvent{Type: core.EventAchievement, PetID: pet.ID, Good: true,
			Message: fmt.Sprintf("🧺 Your household completed a set of souvenirs! +%d Home Stars.", starsSet*len(news.Sets))})
	}
}

// souvenirIDs returns the souvenirs a pet has found
func souvenirIDs(pet *core.DigitalPet) []string {
	var ids []string
	for _, c := range pet.SouvenirCollection() {
		ids = append(ids, c.ID)
	}
	return ids
}

// collectionView is the collection as the page shows it
type collectionView struct {
	Places []placeSet `json:"places"`
	Shelf  []string   `json:"shelf"`
	Mine   []string   `json:"mine"` // The active pet's souvenirs, which it can give
}

type placeSet struct {
	Place     string          `json:"place"`
	Souvenirs []core.Souvenir `json:"souvenirs"`
	Found     []string        `json:"found"`
	Complete  bool            `json:"complete"`
}

// handleCollection handles GET /api/collection - the household's
// collection, the shelf and the active pet's souvenirs
func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	if s.house == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No household"})
		return
	}
	d := s.house.snapshot()
	view := collectionView{Shelf: append([]string{}, d.Shelf...), Places: []placeSet{}, Mine: []string{}}
	done := setsComplete(d.Collected)
	sets := core.SouvenirSets()
	var places []string
	for place := range sets {
		places = append(places, place)
	}
	sort.Strings(places)
	for _, place := range places {
		set := placeSet{Place: place, Souvenirs: sets[place], Found: []string{}, Complete: slices.Contains(done, place)}
		for _, sv := range sets[place] {
			if slices.Contains(d.Collected, sv.ID) {
				set.Found = append(set.Found, sv.ID)
			}
		}
		view.Places = append(view.Places, set)
	}
	s.mu.RLock()
	if s.pet != nil {
		view.Mine = souvenirIDs(s.pet)
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: view})
}

// handleShelf handles POST /api/home/shelf with {"items": [souvenir IDs]}
func (s *Server) handleShelf(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []string `json:"items"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if s.house == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No household"})
		return
	}
	if len(req.Items) > shelfSize {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: capitalize(errBadShelf.Error()) + "."})
		return
	}
	if err := s.house.setShelf(req.Items); err != nil {
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: capitalize(err.Error()) + "."})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"shelf": req.Items, "set": shelfSetPlace(req.Items)}})
}

// handleGiveSouvenir handles POST /api/souvenirs/gift with {"souvenir",
// "to_slot"}: the active pet gives a souvenir to another saved pet
func (s *Server) handleGiveSouvenir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Souvenir string `json:"souvenir"`
		ToSlot   string `json:"to_slot"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	s.withOtherPet(w, req.ToSlot, func(pet, other *core.DigitalPet) (string, error) {
		return core.GiveSouvenir(pet, other, req.Souvenir)
	}, func(w http.ResponseWriter, pet, other string, err error) {
		message := "Something went wrong"
		switch {
		case errors.Is(err, core.ErrNoSouvenir):
			message = pet + " doesn't have that souvenir"
		case errors.Is(err, core.ErrNotFriends):
			message = pet + " and " + other + " need to meet first"
		case errors.Is(err, core.ErrPetNotAvailable):
			message = "Both pets need to be alive"
		}
		writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: message})
	})
}
