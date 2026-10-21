package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Breeds: the breeds a new pet can be, which unlock as the player's pets
// earn care achievements (recorded in the household, so they count on
// every device).

// errBreedLocked is returned for a breed that isn't unlocked yet
var errBreedLocked = errors.New("that breed isn't unlocked yet")

// breedTier returns the highest breed tier the player has unlocked
func (s *Server) breedTier() int {
	if s.house == nil {
		return core.TierEasy
	}
	return s.house.breedTier()
}

// chooseBreed checks the breed asked for a new pet: "" for the species'
// default, "random" for any unlocked breed of the species
func (s *Server) chooseBreed(species core.Species, id string) (string, error) {
	tier := s.breedTier()
	if id == "random" {
		var choices []string
		for _, b := range core.Breeds(species) {
			if b.Tier <= tier {
				choices = append(choices, b.ID)
			}
		}
		return choices[rand.IntN(len(choices))], nil
	}
	if id == "" {
		return "", nil
	}
	b, ok := core.FindBreed(id)
	switch {
	case !ok || b.Species != species:
		return "", errors.New("unknown breed")
	case b.Tier > tier:
		return "", errBreedLocked
	}
	return id, nil
}

// tierView is one breed tier and what unlocks it
type tierView struct {
	Tier     int           `json:"tier"`
	Name     string        `json:"name"`
	Unlocked bool          `json:"unlocked"`
	Needs    []achievement `json:"needs"`
}

type achievement struct {
	core.Achievement
	Earned bool `json:"earned"`
}

// breedView is a breed for the page
type breedView struct {
	core.BreedDef
	Unlocked   bool                    `json:"unlocked"`
	Typical    core.PetLook            `json:"typical"`
	Tendencies []core.TendencyStrength `json:"tendency_details"`
}

// tierNames name the breed tiers
var tierNames = map[int]string{core.TierEasy: "Easy care", core.TierModerate: "More care", core.TierDemanding: "Demanding care"}

// handleBreeds handles GET /api/breeds?species=cat - the breeds of a
// species, which are unlocked, and what unlocks the rest
func (s *Server) handleBreeds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	species, ok := core.ParseSpecies(r.URL.Query().Get("species"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown kind of pet"})
		return
	}
	tier := s.breedTier()
	var achieved []string
	if s.house != nil {
		achieved = s.house.achieved()
	}
	list := []breedView{}
	for _, b := range core.Breeds(species) {
		view := breedView{BreedDef: b, Unlocked: b.Tier <= tier, Typical: core.TypicalLook(b.ID),
			Tendencies: []core.TendencyStrength{}}
		for _, id := range b.Tendencies {
			if t, ok := core.FindTendency(id); ok {
				view.Tendencies = append(view.Tendencies, core.TendencyStrength{Tendency: t, Strength: 1})
			}
		}
		list = append(list, view)
	}
	tiers := []tierView{}
	for _, t := range []int{core.TierEasy, core.TierModerate, core.TierDemanding} {
		view := tierView{Tier: t, Name: tierNames[t], Unlocked: t <= tier, Needs: []achievement{}}
		for _, id := range core.TierAchievements[t] {
			if a, ok := core.FindAchievement(id); ok {
				view.Needs = append(view.Needs, achievement{a, slices.Contains(achieved, id)})
			}
		}
		tiers = append(tiers, view)
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"breeds": list, "tiers": tiers, "tier": tier, "colors": core.CoatColors(),
	}})
}

// noteAchievementsLocked records a pet's achievements in the household and
// tells the player when they unlock more breeds. Caller must hold s.mu.
func (s *Server) noteAchievementsLocked(pet *core.DigitalPet) {
	if s.house == nil || pet == nil {
		return
	}
	var ids []string
	for _, a := range pet.EarnedAchievements() {
		ids = append(ids, a.ID)
	}
	if !s.house.recordAchievements(ids) {
		return
	}
	tier := s.house.breedTier()
	var names []string
	for _, b := range core.Breeds("") {
		if b.Tier == tier {
			names = append(names, b.Name)
		}
	}
	s.addPopupLocked(popup{ID: fmt.Sprintf("breeds_%d", tier), Icon: "🐾", Title: "New breeds to adopt",
		Text: fmt.Sprintf("Your care has unlocked %d more breeds (%s): %s. Choose one when you create a new pet.",
			len(names), strings.ToLower(tierNames[tier]), strings.Join(names, ", "))})
}

// scanAchievements records the achievements of every saved pet once, for
// pets saved before achievements unlocked breeds
func (s *Server) scanAchievements() {
	if s.house == nil || s.store == nil || s.house.snapshot().AchievedScanned {
		return
	}
	saved, err := s.store.List()
	if err != nil {
		log.Printf("Could not list the saved pets for their achievements: %v", err)
		return
	}
	for _, entry := range saved {
		if entry.Unreadable {
			continue
		}
		if pet, _, _, err := s.store.Load(entry.Slot); err == nil {
			s.mu.Lock()
			s.noteAchievementsLocked(pet)
			s.mu.Unlock()
		}
	}
	s.house.mu.Lock()
	s.house.data.AchievedScanned = true
	s.house.saveLocked()
	s.house.mu.Unlock()
}
