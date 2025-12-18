package environment

import (
	"math"
	"math/rand"
	"time"
)

// WeatherType represents different weather conditions
type WeatherType int

const (
	WeatherClear WeatherType = iota
	WeatherPartlyCloudy
	WeatherCloudy
	WeatherRainy
	WeatherStormy
	WeatherSnowy
	WeatherFoggy
	WeatherWindy
	WeatherHot
	WeatherCold
)

// String returns the string representation of WeatherType
func (w WeatherType) String() string {
	return [...]string{
		"Clear", "Partly Cloudy", "Cloudy", "Rainy",
		"Stormy", "Snowy", "Foggy", "Windy", "Hot", "Cold",
	}[w]
}

// WeatherConditions represents current weather state
type WeatherConditions struct {
	Type        WeatherType `json:"type"`
	Temperature float64     `json:"temperature"`  // In Celsius
	Humidity    float64     `json:"humidity"`     // 0.0 to 1.0
	WindSpeed   float64     `json:"wind_speed"`   // In km/h
	Visibility  float64     `json:"visibility"`   // 0.0 to 1.0
	UVIndex     float64     `json:"uv_index"`     // 0 to 11+
	Comfort     float64     `json:"comfort"`      // Overall comfort level 0.0 to 1.0
}

// WeatherSystem manages weather simulation
type WeatherSystem struct {
	Current          WeatherConditions `json:"current"`
	TransitionTarget *WeatherConditions `json:"transition_target,omitempty"`
	TransitionRate   float64           `json:"transition_rate"`
	LastUpdate       time.Time         `json:"last_update"`

	// Configuration
	BaseTemperature  float64 `json:"base_temperature"`
	TemperatureRange float64 `json:"temperature_range"`
	ChangeFrequency  float64 `json:"change_frequency"` // Average hours between weather changes

	// History
	History []WeatherConditions `json:"history"`
}

// NewWeatherSystem creates a new weather system with default settings
func NewWeatherSystem() *WeatherSystem {
	ws := &WeatherSystem{
		TransitionRate:   0.1,
		LastUpdate:       time.Now(),
		BaseTemperature:  20.0,
		TemperatureRange: 15.0,
		ChangeFrequency:  6.0,
		History:          make([]WeatherConditions, 0),
	}

	ws.Current = ws.generateWeather(WeatherClear)
	return ws
}

// NewWeatherSystemWithConfig creates a weather system with custom settings
func NewWeatherSystemWithConfig(baseTemp, tempRange, changeFreq float64) *WeatherSystem {
	ws := NewWeatherSystem()
	ws.BaseTemperature = baseTemp
	ws.TemperatureRange = tempRange
	ws.ChangeFrequency = changeFreq
	ws.Current = ws.generateWeather(WeatherClear)
	return ws
}

// Update advances the weather simulation
func (ws *WeatherSystem) Update(deltaTime float64, season Season, timeOfDay float64) {
	// Check if we should start a weather transition
	hoursSinceUpdate := time.Since(ws.LastUpdate).Hours()
	if ws.TransitionTarget == nil && hoursSinceUpdate >= ws.ChangeFrequency {
		if rand.Float64() < 0.3 { // 30% chance to change weather
			ws.startWeatherTransition(season)
		}
		ws.LastUpdate = time.Now()
	}

	// Process ongoing transition
	if ws.TransitionTarget != nil {
		ws.processTransition(deltaTime)
	}

	// Apply time-of-day effects
	ws.applyTimeOfDayEffects(timeOfDay)

	// Apply seasonal effects
	ws.applySeasonalEffects(season)

	// Calculate comfort
	ws.Current.Comfort = ws.calculateComfort()
}

// generateWeather creates weather conditions for a given type
func (ws *WeatherSystem) generateWeather(weatherType WeatherType) WeatherConditions {
	conditions := WeatherConditions{
		Type: weatherType,
	}

	switch weatherType {
	case WeatherClear:
		conditions.Temperature = ws.BaseTemperature + (rand.Float64()-0.5)*ws.TemperatureRange*0.5
		conditions.Humidity = 0.3 + rand.Float64()*0.2
		conditions.WindSpeed = rand.Float64() * 10
		conditions.Visibility = 0.9 + rand.Float64()*0.1
		conditions.UVIndex = 5 + rand.Float64()*4

	case WeatherPartlyCloudy:
		conditions.Temperature = ws.BaseTemperature + (rand.Float64()-0.5)*ws.TemperatureRange*0.4
		conditions.Humidity = 0.4 + rand.Float64()*0.2
		conditions.WindSpeed = 5 + rand.Float64()*15
		conditions.Visibility = 0.8 + rand.Float64()*0.15
		conditions.UVIndex = 3 + rand.Float64()*4

	case WeatherCloudy:
		conditions.Temperature = ws.BaseTemperature + (rand.Float64()-0.5)*ws.TemperatureRange*0.3
		conditions.Humidity = 0.5 + rand.Float64()*0.3
		conditions.WindSpeed = 5 + rand.Float64()*20
		conditions.Visibility = 0.6 + rand.Float64()*0.2
		conditions.UVIndex = 1 + rand.Float64()*3

	case WeatherRainy:
		conditions.Temperature = ws.BaseTemperature - 5 + (rand.Float64()-0.5)*ws.TemperatureRange*0.3
		conditions.Humidity = 0.7 + rand.Float64()*0.3
		conditions.WindSpeed = 10 + rand.Float64()*25
		conditions.Visibility = 0.3 + rand.Float64()*0.3
		conditions.UVIndex = rand.Float64() * 2

	case WeatherStormy:
		conditions.Temperature = ws.BaseTemperature - 8 + (rand.Float64()-0.5)*ws.TemperatureRange*0.2
		conditions.Humidity = 0.8 + rand.Float64()*0.2
		conditions.WindSpeed = 30 + rand.Float64()*50
		conditions.Visibility = 0.1 + rand.Float64()*0.2
		conditions.UVIndex = rand.Float64()

	case WeatherSnowy:
		conditions.Temperature = -5 + (rand.Float64()-0.5)*10
		conditions.Humidity = 0.6 + rand.Float64()*0.3
		conditions.WindSpeed = 5 + rand.Float64()*20
		conditions.Visibility = 0.2 + rand.Float64()*0.4
		conditions.UVIndex = 1 + rand.Float64()*2

	case WeatherFoggy:
		conditions.Temperature = ws.BaseTemperature - 3 + (rand.Float64()-0.5)*5
		conditions.Humidity = 0.9 + rand.Float64()*0.1
		conditions.WindSpeed = rand.Float64() * 5
		conditions.Visibility = 0.05 + rand.Float64()*0.15
		conditions.UVIndex = rand.Float64() * 2

	case WeatherWindy:
		conditions.Temperature = ws.BaseTemperature + (rand.Float64()-0.5)*ws.TemperatureRange*0.4
		conditions.Humidity = 0.3 + rand.Float64()*0.3
		conditions.WindSpeed = 40 + rand.Float64()*40
		conditions.Visibility = 0.7 + rand.Float64()*0.2
		conditions.UVIndex = 3 + rand.Float64()*5

	case WeatherHot:
		conditions.Temperature = ws.BaseTemperature + 15 + rand.Float64()*10
		conditions.Humidity = 0.2 + rand.Float64()*0.4
		conditions.WindSpeed = rand.Float64() * 15
		conditions.Visibility = 0.85 + rand.Float64()*0.1
		conditions.UVIndex = 8 + rand.Float64()*4

	case WeatherCold:
		conditions.Temperature = ws.BaseTemperature - 20 - rand.Float64()*10
		conditions.Humidity = 0.4 + rand.Float64()*0.3
		conditions.WindSpeed = 10 + rand.Float64()*30
		conditions.Visibility = 0.7 + rand.Float64()*0.2
		conditions.UVIndex = 1 + rand.Float64()*2
	}

	return conditions
}

// startWeatherTransition begins transitioning to new weather
func (ws *WeatherSystem) startWeatherTransition(season Season) {
	// Determine likely next weather based on current and season
	nextType := ws.selectNextWeather(season)
	newWeather := ws.generateWeather(nextType)
	ws.TransitionTarget = &newWeather
}

// selectNextWeather chooses next weather type based on current conditions and season
func (ws *WeatherSystem) selectNextWeather(season Season) WeatherType {
	// Weather transition probabilities based on season
	var probabilities map[WeatherType]float64

	switch season {
	case SeasonSpring:
		probabilities = map[WeatherType]float64{
			WeatherClear:        0.2,
			WeatherPartlyCloudy: 0.25,
			WeatherCloudy:       0.15,
			WeatherRainy:        0.25,
			WeatherWindy:        0.1,
			WeatherFoggy:        0.05,
		}
	case SeasonSummer:
		probabilities = map[WeatherType]float64{
			WeatherClear:        0.35,
			WeatherPartlyCloudy: 0.25,
			WeatherHot:          0.2,
			WeatherStormy:       0.1,
			WeatherWindy:        0.1,
		}
	case SeasonFall:
		probabilities = map[WeatherType]float64{
			WeatherClear:        0.15,
			WeatherPartlyCloudy: 0.2,
			WeatherCloudy:       0.25,
			WeatherRainy:        0.2,
			WeatherWindy:        0.15,
			WeatherFoggy:        0.05,
		}
	case SeasonWinter:
		probabilities = map[WeatherType]float64{
			WeatherClear:        0.15,
			WeatherCloudy:       0.2,
			WeatherSnowy:        0.3,
			WeatherCold:         0.2,
			WeatherWindy:        0.1,
			WeatherFoggy:        0.05,
		}
	}

	// Select based on probabilities
	roll := rand.Float64()
	cumulative := 0.0

	for weatherType, prob := range probabilities {
		cumulative += prob
		if roll < cumulative {
			return weatherType
		}
	}

	return WeatherClear
}

// processTransition gradually transitions to target weather
func (ws *WeatherSystem) processTransition(deltaTime float64) {
	if ws.TransitionTarget == nil {
		return
	}

	rate := ws.TransitionRate * deltaTime

	// Interpolate values
	ws.Current.Temperature = lerp(ws.Current.Temperature, ws.TransitionTarget.Temperature, rate)
	ws.Current.Humidity = lerp(ws.Current.Humidity, ws.TransitionTarget.Humidity, rate)
	ws.Current.WindSpeed = lerp(ws.Current.WindSpeed, ws.TransitionTarget.WindSpeed, rate)
	ws.Current.Visibility = lerp(ws.Current.Visibility, ws.TransitionTarget.Visibility, rate)
	ws.Current.UVIndex = lerp(ws.Current.UVIndex, ws.TransitionTarget.UVIndex, rate)

	// Check if transition is complete
	tempDiff := math.Abs(ws.Current.Temperature - ws.TransitionTarget.Temperature)
	humidDiff := math.Abs(ws.Current.Humidity - ws.TransitionTarget.Humidity)

	if tempDiff < 0.5 && humidDiff < 0.05 {
		// Record in history before changing type
		ws.History = append(ws.History, ws.Current)
		if len(ws.History) > 24 { // Keep last 24 weather states
			ws.History = ws.History[1:]
		}

		ws.Current.Type = ws.TransitionTarget.Type
		ws.TransitionTarget = nil
	}
}

// applyTimeOfDayEffects modifies weather based on time of day
func (ws *WeatherSystem) applyTimeOfDayEffects(timeOfDay float64) {
	// timeOfDay is 0-24 hours

	// Temperature variation (cooler at night)
	nightFactor := 1.0
	if timeOfDay < 6 || timeOfDay > 20 {
		nightFactor = 0.85 // 15% cooler at night
	} else if timeOfDay > 12 && timeOfDay < 16 {
		nightFactor = 1.1 // 10% warmer in early afternoon
	}

	ws.Current.Temperature *= nightFactor

	// UV varies with sun position
	if timeOfDay < 6 || timeOfDay > 19 {
		ws.Current.UVIndex = 0
	} else if timeOfDay > 10 && timeOfDay < 14 {
		// Peak UV at midday
		ws.Current.UVIndex *= 1.2
	}
}

// applySeasonalEffects modifies weather based on season
func (ws *WeatherSystem) applySeasonalEffects(season Season) {
	switch season {
	case SeasonWinter:
		ws.Current.Temperature -= 10
	case SeasonSummer:
		ws.Current.Temperature += 10
	case SeasonSpring, SeasonFall:
		// No adjustment
	}
}

// calculateComfort determines overall comfort level
func (ws *WeatherSystem) calculateComfort() float64 {
	comfort := 1.0

	// Temperature comfort (ideal around 20-25C)
	tempComfort := 1.0 - math.Abs(ws.Current.Temperature-22.5)/30.0
	comfort *= clamp(tempComfort, 0.0, 1.0)

	// Humidity comfort (ideal around 40-60%)
	humidComfort := 1.0 - math.Abs(ws.Current.Humidity-0.5)*2
	comfort *= clamp(humidComfort, 0.5, 1.0)

	// Wind comfort (high wind is uncomfortable)
	windComfort := 1.0 - ws.Current.WindSpeed/100.0
	comfort *= clamp(windComfort, 0.3, 1.0)

	// Visibility comfort
	comfort *= 0.5 + ws.Current.Visibility*0.5

	// Weather type penalties
	switch ws.Current.Type {
	case WeatherStormy:
		comfort *= 0.3
	case WeatherRainy:
		comfort *= 0.6
	case WeatherSnowy:
		comfort *= 0.5
	case WeatherFoggy:
		comfort *= 0.7
	case WeatherHot:
		comfort *= 0.5
	case WeatherCold:
		comfort *= 0.4
	}

	return clamp(comfort, 0.0, 1.0)
}

// GetWeatherEffects returns how weather affects various pet needs
func (ws *WeatherSystem) GetWeatherEffects() map[string]float64 {
	effects := make(map[string]float64)

	// Energy drain (harsh weather drains more energy)
	effects["energy_drain"] = 1.0
	if ws.Current.Type == WeatherStormy || ws.Current.Type == WeatherCold {
		effects["energy_drain"] = 1.5
	} else if ws.Current.Type == WeatherHot {
		effects["energy_drain"] = 1.3
	}

	// Thirst increase (hot weather increases thirst)
	effects["thirst_rate"] = 1.0
	if ws.Current.Type == WeatherHot || ws.Current.Temperature > 30 {
		effects["thirst_rate"] = 1.5
	}

	// Mood modifier
	effects["mood_modifier"] = ws.Current.Comfort

	// Outdoor activity suitability
	effects["outdoor_suitability"] = ws.Current.Comfort
	if ws.Current.Type == WeatherStormy || ws.Current.Type == WeatherRainy {
		effects["outdoor_suitability"] = 0.2
	}

	// Sleep quality modifier
	effects["sleep_quality"] = 1.0
	if ws.Current.Type == WeatherStormy {
		effects["sleep_quality"] = 0.6
	} else if ws.Current.Type == WeatherClear && ws.Current.Comfort > 0.7 {
		effects["sleep_quality"] = 1.1
	}

	return effects
}

// IsOutdoorSafe returns whether it's safe for outdoor activities
func (ws *WeatherSystem) IsOutdoorSafe() bool {
	unsafeTypes := []WeatherType{WeatherStormy, WeatherCold}
	for _, t := range unsafeTypes {
		if ws.Current.Type == t {
			return false
		}
	}

	return ws.Current.Temperature > -10 && ws.Current.Temperature < 40 &&
		ws.Current.WindSpeed < 60
}

// GetDescription returns a human-readable weather description
func (ws *WeatherSystem) GetDescription() string {
	desc := ws.Current.Type.String()

	if ws.Current.Temperature < 0 {
		desc += ", freezing"
	} else if ws.Current.Temperature < 10 {
		desc += ", cold"
	} else if ws.Current.Temperature > 35 {
		desc += ", very hot"
	} else if ws.Current.Temperature > 28 {
		desc += ", warm"
	}

	if ws.Current.WindSpeed > 50 {
		desc += ", very windy"
	} else if ws.Current.WindSpeed > 30 {
		desc += ", breezy"
	}

	return desc
}

// SetWeather forces a specific weather type (for testing or events)
func (ws *WeatherSystem) SetWeather(weatherType WeatherType) {
	ws.Current = ws.generateWeather(weatherType)
	ws.TransitionTarget = nil
	ws.Current.Comfort = ws.calculateComfort()
}

// helper function
func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}
