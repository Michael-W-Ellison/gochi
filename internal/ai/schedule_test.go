package ai

import (
	"encoding/json"
	"testing"
	"time"
)

// playEvenings records an owner who plays 6-8 pm on weekdays for weeks
func playEvenings(s *OwnerSchedule, start time.Time, weeks int) {
	for d := 0; d < 7*weeks; d++ {
		day := start.AddDate(0, 0, d)
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		for m := 0; m < 120; m += 5 {
			s.RecordPresence(time.Date(day.Year(), day.Month(), day.Day(), 18, m, 0, 0, time.Local))
		}
	}
}

func TestOwnerScheduleLearnsUsualTimes(t *testing.T) {
	s := &OwnerSchedule{}
	monday := time.Date(2026, 1, 5, 0, 0, 0, 0, time.Local)
	if s.Summary() != "" {
		t.Error("Nothing is known at first")
	}
	playEvenings(s, monday, 3)

	if got := s.Summary(); got != "weekday evenings" {
		t.Errorf("Summary = %q", got)
	}
	if s.Visits != 15 {
		t.Errorf("Visits = %d, want 15", s.Visits)
	}

	// On a weekday at 6:30 pm the owner usually stops at 8 pm
	weds := time.Date(2026, 1, 28, 18, 30, 0, 0, time.Local)
	if end, ok := s.UsualEnd(weds); !ok || end.Hour() != 20 || end.Day() != 28 {
		t.Errorf("UsualEnd = %v, %v", end, ok)
	}
	// At noon it isn't a usual time; the next visit is that evening
	noon := time.Date(2026, 1, 28, 12, 0, 0, 0, time.Local)
	if _, ok := s.UsualEnd(noon); ok {
		t.Error("Noon isn't a usual time")
	}
	if next, ok := s.NextVisit(noon); !ok || next.Hour() != 18 || next.Day() != 28 {
		t.Errorf("NextVisit = %v, %v", next, ok)
	}
	// On Friday evening the next visit is Monday evening
	friday := time.Date(2026, 1, 30, 21, 0, 0, 0, time.Local)
	if next, ok := s.NextVisit(friday); !ok || next.Weekday() != time.Monday || next.Hour() != 18 {
		t.Errorf("After Friday: %v, %v", next, ok)
	}
}

func TestOwnerScheduleAdaptsToChange(t *testing.T) {
	s := &OwnerSchedule{}
	start := time.Date(2026, 1, 5, 0, 0, 0, 0, time.Local)
	playEvenings(s, start, 2)
	// Then the owner switches to mornings for many weeks
	for d := 14; d < 14+7*10; d++ {
		day := start.AddDate(0, 0, d)
		for m := 0; m < 60; m += 5 {
			s.RecordPresence(time.Date(day.Year(), day.Month(), day.Day(), 8, m, 0, 0, time.Local))
		}
	}
	if got := s.Summary(); got != "weekday early mornings and weekend early mornings" &&
		got != "weekend early mornings and weekday early mornings" {
		t.Errorf("The schedule should follow the new habit, got %q", got)
	}
}

func TestOwnerScheduleCountsTimeNotClicks(t *testing.T) {
	s := &OwnerSchedule{}
	at := time.Date(2026, 1, 5, 18, 0, 0, 0, time.Local)
	for i := 0; i < 100; i++ {
		s.RecordPresence(at.Add(time.Duration(i) * time.Second))
	}
	if w := s.Weights[hourOfWeek(at)]; w != presenceStep.Hours() {
		t.Errorf("100 clicks in a minute count as one presence, got %.3f", w)
	}

	data, _ := json.Marshal(s)
	var loaded OwnerSchedule
	if err := json.Unmarshal(data, &loaded); err != nil || loaded.Weights != s.Weights || loaded.Visits != 1 {
		t.Errorf("Save and load: %v", err)
	}
}
