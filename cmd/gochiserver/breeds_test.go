package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

func TestBreedsUnlockWithCareAchievements(t *testing.T) {
	server := newTestServer(t)
	server.house = newTestHousehold(t, t.TempDir())

	if _, err := server.chooseBreed(core.SpeciesCat, "persian"); err != errBreedLocked {
		t.Errorf("A demanding breed at the start: %v", err)
	}
	if _, err := server.chooseBreed(core.SpeciesCat, "labrador"); err == nil {
		t.Error("A dog breed for a cat should be refused")
	}
	for i := 0; i < 20; i++ {
		id, err := server.chooseBreed(core.SpeciesDog, "random")
		if b, _ := core.FindBreed(id); err != nil || b.Tier != core.TierEasy || b.Species != core.SpeciesDog {
			t.Fatalf("A random breed must be an unlocked one: %s %v", id, err)
		}
	}
	if rec := post(t, server.handleCreate, `{"name":"Mochi","species":"cat","breed":"sphynx"}`); rec.Code != http.StatusConflict {
		t.Errorf("Creating a locked breed: %d", rec.Code)
	}
	if rec := post(t, server.handleCreate, `{"name":"Mochi","species":"cat","breed":"british_shorthair"}`); rec.Code != http.StatusOK {
		t.Fatalf("Creating an easy breed: %d %s", rec.Code, rec.Body.String())
	}
	status := statusOf(t, server)["breed"].(map[string]interface{})
	if status["breed"].(map[string]interface{})["id"] != "british_shorthair" || status["look"] == nil {
		t.Errorf("The status shows the breed and look: %v", status)
	}

	// Achievements by any pet unlock the next tier, with a pop-up
	server.mu.Lock()
	server.pet.GoalDays = 3
	server.pet.Biology.Processes.Age = 100
	server.pet.CheckAchievements()
	server.noteAchievementsLocked(server.pet)
	popups := append([]popup{}, server.popups...)
	server.mu.Unlock()
	if server.breedTier() != core.TierModerate || len(popups) != 1 || !strings.Contains(popups[0].Text, "Siamese") {
		t.Errorf("Tier %d, pop-ups %+v", server.breedTier(), popups)
	}
	if _, err := server.chooseBreed(core.SpeciesCat, "siamese"); err != nil {
		t.Errorf("Siamese once unlocked: %v", err)
	}

	data := getJSON(t, server.handleBreeds, "/api/breeds?species=cat")
	if data["tier"].(float64) != core.TierModerate || len(data["breeds"].([]interface{})) != 21 || len(data["tiers"].([]interface{})) != 3 {
		t.Errorf("The breed list: tier %v, %d breeds", data["tier"], len(data["breeds"].([]interface{})))
	}

	// The household remembers, and another device merges it in
	other := newTestHousehold(t, "")
	other.merge(server.house.snapshot())
	if other.breedTier() != core.TierModerate {
		t.Error("Unlocked breeds go to the player's other devices")
	}

	// The pet list shows each pet's look for its drawing
	pets, err := server.store.List()
	if err != nil || len(pets) != 1 || pets[0].Breed != "british_shorthair" || pets[0].Look == nil {
		t.Errorf("The saved pet's breed and look: %+v %v", pets, err)
	}
}

func TestAchievementsOfOldSavesAreScanned(t *testing.T) {
	server := newTestServer(t)
	server.house = newTestHousehold(t, t.TempDir())
	pet := core.NewDigitalPet("Old", "owner")
	pet.Biology.Processes.Age = 100
	pet.GoalDays = 3
	pet.CheckAchievements()
	savePet(t, server.store, "old", pet)

	server.scanAchievements()
	if server.breedTier() != core.TierModerate || !server.house.snapshot().AchievedScanned {
		t.Errorf("Saved pets' achievements count: tier %d", server.breedTier())
	}
}
