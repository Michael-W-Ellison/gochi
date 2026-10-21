package biology

import (
	"fmt"
	"time"
)

// VitalStats represents the core vital statistics of a digital pet.
// All values range from 0.0 (critical/depleted) to 1.0 (optimal/full).
type VitalStats struct {
	Health      float64 `json:"health"`      // Overall health status
	Energy      float64 `json:"energy"`      // Current energy level
	Hydration   float64 `json:"hydration"`   // Water level
	Nutrition   float64 `json:"nutrition"`   // Food/nutrient level
	Happiness   float64 `json:"happiness"`   // Emotional well-being
	Stress      float64 `json:"stress"`      // Stress level (higher is worse)
	Fatigue     float64 `json:"fatigue"`     // Tiredness level (higher is worse)
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
	MetabolicRate       float64 `json:"metabolic_rate"`       // Rate of energy consumption
	DigestiveEfficiency float64 `json:"digestive_efficiency"` // How well food is processed
	ImmuneStrength      float64 `json:"immune_strength"`      // Disease resistance
	CognitiveCapacity   float64 `json:"cognitive_capacity"`   // Learning and memory ability
	EmotionalResilience float64 `json:"emotional_resilience"` // Stress resistance
	Age                 float64 `json:"age"`                  // Age in days
}

// NewPhysiologicalProcesses creates a new PhysiologicalProcesses with healthy defaults
func NewPhysiologicalProcesses() *PhysiologicalProcesses {
	return &PhysiologicalProcesses{
		MetabolicRate:       1.0,
		DigestiveEfficiency: 0.9,
		ImmuneStrength:      0.9,
		CognitiveCapacity:   0.8,
		EmotionalResilience: 0.7,
		Age:                 0.0,
	}
}

// BiologicalSystems manages all biological simulation aspects of a pet
type BiologicalSystems struct {
	Vitals       *VitalStats
	Processes    *PhysiologicalProcesses
	Physiology   *Physiology      `json:"physiology"` // Organ systems; nil in saves made before they existed
	Hormones     *Hormones        `json:"hormones"`   // nil in saves made before they existed
	Circadian    *CircadianRhythm `json:"circadian"`  // Sleep phases and sleep debt; nil in older saves
	BirthTime    time.Time
	LastUpdate   time.Time
	IsAlive      bool
	CauseOfDeath string

	// Aging: the record of how well the pet has been cared for (0.0 to
	// 1.0, 0 meaning no record yet), its inherited longevity (0.0 to 1.0,
	// 0 meaning average) and game days added to its lifespan
	CareQuality   float64 `json:"care_quality"`
	Longevity     float64 `json:"longevity"`
	LifeExtension float64 `json:"life_extension"`

	// Set by the pet before each update (not saved): how long its species
	// lives relative to BaseLifespan (0 means 1), whether it can die of
	// old age, and what its body responds to from outside
	SpeciesLifespan float64         `json:"-"`
	Predispositions Predispositions `json:"-"` // What its breed and species make it prone to
	OldAgeDeath     bool            `json:"-"`
	Inputs          BodyInputs      `json:"-"`

	// DecayMultipliers scale how fast needs build up (set by the pet's
	// belongings each update; not saved)
	DecayMultipliers DecayMultipliers `json:"-"`

	// AirPollution is how foul the air the pet breathes is, 0.0 (fresh) to
	// 1.0 (foul); set from its surroundings each update (not saved, so a
	// loaded pet starts with fresh air)
	AirPollution float64 `json:"-"`
}

// DecayMultipliers scale natural decay rates. A value of 0 means 1.0
// (normal rate); 0.8 makes that need build up 20% slower.
type DecayMultipliers struct {
	Nutrition float64
	Hydration float64
	Fatigue   float64
	Energy    float64

	// What the pet's belongings and home change besides needs: how fast
	// cleanliness and happiness fall, stress eases and builds (and the
	// most it can reach, 0 for no limit), fitness fades and builds (from
	// walks and play), the airways are harmed and heal, sleep restores
	// and sleep debt is repaid, and the gut recovers
	Cleanliness       float64
	HappinessDrop     float64
	StressRecovery    float64
	StressGain        float64
	StressCap         float64
	FitnessLoss       float64
	FitnessGain       float64
	AirDamage         float64
	AirRecovery       float64
	Rest              float64
	SleepDebt         float64
	DigestiveRecovery float64

	// How fast the kidneys' reserve is lost, the eyes are irritated,
	// plaque builds on the teeth and the heart ages
	KidneyDecline float64
	EyeIrritation float64
	DentalDecline float64
	HeartAging    float64
}

// rate returns the multiplier to apply, treating 0 as the normal rate
func rate(multiplier float64) float64 {
	if multiplier <= 0 {
		return 1.0
	}
	return multiplier
}

// NewBiologicalSystems creates a new biological system for a pet
func NewBiologicalSystems() *BiologicalSystems {
	now := time.Now()
	return &BiologicalSystems{
		Vitals:     NewVitalStats(),
		Processes:  NewPhysiologicalProcesses(),
		Physiology: NewPhysiology(),
		Hormones:   NewHormones(),
		Circadian:  NewCircadianRhythm(),
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
	b.EnsurePhysiology()
	b.EnsureHormones()
	b.EnsureCircadian()

	// The sleep cycle, hormones, and the abilities age and hormones give
	b.Circadian.DebtRepay = b.DecayMultipliers.SleepDebt
	b.Circadian.Follow(deltaTime*24, b.Inputs.TimeOfDay, b.Inputs.Asleep)
	b.updateHormones(deltaTime)
	b.updateAging(deltaTime)

	// Process metabolism - convert nutrition to energy
	b.processMetabolism(deltaTime)

	// Digestion, fitness, breathing and stress regulation
	b.updatePhysiology(deltaTime, 1-b.AirPollution)

	// Natural decay of vital stats
	b.decayVitalStats(deltaTime)

	// Check for death conditions
	b.CheckDeathConditions()
	b.checkOldAge()

	// Ensure all values stay in valid range
	b.Vitals.Clamp()
}

// EnsurePhysiology gives a pet saved before organ systems existed a
// healthy body
func (b *BiologicalSystems) EnsurePhysiology() {
	if b.Physiology == nil {
		b.Physiology = NewPhysiology()
	}
	b.Physiology.EnsureSenses()
}

// EnsureCircadian gives a pet saved before the sleep cycle was tracked a
// fresh one
func (b *BiologicalSystems) EnsureCircadian() {
	if b.Circadian == nil {
		b.Circadian = NewCircadianRhythm()
	}
}

// SleepDebtLevel returns how much sleep the pet owes, 0.0 (none) to 1.0
// (12 hours or more)
func (b *BiologicalSystems) SleepDebtLevel() float64 {
	b.EnsureCircadian()
	return min(b.Circadian.GetSleepDebt()/12, 1)
}

// ProcessRest lets a sleeping pet recover over deltaTime game days:
// tiredness falls and energy returns, faster with better sleep quality
// (1.0 is normal; hormones change it further) and slower in old age
func (b *BiologicalSystems) ProcessRest(deltaTime, sleepQuality float64) {
	if !b.IsAlive {
		return
	}
	quality := sleepQuality * b.SleepQualityFactor()
	b.Vitals.Fatigue -= 0.5 * quality * deltaTime * rate(b.DecayMultipliers.Rest)
	b.Vitals.Energy += 0.15 * quality * b.agingFactor(0.3) * deltaTime
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
}

// decayVitalStats handles natural decay of stats over time
func (b *BiologicalSystems) decayVitalStats(deltaTime float64) {
	// Natural decay rates (per game day)
	b.Vitals.Hydration -= deltaTime * 0.05 * rate(b.DecayMultipliers.Hydration) * b.KidneyThirst()
	b.Vitals.Nutrition -= deltaTime * 0.03 * rate(b.DecayMultipliers.Nutrition) * b.Processes.MetabolicRate
	b.Vitals.Energy -= deltaTime * 0.02 * rate(b.DecayMultipliers.Energy) * b.EnergyDrain()

	// Fatigue builds up over time awake, faster at night (melatonin) and
	// when the pet owes sleep
	if !b.Inputs.Asleep {
		b.Vitals.Fatigue += deltaTime * 0.01 * rate(b.DecayMultipliers.Fatigue) *
			(1 + b.Hormones.Melatonin) * (1 + b.SleepDebtLevel())
	}

	// Cleanliness decreases slowly
	b.Vitals.Cleanliness -= deltaTime * 0.01 * rate(b.DecayMultipliers.Cleanliness)

	// Happiness slowly drifts towards the mood serotonin sets
	setPoint := b.moodSetPoint()
	if b.Vitals.Happiness > setPoint {
		b.Vitals.Happiness = max(b.Vitals.Happiness-deltaTime*0.005*rate(b.DecayMultipliers.HappinessDrop), setPoint)
	} else {
		b.Vitals.Happiness = min(b.Vitals.Happiness+deltaTime*0.005, setPoint)
	}

	// Stress slowly decreases if conditions are good, faster with
	// oxytocin from affection
	if b.Vitals.GetOverallWellbeing() > 0.7 {
		b.Vitals.Stress -= deltaTime * (0.02 + 0.03*b.Hormones.Oxytocin) * rate(b.DecayMultipliers.StressRecovery)
	} else {
		// Stress increases if wellbeing is poor, faster with a worn
		// nervous system
		b.Vitals.Stress += deltaTime * 0.01 * b.StressSensitivity() * rate(b.DecayMultipliers.StressGain)
	}
	if limit := b.DecayMultipliers.StressCap; limit > 0 {
		b.Vitals.Stress = min(b.Vitals.Stress, limit)
	}

	// Health is affected by other vital stats
	wellbeing := b.Vitals.GetOverallWellbeing()
	if wellbeing < 0.4 {
		b.Vitals.Health -= deltaTime * 0.01
	} else if wellbeing > 0.8 {
		// Slowly recover health when wellbeing is high, faster with
		// growth hormone, which repairs the body
		b.Vitals.Health += deltaTime * (0.005 + 0.01*b.Hormones.Growth)
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
