package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
	"github.com/Michael-W-Ellison/gochi/web"
)

// A test harness for the whole game server: a server set up as main sets
// one up, with its files in a temporary data folder, and the page's
// routes behind the same guard as in play. The tests that sweep every
// route and play at random use it.

// harness is a game server under test
type harness struct {
	t       testing.TB
	server  *Server
	handler http.Handler
	root    string    // The data folder
	dir     string    // The folder of the player account being played when the server started
	now     time.Time // The time the next tick runs at
}

// newHarness sets up a server as main does, with an empty data folder
func newHarness(t testing.TB) *harness {
	t.Helper()
	dir := t.TempDir()
	return openHarness(t, dir)
}

// openHarness sets up a server on an existing data folder, as main does
// when the game starts. A data folder from before accounts (or an
// account's folder, which is laid out the same) has its files moved into
// a first account.
func openHarness(t testing.TB, root string) *harness {
	t.Helper()
	accounts, err := openAccounts(root)
	if err != nil {
		t.Fatal(err)
	}
	dir := accounts.dir(accounts.current().ID)
	parts, err := openAccountParts(dir, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	parental, err := loadParentalControls(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := parental.adoptHouseSettings(parts.house.snapshot()); err != nil {
		t.Fatal(err)
	}
	parental.apply()
	pairing, err := loadDevicePairing(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { // The game's household settings are shared by every test
		core.SetHolidays(false, nil)
		core.SetHouse(nil, false)
		core.SetShelfSet(false)
		environment.StopCalendar()
	})
	now := time.Now()
	s := &Server{
		static:       web.Static(),
		clock:        simulation.NewTimeManager(types.TimeScaleGame),
		accounts:     accounts,
		store:        parts.store,
		lastActivity: now,
		awayTimeout:  2 * time.Minute,
		events:       newEventSystem(),
		cloud:        parts.cloud,
		shares:       make(chan sharedAchievement, shareQueue),
		owner:        parts.owner,
		ownerPath:    parts.ownerPath,
		stats:        parts.stats,
		house:        parts.house,
		popups:       parts.popups,
		popupsPath:   parts.popupsPath,
		parental:     parental,
		pairing:      pairing,
		listenAddr:   "127.0.0.1:8080",
	}
	s.recordSavedPets()
	s.scanAchievements()
	mux := http.NewServeMux()
	for _, r := range s.routes() {
		mux.HandleFunc(r.pattern, r.handler)
	}
	if s.staticVersion, err = staticVersion(s.static); err != nil {
		t.Fatal(err)
	}
	mux.Handle("/static/", s.staticHandler(s.static, true))
	mux.HandleFunc("/", s.handleIndex)
	return &harness{t: t, server: s, handler: securityHeaders(s.guard(s.accountGate(mux))), root: root, dir: dir, now: now}
}

// do sends a request as the page on this computer would, and returns the
// reply. A handler that panics fails the test, naming the request.
func (h *harness) do(method, path, body string) (rec *httptest.ResponseRecorder) {
	h.t.Helper()
	req := newRequest(method, path, body)
	rec = httptest.NewRecorder()
	defer func() {
		if r := recover(); r != nil {
			h.t.Fatalf("%s %s %s panicked: %v", method, path, body, r)
		}
	}()
	h.handler.ServeHTTP(rec, req)
	return rec
}

// newRequest makes a request as the page on this computer sends it
func newRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, "http://localhost:8080"+path, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:50000"
	req.Header.Set("Origin", "http://localhost:8080")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// post sends a POST with a JSON body
func (h *harness) post(path, body string) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.do(http.MethodPost, path, body)
}

// tick advances the game by d of play, a second at a time, as the update
// loop does. The player counts as present throughout. Ticks that panic
// fail the test (safeTick would hide them).
func (h *harness) tick(d time.Duration) {
	h.t.Helper()
	defer func() {
		if r := recover(); r != nil {
			h.t.Fatalf("A tick panicked: %v", r)
		}
	}()
	for end := h.now.Add(d); h.now.Before(end); {
		h.now = h.now.Add(time.Second)
		h.server.mu.Lock()
		h.server.markActiveLocked(h.now)
		h.server.mu.Unlock()
		h.server.tick(h.now)
		h.server.shareTripWithFriend()
		h.server.householdTime(h.now)
	}
}

// createPet creates a pet through the page's route
func (h *harness) createPet(name string, species core.Species) {
	h.t.Helper()
	rec := h.post("/api/create", fmt.Sprintf(`{"name":%q,"species":%q}`, name, species))
	if rec.Code != http.StatusOK {
		h.t.Fatalf("Creating %s: %d %s", name, rec.Code, rec.Body.String())
	}
}

// checkPet checks what must always hold for the active pet: every number
// in it is finite, its vitals are in range, and it saves and loads back to
// the same save
func (h *harness) checkPet(context string) {
	h.t.Helper()
	h.server.mu.RLock()
	pet := h.server.pet
	h.server.mu.RUnlock()
	if pet == nil {
		return
	}
	data, err := pet.Save()
	if err != nil {
		h.t.Fatalf("%s: saving: %v", context, err)
	}
	if path := nonFinite(reflect.ValueOf(pet).Elem(), "pet"); path != "" {
		h.t.Fatalf("%s: %s is not a finite number", context, path)
	}
	v := pet.Biology.Vitals
	for name, x := range map[string]float64{"health": v.Health, "energy": v.Energy, "hydration": v.Hydration,
		"nutrition": v.Nutrition, "happiness": v.Happiness, "stress": v.Stress, "fatigue": v.Fatigue,
		"cleanliness": v.Cleanliness} {
		if x < 0 || x > 1 {
			h.t.Fatalf("%s: %s is %v", context, name, x)
		}
	}
	if pet.FriendshipPoints < 0 {
		h.t.Fatalf("%s: %d Friendship Points", context, pet.FriendshipPoints)
	}
	loaded, err := core.Load(data)
	if err != nil {
		h.t.Fatalf("%s: loading the save: %v", context, err)
	}
	again, err := loaded.Save()
	if err != nil {
		h.t.Fatalf("%s: saving the loaded pet: %v", context, err)
	}
	if !sameJSON(data, again) {
		h.t.Fatalf("%s: a save changes when loaded and saved again:\n%s", context, jsonDiff(data, again))
	}
}

// checkHousehold checks what must always hold for the household
func (h *harness) checkHousehold(context string) {
	h.t.Helper()
	if h.server.house == nil {
		return
	}
	d := h.server.house.snapshot()
	if b := balance(h.server.house); b < 0 {
		h.t.Fatalf("%s: %d Home Stars", context, b)
	}
	for _, id := range d.Shelf {
		if !contains(d.Collected, id) {
			h.t.Fatalf("%s: %s is on the shelf but not collected", context, id)
		}
	}
	for _, id := range d.TipsSeen {
		if !contains(tipIDs, id) {
			h.t.Fatalf("%s: unknown tip %q recorded", context, id)
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// nonFinite returns the path of a NaN or infinite number in v, or ""
func nonFinite(v reflect.Value, path string) string {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		if f := v.Float(); math.IsNaN(f) || math.IsInf(f, 0) {
			return path
		}
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			return nonFinite(v.Elem(), path)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if f := v.Type().Field(i); f.IsExported() {
				if p := nonFinite(v.Field(i), path+"."+f.Name); p != "" {
					return p
				}
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if p := nonFinite(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); p != "" {
				return p
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			if p := nonFinite(v.MapIndex(k), fmt.Sprintf("%s[%v]", path, k)); p != "" {
				return p
			}
		}
	}
	return ""
}

// sameJSON reports whether two JSON documents hold the same values
func sameJSON(a, b []byte) bool {
	var x, y interface{}
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// jsonDiff lists the top-level fields that differ between two JSON
// objects
func jsonDiff(a, b []byte) string {
	var x, y map[string]interface{}
	if err := json.Unmarshal(a, &x); err != nil {
		return "first document: " + err.Error()
	}
	if err := json.Unmarshal(b, &y); err != nil {
		return "second document: " + err.Error()
	}
	var diffs []string
	for k := range x {
		if !reflect.DeepEqual(x[k], y[k]) {
			diffs = append(diffs, fmt.Sprintf("%s: %.200v -> %.200v", k, x[k], y[k]))
		}
	}
	for k := range y {
		if _, ok := x[k]; !ok {
			diffs = append(diffs, k+" added")
		}
	}
	return strings.Join(diffs, "\n")
}
