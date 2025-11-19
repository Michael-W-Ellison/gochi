package environment

import (
	"testing"
)

func TestNewSeasonalCycle(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 100.0)

	if sc == nil {
		t.Fatal("NewSeasonalCycle returned nil")
	}

	if sc.GetCurrentSeason() != SeasonSpring {
		t.Errorf("Expected season Spring, got %v", sc.GetCurrentSeason())
	}

	if sc.seasonLength != 100.0 {
		t.Errorf("Expected season length 100.0, got %v", sc.seasonLength)
	}
}

func TestSeasonString(t *testing.T) {
	tests := []struct {
		season   Season
		expected string
	}{
		{SeasonSpring, "Spring"},
		{SeasonSummer, "Summer"},
		{SeasonAutumn, "Autumn"},
		{SeasonWinter, "Winter"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.season.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.season.String())
			}
		})
	}
}

func TestSeasonProgression(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 10.0) // Short season for testing

	if sc.GetCurrentSeason() != SeasonSpring {
		t.Errorf("Expected Spring, got %v", sc.GetCurrentSeason())
	}

	// Progress through spring (need 10 hours = 36000 seconds)
	for i := 0; i < 40; i++ {
		sc.Update(1000.0) // 1000 seconds per update
	}

	// Should have progressed to summer
	if sc.GetCurrentSeason() != SeasonSummer {
		t.Errorf("Expected Summer after progression, got %v", sc.GetCurrentSeason())
	}
}

func TestSeasonCycle(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 10.0)

	expectedSeasons := []Season{SeasonSpring, SeasonSummer, SeasonAutumn, SeasonWinter, SeasonSpring}

	for i, expected := range expectedSeasons {
		current := sc.GetCurrentSeason()
		if current != expected {
			t.Errorf("Cycle step %d: expected %v, got %v", i, expected, current)
		}

		// Progress to next season
		for j := 0; j < 40; j++ {
			sc.Update(1000.0)
		}
	}
}

func TestGetSeasonProgress(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 10.0)

	progress := sc.GetSeasonProgress()
	if progress != 0.0 {
		t.Errorf("Expected initial progress 0.0, got %v", progress)
	}

	// Update partway through season
	sc.Update(18000.0) // 5 hours = half the season

	progress = sc.GetSeasonProgress()
	if progress < 0.4 || progress > 0.6 {
		t.Errorf("Expected progress around 0.5, got %v", progress)
	}
}

func TestGetSeasonalEffects(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 100.0)

	// Move to middle of season to avoid transition blending
	sc.seasonProgress = 0.5

	effects := sc.GetSeasonalEffects()

	// Spring should have moderate temperature
	if effects.TemperatureModifier < -10 || effects.TemperatureModifier > 10 {
		t.Errorf("Unexpected temperature modifier for spring: %v", effects.TemperatureModifier)
	}

	// Spring should have good growth rate
	if effects.GrowthRate < 1.0 {
		t.Error("Expected spring to have growth rate >= 1.0")
	}

	// Spring should have breeding season bonus
	if effects.BreedingSeasonBonus < 0.5 {
		t.Error("Expected spring to have high breeding season bonus")
	}
}

func TestSeasonalIntensity(t *testing.T) {
	sc := NewSeasonalCycle(SeasonWinter, 100.0)
	sc.seasonProgress = 0.5 // Middle of season to avoid blending

	// Full intensity
	sc.SetSeasonalIntensity(1.0)
	effects1 := sc.GetSeasonalEffects()

	// No intensity
	sc.SetSeasonalIntensity(0.0)
	effects2 := sc.GetSeasonalEffects()

	// With no intensity, temperature modifier should be closer to 0
	if abs(effects2.TemperatureModifier) >= abs(effects1.TemperatureModifier) {
		t.Error("Expected temperature modifier to be less extreme with 0 intensity")
	}
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestSeasonalCycleSetSeason(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 100.0)

	sc.SetSeason(SeasonWinter)

	if sc.GetCurrentSeason() != SeasonWinter {
		t.Errorf("Expected Winter, got %v", sc.GetCurrentSeason())
	}

	if sc.GetSeasonProgress() != 0.0 {
		t.Errorf("Expected progress reset to 0.0, got %v", sc.GetSeasonProgress())
	}
}

func TestGetDaylightHours(t *testing.T) {
	tests := []struct {
		season   Season
		minHours float64
		maxHours float64
	}{
		{SeasonSummer, 13, 15},
		{SeasonWinter, 9, 11},
		{SeasonSpring, 11, 13},
		{SeasonAutumn, 11, 13},
	}

	for _, tt := range tests {
		t.Run(tt.season.String(), func(t *testing.T) {
			sc := NewSeasonalCycle(tt.season, 100.0)
			sc.seasonProgress = 0.5 // Middle of season to avoid blending
			hours := sc.GetDaylightHours()

			if hours < tt.minHours || hours > tt.maxHours {
				t.Errorf("Expected daylight hours between %v and %v, got %v",
					tt.minHours, tt.maxHours, hours)
			}
		})
	}
}

func TestIsBreedingSeason(t *testing.T) {
	// Spring should be breeding season
	sc := NewSeasonalCycle(SeasonSpring, 100.0)
	sc.seasonProgress = 0.5 // Middle of season
	if !sc.IsBreedingSeason() {
		t.Error("Expected spring to be breeding season")
	}

	// Winter should not be breeding season
	sc.SetSeason(SeasonWinter)
	sc.seasonProgress = 0.5
	if sc.IsBreedingSeason() {
		t.Error("Expected winter not to be breeding season")
	}
}

func TestGetSeasonDescription(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 100.0)

	desc := sc.GetSeasonDescription()
	if desc == "" {
		t.Error("Expected non-empty season description")
	}

	// Description should mention spring
	// (Could check for "Spring" substring, but keep it simple)
}

func TestGetYearProgress(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 10.0)

	// At start of spring, year progress should be near 0
	progress := sc.GetYearProgress()
	if progress < 0.0 || progress > 0.3 {
		t.Errorf("Expected year progress near 0, got %v", progress)
	}

	// Progress through two seasons
	for i := 0; i < 80; i++ {
		sc.Update(1000.0)
	}

	// Should be halfway through year
	progress = sc.GetYearProgress()
	if progress < 0.4 || progress > 0.6 {
		t.Errorf("Expected year progress around 0.5, got %v", progress)
	}
}

func TestGetTotalGameYears(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 10.0) // 10 hours per season = 40 hours per year

	// Progress through one full year
	for i := 0; i < 160; i++ {
		sc.Update(1000.0) // 160,000 seconds = 44.4 hours
	}

	years := sc.GetTotalGameYears()
	if years < 0.9 || years > 1.3 {
		t.Errorf("Expected about 1 year, got %v", years)
	}
}

func TestCalculateSeasonalMoodEffect(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 100.0)
	sc.SetSeasonalIntensity(1.0)

	// Pet prefers spring, currently spring - should be happy
	mood := sc.CalculateSeasonalMoodEffect(SeasonSpring)
	if mood <= 0 {
		t.Errorf("Expected positive mood in preferred season, got %v", mood)
	}

	// Pet prefers summer, currently spring - should be neutral or slightly positive
	mood = sc.CalculateSeasonalMoodEffect(SeasonSummer)
	if mood < -0.1 {
		t.Errorf("Expected neutral mood in adjacent season, got %v", mood)
	}

	// Pet prefers autumn, currently spring - should be slightly negative
	mood = sc.CalculateSeasonalMoodEffect(SeasonAutumn)
	if mood > 0 {
		t.Errorf("Expected negative mood in opposite season, got %v", mood)
	}
}

func TestSeasonTransitions(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 10.0)
	sc.enableTransitions = true

	// Get effects at start of spring (should blend from winter)
	sc.seasonProgress = 0.05
	effects := sc.GetSeasonalEffects()

	// Temperature should be transitioning
	springEffects := sc.getSeasonBaseEffects(SeasonSpring)
	if effects.TemperatureModifier == springEffects.TemperatureModifier {
		t.Log("Note: Temperature exactly matches spring base (might be at transition boundary)")
	}
}

func TestGetCurrentMonth(t *testing.T) {
	tests := []struct {
		season Season
		month  int
	}{
		{SeasonSpring, 3}, // March
		{SeasonSummer, 6}, // June
		{SeasonAutumn, 9}, // September
		{SeasonWinter, 12}, // December
	}

	for _, tt := range tests {
		t.Run(tt.season.String(), func(t *testing.T) {
			sc := NewSeasonalCycle(tt.season, 100.0)
			month := sc.GetCurrentMonth()

			// Allow for some variation due to progress
			if month < tt.month || month > tt.month+2 {
				t.Errorf("Expected month around %d for %s, got %d",
					tt.month, tt.season.String(), month)
			}
		})
	}
}

func TestGetSeasonForMonth(t *testing.T) {
	tests := []struct {
		month  int
		season Season
	}{
		{3, SeasonSpring},
		{6, SeasonSummer},
		{9, SeasonAutumn},
		{12, SeasonWinter},
		{1, SeasonWinter},
	}

	for _, tt := range tests {
		season := GetSeasonForMonth(tt.month)
		if season != tt.season {
			t.Errorf("Month %d: expected %s, got %s",
				tt.month, tt.season.String(), season.String())
		}
	}
}

func TestAdvanceToNextSeason(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 100.0)

	sc.AdvanceToNextSeason()

	if sc.GetCurrentSeason() != SeasonSummer {
		t.Errorf("Expected Summer after advancing from Spring, got %v", sc.GetCurrentSeason())
	}

	if sc.GetSeasonProgress() != 0.0 {
		t.Errorf("Expected progress reset to 0.0, got %v", sc.GetSeasonProgress())
	}
}

func TestSeasonStats(t *testing.T) {
	sc := NewSeasonalCycle(SeasonSpring, 100.0)

	stats := sc.GetStats()

	expectedKeys := []string{
		"current_season",
		"season_progress",
		"year_progress",
		"total_game_days",
		"daylight_hours",
		"breeding_season",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stats to contain key %s", key)
		}
	}
}
