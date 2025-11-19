package environment

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

// WeatherType represents different weather conditions
type WeatherType int

const (
	WeatherClear WeatherType = iota
	WeatherCloudy
	WeatherRainy
	WeatherStormy
	WeatherSnowy
	WeatherFoggy
	WeatherWindy
	WeatherHot
	WeatherCold
)

// String returns the string representation of weather type
func (w WeatherType) String() string {
	switch w {
	case WeatherClear:
		return "Clear"
	case WeatherCloudy:
		return "Cloudy"
	case WeatherRainy:
		return "Rainy"
	case WeatherStormy:
		return "Stormy"
	case WeatherSnowy:
		return "Snowy"
	case WeatherFoggy:
		return "Foggy"
	case WeatherWindy:
		return "Windy"
	case WeatherHot:
		return "Hot"
	case WeatherCold:
		return "Cold"
	default:
		return "Unknown"
	}
}

// WeatherConditions represents current weather state
type WeatherConditions struct {
	Type         WeatherType
	Temperature  float64 // Celsius
	Humidity     float64 // 0-1
	Precipitation float64 // 0-1 (intensity)
	WindSpeed    float64 // km/h
	Pressure     float64 // kPa
	Visibility   float64 // 0-1
}

// WeatherSystem manages dynamic weather simulation
type WeatherSystem struct {
	mu sync.RWMutex

	current          WeatherConditions
	targetConditions WeatherConditions
	transitionTime   float64 // Time to transition to target
	elapsedTransition float64

	// Weather change parameters
	changeInterval    float64 // Hours between weather changes
	timeSinceChange   float64
	volatility        float64 // 0-1, how quickly weather changes
	seasonalInfluence float64 // 0-1, how much season affects weather

	// Random number generator with seed
	rng *rand.Rand
}

// NewWeatherSystem creates a new weather simulation system
func NewWeatherSystem() *WeatherSystem {
	ws := &WeatherSystem{
		current: WeatherConditions{
			Type:         WeatherClear,
			Temperature:  20.0,
			Humidity:     0.5,
			Precipitation: 0.0,
			WindSpeed:    5.0,
			Pressure:     101.3,
			Visibility:   1.0,
		},
		changeInterval:    4.0,  // Change weather every 4 game hours
		volatility:        0.5,
		seasonalInfluence: 0.7,
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	ws.targetConditions = ws.current
	return ws
}

// Update advances the weather simulation
func (ws *WeatherSystem) Update(deltaTime float64, season Season) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.timeSinceChange += deltaTime / 3600.0 // Convert to hours

	// Check if it's time to change weather
	if ws.timeSinceChange >= ws.changeInterval {
		ws.generateNewWeather(season)
		ws.timeSinceChange = 0
	}

	// Transition to target conditions
	if ws.elapsedTransition < ws.transitionTime {
		ws.elapsedTransition += deltaTime / 3600.0
		progress := ws.elapsedTransition / ws.transitionTime
		if progress > 1.0 {
			progress = 1.0
		}

		// Smooth interpolation
		smoothProgress := smoothStep(progress)

		ws.current.Temperature = lerp(ws.current.Temperature, ws.targetConditions.Temperature, smoothProgress)
		ws.current.Humidity = lerp(ws.current.Humidity, ws.targetConditions.Humidity, smoothProgress)
		ws.current.Precipitation = lerp(ws.current.Precipitation, ws.targetConditions.Precipitation, smoothProgress)
		ws.current.WindSpeed = lerp(ws.current.WindSpeed, ws.targetConditions.WindSpeed, smoothProgress)
		ws.current.Pressure = lerp(ws.current.Pressure, ws.targetConditions.Pressure, smoothProgress)
		ws.current.Visibility = lerp(ws.current.Visibility, ws.targetConditions.Visibility, smoothProgress)
	}

	// Update weather type based on current conditions
	ws.updateWeatherType()
}

// generateNewWeather creates new target weather conditions
func (ws *WeatherSystem) generateNewWeather(season Season) {
	// Base temperature on season
	baseTemp := ws.getSeasonalBaseTemperature(season)

	// Add randomness
	tempVariation := (ws.rng.Float64() - 0.5) * 20.0 * ws.volatility
	ws.targetConditions.Temperature = baseTemp + tempVariation

	// Generate other conditions
	ws.targetConditions.Humidity = ws.rng.Float64()*0.6 + 0.2 // 0.2 to 0.8
	ws.targetConditions.WindSpeed = ws.rng.Float64() * 30.0   // 0 to 30 km/h

	// Precipitation depends on season and humidity
	precipChance := ws.targetConditions.Humidity * ws.getSeasonalPrecipitationFactor(season)
	if ws.rng.Float64() < precipChance {
		ws.targetConditions.Precipitation = ws.rng.Float64()
	} else {
		ws.targetConditions.Precipitation = 0.0
	}

	// Pressure varies slightly
	ws.targetConditions.Pressure = 101.3 + (ws.rng.Float64()-0.5)*10.0

	// Visibility affected by precipitation and humidity
	ws.targetConditions.Visibility = 1.0 - (ws.targetConditions.Precipitation * 0.5) - (ws.targetConditions.Humidity * 0.2)
	if ws.targetConditions.Visibility < 0.1 {
		ws.targetConditions.Visibility = 0.1
	}

	// Set transition time (faster transitions for volatile weather)
	ws.transitionTime = 1.0 + (1.0-ws.volatility)*3.0 // 1-4 hours
	ws.elapsedTransition = 0
}

// updateWeatherType determines weather type from conditions
func (ws *WeatherSystem) updateWeatherType() {
	// Priority-based weather type determination
	if ws.current.Temperature < 0 && ws.current.Precipitation > 0.3 {
		ws.current.Type = WeatherSnowy
	} else if ws.current.Precipitation > 0.7 && ws.current.WindSpeed > 20 {
		ws.current.Type = WeatherStormy
	} else if ws.current.Precipitation > 0.3 {
		ws.current.Type = WeatherRainy
	} else if ws.current.Visibility < 0.5 {
		ws.current.Type = WeatherFoggy
	} else if ws.current.WindSpeed > 25 {
		ws.current.Type = WeatherWindy
	} else if ws.current.Temperature > 30 {
		ws.current.Type = WeatherHot
	} else if ws.current.Temperature < 5 {
		ws.current.Type = WeatherCold
	} else if ws.current.Humidity > 0.6 {
		ws.current.Type = WeatherCloudy
	} else {
		ws.current.Type = WeatherClear
	}
}

// getSeasonalBaseTemperature returns base temperature for a season
func (ws *WeatherSystem) getSeasonalBaseTemperature(season Season) float64 {
	switch season {
	case SeasonSpring:
		return 15.0
	case SeasonSummer:
		return 25.0
	case SeasonAutumn:
		return 15.0
	case SeasonWinter:
		return 5.0
	default:
		return 20.0
	}
}

// getSeasonalPrecipitationFactor returns precipitation likelihood for a season
func (ws *WeatherSystem) getSeasonalPrecipitationFactor(season Season) float64 {
	switch season {
	case SeasonSpring:
		return 0.7 // Spring showers
	case SeasonSummer:
		return 0.4
	case SeasonAutumn:
		return 0.6
	case SeasonWinter:
		return 0.5
	default:
		return 0.5
	}
}

// GetConditions returns current weather conditions (thread-safe)
func (ws *WeatherSystem) GetConditions() WeatherConditions {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.current
}

// GetWeatherType returns current weather type
func (ws *WeatherSystem) GetWeatherType() WeatherType {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.current.Type
}

// GetTemperature returns current temperature
func (ws *WeatherSystem) GetTemperature() float64 {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.current.Temperature
}

// SetVolatility adjusts how quickly weather changes
func (ws *WeatherSystem) SetVolatility(volatility float64) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if volatility < 0 {
		volatility = 0
	}
	if volatility > 1 {
		volatility = 1
	}
	ws.volatility = volatility
}

// SetChangeInterval sets how often weather changes (in game hours)
func (ws *WeatherSystem) SetChangeInterval(hours float64) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if hours < 0.5 {
		hours = 0.5
	}
	ws.changeInterval = hours
}

// ForceWeatherType immediately sets weather to a specific type
func (ws *WeatherSystem) ForceWeatherType(weatherType WeatherType) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.current.Type = weatherType
	ws.targetConditions.Type = weatherType

	// Set appropriate conditions for the weather type
	switch weatherType {
	case WeatherClear:
		ws.current.Precipitation = 0.0
		ws.current.Visibility = 1.0
	case WeatherRainy:
		ws.current.Precipitation = 0.6
		ws.current.Visibility = 0.7
	case WeatherStormy:
		ws.current.Precipitation = 0.9
		ws.current.WindSpeed = 30.0
		ws.current.Visibility = 0.4
	case WeatherSnowy:
		ws.current.Precipitation = 0.5
		ws.current.Temperature = -2.0
		ws.current.Visibility = 0.6
	case WeatherFoggy:
		ws.current.Visibility = 0.3
		ws.current.Humidity = 0.9
	}

	ws.targetConditions = ws.current
}

// GetComfortLevel returns how comfortable the weather is (0-1)
func (ws *WeatherSystem) GetComfortLevel() float64 {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	comfort := 1.0

	// Temperature comfort (optimal 15-25°C)
	if ws.current.Temperature < 15 {
		comfort -= (15 - ws.current.Temperature) * 0.02
	} else if ws.current.Temperature > 25 {
		comfort -= (ws.current.Temperature - 25) * 0.02
	}

	// Precipitation discomfort
	comfort -= ws.current.Precipitation * 0.3

	// Wind discomfort
	if ws.current.WindSpeed > 20 {
		comfort -= (ws.current.WindSpeed - 20) * 0.01
	}

	// Visibility issues
	comfort -= (1.0 - ws.current.Visibility) * 0.2

	if comfort < 0 {
		comfort = 0
	}
	if comfort > 1 {
		comfort = 1
	}

	return comfort
}

// Helper functions

func lerp(start, end, t float64) float64 {
	return start + (end-start)*t
}

func smoothStep(t float64) float64 {
	return t * t * (3 - 2*t)
}

// GetWeatherEffects returns environmental multipliers for pet needs
type WeatherEffects struct {
	EnergyDrainMultiplier float64 // How much faster energy drains
	HydrationMultiplier   float64 // How much faster hydration drains
	TemperatureStress     float64 // Stress from extreme temperatures
	ActivityRestriction   float64 // How much outdoor activity is restricted (0-1)
}

// CalculateWeatherEffects computes environmental effects on pets
func (ws *WeatherSystem) CalculateWeatherEffects() WeatherEffects {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	effects := WeatherEffects{
		EnergyDrainMultiplier: 1.0,
		HydrationMultiplier:   1.0,
		TemperatureStress:     0.0,
		ActivityRestriction:   0.0,
	}

	// Temperature effects
	if ws.current.Temperature > 30 {
		effects.HydrationMultiplier += (ws.current.Temperature - 30) * 0.05
		effects.TemperatureStress = (ws.current.Temperature - 30) * 0.02
		effects.ActivityRestriction += 0.3
	} else if ws.current.Temperature < 5 {
		effects.EnergyDrainMultiplier += (5 - ws.current.Temperature) * 0.03
		effects.TemperatureStress = (5 - ws.current.Temperature) * 0.02
		effects.ActivityRestriction += 0.2
	}

	// Weather type effects
	switch ws.current.Type {
	case WeatherStormy:
		effects.ActivityRestriction += 0.7
		effects.TemperatureStress += 0.2
	case WeatherRainy:
		effects.ActivityRestriction += 0.4
	case WeatherSnowy:
		effects.ActivityRestriction += 0.5
		effects.EnergyDrainMultiplier += 0.2
	case WeatherFoggy:
		effects.ActivityRestriction += 0.2
	}

	// Clamp values
	if effects.EnergyDrainMultiplier > 2.0 {
		effects.EnergyDrainMultiplier = 2.0
	}
	if effects.HydrationMultiplier > 2.0 {
		effects.HydrationMultiplier = 2.0
	}
	if effects.TemperatureStress > 1.0 {
		effects.TemperatureStress = 1.0
	}
	if effects.ActivityRestriction > 1.0 {
		effects.ActivityRestriction = 1.0
	}

	return effects
}

// IsOutdoorActivitySafe returns whether it's safe for outdoor activities
func (ws *WeatherSystem) IsOutdoorActivitySafe() bool {
	effects := ws.CalculateWeatherEffects()
	return effects.ActivityRestriction < 0.7
}

// GetWeatherDescription returns a text description of current weather
func (ws *WeatherSystem) GetWeatherDescription() string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	temp := ws.current.Temperature
	tempDesc := ""
	if temp > 30 {
		tempDesc = "very hot"
	} else if temp > 25 {
		tempDesc = "warm"
	} else if temp > 15 {
		tempDesc = "mild"
	} else if temp > 5 {
		tempDesc = "cool"
	} else if temp > 0 {
		tempDesc = "cold"
	} else {
		tempDesc = "freezing"
	}

	return ws.current.Type.String() + " and " + tempDesc
}

// GetStats returns weather system statistics
func (ws *WeatherSystem) GetStats() map[string]interface{} {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	return map[string]interface{}{
		"weather_type":    ws.current.Type.String(),
		"temperature":     math.Round(ws.current.Temperature*10) / 10,
		"humidity":        math.Round(ws.current.Humidity*100) / 100,
		"precipitation":   math.Round(ws.current.Precipitation*100) / 100,
		"wind_speed":      math.Round(ws.current.WindSpeed*10) / 10,
		"visibility":      math.Round(ws.current.Visibility*100) / 100,
		"comfort_level":   math.Round(ws.GetComfortLevel()*100) / 100,
		"change_interval": ws.changeInterval,
		"volatility":      ws.volatility,
	}
}
