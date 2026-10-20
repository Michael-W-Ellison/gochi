package biology

import (
	"math"
	"testing"
)

// update advances the body by days in small steps
func update(b *BiologicalSystems, days float64) {
	for elapsed := 0.0; elapsed < days && b.IsAlive; elapsed += 0.05 {
		b.Update(min(0.05, days-elapsed))
	}
}

func TestCortisolFollowsStress(t *testing.T) {
	calm, stressed := NewBiologicalSystems(), NewBiologicalSystems()
	stressed.Vitals.Stress = 0.9
	for i := 0; i < 20; i++ {
		calm.Vitals.Stress, stressed.Vitals.Stress = 0, 0.9
		calm.Update(0.05)
		stressed.Update(0.05)
	}
	if stressed.Hormones.Cortisol <= calm.Hormones.Cortisol+0.3 {
		t.Errorf("Stress should raise cortisol: %.2f vs %.2f", stressed.Hormones.Cortisol, calm.Hormones.Cortisol)
	}
	if stressed.ImmuneResponse() >= calm.ImmuneResponse() {
		t.Errorf("Cortisol should weaken immunity: %.2f vs %.2f", stressed.ImmuneResponse(), calm.ImmuneResponse())
	}
}

func TestOxytocinPulseFades(t *testing.T) {
	b := NewBiologicalSystems()
	b.Hormones.Pulse(HormoneOxytocin, 0.6)
	high := b.Hormones.Oxytocin
	update(b, 1)
	if b.Hormones.Oxytocin >= high-0.3 {
		t.Errorf("Oxytocin should fall back after affection: %.2f -> %.2f", high, b.Hormones.Oxytocin)
	}
	b.Hormones.Pulse(HormoneOxytocin, 2)
	if b.Hormones.Oxytocin != 1 {
		t.Error("Hormones stay within 0-1")
	}
}

func TestMelatoninFollowsDarkness(t *testing.T) {
	b := NewBiologicalSystems()
	b.Inputs.Daylight = 0
	update(b, 0.5)
	dark := b.Hormones.Melatonin
	b.Inputs.Daylight = 1
	update(b, 0.5)
	if dark < 0.7 || b.Hormones.Melatonin > 0.2 {
		t.Errorf("Melatonin should be high in the dark (%.2f) and low in daylight (%.2f)", dark, b.Hormones.Melatonin)
	}
}

func TestDarknessMakesThePetTireFaster(t *testing.T) {
	day, night := NewBiologicalSystems(), NewBiologicalSystems()
	day.Inputs.Daylight, night.Inputs.Daylight = 1, 0
	update(day, 2)
	update(night, 2)
	if night.Vitals.Fatigue <= day.Vitals.Fatigue {
		t.Errorf("Fatigue should build faster at night: %.3f vs %.3f", night.Vitals.Fatigue, day.Vitals.Fatigue)
	}
}

func TestSerotoninSetsTheMood(t *testing.T) {
	low, high := NewBiologicalSystems(), NewBiologicalSystems()
	low.Hormones.Serotonin, high.Hormones.Serotonin = 0.1, 0.9
	if low.moodSetPoint() >= high.moodSetPoint() {
		t.Error("More serotonin should mean a higher resting mood")
	}
}

func TestLifeStages(t *testing.T) {
	b := NewBiologicalSystems()
	for _, c := range []struct {
		age   float64
		stage LifeStage
	}{{0, StageYoung}, {AdultAge - 1, StageYoung}, {AdultAge, StageAdult}, {SeniorAge, StageSenior}} {
		b.Processes.Age = c.age
		if got := b.LifeStage(); got != c.stage {
			t.Errorf("Age %.0f: %s, want %s", c.age, got, c.stage)
		}
	}
}

func TestLifespanDependsOnSpeciesGenesCareAndExtension(t *testing.T) {
	b := NewBiologicalSystems()
	if got := b.ExpectedLifespan(); got != BaseLifespan {
		t.Errorf("An average pet with average care should live %v, got %v", BaseLifespan, got)
	}
	b.CareQuality = 0.9
	well := b.ExpectedLifespan()
	b.CareQuality = 0.2
	if poorly := b.ExpectedLifespan(); poorly >= well {
		t.Errorf("Good care should mean a longer life: %.0f vs %.0f", well, poorly)
	}
	b.CareQuality = 0.5
	b.Longevity = 1
	if b.ExpectedLifespan() <= BaseLifespan {
		t.Error("Longevity genes should lengthen life")
	}
	b.Longevity, b.SpeciesLifespan = 0.5, 0.9
	if math.Abs(b.ExpectedLifespan()-0.9*BaseLifespan) > 1e-6 {
		t.Errorf("Species should scale lifespan, got %.0f", b.ExpectedLifespan())
	}
	b.SpeciesLifespan, b.LifeExtension = 1, 1500
	if b.ExpectedLifespan() != BaseLifespan+1500 {
		t.Errorf("Extension should add days, got %.0f", b.ExpectedLifespan())
	}
	b.Processes.Age = b.ExpectedLifespan() + 10
	if b.CalculateLifespan() != 0 {
		t.Error("No time is left past the expected lifespan")
	}
}

func TestCareRecordFollowsWellbeing(t *testing.T) {
	b := NewBiologicalSystems()
	b.CareQuality = 0.5
	b.updateAging(100)
	if b.CareQuality <= 0.5 {
		t.Errorf("A healthy pet's care record should rise, got %.2f", b.CareQuality)
	}
}

func TestOldAgeDeath(t *testing.T) {
	old := NewBiologicalSystems()
	old.Processes.Age = old.ExpectedLifespan() + 100 // Past it, whatever this update does to its care record
	old.OldAgeDeath = true
	old.Update(0.05)
	if old.IsAlive || old.CauseOfDeath != "Old age" {
		t.Errorf("A pet past its lifespan should die of old age: alive %v, %q", old.IsAlive, old.CauseOfDeath)
	}

	gentle := NewBiologicalSystems()
	gentle.Processes.Age = gentle.ExpectedLifespan() + 100
	gentle.Update(0.05)
	if !gentle.IsAlive {
		t.Error("Without OldAgeDeath a pet lives on")
	}
}

func TestOldAgeWeakensTheBody(t *testing.T) {
	young, old := NewBiologicalSystems(), NewBiologicalSystems()
	young.Processes.Age, old.Processes.Age = 1000, BaseLifespan-100
	for _, b := range []*BiologicalSystems{young, old} {
		b.updateAging(0)
	}
	if old.Senescence() <= 0.9 || young.Senescence() != 0 {
		t.Errorf("Senescence: young %.2f, old %.2f", young.Senescence(), old.Senescence())
	}
	if old.ImmuneResponse() >= young.ImmuneResponse() || old.Processes.MetabolicRate >= young.Processes.MetabolicRate ||
		old.Processes.CognitiveCapacity >= young.Processes.CognitiveCapacity {
		t.Errorf("Old age should weaken immunity, metabolism and learning: %+v vs %+v", old.Processes, young.Processes)
	}

	young.Vitals.Energy, old.Vitals.Energy = 0.2, 0.2
	young.ProcessRest(1, 1)
	old.ProcessRest(1, 1)
	if old.Vitals.Energy >= young.Vitals.Energy {
		t.Error("Old pets recover energy more slowly in sleep")
	}
}

func TestSleepCycleAndDebt(t *testing.T) {
	b := NewBiologicalSystems()
	c := b.Circadian
	for i := 0; i < 48; i++ { // Two game days awake
		c.Follow(1, float64(i%24), false)
	}
	debt := c.GetSleepDebt()
	if debt < 10 {
		t.Fatalf("Staying awake should build sleep debt, got %.1f h", debt)
	}

	c.Follow(0.5, 2, true)
	if c.GetCurrentPhase() != PhaseDeepSleep {
		t.Errorf("Half an hour into sleep the pet should be in deep sleep, got %s", c.GetCurrentPhase())
	}
	c.Follow(0.6, 2, true)
	if c.GetCurrentPhase() != PhaseREMSleep {
		t.Errorf("Later in the cycle comes REM sleep, got %s", c.GetCurrentPhase())
	}
	for i := 0; i < 20; i++ {
		c.Follow(1, 2, true)
	}
	if c.GetSleepDebt() >= debt {
		t.Errorf("Sleeping should pay off sleep debt: %.1f -> %.1f", debt, c.GetSleepDebt())
	}
	c.Follow(0.1, 8, false)
	if !c.IsAwakePhase() {
		t.Error("The rhythm should follow the pet waking")
	}
}

func TestDeepSleepReleasesGrowthHormone(t *testing.T) {
	awake, asleep := NewBiologicalSystems(), NewBiologicalSystems()
	awake.Processes.Age, asleep.Processes.Age = 100, 100
	asleep.Inputs.Asleep = true
	asleep.Circadian.Follow(0.5, 2, true) // Into deep sleep
	for i := 0; i < 2; i++ {
		awake.Update(0.01)
		asleep.Update(0.01)
		asleep.Circadian.CurrentCycleTime = 30 // Stay in deep sleep
	}
	if asleep.Hormones.Growth <= awake.Hormones.Growth {
		t.Errorf("Deep sleep should raise growth hormone: %.2f vs %.2f", asleep.Hormones.Growth, awake.Hormones.Growth)
	}
}

func TestProcessDigestion(t *testing.T) {
	b := NewBiologicalSystems()
	b.Vitals.Nutrition, b.Vitals.Hydration = 0.4, 0.5
	if b.ProcessDigestion(FoodItem{Bulk: 0.3, Nutrition: 0.3, Hydration: 0.1, Digestion: -0.1}) {
		t.Fatal("A meal should not overfill an empty stomach")
	}
	if math.Abs(b.Vitals.Nutrition-0.7) > 1e-9 || math.Abs(b.Vitals.Hydration-0.6) > 1e-9 || b.Physiology.DigestiveHealth != 0.9 {
		t.Errorf("Meal effects: nutrition %.2f, hydration %.2f, digestion %.2f", b.Vitals.Nutrition, b.Vitals.Hydration,
			b.Physiology.DigestiveHealth)
	}
	if !b.ProcessDigestion(FoodItem{Bulk: 0.9}) {
		t.Error("Overfilling the stomach should make the pet throw up")
	}
}

func TestProcessExercise(t *testing.T) {
	young, old := NewBiologicalSystems(), NewBiologicalSystems()
	young.Hormones.Growth = 0.8
	old.Processes.Age = BaseLifespan - 10
	for _, b := range []*BiologicalSystems{young, old} {
		b.Physiology.CardiovascularHealth = 0.5
		b.ProcessExercise(ExerciseWalk, 1)
	}
	if young.Physiology.CardiovascularHealth <= old.Physiology.CardiovascularHealth || old.Physiology.CardiovascularHealth <= 0.5 {
		t.Errorf("Exercise builds fitness, faster when young: %.3f vs %.3f", young.Physiology.CardiovascularHealth,
			old.Physiology.CardiovascularHealth)
	}
	before := young.Hormones.Serotonin
	young.ProcessExercise(ExercisePlay, 1)
	if young.Hormones.Serotonin <= before {
		t.Error("Exercise should lift serotonin")
	}
}

func TestOlderSavesGetHormonesAndRhythm(t *testing.T) {
	b := NewBiologicalSystems()
	b.Hormones, b.Circadian = nil, nil
	b.Update(0.05)
	if b.Hormones == nil || b.Circadian == nil {
		t.Error("Updating should create missing hormones and sleep cycle")
	}
}
