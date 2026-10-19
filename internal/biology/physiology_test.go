package biology

import "testing"

func TestDigestionMakesWaste(t *testing.T) {
	b := NewBiologicalSystems()
	if b.Eat(0.6) {
		t.Fatal("A normal meal should not overfill the stomach")
	}
	b.Update(1)
	ph := b.Physiology
	if ph.Stomach >= 0.6 || ph.Bowel <= 0 {
		t.Errorf("Digestion should move food to the bowel: stomach %f, bowel %f", ph.Stomach, ph.Bowel)
	}
}

func TestOvereatingUpsetsStomach(t *testing.T) {
	b := NewBiologicalSystems()
	b.Eat(0.9)
	if !b.Eat(0.3) {
		t.Fatal("Eating past the stomach's capacity should make the pet throw up")
	}
	if b.Physiology.DigestiveHealth >= 1 || b.Physiology.Stomach > stomachCapacity {
		t.Errorf("Throwing up should empty part of the stomach and upset digestion: %+v", *b.Physiology)
	}
	if b.Absorption() >= 1 {
		t.Error("An upset gut should absorb less of a meal")
	}
}

func TestFitnessNeedsExercise(t *testing.T) {
	b := NewBiologicalSystems()
	for i := 0; i < 30; i++ {
		b.Vitals.Nutrition, b.Vitals.Hydration, b.Vitals.Energy = 1, 1, 1
		b.Update(1)
	}
	fitness := b.Physiology.CardiovascularHealth
	if fitness > 0.5 {
		t.Errorf("Fitness should fade without exercise, got %f", fitness)
	}
	if b.EnergyDrain() <= 1 {
		t.Error("An unfit pet should tire faster")
	}
	b.Exercise(2)
	if b.Physiology.CardiovascularHealth <= fitness {
		t.Error("Exercise should restore fitness")
	}
}

func TestFoulAirHarmsLungs(t *testing.T) {
	b := NewBiologicalSystems()
	b.AirPollution = 1
	b.Update(1)
	if b.Physiology.RespiratoryHealth >= 1 {
		t.Error("Foul air should harm the airways")
	}
	harmed := b.Physiology.RespiratoryHealth
	b.AirPollution = 0
	b.Update(1)
	if b.Physiology.RespiratoryHealth <= harmed {
		t.Error("Fresh air should let the airways recover")
	}
}

func TestStressWearsNervousSystem(t *testing.T) {
	b := NewBiologicalSystems()
	b.Vitals.Stress = 1
	b.Update(1)
	if b.Physiology.NervousHealth >= 1 || b.StressSensitivity() <= 1 {
		t.Errorf("Long-term stress should wear the nervous system: %f", b.Physiology.NervousHealth)
	}
}

func TestFailingOrgansCostHealth(t *testing.T) {
	b := NewBiologicalSystems()
	b.Physiology.RespiratoryHealth = 0.1
	health := b.Vitals.Health
	b.Vitals.Happiness, b.Vitals.Stress = 0.5, 0.5 // Neither healing nor harming wellbeing
	b.Update(0.05)
	if b.Vitals.Health >= health {
		t.Error("A failing organ system should cost health")
	}
}

func TestOldSavesGetHealthyBody(t *testing.T) {
	b := NewBiologicalSystems()
	b.Physiology = nil
	b.Update(0.1)
	if b.Physiology == nil || b.Physiology.NervousHealth != 1 {
		t.Errorf("A pet saved before organ systems existed should get a healthy body: %+v", b.Physiology)
	}
}
