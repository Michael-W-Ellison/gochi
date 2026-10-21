package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestDamagedSettingsAreSetAside(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{statsFileName, "cloud.json", "devices.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{{{"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	stats := loadOrSetAside("statistics", filepath.Join(dir, statsFileName),
		func() (*playerStats, error) { return loadPlayerStats(dir) })
	cloudConn := loadOrSetAside("cloud", filepath.Join(dir, "cloud.json"),
		func() (*cloud, error) { return newCloud(dir) })
	pairing := loadOrSetAside("devices", filepath.Join(dir, "devices.json"),
		func() (*devicePairing, error) { return loadDevicePairing(dir) })
	if stats == nil || cloudConn == nil || pairing == nil || len(stats.pets) != 0 || pairing.Count() != 0 {
		t.Fatal("Damaged settings should start afresh")
	}
	entries, _ := os.ReadDir(dir)
	aside := 0
	for _, e := range entries {
		if strings.Contains(e.Name(), ".damaged-") {
			aside++
		}
	}
	if aside != 3 {
		t.Errorf("The damaged files should be kept aside: %d", aside)
	}
}

func TestOldTempFilesAreRemoved(t *testing.T) {
	dir := t.TempDir()
	old, fresh, save := filepath.Join(dir, "saves", "a.json.tmp-123"), filepath.Join(dir, "stats.json.tmp"), filepath.Join(dir, "saves", "a.json")
	if err := os.MkdirAll(filepath.Join(dir, "saves"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{old, fresh, save} {
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	long := time.Now().Add(-2 * tempFileAge)
	for _, path := range []string{old, save} {
		if err := os.Chtimes(path, long, long); err != nil {
			t.Fatal(err)
		}
	}
	removeTempFiles(dir)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("An old temporary file should be removed")
	}
	for _, path := range []string{fresh, save} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s should be kept: %v", path, err)
		}
	}
}

func TestStatsAfterAnInterruptedDelete(t *testing.T) {
	s, _ := statsServer(t)
	pet := core.NewDigitalPet("Mochi", localOwner)
	savePet(t, s.store, "mochi", pet)
	s.recordSavedPets()

	// The game stopped after recording the delete, before removing the save
	s.stats.deleted(pet.ID)
	s.recordSavedPets()
	if stats := getStats(t, s); stats.Deleted != 0 || stats.Raised != 1 {
		t.Errorf("A pet whose save is still there isn't deleted: %+v", stats)
	}

	// Hand-edited negative counts read as 0
	s.stats.pets[types.PetID("pet_x")] = &petLog{Name: "X", Interactions: -5, Care: map[string]int{"Petting": -3}}
	s.stats.dirty = true
	if err := s.stats.save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := loadPlayerStats(filepath.Dir(s.stats.path))
	if err != nil {
		t.Fatal(err)
	}
	if x := reloaded.pets["pet_x"]; x.Interactions != 0 || len(x.Care) != 0 {
		t.Errorf("Negative counts: %+v", x)
	}
}
