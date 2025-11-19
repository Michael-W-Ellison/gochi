package environment

import (
	"testing"
)

func TestNewEnvironmentManager(t *testing.T) {
	em := NewEnvironmentManager(nil)

	if em == nil {
		t.Fatal("NewEnvironmentManager returned nil")
	}

	if !em.IsInitialized() {
		t.Error("Expected environment manager to be initialized")
	}
}

func TestEnvironmentManagerWithConfig(t *testing.T) {
	config := &EnvironmentConfig{
		StartSeason:           SeasonSummer,
		SeasonLengthHours:     200.0,
		SeasonalIntensity:     0.5,
		WeatherVolatility:     0.7,
		WeatherChangeInterval: 2.0,
		StartingBiome:         BiomeForest,
	}

	em := NewEnvironmentManager(config)

	if em.GetCurrentSeason() != SeasonSummer {
		t.Errorf("Expected season Summer, got %v", em.GetCurrentSeason())
	}
}

func TestEnvironmentManagerUpdate(t *testing.T) {
	em := NewEnvironmentManager(nil)

	// Update multiple times
	for i := 0; i < 10; i++ {
		em.Update(1.0)
	}

	// Should complete without errors
	if em.totalGameTime == 0 {
		t.Error("Expected total game time to increase")
	}
}

func TestGetWeatherConditions(t *testing.T) {
	em := NewEnvironmentManager(nil)

	conditions := em.GetWeatherConditions()

	if conditions.Temperature < -50 || conditions.Temperature > 50 {
		t.Errorf("Temperature out of reasonable range: %v", conditions.Temperature)
	}
}

func TestGetCurrentSeason(t *testing.T) {
	config := DefaultEnvironmentConfig()
	config.StartSeason = SeasonWinter

	em := NewEnvironmentManager(config)

	if em.GetCurrentSeason() != SeasonWinter {
		t.Errorf("Expected Winter, got %v", em.GetCurrentSeason())
	}
}

func TestGetCurrentLocation(t *testing.T) {
	em := NewEnvironmentManager(nil)

	location := em.GetCurrentLocation()
	if location == nil {
		t.Fatal("Expected a current location")
	}

	// Default location should be Home Meadow
	if location.Biome != BiomeGrassland {
		t.Errorf("Expected Grassland biome, got %v", location.Biome)
	}
}

func TestTravelToLocation(t *testing.T) {
	em := NewEnvironmentManager(nil)
	em.CreateStandardWorld()

	// Travel to forest
	success := em.TravelToLocation("Whispering Forest")
	if !success {
		t.Error("Failed to travel to Whispering Forest")
	}

	// Check current location
	location := em.GetCurrentLocation()
	if location.Biome != BiomeForest {
		t.Errorf("Expected Forest biome, got %v", location.Biome)
	}

	// Try to travel to non-existent location
	success = em.TravelToLocation("Nonexistent Place")
	if success {
		t.Error("Should not be able to travel to nonexistent location")
	}
}

func TestAddLocation(t *testing.T) {
	em := NewEnvironmentManager(nil)

	newLoc := NewLocation("Test Cave", BiomeCave, Coordinates{X: 50, Y: 50})
	em.AddLocation(newLoc)

	// Should be able to travel there now
	success := em.TravelToLocation("Test Cave")
	if !success {
		t.Error("Failed to travel to added location")
	}
}

func TestGetDiscoveredLocations(t *testing.T) {
	em := NewEnvironmentManager(nil)
	em.CreateStandardWorld()

	// Initially only home should be discovered
	discovered := em.GetDiscoveredLocations()
	if len(discovered) != 1 {
		t.Errorf("Expected 1 discovered location, got %d", len(discovered))
	}

	// Travel to a location to discover it
	em.TravelToLocation("Whispering Forest")
	discovered = em.GetDiscoveredLocations()
	if len(discovered) != 2 {
		t.Errorf("Expected 2 discovered locations, got %d", len(discovered))
	}
}

func TestCalculateEnvironmentalEffects(t *testing.T) {
	em := NewEnvironmentManager(nil)

	effects := em.CalculateEnvironmentalEffects()

	// Check that all fields have reasonable values
	if effects.FoodAvailability < 0 || effects.FoodAvailability > 1 {
		t.Errorf("Food availability out of range: %v", effects.FoodAvailability)
	}

	if effects.WaterAvailability < 0 || effects.WaterAvailability > 1 {
		t.Errorf("Water availability out of range: %v", effects.WaterAvailability)
	}

	if effects.OverallComfort < 0 || effects.OverallComfort > 1 {
		t.Errorf("Overall comfort out of range: %v", effects.OverallComfort)
	}

	if effects.MoodModifier < -1 || effects.MoodModifier > 1 {
		t.Errorf("Mood modifier out of range: %v", effects.MoodModifier)
	}
}

func TestGetEnvironmentDescription(t *testing.T) {
	em := NewEnvironmentManager(nil)

	desc := em.GetEnvironmentDescription()
	if desc == "" {
		t.Error("Expected non-empty environment description")
	}
}

func TestIsOutdoorActivityRecommended(t *testing.T) {
	em := NewEnvironmentManager(nil)

	// In default conditions, outdoor activity should generally be safe
	// (Though this depends on random weather generation)
	recommended := em.IsOutdoorActivityRecommended()
	t.Logf("Outdoor activity recommended: %v", recommended)

	// Force stormy weather
	em.SetWeather(WeatherStormy)
	recommended = em.IsOutdoorActivityRecommended()
	if recommended {
		t.Error("Outdoor activity should not be recommended in stormy weather")
	}
}

func TestSetSeason(t *testing.T) {
	em := NewEnvironmentManager(nil)

	em.SetSeason(SeasonWinter)

	if em.GetCurrentSeason() != SeasonWinter {
		t.Errorf("Expected Winter, got %v", em.GetCurrentSeason())
	}
}

func TestSetWeather(t *testing.T) {
	em := NewEnvironmentManager(nil)

	em.SetWeather(WeatherSnowy)

	conditions := em.GetWeatherConditions()
	if conditions.Type != WeatherSnowy {
		t.Errorf("Expected Snowy weather, got %v", conditions.Type)
	}
}

func TestGetSeasonalBreedingBonus(t *testing.T) {
	em := NewEnvironmentManager(nil)

	// Set to spring (breeding season)
	em.SetSeason(SeasonSpring)
	em.seasons.seasonProgress = 0.5 // Mid-season to avoid transition blending
	bonus := em.GetSeasonalBreedingBonus()

	if bonus < 0.5 {
		t.Errorf("Expected high breeding bonus in spring, got %v", bonus)
	}

	// Set to winter (not breeding season)
	em.SetSeason(SeasonWinter)
	em.seasons.seasonProgress = 0.5
	bonus = em.GetSeasonalBreedingBonus()

	if bonus > 0.3 {
		t.Errorf("Expected low breeding bonus in winter, got %v", bonus)
	}
}

func TestCalculateSeasonalMoodForPet(t *testing.T) {
	em := NewEnvironmentManager(nil)

	em.SetSeason(SeasonSpring)
	em.seasons.seasonProgress = 0.5 // Mid-season to avoid transition blending

	// Pet prefers spring
	mood := em.CalculateSeasonalMoodForPet(SeasonSpring)
	if mood <= 0 {
		t.Errorf("Expected positive mood in preferred season, got %v", mood)
	}

	// Pet prefers autumn (opposite season to spring)
	mood = em.CalculateSeasonalMoodForPet(SeasonAutumn)
	if mood >= 0 {
		t.Errorf("Expected negative mood in opposite season, got %v", mood)
	}
}

func TestGetResourceAvailability(t *testing.T) {
	em := NewEnvironmentManager(nil)

	availability := em.GetResourceAvailability()

	if availability < 0 || availability > 1 {
		t.Errorf("Resource availability out of range: %v", availability)
	}
}

func TestDepleteLocationResources(t *testing.T) {
	em := NewEnvironmentManager(nil)

	initialAvailability := em.GetResourceAvailability()

	em.DepleteLocationResources(0.3)

	newAvailability := em.GetResourceAvailability()

	if newAvailability >= initialAvailability {
		t.Error("Expected resource availability to decrease after depletion")
	}
}

func TestGetStats(t *testing.T) {
	em := NewEnvironmentManager(nil)

	stats := em.GetStats()

	// Check that stats contains various keys
	if len(stats) == 0 {
		t.Error("Expected non-empty stats map")
	}

	// Check for some specific keys
	if _, exists := stats["overall_comfort"]; !exists {
		t.Error("Expected stats to contain overall_comfort")
	}

	if _, exists := stats["season_current_season"]; !exists {
		t.Error("Expected stats to contain season_current_season")
	}

	if _, exists := stats["weather_weather_type"]; !exists {
		t.Error("Expected stats to contain weather_weather_type")
	}
}

func TestCreateStandardWorld(t *testing.T) {
	em := NewEnvironmentManager(nil)
	em.CreateStandardWorld()

	// Should have multiple locations
	discovered := em.GetDiscoveredLocations()
	if len(discovered) < 1 {
		t.Error("Expected at least one discovered location after creating standard world")
	}

	// Should be able to travel to various locations
	locations := []string{
		"Whispering Forest",
		"Sunny Beach",
		"Misty Peak",
		"Golden Sands",
	}

	for _, locName := range locations {
		success := em.TravelToLocation(locName)
		if !success {
			t.Errorf("Failed to travel to standard location: %s", locName)
		}
	}
}

func TestGetConfiguration(t *testing.T) {
	config := DefaultEnvironmentConfig()
	em := NewEnvironmentManager(config)

	retrievedConfig := em.GetConfiguration()

	if retrievedConfig.StartSeason != config.StartSeason {
		t.Error("Configuration not preserved")
	}
}

func TestEnvironmentReset(t *testing.T) {
	em := NewEnvironmentManager(nil)

	// Make some changes
	em.Update(1000.0)
	em.SetSeason(SeasonWinter)

	// Reset
	em.Reset()

	// Should be back to initial state
	if em.GetCurrentSeason() != em.config.StartSeason {
		t.Error("Season not reset properly")
	}

	if em.totalGameTime != 0 {
		t.Error("Game time not reset properly")
	}
}

func TestEnvironmentIntegration(t *testing.T) {
	// Test that all systems work together
	em := NewEnvironmentManager(nil)
	em.CreateStandardWorld()

	// Simulate game time
	for i := 0; i < 100; i++ {
		em.Update(100.0) // 100 seconds per update
	}

	// Check that everything still works
	effects := em.CalculateEnvironmentalEffects()
	if effects.OverallComfort < 0 || effects.OverallComfort > 1 {
		t.Errorf("Overall comfort out of range after updates: %v", effects.OverallComfort)
	}

	// Travel around
	em.TravelToLocation("Whispering Forest")
	em.Update(100.0)

	em.TravelToLocation("Sunny Beach")
	em.Update(100.0)

	// Get final stats
	stats := em.GetStats()
	if len(stats) == 0 {
		t.Error("Expected stats after full integration test")
	}
}

func TestDefaultEnvironmentConfig(t *testing.T) {
	config := DefaultEnvironmentConfig()

	if config.StartSeason != SeasonSpring {
		t.Error("Expected default start season to be Spring")
	}

	if config.SeasonLengthHours <= 0 {
		t.Error("Expected positive season length")
	}

	if config.SeasonalIntensity < 0 || config.SeasonalIntensity > 1 {
		t.Error("Seasonal intensity out of range")
	}
}

func TestGetNearbyLocations(t *testing.T) {
	em := NewEnvironmentManager(nil)
	em.CreateStandardWorld()

	// From home, should have some nearby locations
	nearby := em.GetNearbyLocations(100.0)

	if len(nearby) == 0 {
		t.Error("Expected some nearby locations from home")
	}

	// Very small range should have fewer locations
	veryClose := em.GetNearbyLocations(10.0)
	if len(veryClose) >= len(nearby) {
		t.Log("Note: Small range has same or more locations (possible if locations are very close)")
	}
}
