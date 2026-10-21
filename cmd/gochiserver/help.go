package main

import (
	"net/http"
	"slices"
)

// Help: the first-run tour and the first-time tips are written in the
// page; the household records which have been shown, so a player with
// several pets sees each once. Tips are named from a fixed list.

// tipIDs are the tour and the tips
var tipIDs = []string{"tour", "shop", "breeding", "shows", "collection", "house", "family", "games", "explore"}

// handleTipSeen handles POST /api/tips/seen with {"id"}: a tip from tipIDs
// has been shown
func (s *Server) handleTipSeen(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !decodePost(w, r, &req) {
		return
	}
	if !slices.Contains(tipIDs, req.ID) {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown tip"})
		return
	}
	if s.house != nil {
		if err := s.house.seeTip(req.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Could not save"})
			return
		}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true})
}
