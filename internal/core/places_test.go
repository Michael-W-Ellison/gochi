package core

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

// placeByID finds a place in the pet's list
func placeByID(t *testing.T, pet *DigitalPet, id string) Place {
	t.Helper()
	for _, place := range pet.Places() {
		if place.ID == id {
			return place
		}
	}
	t.Fatalf("No place %q", id)
	return Place{}
}

func TestPlacesAreDiscoveredByTrips(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	useRandom(t, 99, 3) // Trips find nothing and are never hurt
	pet := NewDigitalPet("Biscuit", "user123")

	places := pet.Places()
	if len(places) != 11 || places[0].ID != "park" || !places[0].Discovered || places[1].Discovered {
		t.Fatalf("A new cat should know only the park, of every place but the dog park: %+v", places)
	}
	if err := pet.SendExploringTo("beach"); !errors.Is(err, ErrPlaceLocked) {
		t.Errorf("The beach should be undiscovered: %v", err)
	}
	if err := pet.SendExploringTo("moon"); !errors.Is(err, ErrUnknownPlace) {
		t.Errorf("Unknown place: %v", err)
	}
	if err := pet.SendExploringTo("home"); !errors.Is(err, ErrUnknownPlace) {
		t.Errorf("Home is not a trip: %v", err)
	}

	exploreOnce(t, pet)
	if a := exploreOnce(t, pet); !strings.HasSuffix(a.Text, "Biscuit can now explore the beach!") {
		t.Errorf("The second trip should discover the beach: %q", a.Text)
	}
	if !placeByID(t, pet, "beach").Discovered || pet.TripsTaken != 2 {
		t.Errorf("The beach should be discovered after 2 trips (%d taken)", pet.TripsTaken)
	}

	setCalmHealthyState(pet)
	if err := pet.SendExploringTo("beach"); err != nil {
		t.Fatal(err)
	}
	pet.Update(exploreDuration + 0.1)
	if a := pet.LastAdventure; a.Place != "beach" || !strings.Contains(a.Text, "at the beach") {
		t.Errorf("The trip should go to the beach: %+v", a)
	}
}

func TestRecalledTripsDiscoverNothing(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	if err := pet.SendExploring(); err != nil {
		t.Fatal(err)
	}
	if _, err := pet.RecallFromExploring(); err != nil {
		t.Fatal(err)
	}
	if pet.TripsTaken != 0 {
		t.Errorf("A trip cut short should not count, %d taken", pet.TripsTaken)
	}
}

func TestFindsDepletePlaces(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	useRandom(t, 50, 0) // A treasure
	pet := NewDigitalPet("Biscuit", "user123")

	exploreOnce(t, pet)
	park := placeByID(t, pet, "park")
	if park.Resources != 1-treasureDepletion {
		t.Errorf("A find should take from the park: resources %.3f", park.Resources)
	}
	if placeByID(t, pet, "beach").Resources != 1 {
		t.Error("Other places should be untouched")
	}

	// Resources grow back over time, at 0.05 a day scaled by the season's
	// food availability (winter 0.6): 0.3 takes 10 game days in winter
	pet.Environment.ForceSeason(environment.SeasonWinter, 10)
	pet.Environment.Update(9 * 24)
	if r := placeByID(t, pet, "park").Resources; r >= 1 {
		t.Errorf("The park should still be growing back after 9 winter days, resources %.3f", r)
	}
	pet.Environment.Update(1.1 * 24)
	if r := placeByID(t, pet, "park").Resources; r != 1 {
		t.Errorf("The park should grow back fully in 10 winter days, resources %.3f", r)
	}
}

func TestPlaceOdds(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	odds := func(place string) (int, float64) {
		pet.ExploringTo = place
		return pet.placeOddsLocked()
	}

	if injury, finds := odds("park"); injury != 0 || finds != 1 {
		t.Errorf("The park is the baseline: injury +%d, finds x%.2f", injury, finds)
	}
	if injury, finds := odds("forest"); injury <= 0 || finds <= 1 {
		t.Errorf("The forest should be wilder and richer: injury +%d, finds x%.2f", injury, finds)
	}

	loc, _ := pet.placeLocked("park")
	loc.Deplete(1)
	if _, finds := odds("park"); finds != depletedFinds {
		t.Errorf("A picked-over park should give %.1f of the finds, got %.2f", depletedFinds, finds)
	}

	// Trips from saves made before places existed went to the park
	pet.ExploringTo = ""
	if place, _ := pet.tripPlaceLocked(); place != "park" {
		t.Errorf("Default place = %q", place)
	}
}
