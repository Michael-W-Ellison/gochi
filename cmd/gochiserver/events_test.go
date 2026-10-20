package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// fetchEvents asks /api/events for the active pet's events after seq
func fetchEvents(t *testing.T, s *Server, after string) (events []core.GameEvent, last int64) {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleEvents(rec, httptest.NewRequest(http.MethodGet, "/api/events?after="+after, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/events: %d %s", rec.Code, rec.Body)
	}
	var resp struct {
		Data struct {
			Events []struct {
				Seq     int64  `json:"seq"`
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"events"`
			Last int64 `json:"last"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.Events == nil {
		t.Error("Events should be an empty list, not null")
	}
	for _, e := range resp.Data.Events {
		events = append(events, core.GameEvent{Seq: e.Seq, Message: e.Message})
		if e.Type != "PetHungry" {
			t.Errorf("Unexpected event type %q", e.Type)
		}
	}
	return events, resp.Data.Last
}

func TestTickRecordsEvents(t *testing.T) {
	s := newTestServer(t)
	s.awayTimeout = 0
	s.pet = core.NewDigitalPet("Mochi", "web_user")

	if events, _ := fetchEvents(t, s, ""); len(events) != 0 {
		t.Fatalf("A new pet should have no events, got %+v", events)
	}

	now := time.Now()
	s.tick(now)
	s.pet.Biology.Vitals.Nutrition = 0.41 // About to be hungry
	s.tick(now.Add(30 * time.Second))     // Half a game day

	events, last := fetchEvents(t, s, "0")
	if len(events) != 1 || events[0].Message != "Mochi is hungry." {
		t.Fatalf("Expected a hunger event, got %+v", events)
	}
	if again, _ := fetchEvents(t, s, "1"); len(again) != 0 || last != 1 {
		t.Errorf("Nothing should be new after event %d, got %+v", last, again)
	}

	// Another pet doesn't see Mochi's events
	s.pet = core.NewDigitalPet("Biscuit", "web_user")
	if events, _ := fetchEvents(t, s, "0"); len(events) != 0 {
		t.Errorf("Biscuit should not see Mochi's events, got %+v", events)
	}
}

func TestEventsRejectsBadNumber(t *testing.T) {
	s := newTestServer(t)
	s.pet = core.NewDigitalPet("Mochi", "web_user")
	rec := httptest.NewRecorder()
	s.handleEvents(rec, httptest.NewRequest(http.MethodGet, "/api/events?after=x", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rec.Code)
	}
}
