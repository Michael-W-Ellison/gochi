package environment

import (
	"fmt"
	"math"
	"sync"
)

// EnvironmentConfig holds configuration for the environment system
type EnvironmentConfig struct {
	// Season settings
	StartSeason        Season
	SeasonLengthHours  float64
	SeasonalIntensity  float64
	EnableTransitions  bool

	// Weather settings
	WeatherVolatility     float64
	WeatherChangeInterval float64

	// Location settings
	StartingBiome BiomeType
}

// DefaultEnvironmentConfig returns default configuration
func DefaultEnvironmentConfig() *EnvironmentConfig {
	return &EnvironmentConfig{
		StartSeason:           SeasonSpring,
		SeasonLengthHours:     168.0, // 1 week = 1 season in game time
		SeasonalIntensity:     0.8,
		EnableTransitions:     true,
		WeatherVolatility:     0.5,
		WeatherChangeInterval: 4.0,
		StartingBiome:         BiomeGrassland,
	}
}

// EnvironmentManager integrates weather, seasons, and biomes
type EnvironmentManager struct {
	mu sync.RWMutex

	config *EnvironmentConfig

	// Core systems
	weather     *WeatherSystem
	seasons     *SeasonalCycle
	biomes      *BiomeManager

	// State
	totalGameTime float64
	isInitialized bool
}

// NewEnvironmentManager creates a new environment manager
func NewEnvironmentManager(config *EnvironmentConfig) *EnvironmentManager {
	if config == nil {
		config = DefaultEnvironmentConfig()
	}

	em := &EnvironmentManager{
		config:        config,
		weather:       NewWeatherSystem(),
		seasons:       NewSeasonalCycle(config.StartSeason, config.SeasonLengthHours),
		biomes:        NewBiomeManager(),
		totalGameTime: 0.0,
		isInitialized: true,
	}

	// Configure subsystems
	em.weather.SetVolatility(config.WeatherVolatility)
	em.weather.SetChangeInterval(config.WeatherChangeInterval)
	em.seasons.SetSeasonalIntensity(config.SeasonalIntensity)

	return em
}

// Update advances all environmental systems
func (em *EnvironmentManager) Update(deltaTime float64) {
	em.mu.Lock()
	em.totalGameTime += deltaTime
	em.mu.Unlock()

	// Update seasons
	em.seasons.Update(deltaTime)

	// Update weather (influenced by current season)
	currentSeason := em.seasons.GetCurrentSeason()
	em.weather.Update(deltaTime, currentSeason)

	// Update biomes (resource regeneration)
	em.biomes.UpdateAll(deltaTime)
}

// GetWeatherConditions returns current weather
func (em *EnvironmentManager) GetWeatherConditions() WeatherConditions {
	return em.weather.GetConditions()
}

// GetCurrentSeason returns current season
func (em *EnvironmentManager) GetCurrentSeason() Season {
	return em.seasons.GetCurrentSeason()
}

// GetCurrentLocation returns current location/biome
func (em *EnvironmentManager) GetCurrentLocation() *Location {
	return em.biomes.GetCurrentLocation()
}

// TravelToLocation moves to a different location
func (em *EnvironmentManager) TravelToLocation(locationName string) bool {
	em.mu.RLock()
	gameTime := em.totalGameTime
	em.mu.RUnlock()

	return em.biomes.TravelTo(locationName, gameTime)
}

// AddLocation adds a new location to explore
func (em *EnvironmentManager) AddLocation(location *Location) {
	em.biomes.AddLocation(location)
}

// GetDiscoveredLocations returns all discovered locations
func (em *EnvironmentManager) GetDiscoveredLocations() []*Location {
	return em.biomes.GetDiscoveredLocations()
}

// GetNearbyLocations returns locations within range
func (em *EnvironmentManager) GetNearbyLocations(maxDistance float64) []*Location {
	return em.biomes.GetNearbyLocations(maxDistance)
}

// CalculateEnvironmentalEffects computes overall environmental impact on pets
type EnvironmentalEffects struct {
	// Weather effects
	WeatherEffects WeatherEffects

	// Seasonal effects
	SeasonalEffects SeasonalEffects

	// Biome effects
	FoodAvailability  float64
	WaterAvailability float64
	ShelterQuality    float64
	DangerLevel       float64
	ExplorationBonus  float64
	SocialOpportunity float64

	// Combined effects
	OverallComfort     float64 // 0-1
	MoodModifier       float64 // -1 to 1
	ActivityMultiplier float64 // 0-2
}

// CalculateEnvironmentalEffects computes all environmental effects
func (em *EnvironmentManager) CalculateEnvironmentalEffects() EnvironmentalEffects {
	effects := EnvironmentalEffects{}

	// Get weather effects
	effects.WeatherEffects = em.weather.CalculateWeatherEffects()

	// Get seasonal effects
	effects.SeasonalEffects = em.seasons.GetSeasonalEffects()

	// Get biome effects
	currentLocation := em.biomes.GetCurrentLocation()
	if currentLocation != nil {
		effects.FoodAvailability = currentLocation.GetEffectiveFoodAvailability() * effects.SeasonalEffects.FoodAvailability
		effects.WaterAvailability = currentLocation.Characteristics.WaterAvailability * effects.SeasonalEffects.WaterAvailability
		effects.ShelterQuality = currentLocation.Characteristics.ShelterQuality
		effects.DangerLevel = currentLocation.Characteristics.DangerLevel
		effects.ExplorationBonus = currentLocation.Characteristics.ExplorationValue
		effects.SocialOpportunity = currentLocation.Characteristics.SocialDensity
	} else {
		// Default values if no location
		effects.FoodAvailability = 0.6
		effects.WaterAvailability = 0.6
		effects.ShelterQuality = 0.5
		effects.DangerLevel = 0.3
		effects.ExplorationBonus = 0.5
		effects.SocialOpportunity = 0.5
	}

	// Calculate overall comfort
	weatherComfort := em.weather.GetComfortLevel()
	locationComfort := 0.5
	if currentLocation != nil {
		locationComfort = currentLocation.GetComfortLevel()
	}
	effects.OverallComfort = (weatherComfort + locationComfort) / 2.0

	// Calculate mood modifier
	effects.MoodModifier = (effects.OverallComfort - 0.5) * 2.0 // Scale to -1 to 1

	// Reduce mood if food/water scarce
	if effects.FoodAvailability < 0.3 {
		effects.MoodModifier -= 0.3
	}
	if effects.WaterAvailability < 0.3 {
		effects.MoodModifier -= 0.2
	}

	// Increase mood in pleasant weather
	if weatherComfort > 0.8 {
		effects.MoodModifier += 0.1
	}

	// Clamp mood modifier
	if effects.MoodModifier < -1.0 {
		effects.MoodModifier = -1.0
	}
	if effects.MoodModifier > 1.0 {
		effects.MoodModifier = 1.0
	}

	// Calculate activity multiplier
	effects.ActivityMultiplier = effects.SeasonalEffects.ActivityLevel
	effects.ActivityMultiplier *= (1.0 - effects.WeatherEffects.ActivityRestriction)

	return effects
}

// GetEnvironmentDescription returns a text description of current environment
func (em *EnvironmentManager) GetEnvironmentDescription() string {
	season := em.seasons.GetSeasonDescription()
	weather := em.weather.GetWeatherDescription()

	location := "somewhere"
	currentLoc := em.biomes.GetCurrentLocation()
	if currentLoc != nil {
		location = currentLoc.Name
	}

	return fmt.Sprintf("It's %s and the weather is %s. You are in %s.", season, weather, location)
}

// IsOutdoorActivityRecommended returns whether outdoor activities are safe and pleasant
func (em *EnvironmentManager) IsOutdoorActivityRecommended() bool {
	effects := em.CalculateEnvironmentalEffects()
	return effects.WeatherEffects.ActivityRestriction < 0.5 && effects.OverallComfort > 0.4
}

// GetTimeOfDay returns current time of day description
func (em *EnvironmentManager) GetTimeOfDay() string {
	// This would integrate with TimeManager in a real implementation
	// For now, return based on season's daylight hours
	daylightHours := em.seasons.GetDaylightHours()

	if daylightHours > 13 {
		return "afternoon"
	} else if daylightHours > 11 {
		return "midday"
	} else {
		return "morning"
	}
}

// SetSeason immediately changes the season (for testing/debugging)
func (em *EnvironmentManager) SetSeason(season Season) {
	em.seasons.SetSeason(season)
}

// SetWeather immediately changes the weather (for testing/debugging)
func (em *EnvironmentManager) SetWeather(weatherType WeatherType) {
	em.weather.ForceWeatherType(weatherType)
}

// GetSeasonalBreedingBonus returns breeding compatibility bonus from season
func (em *EnvironmentManager) GetSeasonalBreedingBonus() float64 {
	effects := em.seasons.GetSeasonalEffects()
	return effects.BreedingSeasonBonus
}

// CalculateSeasonalMoodForPet calculates seasonal mood effect for a pet's preferred season
func (em *EnvironmentManager) CalculateSeasonalMoodForPet(preferredSeason Season) float64 {
	return em.seasons.CalculateSeasonalMoodEffect(preferredSeason)
}

// GetResourceAvailability returns how plentiful resources are (0-1)
func (em *EnvironmentManager) GetResourceAvailability() float64 {
	effects := em.CalculateEnvironmentalEffects()
	return (effects.FoodAvailability + effects.WaterAvailability) / 2.0
}

// GetSeasonProgress returns progress through current season (0-1)
func (em *EnvironmentManager) GetSeasonProgress() float64 {
	return em.seasons.GetSeasonProgress()
}

// DepleteLocationResources reduces resources at current location
func (em *EnvironmentManager) DepleteLocationResources(amount float64) {
	currentLocation := em.biomes.GetCurrentLocation()
	if currentLocation != nil {
		currentLocation.DepleteResources(amount)
	}
}

// GetStats returns comprehensive environmental statistics
func (em *EnvironmentManager) GetStats() map[string]interface{} {
	em.mu.RLock()
	totalTime := em.totalGameTime
	em.mu.RUnlock()

	stats := make(map[string]interface{})

	// Weather stats
	weatherStats := em.weather.GetStats()
	for k, v := range weatherStats {
		stats["weather_"+k] = v
	}

	// Season stats
	seasonStats := em.seasons.GetStats()
	for k, v := range seasonStats {
		stats["season_"+k] = v
	}

	// Biome stats
	biomeStats := em.biomes.GetStats()
	for k, v := range biomeStats {
		stats["biome_"+k] = v
	}

	// Environmental effects
	effects := em.CalculateEnvironmentalEffects()
	stats["overall_comfort"] = math.Round(effects.OverallComfort*100) / 100
	stats["mood_modifier"] = math.Round(effects.MoodModifier*100) / 100
	stats["food_availability"] = math.Round(effects.FoodAvailability*100) / 100
	stats["water_availability"] = math.Round(effects.WaterAvailability*100) / 100
	stats["activity_multiplier"] = math.Round(effects.ActivityMultiplier*100) / 100

	stats["total_game_time_hours"] = math.Round(totalTime/3600.0*10) / 10

	return stats
}

// CreateStandardWorld populates the biome manager with standard locations
func (em *EnvironmentManager) CreateStandardWorld() {
	// Home/starting area
	home := NewLocation("Home Meadow", BiomeGrassland, Coordinates{X: 0, Y: 0})
	home.Discovered = true
	em.biomes.AddLocation(home)

	// Nearby locations
	forest := NewLocation("Whispering Forest", BiomeForest, Coordinates{X: 50, Y: 30})
	em.biomes.AddLocation(forest)

	beach := NewLocation("Sunny Beach", BiomeOcean, Coordinates{X: 80, Y: 0})
	em.biomes.AddLocation(beach)

	hills := NewLocation("Rolling Hills", BiomeGrassland, Coordinates{X: 30, Y: 50})
	em.biomes.AddLocation(hills)

	mountain := NewLocation("Misty Peak", BiomeMountain, Coordinates{X: 100, Y: 100})
	em.biomes.AddLocation(mountain)

	// Distant locations
	desert := NewLocation("Golden Sands", BiomeDesert, Coordinates{X: 200, Y: 50})
	em.biomes.AddLocation(desert)

	jungle := NewLocation("Emerald Canopy", BiomeJungle, Coordinates{X: -80, Y: 120})
	em.biomes.AddLocation(jungle)

	swamp := NewLocation("Misty Marshes", BiomeSwamp, Coordinates{X: -50, Y: -40})
	em.biomes.AddLocation(swamp)

	tundra := NewLocation("Frozen Wastes", BiomeTundra, Coordinates{X: 50, Y: 250})
	em.biomes.AddLocation(tundra)

	cave := NewLocation("Crystal Caverns", BiomeCave, Coordinates{X: 120, Y: 90})
	em.biomes.AddLocation(cave)

	city := NewLocation("Bustling Town", BiomeUrban, Coordinates{X: -100, Y: 0})
	em.biomes.AddLocation(city)

	// Set home as current location
	em.biomes.TravelTo("Home Meadow", 0)
}

// GetConfiguration returns current environment configuration
func (em *EnvironmentManager) GetConfiguration() *EnvironmentConfig {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.config
}

// IsInitialized returns whether the environment system is ready
func (em *EnvironmentManager) IsInitialized() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.isInitialized
}

// Reset resets the environment to initial state
func (em *EnvironmentManager) Reset() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.totalGameTime = 0.0
	em.weather = NewWeatherSystem()
	em.seasons = NewSeasonalCycle(em.config.StartSeason, em.config.SeasonLengthHours)
	em.biomes = NewBiomeManager()

	// Reconfigure
	em.weather.SetVolatility(em.config.WeatherVolatility)
	em.weather.SetChangeInterval(em.config.WeatherChangeInterval)
	em.seasons.SetSeasonalIntensity(em.config.SeasonalIntensity)
}
