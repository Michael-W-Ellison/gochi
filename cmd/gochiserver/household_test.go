package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// newTestHousehold returns a household kept in dir ("" for memory only),
// and puts the game's household settings back afterwards
func newTestHousehold(t *testing.T, dir string) *household {
	t.Helper()
	h, err := loadHousehold(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		core.SetHolidays(false, nil)
		core.SetHouse(nil, false)
		environment.StopCalendar()
	})
	return h
}

func balance(h *household) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.balanceLocked()
}

func TestHomeStarsLedger(t *testing.T) {
	dir := t.TempDir()
	h := newTestHousehold(t, dir)
	if got := h.earn(starsTrip, "trip", false) + h.earn(starsTrip, "trip", false); got != 2*starsTrip {
		t.Errorf("Two trips earned %d", got)
	}
	if h.earn(10, "achievement:first_trick", true) != 10 || h.earn(10, "achievement:first_trick", true) != 0 {
		t.Error("A once-only earning should count once")
	}
	if b := balance(h); b != 16 {
		t.Fatalf("Balance %d, want 16", b)
	}

	if err := h.buy("air_filtration"); !errors.Is(err, errNotEnoughStars) {
		t.Errorf("Buying an upgrade without the stars: %v", err)
	}
	if err := h.buy("no_such_thing"); !errors.Is(err, errUnknownHome) {
		t.Errorf("Buying something unknown: %v", err)
	}
	h.earn(100, "test", false)
	if err := h.buy("wall_mint"); err != nil {
		t.Fatal(err)
	}
	if err := h.buy("wall_mint"); !errors.Is(err, errOwnedHome) {
		t.Errorf("Buying a look twice: %v", err)
	}
	if st := h.status(time.Now()); st.Look[core.LookWallpaper] != "wall_mint" || st.Stars != 76 {
		t.Errorf("A bought look goes up straight away and costs 40: %+v, %d", st.Look, st.Stars)
	}
	if err := h.setLook(core.LookWallpaper, "wall_cream"); !errors.Is(err, errUnknownHome) {
		t.Errorf("Putting up a look that wasn't bought: %v", err)
	}

	// Saved, and read back the same
	again := newTestHousehold(t, dir)
	if balance(again) != balance(h) || again.snapshot().Look[core.LookWallpaper] != "wall_mint" {
		t.Errorf("Reloaded household differs: %d vs %d", balance(again), balance(h))
	}
}

func TestHomeStarsUpgradeAndFilter(t *testing.T) {
	h := newTestHousehold(t, "")
	h.earn(500, "test", false)
	if err := h.buyFilter(); !errors.Is(err, errUnknownHome) {
		t.Errorf("A filter without the upgrade: %v", err)
	}
	if err := h.buy("air_filtration"); err != nil {
		t.Fatal(err)
	}
	if upgrades, fresh := h.upgradesLocked(), h.data.FilterUntil > h.data.GameDays; len(upgrades) != 1 || !fresh {
		t.Fatalf("Upgrades %v, filter fresh %v", upgrades, fresh)
	}
	if err := h.buyFilter(); !errors.Is(err, errFilterNotDue) {
		t.Errorf("Replacing a fresh filter: %v", err)
	}
	h.data.GameDays = h.data.FilterUntil - 1 // Nearly worn out
	before := balance(h)
	if err := h.buyFilter(); err != nil {
		t.Fatal(err)
	}
	if balance(h) != before-core.AirFilterStars || h.data.FilterUntil-h.data.GameDays != core.AirFilterDays {
		t.Errorf("A new filter: %d stars, %.0f days", balance(h), h.data.FilterUntil-h.data.GameDays)
	}
}

// play ticks the household every 4 seconds for d, from start
func play(h *household, start time.Time, d time.Duration) (int, time.Time) {
	stars := 0
	now := start
	h.tick(now, true, 0) // The first tick starts the count
	for end := start.Add(d); now.Before(end); {
		now = now.Add(4 * time.Second)
		got, _ := h.tick(now, true, 0)
		stars += got
	}
	return stars, now
}

func TestDailyVisitsAndStreaks(t *testing.T) {
	h := newTestHousehold(t, "")
	day := time.Date(2026, 3, 10, 9, 0, 0, 0, time.Local)
	if stars, _ := play(h, day, visitPlay-time.Minute); stars != 0 {
		t.Errorf("Nine minutes of play earned %d", stars)
	}
	if stars, _ := play(h, day.Add(time.Hour), 2*time.Minute); stars != starsVisit {
		t.Errorf("Ten minutes of play earned %d, want %d", stars, starsVisit)
	}
	if stars, _ := play(h, day.Add(2*time.Hour), visitPlay); stars != 0 {
		t.Errorf("A second visit the same day earned %d", stars)
	}

	// Long gaps between ticks aren't play
	h.tick(day.AddDate(0, 0, 1), true, 0)
	if got, _ := h.tick(day.AddDate(0, 0, 1).Add(time.Hour), true, 0); got != 0 ||
		h.status(day.AddDate(0, 0, 1)).VisitLeft != int(visitPlay.Seconds()) {
		t.Error("An hour between ticks shouldn't count as play")
	}

	// Seven days in a row earn the weekly bonus on the seventh
	total := 0
	for i := 1; i < 7; i++ {
		stars, _ := play(h, day.AddDate(0, 0, i), visitPlay+time.Minute)
		total += stars
	}
	if h.data.Streak != 7 || total != 6*starsVisit+starsStreak7 {
		t.Errorf("Streak %d, stars %d", h.data.Streak, total)
	}
	// Missing a day starts over
	if play(h, day.AddDate(0, 0, 9), visitPlay+time.Minute); h.data.Streak != 1 {
		t.Errorf("Streak after a missed day: %d", h.data.Streak)
	}
	// The clock going back earns nothing
	if stars, _ := play(h, day.AddDate(0, 0, 3), visitPlay+time.Minute); stars != 0 {
		t.Errorf("Setting the clock back earned %d", stars)
	}
}

func TestHouseholdsMerge(t *testing.T) {
	a, b := newTestHousehold(t, ""), newTestHousehold(t, "")
	a.earn(3, "trip", false)
	b.earn(3, "trip", false)
	a.earn(10, "achievement:first_trick", true)
	b.earn(10, "achievement:first_trick", true) // The same earning on both devices
	if err := b.setHemisphere(environment.Southern); err != nil {
		t.Fatal(err)
	}

	if !a.merge(b.snapshot()) {
		t.Fatal("Merging another device's stars should change the household")
	}
	if balance(a) != 16 {
		t.Errorf("Merged balance %d, want 16 (both trips, the achievement once)", balance(a))
	}
	if a.snapshot().Hemisphere != environment.Southern {
		t.Error("The newer setting should win")
	}
	if a.merge(b.snapshot()) {
		t.Error("Merging the same copy again should change nothing")
	}
	b.merge(a.snapshot())
	if !sameHousehold(a.snapshot(), b.snapshot()) {
		t.Error("Both devices should end up the same")
	}
}

func TestHemisphereSetting(t *testing.T) {
	server := newTestServer(t)
	server.house = newTestHousehold(t, "")
	if data := getJSON(t, server.handleSettings, "/api/settings"); data["hemisphere_set"] != false {
		t.Errorf("A new household hasn't chosen: %v", data)
	}
	if rec := post(t, server.handleSettings, `{"hemisphere":"middle"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("An unknown hemisphere: %d", rec.Code)
	}
	if rec := post(t, server.handleSettings, `{"hemisphere":"southern"}`); rec.Code != http.StatusOK {
		t.Fatalf("Choosing the southern hemisphere: %d %s", rec.Code, rec.Body.String())
	}
	if core.Hemisphere() != environment.Southern {
		t.Error("The game should follow the southern hemisphere")
	}
}

func TestHolidaySwitchesNeedThePIN(t *testing.T) {
	server := newTestServer(t)
	server.house = newTestHousehold(t, "")
	server.parental, _ = loadParentalControls("")
	if err := server.parental.SetPIN("", "2468"); err != nil {
		t.Fatal(err)
	}
	if rec := post(t, server.handlePrivacy, `{"holidays_off":["halloween"]}`); rec.Code == http.StatusOK {
		t.Error("Turning a holiday off without the PIN should be refused")
	}
	if core.HolidayOn("halloween") == false {
		t.Error("Refused: Halloween should still be on")
	}
	rec := post(t, server.handlePrivacy, `{"holidays_off":["halloween","no_such_day"],"parent_pin":"2468"}`)
	if rec.Code != http.StatusOK || core.HolidayOn("halloween") || !core.HolidayOn("christmas") {
		t.Fatalf("Turning Halloween off with the PIN: %d %s", rec.Code, rec.Body.String())
	}
	if off := server.house.snapshot().HolidaysOff; len(off) != 1 {
		t.Errorf("Only known holidays are kept: %v", off)
	}
	post(t, server.handlePrivacy, `{"all_holidays_off":true,"parent_pin":"2468"}`)
	if core.HolidayOn("christmas") {
		t.Error("Every holiday should be off")
	}

	// Only from this computer
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"all_holidays_off":false,"parent_pin":"2468"}`))
	req.RemoteAddr = "192.168.1.20:50000"
	server.handlePrivacy(rec, req)
	if rec.Code == http.StatusOK || core.HolidayOn("christmas") {
		t.Errorf("A phone changed the holidays: %d", rec.Code)
	}
}

func TestOneHolidayItemPerSession(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Mochi", core.SpeciesCat, false, false); err != nil {
		t.Fatal(err)
	}
	bought := 0
	buy := func() error { bought++; return nil }
	server.buyLocked(httptest.NewRecorder(), "pumpkin_costume", buy)
	rec := httptest.NewRecorder()
	server.buyLocked(rec, "santa_hat", buy)
	if rec.Code != http.StatusConflict || bought != 1 {
		t.Errorf("A second holiday item: %d, %d bought", rec.Code, bought)
	}
	rec = httptest.NewRecorder()
	server.buyLocked(rec, "squeaky_toy", buy)
	if rec.Code != http.StatusOK || bought != 2 {
		t.Errorf("Everyday items aren't limited: %d", rec.Code)
	}
}

// family returns a server whose active pet, Mochi, has a young child Pip
// saved in pipSlot
func family(t *testing.T) (server *Server, pipSlot string, pip *core.DigitalPet) {
	t.Helper()
	server = newTestServer(t)
	server.parental, _ = loadParentalControls("")
	if err := server.createPet("Mochi", core.SpeciesCat, false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.FriendshipPoints = 500
	pip = core.NewDigitalPet("Pip", "test")
	pip.Parents = []types.PetID{server.pet.ID}
	pipSlot = "pip"
	savePet(t, server.store, pipSlot, pip)
	return server, pipSlot, pip
}

func TestFamilyTransfers(t *testing.T) {
	server, pipSlot, _ := family(t)
	if rec := post(t, server.handleFamilyTransfer, `{"child_slot":"`+pipSlot+`","amount":600}`); rec.Code != http.StatusConflict {
		t.Errorf("Giving more than Mochi has: %d", rec.Code)
	}
	if rec := post(t, server.handleFamilyTransfer, `{"child_slot":"`+pipSlot+`","amount":100}`); rec.Code != http.StatusOK {
		t.Fatalf("Giving a young child points: %d %s", rec.Code, rec.Body.String())
	}
	pip, _, _, err := server.store.Load(pipSlot)
	if err != nil {
		t.Fatal(err)
	}
	if pip.FriendshipPoints != 100 || server.pet.FriendshipPoints != 400 {
		t.Errorf("Pip has %d, Mochi %d", pip.FriendshipPoints, server.pet.FriendshipPoints)
	}

	// Grown up: no more points while Mochi lives
	pip.Biology.Processes.Age = 70
	savePet(t, server.store, pipSlot, pip)
	if rec := post(t, server.handleFamilyTransfer, `{"child_slot":"`+pipSlot+`","amount":10}`); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), "grown up") {
		t.Errorf("Giving a grown-up child points: %d %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, server.handleFamilyPassOn, `{"child_slot":"`+pipSlot+`","points":10}`); rec.Code != http.StatusConflict {
		t.Errorf("Passing on while alive: %d", rec.Code)
	}

	// After Mochi dies, its belongings and points can pass to Pip once
	server.pet.Inventory = []string{"squeaky_toy"}
	server.pet.Biology.IsAlive = false
	rec := post(t, server.handleFamilyPassOn, `{"child_slot":"`+pipSlot+`","points":400}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Passing on: %d %s", rec.Code, rec.Body.String())
	}
	pip, _, _, _ = server.store.Load(pipSlot)
	if pip.FriendshipPoints != 500 || len(pip.Inventory) != 1 || server.pet.PassedOnTo != pip.ID {
		t.Errorf("Pip after inheriting: %d points, %v", pip.FriendshipPoints, pip.Inventory)
	}
	if rec := post(t, server.handleFamilyPassOn, `{"child_slot":"`+pipSlot+`","points":0}`); rec.Code != http.StatusConflict {
		t.Errorf("Passing on twice: %d", rec.Code)
	}

	// Not a child
	stranger := core.NewDigitalPet("Rex", "test")
	savePet(t, server.store, "rex", stranger)
	if rec := post(t, server.handleFamilyPassOn, `{"child_slot":"rex","points":0}`); rec.Code != http.StatusConflict {
		t.Errorf("Passing on to a stranger: %d", rec.Code)
	}
}

func TestGrowingUpPopups(t *testing.T) {
	server, pipSlot, pip := family(t)
	server.mu.Lock()
	server.birthPopupLocked(pip)
	server.birthPopupLocked(pip) // Queued once
	popups := append([]popup{}, server.popups...)
	server.mu.Unlock()
	if len(popups) != 1 || !strings.Contains(popups[0].Text, "about 60 minutes") {
		t.Fatalf("Birth pop-up: %+v", popups)
	}
	if rec := post(t, server.handlePopupDismiss, `{"id":"`+popups[0].ID+`"}`); rec.Code != http.StatusOK || len(server.popups) != 0 {
		t.Errorf("Dismissing: %d, %d left", rec.Code, len(server.popups))
	}

	// Pip becomes the active pet with ten minutes left: no warning yet
	if rec := post(t, server.handleLoad, `{"slot":"`+pipSlot+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("Switching to Pip: %d %s", rec.Code, rec.Body.String())
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	server.pet.Biology.Processes.Age = 50 // Ten game days: ten minutes
	server.checkGrowingUpLocked()
	if len(server.popups) != 0 {
		t.Errorf("Warned with ten minutes left: %+v", server.popups)
	}
	server.pet.Biology.Processes.Age = 56 // Four minutes
	server.checkGrowingUpLocked()
	if len(server.popups) != 1 || !strings.Contains(server.popups[0].Text, "Mochi (") ||
		!strings.Contains(server.popups[0].Text, "about 4 minutes") {
		t.Fatalf("Five-minute warning: %+v", server.popups)
	}
	server.popups = nil
	server.checkGrowingUpLocked()
	if len(server.popups) != 0 || !server.pet.AdultWarned {
		t.Error("The warning is shown once")
	}
}
