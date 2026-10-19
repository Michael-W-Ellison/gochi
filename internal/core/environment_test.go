package core

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

func TestEnvironmentAdvancesWithPet(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	before := pet.Environment.TimeOfDay

	pet.Update(0.25) // Six game hours

	if got := pet.Environment.TimeOfDay; got != before+6 {
		t.Errorf("Time of day %f, want %f", got, before+6)
	}
	if s := pet.EnvironmentStatus(); s.DayPhase == "" || s.Season == "" || s.Weather == "" || s.Icon == "" {
		t.Errorf("Incomplete environment status: %+v", s)
	}
}

func TestBadWeatherAndNightMakeTripsWorse(t *testing.T) {
	outcome := func(setup func(*DigitalPet), roll int) string {
		useRandom(t, roll)
		pet := NewDigitalPet("Biscuit", "user123")
		setup(pet)
		return pet.exploreOutcomeLocked()
	}
	fineDay := func(p *DigitalPet) {
		p.Environment.ForceWeather(environment.WeatherClear)
		p.Environment.SetTimeOfDay(12)
	}
	stormy := func(p *DigitalPet) {
		p.Environment.ForceWeather(environment.WeatherStormy)
		p.Environment.SetTimeOfDay(12)
	}
	night := func(p *DigitalPet) {
		p.Environment.ForceWeather(environment.WeatherClear)
		p.Environment.SetTimeOfDay(2)
	}

	// A roll of 15 is safe on a fine day but an injury in a storm
	if got := outcome(fineDay, 15); got == OutcomeInjured {
		t.Error("A fine day should not turn this roll into an injury")
	}
	if got := outcome(stormy, 15); got != OutcomeInjured {
		t.Errorf("A storm should make injuries more likely, got %s", got)
	}

	// A roll of 55 finds treasure by day but not at night
	if got := outcome(fineDay, 55); got != OutcomeTreasure {
		t.Errorf("Day trip: %s", got)
	}
	if got := outcome(night, 55); got != OutcomeNothing {
		t.Errorf("Night trip should find less, got %s", got)
	}
}

func TestEnvironmentSurvivesSaveLoad(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Environment.ForceWeather(environment.WeatherSnowy)
	pet.Environment.SetTimeOfDay(20)

	data, _ := pet.Save()
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if s := loaded.EnvironmentStatus(); s.Weather != "Snowy" || s.DayPhase != "Dusk" {
		t.Errorf("Environment after load: %+v", s)
	}

	loaded.Environment = nil
	if s := loaded.EnvironmentStatus(); s.Weather == "" {
		t.Error("Pets saved before environments existed should get one")
	}
}
