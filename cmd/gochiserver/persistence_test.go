package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func newTestStore(t *testing.T) *petStore {
	t.Helper()
	store, err := newPetStore(t.TempDir())
	if err != nil {
		t.Fatalf("newPetStore failed: %v", err)
	}
	return store
}

func savePet(t *testing.T, store *petStore, slot string, pet *core.DigitalPet) {
	t.Helper()
	data, err := pet.Save()
	if err != nil {
		t.Fatalf("pet.Save failed: %v", err)
	}
	if err := store.Save(slot, data); err != nil {
		t.Fatalf("store.Save failed: %v", err)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	now := time.Now()
	return &Server{
		store:        newTestStore(t),
		clock:        simulation.NewTimeManager(types.TimeScaleGame),
		lastActivity: now,
		awayTimeout:  2 * time.Minute,
	}
}

func post(t *testing.T, handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	return rec
}

func TestPetStoreLoadWithoutSave(t *testing.T) {
	store := newTestStore(t)

	if _, _, _, _, err := store.LoadMostRecent(); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist with no saves, got %v", err)
	}
	if _, _, _, err := store.Load("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist for a missing slot, got %v", err)
	}
}

func TestPetStoreRoundTrip(t *testing.T) {
	store := newTestStore(t)

	pet := core.NewDigitalPet("Mochi", "web_user")
	for i := 0; i < 10; i++ {
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
	}
	pet.Update(1)
	savePet(t, store, "mochi", pet)

	// A new store instance simulates restarting the program
	restarted, err := newPetStore(filepath.Dir(store.dm.SavePath))
	if err != nil {
		t.Fatal(err)
	}
	restored, slot, _, _, err := restarted.LoadMostRecent()
	if err != nil {
		t.Fatalf("LoadMostRecent failed: %v", err)
	}

	if slot != "mochi" {
		t.Errorf("Restored slot %q, want mochi", slot)
	}
	if restored.Name != "Mochi" || restored.ID != pet.ID {
		t.Errorf("Restored identity %q/%q, want %q/%q", restored.Name, restored.ID, "Mochi", pet.ID)
	}
	if restored.TotalInteractions != pet.TotalInteractions {
		t.Errorf("TotalInteractions = %d, want %d", restored.TotalInteractions, pet.TotalInteractions)
	}
	if restored.GetAge() != pet.GetAge() {
		t.Errorf("Age = %f, want %f", restored.GetAge(), pet.GetAge())
	}
	want := pet.Memory.GetLearnedAffinity(types.InteractionPlaying)
	if got := restored.Memory.GetLearnedAffinity(types.InteractionPlaying); got != want {
		t.Errorf("Learned play affinity = %f, want %f", got, want)
	}
	stPet, ltPet := pet.Memory.GetMemoryCount()
	stRestored, ltRestored := restored.Memory.GetMemoryCount()
	if stPet != stRestored || ltPet != ltRestored {
		t.Errorf("Memory counts = %d/%d, want %d/%d", stRestored, ltRestored, stPet, ltPet)
	}

	// The restored pet keeps working
	restored.ProcessUserInteraction(types.InteractionPetting, 1.0)
	restored.Update(1)
}

func TestPetStoreRestoresMostRecentSave(t *testing.T) {
	store := newTestStore(t)

	pet := core.NewDigitalPet("Mochi", "web_user")
	savePet(t, store, "mochi", pet)
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	savePet(t, store, "mochi", pet)

	restored, _, _, err := store.Load("mochi")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if restored.TotalInteractions != 1 {
		t.Errorf("Expected the most recent save (1 interaction), got %d", restored.TotalInteractions)
	}
}

func TestPetStoreLoadMostRecentPicksNewestPet(t *testing.T) {
	store := newTestStore(t)

	savePet(t, store, "older", core.NewDigitalPet("Older", "web_user"))
	// Make the first save clearly older
	old := time.Now().Add(-time.Hour)
	os.Chtimes(store.SavePath("older"), old, old)
	savePet(t, store, "newer", core.NewDigitalPet("Newer", "web_user"))

	pet, slot, _, _, err := store.LoadMostRecent()
	if err != nil {
		t.Fatal(err)
	}
	if slot != "newer" || pet.Name != "Newer" {
		t.Errorf("Expected most recently saved pet, got %q in slot %q", pet.Name, slot)
	}
}

func TestPetStoreFallsBackToBackup(t *testing.T) {
	store := newTestStore(t)

	pet := core.NewDigitalPet("Mochi", "web_user")
	savePet(t, store, "mochi", pet)
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	savePet(t, store, "mochi", pet) // Backs up the first save

	if err := os.WriteFile(store.SavePath("mochi"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	restored, _, warning, err := store.Load("mochi")
	if err != nil {
		t.Fatalf("Load should fall back to the backup: %v", err)
	}
	if restored.Name != "Mochi" {
		t.Errorf("Restored name %q, want Mochi", restored.Name)
	}
	if warning == nil {
		t.Error("Skipping the corrupt save should produce a warning")
	}
}

func TestPetStoreList(t *testing.T) {
	store := newTestStore(t)

	alive := core.NewDigitalPet("Alive", "web_user")
	alive.Update(2)
	savePet(t, store, "alive", alive)

	gone := core.NewDigitalPet("Gone", "web_user")
	gone.Biology.IsAlive = false
	savePet(t, store, "gone", gone)

	pets, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(pets) != 2 {
		t.Fatalf("Expected 2 saved pets, got %d", len(pets))
	}

	bySlot := map[string]savedPet{}
	for _, p := range pets {
		bySlot[p.Slot] = p
	}
	if p := bySlot["alive"]; p.Name != "Alive" || !p.IsAlive || p.Age != alive.GetAge() {
		t.Errorf("Unexpected summary for alive pet: %+v", p)
	}
	if p := bySlot["gone"]; p.Name != "Gone" || p.IsAlive {
		t.Errorf("Unexpected summary for gone pet: %+v", p)
	}
}

func TestNewSlotIsUniqueAndSafe(t *testing.T) {
	store := newTestStore(t)

	pet := core.NewDigitalPet("../Mo chi", "web_user")
	slot := store.NewSlot(pet)
	if !ValidSlot(slot) {
		t.Fatalf("NewSlot returned invalid slot %q", slot)
	}
	savePet(t, store, slot, pet)

	if again := store.NewSlot(pet); again == slot || !ValidSlot(again) {
		t.Errorf("Second slot %q should differ from %q and be valid", again, slot)
	}

	for _, bad := range []string{"", "..", "a/b", "a b"} {
		if ValidSlot(bad) {
			t.Errorf("ValidSlot(%q) should be false", bad)
		}
	}
}

func TestHandleSave(t *testing.T) {
	server := newTestServer(t)

	// No pet yet
	if rec := post(t, server.handleSave, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("Save without a pet: status %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if err := server.createPet("Mochi", false, false); err != nil {
		t.Fatal(err)
	}
	if rec := post(t, server.handleSave, ""); rec.Code != http.StatusOK {
		t.Fatalf("Save: status %d, body %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(server.store.SavePath(server.slot)); err != nil {
		t.Errorf("Save file should exist: %v", err)
	}

	rec := httptest.NewRecorder()
	server.handleSave(rec, httptest.NewRequest(http.MethodGet, "/api/save", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/save: status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestCreateKeepsPreviousPet(t *testing.T) {
	server := newTestServer(t)

	if err := server.createPet("First", false, false); err != nil {
		t.Fatal(err)
	}
	firstSlot := server.slot
	server.pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)

	if err := server.createPet("Second", false, false); err != nil {
		t.Fatal(err)
	}
	if server.pet.Name != "Second" || server.slot == firstSlot {
		t.Fatalf("Expected new active pet in a new slot, got %q in %q", server.pet.Name, server.slot)
	}

	// The first pet was saved, including the interaction made after creation
	first, _, _, err := server.store.Load(firstSlot)
	if err != nil {
		t.Fatalf("First pet should still be saved: %v", err)
	}
	if first.Name != "First" || first.TotalInteractions != 1 {
		t.Errorf("First pet = %q with %d interactions, want First with 1", first.Name, first.TotalInteractions)
	}
}

func TestHandleLoadSwitchesPets(t *testing.T) {
	server := newTestServer(t)

	if err := server.createPet("First", false, false); err != nil {
		t.Fatal(err)
	}
	firstSlot := server.slot
	if err := server.createPet("Second", false, false); err != nil {
		t.Fatal(err)
	}
	secondSlot := server.slot
	server.pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)

	rec := post(t, server.handleLoad, `{"slot":"`+firstSlot+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Load: status %d, body %s", rec.Code, rec.Body.String())
	}
	if server.pet.Name != "First" || server.slot != firstSlot {
		t.Errorf("Active pet = %q in %q, want First in %q", server.pet.Name, server.slot, firstSlot)
	}

	// Switching away saved the second pet's progress
	second, _, _, err := server.store.Load(secondSlot)
	if err != nil {
		t.Fatal(err)
	}
	if second.TotalInteractions != 1 {
		t.Errorf("Second pet should be saved with 1 interaction, got %d", second.TotalInteractions)
	}

	// The list includes both pets and marks the active one
	listRec := httptest.NewRecorder()
	server.handleListSaves(listRec, httptest.NewRequest(http.MethodGet, "/api/saves", nil))
	var resp struct {
		Data struct {
			Pets       []savedPet `json:"pets"`
			ActiveSlot string     `json:"active_slot"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data.Pets) != 2 || resp.Data.ActiveSlot != firstSlot {
		t.Errorf("List = %d pets, active %q; want 2, %q", len(resp.Data.Pets), resp.Data.ActiveSlot, firstSlot)
	}
}

func TestHandleLoadRejectsBadSlots(t *testing.T) {
	server := newTestServer(t)

	if rec := post(t, server.handleLoad, `{"slot":"../etc/passwd"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Path traversal slot: status %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if rec := post(t, server.handleLoad, `{"slot":"missing"}`); rec.Code != http.StatusNotFound {
		t.Errorf("Missing slot: status %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestManualPauseStopsTime(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Mochi", false, false); err != nil {
		t.Fatal(err)
	}

	post(t, server.handlePause, `{"paused":true}`)
	age := server.pet.GetAge()

	// A whole day of real time passes while paused
	server.tick(time.Now().Add(24 * time.Hour))
	if server.pet.GetAge() != age || !server.pet.IsAlive() {
		t.Errorf("No time should pass while paused: age %f -> %f, alive %v", age, server.pet.GetAge(), server.pet.IsAlive())
	}

	// Resuming does not apply the paused time
	post(t, server.handlePause, `{"paused":false}`)
	server.tick(time.Now().Add(24*time.Hour + time.Second))
	if server.pet.GetAge() > age+0.1 || !server.pet.IsAlive() {
		t.Errorf("Resuming should continue from the pause, age %f -> %f", age, server.pet.GetAge())
	}
}

func TestAwayPauseKeepsPetAlive(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Mochi", false, false); err != nil {
		t.Fatal(err)
	}

	// The player leaves for a school day; the server ticks every second
	start := time.Now()
	for elapsed := time.Duration(0); elapsed <= 8*time.Hour; elapsed += time.Second {
		server.tick(start.Add(elapsed))
	}

	if !server.pet.IsAlive() {
		t.Fatal("Pet should survive the player being away")
	}
	// Only the time before the away timeout counts: 2 minutes = 2 game days
	if age := server.pet.GetAge(); age > 2.1 {
		t.Errorf("Time should stop after the away timeout, age %f", age)
	}

	// Status (which uses the real clock) reports the away pause until the
	// player returns
	server.lastActivity = time.Now().Add(-8 * time.Hour)
	rec := httptest.NewRecorder()
	server.handleStatus(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if !strings.Contains(rec.Body.String(), `"pause_reason":"away"`) {
		t.Errorf("Status should report away pause: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.handleStatus(rec, httptest.NewRequest(http.MethodGet, "/api/status?active=1", nil))
	if !strings.Contains(rec.Body.String(), `"paused":false`) {
		t.Errorf("Returning player should resume time: %s", rec.Body.String())
	}
}

func TestHandleDeleteRequiresName(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Mochi", false, false); err != nil {
		t.Fatal(err)
	}
	slot := server.slot
	if err := server.createPet("Bean", false, false); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"", "mochi", "Bean"} {
		rec := post(t, server.handleDelete, `{"slot":"`+slot+`","confirm_name":"`+name+`"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Confirm name %q: status %d, want %d", name, rec.Code, http.StatusBadRequest)
		}
	}
	if _, err := server.store.Summary(slot); err != nil {
		t.Fatalf("Pet should not be deleted without the right name: %v", err)
	}

	rec := post(t, server.handleDelete, `{"slot":"`+slot+`","confirm_name":"Mochi"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Delete: status %d, body %s", rec.Code, rec.Body.String())
	}
	if _, err := server.store.Summary(slot); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Save should be gone, got %v", err)
	}
	if backups, _ := server.store.dm.ListBackups(slot); len(backups) != 0 {
		t.Errorf("Backups should be gone, %d remain", len(backups))
	}
	if server.pet == nil || server.pet.Name != "Bean" {
		t.Error("Deleting another pet should not change the active pet")
	}

	if rec := post(t, server.handleDelete, `{"slot":"`+slot+`","confirm_name":"Mochi"}`); rec.Code != http.StatusNotFound {
		t.Errorf("Deleting again: status %d, want %d", rec.Code, http.StatusNotFound)
	}
	if rec := post(t, server.handleDelete, `{"slot":"../x","confirm_name":"x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Bad slot: status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleDeleteActivePet(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Mochi", false, false); err != nil {
		t.Fatal(err)
	}
	slot := server.slot

	// Renamed since the last save: the current name must be typed
	server.pet.Name = "Mochi Jr"
	if rec := post(t, server.handleDelete, `{"slot":"`+slot+`","confirm_name":"Mochi"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Old name should not confirm: status %d", rec.Code)
	}

	rec := post(t, server.handleDelete, `{"slot":"`+slot+`","confirm_name":"Mochi Jr"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"was_active":true`) {
		t.Fatalf("Delete active: status %d, body %s", rec.Code, rec.Body.String())
	}
	if server.pet != nil || server.slot != "" {
		t.Error("Deleting the active pet should unload it")
	}

	// An autosave afterwards must not bring the pet back
	if saved, err := server.saveNow(); saved || err != nil {
		t.Errorf("Nothing should be saved after deleting the active pet: saved=%v err=%v", saved, err)
	}
	if pets, _ := server.store.List(); len(pets) != 0 {
		t.Errorf("Expected no saved pets, got %d", len(pets))
	}
}

func TestHibernationThroughServer(t *testing.T) {
	server := newTestServer(t)
	server.awayTimeout = 0
	if err := server.createPet("Mochi", false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.HibernationEnabled = true

	// Neglect for a long time while "present"
	start := time.Now()
	for elapsed := time.Duration(0); elapsed <= 3*time.Hour; elapsed += time.Minute {
		server.tick(start.Add(elapsed))
	}
	if !server.pet.IsAlive() || !server.pet.IsHibernating() {
		t.Fatalf("Neglected pet should hibernate, alive=%v hibernating=%v", server.pet.IsAlive(), server.pet.IsHibernating())
	}

	rec := httptest.NewRecorder()
	server.handleStatus(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if !strings.Contains(rec.Body.String(), `"hibernating":true`) {
		t.Errorf("Status should report hibernation: %s", rec.Body.String())
	}

	if rec := post(t, server.handleInteract, `{"action":"feed"}`); rec.Code != http.StatusConflict {
		t.Errorf("Interacting while hibernating: status %d, want %d", rec.Code, http.StatusConflict)
	}

	// The saved list shows hibernation after a save
	server.saveNow()
	if pets, _ := server.store.List(); len(pets) != 1 || !pets[0].Hibernating {
		t.Errorf("Saved list should show hibernation: %+v", pets)
	}

	if rec := post(t, server.handleRevive, ""); rec.Code != http.StatusOK {
		t.Fatalf("Revive: status %d, body %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, server.handleInteract, `{"action":"feed"}`); rec.Code != http.StatusOK {
		t.Errorf("Interacting after revive: status %d, body %s", rec.Code, rec.Body.String())
	}
	if rec := post(t, server.handleRevive, ""); rec.Code != http.StatusConflict {
		t.Errorf("Reviving an awake pet: status %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestDrinkAndPointsThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.Biology.Vitals.Hydration = 0.3

	rec := post(t, server.handleInteract, `{"action":"drink"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Drink: status %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"points_earned":5`) {
		t.Errorf("First drink should earn 5 points: %s", rec.Body.String())
	}
	if server.pet.Biology.Vitals.Hydration <= 0.3 {
		t.Error("Drinking should restore water")
	}

	rec = post(t, server.handleInteract, `{"action":"drink"}`)
	if !strings.Contains(rec.Body.String(), `"points_earned":0`) {
		t.Errorf("Second drink within the cooldown should earn nothing: %s", rec.Body.String())
	}

	statusRec := httptest.NewRecorder()
	server.handleStatus(statusRec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	body := statusRec.Body.String()
	if !strings.Contains(body, `"friendship_points":5`) || !strings.Contains(body, `"water":`) || !strings.Contains(body, `"items":[]`) {
		t.Errorf("Status should report points, water and items: %s", body)
	}
}

func TestShopThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}

	if rec := post(t, server.handleBuy, `{"item":"balloon"}`); rec.Code != http.StatusBadRequest ||
		!strings.Contains(rec.Body.String(), "Not enough Friendship Points") {
		t.Errorf("Buying without points: status %d, body %s", rec.Code, rec.Body.String())
	}

	server.pet.FriendshipPoints = 100
	rec := post(t, server.handleBuy, `{"item":"balloon"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Buy: status %d, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"friendship_points":70`) {
		t.Errorf("Response should show the new balance: %s", rec.Body.String())
	}

	if rec := post(t, server.handleBuy, `{"item":"balloon"}`); rec.Code != http.StatusConflict {
		t.Errorf("Buying twice: status %d, want %d", rec.Code, http.StatusConflict)
	}
	if rec := post(t, server.handleBuy, `{"item":"spaceship"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("Unknown item: status %d, want %d", rec.Code, http.StatusBadRequest)
	}

	// The purchase was saved with the pet
	saved, _, _, err := server.store.Load(server.slot)
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Owns("balloon") || saved.FriendshipPoints != 70 {
		t.Errorf("Saved pet: owns balloon=%v, points %d; want true, 70", saved.Owns("balloon"), saved.FriendshipPoints)
	}

	shopRec := httptest.NewRecorder()
	server.handleShop(shopRec, httptest.NewRequest(http.MethodGet, "/api/shop", nil))
	var shop struct {
		Data struct {
			Points int `json:"friendship_points"`
			Items  []struct {
				ID    string `json:"id"`
				Price int    `json:"price"`
				Owned bool   `json:"owned"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(shopRec.Body.Bytes(), &shop); err != nil {
		t.Fatal(err)
	}
	if shop.Data.Points != 70 || len(shop.Data.Items) != len(core.ShopCatalog()) {
		t.Errorf("Shop: %d points, %d items", shop.Data.Points, len(shop.Data.Items))
	}
	for _, item := range shop.Data.Items {
		if item.Owned != (item.ID == "balloon") || item.Price <= 0 {
			t.Errorf("Unexpected shop entry %+v", item)
		}
	}
}

func TestInteractionEffectsThroughServer(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.FriendshipPoints = 500
	if err := server.pet.BuyItem("puzzle_ball"); err != nil {
		t.Fatal(err)
	}
	server.pet.Biology.Vitals.Happiness = 0.3

	rec := post(t, server.handleInteract, `{"action":"play"}`)
	var resp struct {
		Data struct {
			Changes []core.StatChange `json:"changes"`
			Boost   *core.ShopItem    `json:"boosted_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.Boost == nil || resp.Data.Boost.ID != "puzzle_ball" {
		t.Errorf("Response should name the boosting toy: %s", rec.Body.String())
	}
	found := false
	for _, c := range resp.Data.Changes {
		if c.Stat == "Happiness" && c.Change > 0 {
			found = true
		}
	}
	if !found {
		t.Errorf("Response should report the happiness gain: %s", rec.Body.String())
	}

	statusRec := httptest.NewRecorder()
	server.handleStatus(statusRec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if !strings.Contains(statusRec.Body.String(), `"effects":[{"id":"puzzle_ball"`) {
		t.Errorf("Status should list the active effect: %s", statusRec.Body.String())
	}
}

func TestStatusShowsPersonalityAndPreferences(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}

	statusBody := func() string {
		rec := httptest.NewRecorder()
		server.handleStatus(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
		return rec.Body.String()
	}

	body := statusBody()
	for _, want := range []string{`"personality":"Biscuit is well-balanced."`, `"likes":[]`, `"dislikes":[]`, `"activity":"is relaxing"`} {
		if !strings.Contains(body, want) {
			t.Errorf("Status missing %s: %s", want, body)
		}
	}

	for i := 0; i < 10; i++ {
		server.pet.ProcessUserInteraction(types.InteractionPetting, 1.0)
	}
	if body := statusBody(); !strings.Contains(body, `"likes":["being petted"]`) {
		t.Errorf("Status should list the learned like: %s", body)
	}
}

func TestStatusShowsRequest(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}

	statusBody := func() string {
		rec := httptest.NewRecorder()
		server.handleStatus(rec, httptest.NewRequest(http.MethodGet, "/api/status", nil))
		return rec.Body.String()
	}

	if body := statusBody(); !strings.Contains(body, `"request":null`) {
		t.Errorf("A content pet should have no request: %s", body)
	}

	server.pet.Biology.Vitals.Hydration = 0.2
	if body := statusBody(); !strings.Contains(body, `"request":{"action":"drink"`) {
		t.Errorf("A thirsty pet should ask for a drink: %s", body)
	}
}

func TestInteractReportsReaction(t *testing.T) {
	server := newTestServer(t)
	if err := server.createPet("Biscuit", false, false); err != nil {
		t.Fatal(err)
	}
	server.pet.Biology.Vitals.Nutrition = 0.1

	rec := post(t, server.handleInteract, `{"action":"feed"}`)
	body := rec.Body.String()
	if !strings.Contains(body, `"reaction":"delighted"`) || !strings.Contains(body, `"reaction_text":"Biscuit gobbles up every bite!"`) {
		t.Errorf("Response should include the reaction: %s", body)
	}
}
