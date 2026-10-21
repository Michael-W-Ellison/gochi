package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// statsServer returns a test server that keeps statistics in a temporary
// directory
func statsServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	s := newTestServer(t)
	stats, err := loadPlayerStats(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.stats = stats
	return s, dir
}

// getStats returns the statistics from /api/stats
func getStats(t *testing.T, s *Server) statsSummary {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleStats(rec, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Stats: %d %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data statsSummary `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.Data
}

// count returns the count named name in a list, or 0
func count(list []statCount, name string) int {
	for _, c := range list {
		if c.Name == name {
			return c.Count
		}
	}
	return 0
}

func TestPlayerStats(t *testing.T) {
	s, dir := statsServer(t)
	if err := s.createPet("Mochi", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	mochi := s.slot
	applied := 0
	for i := 0; i < 3; i++ {
		if s.pet.Interact(types.InteractionPetting, 1).Applied {
			applied++
		}
	}
	s.tick(time.Now())
	if err := s.createPet("Rex", core.SpeciesDog, "", false, false); err != nil {
		t.Fatal(err)
	}
	if s.pet.Interact(types.InteractionPetting, 1).Applied {
		applied++
	}

	// Rex dies of thirst while the game runs
	s.pet.Biology.Vitals.Hydration = 0
	start := time.Now()
	for i := 0; i < 5 && s.pet.IsAlive(); i++ {
		s.tick(start.Add(time.Duration(i) * time.Second))
	}
	if s.pet.IsAlive() {
		t.Fatal("Rex should have died")
	}
	cause := s.pet.Biology.CauseOfDeath

	// Mochi's save is deleted; Mochi stays in the statistics
	if rec := post(t, s.handleDelete, `{"slot":"`+mochi+`","confirm_name":"Mochi"}`); rec.Code != http.StatusOK {
		t.Fatalf("Delete: %s", rec.Body.String())
	}

	stats := getStats(t, s)
	if stats.Raised != 2 || stats.Adopted != 2 || stats.Bred != 0 || stats.Alive != 1 || stats.Died != 1 || stats.Deleted != 1 {
		t.Errorf("Counts: %+v", stats)
	}
	if count(stats.Causes, cause) != 1 || len(stats.Causes) != 1 {
		t.Errorf("Causes = %+v, want one %q", stats.Causes, cause)
	}
	if applied == 0 || count(stats.Care, "Petting") != applied {
		t.Errorf("Care = %+v, want Petting %d", stats.Care, applied)
	}
	total := 0
	for _, c := range stats.Care {
		total += c.Count
	}
	if total != stats.Interactions {
		t.Errorf("Care by kind adds up to %d, want %d", total, stats.Interactions)
	}
	if count(stats.Species, "Cat") != 1 || count(stats.Species, "Dog") != 1 {
		t.Errorf("Species = %+v", stats.Species)
	}
	if len(stats.Pets) != 2 || stats.Pets[0].Name != "Rex" || stats.Pets[0].Alive || stats.Pets[1].Name != "Mochi" || !stats.Pets[1].Deleted {
		t.Errorf("Pets = %+v", stats.Pets)
	}

	// Kept in the data directory where only the player's account can read
	// them, and read back the same
	if err := s.stats.save(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, statsFileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("Statistics file mode = %v, want 0600", info.Mode().Perm())
	}
	reloaded, err := loadPlayerStats(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.stats = reloaded
	if again := getStats(t, s); again.Raised != 2 || again.Died != 1 || again.Interactions != stats.Interactions {
		t.Errorf("Reloaded statistics = %+v", again)
	}

	if rec := post(t, s.handleStats, `{}`); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Statistics can only be read: %d", rec.Code)
	}
}

func TestStatsIncludeEarlierPets(t *testing.T) {
	s, _ := statsServer(t)

	// A pet saved before statistics were kept, with care not counted by kind
	old := core.NewDigitalPet("Bean", localOwner)
	old.TotalInteractions = 7
	savePet(t, s.store, "bean", old)
	born := core.NewDigitalPet("Pip", localOwner)
	born.Genome.Generation = 2
	savePet(t, s.store, "pip", born)

	s.recordSavedPets()
	stats := getStats(t, s)
	if stats.Raised != 2 || stats.Bred != 1 || stats.Adopted != 1 {
		t.Errorf("Counts: %+v", stats)
	}
	if count(stats.Care, notCountedByKind) != 7 || stats.Interactions != 7 {
		t.Errorf("Care = %+v, interactions %d", stats.Care, stats.Interactions)
	}

	// Seeing them again changes nothing
	s.recordSavedPets()
	if again := getStats(t, s); again.Raised != 2 {
		t.Errorf("Pets counted twice: %+v", again)
	}
}

func TestStatsStayOnThisComputer(t *testing.T) {
	s, h := guarded(t)
	stats, err := loadPlayerStats(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.stats = stats
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", s.handleStats)
	h = s.guard(mux)

	if rec := request(h, http.MethodGet, "/api/stats", local, "", nil); rec.Code != http.StatusOK {
		t.Errorf("The player's page should read their statistics: %d", rec.Code)
	}
	if rec := request(h, http.MethodGet, "/api/stats", local, "", map[string]string{"Sec-Fetch-Site": "cross-site"}); rec.Code == http.StatusOK {
		t.Error("Another site must not read the statistics")
	}
	if rec := request(h, http.MethodGet, "/api/stats", remote, "", nil); rec.Code == http.StatusOK {
		t.Error("A device that isn't paired must not read the statistics")
	}

	// The statistics are not a pet save, so cloud sync never sees them
	if err := s.stats.save(); err != nil {
		t.Fatal(err)
	}
	saved, err := s.store.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, pet := range saved {
		if pet.Slot == "stats" {
			t.Error("The statistics should not be in the pet saves")
		}
	}
}

func TestDamagedSaveDoesNotStopTheGame(t *testing.T) {
	s, _ := statsServer(t)
	if err := s.store.Save("bad", []byte(`{"id":"pet_bad","name":"Bad"}`)); err != nil {
		t.Fatal(err)
	}
	good := core.NewDigitalPet("Good", localOwner)
	savePet(t, s.store, "good", good)

	s.recordSavedPets() // Used to crash at startup
	if stats := getStats(t, s); stats.Raised != 1 || stats.Pets[0].Name != "Good" {
		t.Errorf("Only the readable pet should be counted: %+v", stats)
	}
	if _, _, _, err := s.store.Load("bad"); err == nil {
		t.Error("A save without the pet's body should be refused")
	}
}

func TestUnreadableSavesAreListed(t *testing.T) {
	s, _ := statsServer(t)
	if err := os.WriteFile(filepath.Join(s.store.SaveDir(), "broken.json"), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	saved, err := s.store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 || !saved[0].Unreadable || saved[0].Name != "broken" {
		t.Fatalf("An unreadable save should be listed: %+v", saved)
	}
	if rec := post(t, s.handleDelete, `{"slot":"broken","confirm_name":"broken"}`); rec.Code != http.StatusOK {
		t.Errorf("Deleting it: %d %s", rec.Code, rec.Body.String())
	}
}

func TestBreedingChecksTheBabyName(t *testing.T) {
	s, _ := statsServer(t)
	if err := s.createPet("Mochi", core.SpeciesCat, "", false, false); err != nil {
		t.Fatal(err)
	}
	partner := core.NewDigitalPet("Bean", localOwner)
	savePet(t, s.store, "bean", partner)
	rec := post(t, s.handleBreed, `{"partner_slot":"bean","name":"`+strings.Repeat("x", 50)+`‮"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "A pet's name is") {
		t.Errorf("A long name with control characters: %d %s", rec.Code, rec.Body.String())
	}
}

func TestSavedPetsAreLimited(t *testing.T) {
	s, _ := statsServer(t)
	for i := 0; i < maxSavedPets; i++ {
		if err := s.store.Save(fmt.Sprintf("p%d", i), []byte(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	rec := post(t, s.handleCreate, `{"name":"One more","species":"cat"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "delete one") {
		t.Errorf("Creating past the limit: %d %s", rec.Code, rec.Body.String())
	}
}
