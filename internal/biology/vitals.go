package biology

import (
	"fmt"
	"time"
)

// VitalStats represents the core vital statistics of a digital pet.
// All values range from 0.0 (critical/depleted) to 1.0 (optimal/full).
type VitalStats struct {
	Health     float64 `json:"health"`     // Overall health status
	Energy     float64 `json:"energy"`     // Current energy level
	Hydration  float64 `json:"hydration"`  // Water level
	Nutrition  float64 `json:"nutrition"`  // Food/nutrient level
	Happiness  float64 `json:"happiness"`  // Emotional well-being
	Stress     float64 `json:"stress"`     // Stress level (higher is worse)
	Fatigue    float64 `json:"fatigue"`    // Tiredness level (higher is worse)
	Cleanliness float64 `json:"cleanliness"` // Hygiene level
}

// NewVitalStats creates a new VitalStats instance with optimal starting values
func NewVitalStats() *VitalStats {
	return &VitalStats{
		Health:      1.0,
		Energy:      1.0,
		Hydration:   1.0,
		Nutrition:   1.0,
		Happiness:   0.8,
		Stress:      0.1,
		Fatigue:     0.0,
		Cleanliness: 1.0,
	}
}

// Clamp ensures all vital stats remain within valid range [0.0, 1.0]
func (v *VitalStats) Clamp() {
	v.Health = clamp(v.Health, 0.0, 1.0)
	v.Energy = clamp(v.Energy, 0.0, 1.0)
	v.Hydration = clamp(v.Hydration, 0.0, 1.0)
	v.Nutrition = clamp(v.Nutrition, 0.0, 1.0)
	v.Happiness = clamp(v.Happiness, 0.0, 1.0)
	v.Stress = clamp(v.Stress, 0.0, 1.0)
	v.Fatigue = clamp(v.Fatigue, 0.0, 1.0)
	v.Cleanliness = clamp(v.Cleanliness, 0.0, 1.0)
}

// GetOverallWellbeing calculates an overall wellness score
// Considers positive stats (health, energy, etc.) and inverts negative stats (stress, fatigue)
func (v *VitalStats) GetOverallWellbeing() float64 {
	positive := (v.Health + v.Energy + v.Hydration + v.Nutrition + v.Happiness + v.Cleanliness) / 6.0
	negative := (v.Stress + v.Fatigue) / 2.0
	wellbeing := (positive + (1.0 - negative)) / 2.0
	return clamp(wellbeing, 0.0, 1.0)
}

// IsCritical returns true if any vital stat is at a critical level
func (v *VitalStats) IsCritical(threshold float64) bool {
	return v.Health < threshold ||
		v.Energy < threshold ||
		v.Hydration < threshold ||
		v.Nutrition < threshold ||
		v.Happiness < threshold ||
		v.Stress > (1.0-threshold) ||
		v.Fatigue > (1.0-threshold)
}

// GetCriticalStats returns a list of stat names that are at critical levels
func (v *VitalStats) GetCriticalStats(threshold float64) []string {
	var critical []string

	if v.Health < threshold {
		critical = append(critical, "Health")
	}
	if v.Energy < threshold {
		critical = append(critical, "Energy")
	}
	if v.Hydration < threshold {
		critical = append(critical, "Hydration")
	}
	if v.Nutrition < threshold {
		critical = append(critical, "Nutrition")
	}
	if v.Happiness < threshold {
		critical = append(critical, "Happiness")
	}
	if v.Stress > (1.0 - threshold) {
		critical = append(critical, "Stress")
	}
	if v.Fatigue > (1.0 - threshold) {
		critical = append(critical, "Fatigue")
	}
	if v.Cleanliness < threshold {
		critical = append(critical, "Cleanliness")
	}

	return critical
}

// PhysiologicalProcesses represents internal biological processes
type PhysiologicalProcesses struct {
	MetabolicRate        float64 `json:"metabolic_rate"`        // Rate of energy consumption
	DigestiveEfficiency  float64 `json:"digestive_efficiency"`  // How well food is processed
	ImmuneStrength       float64 `json:"immune_strength"`       // Disease resistance
	CognitiveCapacity    float64 `json:"cognitive_capacity"`    // Learning and memory ability
	EmotionalResilience  float64 `json:"emotional_resilience"`  // Stress resistance
	CardiovascularHealth float64 `json:"cardiovascular_health"` // Heart/circulation health
	RespiratoryHealth    float64 `json:"respiratory_health"`    // Lung health
	Age                  float64 `json:"age"`                   // Age in days
}

// NewPhysiologicalProcesses creates a new PhysiologicalProcesses with healthy defaults
func NewPhysiologicalProcesses() *PhysiologicalProcesses {
	return &PhysiologicalProcesses{
		MetabolicRate:        1.0,
		DigestiveEfficiency:  0.9,
		ImmuneStrength:       0.9,
		CognitiveCapacity:    0.8,
		EmotionalResilience:  0.7,
		CardiovascularHealth: 1.0,
		RespiratoryHealth:    1.0,
		Age:                  0.0,
	}
}

// BiologicalSystems manages all biological simulation aspects of a pet
type BiologicalSystems struct {
	Vitals      *VitalStats
	Processes   *PhysiologicalProcesses
	BirthTime   time.Time
	LastUpdate  time.Time
	IsAlive     bool
	CauseOfDeath string
}

// NewBiologicalSystems creates a new biological system for a pet
func NewBiologicalSystems() *BiologicalSystems {
	now := time.Now()
	return &BiologicalSystems{
		Vitals:     NewVitalStats(),
		Processes:  NewPhysiologicalProcesses(),
		BirthTime:  now,
		LastUpdate: now,
		IsAlive:    true,
	}
}

// Update processes biological changes over time
func (b *BiologicalSystems) Update(deltaTime float64) {
	if !b.IsAlive {
		return
	}

	b.LastUpdate = time.Now()

	// Age the pet
	b.Processes.Age += deltaTime

	// Process metabolism - convert nutrition to energy
	b.processMetabolism(deltaTime)

	// Natural decay of vital stats
	b.decayVitalStats(deltaTime)

	// Check for death conditions
	b.CheckDeathConditions()

	// Ensure all values stay in valid range
	b.Vitals.Clamp()
}

// processMetabolism handles energy conversion and consumption
func (b *BiologicalSystems) processMetabolism(deltaTime float64) {
	// Base metabolic rate consumption
	energyConsumption := b.Processes.MetabolicRate * deltaTime * 0.01

	// Convert nutrition to energy if energy is low
	if b.Vitals.Energy < 0.5 && b.Vitals.Nutrition > 0.1 {
		nutritionToEnergy := b.Processes.DigestiveEfficiency * deltaTime * 0.05
		b.Vitals.Nutrition -= nutritionToEnergy
		b.Vitals.Energy += nutritionToEnergy * 0.8 // 80% efficiency
	}

	// Consume energy
	b.Vitals.Energy -= energyConsumption

	// Stress increases metabolic rate
	b.Processes.MetabolicRate = 1.0 + (b.Vitals.Stress * 0.5)
}

// decayVitalStats handles natural decay of stats over time
func (b *BiologicalSystems) decayVitalStats(deltaTime float64) {
	// Natural decay rates (per game day)
	b.Vitals.Hydration -= deltaTime * 0.05
	b.Vitals.Nutrition -= deltaTime * 0.03
	b.Vitals.Energy -= deltaTime * 0.02

	// Fatigue builds up over time awake
	b.Vitals.Fatigue += deltaTime * 0.01

	// Cleanliness decreases slowly
	b.Vitals.Cleanliness -= deltaTime * 0.01

	// Happiness slowly trends toward neutral
	if b.Vitals.Happiness > 0.5 {
		b.Vitals.Happiness -= deltaTime * 0.005
	}

	// Stress slowly decreases if conditions are good
	if b.Vitals.GetOverallWellbeing() > 0.7 {
		b.Vitals.Stress -= deltaTime * 0.02
	} else {
		// Stress increases if wellbeing is poor
		b.Vitals.Stress += deltaTime * 0.01
	}

	// Health is affected by other vital stats
	wellbeing := b.Vitals.GetOverallWellbeing()
	if wellbeing < 0.4 {
		b.Vitals.Health -= deltaTime * 0.01
	} else if wellbeing > 0.8 {
		// Slowly recover health when wellbeing is high
		b.Vitals.Health += deltaTime * 0.005
	}
}

// CheckDeathConditions determines if the pet has died
func (b *BiologicalSystems) CheckDeathConditions() {
	if b.Vitals.Health <= 0.0 {
		b.IsAlive = false
		b.CauseOfDeath = "Health depleted"
	} else if b.Vitals.Hydration <= 0.0 {
		b.IsAlive = false
		b.CauseOfDeath = "Dehydration"
	} else if b.Vitals.Nutrition <= 0.0 && b.Vitals.Energy <= 0.0 {
		b.IsAlive = false
		b.CauseOfDeath = "Starvation"
	}
}

// GetAgeInDays returns the pet's age in days
func (b *BiologicalSystems) GetAgeInDays() float64 {
	return b.Processes.Age
}

// CalculateLifespan estimates remaining lifespan based on current health
func (b *BiologicalSystems) CalculateLifespan() float64 {
	// Base lifespan of 30 days, modified by health
	baseLifespan := 30.0
	healthModifier := b.Vitals.Health
	immuneModifier := b.Processes.ImmuneStrength

	estimatedTotal := baseLifespan * healthModifier * immuneModifier
	remaining := estimatedTotal - b.Processes.Age

	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// GetStatus returns a human-readable status string
func (b *BiologicalSystems) GetStatus() string {
	if !b.IsAlive {
		return fmt.Sprintf("Deceased (%s)", b.CauseOfDeath)
	}

	wellbeing := b.Vitals.GetOverallWellbeing()

	switch {
	case wellbeing >= 0.9:
		return "Excellent"
	case wellbeing >= 0.7:
		return "Good"
	case wellbeing >= 0.5:
		return "Fair"
	case wellbeing >= 0.3:
		return "Poor"
	default:
		return "Critical"
	}
}

// Helper function to clamp values
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
