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

// outdoorSuitabilityLocked returns how good it is to be outdoors now, 0.0
// to 1.0: the weather's comfort (rain and storms 0.2), halved at night.
// Caller must hold p.mu.
func (p *DigitalPet) outdoorSuitabilityLocked() float64 {
	p.ensureEnvironmentLocked()
	suitability := p.Environment.Weather.GetWeatherEffects()["outdoor_suitability"]
	if !p.Environment.IsDaytime() {
		suitability *= 0.5
	}
	return min(max(suitability, 0), 1)
}

// goodOutdoorLevel is the outdoor suitability above which the weather
// doesn't make trips riskier, and pets ask to go out
const goodOutdoorLevel = 0.6

// explorationConditionsLocked returns how the surroundings change a trip's
// odds: extra injury chance (percentage points) and a souvenir chance
// multiplier. Poor weather and darkness are dangerous (up to 12 points
// more), and night makes finds rarer. Caller must hold p.mu.
func (p *DigitalPet) explorationConditionsLocked() (extraInjury int, treasureFactor float64) {
	p.ensureEnvironmentLocked()
	treasureFactor = 1.0
	if suitability := p.outdoorSuitabilityLocked(); suitability < goodOutdoorLevel {
		extraInjury += int((goodOutdoorLevel - suitability) / goodOutdoorLevel * 12)
	}
	if !p.Environment.IsDaytime() {
		treasureFactor = 0.6
	}
	return extraInjury, treasureFactor
}

// activityDriveLocked returns how the season and weather change how
// active the pet wants to be (1.0 = no change): the season's activity
// level, lower in heat, storms, rain and cold, and lower at night except
// for cats, which are most active at dawn and dusk. Caller must hold p.mu.
func (p *DigitalPet) activityDriveLocked() float64 {
	p.ensureEnvironmentLocked()
	env := p.Environment
	drive := env.GetSeasonalEffects().ActivityLevel
	if drive <= 0 {
		drive = 1
	}
	switch weather := env.GetWeather(); weather.Type {
	case environment.WeatherHot:
		drive *= 0.8
	case environment.WeatherStormy:
		drive *= 0.85
	case environment.WeatherRainy, environment.WeatherCold, environment.WeatherSnowy:
		drive *= 0.9
	case environment.WeatherClear:
		if weather.Comfort > 0.7 {
			drive *= 1.05
		}
	}
	phase := env.GetDayPhase()
	switch {
	case p.speciesLocked() == SpeciesCat && (phase == "Dawn" || phase == "Dusk"):
		drive *= 1.1
	case !env.IsDaytime():
		drive *= 0.9
	}
	return drive
}

// comfortSeekingLocked returns how much more the pet seeks company in bad
// weather: storms, cold and snow draw pets to their owners, nervous ones
// most. Caller must hold p.mu.
func (p *DigitalPet) comfortSeekingLocked() float64 {
	p.ensureEnvironmentLocked()
	switch p.Environment.GetWeather().Type {
	case environment.WeatherStormy:
		return 0.1 + 0.15*p.Personality.Traits.Neuroticism
	case environment.WeatherCold, environment.WeatherSnowy:
		return 0.05 + 0.05*p.Personality.Traits.Affectionate
	}
	return 0
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
		OutdoorSafe:    p.outdoorSuitabilityLocked() >= goodOutdoorLevel,
		Location:       env.Locations.GetCurrentLocationName(),
	}
}
