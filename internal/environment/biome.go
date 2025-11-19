package environment

import (
	"sync"
)

// BiomeType represents different environmental biomes
type BiomeType int

const (
	BiomeForest BiomeType = iota
	BiomeDesert
	BiomeOcean
	BiomeMountain
	BiomeGrassland
	BiomeTundra
	BiomeJungle
	BiomeSwamp
	BiomeUrban
	BiomeCave
)

// String returns the string representation of biome type
func (b BiomeType) String() string {
	switch b {
	case BiomeForest:
		return "Forest"
	case BiomeDesert:
		return "Desert"
	case BiomeOcean:
		return "Ocean"
	case BiomeMountain:
		return "Mountain"
	case BiomeGrassland:
		return "Grassland"
	case BiomeTundra:
		return "Tundra"
	case BiomeJungle:
		return "Jungle"
	case BiomeSwamp:
		return "Swamp"
	case BiomeUrban:
		return "Urban"
	case BiomeCave:
		return "Cave"
	default:
		return "Unknown"
	}
}

// BiomeCharacteristics defines the properties of a biome
type BiomeCharacteristics struct {
	Type                 BiomeType
	BaseTemperature      float64   // Average temperature
	TemperatureVariation float64   // How much temperature varies
	Humidity             float64   // 0-1
	Precipitation        float64   // 0-1 average rainfall
	FoodAbundance        float64   // 0-1 base food availability
	WaterAvailability    float64   // 0-1 base water access
	ShelterQuality       float64   // 0-1 natural shelter
	DangerLevel          float64   // 0-1 environmental hazards
	ExplorationValue     float64   // 0-1 novelty/interest
	SocialDensity        float64   // 0-1 likelihood of meeting others
	ActivityTypes        []string  // Available activities
	NaturalResources     []string  // Resources found here
	SeasonalVariation    float64   // 0-1 how much seasons affect this biome
}

// Location represents a specific place within a biome
type Location struct {
	mu sync.RWMutex

	Name         string
	Biome        BiomeType
	Coordinates  Coordinates
	Discovered   bool
	VisitCount   int
	LastVisited  float64 // Game time
	Characteristics BiomeCharacteristics

	// Dynamic state
	CurrentPopulation int     // Number of pets currently here
	ResourceLevel     float64 // 0-1, how depleted resources are
}

// Coordinates represents a location in 2D space
type Coordinates struct {
	X float64
	Y float64
}

// Distance calculates distance between two coordinates
func (c Coordinates) Distance(other Coordinates) float64 {
	dx := c.X - other.X
	dy := c.Y - other.Y
	return (dx*dx + dy*dy) // Squared distance (avoiding sqrt for performance)
}

// NewLocation creates a new location
func NewLocation(name string, biome BiomeType, coords Coordinates) *Location {
	return &Location{
		Name:         name,
		Biome:        biome,
		Coordinates:  coords,
		Discovered:   false,
		VisitCount:   0,
		LastVisited:  0,
		Characteristics: GetBiomeCharacteristics(biome),
		CurrentPopulation: 0,
		ResourceLevel: 1.0,
	}
}

// GetBiomeCharacteristics returns the characteristics for a biome type
func GetBiomeCharacteristics(biome BiomeType) BiomeCharacteristics {
	switch biome {
	case BiomeForest:
		return BiomeCharacteristics{
			Type:                 BiomeForest,
			BaseTemperature:      15.0,
			TemperatureVariation: 15.0,
			Humidity:             0.7,
			Precipitation:        0.6,
			FoodAbundance:        0.8,
			WaterAvailability:    0.7,
			ShelterQuality:       0.8,
			DangerLevel:          0.3,
			ExplorationValue:     0.7,
			SocialDensity:        0.5,
			ActivityTypes:        []string{"exploring", "foraging", "climbing", "hiding"},
			NaturalResources:     []string{"wood", "berries", "herbs", "mushrooms"},
			SeasonalVariation:    0.9,
		}

	case BiomeDesert:
		return BiomeCharacteristics{
			Type:                 BiomeDesert,
			BaseTemperature:      30.0,
			TemperatureVariation: 20.0,
			Humidity:             0.2,
			Precipitation:        0.1,
			FoodAbundance:        0.3,
			WaterAvailability:    0.2,
			ShelterQuality:       0.4,
			DangerLevel:          0.7,
			ExplorationValue:     0.8,
			SocialDensity:        0.2,
			ActivityTypes:        []string{"exploring", "digging", "sunbathing"},
			NaturalResources:     []string{"sand", "cacti", "minerals"},
			SeasonalVariation:    0.4,
		}

	case BiomeOcean:
		return BiomeCharacteristics{
			Type:                 BiomeOcean,
			BaseTemperature:      18.0,
			TemperatureVariation: 8.0,
			Humidity:             1.0,
			Precipitation:        0.7,
			FoodAbundance:        0.7,
			WaterAvailability:    1.0,
			ShelterQuality:       0.3,
			DangerLevel:          0.5,
			ExplorationValue:     0.9,
			SocialDensity:        0.6,
			ActivityTypes:        []string{"swimming", "diving", "fishing", "surfing"},
			NaturalResources:     []string{"fish", "shells", "seaweed", "coral"},
			SeasonalVariation:    0.5,
		}

	case BiomeMountain:
		return BiomeCharacteristics{
			Type:                 BiomeMountain,
			BaseTemperature:      5.0,
			TemperatureVariation: 18.0,
			Humidity:             0.5,
			Precipitation:        0.6,
			FoodAbundance:        0.4,
			WaterAvailability:    0.6,
			ShelterQuality:       0.7,
			DangerLevel:          0.6,
			ExplorationValue:     0.9,
			SocialDensity:        0.3,
			ActivityTypes:        []string{"climbing", "exploring", "soaring", "jumping"},
			NaturalResources:     []string{"stone", "crystals", "snow", "herbs"},
			SeasonalVariation:    0.8,
		}

	case BiomeGrassland:
		return BiomeCharacteristics{
			Type:                 BiomeGrassland,
			BaseTemperature:      20.0,
			TemperatureVariation: 15.0,
			Humidity:             0.5,
			Precipitation:        0.5,
			FoodAbundance:        0.7,
			WaterAvailability:    0.6,
			ShelterQuality:       0.4,
			DangerLevel:          0.4,
			ExplorationValue:     0.5,
			SocialDensity:        0.7,
			ActivityTypes:        []string{"running", "grazing", "playing", "socializing"},
			NaturalResources:     []string{"grass", "flowers", "seeds", "insects"},
			SeasonalVariation:    0.8,
		}

	case BiomeTundra:
		return BiomeCharacteristics{
			Type:                 BiomeTundra,
			BaseTemperature:      -5.0,
			TemperatureVariation: 15.0,
			Humidity:             0.4,
			Precipitation:        0.3,
			FoodAbundance:        0.2,
			WaterAvailability:    0.5,
			ShelterQuality:       0.5,
			DangerLevel:          0.8,
			ExplorationValue:     0.6,
			SocialDensity:        0.2,
			ActivityTypes:        []string{"foraging", "huddling", "digging"},
			NaturalResources:     []string{"ice", "lichen", "roots"},
			SeasonalVariation:    0.7,
		}

	case BiomeJungle:
		return BiomeCharacteristics{
			Type:                 BiomeJungle,
			BaseTemperature:      26.0,
			TemperatureVariation: 8.0,
			Humidity:             0.9,
			Precipitation:        0.9,
			FoodAbundance:        0.9,
			WaterAvailability:    0.9,
			ShelterQuality:       0.9,
			DangerLevel:          0.5,
			ExplorationValue:     1.0,
			SocialDensity:        0.8,
			ActivityTypes:        []string{"swinging", "climbing", "foraging", "exploring"},
			NaturalResources:     []string{"fruits", "vines", "flowers", "nectar"},
			SeasonalVariation:    0.3,
		}

	case BiomeSwamp:
		return BiomeCharacteristics{
			Type:                 BiomeSwamp,
			BaseTemperature:      22.0,
			TemperatureVariation: 10.0,
			Humidity:             0.95,
			Precipitation:        0.8,
			FoodAbundance:        0.7,
			WaterAvailability:    1.0,
			ShelterQuality:       0.6,
			DangerLevel:          0.6,
			ExplorationValue:     0.7,
			SocialDensity:        0.4,
			ActivityTypes:        []string{"wading", "fishing", "hiding", "exploring"},
			NaturalResources:     []string{"mud", "reeds", "fish", "amphibians"},
			SeasonalVariation:    0.6,
		}

	case BiomeUrban:
		return BiomeCharacteristics{
			Type:                 BiomeUrban,
			BaseTemperature:      18.0,
			TemperatureVariation: 12.0,
			Humidity:             0.5,
			Precipitation:        0.5,
			FoodAbundance:        0.8,
			WaterAvailability:    0.9,
			ShelterQuality:       0.9,
			DangerLevel:          0.4,
			ExplorationValue:     0.6,
			SocialDensity:        1.0,
			ActivityTypes:        []string{"exploring", "socializing", "playing", "scavenging"},
			NaturalResources:     []string{"food", "toys", "structures", "people"},
			SeasonalVariation:    0.5,
		}

	case BiomeCave:
		return BiomeCharacteristics{
			Type:                 BiomeCave,
			BaseTemperature:      12.0,
			TemperatureVariation: 3.0,
			Humidity:             0.8,
			Precipitation:        0.2,
			FoodAbundance:        0.3,
			WaterAvailability:    0.5,
			ShelterQuality:       1.0,
			DangerLevel:          0.5,
			ExplorationValue:     0.8,
			SocialDensity:        0.3,
			ActivityTypes:        []string{"exploring", "hiding", "resting", "spelunking"},
			NaturalResources:     []string{"minerals", "crystals", "water", "mushrooms"},
			SeasonalVariation:    0.1,
		}

	default:
		return BiomeCharacteristics{
			Type:                 BiomeGrassland,
			BaseTemperature:      20.0,
			TemperatureVariation: 15.0,
			Humidity:             0.5,
			Precipitation:        0.5,
			FoodAbundance:        0.6,
			WaterAvailability:    0.6,
			ShelterQuality:       0.5,
			DangerLevel:          0.4,
			ExplorationValue:     0.5,
			SocialDensity:        0.5,
			ActivityTypes:        []string{"exploring", "playing"},
			NaturalResources:     []string{"grass", "water"},
			SeasonalVariation:    0.7,
		}
	}
}

// Visit marks a location as visited
func (l *Location) Visit(gameTime float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Discovered = true
	l.VisitCount++
	l.LastVisited = gameTime
}

// Enter adds a pet to the location
func (l *Location) Enter() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.CurrentPopulation++
}

// Leave removes a pet from the location
func (l *Location) Leave() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.CurrentPopulation > 0 {
		l.CurrentPopulation--
	}
}

// GetPopulation returns current population
func (l *Location) GetPopulation() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.CurrentPopulation
}

// DepleteResources reduces available resources
func (l *Location) DepleteResources(amount float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.ResourceLevel -= amount
	if l.ResourceLevel < 0 {
		l.ResourceLevel = 0
	}
}

// RegenerateResources restores resources over time
func (l *Location) RegenerateResources(deltaTime float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Resources regenerate slowly when not overpopulated
	if l.CurrentPopulation < 5 {
		regenRate := 0.01 * deltaTime // 1% per second when empty
		l.ResourceLevel += regenRate
		if l.ResourceLevel > 1.0 {
			l.ResourceLevel = 1.0
		}
	}
}

// GetEffectiveFoodAvailability returns food availability accounting for depletion
func (l *Location) GetEffectiveFoodAvailability() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Characteristics.FoodAbundance * l.ResourceLevel
}

// GetDescription returns a text description of the location
func (l *Location) GetDescription() string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	desc := l.Name + " (" + l.Biome.String() + ")"

	if !l.Discovered {
		return desc + " - Undiscovered"
	}

	if l.CurrentPopulation > 0 {
		desc += " - Currently populated"
	}

	if l.ResourceLevel < 0.3 {
		desc += " - Resources depleted"
	} else if l.ResourceLevel > 0.9 {
		desc += " - Abundant resources"
	}

	return desc
}

// IsHostile returns whether the location is dangerous
func (l *Location) IsHostile() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.Characteristics.DangerLevel > 0.6
}

// GetComfortLevel returns how comfortable the location is (0-1)
func (l *Location) GetComfortLevel() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	comfort := 1.0

	// Danger reduces comfort
	comfort -= l.Characteristics.DangerLevel * 0.5

	// Resource depletion reduces comfort
	comfort -= (1.0 - l.ResourceLevel) * 0.3

	// Overcrowding reduces comfort
	if l.CurrentPopulation > 10 {
		comfort -= 0.2
	}

	if comfort < 0 {
		comfort = 0
	}

	return comfort
}

// BiomeManager manages multiple locations
type BiomeManager struct {
	mu sync.RWMutex

	locations      map[string]*Location
	currentLocation *Location
}

// NewBiomeManager creates a new biome manager
func NewBiomeManager() *BiomeManager {
	bm := &BiomeManager{
		locations: make(map[string]*Location),
	}

	// Create default starting location
	startLocation := NewLocation("Home Meadow", BiomeGrassland, Coordinates{X: 0, Y: 0})
	startLocation.Discovered = true
	bm.locations[startLocation.Name] = startLocation
	bm.currentLocation = startLocation

	return bm
}

// AddLocation adds a new location
func (bm *BiomeManager) AddLocation(location *Location) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.locations[location.Name] = location
}

// GetLocation retrieves a location by name
func (bm *BiomeManager) GetLocation(name string) (*Location, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	loc, exists := bm.locations[name]
	return loc, exists
}

// GetCurrentLocation returns the current location
func (bm *BiomeManager) GetCurrentLocation() *Location {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.currentLocation
}

// TravelTo moves to a different location
func (bm *BiomeManager) TravelTo(locationName string, gameTime float64) bool {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	newLocation, exists := bm.locations[locationName]
	if !exists {
		return false
	}

	// Leave current location
	if bm.currentLocation != nil {
		bm.currentLocation.Leave()
	}

	// Enter new location
	newLocation.Visit(gameTime)
	newLocation.Enter()
	bm.currentLocation = newLocation

	return true
}

// GetDiscoveredLocations returns all discovered locations
func (bm *BiomeManager) GetDiscoveredLocations() []*Location {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	discovered := make([]*Location, 0)
	for _, loc := range bm.locations {
		if loc.Discovered {
			discovered = append(discovered, loc)
		}
	}

	return discovered
}

// GetNearbyLocations returns locations within a certain distance
func (bm *BiomeManager) GetNearbyLocations(maxDistance float64) []*Location {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if bm.currentLocation == nil {
		return []*Location{}
	}

	nearby := make([]*Location, 0)
	currentCoords := bm.currentLocation.Coordinates

	for _, loc := range bm.locations {
		if loc.Name == bm.currentLocation.Name {
			continue
		}

		distSquared := currentCoords.Distance(loc.Coordinates)
		if distSquared <= maxDistance*maxDistance {
			nearby = append(nearby, loc)
		}
	}

	return nearby
}

// UpdateAll updates all locations
func (bm *BiomeManager) UpdateAll(deltaTime float64) {
	bm.mu.RLock()
	locations := make([]*Location, 0, len(bm.locations))
	for _, loc := range bm.locations {
		locations = append(locations, loc)
	}
	bm.mu.RUnlock()

	// Update each location without holding the main lock
	for _, loc := range locations {
		loc.RegenerateResources(deltaTime)
	}
}

// GetStats returns biome manager statistics
func (bm *BiomeManager) GetStats() map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	discoveredCount := 0
	for _, loc := range bm.locations {
		if loc.Discovered {
			discoveredCount++
		}
	}

	currentBiome := "None"
	if bm.currentLocation != nil {
		currentBiome = bm.currentLocation.Biome.String()
	}

	return map[string]interface{}{
		"total_locations":      len(bm.locations),
		"discovered_locations": discoveredCount,
		"current_location":     currentBiome,
	}
}
