package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/internal/hub"
)

// The household: what belongs to the player rather than a pet. Home
// Stars are earned by playing (trips, new places, achievements, a day's
// goals, daily visits) and spent on house upgrades and house cosmetics,
// which apply to every pet. Household settings are the hemisphere seasons
// follow and the holidays a parent has turned off. It is all kept in
// household.json, which only the player's account can read, and goes to
// the player's other devices through the cloud hub, sealed like a pet's
// save.
//
// Home Stars are kept as a ledger: each earning and spending has an ID,
// and devices merge by taking every entry from both, so stars earned on
// two devices at once both count and nothing is counted twice. Earnings
// that can happen only once (an achievement, a day's visit) have IDs made
// from what they were for, so two devices earning them count once.

// Home Stars for each way of earning them
const (
	starsTrip      = 3
	starsNewPlace  = 10
	starsGoals     = 5  // All three of a day's goals
	starsVisit     = 5  // A day with at least visitPlay of play
	starsStreak7   = 20 // 7 daily visits in a row
	starsStreak30  = 60 // 30 daily visits in a row
	visitPlay      = 10 * time.Minute
	maxTickElapsed = 5 * time.Second // Longer gaps between ticks don't count as play
)

// achievementStars are the Home Stars each achievement earns: 10, 25 for
// the harder ones, 50 for the rarest
var achievementStars = map[string]int{
	"first_trick": 10, "first_treasure": 10, "grown_up": 10,
	"all_tricks": 25, "crystal": 25, "senior": 25, "long_life": 25, "best_friends": 25, "family": 25, "well_equipped": 25,
	"explorer": 50, "devoted": 50, "daily_carer": 25, "nurse": 25, "attentive": 50,
}

// ledgerEntry is one earning (positive) or spending (negative) of Home Stars
type ledgerEntry struct {
	ID    string    `json:"id"`
	Time  time.Time `json:"time"`
	Stars int       `json:"stars"`
	What  string    `json:"what"` // e.g. "trip", "upgrade:playroom", "look:wall_mint"
}

// householdData is household.json
type householdData struct {
	Ledger []ledgerEntry `json:"ledger"`

	// Settings, and when they last changed (the newer wins when devices merge)
	Hemisphere     environment.Hemisphere `json:"hemisphere"`
	HolidaysOff    []string               `json:"holidays_off"`
	AllHolidaysOff bool                   `json:"all_holidays_off"`
	SoundOff       bool                   `json:"sound_off"` // A parent has turned the game's sounds off
	Look           map[string]string      `json:"look"`      // House cosmetics in use, by slot
	SettingsTime   time.Time              `json:"settings_time"`

	// Daily visits: the day being counted and how long it has been
	// played, the last day that earned its stars, and the streak
	VisitDay     string  `json:"visit_day"`
	VisitSeconds float64 `json:"visit_seconds"`
	PaidDay      string  `json:"paid_day"`
	Streak       int     `json:"streak"`

	// Game days played, which time the air filtration's filter, and when
	// it runs out
	GameDays    float64 `json:"game_days"`
	FilterUntil float64 `json:"filter_until"`

	// The achievements any pet has earned, which unlock breeds, and
	// whether the pets saved before they were recorded here were looked at
	Achieved        []string `json:"achieved"`
	AchievedScanned bool     `json:"achieved_scanned"`

	// Souvenirs any pet has found, whether the saved pets were looked at
	// for theirs, and the ones on the shelf (a setting: the newer wins when
	// devices merge)
	Collected        []string `json:"collected"`
	CollectedScanned bool     `json:"collected_scanned"`
	Shelf            []string `json:"shelf"`

	// The tour and first-time tips that have been shown (help.go)
	TipsSeen []string `json:"tips_seen"`
}

// household is the player's household
type household struct {
	mu   sync.Mutex
	path string // "" keeps it in memory only
	data householdData

	lastTick time.Time // The last tick counted as play (not saved)
}

// householdFileName is the household file in the data directory
const householdFileName = "household.json"

// loadHousehold reads household.json from dir ("" for memory only)
func loadHousehold(dir string) (*household, error) {
	h := &household{}
	if dir != "" {
		h.path = filepath.Join(dir, householdFileName)
		contents, err := os.ReadFile(h.path)
		switch {
		case errors.Is(err, os.ErrNotExist):
		case err != nil:
			return nil, err
		default:
			if err := json.Unmarshal(contents, &h.data); err != nil {
				return nil, fmt.Errorf("%s: %w", h.path, err)
			}
		}
	}
	h.clean()
	h.apply()
	return h, nil
}

// clean puts settings that aren't valid back to their defaults
func (h *household) clean() {
	if !h.data.Hemisphere.Valid() {
		h.data.Hemisphere = environment.Northern
	}
	if h.data.Look == nil {
		h.data.Look = map[string]string{}
	}
	for slot, id := range h.data.Look {
		if l, ok := core.FindHouseLook(id); !ok || l.Slot != slot || !h.ownsLocked("look:"+id) {
			delete(h.data.Look, slot)
		}
	}
	h.data.Streak = max(h.data.Streak, 0)
	h.data.Collected = slices.DeleteFunc(h.data.Collected, func(id string) bool {
		_, ok := core.FindSouvenir(id)
		return !ok
	})
	h.data.Shelf = validShelf(h.data.Shelf, h.data.Collected)
	h.data.TipsSeen = slices.DeleteFunc(h.data.TipsSeen, func(id string) bool { return !slices.Contains(tipIDs, id) })
}

// apply tells the game the household's settings and upgrades
func (h *household) apply() {
	core.UseCalendar(h.data.Hemisphere)
	core.SetHolidays(h.data.AllHolidaysOff, h.data.HolidaysOff)
	core.SetHouse(h.upgradesLocked(), h.data.FilterUntil > h.data.GameDays)
	core.SetShelfSet(shelfSetPlace(h.data.Shelf) != "")
}

// saveLocked writes household.json. Caller must hold h.mu.
func (h *household) saveLocked() error {
	if h.path == "" {
		return nil
	}
	contents, err := json.Marshal(h.data)
	if err != nil {
		return err
	}
	tmp := h.path + ".tmp"
	if err := os.WriteFile(tmp, contents, 0o600); err != nil {
		os.Remove(tmp) // Cut short (a full disk)
		return err
	}
	return os.Rename(tmp, h.path)
}

// save writes household.json
func (h *household) save() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.saveLocked()
}

// randomID returns a random ledger entry ID
func randomID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// onceID returns the ledger ID for an earning that can happen only once
func onceID(what string) string {
	sum := sha256.Sum256([]byte("once:" + what))
	return hex.EncodeToString(sum[:12])
}

// balanceLocked returns the Home Stars. Caller must hold h.mu.
func (h *household) balanceLocked() int {
	total := 0
	for _, e := range h.data.Ledger {
		total += e.Stars
	}
	return total
}

// hasLocked reports whether the ledger has an entry. Caller must hold h.mu.
func (h *household) hasLocked(id string) bool {
	return slices.ContainsFunc(h.data.Ledger, func(e ledgerEntry) bool { return e.ID == id })
}

// ownsLocked reports whether something was bought: an upgrade
// ("upgrade:<id>") or a look ("look:<id>"). Caller must hold h.mu.
func (h *household) ownsLocked(what string) bool {
	return h.hasLocked(onceID(what))
}

// upgradesLocked returns the upgrades owned. Caller must hold h.mu.
func (h *household) upgradesLocked() []string {
	var owned []string
	for _, u := range core.HouseUpgrades() {
		if h.ownsLocked("upgrade:" + u.ID) {
			owned = append(owned, u.ID)
		}
	}
	return owned
}

// earnLocked adds Home Stars, once for a once-only earning (once is
// true). It reports whether they were added. Caller must hold h.mu.
func (h *household) earnLocked(stars int, what string, once bool) bool {
	id := randomID()
	if once {
		id = onceID(what)
		if h.hasLocked(id) {
			return false
		}
	}
	h.data.Ledger = append(h.data.Ledger, ledgerEntry{ID: id, Time: time.Now(), Stars: stars, What: what})
	return true
}

// earn adds Home Stars and saves; it returns the stars added (0 if a
// once-only earning was already counted)
func (h *household) earn(stars int, what string, once bool) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.earnLocked(stars, what, once) {
		return 0
	}
	h.saveLocked()
	return stars
}

// Errors buying for the house
var (
	errNotEnoughStars = errors.New("not enough Home Stars")
	errUnknownHome    = errors.New("unknown house upgrade or look")
	errOwnedHome      = errors.New("already bought")
	errFilterNotDue   = errors.New("the filter is still fresh")
)

// buy spends Home Stars on a house upgrade or look
func (h *household) buy(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	what, price := "", 0
	if u, ok := core.FindUpgrade(id); ok {
		what, price = "upgrade:"+id, u.Price
	} else if l, ok := core.FindHouseLook(id); ok {
		what, price = "look:"+id, l.Price
	} else {
		return errUnknownHome
	}
	switch {
	case h.ownsLocked(what):
		return errOwnedHome
	case h.balanceLocked() < price:
		return errNotEnoughStars
	}
	h.earnLocked(-price, what, true)
	if id == "air_filtration" {
		h.data.FilterUntil = h.data.GameDays + core.AirFilterDays
	}
	if l, ok := core.FindHouseLook(id); ok { // Put it up straight away
		h.data.Look[l.Slot] = id
		h.data.SettingsTime = time.Now()
	}
	h.apply()
	return h.saveLocked()
}

// buyFilter replaces the air filtration's filter
func (h *household) buyFilter() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch {
	case !h.ownsLocked("upgrade:air_filtration"):
		return errUnknownHome
	case h.data.FilterUntil-h.data.GameDays > core.AirFilterDays/10:
		return errFilterNotDue
	case h.balanceLocked() < core.AirFilterStars:
		return errNotEnoughStars
	}
	h.earnLocked(-core.AirFilterStars, "filter", false)
	h.data.FilterUntil = h.data.GameDays + core.AirFilterDays
	h.apply()
	return h.saveLocked()
}

// setLook puts up a house look the player owns, or with id "" takes down
// what is in slot
func (h *household) setLook(slot, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if id == "" {
		delete(h.data.Look, slot)
	} else if l, ok := core.FindHouseLook(id); !ok || l.Slot != slot || !h.ownsLocked("look:"+id) {
		return errUnknownHome
	} else {
		h.data.Look[slot] = id
	}
	h.data.SettingsTime = time.Now()
	return h.saveLocked()
}

// setHemisphere sets the hemisphere seasons follow
func (h *household) setHemisphere(hemi environment.Hemisphere) error {
	if !hemi.Valid() {
		return errors.New("unknown hemisphere")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data.Hemisphere, h.data.SettingsTime = hemi, time.Now()
	h.apply()
	return h.saveLocked()
}

// setHolidays sets which holidays are turned off
func (h *household) setHolidays(allOff bool, off []string) error {
	var known []string
	for _, id := range off {
		if _, ok := core.FindHoliday(id); ok && !slices.Contains(known, id) {
			known = append(known, id)
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data.AllHolidaysOff, h.data.HolidaysOff, h.data.SettingsTime = allOff, known, time.Now()
	h.apply()
	return h.saveLocked()
}

// setSoundOff sets whether a parent has turned the game's sounds off
func (h *household) setSoundOff(off bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data.SoundOff, h.data.SettingsTime = off, time.Now()
	return h.saveLocked()
}

// seeTip records that a tip has been shown
func (h *household) seeTip(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if slices.Contains(h.data.TipsSeen, id) {
		return nil
	}
	h.data.TipsSeen = append(h.data.TipsSeen, id)
	return h.saveLocked()
}

// tick counts play towards today's visit and game days towards the air
// filter. It returns the Home Stars earned by a visit, if any.
func (h *household) tick(now time.Time, playing bool, gameDays float64) (int, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	elapsed := now.Sub(h.lastTick) // Monotonic: changing the clock doesn't count
	h.lastTick = now
	if !playing || elapsed <= 0 || elapsed > maxTickElapsed {
		return 0, ""
	}
	h.data.GameDays += gameDays
	filterWasFresh := h.data.FilterUntil > h.data.GameDays-gameDays
	if filterWasFresh != (h.data.FilterUntil > h.data.GameDays) {
		h.apply() // The filter just ran out
	}

	day := now.Format("2006-01-02")
	switch {
	case day < h.data.VisitDay:
		return 0, "" // The clock went back
	case day > h.data.VisitDay:
		h.data.VisitDay, h.data.VisitSeconds = day, 0
	}
	h.data.VisitSeconds += elapsed.Seconds()
	if h.data.VisitSeconds < visitPlay.Seconds() || h.data.PaidDay >= day {
		return 0, ""
	}

	// A visit: today's stars, and a streak bonus every 7 and 30 days
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	if h.data.PaidDay == yesterday {
		h.data.Streak++
	} else {
		h.data.Streak = 1
	}
	h.data.PaidDay = day
	stars, note := starsVisit, "a day's visit"
	h.earnLocked(starsVisit, "visit:"+day, true)
	if h.data.Streak%7 == 0 {
		h.earnLocked(starsStreak7, fmt.Sprintf("streak7:%s", day), true)
		stars, note = stars+starsStreak7, fmt.Sprintf("%d days in a row", h.data.Streak)
	}
	if h.data.Streak%30 == 0 {
		h.earnLocked(starsStreak30, fmt.Sprintf("streak30:%s", day), true)
		stars, note = stars+starsStreak30, fmt.Sprintf("%d days in a row", h.data.Streak)
	}
	h.saveLocked()
	return stars, note
}

// merge takes another device's household into this one: every ledger
// entry from both, the newer settings, and the furthest timers
func (h *household) merge(other householdData) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	changed := false
	for _, e := range other.Ledger {
		if !h.hasLocked(e.ID) {
			h.data.Ledger = append(h.data.Ledger, e)
			changed = true
		}
	}
	sort.SliceStable(h.data.Ledger, func(i, j int) bool { return h.data.Ledger[i].Time.Before(h.data.Ledger[j].Time) })
	if other.SettingsTime.After(h.data.SettingsTime) {
		h.data.Hemisphere, h.data.HolidaysOff, h.data.AllHolidaysOff = other.Hemisphere, other.HolidaysOff, other.AllHolidaysOff
		h.data.Look, h.data.SettingsTime = other.Look, other.SettingsTime
		h.data.Shelf = slices.Clone(other.Shelf)
		h.data.SoundOff = other.SoundOff
		changed = true
	}
	if other.PaidDay > h.data.PaidDay {
		h.data.PaidDay, h.data.Streak = other.PaidDay, other.Streak
		changed = true
	}
	if other.GameDays > h.data.GameDays {
		h.data.GameDays, changed = other.GameDays, true
	}
	if other.FilterUntil > h.data.FilterUntil {
		h.data.FilterUntil, changed = other.FilterUntil, true
	}
	for _, id := range other.Achieved {
		if !slices.Contains(h.data.Achieved, id) {
			h.data.Achieved, changed = append(h.data.Achieved, id), true
		}
	}
	for _, id := range other.Collected {
		if !slices.Contains(h.data.Collected, id) {
			h.data.Collected, changed = append(h.data.Collected, id), true
		}
	}
	for _, id := range other.TipsSeen {
		if !slices.Contains(h.data.TipsSeen, id) {
			h.data.TipsSeen, changed = append(h.data.TipsSeen, id), true
		}
	}
	h.clean()
	h.apply()
	if changed {
		h.saveLocked()
	}
	return changed
}

// recordAchievements records achievements a pet has earned; it returns
// whether a breed tier was unlocked by them
func (h *household) recordAchievements(ids []string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	before, added := core.UnlockedTier(h.data.Achieved), false
	for _, id := range ids {
		if !slices.Contains(h.data.Achieved, id) {
			h.data.Achieved, added = append(h.data.Achieved, id), true
		}
	}
	if !added {
		return false
	}
	h.saveLocked()
	return core.UnlockedTier(h.data.Achieved) > before
}

// breedTier returns the highest breed tier unlocked
func (h *household) breedTier() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return core.UnlockedTier(h.data.Achieved)
}

// achieved returns the achievements any pet has earned
func (h *household) achieved() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.data.Achieved)
}

// snapshot returns a copy of the household's data
func (h *household) snapshot() householdData {
	h.mu.Lock()
	defer h.mu.Unlock()
	d := h.data
	d.Ledger = slices.Clone(h.data.Ledger)
	d.HolidaysOff = slices.Clone(h.data.HolidaysOff)
	d.Achieved = slices.Clone(h.data.Achieved)
	d.Collected = slices.Clone(h.data.Collected)
	d.Shelf = slices.Clone(h.data.Shelf)
	d.TipsSeen = slices.Clone(h.data.TipsSeen)
	d.Look = make(map[string]string, len(h.data.Look))
	for k, v := range h.data.Look {
		d.Look[k] = v
	}
	return d
}

// houseStatus is the household as the page shows it
type houseStatus struct {
	Stars       int                    `json:"stars"`
	Upgrades    []houseUpgradeView     `json:"upgrades"`
	Looks       []houseLookView        `json:"looks"`
	Look        map[string]string      `json:"look"` // In use, by slot
	Hemisphere  environment.Hemisphere `json:"hemisphere"`
	Season      string                 `json:"season"`
	Streak      int                    `json:"streak"`
	VisitedDay  bool                   `json:"visited_today"`
	VisitLeft   int                    `json:"visit_seconds_left"` // Play still needed today
	FilterDays  float64                `json:"filter_days"`        // Game days of the air filter left
	FilterStars int                    `json:"filter_stars"`
	Recent      []ledgerEntry          `json:"recent"` // Latest earnings and spendings
}

type houseUpgradeView struct {
	core.HouseUpgrade
	Owned bool `json:"owned"`
}

type houseLookView struct {
	core.HouseLook
	Owned bool `json:"owned"`
}

// status returns the household as the page shows it
func (h *household) status(now time.Time) houseStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	st := houseStatus{Stars: h.balanceLocked(), Look: map[string]string{}, Hemisphere: h.data.Hemisphere,
		Season: core.RealSeason().String(), Streak: h.data.Streak, FilterStars: core.AirFilterStars,
		Upgrades: []houseUpgradeView{}, Looks: []houseLookView{}, Recent: []ledgerEntry{}}
	for k, v := range h.data.Look {
		st.Look[k] = v
	}
	for _, u := range core.HouseUpgrades() {
		st.Upgrades = append(st.Upgrades, houseUpgradeView{HouseUpgrade: u, Owned: h.ownsLocked("upgrade:" + u.ID)})
	}
	for _, l := range core.HouseLooks() {
		st.Looks = append(st.Looks, houseLookView{HouseLook: l, Owned: h.ownsLocked("look:" + l.ID)})
	}
	day := now.Format("2006-01-02")
	st.VisitedDay = h.data.PaidDay >= day
	if !st.VisitedDay {
		played := 0.0
		if h.data.VisitDay == day {
			played = h.data.VisitSeconds
		}
		st.VisitLeft = int(max(visitPlay.Seconds()-played, 0))
	}
	if h.ownsLocked("upgrade:air_filtration") {
		st.FilterDays = max(h.data.FilterUntil-h.data.GameDays, 0)
	}
	for i := len(h.data.Ledger) - 1; i >= 0 && len(st.Recent) < 10; i-- {
		e := h.data.Ledger[i]
		e.What = describeEarning(e.What)
		st.Recent = append(st.Recent, e)
	}
	return st
}

// describeEarning describes a ledger entry for the page
func describeEarning(what string) string {
	kind, rest, _ := strings.Cut(what, ":")
	switch kind {
	case "trip":
		return "A trip"
	case "place":
		return "A new place discovered"
	case "achievement":
		if a, ok := core.FindAchievement(rest[strings.LastIndex(rest, ":")+1:]); ok {
			return "Achievement: " + a.Name
		}
		return "An achievement"
	case "goals":
		return "All of a day's goals"
	case "visit":
		return "A day's visit"
	case "streak7", "streak30":
		return "A visit streak"
	case "upgrade":
		if u, ok := core.FindUpgrade(rest); ok {
			return u.Name
		}
	case "look":
		if l, ok := core.FindHouseLook(rest); ok {
			return l.Name
		}
	case "filter":
		return "Air filter"
	}
	return "Home Stars"
}

// householdKey is the household record's key on the hub, sealed like a
// pet's save
var householdKey = hub.PetKey("household")

// syncHousehold merges the household with the copy on the hub (remote is
// nil if there is none) and uploads the result if it differs
func (s *Server) syncHousehold(ctx context.Context, client *hub.Client, remote *hub.PetInfo) error {
	for attempt := 0; attempt < 3; attempt++ {
		base := 0
		if remote != nil && !remote.Deleted {
			info, contents, err := client.DownloadPetData(ctx, householdKey)
			if err != nil {
				return err
			}
			var theirs householdData
			if contents != nil {
				if err := json.Unmarshal(contents, &theirs); err != nil {
					return fmt.Errorf("the hub's copy can't be read: %w", err)
				}
				s.house.merge(theirs)
			}
			base = info.Version
			if contents != nil && sameHousehold(s.house.snapshot(), theirs) {
				return nil // Nothing new to send
			}
		} else if remote != nil {
			base = remote.Version
		}
		mine, err := json.Marshal(s.house.snapshot())
		if err != nil {
			return err
		}
		_, err = client.UploadPetData(ctx, hub.PetInfo{Key: householdKey, SavedAt: time.Now()}, mine, base)
		var conflict hub.ConflictError
		if !errors.As(err, &conflict) {
			return err
		}
		// Another device uploaded meanwhile: merge its copy and try again
		remote = &hub.PetInfo{Key: householdKey, Version: conflict.Version}
	}
	return errors.New("the hub's copy kept changing; try again later")
}

// sameHousehold reports whether two households hold the same ledger and
// settings
func sameHousehold(a, b householdData) bool {
	if len(a.Ledger) != len(b.Ledger) || len(a.Achieved) != len(b.Achieved) || len(a.Collected) != len(b.Collected) || len(a.TipsSeen) != len(b.TipsSeen) || !a.SettingsTime.Equal(b.SettingsTime) || a.PaidDay != b.PaidDay ||
		a.GameDays != b.GameDays || a.FilterUntil != b.FilterUntil {
		return false
	}
	ids := make(map[string]bool, len(a.Ledger))
	for _, e := range a.Ledger {
		ids[e.ID] = true
	}
	for _, e := range b.Ledger {
		if !ids[e.ID] {
			return false
		}
	}
	return true
}
