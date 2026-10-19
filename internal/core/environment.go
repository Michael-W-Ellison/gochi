package core

import (
	"math"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

// Each pet lives in its own environment from the environment package:
// weather, seasons, time of day and location. It advances on the pet's own
// timeline, so it pauses with the game and is saved with the pet.

// Environment influence per game day: how far mood and stress modifiers
// (1.0 = neutral) move happiness and stress
const environmentMoodEffect = 0.05

// weatherIcons show each weather type
var weatherIcons = map[environment.WeatherType]string{
	environment.WeatherClear:        "☀️",
	environment.WeatherPartlyCloudy: "⛅",
	environment.WeatherCloudy:       "☁️",
	environment.WeatherRainy:        "🌧️",
	environment.WeatherStormy:       "⛈️",
	environment.WeatherSnowy:        "🌨️",
	environment.WeatherFoggy:        "🌫️",
	environment.WeatherWindy:        "💨",
	environment.WeatherHot:          "🔥",
	environment.WeatherCold:         "🥶",
}

// EnvironmentStatus describes the pet's surroundings for display
type EnvironmentStatus struct {
	Weather     string  `json:"weather"`
	Icon        string  `json:"icon"`
	Temperature float64 `json:"temperature"` // Celsius
	Season      string  `json:"season"`
	DayPhase    string  `json:"day_phase"`
	Daytime     bool    `json:"daytime"`
	OutdoorSafe bool    `json:"outdoor_safe"` // Exploring is riskier when false
	Location    string  `json:"location"`
}

// ensureEnvironmentLocked creates the environment for pets saved before it
// was tracked. Caller must hold p.mu.
func (p *DigitalPet) ensureEnvironmentLocked() {
	if p.Environment == nil {
		p.Environment = environment.NewEnvironment()
	}
}

// updateEnvironmentLocked advances the environment by deltaTime game days
// and lets its mood and stress modifiers act on the pet. Caller must hold
// p.mu.
func (p *DigitalPet) updateEnvironmentLocked(deltaTime float64) {
	p.ensureEnvironmentLocked()
	p.Environment.Update(deltaTime * 24)

	effects := p.Environment.GetEffects()
	v := p.Biology.Vitals
	v.Happiness += (effects.MoodModifier - 1.0) * environmentMoodEffect * deltaTime
	v.Stress += (effects.StressModifier - 1.0) * environmentMoodEffect * deltaTime
	v.Clamp()
}

// environmentRatesLocked returns how the environment speeds up or slows
// the pet's hunger, thirst and energy use. Caller must hold p.mu.
func (p *DigitalPet) environmentRatesLocked() (hunger, thirst, energy float64) {
	p.ensureEnvironmentLocked()
	effects := p.Environment.GetEffects()
	return effects.HungerRate, effects.ThirstRate, effects.EnergyDrainRate
}

// explorationConditionsLocked returns how the surroundings change a trip's
// odds: extra injury chance (percentage points) and a souvenir chance
// multiplier. Bad weather is dangerous and night makes finds rarer.
// Caller must hold p.mu.
func (p *DigitalPet) explorationConditionsLocked() (extraInjury int, treasureFactor float64) {
	p.ensureEnvironmentLocked()
	treasureFactor = 1.0
	if !p.Environment.Weather.IsOutdoorSafe() {
		extraInjury += 10
	}
	if !p.Environment.IsDaytime() {
		extraInjury += 5
		treasureFactor = 0.6
	}
	return extraInjury, treasureFactor
}

// EnvironmentStatus describes the pet's current surroundings
func (p *DigitalPet) EnvironmentStatus() EnvironmentStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ensureEnvironmentLocked()

	env := p.Environment
	weather := env.GetWeather()
	icon := weatherIcons[weather.Type]
	if weather.Type == environment.WeatherClear && !env.IsDaytime() {
		icon = "🌙"
	}
	return EnvironmentStatus{
		Weather:     weather.Type.String(),
		Icon:        icon,
		Temperature: math.Round(weather.Temperature),
		Season:      env.GetSeason().String(),
		DayPhase:    env.GetDayPhase(),
		Daytime:     env.IsDaytime(),
		OutdoorSafe: env.Weather.IsOutdoorSafe(),
		Location:    env.Locations.GetCurrentLocationName(),
	}
}
