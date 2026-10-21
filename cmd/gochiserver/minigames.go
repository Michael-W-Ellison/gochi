package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Mini-games: the games for the active pet, and finishing one. The page
// plays the game and reports how well it went (0 to 1); there are no
// scores.

// handleMiniGames handles GET /api/games - the mini-games for the active
// pet
func (s *Server) handleMiniGames(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: pet.MiniGames()})
}

// handleFinishMiniGame handles POST /api/games/finish with {"game",
// "quality"}: how well a game went, from 0 to 1
func (s *Server) handleFinishMiniGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Game    string  `json:"game"`
		Quality float64 `json:"quality"`
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}
	}
	s.withActivePet(w, r, func() {
		result, err := s.pet.FinishMiniGame(req.Game, req.Quality)
		if errors.Is(err, core.ErrUnknownGame) {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Unknown game"})
			return
		}
		if err != nil || !result.Applied {
			s.writeCareResult(w, result.InteractionResult, err)
			return
		}
		data := s.interactionData(result.InteractionResult)
		data["text"], data["enjoyment"], data["game_lesson"] = result.Text, result.Enjoyment, result.Lesson
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
	})
}
