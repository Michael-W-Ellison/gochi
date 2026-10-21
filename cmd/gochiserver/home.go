package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

// Home Stars as the game is played, and the household's page routes

// awardStarsLocked adds Home Stars and tells the player. Caller must hold
// s.mu.
func (s *Server) awardStarsLocked(stars int, what string, once bool, why string) {
	if s.house == nil {
		return
	}
	if added := s.house.earn(stars, what, once); added > 0 {
		s.events.Emit(core.GameEvent{Type: core.EventAchievement, PetID: s.pet.ID, Good: true,
			Message: fmt.Sprintf("⭐ +%d Home Stars for %s.", added, why)})
	}
}

// starsForTripLocked awards Home Stars for a trip just finished. Caller
// must hold s.mu.
func (s *Server) starsForTripLocked() {
	s.awardStarsLocked(starsTrip, "trip", false, "a trip")
	if a := s.pet.LastAdventure; a != nil && a.Discovered != "" {
		s.awardStarsLocked(starsNewPlace, "place:"+string(s.pet.ID)+":"+a.Discovered, true, "discovering a new place")
	}
}

// starsForAchievementLocked awards Home Stars for an achievement. Caller
// must hold s.mu.
func (s *Server) starsForAchievementLocked(a core.Achievement) {
	stars := achievementStars[a.ID]
	if stars == 0 {
		stars = 10
	}
	s.awardStarsLocked(stars, "achievement:"+string(s.pet.ID)+":"+a.ID, true, "the "+a.Name+" achievement")
	s.noteAchievementsLocked(s.pet)
}

// starsForGoalsLocked awards Home Stars when all of today's goals are
// done. Caller must hold s.mu.
func (s *Server) starsForGoalsLocked() {
	goals, _ := s.pet.DailyGoals()
	for _, g := range goals {
		if !g.Done {
			return
		}
	}
	if len(goals) > 0 {
		day := time.Now().Format("2006-01-02")
		s.awardStarsLocked(starsGoals, "goals:"+string(s.pet.ID)+":"+day, true, "all of today's goals")
	}
}

// houseTickLocked counts play towards the day's visit. Caller must hold
// s.mu.
func (s *Server) houseTickLocked(now time.Time, playing bool, gameSeconds float64) {
	if s.house == nil {
		return
	}
	if stars, why := s.house.tick(now, playing, gameSeconds/secondsPerGameDay); stars > 0 && s.pet != nil {
		s.events.Emit(core.GameEvent{Type: core.EventAchievement, PetID: s.pet.ID, Good: true,
			Message: fmt.Sprintf("⭐ +%d Home Stars for %s.", stars, why)})
	}
}

// handleHome handles GET /api/home - Home Stars, house upgrades and looks,
// and the household's settings
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	if s.house == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No household"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.house.status(time.Now())})
}

// handleHomeBuy handles POST /api/home/buy with {"id"}: a house upgrade or
// look for Home Stars, or {"filter": true}: a new air filter
func (s *Server) handleHomeBuy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		Filter bool   `json:"filter"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if s.house == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No household"})
		return
	}
	var err error
	if req.Filter {
		err = s.house.buyFilter()
	} else {
		err = s.house.buy(req.ID)
	}
	if err != nil {
		writeHomeError(w, err)
		return
	}
	s.markActive()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.house.status(time.Now())})
}

// handleHomeLook handles POST /api/home/look with {"slot", "id"}: puts up a
// house look the player owns ("" takes it down)
func (s *Server) handleHomeLook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slot string `json:"slot"`
		ID   string `json:"id"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if s.house == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No household"})
		return
	}
	if err := s.house.setLook(req.Slot, req.ID); err != nil {
		writeHomeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.house.status(time.Now())})
}

// handleSettings handles GET /api/settings and POST /api/settings with
// {"hemisphere": "northern" or "southern"}: the half of the world the
// seasons follow
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if s.house == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No household"})
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Hemisphere environment.Hemisphere `json:"hemisphere"`
		}
		if !decodePost(w, r, &req) {
			return
		}
		if err := s.house.setHemisphere(req.Hemisphere); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Choose the northern or southern hemisphere"})
			return
		}
	} else if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	d := s.house.snapshot()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"hemisphere":     d.Hemisphere,
		"hemisphere_set": !d.SettingsTime.IsZero(),
		"season":         core.RealSeason().String(),
	}})
}

// writeHomeError reports why a house purchase or change failed
func writeHomeError(w http.ResponseWriter, err error) {
	status := http.StatusConflict
	message := "Something went wrong"
	switch {
	case errors.Is(err, errNotEnoughStars):
		message = "Not enough Home Stars"
	case errors.Is(err, errOwnedHome):
		message = "Already bought"
	case errors.Is(err, errFilterNotDue):
		message = "The filter is still fresh"
	case errors.Is(err, errUnknownHome):
		status, message = http.StatusBadRequest, "Unknown house upgrade or look"
	default:
		log.Printf("Household: %v", err)
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, APIResponse{Success: false, Error: message})
}

// markActive records that the player is present
func (s *Server) markActive() {
	s.mu.Lock()
	s.markActiveLocked(time.Now())
	s.mu.Unlock()
}
