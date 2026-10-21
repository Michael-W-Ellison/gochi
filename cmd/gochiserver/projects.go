package main

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Long-term projects: goals that take many weeks, tracked from what the
// household has already done. They add no new rules.

// project is a long-term project and how far along it is
type project struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Progress int    `json:"progress"`
	Target   int    `json:"target"`
	Detail   string `json:"detail"`
}

// handleProjects handles GET /api/projects
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	s.mu.RLock()
	pet := s.pet
	s.mu.RUnlock()
	var d householdData
	upgrades := 0
	if s.house != nil {
		d = s.house.snapshot()
		s.house.mu.Lock()
		upgrades = len(s.house.upgradesLocked())
		s.house.mu.Unlock()
	}
	var saved []savedPet
	if s.store != nil {
		saved, _ = s.store.List()
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: projectsFor(pet, d, upgrades, saved)})
}

// projectsFor works out the projects from the active pet, the household
// and the saved pets
func projectsFor(pet *core.DigitalPet, d householdData, upgrades int, saved []savedPet) []project {
	var list []project

	// Raise a Grand Champion
	wins := 0
	if slices.Contains(d.Achieved, "grand_champion") {
		wins = 3
	} else if pet != nil {
		wins = min(pet.ChampionWins(), 3)
	}
	list = append(list, project{ID: "grand_champion", Name: "Raise a Grand Champion", Icon: "👑", Progress: wins, Target: 3,
		Detail: "Win three different shows at Champion level with one pet."})

	// Complete the collection
	sets := len(core.SouvenirSets())
	list = append(list, project{ID: "collection", Name: "Complete the collection", Icon: "🧺",
		Progress: len(setsComplete(d.Collected)), Target: sets,
		Detail: fmt.Sprintf("Find every place's souvenirs: %d sets.", sets)})

	// Furnish the house
	total := len(core.HouseUpgrades())
	list = append(list, project{ID: "house", Name: "Furnish the house", Icon: "🏠", Progress: upgrades, Target: total,
		Detail: "Buy every house upgrade with Home Stars."})

	// Five generations
	generation := 1
	for _, p := range saved {
		generation = max(generation, p.Generation)
	}
	list = append(list, project{ID: "generations", Name: "Five generations", Icon: "🌳", Progress: min(generation, 5), Target: 5,
		Detail: "Raise a family to its fifth generation."})

	// Raise every breed of a tier: the one nearest done
	raised := map[string]bool{}
	for _, p := range saved {
		raised[p.Breed] = true
	}
	best := project{ID: "breeds", Name: "Raise every breed of a tier", Icon: "🐾"}
	for _, species := range []core.Species{core.SpeciesCat, core.SpeciesDog} {
		for tier := core.TierEasy; tier <= core.TierDemanding; tier++ {
			n, have := 0, 0
			for _, b := range core.Breeds(species) {
				if b.Tier == tier && !b.Mixed {
					n++
					if raised[b.ID] {
						have++
					}
				}
			}
			if n > 0 && (best.Target == 0 || have*best.Target > best.Progress*n) {
				best.Progress, best.Target = have, n
				best.Detail = fmt.Sprintf("Raise every %s %s breed.", strings.ToLower(tierNames[tier]),
					strings.ToLower(species.Def().Name))
			}
		}
	}
	list = append(list, best)
	return list
}
