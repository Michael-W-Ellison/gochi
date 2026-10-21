package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Pet shows: the shows the active pet can enter, and entering one. Home
// Stars won at a show go to the household.

// handleShows handles GET /api/shows - the shows the active pet can enter,
// and the ribbons and titles it has won
func (s *Server) handleShows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	s.mu.RLock()
	pet := s.pet
	s.mu.RUnlock()
	if pet == nil {
		writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No pet found"})
		return
	}
	ribbons, titles := pet.ShowRecord()
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"shows": pet.Shows(), "ribbons": ribbons, "titles": titles, "tricks": pet.Tricks(),
	}})
}

// handleEnterShow handles POST /api/shows/enter with {"show", "level",
// "routine": [trick IDs]}
func (s *Server) handleEnterShow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Show    string   `json:"show"`
		Level   int      `json:"level"`
		Routine []string `json:"routine"`
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Routine) > 10 {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}
	}
	s.withActivePet(w, r, func() {
		result, err := s.pet.EnterShow(req.Show, req.Level, req.Routine)
		if err != nil {
			var refused core.RefusedError
			switch {
			case errors.As(err, &refused):
				writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: refused.Reason})
			case errors.Is(err, core.ErrUnknownShow), errors.Is(err, core.ErrShowSpecies):
				writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown show"})
			default:
				writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: capitalize(err.Error()) + "."})
			}
			return
		}
		if result.Stars > 0 {
			s.awardStarsLocked(result.Stars, "show", false, "a show ribbon")
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
	})
}
