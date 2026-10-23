package core

import (
	"math"
	"reflect"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestForecastSeesNeedsComing(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Nutrition = needRequestLevel + 0.05
	if r := pet.CurrentRequest(); r != nil {
		t.Fatalf("The pet shouldn't be asking for anything yet, got %+v", r)
	}

	forecasts := pet.Forecast(3)
	var feed *NeedForecast
	for i := range forecasts {
		if forecasts[i].Action == "feed" {
			feed = &forecasts[i]
		}
	}
	if feed == nil {
		t.Fatalf("The pet should be hungry within 3 days, got %+v", forecasts)
	}
	if feed.InDays <= 0 || feed.InDays > 3 {
		t.Errorf("Hunger forecast for %v days, want within 3", feed.InDays)
	}
	for i := 1; i < len(forecasts); i++ {
		if forecasts[i].InDays < forecasts[i-1].InDays {
			t.Errorf("Forecasts should be soonest first, got %+v", forecasts)
		}
	}

	// The forecast doesn't change the pet
	if pet.Biology.Vitals.Nutrition != needRequestLevel+0.05 {
		t.Errorf("Forecasting changed the pet's nutrition to %v", pet.Biology.Vitals.Nutrition)
	}
}

func TestForecastSkipsCurrentRequest(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Nutrition = 0.1
	for _, f := range pet.Forecast(3) {
		if f.Action == "feed" {
			t.Errorf("A need the pet is already asking for shouldn't be forecast, got %+v", f)
		}
	}
}

func TestForecastNothingForDeadPet(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	pet.Biology.IsAlive = false
	if f := pet.Forecast(3); len(f) != 0 {
		t.Errorf("A dead pet should have no forecasts, got %+v", f)
	}
	if f := NewDigitalPet("Mochi", "user123").Forecast(0); len(f) != 0 {
		t.Errorf("A zero horizon should have no forecasts, got %+v", f)
	}
}

// A need that starts while a more urgent one is waiting is still coming:
// the forecast once saw only the most urgent need at each step, so hunger
// starting just after thirst was never listed
func TestForecastSeesNeedsBehindMoreUrgentOnes(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Hydration = needRequestLevel + 0.03
	pet.Biology.Vitals.Nutrition = needRequestLevel + 0.06
	found := map[string]float64{}
	for _, f := range pet.Forecast(3) {
		found[f.Action] = f.InDays
	}
	drink, thirsty := found["drink"]
	feed, hungry := found["feed"]
	if !thirsty || !hungry {
		t.Fatalf("Both thirst and hunger should be forecast, got %+v", pet.Forecast(3))
	}
	if feed < drink {
		t.Errorf("Hunger (in %v days) should come after thirst (in %v days) here", feed, drink)
	}
}

// Forecasts made a moment apart look at the same moments of the pet's
// future, so a need keeps its time; they once looked at moments counted
// from now, so a need's time moved, and a need on top only briefly came
// and went
func TestForecastsAMomentApartAgree(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	at := func() map[string]float64 {
		times := map[string]float64{}
		for _, f := range pet.Forecast(30) {
			times[f.Action] = pet.Biology.GetAgeInDays() + f.InDays
		}
		return times
	}
	before := at()
	if len(before) == 0 {
		t.Fatal("A new pet should have needs coming within 30 days")
	}
	for i := 0; i < 20; i++ {
		pet.Update(1.0 / 60)
		after := at()
		for action, when := range before {
			if got, ok := after[action]; !ok || math.Abs(got-when) > 1e-6 {
				t.Fatalf("After %d updates, %s moved from day %.3f to day %.3f (present: %v)", i+1, action, when, got, ok)
			}
		}
	}
}

func TestForecastStep(t *testing.T) {
	for _, c := range []struct{ horizon, step float64 }{{30, 0.25}, {120, 0.25}, {1, 1.0 / 64}, {1.0 / 12, 1.0 / 1024}} {
		if got := forecastStep(c.horizon); got != c.step {
			t.Errorf("forecastStep(%v) = %v, want %v", c.horizon, got, c.step)
		}
		if c.horizon/forecastStep(c.horizon) < minForecastSteps {
			t.Errorf("A forecast over %v days takes fewer than %d steps", c.horizon, minForecastSteps)
		}
	}
}

// Forecasts of the same pet agree: the copy's weather, which decides when
// the pet wants to explore, once came from the shared generator, so after
// care made exploring one of the first needs, each forecast placed it on a
// different day
func TestForecastIsTheSameEachTime(t *testing.T) {
	explorers := 0
	for i := 0; i < 10; i++ {
		pet := NewPetOfBreed("Mochi", "user123", SpeciesCat, "")
		pet.Update(0.1)
		pet.Interact(types.InteractionPlaying, 0.8)
		first := pet.Forecast(30)
		for _, f := range first {
			if f.Action == "explore" {
				explorers++
			}
		}
		for j := 0; j < 10; j++ {
			if again := pet.Forecast(30); !reflect.DeepEqual(again, first) {
				t.Fatalf("Pet %d, forecast %d differs: %+v, then %+v", i+1, j+2, first, again)
			}
		}
	}
	if explorers == 0 {
		t.Error("Some of the pets should want to explore within 30 days")
	}
}

// upcoming returns UpcomingNeeds's needs by action
func upcoming(pet *DigitalPet) map[string]NeedForecast {
	needs := map[string]NeedForecast{}
	for _, f := range pet.UpcomingNeeds(3) {
		needs[f.Action] = f
	}
	return needs
}

// A need once shown stays until care meets it, even when it is no longer
// forecast: the list once lost needs and got them back between refreshes
func TestUpcomingNeedsKeepAShownNeedUntilMet(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Hydration = needRequestLevel + 0.03
	drink, ok := upcoming(pet)["drink"]
	if !ok {
		t.Fatalf("Thirst should be coming: %+v", pet.UpcomingNeeds(3))
	}

	pet.Biology.Vitals.Hydration = 1 // No longer coming, but not by care
	if len(pet.Forecast(3)) != 0 {
		t.Fatalf("Nothing should be forecast now: %+v", pet.Forecast(3))
	}
	if got, ok := upcoming(pet)["drink"]; !ok || got.InDays != drink.InDays {
		t.Fatalf("Thirst should stay at %v days until met, got %+v", drink.InDays, pet.UpcomingNeeds(3))
	}

	before := pet.TotalInteractions
	pet.Interact(types.InteractionPetting, 0.5)
	if pet.TotalInteractions == before {
		t.Fatal("The petting should count as care")
	}
	if needs := upcoming(pet); len(needs) != 0 {
		t.Errorf("Care has been given and thirst isn't coming any more: %+v", needs)
	}
}

// A need shown leaves when it is due, or when the pet has it
func TestUpcomingNeedsLeaveWhenDueOrHere(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Hydration = needRequestLevel + 0.03
	if _, ok := upcoming(pet)["drink"]; !ok {
		t.Fatalf("Thirst should be coming: %+v", pet.UpcomingNeeds(3))
	}
	pet.Update(1)
	if r := pet.CurrentRequest(); r == nil || r.Action != "drink" {
		t.Fatalf("The pet should be thirsty now, asks %+v", r)
	}
	if _, ok := upcoming(pet)["drink"]; ok {
		t.Error("Thirst is here, not coming")
	}

	pet = NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Hydration = needRequestLevel + 0.03
	upcoming(pet)
	pet.Biology.Vitals.Hydration = 1 // Kept, though no longer coming
	pet.Update(1)                    // Past its time
	if _, ok := upcoming(pet)["drink"]; ok {
		t.Errorf("A need past its time should leave the list: %+v", pet.UpcomingNeeds(3))
	}
}

// A new need doesn't push one already shown off the list; it takes a place
// when one is free
func TestUpcomingNeedsKeepTheirPlaces(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Nutrition = needRequestLevel + 0.05
	shown := pet.UpcomingNeeds(30)
	if len(shown) != maxForecasts {
		t.Fatalf("Three needs should be coming within 30 days: %+v", shown)
	}

	pet.Biology.Vitals.Hydration = needRequestLevel + 0.03 // Thirst, soon: sooner than the rest
	if f := pet.Forecast(30); len(f) == 0 || f[0].Action != "drink" {
		t.Fatalf("Thirst should now be forecast first: %+v", f)
	}
	again := pet.UpcomingNeeds(30)
	if len(again) != len(shown) {
		t.Fatalf("The list changed from %+v to %+v", shown, again)
	}
	for i := range shown {
		if again[i].Action != shown[i].Action {
			t.Fatalf("The list changed from %+v to %+v", shown, again)
		}
	}

	// Feeding meets hunger: thirst takes its place
	if shown[0].Action != "feed" {
		t.Fatalf("Hunger should be coming first: %+v", shown)
	}
	if r := pet.Interact(types.InteractionFeeding, 0.8); !r.Applied {
		t.Fatalf("The pet should eat: %+v", r)
	}
	needs := map[string]bool{}
	for _, f := range pet.UpcomingNeeds(30) {
		needs[f.Action] = true
	}
	if needs["feed"] || !needs["drink"] {
		t.Errorf("Thirst should take the place feeding freed: %+v", pet.UpcomingNeeds(30))
	}
}

func TestUpcomingNeedsNothingWhileAway(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Hydration = needRequestLevel + 0.03
	upcoming(pet)
	pet.Biology.IsAlive = false
	if needs := pet.UpcomingNeeds(3); len(needs) != 0 {
		t.Errorf("A dead pet needs nothing: %+v", needs)
	}
}
