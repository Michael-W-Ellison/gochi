package core

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestTripEventCatalog(t *testing.T) {
	if len(tripEvents) != 30 {
		t.Errorf("%d trip events", len(tripEvents))
	}
	perPlace := map[string]int{}
	for _, h := range tripEvents {
		if len(h.places) != 1 {
			t.Fatalf("%s needs one place", h.id)
		}
		if _, ok := placeDetails[h.places[0]]; !ok {
			t.Errorf("%s: no place %s", h.id, h.places[0])
		}
		perPlace[h.places[0]]++
		right := 0
		for _, c := range h.choices {
			if c.right {
				right++
			}
		}
		if right != 1 || h.lesson == "" || !strings.HasPrefix(h.text, "While ") {
			t.Errorf("%s is incomplete", h.id)
		}
		if _, ok := findHappening(h.id); !ok {
			t.Errorf("%s can't be found", h.id)
		}
	}
	for id := range placeDetails {
		if n := perPlace[id]; n < 2 || n > 3 {
			t.Errorf("%s has %d trip events", id, n)
		}
	}
}

func TestPlacesForEachSpeciesAndSeason(t *testing.T) {
	cat, dog := NewDigitalPet("Mochi", "owner"), NewDigitalPet("Rex", "owner")
	dog.Species = SpeciesDog
	cat.TripsTaken, dog.TripsTaken = 12, 12
	has := func(p *DigitalPet, id string) bool {
		for _, place := range p.Places() {
			if place.ID == id {
				return true
			}
		}
		return false
	}
	if !has(cat, "garden") || has(cat, "dog_park") || !has(dog, "dog_park") || has(dog, "garden") {
		t.Error("The garden is for cats and the dog park for dogs")
	}
	setCalmHealthyState(dog)
	if err := dog.SendExploringTo("garden"); !errors.Is(err, ErrUnknownPlace) {
		t.Errorf("A dog to the cats' garden: %v", err)
	}

	// The snowy hill only in winter
	setSeason := func(p *DigitalPet, s environment.Season) {
		p.mu.Lock()
		p.ensureEnvironmentLocked()
		p.Environment.Seasons.SetSeason(s, 1)
		p.mu.Unlock()
	}
	setSeason(dog, environment.SeasonSummer)
	var refused RefusedError
	if err := dog.SendExploringTo("snowy_hill"); !errors.As(err, &refused) || !strings.Contains(refused.Reason, "winter") {
		t.Errorf("The snowy hill in summer: %v", err)
	}
	setSeason(dog, environment.SeasonWinter)
	if err := dog.SendExploringTo("snowy_hill"); err != nil {
		t.Errorf("The snowy hill in winter: %v", err)
	}

	// Saves from before the new places get them
	old := NewDigitalPet("Old", "owner")
	old.mu.Lock()
	old.ensureLocationsLocked()
	delete(old.Environment.Locations.Locations, "meadow")
	old.mu.Unlock()
	if !has(old, "meadow") {
		t.Error("An old save should get the meadow")
	}
}

// tripReady returns an adult pet that comes when called
func tripReady(t *testing.T, name string) *DigitalPet {
	t.Helper()
	pet := NewDigitalPet(name, "owner")
	pet.Biology.Processes.Age = 200
	pet.TricksLearned = map[string]bool{"come": true}
	pet.TrickSkills = map[string]float64{"come": 1}
	setCalmHealthyState(pet)
	return pet
}

func TestLongTrips(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	young := NewDigitalPet("Kit", "owner")
	setCalmHealthyState(young)
	if err := young.SendExploringWith(TripOptions{Place: "park", Long: true}); err == nil ||
		!strings.Contains(err.Error(), "too young") {
		t.Errorf("A young pet's long trip: %v", err)
	}
	pet := tripReady(t, "Mochi")
	pet.TrickSkills["come"] = 0.8 // Rusty
	if err := pet.SendExploringWith(TripOptions{Place: "park", Long: true}); err == nil ||
		!strings.Contains(err.Error(), "Come trick") {
		t.Errorf("A long trip with rusty recall: %v", err)
	}

	pet.TrickSkills["come"] = 1
	useRandom(t, 50) // Finds on both chances
	if err := pet.SendExploringWith(TripOptions{Place: "park", Long: true}); err != nil {
		t.Fatal(err)
	}
	pet.Update(exploreDuration + 0.1)
	if !pet.IsExploring() {
		t.Fatal("A long trip lasts longer than a short one")
	}
	pet.Update(longExploreDuration)
	a := pet.LastAdventure
	if a == nil || !a.Long || a.Souvenir == nil || len(a.Extra) != 1 || len(pet.Souvenirs) != 2 {
		t.Errorf("A long trip's finds: %+v", a)
	}
}

func TestTripWithAFriend(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	useRandom(t, 50)
	mochi, pip := tripReady(t, "Mochi"), tripReady(t, "Pip")
	if err := mochi.SendExploringWith(TripOptions{Place: "park", Friend: pip}); err == nil ||
		!strings.Contains(err.Error(), "closer friends") {
		t.Errorf("A trip with a pet it doesn't know: %v", err)
	}
	mochi.Relationships.AddRelationship(pip.ID, types.RelationshipFriend).BondStrength = 0.6
	dog := tripReady(t, "Rex")
	dog.Species = SpeciesDog
	mochi.Relationships.AddRelationship(dog.ID, types.RelationshipFriend).BondStrength = 0.6
	if err := mochi.SendExploringWith(TripOptions{Place: "park", Friend: dog}); err == nil ||
		!strings.Contains(err.Error(), "same species") {
		t.Errorf("A cat and a dog on a trip: %v", err)
	}

	if err := mochi.SendExploringWith(TripOptions{Place: "park", Friend: pip, FriendSlot: "pip"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := mochi.TripFriendToShare(); ok {
		t.Error("Nothing to share during the trip")
	}
	mochi.Update(exploreDuration + 0.1)
	slot, ok := mochi.TripFriendToShare()
	if !ok || slot != "pip" || mochi.LastAdventure.Friend != "Pip" {
		t.Fatalf("After the trip: %q %v %+v", slot, ok, mochi.LastAdventure)
	}
	if _, err := ShareTrip(mochi, pip); err != nil {
		t.Fatal(err)
	}
	rel, _ := pip.Relationships.GetRelationship(mochi.ID)
	if len(pip.Souvenirs) != 1 || pip.Souvenirs[0] != mochi.LastAdventure.Souvenir.ID || rel == nil || rel.BondStrength <= 0 {
		t.Errorf("Pip's share: %v %+v", pip.Souvenirs, rel)
	}
	if _, ok := mochi.TripFriendToShare(); ok {
		t.Error("The finds are shared once")
	}
}

func TestTripEventsAfterATrip(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	useRandom(t, 0) // Always an event
	pet := tripReady(t, "Mochi")
	pet.mu.Lock()
	pet.ExploringTo = "forest"
	pet.LastAdventure = &Adventure{Place: "forest"}
	pet.startTripEventLocked("forest")
	pet.mu.Unlock()
	h := pet.CurrentHappening()
	if h == nil || !strings.HasPrefix(h.ID, "trip_forest") || !strings.HasPrefix(h.Text, "While in the forest") {
		t.Fatalf("A forest event: %+v", h)
	}

	// A good choice can bring an extra find
	pendingHappening(pet, "trip_forest_mushroom")
	result, err := pet.ChooseHappening("away")
	if err != nil || !result.Right || len(pet.Souvenirs) != 1 || len(pet.LastAdventure.Extra) != 1 {
		t.Errorf("Leading the pet away from a mushroom: %+v %v %v", result, err, pet.Souvenirs)
	}
}
