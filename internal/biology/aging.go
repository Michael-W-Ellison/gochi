package biology

// Aging: a pet is young, then an adult, then a senior. Seniors slow down
// as they near the end of their lifespan: their metabolism, immunity,
// fitness and memory decline. A pet's lifespan depends on its species,
// its genes (longevity) and how well it has been cared for over its life,
// and can be extended (LifeExtension).

// LifeStage is a stage of life
type LifeStage int

const (
	StageYoung LifeStage = iota
	StageAdult
	StageSenior
)

// String names the life stage
func (s LifeStage) String() string {
	return [...]string{"Young", "Adult", "Senior"}[s]
}

// Ages (in game days) at which life stages begin, and the lifespan of an
// average pet given average care. At the default pace (one game day per
// real minute of play) a pet is young for an hour of play, an adult until
// 100 hours and lives about 150.
const (
	AdultAge     = 60.0
	SeniorAge    = 6000.0
	BaseLifespan = 9000.0
)

// careQualityRate is how fast the record of care follows the pet's
// wellbeing: the share of the gap closed per game day, so it reflects
// roughly the last 200 game days
const careQualityRate = 0.005

// defaultCareQuality is the care record of a pet that has no record yet
const defaultCareQuality = 0.5

// LifeStage returns the pet's stage of life
func (b *BiologicalSystems) LifeStage() LifeStage {
	switch age := b.Processes.Age; {
	case age < AdultAge:
		return StageYoung
	case age < SeniorAge:
		return StageAdult
	}
	return StageSenior
}

// ExpectedLifespan returns the age (in game days) the pet can expect to
// reach: the base lifespan scaled by its species, its genes and the care
// it has had, plus any extension
func (b *BiologicalSystems) ExpectedLifespan() float64 {
	species := b.SpeciesLifespan
	if species <= 0 {
		species = 1
	}
	longevity := b.Longevity
	if longevity <= 0 {
		longevity = 0.5
	}
	care := b.CareQuality
	if care <= 0 {
		care = defaultCareQuality
	}
	return BaseLifespan*species*(0.9+0.2*longevity)*(0.85+0.3*care) + b.LifeExtension
}

// CalculateLifespan returns how many game days the pet can expect to live
// (0 once it has reached its expected lifespan)
func (b *BiologicalSystems) CalculateLifespan() float64 {
	return max(b.ExpectedLifespan()-b.Processes.Age, 0)
}

// Senescence returns how far the pet is through its senior years, 0.0
// (not yet a senior) to 1.0 (at its expected lifespan)
func (b *BiologicalSystems) Senescence() float64 {
	age, lifespan := b.Processes.Age, b.ExpectedLifespan()
	start := min(SeniorAge, lifespan*2/3) // A short-lived pet ages sooner
	if age <= start {
		return 0
	}
	return clamp((age-start)/(lifespan-start), 0, 1)
}

// agingFactor scales an ability that old age weakens by up to loss
func (b *BiologicalSystems) agingFactor(loss float64) float64 {
	return 1 - loss*b.Senescence()
}

// growthTarget returns the growth hormone level for the pet's age: high
// while young, lower in adults and lower still in seniors
func (b *BiologicalSystems) growthTarget() float64 {
	switch b.LifeStage() {
	case StageYoung:
		return 0.8
	case StageAdult:
		return 0.3
	}
	return 0.3 - 0.15*b.Senescence()
}

// updateAging records how well the pet is being cared for and works out
// the abilities its age, hormones and sleep give it. Caller has advanced
// its age.
func (b *BiologicalSystems) updateAging(deltaTime float64) {
	if b.CareQuality <= 0 {
		b.CareQuality = defaultCareQuality
	}
	b.CareQuality = approach(b.CareQuality, b.Vitals.GetOverallWellbeing(), careQualityRate, deltaTime)

	h, p := b.Hormones, b.Processes

	// Metabolism: fast while growing, slowing with old age; stress
	// hormones speed it up and the night slows it down
	stage := 1.0
	if b.LifeStage() == StageYoung {
		stage = 1.2
	}
	p.MetabolicRate = stage * b.agingFactor(0.25) * (1 + 0.3*h.Cortisol) * (1 - 0.1*h.Melatonin)

	// Learning and memory: sharpest when young, dulled by old age and
	// lack of sleep
	cognition := 0.8
	if b.LifeStage() == StageYoung {
		cognition = 0.9
	}
	p.CognitiveCapacity = clamp(cognition*b.agingFactor(0.3)*(1-0.3*b.SleepDebtLevel()), 0.1, 1)

	// Coping with stress: helped by serotonin, oxytocin and a healthy
	// nervous system, hurt by lasting stress
	p.EmotionalResilience = clamp(0.25+0.3*h.Serotonin+0.2*h.Oxytocin+0.25*b.Physiology.NervousHealth-0.2*h.Cortisol, 0.1, 1)

	// Digestion weakens in old age
	p.DigestiveEfficiency = 0.9 * b.agingFactor(0.2)
}

// CheckOldAge ends the life of a pet that has reached its expected
// lifespan, unless OldAgeDeath is off
func (b *BiologicalSystems) CheckOldAge() {
	if b.OldAgeDeath && b.IsAlive && b.Processes.Age >= b.ExpectedLifespan() {
		b.IsAlive = false
		b.CauseOfDeath = "Old age"
	}
}
