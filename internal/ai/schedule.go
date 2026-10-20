package ai

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// OwnerSchedule learns when the owner is usually around: for each hour of
// the week, how often they have been present, with older weeks counting
// less (each week's presence counts scheduleDecay as much as the week
// after it). From it the game can tell when the owner usually plays,
// when they usually stop and when they are likely back.
type OwnerSchedule struct {
	mu sync.Mutex

	// Weights has a weight per hour of the week (Sunday 0:00 is 0)
	Weights [7 * 24]float64 `json:"weights"`
	// Updated is when the weights were last decayed
	Updated time.Time `json:"updated"`
	// LastSeen is when presence was last recorded
	LastSeen time.Time `json:"last_seen"`
	// Visits counts the separate visits recorded
	Visits int `json:"visits"`
	// Off stops learning: presence isn't recorded while it is set
	Off bool `json:"off"`
}

// Schedule settings
const (
	scheduleDecay      = 0.85             // Per week
	presenceStep       = 10 * time.Minute // Presence counts at most once per step
	newVisitGap        = time.Hour        // Absence that starts a new visit
	usualLevel         = 0.35             // Share of the busiest hour that makes an hour "usual"
	hoursPerWeek       = 7 * 24
	minScheduleWeights = 6.0 // Weight needed before predictions are made
)

// MarshalJSON encodes the schedule, holding its lock so it isn't changed
// mid-encoding
func (s *OwnerSchedule) MarshalJSON() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	type plain struct {
		Weights  [hoursPerWeek]float64 `json:"weights"`
		Updated  time.Time             `json:"updated"`
		LastSeen time.Time             `json:"last_seen"`
		Visits   int                   `json:"visits"`
		Off      bool                  `json:"off"`
	}
	return json.Marshal(plain{s.Weights, s.Updated, s.LastSeen, s.Visits, s.Off})
}

// hourOfWeek returns t's hour of the week
func hourOfWeek(t time.Time) int {
	return int(t.Weekday())*24 + t.Hour()
}

// decayLocked ages the weights to now. Caller must hold s.mu.
func (s *OwnerSchedule) decayLocked(now time.Time) {
	if s.Updated.IsZero() {
		s.Updated = now
		return
	}
	weeks := now.Sub(s.Updated).Hours() / hoursPerWeek
	if weeks <= 0 {
		return
	}
	factor := math.Pow(scheduleDecay, weeks)
	for i := range s.Weights {
		s.Weights[i] *= factor
	}
	s.Updated = now
}

// RecordPresence records that the owner is around at now. Presence is
// counted at most once every 10 minutes, so the weights measure time
// spent rather than clicks.
func (s *OwnerSchedule) RecordPresence(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Off {
		return
	}
	if !s.LastSeen.IsZero() && now.Sub(s.LastSeen) < presenceStep {
		return
	}
	if s.LastSeen.IsZero() || now.Sub(s.LastSeen) >= newVisitGap {
		s.Visits++
	}
	s.decayLocked(now)
	s.Weights[hourOfWeek(now)] += presenceStep.Hours()
	s.LastSeen = now
}

// SetLearning turns learning on or off. Turning it off keeps what was
// learned until Clear.
func (s *OwnerSchedule) SetLearning(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Off = !on
}

// Learning reports whether presence is being recorded
func (s *OwnerSchedule) Learning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.Off
}

// Clear forgets everything learned (but not whether learning is on)
func (s *OwnerSchedule) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Weights, s.Updated, s.LastSeen, s.Visits = [hoursPerWeek]float64{}, time.Time{}, time.Time{}, 0
}

// readyLocked reports whether there is enough to predict from. Caller
// must hold s.mu.
func (s *OwnerSchedule) readyLocked() bool {
	total := 0.0
	for _, w := range s.Weights {
		total += w
	}
	return total >= minScheduleWeights*presenceStep.Hours() && s.Visits >= 3
}

// usualLocked reports whether an hour of the week is one the owner is
// usually around. Caller must hold s.mu.
func (s *OwnerSchedule) usualLocked(hour int) bool {
	busiest := 0.0
	for _, w := range s.Weights {
		busiest = max(busiest, w)
	}
	return busiest > 0 && s.Weights[hour%hoursPerWeek] >= usualLevel*busiest
}

// UsualEnd returns when the owner usually stops if they are around at
// now: the end of the run of usual hours that includes now. ok is false
// if there isn't enough to predict from or now isn't a usual time.
func (s *OwnerSchedule) UsualEnd(now time.Time) (end time.Time, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.readyLocked() {
		return time.Time{}, false
	}
	hour := hourOfWeek(now)
	if !s.usualLocked(hour) {
		return time.Time{}, false
	}
	start := now.Truncate(time.Hour)
	for i := 1; i < hoursPerWeek; i++ {
		if !s.usualLocked(hour + i) {
			return start.Add(time.Duration(i) * time.Hour), true
		}
	}
	return time.Time{}, false // Always around
}

// NextVisit returns when the owner is next likely to be around after now:
// the start of the next usual hour after a gap. ok is false if there isn't
// enough to predict from.
func (s *OwnerSchedule) NextVisit(now time.Time) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.readyLocked() {
		return time.Time{}, false
	}
	hour := hourOfWeek(now)
	start := now.Truncate(time.Hour)
	inUsual := s.usualLocked(hour)
	for i := 1; i <= hoursPerWeek; i++ {
		usual := s.usualLocked(hour + i)
		if usual && !inUsual {
			return start.Add(time.Duration(i) * time.Hour), true
		}
		inUsual = usual
	}
	return time.Time{}, false
}

// dayParts name parts of the day by hour
var dayParts = []struct {
	from int
	name string
}{{0, "night"}, {5, "early morning"}, {9, "morning"}, {12, "afternoon"}, {17, "evening"}, {21, "late evening"}}

// dayPart names the part of the day an hour falls in
func dayPart(hour int) string {
	name := dayParts[0].name
	for _, p := range dayParts {
		if hour >= p.from {
			name = p.name
		}
	}
	return name
}

// Summary describes when the owner usually plays, such as "weekday
// evenings and weekend mornings", or "" if it isn't known yet
func (s *OwnerSchedule) Summary() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.readyLocked() {
		return ""
	}
	// Weight per part of the day, for weekdays and weekends
	type key struct {
		weekend bool
		part    string
	}
	weights := make(map[key]float64)
	for hour, w := range s.Weights {
		if !s.usualLocked(hour) {
			continue
		}
		day := hour / 24
		weights[key{day == 0 || day == 6, dayPart(hour % 24)}] += w
	}
	keys := make([]key, 0, len(weights))
	for k := range weights {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return weights[keys[i]] > weights[keys[j]] })
	if len(keys) > 3 {
		keys = keys[:3]
	}
	var parts []string
	for _, k := range keys {
		days := "weekday"
		if k.weekend {
			days = "weekend"
		}
		parts = append(parts, fmt.Sprintf("%s %ss", days, k.part))
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}
