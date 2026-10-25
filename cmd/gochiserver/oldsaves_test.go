package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// Saves and household files written by older versions of the game lack
// the fields added since. The old-save tests take files written after
// plenty of play and remove fields from them, then start the game on them
// and play: the game must start, play must not crash, and what the
// household records must still match the saved pets. (A household from
// before souvenirs were collected once never collected its pets' old
// souvenirs, because a flag from an earlier version said the saved pets
// had already been looked at.)

// playedData plays two pets for a while in a new data folder and saves
// everything. It returns the folder and the pets' save slots.
func playedData(t *testing.T) (string, []string) {
	t.Helper()
	h := newHarness(t)
	withPets(h)
	pet := h.activePet()
	pet.Biology.Processes.Age = 300
	pet.FriendshipPoints = 3000
	pet.TricksLearned = map[string]bool{"sit": true, "come": true, "touch": true}
	pet.TrickSkills = map[string]float64{"sit": 1, "come": 1, "touch": 1}
	pet.Souvenirs = []string{"pebble", "acorn", "pinecone", "dandelion", "pearl"}
	setCalmForTest(pet)
	for _, req := range [][2]string{
		{"/api/interact", `{"action":"play"}`},
		{"/api/interact", `{"action":"feed"}`},
		{"/api/shows/enter", `{"show":"happiest","level":1}`},
		{"/api/tips/seen", `{"id":"tour"}`},
		{"/api/home/shelf", `{"items":["pebble","acorn"]}`},
		{"/api/explore", `{"place":"park","long":true}`},
	} {
		if rec := h.post(req[0], req[1]); rec.Code >= 500 {
			t.Fatalf("%s: %d %s", req[0], rec.Code, rec.Body.String())
		}
	}
	h.tick(6 * time.Minute) // The trip ends, and the household's time passes
	h.server.collectLocked(nil, souvenirIDs(pet))
	if _, err := h.server.saveNow(); err != nil {
		t.Fatal(err)
	}
	if err := h.server.house.save(); err != nil {
		t.Fatal(err)
	}
	var slots []string
	list, _ := h.server.store.List()
	for _, p := range list {
		slots = append(slots, p.Slot)
	}
	return h.dir, slots
}

// setCalmForTest makes a pet well enough for anything
func setCalmForTest(p *core.DigitalPet) {
	v := p.Biology.Vitals
	v.Health, v.Energy, v.Hydration, v.Nutrition, v.Happiness, v.Stress, v.Fatigue, v.Cleanliness = 1, 1, 1, 1, 1, 0, 0, 1
}

// playAfterStart plays a little on a game just started, checking it
func playAfterStart(h *harness, context string) {
	h.t.Helper()
	for _, req := range [][2]string{
		{"/api/interact", `{"action":"feed"}`},
		{"/api/interact", `{"action":"play"}`},
		{"/api/explore", `{"place":"park"}`},
		{"/api/games/finish", `{"game":"puzzle","quality":0.5}`},
		{"/api/happening", `{"choice":"comfort"}`},
	} {
		if rec := h.post(req[0], req[1]); rec.Code >= 500 {
			h.t.Fatalf("%s: %s: %d %s", context, req[0], rec.Code, rec.Body.String())
		}
	}
	for _, path := range []string{"/api/status", "/api/collection", "/api/books", "/api/projects", "/api/explore/places",
		"/api/games", "/api/shows", "/api/saves"} {
		if rec := h.do(http.MethodGet, path, ""); rec.Code >= 500 || !json.Valid(rec.Body.Bytes()) {
			h.t.Fatalf("%s: %s: %d %s", context, path, rec.Code, rec.Body.String())
		}
	}
	h.tick(10 * time.Second)
	h.checkPet(context)
	h.checkHousehold(context)
}

// checkHouseholdRecords checks that the household has recorded every saved
// pet's souvenirs and achievements
func checkHouseholdRecords(h *harness, context string) {
	h.t.Helper()
	d := h.server.house.snapshot()
	list, _ := h.server.store.List()
	for _, entry := range list {
		pet, _, _, err := h.server.store.Load(entry.Slot)
		if err != nil {
			continue
		}
		for _, id := range pet.Souvenirs {
			if !contains(d.Collected, id) {
				h.t.Errorf("%s: %s's %s isn't in the household's collection", context, pet.Name, id)
			}
		}
		for _, a := range pet.EarnedAchievements() {
			if !contains(d.Achieved, a.ID) {
				h.t.Errorf("%s: %s's achievement %s isn't recorded for the household", context, pet.Name, a.ID)
			}
		}
	}
}

// copyDir copies a data folder
func copyDir(t *testing.T, from string) string {
	t.Helper()
	to := t.TempDir()
	err := filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(to, rel), 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), data, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return to
}

// householdFields are the household file's fields, in the order they
// were added (the struct's order)
func householdFields() []string {
	var names []string
	ty := reflect.TypeOf(householdData{})
	for i := 0; i < ty.NumField(); i++ {
		if tag := strings.Split(ty.Field(i).Tag.Get("json"), ",")[0]; tag != "" && tag != "-" {
			names = append(names, tag)
		}
	}
	return names
}

// TestOlderHouseholdFiles starts the game on household files as older
// versions wrote them: without the newest field, then without the two
// newest, and so on
func TestOlderHouseholdFiles(t *testing.T) {
	dir, _ := playedData(t)
	fields := householdFields()
	for n := 1; n <= len(fields); n++ {
		removed := fields[len(fields)-n:]
		// Named by the oldest field removed: a list of them all is too
		// long for the test's folder name
		t.Run(fmt.Sprintf("without %d, from %s", n, removed[0]), func(t *testing.T) {
			old := copyDir(t, dir)
			path := filepath.Join(old, householdFileName)
			var file map[string]json.RawMessage
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &file); err != nil {
				t.Fatal(err)
			}
			for _, f := range removed {
				delete(file, f)
			}
			data, _ = json.Marshal(file)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			h := openHarness(t, old)
			checkHouseholdRecords(h, "after starting")
			playAfterStart(h, "playing")
		})
	}
}

// keysOf returns an object's keys, sorted
func keysOf(m map[string]json.RawMessage) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestPetSavesMissingFields starts the game on a pet save missing each of
// its fields in turn, and each field of the objects within it, then plays
func TestPetSavesMissingFields(t *testing.T) {
	dir, slots := playedData(t)
	store, err := newPetStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	pet, _, _, err := store.Load(slots[0])
	if err != nil {
		t.Fatal(err)
	}
	original, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(original, &top); err != nil {
		t.Fatal(err)
	}

	// Each field, and each field of each object field
	type variant struct {
		name string
		data []byte
	}
	var variants []variant
	for _, k := range keysOf(top) {
		edited := map[string]json.RawMessage{}
		for kk, v := range top {
			edited[kk] = v
		}
		delete(edited, k)
		data, _ := json.Marshal(edited)
		variants = append(variants, variant{k, data})

		var inner map[string]json.RawMessage
		if json.Unmarshal(top[k], &inner) != nil {
			continue
		}
		for _, ik := range keysOf(inner) {
			editedInner := map[string]json.RawMessage{}
			for kk, v := range inner {
				editedInner[kk] = v
			}
			delete(editedInner, ik)
			innerData, _ := json.Marshal(editedInner)
			edited := map[string]json.RawMessage{}
			for kk, v := range top {
				edited[kk] = v
			}
			edited[k] = innerData
			data, _ := json.Marshal(edited)
			variants = append(variants, variant{k + "." + ik, data})
		}
	}
	if len(variants) < 100 {
		t.Fatalf("Only %d variants: the save should have many fields", len(variants))
	}

	for _, v := range variants {
		t.Run("without "+v.name, func(t *testing.T) {
			old := copyDir(t, dir)
			store, err := newPetStore(old)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Save(slots[0], v.data); err != nil {
				t.Fatal(err)
			}
			h := openHarness(t, old)
			rec := h.post("/api/load", `{"slot":"`+slots[0]+`"}`)
			if rec.Code >= 500 {
				t.Fatalf("Loading: %d %s", rec.Code, rec.Body.String())
			}
			playAfterStart(h, "playing")
		})
	}
}

// TestDamagedPetFallsBackToBackup restores a pet from its backup when its
// save is intact as a file but the pet in it is damaged
func TestDamagedPetFallsBackToBackup(t *testing.T) {
	store := newTestStore(t)
	pet := core.NewDigitalPet("Mochi", "owner")
	if err := store.SavePet("mochi", pet); err != nil {
		t.Fatal(err)
	}
	if err := store.Save("mochi", []byte(`{"name":"Mochi"}`)); err != nil { // No body or personality
		t.Fatal(err)
	}
	loaded, path, warning, err := store.Load("mochi")
	if err != nil || loaded.ID != pet.ID || path == store.SavePath("mochi") || warning == nil {
		t.Errorf("Expected the backup: %v %s %v", err, path, warning)
	}
}
