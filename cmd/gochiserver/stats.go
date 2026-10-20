package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// The player's own statistics: every pet they have raised on this
// computer, how each one's life went and how they cared for it. They are
// kept in stats.json in the data directory, apart from the pets' saves, so
// they last after a pet is deleted. They are never sent anywhere: not to a
// cloud hub, not through the REST API, and the game collects nothing else
// about how it is played.

// petLog is the player's record of one pet
type petLog struct {
	Name         string         `json:"name"`
	Species      core.Species   `json:"species"`
	Bred         bool           `json:"bred"` // Born to two of the player's pets, rather than adopted
	Generation   int            `json:"generation"`
	FirstSeen    time.Time      `json:"first_seen"`
	Alive        bool           `json:"alive"`
	CauseOfDeath string         `json:"cause_of_death,omitempty"`
	DiedAt       time.Time      `json:"died_at"` // Zero if it died before it was recorded
	Age          float64        `json:"age"`     // Game days, when last seen
	Deleted      bool           `json:"deleted"` // Its save was deleted
	Interactions int            `json:"interactions"`
	Care         map[string]int `json:"care"` // Interactions by kind
	Hibernations int            `json:"hibernations"`
}

// clean makes counts from a hand-edited file at least 0
func (entry *petLog) clean() {
	entry.Interactions, entry.Hibernations = max(entry.Interactions, 0), max(entry.Hibernations, 0)
	for kind, n := range entry.Care {
		if n < 0 {
			delete(entry.Care, kind)
		}
	}
	if !(entry.Age >= 0) { // Also NaN
		entry.Age = 0
	}
}

// playerStats is the player's statistics, saved in stats.json
type playerStats struct {
	mu    sync.Mutex // Taken after any other lock and never held while taking one
	path  string     // "" to not save them
	pets  map[types.PetID]*petLog
	dirty bool // Changed since last saved
}

// statsFileName is the statistics file in the data directory
const statsFileName = "stats.json"

// loadPlayerStats reads the player's statistics from dir, or starts new
// ones
func loadPlayerStats(dir string) (*playerStats, error) {
	stats := &playerStats{path: filepath.Join(dir, statsFileName), pets: map[types.PetID]*petLog{}}
	contents, err := os.ReadFile(stats.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		var saved struct {
			Pets map[types.PetID]*petLog `json:"pets"`
		}
		if err := json.Unmarshal(contents, &saved); err != nil {
			return nil, fmt.Errorf("%s: %w", stats.path, err)
		}
		for id, entry := range saved.Pets {
			if entry != nil {
				entry.clean()
				stats.pets[id] = entry
			}
		}
	}
	return stats, nil
}

// recorded reports whether a pet is in the statistics as a pet that is
// still saved
func (ps *playerStats) recorded(id types.PetID) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	entry, ok := ps.pets[id]
	return ok && !entry.Deleted
}

// observe brings a pet's record up to date. bred is whether it was just
// born to two of the player's pets; a pet seen for the first time
// otherwise counts as bred if it is past the first generation. It reports
// whether the pet is new or has just died, which are worth saving at once.
func (ps *playerStats) observe(t core.Tally, bred bool) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	entry, ok := ps.pets[t.ID]
	if !ok {
		entry = &petLog{FirstSeen: time.Now(), Bred: bred || t.Generation > 1, Alive: true}
		ps.pets[t.ID] = entry
	}
	died := entry.Alive && !t.Alive
	if died && ok {
		entry.DiedAt = time.Now()
	}
	if entry.Deleted { // Its save is back: the game stopped while deleting it
		entry.Deleted, ps.dirty = false, true
	}
	before := *entry
	entry.Name, entry.Species, entry.Generation = t.Name, t.Species, t.Generation
	entry.Alive, entry.CauseOfDeath, entry.Age = t.Alive, t.CauseOfDeath, t.Age
	entry.Interactions, entry.Hibernations = t.Interactions, t.Hibernations
	careChanged := !maps.Equal(entry.Care, t.Care)
	if careChanged {
		entry.Care = t.Care
	}
	if !ok || careChanged || before.Name != entry.Name || before.Species != entry.Species ||
		before.Generation != entry.Generation || before.Alive != entry.Alive ||
		before.CauseOfDeath != entry.CauseOfDeath || before.Age != entry.Age ||
		before.Interactions != entry.Interactions || before.Hibernations != entry.Hibernations {
		ps.dirty = true
	}
	return !ok || died
}

// undelete records that a pet's save wasn't deleted after all
func (ps *playerStats) undelete(id types.PetID) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if entry, ok := ps.pets[id]; ok && entry.Deleted {
		entry.Deleted, ps.dirty = false, true
	}
}

// deleted records that a pet's save was deleted
func (ps *playerStats) deleted(id types.PetID) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if entry, ok := ps.pets[id]; ok && !entry.Deleted {
		entry.Deleted, ps.dirty = true, true
	}
}

// save writes the statistics if they have changed. Only the player's
// account on this computer can read the file.
func (ps *playerStats) save() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if !ps.dirty || ps.path == "" {
		return nil
	}
	contents, err := json.Marshal(struct {
		Pets map[types.PetID]*petLog `json:"pets"`
	}{ps.pets})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ps.path), 0o700); err != nil {
		return err
	}
	tmp := ps.path + ".tmp"
	if err := os.WriteFile(tmp, contents, 0o600); err != nil {
		os.Remove(tmp) // Cut short (a full disk)
		return err
	}
	if err := os.Rename(tmp, ps.path); err != nil {
		return err
	}
	ps.dirty = false
	return nil
}

// statCount is a count of something, for lists sorted by count
type statCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// statsPet is one pet in the statistics shown on the page
type statsPet struct {
	Name         string       `json:"name"`
	Species      core.Species `json:"species"`
	Bred         bool         `json:"bred"`
	Generation   int          `json:"generation"`
	Alive        bool         `json:"alive"`
	CauseOfDeath string       `json:"cause_of_death,omitempty"`
	Age          float64      `json:"age"`
	Deleted      bool         `json:"deleted"`
	Interactions int          `json:"interactions"`
	FirstSeen    time.Time    `json:"first_seen"`
}

// statsSummary is the player's statistics as the page shows them
type statsSummary struct {
	Raised       int         `json:"raised"`  // Every pet
	Adopted      int         `json:"adopted"` // Started as new pets
	Bred         int         `json:"bred"`    // Born to the player's pets
	Alive        int         `json:"alive"`
	Died         int         `json:"died"`
	Deleted      int         `json:"deleted"`
	Hibernations int         `json:"hibernations"`
	Causes       []statCount `json:"causes"` // How pets died
	Interactions int         `json:"interactions"`
	Care         []statCount `json:"care"` // Interactions by kind
	Species      []statCount `json:"species"`
	Pets         []statsPet  `json:"pets"` // Most recent first
}

// notCountedByKind names interactions from before they were counted by kind
const notCountedByKind = "Other (before kinds were counted)"

// summary returns the statistics as the page shows them
func (ps *playerStats) summary() statsSummary {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	sum := statsSummary{Causes: []statCount{}, Care: []statCount{}, Species: []statCount{}, Pets: []statsPet{}}
	causes, care, species := map[string]int{}, map[string]int{}, map[string]int{}
	for _, entry := range ps.pets {
		sum.Raised++
		if entry.Bred {
			sum.Bred++
		} else {
			sum.Adopted++
		}
		if entry.Alive {
			sum.Alive++
		} else {
			sum.Died++
			cause := entry.CauseOfDeath
			if cause == "" {
				cause = "Unknown"
			}
			causes[cause]++
		}
		if entry.Deleted {
			sum.Deleted++
		}
		sum.Hibernations += entry.Hibernations
		sum.Interactions += entry.Interactions
		byKind := 0
		for kind, n := range entry.Care {
			care[kind] += n
			byKind += n
		}
		if other := entry.Interactions - byKind; other > 0 {
			care[notCountedByKind] += other
		}
		species[entry.Species.Def().Name]++
		sum.Pets = append(sum.Pets, statsPet{Name: entry.Name, Species: entry.Species, Bred: entry.Bred,
			Generation: entry.Generation, Alive: entry.Alive, CauseOfDeath: entry.CauseOfDeath, Age: entry.Age,
			Deleted: entry.Deleted, Interactions: entry.Interactions, FirstSeen: entry.FirstSeen})
	}
	sum.Causes, sum.Care, sum.Species = sortedCounts(causes), sortedCounts(care), sortedCounts(species)
	sort.SliceStable(sum.Pets, func(i, j int) bool {
		a, b := sum.Pets[i], sum.Pets[j]
		if !a.FirstSeen.Equal(b.FirstSeen) {
			return a.FirstSeen.After(b.FirstSeen)
		}
		return a.Name < b.Name
	})
	return sum
}

// sortedCounts returns counts, largest first
func sortedCounts(counts map[string]int) []statCount {
	list := make([]statCount, 0, len(counts))
	for name, n := range counts {
		list = append(list, statCount{Name: name, Count: n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Count != list[j].Count {
			return list[i].Count > list[j].Count
		}
		return list[i].Name < list[j].Name
	})
	return list
}

// observePet brings a pet's record up to date and saves the statistics at
// once if the pet is new or has just died
func (s *Server) observePet(pet *core.DigitalPet, bred bool) {
	if s.stats == nil || pet == nil {
		return
	}
	if s.stats.observe(pet.Tally(), bred) {
		if err := s.stats.save(); err != nil {
			log.Printf("Could not save your statistics: %v", err)
		}
	}
}

// recordSavedPets adds the saved pets the statistics don't have yet, such
// as pets from before they were kept
func (s *Server) recordSavedPets() {
	if s.stats == nil || s.store == nil {
		return
	}
	saved, err := s.store.List()
	if err != nil {
		log.Printf("Could not list the saved pets for your statistics: %v", err)
		return
	}
	for _, entry := range saved {
		if entry.ID == "" || s.stats.recorded(types.PetID(entry.ID)) {
			continue
		}
		s.recordSavedPet(entry.Slot)
	}
}

// recordSavedPet adds one saved pet to the statistics. A save that can't
// be read is left for the player to see under Switch Pet.
func (s *Server) recordSavedPet(slot string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Could not read the pet in %s for your statistics: %v", slot, r)
		}
	}()
	if pet, _, _, err := s.store.Load(slot); err == nil {
		s.observePet(pet, false)
	}
}

// handleStats handles GET /api/stats - the player's statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	if s.stats == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "Statistics are not kept"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.stats.summary()})
}
