package environment

import (
	"math"
	"sync"
)

// Season represents the four seasons
type Season int

const (
	SeasonSpring Season = iota
	SeasonSummer
	SeasonAutumn
	SeasonWinter
)

// String returns the string representation of season
func (s Season) String() string {
	switch s {
	case SeasonSpring:
		return "Spring"
	case SeasonSummer:
		return "Summer"
	case SeasonAutumn:
		return "Autumn"
	case SeasonWinter:
		return "Winter"
	default:
		return "Unknown"
	}
}

// SeasonalEffects represents how a season affects the environment
type SeasonalEffects struct {
	TemperatureModifier   float64 // Additive modifier
	DaylightHours         float64 // Hours of daylight
	GrowthRate            float64 // Plant/food growth multiplier
	ActivityLevel         float64 // Natural activity level
	SocialityBonus        float64 // Bonus to social interactions
	RestRequirement       float64 // How much rest is needed
	FoodAvailability      float64 // Base food availability
	WaterAvailability     float64 // Base water availability
	ShelterImportance     float64 // How important shelter is
	MigrationUrge         float64 // Urge to travel/explore
	BreedingSeasonBonus   float64 // Bonus to breeding compatibility
}

// SeasonalCycle manages the progression of seasons
type SeasonalCycle struct {
	mu sync.RWMutex

	currentSeason   Season
	seasonProgress  float64 // 0-1 through current season
	seasonLength    float64 // Game hours per season
	totalGameTime   float64 // Total elapsed game time

	// Customization
	seasonalIntensity float64 // 0-1, how pronounced seasonal changes are
	enableTransitions bool    // Whether to smoothly transition between seasons
}

// NewSeasonalCycle creates a new seasonal cycle manager
func NewSeasonalCycle(startSeason Season, seasonLengthHours float64) *SeasonalCycle {
	return &SeasonalCycle{
		currentSeason:     startSeason,
		seasonProgress:    0.0,
		seasonLength:      seasonLengthHours,
		totalGameTime:     0.0,
		seasonalIntensity: 1.0,
		enableTransitions: true,
	}
}

// Update advances the seasonal cycle
func (sc *SeasonalCycle) Update(deltaTime float64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Convert delta time to hours
	deltaHours := deltaTime / 3600.0

	sc.totalGameTime += deltaHours
	sc.seasonProgress += deltaHours / sc.seasonLength

	// Check if season has changed
	if sc.seasonProgress >= 1.0 {
		sc.seasonProgress = 0.0
		sc.currentSeason = sc.getNextSeason()
	}
}

// getNextSeason returns the next season in the cycle
func (sc *SeasonalCycle) getNextSeason() Season {
	switch sc.currentSeason {
	case SeasonSpring:
		return SeasonSummer
	case SeasonSummer:
		return SeasonAutumn
	case SeasonAutumn:
		return SeasonWinter
	case SeasonWinter:
		return SeasonSpring
	default:
		return SeasonSpring
	}
}

// GetCurrentSeason returns the current season
func (sc *SeasonalCycle) GetCurrentSeason() Season {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.currentSeason
}

// GetSeasonProgress returns progress through current season (0-1)
func (sc *SeasonalCycle) GetSeasonProgress() float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.seasonProgress
}

// GetSeasonalEffects returns the environmental effects of the current season
func (sc *SeasonalCycle) GetSeasonalEffects() SeasonalEffects {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Get base effects for current season
	baseEffects := sc.getSeasonBaseEffects(sc.currentSeason)

	// If transitions enabled, blend with next season near boundaries
	if sc.enableTransitions {
		if sc.seasonProgress < 0.1 {
			// Transitioning from previous season
			prevEffects := sc.getSeasonBaseEffects(sc.getPreviousSeason())
			blendFactor := sc.seasonProgress * 10 // 0-1 over first 10%
			baseEffects = sc.blendEffects(prevEffects, baseEffects, blendFactor)
		} else if sc.seasonProgress > 0.9 {
			// Transitioning to next season
			nextEffects := sc.getSeasonBaseEffects(sc.getNextSeason())
			blendFactor := (sc.seasonProgress - 0.9) * 10 // 0-1 over last 10%
			baseEffects = sc.blendEffects(baseEffects, nextEffects, blendFactor)
		}
	}

	// Apply intensity modifier
	baseEffects = sc.applyIntensity(baseEffects)

	return baseEffects
}

// getSeasonBaseEffects returns base effects for a season
func (sc *SeasonalCycle) getSeasonBaseEffects(season Season) SeasonalEffects {
	switch season {
	case SeasonSpring:
		return SeasonalEffects{
			TemperatureModifier:   0.0,
			DaylightHours:         12.0,
			GrowthRate:            1.5,
			ActivityLevel:         0.8,
			SocialityBonus:        0.3,
			RestRequirement:       0.8,
			FoodAvailability:      0.9,
			WaterAvailability:     0.9,
			ShelterImportance:     0.5,
			MigrationUrge:         0.4,
			BreedingSeasonBonus:   0.8,
		}

	case SeasonSummer:
		return SeasonalEffects{
			TemperatureModifier:   10.0,
			DaylightHours:         14.0,
			GrowthRate:            1.2,
			ActivityLevel:         0.9,
			SocialityBonus:        0.5,
			RestRequirement:       0.7,
			FoodAvailability:      1.0,
			WaterAvailability:     0.7,
			ShelterImportance:     0.3,
			MigrationUrge:         0.6,
			BreedingSeasonBonus:   0.4,
		}

	case SeasonAutumn:
		return SeasonalEffects{
			TemperatureModifier:   0.0,
			DaylightHours:         12.0,
			GrowthRate:            0.8,
			ActivityLevel:         0.7,
			SocialityBonus:        0.2,
			RestRequirement:       0.9,
			FoodAvailability:      0.8,
			WaterAvailability:     0.8,
			ShelterImportance:     0.6,
			MigrationUrge:         0.3,
			BreedingSeasonBonus:   0.2,
		}

	case SeasonWinter:
		return SeasonalEffects{
			TemperatureModifier:   -10.0,
			DaylightHours:         10.0,
			GrowthRate:            0.3,
			ActivityLevel:         0.5,
			SocialityBonus:        0.1,
			RestRequirement:       1.2,
			FoodAvailability:      0.5,
			WaterAvailability:     0.9,
			ShelterImportance:     1.0,
			MigrationUrge:         0.1,
			BreedingSeasonBonus:   0.0,
		}

	default:
		return SeasonalEffects{
			TemperatureModifier:   0.0,
			DaylightHours:         12.0,
			GrowthRate:            1.0,
			ActivityLevel:         0.7,
			SocialityBonus:        0.3,
			RestRequirement:       1.0,
			FoodAvailability:      0.8,
			WaterAvailability:     0.8,
			ShelterImportance:     0.5,
			MigrationUrge:         0.3,
			BreedingSeasonBonus:   0.3,
		}
	}
}

// getPreviousSeason returns the previous season in the cycle
func (sc *SeasonalCycle) getPreviousSeason() Season {
	switch sc.currentSeason {
	case SeasonSpring:
		return SeasonWinter
	case SeasonSummer:
		return SeasonSpring
	case SeasonAutumn:
		return SeasonSummer
	case SeasonWinter:
		return SeasonAutumn
	default:
		return SeasonSpring
	}
}

// blendEffects smoothly blends two seasonal effects
func (sc *SeasonalCycle) blendEffects(from, to SeasonalEffects, t float64) SeasonalEffects {
	return SeasonalEffects{
		TemperatureModifier:   lerp(from.TemperatureModifier, to.TemperatureModifier, t),
		DaylightHours:         lerp(from.DaylightHours, to.DaylightHours, t),
		GrowthRate:            lerp(from.GrowthRate, to.GrowthRate, t),
		ActivityLevel:         lerp(from.ActivityLevel, to.ActivityLevel, t),
		SocialityBonus:        lerp(from.SocialityBonus, to.SocialityBonus, t),
		RestRequirement:       lerp(from.RestRequirement, to.RestRequirement, t),
		FoodAvailability:      lerp(from.FoodAvailability, to.FoodAvailability, t),
		WaterAvailability:     lerp(from.WaterAvailability, to.WaterAvailability, t),
		ShelterImportance:     lerp(from.ShelterImportance, to.ShelterImportance, t),
		MigrationUrge:         lerp(from.MigrationUrge, to.MigrationUrge, t),
		BreedingSeasonBonus:   lerp(from.BreedingSeasonBonus, to.BreedingSeasonBonus, t),
	}
}

// applyIntensity scales seasonal effects by intensity setting
func (sc *SeasonalCycle) applyIntensity(effects SeasonalEffects) SeasonalEffects {
	// Temperature modifier is scaled linearly
	effects.TemperatureModifier *= sc.seasonalIntensity

	// Other effects are scaled toward neutral (1.0 or 0.5)
	effects.GrowthRate = 1.0 + (effects.GrowthRate-1.0)*sc.seasonalIntensity
	effects.ActivityLevel = 0.7 + (effects.ActivityLevel-0.7)*sc.seasonalIntensity
	effects.SocialityBonus *= sc.seasonalIntensity
	effects.RestRequirement = 1.0 + (effects.RestRequirement-1.0)*sc.seasonalIntensity
	effects.FoodAvailability = 0.8 + (effects.FoodAvailability-0.8)*sc.seasonalIntensity
	effects.WaterAvailability = 0.8 + (effects.WaterAvailability-0.8)*sc.seasonalIntensity
	effects.ShelterImportance *= sc.seasonalIntensity
	effects.MigrationUrge *= sc.seasonalIntensity
	effects.BreedingSeasonBonus *= sc.seasonalIntensity

	return effects
}

// SetSeasonalIntensity adjusts how pronounced seasonal changes are
func (sc *SeasonalCycle) SetSeasonalIntensity(intensity float64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}
	sc.seasonalIntensity = intensity
}

// SetSeasonLength changes how long each season lasts (in game hours)
func (sc *SeasonalCycle) SetSeasonLength(hours float64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if hours < 1 {
		hours = 1
	}
	sc.seasonLength = hours
}

// SetSeason immediately changes to a specific season
func (sc *SeasonalCycle) SetSeason(season Season) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.currentSeason = season
	sc.seasonProgress = 0.0
}

// GetDaylightHours returns current hours of daylight
func (sc *SeasonalCycle) GetDaylightHours() float64 {
	effects := sc.GetSeasonalEffects()
	return effects.DaylightHours
}

// IsBreedingSeason returns whether it's a good time for breeding
func (sc *SeasonalCycle) IsBreedingSeason() bool {
	effects := sc.GetSeasonalEffects()
	return effects.BreedingSeasonBonus > 0.5
}

// GetSeasonDescription returns a text description of the current season
func (sc *SeasonalCycle) GetSeasonDescription() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	progress := sc.seasonProgress
	var phase string

	if progress < 0.25 {
		phase = "early"
	} else if progress < 0.5 {
		phase = "mid"
	} else if progress < 0.75 {
		phase = "late"
	} else {
		phase = "end of"
	}

	return phase + " " + sc.currentSeason.String()
}

// GetYearProgress returns progress through the full year cycle (0-1)
func (sc *SeasonalCycle) GetYearProgress() float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	seasonIndex := float64(sc.currentSeason)
	return (seasonIndex + sc.seasonProgress) / 4.0
}

// GetTotalGameDays returns total elapsed game days
func (sc *SeasonalCycle) GetTotalGameDays() float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.totalGameTime / 24.0
}

// GetTotalGameYears returns total elapsed game years
func (sc *SeasonalCycle) GetTotalGameYears() float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	totalSeasons := sc.totalGameTime / sc.seasonLength
	return totalSeasons / 4.0
}

// CalculateSeasonalMoodEffect returns mood modifier based on season
func (sc *SeasonalCycle) CalculateSeasonalMoodEffect(preferredSeason Season) float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Pets are happier in their preferred season
	if sc.currentSeason == preferredSeason {
		return 0.2 * sc.seasonalIntensity
	}

	// Slightly less happy in adjacent seasons
	if (sc.currentSeason == SeasonSpring && preferredSeason == SeasonSummer) ||
		(sc.currentSeason == SeasonSummer && preferredSeason == SeasonSpring) ||
		(sc.currentSeason == SeasonSummer && preferredSeason == SeasonAutumn) ||
		(sc.currentSeason == SeasonAutumn && preferredSeason == SeasonSummer) ||
		(sc.currentSeason == SeasonAutumn && preferredSeason == SeasonWinter) ||
		(sc.currentSeason == SeasonWinter && preferredSeason == SeasonAutumn) ||
		(sc.currentSeason == SeasonWinter && preferredSeason == SeasonSpring) ||
		(sc.currentSeason == SeasonSpring && preferredSeason == SeasonWinter) {
		return 0.0
	}

	// Unhappy in opposite season
	return -0.15 * sc.seasonalIntensity
}

// GetStats returns seasonal cycle statistics
func (sc *SeasonalCycle) GetStats() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	effects := sc.GetSeasonalEffects()

	return map[string]interface{}{
		"current_season":      sc.currentSeason.String(),
		"season_progress":     math.Round(sc.seasonProgress*100) / 100,
		"year_progress":       math.Round(sc.GetYearProgress()*100) / 100,
		"total_game_days":     math.Round(sc.GetTotalGameDays()*10) / 10,
		"total_game_years":    math.Round(sc.GetTotalGameYears()*100) / 100,
		"daylight_hours":      math.Round(effects.DaylightHours*10) / 10,
		"food_availability":   math.Round(effects.FoodAvailability*100) / 100,
		"breeding_season":     sc.IsBreedingSeason(),
		"seasonal_intensity":  math.Round(sc.seasonalIntensity*100) / 100,
	}
}

// GetCurrentMonth returns a month number (1-12) based on season and progress
func (sc *SeasonalCycle) GetCurrentMonth() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Map seasons to months (assuming Spring starts at month 3)
	baseMonth := 0
	switch sc.currentSeason {
	case SeasonSpring:
		baseMonth = 3 // March
	case SeasonSummer:
		baseMonth = 6 // June
	case SeasonAutumn:
		baseMonth = 9 // September
	case SeasonWinter:
		baseMonth = 12 // December
	}

	// Add progress through season (3 months per season)
	monthOffset := int(sc.seasonProgress * 3)
	month := baseMonth + monthOffset

	// Wrap around
	if month > 12 {
		month -= 12
	}
	if month < 1 {
		month = 1
	}

	return month
}

// GetSeasonForMonth returns which season a month falls in
func GetSeasonForMonth(month int) Season {
	switch {
	case month >= 3 && month <= 5:
		return SeasonSpring
	case month >= 6 && month <= 8:
		return SeasonSummer
	case month >= 9 && month <= 11:
		return SeasonAutumn
	default:
		return SeasonWinter
	}
}

// AdvanceToNextSeason immediately advances to the next season
func (sc *SeasonalCycle) AdvanceToNextSeason() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.currentSeason = sc.getNextSeason()
	sc.seasonProgress = 0.0
}
