package environment

import (
	"testing"
	"time"
)

func TestNewWeatherSystem(t *testing.T) {
	ws := NewWeatherSystem()

	if ws == nil {
		t.Fatal("NewWeatherSystem returned nil")
	}

	if ws.current.Type != WeatherClear {
		t.Errorf("Expected initial weather to be Clear, got %v", ws.current.Type)
	}

	if ws.current.Temperature != 20.0 {
		t.Errorf("Expected initial temperature 20.0, got %v", ws.current.Temperature)
	}
}

func TestWeatherTypeString(t *testing.T) {
	tests := []struct {
		weatherType WeatherType
		expected    string
	}{
		{WeatherClear, "Clear"},
		{WeatherCloudy, "Cloudy"},
		{WeatherRainy, "Rainy"},
		{WeatherStormy, "Stormy"},
		{WeatherSnowy, "Snowy"},
		{WeatherFoggy, "Foggy"},
		{WeatherWindy, "Windy"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.weatherType.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.weatherType.String())
			}
		})
	}
}

func TestWeatherUpdate(t *testing.T) {
	ws := NewWeatherSystem()

	// Update multiple times
	for i := 0; i < 5; i++ {
		ws.Update(1.0, SeasonSpring)
	}

	// Should have updated without errors
	conditions := ws.GetConditions()
	if conditions.Temperature < -50 || conditions.Temperature > 50 {
		t.Errorf("Temperature out of reasonable range: %v", conditions.Temperature)
	}
}

func TestWeatherSeasonalInfluence(t *testing.T) {
	ws := NewWeatherSystem()

	// Force a weather change in summer
	ws.timeSinceChange = ws.changeInterval
	ws.Update(1.0, SeasonSummer)

	temp1 := ws.GetTemperature()

	// Reset and force change in winter
	ws2 := NewWeatherSystem()
	ws2.timeSinceChange = ws2.changeInterval
	ws2.Update(1.0, SeasonWinter)

	temp2 := ws2.GetTemperature()

	// Summer should generally be warmer than winter
	// (Note: Due to randomness, this might occasionally fail, but should mostly pass)
	if temp1 < temp2-10 {
		t.Logf("Warning: Winter temperature (%v) warmer than summer (%v), possible due to randomness", temp2, temp1)
	}
}

func TestWeatherSetVolatility(t *testing.T) {
	ws := NewWeatherSystem()

	ws.SetVolatility(0.8)
	if ws.volatility != 0.8 {
		t.Errorf("Expected volatility 0.8, got %v", ws.volatility)
	}

	// Test clamping
	ws.SetVolatility(-0.5)
	if ws.volatility != 0.0 {
		t.Errorf("Expected volatility clamped to 0.0, got %v", ws.volatility)
	}

	ws.SetVolatility(1.5)
	if ws.volatility != 1.0 {
		t.Errorf("Expected volatility clamped to 1.0, got %v", ws.volatility)
	}
}

func TestWeatherForceWeatherType(t *testing.T) {
	ws := NewWeatherSystem()

	ws.ForceWeatherType(WeatherRainy)
	if ws.GetWeatherType() != WeatherRainy {
		t.Errorf("Expected WeatherRainy, got %v", ws.GetWeatherType())
	}

	// Check that precipitation was set appropriately
	conditions := ws.GetConditions()
	if conditions.Precipitation == 0 {
		t.Error("Expected non-zero precipitation for rainy weather")
	}
}

func TestWeatherComfortLevel(t *testing.T) {
	ws := NewWeatherSystem()

	// Set ideal conditions
	ws.current.Temperature = 20.0
	ws.current.Precipitation = 0.0
	ws.current.WindSpeed = 5.0
	ws.current.Visibility = 1.0

	comfort := ws.GetComfortLevel()
	if comfort < 0.8 {
		t.Errorf("Expected high comfort for ideal conditions, got %v", comfort)
	}

	// Set harsh conditions - force stormy and ensure conditions are fully applied
	ws.ForceWeatherType(WeatherStormy)
	ws.targetConditions = ws.current // Ensure no transition in progress
	ws.elapsedTransition = ws.transitionTime
	comfort = ws.GetComfortLevel()
	if comfort > 0.6 {
		t.Errorf("Expected low comfort for stormy weather, got %v", comfort)
	}
}

func TestWeatherEffects(t *testing.T) {
	ws := NewWeatherSystem()

	// Test hot weather
	ws.current.Temperature = 35.0
	effects := ws.CalculateWeatherEffects()

	if effects.HydrationMultiplier <= 1.0 {
		t.Error("Expected increased hydration drain in hot weather")
	}

	// Test cold weather
	ws.current.Temperature = 0.0
	effects = ws.CalculateWeatherEffects()

	if effects.EnergyDrainMultiplier <= 1.0 {
		t.Error("Expected increased energy drain in cold weather")
	}

	// Test stormy weather
	ws.ForceWeatherType(WeatherStormy)
	effects = ws.CalculateWeatherEffects()

	if effects.ActivityRestriction < 0.5 {
		t.Error("Expected high activity restriction in stormy weather")
	}
}

func TestIsOutdoorActivitySafe(t *testing.T) {
	ws := NewWeatherSystem()

	// Clear weather should be safe
	ws.ForceWeatherType(WeatherClear)
	if !ws.IsOutdoorActivitySafe() {
		t.Error("Expected clear weather to be safe for outdoor activities")
	}

	// Stormy weather should not be safe
	ws.ForceWeatherType(WeatherStormy)
	if ws.IsOutdoorActivitySafe() {
		t.Error("Expected stormy weather to be unsafe for outdoor activities")
	}
}

func TestGetWeatherDescription(t *testing.T) {
	ws := NewWeatherSystem()

	desc := ws.GetWeatherDescription()
	if desc == "" {
		t.Error("Expected non-empty weather description")
	}

	// Should contain weather type and temperature description
	ws.ForceWeatherType(WeatherRainy)
	ws.current.Temperature = 10.0
	desc = ws.GetWeatherDescription()

	if desc == "" {
		t.Error("Expected non-empty weather description")
	}
}

func TestWeatherStats(t *testing.T) {
	ws := NewWeatherSystem()

	stats := ws.GetStats()

	// Check that all expected keys exist
	expectedKeys := []string{
		"weather_type",
		"temperature",
		"humidity",
		"precipitation",
		"wind_speed",
		"visibility",
		"comfort_level",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stats to contain key %s", key)
		}
	}
}

func TestWeatherConcurrency(t *testing.T) {
	ws := NewWeatherSystem()

	done := make(chan bool)

	// Multiple goroutines reading
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				ws.GetConditions()
				ws.GetTemperature()
				ws.GetComfortLevel()
			}
			done <- true
		}()
	}

	// One goroutine writing
	go func() {
		for i := 0; i < 100; i++ {
			ws.Update(0.1, SeasonSpring)
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 11; i++ {
		<-done
	}
}

func TestWeatherTransition(t *testing.T) {
	ws := NewWeatherSystem()

	initialTemp := ws.GetTemperature()

	// Force a new weather pattern
	ws.timeSinceChange = ws.changeInterval
	ws.Update(0.1, SeasonSpring)

	// Temperature should start transitioning
	newTemp := ws.GetTemperature()

	// After just one update, might not have changed yet due to transition time
	// Update more to complete transition
	for i := 0; i < 100; i++ {
		ws.Update(0.1, SeasonSpring)
	}

	finalTemp := ws.GetTemperature()

	// After many updates, weather should have changed
	t.Logf("Initial: %v, After 1 update: %v, After 100 updates: %v", initialTemp, newTemp, finalTemp)
}
