package environment

import (
	"math"
)

// LocationType represents different types of locations/biomes
type LocationType int

const (
	LocationHome LocationType = iota
	LocationPark
	LocationBeach
	LocationForest
	LocationMountain
	LocationCity
	LocationDesert
	LocationTundra
	LocationTropical
	LocationFarm
)

// String returns the string representation of LocationType
func (l LocationType) String() string {
	return [...]string{
		"Home", "Park", "Beach", "Forest", "Mountain",
		"City", "Desert", "Tundra", "Tropical", "Farm",
	}[l]
}

// LocationFeatures describes what a location offers
type LocationFeatures struct {
	HasWater         bool    `json:"has_water"`
	HasShelter       bool    `json:"has_shelter"`
	HasFood          bool    `json:"has_food"`
	HasToys          bool    `json:"has_toys"`
	SocialOpportunity float64 `json:"social_opportunity"` // 0.0 to 1.0
	ExplorationValue  float64 `json:"exploration_value"`  // 0.0 to 1.0
	DangerLevel       float64 `json:"danger_level"`       // 0.0 to 1.0
	Comfort           float64 `json:"comfort"`            // 0.0 to 1.0
}

// LocationEffects describes how a location affects the pet
type LocationEffects struct {
	EnergyDrain      float64 `json:"energy_drain"`      // Multiplier
	HappinessBonus   float64 `json:"happiness_bonus"`   // Additive per update
	StressModifier   float64 `json:"stress_modifier"`   // Multiplier
	ExplorationBonus float64 `json:"exploration_bonus"` // Curiosity satisfaction
	SocialBonus      float64 `json:"social_bonus"`      // Social need satisfaction
	HealthRisk       float64 `json:"health_risk"`       // Chance of health issues
}

// Location represents a place where the pet can be
type Location struct {
	Type         LocationType    `json:"type"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Features     LocationFeatures `json:"features"`
	Effects      LocationEffects  `json:"effects"`
	Climate      ClimateType     `json:"climate"`
	Unlocked     bool            `json:"unlocked"`
	VisitCount   int             `json:"visit_count"`
	TimeSpent    float64         `json:"time_spent"` // Total hours spent
}

// ClimateType represents the climate of a location
type ClimateType int

const (
	ClimateTemperate ClimateType = iota
	ClimateTropical
	ClimateArid
	ClimateCold
	ClimateCoastal
	ClimateMountainous
)

// String returns the string representation of ClimateType
func (c ClimateType) String() string {
	return [...]string{
		"Temperate", "Tropical", "Arid", "Cold", "Coastal", "Mountainous",
	}[c]
}

// LocationManager manages available locations and current location
type LocationManager struct {
	CurrentLocation *Location            `json:"current_location"`
	Locations       map[string]*Location `json:"locations"`
	TravelHistory   []TravelRecord       `json:"travel_history"`
}

// TravelRecord tracks visits to locations
type TravelRecord struct {
	LocationName string  `json:"location_name"`
	ArrivalTime  float64 `json:"arrival_time"`  // Game time
	Duration     float64 `json:"duration"`      // Hours spent
}

// NewLocationManager creates a new location manager with default locations
func NewLocationManager() *LocationManager {
	lm := &LocationManager{
		Locations:     make(map[string]*Location),
		TravelHistory: make([]TravelRecord, 0),
	}

	// Initialize default locations
	lm.initializeLocations()

	// Start at home
	lm.CurrentLocation = lm.Locations["home"]

	return lm
}

// initializeLocations creates all available locations
func (lm *LocationManager) initializeLocations() {
	// Home - always available, safest
	lm.Locations["home"] = &Location{
		Type:        LocationHome,
		Name:        "Home",
		Description: "A cozy and safe home environment",
		Features: LocationFeatures{
			HasWater:          true,
			HasShelter:        true,
			HasFood:           true,
			HasToys:           true,
			SocialOpportunity: 0.3,
			ExplorationValue:  0.2,
			DangerLevel:       0.0,
			Comfort:           1.0,
		},
		Effects: LocationEffects{
			EnergyDrain:      0.8,
			HappinessBonus:   0.05,
			StressModifier:   0.7,
			ExplorationBonus: 0.1,
			SocialBonus:      0.2,
			HealthRisk:       0.0,
		},
		Climate:  ClimateTemperate,
		Unlocked: true,
	}

	// Park - good for exercise and socialization
	lm.Locations["park"] = &Location{
		Type:        LocationPark,
		Name:        "Park",
		Description: "A spacious park with grass, trees, and other pets",
		Features: LocationFeatures{
			HasWater:          true,
			HasShelter:        false,
			HasFood:           false,
			HasToys:           true,
			SocialOpportunity: 0.9,
			ExplorationValue:  0.7,
			DangerLevel:       0.1,
			Comfort:           0.7,
		},
		Effects: LocationEffects{
			EnergyDrain:      1.2,
			HappinessBonus:   0.15,
			StressModifier:   0.8,
			ExplorationBonus: 0.4,
			SocialBonus:      0.6,
			HealthRisk:       0.05,
		},
		Climate:  ClimateTemperate,
		Unlocked: true,
	}

	// Beach - fun but weather dependent
	lm.Locations["beach"] = &Location{
		Type:        LocationBeach,
		Name:        "Beach",
		Description: "A sandy beach with waves and seashells",
		Features: LocationFeatures{
			HasWater:          true,
			HasShelter:        false,
			HasFood:           false,
			HasToys:           false,
			SocialOpportunity: 0.6,
			ExplorationValue:  0.8,
			DangerLevel:       0.2,
			Comfort:           0.5,
		},
		Effects: LocationEffects{
			EnergyDrain:      1.4,
			HappinessBonus:   0.2,
			StressModifier:   0.6,
			ExplorationBonus: 0.6,
			SocialBonus:      0.4,
			HealthRisk:       0.1,
		},
		Climate:  ClimateCoastal,
		Unlocked: false,
	}

	// Forest - great for exploration
	lm.Locations["forest"] = &Location{
		Type:        LocationForest,
		Name:        "Forest",
		Description: "A dense forest full of wildlife and mystery",
		Features: LocationFeatures{
			HasWater:          true,
			HasShelter:        true,
			HasFood:           true,
			HasToys:           false,
			SocialOpportunity: 0.2,
			ExplorationValue:  1.0,
			DangerLevel:       0.3,
			Comfort:           0.4,
		},
		Effects: LocationEffects{
			EnergyDrain:      1.3,
			HappinessBonus:   0.1,
			StressModifier:   0.9,
			ExplorationBonus: 0.8,
			SocialBonus:      0.1,
			HealthRisk:       0.15,
		},
		Climate:  ClimateTemperate,
		Unlocked: false,
	}

	// Mountain - challenging but rewarding
	lm.Locations["mountain"] = &Location{
		Type:        LocationMountain,
		Name:        "Mountain",
		Description: "A majestic mountain with breathtaking views",
		Features: LocationFeatures{
			HasWater:          true,
			HasShelter:        false,
			HasFood:           false,
			HasToys:           false,
			SocialOpportunity: 0.1,
			ExplorationValue:  0.9,
			DangerLevel:       0.4,
			Comfort:           0.3,
		},
		Effects: LocationEffects{
			EnergyDrain:      1.6,
			HappinessBonus:   0.15,
			StressModifier:   1.1,
			ExplorationBonus: 0.7,
			SocialBonus:      0.05,
			HealthRisk:       0.2,
		},
		Climate:  ClimateMountainous,
		Unlocked: false,
	}

	// City - lots of stimulation
	lm.Locations["city"] = &Location{
		Type:        LocationCity,
		Name:        "City",
		Description: "A bustling city with shops and activities",
		Features: LocationFeatures{
			HasWater:          true,
			HasShelter:        true,
			HasFood:           true,
			HasToys:           true,
			SocialOpportunity: 0.8,
			ExplorationValue:  0.6,
			DangerLevel:       0.25,
			Comfort:           0.5,
		},
		Effects: LocationEffects{
			EnergyDrain:      1.1,
			HappinessBonus:   0.1,
			StressModifier:   1.3,
			ExplorationBonus: 0.5,
			SocialBonus:      0.5,
			HealthRisk:       0.1,
		},
		Climate:  ClimateTemperate,
		Unlocked: false,
	}

	// Farm - peaceful rural environment
	lm.Locations["farm"] = &Location{
		Type:        LocationFarm,
		Name:        "Farm",
		Description: "A peaceful farm with animals and open fields",
		Features: LocationFeatures{
			HasWater:          true,
			HasShelter:        true,
			HasFood:           true,
			HasToys:           false,
			SocialOpportunity: 0.4,
			ExplorationValue:  0.7,
			DangerLevel:       0.1,
			Comfort:           0.8,
		},
		Effects: LocationEffects{
			EnergyDrain:      1.0,
			HappinessBonus:   0.12,
			StressModifier:   0.6,
			ExplorationBonus: 0.5,
			SocialBonus:      0.3,
			HealthRisk:       0.05,
		},
		Climate:  ClimateTemperate,
		Unlocked: false,
	}
}

// TravelTo moves the pet to a new location
func (lm *LocationManager) TravelTo(locationName string, currentGameTime float64) bool {
	location, exists := lm.Locations[locationName]
	if !exists {
		return false
	}

	if !location.Unlocked {
		return false
	}

	// Record departure from current location
	if lm.CurrentLocation != nil && len(lm.TravelHistory) > 0 {
		lastRecord := &lm.TravelHistory[len(lm.TravelHistory)-1]
		lastRecord.Duration = currentGameTime - lastRecord.ArrivalTime
		lm.CurrentLocation.TimeSpent += lastRecord.Duration
	}

	// Move to new location
	lm.CurrentLocation = location
	location.VisitCount++

	// Record arrival
	lm.TravelHistory = append(lm.TravelHistory, TravelRecord{
		LocationName: locationName,
		ArrivalTime:  currentGameTime,
		Duration:     0,
	})

	// Keep history manageable
	if len(lm.TravelHistory) > 100 {
		lm.TravelHistory = lm.TravelHistory[1:]
	}

	return true
}

// UnlockLocation makes a location available for travel
func (lm *LocationManager) UnlockLocation(locationName string) bool {
	location, exists := lm.Locations[locationName]
	if !exists {
		return false
	}

	location.Unlocked = true
	return true
}

// GetUnlockedLocations returns all unlocked locations
func (lm *LocationManager) GetUnlockedLocations() []*Location {
	var unlocked []*Location
	for _, loc := range lm.Locations {
		if loc.Unlocked {
			unlocked = append(unlocked, loc)
		}
	}
	return unlocked
}

// GetLocationEffects returns effects for the current location
func (lm *LocationManager) GetLocationEffects() LocationEffects {
	if lm.CurrentLocation == nil {
		return LocationEffects{
			EnergyDrain:    1.0,
			StressModifier: 1.0,
		}
	}
	return lm.CurrentLocation.Effects
}

// GetLocationFeatures returns features of the current location
func (lm *LocationManager) GetLocationFeatures() LocationFeatures {
	if lm.CurrentLocation == nil {
		return LocationFeatures{}
	}
	return lm.CurrentLocation.Features
}

// ApplyWeatherToLocation modifies location effects based on weather
func (lm *LocationManager) ApplyWeatherToLocation(weather *WeatherConditions) LocationEffects {
	effects := lm.GetLocationEffects()

	if lm.CurrentLocation == nil {
		return effects
	}

	// Locations without shelter are more affected by weather
	if !lm.CurrentLocation.Features.HasShelter {
		// Bad weather increases stress and energy drain
		if weather.Type == WeatherRainy || weather.Type == WeatherStormy {
			effects.EnergyDrain *= 1.3
			effects.StressModifier *= 1.4
			effects.HappinessBonus *= 0.5
		}

		// Extreme temperatures
		if weather.Temperature < 5 || weather.Temperature > 35 {
			effects.EnergyDrain *= 1.2
			effects.HealthRisk += 0.1
		}
	}

	// Weather affects exploration value
	if weather.Visibility < 0.3 {
		effects.ExplorationBonus *= 0.5
	}

	// Storms affect social opportunities
	if weather.Type == WeatherStormy {
		effects.SocialBonus *= 0.2
	}

	return effects
}

// GetCurrentLocationName returns the name of the current location
func (lm *LocationManager) GetCurrentLocationName() string {
	if lm.CurrentLocation == nil {
		return "Unknown"
	}
	return lm.CurrentLocation.Name
}

// IsAtHome returns whether the pet is currently at home
func (lm *LocationManager) IsAtHome() bool {
	return lm.CurrentLocation != nil && lm.CurrentLocation.Type == LocationHome
}

// GetExplorationProgress returns overall exploration progress
func (lm *LocationManager) GetExplorationProgress() float64 {
	totalLocations := len(lm.Locations)
	unlockedCount := 0

	for _, loc := range lm.Locations {
		if loc.Unlocked {
			unlockedCount++
		}
	}

	return float64(unlockedCount) / float64(totalLocations)
}

// GetMostVisitedLocation returns the location with most visits
func (lm *LocationManager) GetMostVisitedLocation() *Location {
	var mostVisited *Location
	maxVisits := 0

	for _, loc := range lm.Locations {
		if loc.VisitCount > maxVisits {
			maxVisits = loc.VisitCount
			mostVisited = loc
		}
	}

	return mostVisited
}

// GetTotalTimeOutdoors returns total time spent outside of home
func (lm *LocationManager) GetTotalTimeOutdoors() float64 {
	var total float64
	for _, loc := range lm.Locations {
		if loc.Type != LocationHome {
			total += loc.TimeSpent
		}
	}
	return total
}

// CalculateLocationCompatibility returns how well a location suits current conditions
func (lm *LocationManager) CalculateLocationCompatibility(loc *Location, weather *WeatherConditions, season Season) float64 {
	compatibility := 1.0

	// Weather compatibility
	if !loc.Features.HasShelter {
		if weather.Type == WeatherRainy || weather.Type == WeatherStormy || weather.Type == WeatherSnowy {
			compatibility *= 0.5
		}
	}

	// Climate compatibility with weather
	switch loc.Climate {
	case ClimateTropical:
		if weather.Temperature < 20 {
			compatibility *= 0.7
		}
	case ClimateArid:
		if weather.Humidity > 0.7 {
			compatibility *= 0.8
		}
	case ClimateCold:
		if weather.Temperature > 25 {
			compatibility *= 0.6
		}
	case ClimateMountainous:
		if weather.WindSpeed > 40 {
			compatibility *= 0.5
		}
	}

	// Seasonal compatibility
	switch season {
	case SeasonWinter:
		if loc.Type == LocationBeach || loc.Type == LocationDesert {
			compatibility *= 0.3
		}
	case SeasonSummer:
		if loc.Type == LocationTundra {
			compatibility *= 0.5
		}
	}

	return math.Max(0.0, math.Min(1.0, compatibility))
}
