package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// interactionData is the response data for anything that cares for the
// pet: what changed, how it reacted and the points it earned. Caller must
// hold s.mu.
func (s *Server) interactionData(result core.InteractionResult) map[string]interface{} {
	return map[string]interface{}{
		"message":           "Interaction successful",
		"points_earned":     result.PointsEarned,
		"friendship_points": s.pet.FriendshipPoints,
		"changes":           result.Changes,
		"boosted_by":        result.BoostedBy,
		"reaction":          result.Reaction,
		"reaction_text":     result.ReactionText,
	}
}

// writeActivityError reports why the pet cannot do an activity
func writeActivityError(w http.ResponseWriter, name string, err error) {
	status, message := http.StatusConflict, ""
	switch {
	case errors.Is(err, core.ErrPetNotAvailable):
		message = name + " is hibernating. Revive it first."
	case errors.Is(err, core.ErrExploring):
		message = name + " is out exploring and will be back soon."
	case errors.Is(err, core.ErrTooTired):
		message = name + " is too tired. Let it rest or give it some food first."
	case errors.Is(err, core.ErrNeedsCare):
		message = name + " needs food, water and energy before an adventure."
	case errors.Is(err, core.ErrTrickNotKnown):
		message = name + " hasn't learned that trick yet."
	case errors.Is(err, core.ErrNoGame):
		message = "Start a game of hide and seek first."
	case errors.Is(err, core.ErrUnknownTrick):
		status, message = http.StatusBadRequest, "Unknown trick"
	case errors.Is(err, core.ErrInvalidSpot):
		status, message = http.StatusBadRequest, "Unknown hiding spot"
	default:
		status, message = http.StatusInternalServerError, "Something went wrong"
	}
	writeJSON(w, status, APIResponse{Success: false, Error: message})
}

// withActivePet runs fn with s.mu held and the player marked present, or
// reports that there is no pet. The request must be a POST.
func (s *Server) withActivePet(w http.ResponseWriter, r *http.Request, fn func()) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pet == nil || !s.pet.IsAlive() {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Pet is not available"})
		return
	}
	s.markActiveLocked(time.Now())
	fn()
}

// trickData is the response data for training or performing a trick.
// Caller must hold s.mu.
func (s *Server) trickData(result core.TrickResult) map[string]interface{} {
	data := s.interactionData(result.InteractionResult)
	data["text"] = result.Text
	data["trick"] = result.Trick
	data["skill"] = result.Skill
	data["just_learned"] = result.JustLearned
	return data
}

// handleTrick handles POST /api/train and /api/perform with {"trick": id}
func (s *Server) handleTrick(perform bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Trick string `json:"trick"`
		}
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
				return
			}
		}

		s.withActivePet(w, r, func() {
			train := s.pet.TrainTrick
			if perform {
				train = s.pet.PerformTrick
			}
			result, err := train(req.Trick)
			if err != nil {
				writeActivityError(w, s.pet.Name, err)
				return
			}
			writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.trickData(result)})
		})
	}
}

// handleHideSeekStart handles POST /api/hideseek/start - the pet hides
func (s *Server) handleHideSeekStart(w http.ResponseWriter, r *http.Request) {
	s.withActivePet(w, r, func() {
		if err := s.pet.StartHideAndSeek(); err != nil {
			writeActivityError(w, s.pet.Name, err)
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
			"spots": core.HidingSpots(),
			"text":  s.pet.Name + " is hiding! Where could it be?",
		}})
	})
}

// handleHideSeekGuess handles POST /api/hideseek/guess with {"spot": index}
func (s *Server) handleHideSeekGuess(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Spot int `json:"spot"`
	}
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid request"})
			return
		}
	}

	s.withActivePet(w, r, func() {
		result, err := s.pet.GuessHidingSpot(req.Spot)
		if err != nil {
			writeActivityError(w, s.pet.Name, err)
			return
		}

		data := map[string]interface{}{}
		if result.Over {
			data = s.interactionData(result.InteractionResult)
		}
		data["found"] = result.Found
		data["over"] = result.Over
		data["guesses_left"] = result.GuessesLeft
		data["spot"] = result.Spot
		data["bonus_points"] = result.BonusPoints
		data["friendship_points"] = s.pet.FriendshipPoints
		data["text"] = result.Text
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
	})
}

// handleExplore handles POST /api/explore - sends the pet exploring
func (s *Server) handleExplore(w http.ResponseWriter, r *http.Request) {
	s.withActivePet(w, r, func() {
		if err := s.pet.SendExploring(); err != nil {
			writeActivityError(w, s.pet.Name, err)
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
			"text":              s.pet.Name + " sets off to explore!",
			"exploring_seconds": gameDaysToSeconds(s.pet.ExploringRemaining()),
		}})
	})
}

// recentAdventureDays is how long (in game days) a finished trip is
// reported in the status, so the page can tell the player about it
const recentAdventureDays = 5.0

// gameDaysToSeconds converts game time to real seconds at the server's
// speed of one game day per real minute
func gameDaysToSeconds(days float64) int {
	return int(days*60 + 0.5)
}
