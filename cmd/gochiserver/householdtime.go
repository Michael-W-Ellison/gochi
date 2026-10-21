package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Household time: while time runs for the active pet, the saved pets live
// on in the household's care (internal/core/householdtime.go). The game
// days played are added up, and every householdTimeEvery the saved pets
// are brought up to date and saved.

// householdTimeEvery is how often the saved pets are brought up to date
const householdTimeEvery = 5 * time.Minute

// householdTime brings the saved pets up to date with the game days played
// since the last time. It takes the save lock itself, so it runs after
// tick rather than inside it.
func (s *Server) householdTime(now time.Time) {
	s.mu.Lock()
	if s.store == nil || s.householdDays <= 0 || now.Sub(s.lastHouseholdTime) < householdTimeEvery {
		s.mu.Unlock()
		return
	}
	days := s.householdDays
	s.householdDays, s.lastHouseholdTime = 0, now
	s.mu.Unlock()
	s.liveInHousehold(days)
}

// liveInHousehold lets every saved pet but the active one live days game
// days in the household's care, and saves them
func (s *Server) liveInHousehold(days float64) {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.RLock()
	active, activeSlot := s.pet, s.slot
	s.mu.RUnlock()
	saved, err := s.store.List()
	if err != nil {
		return
	}

	type livingPet struct {
		slot string
		pet  *core.DigitalPet
		news core.HouseholdNews
	}
	var living []livingPet
	var pets []*core.DigitalPet
	for _, entry := range saved {
		if entry.Slot == activeSlot || entry.Unreadable || !entry.IsAlive || entry.Hibernating {
			continue
		}
		pet, _, _, err := s.store.Load(entry.Slot)
		if err != nil || pet.IsHouseholdPaused() {
			continue
		}
		if news := pet.LiveInHousehold(days); news.Lived {
			living = append(living, livingPet{slot: entry.Slot, pet: pet, news: news})
			pets = append(pets, pet)
		}
	}
	core.LiveTogether(days, active, pets...)

	s.mu.Lock()
	for _, l := range living {
		if l.news.Died {
			s.addPopupLocked(popup{ID: "died_" + string(l.pet.ID), Icon: "🕊️", Title: l.pet.Name + " has passed away",
				Text: l.pet.Name + " died of old age at home, cared for by the household. You can see its life in the pet list."})
		}
		s.growingUpWarningLocked(l.pet, l.slot)
		s.observePet(l.pet, false)
	}
	s.mu.Unlock()
	for _, l := range living {
		if err := s.store.SavePet(l.slot, l.pet); err != nil {
			log.Printf("Failed to save %s after household time: %v", l.pet.Name, err)
		}
	}
}

// handlePausePet handles POST /api/saves/pause with {"slot", "paused"}:
// keep a pet as it is while it isn't the active pet, or let it live on
// with the household
func (s *Server) handlePausePet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slot   string `json:"slot"`
		Paused bool   `json:"paused"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if !ValidSlot(req.Slot) || s.store == nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid pet"})
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markActiveLocked(time.Now())
	if req.Slot == s.slot && s.pet != nil {
		s.pet.SetHouseholdPaused(req.Paused)
		if err := s.saveActiveLocked(); err != nil {
			log.Printf("Failed to save %s: %v", s.pet.Name, err)
		}
	} else {
		pet, _, _, err := s.store.Load(req.Slot)
		if err != nil {
			writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No saved pet in that slot"})
			return
		}
		pet.SetHouseholdPaused(req.Paused)
		if err := s.store.SavePet(req.Slot, pet); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save"})
			return
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true})
}
