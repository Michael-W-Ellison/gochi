package environment

import (
	"math"
	"time"
)

// Season represents the four seasons
type Season int

const (
	SeasonSpring Season = iota
	SeasonSummer
	SeasonFall
	SeasonWinter
)

// String returns the string representation of Season
func (s Season) String() string {
	return [...]string{"Spring", "Summer", "Fall", "Winter"}[s]
}

// SeasonalEffects contains modifiers that seasons apply to the game
type SeasonalEffects struct {
	TemperatureModifier float64 `json:"temperature_modifier"`
	DaylightHours       float64 `json:"daylight_hours"`
	GrowthRate          float64 `json:"growth_rate"`          // For any growth mechanics
	ActivityLevel       float64 `json:"activity_level"`       // Natural pet activity level
	FoodAvailability    float64 `json:"food_availability"`    // Wild food availability
	BreedingModifier    float64 `json:"breeding_modifier"`    // Breeding success modifier
	MoodModifier        float64 `json:"mood_modifier"`        // Base mood effect
	EnergyModifier      float64 `json:"energy_modifier"`      // Energy consumption rate
}

// SeasonalCycle manages the progression of seasons
type SeasonalCycle struct {
	CurrentSeason    Season         `json:"current_season"`
	DayOfSeason      int            `json:"day_of_season"`
	DaysPerSeason    int            `json:"days_per_season"`
	Year             int            `json:"year"`
	SeasonProgress   float64        `json:"season_progress"`   // 0.0 to 1.0 within current season
	Effects          SeasonalEffects `json:"effects"`
	TransitionPhase  float64        `json:"transition_phase"`  // 0.0 to 1.0 during season transitions
	StartDate        time.Time      `json:"start_date"`
}

// NewSeasonalCycle creates a new seasonal cycle
func NewSeasonalCycle() *SeasonalCycle {
	sc := &SeasonalCycle{
		CurrentSeason: SeasonSpring,
		DayOfSeason:   1,
		DaysPerSeason: 30, // 30 game days per season
		Year:          1,
		SeasonProgress: 0.0,
		StartDate:     time.Now(),
	}
	sc.Effects = sc.calculateSeasonalEffects()
	return sc
}

// NewSeasonalCycleWithConfig creates a seasonal cycle with custom settings
func NewSeasonalCycleWithConfig(daysPerSeason int, startSeason Season) *SeasonalCycle {
	sc := NewSeasonalCycle()
	sc.DaysPerSeason = daysPerSeason
	sc.CurrentSeason = startSeason
	sc.Effects = sc.calculateSeasonalEffects()
	return sc
}

// Update advances the seasonal cycle based on elapsed game days
func (sc *SeasonalCycle) Update(gameDaysPassed float64) {
	// Accumulate days
	totalDays := float64(sc.DayOfSeason) + gameDaysPassed

	// Check for season change
	for totalDays > float64(sc.DaysPerSeason) {
		totalDays -= float64(sc.DaysPerSeason)
		sc.advanceSeason()
	}

	sc.DayOfSeason = int(totalDays)
	if sc.DayOfSeason < 1 {
		sc.DayOfSeason = 1
	}

	// Calculate progress within season
	sc.SeasonProgress = float64(sc.DayOfSeason) / float64(sc.DaysPerSeason)

	// Calculate transition phase (smooth transition in last/first 5 days)
	transitionDays := 5.0
	if float64(sc.DayOfSeason) <= transitionDays {
		// Beginning of season - transitioning from previous
		sc.TransitionPhase = float64(sc.DayOfSeason) / transitionDays
	} else if float64(sc.DaysPerSeason-sc.DayOfSeason) <= transitionDays {
		// End of season - transitioning to next
		sc.TransitionPhase = float64(sc.DaysPerSeason-sc.DayOfSeason) / transitionDays
	} else {
		sc.TransitionPhase = 1.0 // Fully in current season
	}

	// Recalculate effects
	sc.Effects = sc.calculateSeasonalEffects()
}

// advanceSeason moves to the next season
func (sc *SeasonalCycle) advanceSeason() {
	sc.CurrentSeason = (sc.CurrentSeason + 1) % 4
	if sc.CurrentSeason == SeasonSpring {
		sc.Year++
	}
	sc.DayOfSeason = 1
}

// calculateSeasonalEffects computes current seasonal effects
func (sc *SeasonalCycle) calculateSeasonalEffects() SeasonalEffects {
	effects := SeasonalEffects{}

	// Base effects per season
	switch sc.CurrentSeason {
	case SeasonSpring:
		effects.TemperatureModifier = 0.0
		effects.DaylightHours = 12.0 + sc.SeasonProgress*2.0 // 12-14 hours
		effects.GrowthRate = 1.2
		effects.ActivityLevel = 1.1
		effects.FoodAvailability = 0.9
		effects.BreedingModifier = 1.3 // Peak breeding season
		effects.MoodModifier = 1.1
		effects.EnergyModifier = 1.0

	case SeasonSummer:
		effects.TemperatureModifier = 15.0
		effects.DaylightHours = 14.0 + (1.0-math.Abs(sc.SeasonProgress-0.5)*2)*2.0 // Peak at 16
		effects.GrowthRate = 1.3
		effects.ActivityLevel = 1.0 // Can be too hot
		effects.FoodAvailability = 1.2 // Abundant
		effects.BreedingModifier = 1.0
		effects.MoodModifier = 1.05
		effects.EnergyModifier = 1.1 // Heat drains energy

	case SeasonFall:
		effects.TemperatureModifier = -5.0
		effects.DaylightHours = 12.0 - sc.SeasonProgress*2.0 // 12-10 hours
		effects.GrowthRate = 0.8
		effects.ActivityLevel = 1.0
		effects.FoodAvailability = 1.1 // Harvest season
		effects.BreedingModifier = 0.8
		effects.MoodModifier = 0.95
		effects.EnergyModifier = 1.0

	case SeasonWinter:
		effects.TemperatureModifier = -20.0
		effects.DaylightHours = 10.0 - (1.0-math.Abs(sc.SeasonProgress-0.5)*2)*2.0 // Low at 8
		effects.GrowthRate = 0.5
		effects.ActivityLevel = 0.8 // Less active
		effects.FoodAvailability = 0.6 // Scarce
		effects.BreedingModifier = 0.5
		effects.MoodModifier = 0.85
		effects.EnergyModifier = 1.2 // Cold requires more energy
	}

	// Apply transition blending if in transition phase
	if sc.TransitionPhase < 1.0 {
		prevSeason := (sc.CurrentSeason + 3) % 4 // Previous season
		prevEffects := sc.getSeasonBaseEffects(prevSeason)
		blend := sc.TransitionPhase

		effects.TemperatureModifier = lerp(prevEffects.TemperatureModifier, effects.TemperatureModifier, blend)
		effects.DaylightHours = lerp(prevEffects.DaylightHours, effects.DaylightHours, blend)
		effects.GrowthRate = lerp(prevEffects.GrowthRate, effects.GrowthRate, blend)
		effects.ActivityLevel = lerp(prevEffects.ActivityLevel, effects.ActivityLevel, blend)
		effects.FoodAvailability = lerp(prevEffects.FoodAvailability, effects.FoodAvailability, blend)
		effects.BreedingModifier = lerp(prevEffects.BreedingModifier, effects.BreedingModifier, blend)
		effects.MoodModifier = lerp(prevEffects.MoodModifier, effects.MoodModifier, blend)
		effects.EnergyModifier = lerp(prevEffects.EnergyModifier, effects.EnergyModifier, blend)
	}

	return effects
}

// getSeasonBaseEffects returns base effects for a given season (used for blending)
func (sc *SeasonalCycle) getSeasonBaseEffects(season Season) SeasonalEffects {
	switch season {
	case SeasonSpring:
		return SeasonalEffects{
			TemperatureModifier: 0.0,
			DaylightHours:       13.0,
			GrowthRate:          1.2,
			ActivityLevel:       1.1,
			FoodAvailability:    0.9,
			BreedingModifier:    1.3,
			MoodModifier:        1.1,
			EnergyModifier:      1.0,
		}
	case SeasonSummer:
		return SeasonalEffects{
			TemperatureModifier: 15.0,
			DaylightHours:       15.0,
			GrowthRate:          1.3,
			ActivityLevel:       1.0,
			FoodAvailability:    1.2,
			BreedingModifier:    1.0,
			MoodModifier:        1.05,
			EnergyModifier:      1.1,
		}
	case SeasonFall:
		return SeasonalEffects{
			TemperatureModifier: -5.0,
			DaylightHours:       11.0,
			GrowthRate:          0.8,
			ActivityLevel:       1.0,
			FoodAvailability:    1.1,
			BreedingModifier:    0.8,
			MoodModifier:        0.95,
			EnergyModifier:      1.0,
		}
	case SeasonWinter:
		return SeasonalEffects{
			TemperatureModifier: -20.0,
			DaylightHours:       9.0,
			GrowthRate:          0.5,
			ActivityLevel:       0.8,
			FoodAvailability:    0.6,
			BreedingModifier:    0.5,
			MoodModifier:        0.85,
			EnergyModifier:      1.2,
		}
	}
	return SeasonalEffects{}
}

// GetSeason returns the current season
func (sc *SeasonalCycle) GetSeason() Season {
	return sc.CurrentSeason
}

// GetYear returns the current year
func (sc *SeasonalCycle) GetYear() int {
	return sc.Year
}

// GetDayOfSeason returns the current day within the season
func (sc *SeasonalCycle) GetDayOfSeason() int {
	return sc.DayOfSeason
}

// GetSeasonProgress returns progress through current season (0.0 to 1.0)
func (sc *SeasonalCycle) GetSeasonProgress() float64 {
	return sc.SeasonProgress
}

// GetDaylightHours returns current daylight duration
func (sc *SeasonalCycle) GetDaylightHours() float64 {
	return sc.Effects.DaylightHours
}

// IsDaytime checks if a given hour (0-24) is during daylight
func (sc *SeasonalCycle) IsDaytime(hour float64) bool {
	sunrise := (24.0 - sc.Effects.DaylightHours) / 2.0
	sunset := sunrise + sc.Effects.DaylightHours
	return hour >= sunrise && hour < sunset
}

// GetSunrise returns the sunrise hour
func (sc *SeasonalCycle) GetSunrise() float64 {
	return (24.0 - sc.Effects.DaylightHours) / 2.0
}

// GetSunset returns the sunset hour
func (sc *SeasonalCycle) GetSunset() float64 {
	return sc.GetSunrise() + sc.Effects.DaylightHours
}

// SetSeason forces a specific season (for testing or events)
func (sc *SeasonalCycle) SetSeason(season Season, dayOfSeason int) {
	sc.CurrentSeason = season
	sc.DayOfSeason = dayOfSeason
	if sc.DayOfSeason < 1 {
		sc.DayOfSeason = 1
	}
	if sc.DayOfSeason > sc.DaysPerSeason {
		sc.DayOfSeason = sc.DaysPerSeason
	}
	sc.SeasonProgress = float64(sc.DayOfSeason) / float64(sc.DaysPerSeason)
	sc.Effects = sc.calculateSeasonalEffects()
}

// GetDescription returns a human-readable season description
func (sc *SeasonalCycle) GetDescription() string {
	phase := ""
	if sc.SeasonProgress < 0.33 {
		phase = "Early "
	} else if sc.SeasonProgress > 0.66 {
		phase = "Late "
	} else {
		phase = "Mid-"
	}

	return phase + sc.CurrentSeason.String()
}

// GetTotalGameDays returns total game days elapsed
func (sc *SeasonalCycle) GetTotalGameDays() int {
	yearsComplete := sc.Year - 1
	seasonsComplete := int(sc.CurrentSeason)
	return yearsComplete*4*sc.DaysPerSeason + seasonsComplete*sc.DaysPerSeason + sc.DayOfSeason
}
