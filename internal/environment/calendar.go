package environment

import (
	"sync"
	"time"
)

// Real-calendar seasons. When the calendar is in use (UseCalendar), the
// season comes from today's date and the player's hemisphere instead of
// advancing with game time, so the weather and seasonal breeding match the
// real time of year. Seasons follow whole months: in the Northern
// Hemisphere spring is March to May, summer June to August, fall
// September to November and winter December to February; the Southern
// Hemisphere's are the other way round.

// Hemisphere is the half of the world the player lives in
type Hemisphere string

const (
	Northern Hemisphere = "northern"
	Southern Hemisphere = "southern"
)

// Valid reports whether h is a hemisphere
func (h Hemisphere) Valid() bool {
	return h == Northern || h == Southern
}

// SeasonOn returns the season on a date in a hemisphere
func SeasonOn(t time.Time, h Hemisphere) Season {
	season := Season(((int(t.Month()) + 9) % 12) / 3) // March to May: spring
	if h == Southern {
		season = (season + 2) % 4
	}
	return season
}

// seasonStart returns the first day of the season a date is in
func seasonStart(t time.Time) time.Time {
	month := t.Month() - (t.Month()+9)%3 // March, June, September or December
	year := t.Year()
	if month < 1 {
		month += 12
		year--
	}
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}

// FollowDate sets the season from a date in a hemisphere
func (sc *SeasonalCycle) FollowDate(t time.Time, h Hemisphere) {
	start := seasonStart(t)
	end := start.AddDate(0, 3, 0)
	sc.CurrentSeason = SeasonOn(t, h)
	sc.Year = t.Year()
	sc.DaysPerSeason = int(end.Sub(start).Hours()/24 + 0.5)
	sc.DayOfSeason = int(t.Sub(start).Hours()/24) + 1
	sc.SeasonProgress = float64(sc.DayOfSeason) / float64(sc.DaysPerSeason)
	sc.updateTransition()
	sc.Effects = sc.calculateSeasonalEffects()
}

// The calendar in use, if any
var calendar struct {
	mu         sync.RWMutex
	on         bool
	hemisphere Hemisphere
	now        func() time.Time
}

// UseCalendar makes every environment's season follow the real date
// (from now) in a hemisphere
func UseCalendar(h Hemisphere, now func() time.Time) {
	calendar.mu.Lock()
	defer calendar.mu.Unlock()
	calendar.on, calendar.hemisphere, calendar.now = true, h, now
}

// StopCalendar makes seasons advance with game time again
func StopCalendar() {
	calendar.mu.Lock()
	defer calendar.mu.Unlock()
	calendar.on = false
}

// followCalendar sets the season from the calendar if it is in use, and
// reports whether it did
func (sc *SeasonalCycle) followCalendar() bool {
	calendar.mu.RLock()
	on, h, now := calendar.on, calendar.hemisphere, calendar.now
	calendar.mu.RUnlock()
	if on {
		sc.FollowDate(now(), h)
	}
	return on
}
