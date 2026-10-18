package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
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

func savePet(t *testing.T, store *petStore, pet *core.DigitalPet) {
	t.Helper()
	data, err := pet.Save()
	if err != nil {
		t.Fatalf("pet.Save failed: %v", err)
	}
	if err := store.Save(data); err != nil {
		t.Fatalf("store.Save failed: %v", err)
	}
}

func TestPetStoreLoadWithoutSave(t *testing.T) {
	store := newTestStore(t)

	if _, _, err := store.Load(); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist with no save, got %v", err)
	}
}

func TestPetStoreRoundTrip(t *testing.T) {
	store := newTestStore(t)

	pet := core.NewDigitalPet("Mochi", "web_user")
	for i := 0; i < 10; i++ {
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
	}
	pet.Update(1)
	savePet(t, store, pet)

	// A new store instance simulates restarting the program
	restarted, err := newPetStore(filepath.Dir(store.dm.SavePath))
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := restarted.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
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
	savePet(t, store, pet)
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	savePet(t, store, pet)

	restored, _, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if restored.TotalInteractions != 1 {
		t.Errorf("Expected the most recent save (1 interaction), got %d", restored.TotalInteractions)
	}
}

func TestPetStoreFallsBackToBackup(t *testing.T) {
	store := newTestStore(t)

	pet := core.NewDigitalPet("Mochi", "web_user")
	savePet(t, store, pet)
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	savePet(t, store, pet) // Backs up the first save

	if err := os.WriteFile(store.SavePath(), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	restored, _, err := store.Load()
	if err != nil {
		t.Fatalf("Load should fall back to the backup: %v", err)
	}
	if restored.Name != "Mochi" {
		t.Errorf("Restored name %q, want Mochi", restored.Name)
	}
	if store.LastWarning() == nil {
		t.Error("Skipping the corrupt save should produce a warning")
	}
}

func TestHandleSave(t *testing.T) {
	store := newTestStore(t)
	server := &Server{store: store}

	// No pet yet
	rec := httptest.NewRecorder()
	server.handleSave(rec, httptest.NewRequest(http.MethodPost, "/api/save", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Save without a pet: status %d, want %d", rec.Code, http.StatusBadRequest)
	}

	server.pet = core.NewDigitalPet("Mochi", "web_user")
	rec = httptest.NewRecorder()
	server.handleSave(rec, httptest.NewRequest(http.MethodPost, "/api/save", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Save: status %d, body %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(store.SavePath()); err != nil {
		t.Errorf("Save file should exist: %v", err)
	}

	rec = httptest.NewRecorder()
	server.handleSave(rec, httptest.NewRequest(http.MethodGet, "/api/save", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/save: status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
