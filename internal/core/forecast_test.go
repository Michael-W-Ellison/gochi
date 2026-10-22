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
