package core

import (
	"slices"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

// Settings for the whole household, set by the game server: the
// hemisphere seasons follow, which holidays a parent has turned off, and
// the house upgrades bought with Home Stars (house.go), which apply to
// every pet.
var household struct {
	mu       sync.RWMutex
	calendar bool                   // Seasons follow the real calendar
	hemi     environment.Hemisphere // Northern unless set
	holidays map[string]bool        // Holidays turned off
	allOff   bool                   // Every holiday turned off
	upgrades []string               // House upgrades owned
	filterOK bool                   // The air filtration's filter is fresh
	shelfSet bool                   // The shelf shows a whole place's set of souvenirs
}

// UseCalendar makes seasons, and the weather and breeding that follow
// them, come from the real date in a hemisphere
func UseCalendar(h environment.Hemisphere) {
	if !h.Valid() {
		h = environment.Northern
	}
	household.mu.Lock()
	household.calendar, household.hemi = true, h
	household.mu.Unlock()
	environment.UseCalendar(h, func() time.Time { return now() })
}

// Hemisphere returns the hemisphere seasons follow
func Hemisphere() environment.Hemisphere {
	household.mu.RLock()
	defer household.mu.RUnlock()
	if household.hemi == "" {
		return environment.Northern
	}
	return household.hemi
}

// RealSeason returns the real season today, for the household's hemisphere
func RealSeason() environment.Season {
	return environment.SeasonOn(now(), Hemisphere())
}

// SetHolidays sets which holidays a parent has turned off
func SetHolidays(allOff bool, off []string) {
	household.mu.Lock()
	defer household.mu.Unlock()
	household.allOff = allOff
	household.holidays = make(map[string]bool, len(off))
	for _, id := range off {
		household.holidays[id] = true
	}
}

// HolidayOn reports whether a holiday is turned on
func HolidayOn(id string) bool {
	household.mu.RLock()
	defer household.mu.RUnlock()
	return !household.allOff && !household.holidays[id]
}

// SetHouse sets the house upgrades owned and whether the air filtration's
// filter is fresh
func SetHouse(upgrades []string, filterFresh bool) {
	household.mu.Lock()
	defer household.mu.Unlock()
	household.upgrades = slices.Clone(upgrades)
	household.filterOK = filterFresh
}

// SetShelfSet sets whether the house's shelf shows a whole place's set of
// souvenirs, which cheers every pet up a little (as a decoration)
func SetShelfSet(on bool) {
	household.mu.Lock()
	defer household.mu.Unlock()
	household.shelfSet = on
}

// shelfSetShown reports whether the shelf shows a whole set
func shelfSetShown() bool {
	household.mu.RLock()
	defer household.mu.RUnlock()
	return household.shelfSet
}

// houseUpgrades returns the house upgrades owned and whether the filter is
// fresh
func houseUpgrades() ([]string, bool) {
	household.mu.RLock()
	defer household.mu.RUnlock()
	return household.upgrades, household.filterOK
}
