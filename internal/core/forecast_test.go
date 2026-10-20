package core

import "testing"

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
