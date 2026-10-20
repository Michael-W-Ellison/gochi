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
	Weather        string  `json:"weather"`
	Icon           string  `json:"icon"`
	Temperature    float64 `json:"temperature"` // Celsius
	Season         string  `json:"season"`
	FavoriteSeason string  `json:"favorite_season"`
	BreedingSeason bool    `json:"breeding_season"` // Babies are likelier now
	DayPhase       string  `json:"day_phase"`
	Daytime        bool    `json:"daytime"`
	OutdoorSafe    bool    `json:"outdoor_safe"` // Exploring is riskier when false
	Location       string  `json:"location"`
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
	v.Happiness += (p.moodModifierLocked() - 1.0) * environmentMoodEffect * deltaTime
	v.Stress += (effects.StressModifier - 1.0) * environmentMoodEffect * deltaTime
	v.Clamp()
}

// Each pet has a favorite season, from its personality, and is happier in
// it and less happy in the opposite season. These add to the season's
// mood modifier, which is the same for every pet.
const (
	favoriteSeasonMood = 0.1
	oppositeSeasonMood = -0.08
)

// favoriteSeasonLocked returns the season the pet's personality suits
// best: curious pets love spring, energetic ones summer, adaptable ones
// fall, and cuddly, low-energy pets winter. Caller must hold p.mu.
func (p *DigitalPet) favoriteSeasonLocked() environment.Season {
	t := p.Personality.Traits
	scores := []struct {
		season environment.Season
		score  float64
	}{
		{environment.SeasonSpring, t.Curiosity},
		{environment.SeasonSummer, t.EnergyLevel},
		{environment.SeasonFall, t.Adaptability},
		{environment.SeasonWinter, (t.Affectionate + 1 - t.EnergyLevel) / 2},
	}
	best := scores[0]
	for _, s := range scores[1:] {
		if s.score > best.score {
			best = s
		}
	}
	return best.season
}

// moodModifierLocked returns how the surroundings affect the pet's mood
// (1.0 = neutral): the weather and season, and how the pet feels about the
// season. Caller must hold p.mu.
func (p *DigitalPet) moodModifierLocked() float64 {
	p.ensureEnvironmentLocked()
	return p.Environment.GetEffects().MoodModifier +
		seasonPreference(p.favoriteSeasonLocked(), p.Environment.GetSeason())
}

// seasonPreference returns how a pet with a favorite season feels about
// the current one, as a mood modifier
func seasonPreference(favorite, current environment.Season) float64 {
	switch (int(current) - int(favorite) + 4) % 4 {
	case 0:
		return favoriteSeasonMood
	case 2:
		return oppositeSeasonMood
	}
	return 0
}

// seasonalBreedingLocked returns how the season changes the chance of a
// baby (1.0 = no change; above 1 in spring, below in winter). Caller must
// hold p.mu.
func (p *DigitalPet) seasonalBreedingLocked() float64 {
	p.ensureEnvironmentLocked()
	if modifier := p.Environment.GetSeasonalEffects().BreedingModifier; modifier > 0 {
		return modifier
	}
	return 1.0 // Not calculated yet
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
		Weather:        weather.Type.String(),
		Icon:           icon,
		Temperature:    math.Round(weather.Temperature),
		Season:         env.GetSeason().String(),
		FavoriteSeason: p.favoriteSeasonLocked().String(),
		BreedingSeason: p.seasonalBreedingLocked() > 1.0,
		DayPhase:       env.GetDayPhase(),
		Daytime:        env.IsDaytime(),
		OutdoorSafe:    env.Weather.IsOutdoorSafe(),
		Location:       env.Locations.GetCurrentLocationName(),
	}
}
