package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Home happenings: events at home with choices (internal/core/happenings.go).
// They come only while the player is using the page; the page shows the
// one waiting in the status and sends the choice here.

// happeningPresence is how recently the player must have used the page for
// a happening to start
const happeningPresence = 2 * time.Minute

// checkHappeningLocked may start a happening for the active pet while the
// player is present. Caller must hold s.mu.
func (s *Server) checkHappeningLocked(now time.Time) {
	if s.pet == nil || now.Sub(s.lastActivity) > happeningPresence {
		return
	}
	s.pet.CheckHappening()
}

// handleHappening handles POST /api/happening with {"choice"}: a choice in
// the happening waiting for one, or "" to close it without choosing
func (s *Server) handleHappening(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Choice string `json:"choice"`
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}
	}
	s.withActivePet(w, r, func() {
		if req.Choice == "" {
			s.pet.DismissHappening()
			writeJSON(w, http.StatusOK, APIResponse{Success: true})
			return
		}
		result, err := s.pet.ChooseHappening(req.Choice)
		switch {
		case errors.Is(err, core.ErrNoHappening):
			writeJSON(w, http.StatusConflict, APIResponse{Success: false, Error: "That has passed."})
		case err != nil:
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown choice"})
		default:
			writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
		}
	})
}
