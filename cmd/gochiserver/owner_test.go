package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestStatusShowsGoalsAndForecasts(t *testing.T) {
	s := newTestServer(t)
	s.owner = &ai.OwnerSchedule{}
	if err := s.createPet("Mochi", core.SpeciesCat, false, false); err != nil {
		t.Fatal(err)
	}
	s.pet.Biology.Vitals.Nutrition = 0.4 // Hungry soon

	status := getJSON(t, s.handleStatus, "/api/status?active=1")
	if goals, _ := status["goals"].([]interface{}); len(goals) != 3 {
		t.Errorf("Status should list three goals, got %v", status["goals"])
	}
	if level, _ := status["goal_level"].(float64); level != 1 {
		t.Errorf("A new pet's goal level should be 1, got %v", status["goal_level"])
	}
	forecasts, _ := status["forecasts"].([]interface{})
	found := false
	for _, f := range forecasts {
		f := f.(map[string]interface{})
		if f["action"] == "feed" {
			found = true
			if secs, _ := f["in_seconds"].(float64); secs <= 0 || secs > defaultForecastHorizon.Seconds() {
				t.Errorf("Hunger should be forecast within the horizon, got %v seconds", f["in_seconds"])
			}
		}
	}
	if !found {
		t.Errorf("Status should forecast hunger, got %v", status["forecasts"])
	}
	if status["play_schedule"] != "" {
		t.Errorf("The play schedule shouldn't be known yet, got %v", status["play_schedule"])
	}
	if s.owner.Visits != 1 {
		t.Errorf("Using the page should be recorded as a visit, got %d", s.owner.Visits)
	}
}

func TestCompletedGoalIsAnnounced(t *testing.T) {
	s := newTestServer(t)
	if err := s.createPet("Mochi", core.SpeciesCat, false, false); err != nil {
		t.Fatal(err)
	}
	s.pet.Goals = &core.GoalBoard{Day: time.Now().Format("2006-01-02"), Level: 1, Goals: []core.Goal{
		{Kind: core.GoalCare, Care: types.InteractionDrinking, Icon: "💧", Text: "Give Mochi fresh water 1 times", Target: 1, Reward: 5},
	}}
	if rec := post(t, s.handleInteract, `{"action":"drink"}`); rec.Code != http.StatusOK {
		t.Fatalf("Drinking: %d %s", rec.Code, rec.Body)
	}
	s.tick(time.Now())
	events, _ := s.events.Since(0, s.pet.ID)
	if len(events) != 1 || events[0].Type != core.EventGoal || !strings.Contains(events[0].Message, "+5 Friendship Points") {
		t.Errorf("Completing a goal should be announced once, got %+v", events)
	}
}

func TestOwnerScheduleIsSaved(t *testing.T) {
	dir := t.TempDir()
	owner, path, err := loadOwnerSchedule(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := newTestServer(t)
	s.owner, s.ownerPath = owner, path
	start := time.Date(2026, 3, 2, 19, 0, 0, 0, time.Local)
	s.markActiveLocked(start)
	s.markActiveLocked(start.Add(20 * time.Minute))
	if err := s.saveOwner(); err != nil {
		t.Fatal(err)
	}

	loaded, _, err := loadOwnerSchedule(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Visits != 1 || loaded.Weights != owner.Weights || !loaded.LastSeen.Equal(start.Add(20*time.Minute)) {
		t.Errorf("Loaded schedule %+v differs from saved %+v", loaded, owner)
	}

	// A damaged file is reported
	if err := os.WriteFile(filepath.Join(dir, "owner.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadOwnerSchedule(dir); err == nil {
		t.Error("A damaged schedule file should be an error")
	}
}
