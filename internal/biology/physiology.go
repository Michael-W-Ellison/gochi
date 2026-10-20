package biology

// Physiology models the pet's organ systems and what is passing through
// its body. Organ health ranges from 0.0 (failing) to 1.0 (healthy); an
// organ system below organWarningLevel starts to cost the pet health.
type Physiology struct {
	// Digestive system: food eaten but not yet digested (above 1.0 the
	// stomach is overfull), digested waste and urine waiting to be passed,
	// and how well food is absorbed
	Stomach         float64 `json:"stomach"`
	Bowel           float64 `json:"bowel"`
	Bladder         float64 `json:"bladder"`
	DigestiveHealth float64 `json:"digestive_health"`

	// Cardiovascular system: heart and circulation, kept fit by exercise
	CardiovascularHealth float64 `json:"cardiovascular_health"`

	// Respiratory system: airways and lungs, harmed by foul or dusty air
	RespiratoryHealth float64 `json:"respiratory_health"`

	// Nervous system: how well the pet copes with stress, worn down by
	// long-term stress
	NervousHealth float64 `json:"nervous_health"`
}

const (
	// stomachCapacity is the most food the stomach holds comfortably
	stomachCapacity = 1.0

	// digestionRate is how much food a healthy stomach digests per game day
	digestionRate = 0.6

	// wasteFraction is the share of digested food that becomes waste
	wasteFraction = 0.5

	// fitnessLoss is how much cardiovascular fitness is lost per game day
	// without exercise, down to minFitness
	fitnessLoss = 0.02
	minFitness  = 0.3

	// organWarningLevel is the organ health below which the pet's overall
	// health suffers
	organWarningLevel = 0.5
)

// NewPhysiology returns a healthy body with an empty stomach
func NewPhysiology() *Physiology {
	return &Physiology{
		DigestiveHealth:      1.0,
		CardiovascularHealth: 1.0,
		RespiratoryHealth:    1.0,
		NervousHealth:        1.0,
	}
}

// Clamp keeps every level in its range
func (ph *Physiology) Clamp() {
	ph.Stomach = clamp(ph.Stomach, 0, 1.5)
	ph.Bowel = clamp(ph.Bowel, 0, 1)
	ph.Bladder = clamp(ph.Bladder, 0, 1)
	ph.DigestiveHealth = clamp(ph.DigestiveHealth, 0, 1)
	ph.CardiovascularHealth = clamp(ph.CardiovascularHealth, 0, 1)
	ph.RespiratoryHealth = clamp(ph.RespiratoryHealth, 0, 1)
	ph.NervousHealth = clamp(ph.NervousHealth, 0, 1)
}

// Eat puts food in the stomach. It returns true if the stomach is now
// overfull, in which case the pet throws up: it loses part of the meal,
// and the digestive system and mood suffer.
func (b *BiologicalSystems) Eat(amount float64) bool {
	ph := b.Physiology
	ph.Stomach += amount
	if ph.Stomach <= stomachCapacity {
		return false
	}

	ph.Stomach = stomachCapacity / 2
	ph.DigestiveHealth -= 0.1
	b.Vitals.Nutrition -= 0.1
	b.Vitals.Happiness -= 0.1
	b.Vitals.Stress += 0.05
	ph.Clamp()
	b.Vitals.Clamp()
	return true
}

// Drink fills the bladder as the body processes water
func (b *BiologicalSystems) Drink(amount float64) {
	b.Physiology.Bladder += amount
	b.Physiology.Clamp()
}

// Absorption is the share of a meal's nutrition the body takes in: lower
// when the digestive system is unhealthy
func (b *BiologicalSystems) Absorption() float64 {
	return 0.5 + 0.5*b.Physiology.DigestiveHealth
}

// Exercise strengthens the heart and circulation; amount is roughly one
// play session per 1.0
func (b *BiologicalSystems) Exercise(amount float64) {
	b.Physiology.CardiovascularHealth += 0.05 * amount
	b.Physiology.Clamp()
}

// EnergyDrain is how much faster than normal the pet tires: an unfit
// heart and unhealthy lungs make everything more tiring
func (b *BiologicalSystems) EnergyDrain() float64 {
	ph := b.Physiology
	drain := 1.0
	if ph.CardiovascularHealth < 0.6 {
		drain += 0.6 - ph.CardiovascularHealth
	}
	if ph.RespiratoryHealth < 0.6 {
		drain += 0.6 - ph.RespiratoryHealth
	}
	return drain
}

// updatePhysiology advances the organ systems by deltaTime game days.
// airQuality (0.0 foul to 1.0 fresh) is what the pet breathes.
func (b *BiologicalSystems) updatePhysiology(deltaTime, airQuality float64) {
	ph := b.Physiology
	v := b.Vitals

	// Digestion moves food from the stomach on to be passed as waste;
	// an unhealthy gut digests slowly
	digested := min(ph.Stomach, digestionRate*ph.DigestiveHealth*deltaTime)
	ph.Stomach -= digested
	ph.Bowel += digested * wasteFraction
	if ph.Stomach <= stomachCapacity*0.8 && ph.DigestiveHealth < 1 {
		ph.DigestiveHealth += 0.02 * deltaTime // The gut recovers when not overloaded
	}

	// Fitness fades without exercise
	if ph.CardiovascularHealth > minFitness {
		ph.CardiovascularHealth = max(ph.CardiovascularHealth-fitnessLoss*deltaTime, minFitness)
	}

	// Foul air irritates the airways; fresh air lets them recover
	if airQuality < 0.6 {
		ph.RespiratoryHealth -= 0.05 * (0.6 - airQuality) / 0.6 * deltaTime
	} else {
		ph.RespiratoryHealth += 0.02 * deltaTime
	}

	// Long-term stress wears down the nervous system; calm restores it
	switch {
	case v.Stress > 0.6:
		ph.NervousHealth -= 0.03 * (v.Stress - 0.6) / 0.4 * deltaTime
	case v.Stress < 0.3:
		ph.NervousHealth += 0.02 * deltaTime
	}

	// Failing organ systems cost the pet health
	for _, organ := range []float64{ph.DigestiveHealth, ph.CardiovascularHealth, ph.RespiratoryHealth, ph.NervousHealth} {
		if organ < organWarningLevel {
			v.Health -= 0.02 * (organWarningLevel - organ) / organWarningLevel * deltaTime
		}
	}

	ph.Clamp()
}

// StressSensitivity is how much faster than normal stress builds: a worn
// nervous system copes less well
func (b *BiologicalSystems) StressSensitivity() float64 {
	return 1.0 + (1.0 - b.Physiology.NervousHealth)
}
