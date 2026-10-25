package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// accountsOf returns the players the page's route lists
func (h *harness) accountsOf() accountsStatus {
	h.t.Helper()
	var reply struct {
		Data accountsStatus `json:"data"`
	}
	if err := json.Unmarshal(h.do(http.MethodGet, "/api/accounts", "").Body.Bytes(), &reply); err != nil {
		h.t.Fatal(err)
	}
	return reply.Data
}

// petNames returns the names of the player's saved pets
func (h *harness) petNames() []string {
	h.t.Helper()
	var saves struct {
		Data struct {
			Pets []savedPet `json:"pets"`
		} `json:"data"`
	}
	if err := json.Unmarshal(h.do(http.MethodGet, "/api/saves", "").Body.Bytes(), &saves); err != nil {
		h.t.Fatal(err)
	}
	var names []string
	for _, p := range saves.Data.Pets {
		names = append(names, p.Name)
	}
	return names
}

// switchTo switches to the account with id through the page's route
func (h *harness) switchTo(id string) {
	h.t.Helper()
	if rec := h.post("/api/accounts/switch", `{"id":"`+id+`"}`); rec.Code != http.StatusOK {
		h.t.Fatalf("Switching to %s: %d %s", id, rec.Code, rec.Body.String())
	}
}

// createAccount adds a player through the page's route and returns their ID
func (h *harness) createAccount(name string) string {
	h.t.Helper()
	rec := h.post("/api/accounts/create", `{"name":"`+name+`"}`)
	var reply struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &reply) != nil {
		h.t.Fatalf("Adding %s: %d %s", name, rec.Code, rec.Body.String())
	}
	return reply.Data.ID
}

// Two players on one device: each has their own pets and Home Stars, and
// switching saves one player's and opens the other's
func TestPlayerAccounts(t *testing.T) {
	h := newHarness(t)
	first := h.accountsOf()
	if len(first.Accounts) != 1 || first.Accounts[0].Name != "Player 1" || !first.Accounts[0].Current {
		t.Fatalf("A new device should have one player, Player 1: %+v", first)
	}
	p1 := first.Accounts[0].ID
	h.createPet("Mochi", core.SpeciesCat)
	h.server.house.earn(10, "trip", false)

	// Names are typed, unique and checked
	sam := h.createAccount("Sam")
	for body, code := range map[string]int{
		`{"name":" sam "}`: http.StatusConflict,
		`{"name":""}`:      http.StatusBadRequest,
		`{"name":"` + strings.Repeat("x", 25) + `"}`: http.StatusBadRequest,
		`{"name":"Tab\u0007"}`:                       http.StatusBadRequest,
		`not json`:                                   http.StatusBadRequest,
	} {
		if rec := h.post("/api/accounts/create", body); rec.Code != code {
			t.Errorf("Adding %s: %d, want %d", body, rec.Code, code)
		}
	}
	req := newRequest(http.MethodPost, "/api/accounts/create", `{"name":"Phone"}`)
	req.RemoteAddr = "192.168.1.20:50000"
	rec := httptest.NewRecorder()
	h.server.handleAccountCreate(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("A phone added a player: %d", rec.Code)
	}
	numbers := map[uint32]bool{}
	for _, a := range h.accountsOf().Accounts {
		if a.Number < 1000 || a.Number > 9999 || numbers[a.Number] {
			t.Errorf("Each player should have their own number from 1000 to 9999: %+v", a)
		}
		numbers[a.Number] = true
	}

	// Sam starts with nothing of Player 1's
	h.switchTo(sam)
	if names := h.petNames(); len(names) != 0 {
		t.Errorf("Sam shouldn't see Player 1's pets: %v", names)
	}
	if stars := h.server.house.status(time.Now()).Stars; stars != 0 {
		t.Errorf("Sam shouldn't have Player 1's Home Stars: %d", stars)
	}
	if status := statusOf(t, h.server); status != nil && status["name"] != nil {
		t.Errorf("Sam shouldn't have an active pet: %v", status["name"])
	}
	h.createPet("Rex", core.SpeciesDog)
	h.tick(3 * time.Second)

	// Back to Player 1: Mochi is active again, with the stars
	h.switchTo(p1)
	if pet := h.activePet(); pet == nil || pet.Name != "Mochi" {
		t.Fatalf("Player 1's Mochi should be active again: %v", pet)
	}
	if names := h.petNames(); len(names) != 1 || names[0] != "Mochi" {
		t.Errorf("Player 1 should see only Mochi: %v", names)
	}
	if stars := h.server.house.status(time.Now()).Stars; stars != 10 {
		t.Errorf("Player 1's Home Stars: %d", stars)
	}
	if got := h.accountsOf(); got.Accounts[0].ID != p1 || !got.Accounts[0].Current || got.Accounts[1].Current {
		t.Errorf("Player 1 should be playing: %+v", got)
	}
	if rec := h.post("/api/accounts/switch", `{"id":"0123456789abcdef"}`); rec.Code != http.StatusNotFound {
		t.Errorf("Switching to a player who doesn't exist: %d", rec.Code)
	}

	// Renaming
	if rec := h.post("/api/accounts/rename", `{"id":"`+sam+`","name":"Samantha"}`); rec.Code != http.StatusOK {
		t.Errorf("Renaming Sam: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.post("/api/accounts/rename", `{"id":"`+p1+`","name":"SAMANTHA"}`); rec.Code != http.StatusConflict {
		t.Errorf("Taking Samantha's name: %d", rec.Code)
	}

	// Deleting: never the player who is playing, and the name is typed to
	// confirm it
	if rec := h.post("/api/accounts/delete", `{"id":"`+p1+`","confirm_name":"Player 1"}`); rec.Code != http.StatusConflict {
		t.Errorf("Deleting the player who is playing: %d", rec.Code)
	}
	if rec := h.post("/api/accounts/delete", `{"id":"`+sam+`","confirm_name":"Sam"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Deleting with the wrong name: %d", rec.Code)
	}
	samDir := h.server.accounts.dir(sam)
	if _, err := os.Stat(filepath.Join(samDir, "saves")); err != nil {
		t.Fatalf("Sam's pets should be in their folder: %v", err)
	}
	if rec := h.post("/api/accounts/delete", `{"id":"`+sam+`","confirm_name":"Samantha"}`); rec.Code != http.StatusOK {
		t.Fatalf("Deleting Samantha: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(samDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Samantha's folder should be gone: %v", err)
	}

	// Starting again opens the player who was playing
	if _, err := h.server.saveNow(); err != nil {
		t.Fatal(err)
	}
	again := openHarness(t, h.root)
	if names := again.petNames(); len(names) != 1 || names[0] != "Mochi" || len(again.accountsOf().Accounts) != 1 {
		t.Errorf("Starting again: %v %+v", names, again.accountsOf())
	}
}

// Deleting a player needs the parent PIN when one is set
func TestDeletingAPlayerNeedsThePIN(t *testing.T) {
	h := newHarness(t)
	sam := h.createAccount("Sam")
	if err := h.server.parental.SetPIN("", "2468"); err != nil {
		t.Fatal(err)
	}
	if rec := h.post("/api/accounts/delete", `{"id":"`+sam+`","confirm_name":"Sam"}`); rec.Code != http.StatusForbidden {
		t.Errorf("Deleting without the PIN: %d", rec.Code)
	}
	if rec := h.post("/api/accounts/delete", `{"id":"`+sam+`","confirm_name":"Sam","parent_pin":"2468"}`); rec.Code != http.StatusOK {
		t.Errorf("Deleting with the PIN: %d %s", rec.Code, rec.Body.String())
	}
}

// The files of a version from before accounts are moved into the first
// player's folder, and the holiday and sound settings to the device's
func TestFilesFromBeforeAccounts(t *testing.T) {
	root := t.TempDir()
	store, err := newPetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	pet := core.NewDigitalPet("Mochi", localOwner)
	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(store.NewSlot(pet), data); err != nil {
		t.Fatal(err)
	}
	old := `{"ledger":[{"id":"a","time":"2026-01-01T00:00:00Z","stars":7,"what":"trip"}],` +
		`"holidays_off":["halloween"],"sound_off":true}`
	if err := os.WriteFile(filepath.Join(root, householdFileName), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	h := openHarness(t, root)
	if names := h.petNames(); len(names) != 1 || names[0] != "Mochi" {
		t.Fatalf("Mochi should be among the first player's pets: %v", names)
	}
	if stars := h.server.house.status(time.Now()).Stars; stars != 7 {
		t.Errorf("The old Home Stars: %d", stars)
	}
	for _, name := range []string{"saves", householdFileName} {
		if _, err := os.Stat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s should have been moved: %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(h.dir, name)); err != nil {
			t.Errorf("%s should be in the first player's folder: %v", name, err)
		}
	}
	if allOff, off := h.server.parental.holidays(); allOff || len(off) != 1 || core.HolidayOn("halloween") ||
		!h.server.parental.soundOff() {
		t.Errorf("The holiday and sound settings should be the device's: %v %v %v", allOff, off, h.server.parental.soundOff())
	}

	// A file left behind (by a crash while moving) is moved next time;
	// one the player already has is left in place
	if err := os.WriteFile(filepath.Join(root, "owner.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, householdFileName), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// The settings were moved once; changing them isn't undone by the
	// household
	if err := h.server.parental.setSoundOff(false); err != nil {
		t.Fatal(err)
	}
	again := openHarness(t, root)
	if _, err := os.Stat(filepath.Join(again.dir, "owner.json")); err != nil {
		t.Errorf("The play schedule left behind should be moved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, householdFileName)); err != nil {
		t.Errorf("A household the player already has shouldn't replace theirs: %v", err)
	}
	if stars := again.server.house.status(time.Now()).Stars; stars != 7 {
		t.Errorf("The player's own household should be kept: %d stars", stars)
	}
	if again.server.parental.soundOff() {
		t.Error("The household's old sound setting shouldn't be moved twice")
	}
}

// A player's number picks their handle on a cloud hub: connecting to two
// hubs gives the same handle, and the player's name is never sent
func TestHandleFromThePlayersNumber(t *testing.T) {
	h := newHarness(t)
	var handles []string
	for i := 0; i < 2; i++ {
		body := `{"hub_url":"` + testHubURL(t) + `","join_code":"code"}`
		if rec := h.post("/api/cloud/connect", body); rec.Code != http.StatusOK {
			t.Fatalf("Connecting: %d %s", rec.Code, rec.Body.String())
		}
		handles = append(handles, h.server.cloudStatus().Handle)
		if err := h.server.cloudDisconnect(); err != nil {
			t.Fatal(err)
		}
	}
	if handles[0] == "" || handles[0] != handles[1] {
		t.Errorf("The same number should give the same handle on both hubs: %v", handles)
	}
	if strings.Contains(handles[0], "Player") {
		t.Errorf("The handle shouldn't hold the player's name: %q", handles[0])
	}
}

// Requests and the update loop run while players switch back and forth;
// run with -race, nothing is read while it is replaced
func TestSwitchingWhilePlaying(t *testing.T) {
	h := newHarness(t)
	h.createPet("Mochi", core.SpeciesCat)
	sam := h.createAccount("Sam")
	p1 := h.server.accounts.current().ID
	h.switchTo(sam)
	h.createPet("Rex", core.SpeciesDog)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for _, path := range []string{"/api/status", "/api/saves", "/api/home", "/api/stats", "/api/accounts"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				rec := httptest.NewRecorder()
				h.handler.ServeHTTP(rec, newRequest(http.MethodGet, path, ""))
				if rec.Code >= 500 {
					t.Errorf("%s: %d %s", path, rec.Code, rec.Body.String())
				}
			}
		}()
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			rec := httptest.NewRecorder()
			h.handler.ServeHTTP(rec, newRequest(http.MethodPost, "/api/interact", `{"action":"pet"}`))
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			h.server.safeTick(time.Now())
			h.server.autosave()
		}
	}()
	for i := 0; i < 10; i++ {
		for _, id := range []string{p1, sam} {
			rec := httptest.NewRecorder()
			h.handler.ServeHTTP(rec, newRequest(http.MethodPost, "/api/accounts/switch", `{"id":"`+id+`"}`))
			if rec.Code != http.StatusOK {
				t.Errorf("Switching: %d %s", rec.Code, rec.Body.String())
			}
		}
	}
	close(stop)
	wg.Wait()
	if pet := h.activePet(); pet == nil || pet.Name != "Rex" {
		t.Errorf("Sam's Rex should be active: %v", pet)
	}
}

// Pop-ups a player hasn't dismissed wait while another player plays, and
// are shown to them only
func TestPopupsStayWithTheirPlayer(t *testing.T) {
	h := newHarness(t)
	h.createPet("Mochi", core.SpeciesCat)
	p1 := h.server.accounts.current().ID
	popupsOf := func() []popup {
		var status struct {
			Data struct {
				Popups []popup `json:"popups"`
			} `json:"data"`
		}
		if err := json.Unmarshal(h.do(http.MethodGet, "/api/status", "").Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		return status.Data.Popups
	}
	h.server.mu.Lock()
	h.server.addPopupLocked(popup{ID: "grown_up", Icon: "🎉", Title: "Grown up", Text: "Mochi is an adult."})
	h.server.mu.Unlock()

	sam := h.createAccount("Sam")
	h.switchTo(sam)
	h.createPet("Rex", core.SpeciesDog)
	if got := popupsOf(); len(got) != 0 {
		t.Errorf("Sam shouldn't see Player 1's pop-up: %v", got)
	}
	h.server.mu.Lock()
	h.server.addPopupLocked(popup{ID: "sams", Icon: "🐾", Title: "Sam's", Text: "For Sam."})
	h.server.mu.Unlock()

	h.switchTo(p1)
	if got := popupsOf(); len(got) != 1 || got[0].ID != "grown_up" {
		t.Errorf("Player 1's pop-up should be back: %v", got)
	}
	if rec := h.post("/api/popup/dismiss", `{"id":"grown_up"}`); rec.Code != http.StatusOK {
		t.Fatal(rec.Body.String())
	}
	h.switchTo(sam)
	if got := popupsOf(); len(got) != 1 || got[0].ID != "sams" {
		t.Errorf("Sam's pop-up should be back: %v", got)
	}
	h.switchTo(p1)
	if got := popupsOf(); len(got) != 0 {
		t.Errorf("A dismissed pop-up shouldn't come back: %v", got)
	}

	// They last after the game is closed: Player 1's new one shows at once,
	// Sam's waits for Sam
	h.server.mu.Lock()
	h.server.addPopupLocked(popup{ID: "again", Icon: "🎉", Title: "Again", Text: "For Player 1."})
	h.server.mu.Unlock()
	h = openHarness(t, h.root)
	if got := h.server.popups; len(got) != 1 || got[0].ID != "again" {
		t.Errorf("Player 1's pop-up should be kept after closing the game: %v", got)
	}
	h.switchTo(sam)
	if got := popupsOf(); len(got) != 1 || got[0].ID != "sams" {
		t.Errorf("Sam's pop-up should be kept after closing the game: %v", got)
	}
}

// A damaged pop-ups file is set aside, and the player starts without
// pop-ups; one with too many keeps the newest
func TestPopupsFile(t *testing.T) {
	h := newHarness(t)
	path := filepath.Join(h.dir, popupsFileName)
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	again := openHarness(t, h.root)
	if len(again.server.popups) != 0 {
		t.Errorf("A damaged file should give no pop-ups: %v", again.server.popups)
	}
	var many []popup
	for i := 0; i < maxPopups+5; i++ {
		many = append(many, popup{ID: "p" + strconv.Itoa(i), Title: "T"}, popup{ID: "p" + strconv.Itoa(i), Title: "Twice"})
	}
	many = append(many, popup{ID: "", Title: "No ID"})
	data, _ := json.Marshal(many)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	got := openHarness(t, h.root).server.popups
	if len(got) != maxPopups || got[0].ID != "p5" || got[len(got)-1].ID != "p"+strconv.Itoa(maxPopups+4) {
		t.Errorf("The newest %d pop-ups should be kept, once each: %d, %v ... %v", maxPopups, len(got), got[0], got[len(got)-1])
	}
}

// A player without a pet still gets their pop-ups, for the new-pet screen
func TestPopupsWithoutAPet(t *testing.T) {
	h := newHarness(t)
	h.server.mu.Lock()
	h.server.addPopupLocked(popup{ID: "died_x", Icon: "🕊️", Title: "Mochi has passed away", Text: "Goodbye."})
	h.server.mu.Unlock()
	rec := h.do(http.MethodGet, "/api/status", "")
	var reply struct {
		Success bool `json:"success"`
		Data    struct {
			Popups []popup `json:"popups"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound || reply.Success || len(reply.Data.Popups) != 1 || reply.Data.Popups[0].ID != "died_x" {
		t.Errorf("The status without a pet should carry the pop-up: %d %s", rec.Code, rec.Body.String())
	}
}

// The week's themed happening comes once for each player: the player's
// first pet this week meets it, their next pet a random one, and another
// player's pet meets it again
func TestThemedHappeningOncePerPlayer(t *testing.T) {
	clock := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	core.UseClock(func() time.Time { return clock })
	t.Cleanup(func() { core.UseClock(nil) })
	for core.WeekTheme().ID != "explorer" {
		clock = clock.AddDate(0, 0, 7)
	}
	h := newHarness(t)
	happening := func(name string, species core.Species) string {
		t.Helper()
		h.createPet(name, species)
		for i := 0; i < 2000; i++ {
			clock = clock.Add(time.Minute)
			h.server.mu.Lock()
			h.server.lastActivity = clock
			h.server.checkHappeningLocked(clock)
			hap := h.server.pet.CurrentHappening()
			if hap != nil {
				h.server.pet.DismissHappening()
			}
			h.server.mu.Unlock()
			if hap != nil {
				return hap.ID
			}
		}
		t.Fatalf("%s met no happening", name)
		return ""
	}
	if got := happening("Mochi", core.SpeciesCat); got != "bee" {
		t.Errorf("The player's first pet this week should meet the bee: %s", got)
	}
	if week := h.server.house.themeHappeningWeek(); week != core.ThemeWeek() {
		t.Errorf("The household should record the week: %q", week)
	}
	if got := happening("Rex", core.SpeciesDog); got == "bee" || got == "gift" {
		t.Errorf("The player's next pet shouldn't meet a themed happening again: %s", got)
	}
	h.switchTo(h.createAccount("Sam"))
	if got := happening("Biscuit", core.SpeciesDog); got != "bee" {
		t.Errorf("Another player's first pet should meet the bee: %s", got)
	}
}
