package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// The owner's schedule, predictive care and daily goals. The server learns
// when the player is usually around from when they use the page, and uses
// it to look ahead: the page shows what the pet will need before the
// player usually stops playing.

// Forecast horizons, in real time
const (
	defaultForecastHorizon = 30 * time.Minute // When the usual end of play isn't known
	minForecastHorizon     = 10 * time.Minute
	maxForecastHorizon     = 2 * time.Hour
)

// forecastStatus is a NeedForecast with the real time until it
type forecastStatus struct {
	core.NeedForecast
	InSeconds int `json:"in_seconds"` // Real seconds from now at the current game speed
}

// loadOwnerSchedule reads the owner's schedule from owner.json in dir, or
// starts a new one
func loadOwnerSchedule(dir string) (*ai.OwnerSchedule, string, error) {
	path := filepath.Join(dir, "owner.json")
	schedule := &ai.OwnerSchedule{}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, "", err
	default:
		if err := json.Unmarshal(data, schedule); err != nil {
			return nil, "", fmt.Errorf("%s: %w", path, err)
		}
	}
	return schedule, path, nil
}

// saveOwner writes the owner's schedule, if the server keeps one
func (s *Server) saveOwner() error {
	if s.owner == nil || s.ownerPath == "" {
		return nil
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	data, err := json.Marshal(s.owner)
	if err != nil {
		return err
	}
	tmp := s.ownerPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.ownerPath)
}

// forecastHorizon returns how far ahead to look: until the player usually
// stops playing, if that is known, otherwise 30 minutes
func (s *Server) forecastHorizon(now time.Time) time.Duration {
	if s.owner != nil {
		if end, ok := s.owner.UsualEnd(now); ok {
			return min(max(end.Sub(now), minForecastHorizon), maxForecastHorizon)
		}
	}
	return defaultForecastHorizon
}

// forecastsLocked returns what the active pet will ask for before the
// player usually stops playing. Caller must hold s.mu.
func (s *Server) forecastsLocked(now time.Time) []forecastStatus {
	forecasts := []forecastStatus{}
	gamePerReal := s.clock.GetTimeScaleMultiplier() // Game seconds per real second
	if s.pet == nil || gamePerReal <= 0 {
		return forecasts
	}
	horizon := s.forecastHorizon(now).Seconds() * gamePerReal / secondsPerGameDay
	for _, f := range s.pet.Forecast(horizon) {
		forecasts = append(forecasts, forecastStatus{f, int(f.InDays*secondsPerGameDay/gamePerReal + 0.5)})
	}
	return forecasts
}

// announceGoalsLocked records goals the active pet's owner has newly
// completed as events. Caller must hold s.mu.
func (s *Server) announceGoalsLocked() {
	for _, g := range s.pet.TakeCompletedGoals() {
		s.events.Emit(core.GameEvent{Type: core.EventGoal, PetID: s.pet.ID, Good: true,
			Message: fmt.Sprintf("🎯 Goal reached: %s %s. +%d Friendship Points.", g.Icon, g.Text, g.Reward)})
	}
}
