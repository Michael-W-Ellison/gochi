package environment

import (
	"time"
)

// Environment represents the complete environmental state
type Environment struct {
	Weather   *WeatherSystem   `json:"weather"`
	Seasons   *SeasonalCycle   `json:"seasons"`
	Locations *LocationManager `json:"locations"`

	// Time tracking
	TimeOfDay       float64   `json:"time_of_day"`       // 0-24 hours
	GameTimeElapsed float64   `json:"game_time_elapsed"` // Total game hours
	LastRealUpdate  time.Time `json:"last_real_update"`

	// Computed effects
	CombinedEffects EnvironmentEffects `json:"combined_effects"`
}

// EnvironmentEffects represents the combined environmental impact on the pet
type EnvironmentEffects struct {
	// Need modifiers (multipliers)
	EnergyDrainRate   float64 `json:"energy_drain_rate"`
	HungerRate        float64 `json:"hunger_rate"`
	ThirstRate        float64 `json:"thirst_rate"`
	StressModifier    float64 `json:"stress_modifier"`
	HappinessModifier float64 `json:"happiness_modifier"`

	// Activity modifiers
	OutdoorSuitability float64 `json:"outdoor_suitability"`
	ExplorationBonus   float64 `json:"exploration_bonus"`
	SocialBonus        float64 `json:"social_bonus"`
	SleepQuality       float64 `json:"sleep_quality"`

	// Health and breeding
	HealthRisk       float64 `json:"health_risk"`
	BreedingModifier float64 `json:"breeding_modifier"`
	GrowthRate       float64 `json:"growth_rate"`

	// Comfort and mood
	OverallComfort float64 `json:"overall_comfort"`
	MoodModifier   float64 `json:"mood_modifier"`
}

// NewEnvironment creates a new environment with all systems initialized
func NewEnvironment() *Environment {
	env := &Environment{
		Weather:         NewWeatherSystem(),
		Seasons:         NewSeasonalCycle(),
		Locations:       NewLocationManager(),
		TimeOfDay:       12.0, // Start at noon
		GameTimeElapsed: 0,
		LastRealUpdate:  time.Now(),
	}

	env.calculateCombinedEffects()
	return env
}

// NewEnvironmentWithConfig creates an environment with custom settings
func NewEnvironmentWithConfig(baseTemp float64, daysPerSeason int) *Environment {
	env := &Environment{
		Weather:         NewWeatherSystemWithConfig(baseTemp, 15.0, 6.0),
		Seasons:         NewSeasonalCycleWithConfig(daysPerSeason, SeasonSpring),
		Locations:       NewLocationManager(),
		TimeOfDay:       12.0,
		GameTimeElapsed: 0,
		LastRealUpdate:  time.Now(),
	}

	env.calculateCombinedEffects()
	return env
}

// Update advances all environmental systems
func (e *Environment) Update(deltaTimeHours float64) {
	// Update game time
	e.GameTimeElapsed += deltaTimeHours
	e.TimeOfDay += deltaTimeHours
	for e.TimeOfDay >= 24.0 {
		e.TimeOfDay -= 24.0
	}

	// Calculate game days passed (for seasonal update)
	gameDaysPassed := deltaTimeHours / 24.0

	// Update seasonal cycle
	e.Seasons.Update(gameDaysPassed)

	// Update weather
	e.Weather.Update(deltaTimeHours, e.Seasons.GetSeason(), e.TimeOfDay)

	// Recalculate combined effects
	e.calculateCombinedEffects()

	e.LastRealUpdate = time.Now()
}

// calculateCombinedEffects computes the total environmental impact
func (e *Environment) calculateCombinedEffects() {
	effects := EnvironmentEffects{
		EnergyDrainRate:   1.0,
		HungerRate:        1.0,
		ThirstRate:        1.0,
		StressModifier:    1.0,
		HappinessModifier: 1.0,
		OutdoorSuitability: 1.0,
		ExplorationBonus:  1.0,
		SocialBonus:       1.0,
		SleepQuality:      1.0,
		HealthRisk:        0.0,
		BreedingModifier:  1.0,
		GrowthRate:        1.0,
		OverallComfort:    1.0,
		MoodModifier:      1.0,
	}

	// Apply weather effects
	weatherEffects := e.Weather.GetWeatherEffects()
	effects.EnergyDrainRate *= weatherEffects["energy_drain"]
	effects.ThirstRate *= weatherEffects["thirst_rate"]
	effects.OutdoorSuitability *= weatherEffects["outdoor_suitability"]
	effects.SleepQuality *= weatherEffects["sleep_quality"]
	effects.MoodModifier *= weatherEffects["mood_modifier"]

	// Apply seasonal effects
	seasonEffects := e.Seasons.Effects
	effects.EnergyDrainRate *= seasonEffects.EnergyModifier
	effects.BreedingModifier *= seasonEffects.BreedingModifier
	effects.GrowthRate *= seasonEffects.GrowthRate
	effects.MoodModifier *= seasonEffects.MoodModifier

	// Activity modifiers based on daylight
	if !e.Seasons.IsDaytime(e.TimeOfDay) {
		effects.OutdoorSuitability *= 0.5
		effects.ExplorationBonus *= 0.6
		effects.SocialBonus *= 0.4
		effects.SleepQuality *= 1.2 // Better sleep at night
	}

	// Apply location effects
	locEffects := e.Locations.ApplyWeatherToLocation(&e.Weather.Current)
	effects.EnergyDrainRate *= locEffects.EnergyDrain
	effects.StressModifier *= locEffects.StressModifier
	effects.ExplorationBonus *= locEffects.ExplorationBonus
	effects.SocialBonus *= locEffects.SocialBonus
	effects.HealthRisk += locEffects.HealthRisk
	effects.HappinessModifier += locEffects.HappinessBonus

	// Calculate overall comfort
	effects.OverallComfort = e.Weather.Current.Comfort
	if e.Locations.CurrentLocation != nil {
		effects.OverallComfort *= e.Locations.CurrentLocation.Features.Comfort
	}

	// Clamp values
	effects.EnergyDrainRate = clamp(effects.EnergyDrainRate, 0.5, 2.0)
	effects.HungerRate = clamp(effects.HungerRate, 0.5, 2.0)
	effects.ThirstRate = clamp(effects.ThirstRate, 0.5, 2.0)
	effects.StressModifier = clamp(effects.StressModifier, 0.3, 2.0)
	effects.HappinessModifier = clamp(effects.HappinessModifier, 0.5, 1.5)
	effects.OutdoorSuitability = clamp(effects.OutdoorSuitability, 0.0, 1.0)
	effects.ExplorationBonus = clamp(effects.ExplorationBonus, 0.0, 2.0)
	effects.SocialBonus = clamp(effects.SocialBonus, 0.0, 2.0)
	effects.SleepQuality = clamp(effects.SleepQuality, 0.5, 1.5)
	effects.HealthRisk = clamp(effects.HealthRisk, 0.0, 1.0)
	effects.BreedingModifier = clamp(effects.BreedingModifier, 0.0, 2.0)
	effects.GrowthRate = clamp(effects.GrowthRate, 0.3, 2.0)
	effects.OverallComfort = clamp(effects.OverallComfort, 0.0, 1.0)
	effects.MoodModifier = clamp(effects.MoodModifier, 0.5, 1.5)

	e.CombinedEffects = effects
}

// GetEffects returns the current combined environmental effects
func (e *Environment) GetEffects() EnvironmentEffects {
	return e.CombinedEffects
}

// GetWeather returns current weather conditions
func (e *Environment) GetWeather() WeatherConditions {
	return e.Weather.Current
}

// GetSeason returns the current season
func (e *Environment) GetSeason() Season {
	return e.Seasons.GetSeason()
}

// GetSeasonalEffects returns current seasonal effects
func (e *Environment) GetSeasonalEffects() SeasonalEffects {
	return e.Seasons.Effects
}

// GetLocation returns the current location
func (e *Environment) GetLocation() *Location {
	return e.Locations.CurrentLocation
}

// TravelTo moves to a new location
func (e *Environment) TravelTo(locationName string) bool {
	return e.Locations.TravelTo(locationName, e.GameTimeElapsed)
}

// UnlockLocation unlocks a new location
func (e *Environment) UnlockLocation(locationName string) bool {
	return e.Locations.UnlockLocation(locationName)
}

// GetTimeOfDay returns the current hour (0-24)
func (e *Environment) GetTimeOfDay() float64 {
	return e.TimeOfDay
}

// SetTimeOfDay sets the current time
func (e *Environment) SetTimeOfDay(hour float64) {
	e.TimeOfDay = hour
	for e.TimeOfDay >= 24.0 {
		e.TimeOfDay -= 24.0
	}
	for e.TimeOfDay < 0 {
		e.TimeOfDay += 24.0
	}
	e.calculateCombinedEffects()
}

// IsDaytime returns whether it's currently daytime
func (e *Environment) IsDaytime() bool {
	return e.Seasons.IsDaytime(e.TimeOfDay)
}

// GetDayPhase returns a description of current time of day
func (e *Environment) GetDayPhase() string {
	hour := e.TimeOfDay

	switch {
	case hour < 5:
		return "Night"
	case hour < 7:
		return "Dawn"
	case hour < 12:
		return "Morning"
	case hour < 14:
		return "Midday"
	case hour < 17:
		return "Afternoon"
	case hour < 20:
		return "Evening"
	case hour < 22:
		return "Dusk"
	default:
		return "Night"
	}
}

// GetDescription returns a human-readable environment description
func (e *Environment) GetDescription() string {
	desc := e.GetDayPhase() + " in " + e.Seasons.GetDescription()
	desc += "\nWeather: " + e.Weather.GetDescription()
	desc += "\nLocation: " + e.Locations.GetCurrentLocationName()
	return desc
}

// IsGoodForActivity returns whether conditions are good for a specific activity
func (e *Environment) IsGoodForActivity(activity string) bool {
	effects := e.CombinedEffects

	switch activity {
	case "play":
		return effects.OutdoorSuitability > 0.5 && effects.OverallComfort > 0.4
	case "explore":
		return effects.OutdoorSuitability > 0.4 && effects.ExplorationBonus > 0.3
	case "social":
		return effects.SocialBonus > 0.3
	case "rest":
		return effects.SleepQuality > 0.6
	case "exercise":
		return effects.OutdoorSuitability > 0.6 && e.Weather.IsOutdoorSafe()
	default:
		return true
	}
}

// GetActivityRecommendations suggests activities based on current conditions
func (e *Environment) GetActivityRecommendations() []string {
	var recommendations []string

	if e.IsGoodForActivity("play") {
		recommendations = append(recommendations, "play")
	}

	if e.IsGoodForActivity("explore") && e.Locations.CurrentLocation != nil {
		if e.Locations.CurrentLocation.Features.ExplorationValue > 0.5 {
			recommendations = append(recommendations, "explore")
		}
	}

	if e.IsGoodForActivity("social") && e.Locations.CurrentLocation != nil {
		if e.Locations.CurrentLocation.Features.SocialOpportunity > 0.5 {
			recommendations = append(recommendations, "socialize")
		}
	}

	if !e.IsDaytime() || e.CombinedEffects.SleepQuality > 1.0 {
		recommendations = append(recommendations, "rest")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "relax at home")
	}

	return recommendations
}

// ForceWeather sets a specific weather type
func (e *Environment) ForceWeather(weatherType WeatherType) {
	e.Weather.SetWeather(weatherType)
	e.calculateCombinedEffects()
}

// ForceSeason sets a specific season
func (e *Environment) ForceSeason(season Season, day int) {
	e.Seasons.SetSeason(season, day)
	e.calculateCombinedEffects()
}

// GetStats returns environment statistics
func (e *Environment) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"game_time_hours":       e.GameTimeElapsed,
		"game_days":             e.GameTimeElapsed / 24.0,
		"current_season":        e.Seasons.GetSeason().String(),
		"season_day":            e.Seasons.GetDayOfSeason(),
		"year":                  e.Seasons.GetYear(),
		"time_of_day":           e.TimeOfDay,
		"weather":               e.Weather.Current.Type.String(),
		"temperature":           e.Weather.Current.Temperature,
		"location":              e.Locations.GetCurrentLocationName(),
		"locations_unlocked":    e.Locations.GetExplorationProgress(),
		"overall_comfort":       e.CombinedEffects.OverallComfort,
	}
}
