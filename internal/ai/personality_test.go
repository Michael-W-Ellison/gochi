package ai

import (
	"testing"
)

func TestNewBalancedTraits(t *testing.T) {
	traits := NewBalancedTraits()

	if traits.Openness != 0.5 {
		t.Errorf("Expected Openness to be 0.5, got %.2f", traits.Openness)
	}

	if traits.Loyalty < 0.5 {
		t.Errorf("Expected Loyalty to be >= 0.5, got %.2f", traits.Loyalty)
	}
}

func TestTraitsClamp(t *testing.T) {
	traits := &Traits{
		Openness:     1.5,
		Intelligence: -0.5,
		Playfulness:  0.5,
	}

	traits.Clamp()

	if traits.Openness != 1.0 {
		t.Errorf("Expected Openness to be clamped to 1.0, got %.2f", traits.Openness)
	}

	if traits.Intelligence != 0.0 {
		t.Errorf("Expected Intelligence to be clamped to 0.0, got %.2f", traits.Intelligence)
	}
}

func TestNewPersonalityMatrix(t *testing.T) {
	pm := NewPersonalityMatrix()

	if pm.Traits == nil {
		t.Error("Traits should be initialized")
	}

	if pm.EvolutionRate != 0.01 {
		t.Errorf("Expected EvolutionRate to be 0.01, got %.2f", pm.EvolutionRate)
	}
}

func TestEvolveTraits(t *testing.T) {
	pm := NewPersonalityMatrix()
	initialOpenness := pm.Traits.Openness

	experiences := []ExperienceData{
		{
			ExperienceType: "positive",
			Intensity:      1.0,
			TraitAffected:  "openness",
			Duration:       1.0,
		},
	}

	pm.EvolveTraits(experiences)

	if pm.Traits.Openness <= initialOpenness {
		t.Error("Openness should increase after positive experience")
	}

	if pm.ExperienceCount != 1 {
		t.Errorf("Expected ExperienceCount to be 1, got %d", pm.ExperienceCount)
	}
}

func TestGetTraitInfluence(t *testing.T) {
	pm := NewPersonalityMatrix()
	pm.Traits.Playfulness = 0.9
	pm.Traits.EnergyLevel = 0.8
	pm.Traits.Extraversion = 0.7

	influence := pm.GetTraitInfluence("play")

	if influence < 0.7 {
		t.Errorf("Expected high play influence, got %.2f", influence)
	}
}

func TestInheritTraits(t *testing.T) {
	parent1 := NewPersonalityMatrix()
	parent1.Traits.Intelligence = 0.9

	parent2 := NewPersonalityMatrix()
	parent2.Traits.Intelligence = 0.8

	child := parent1.InheritTraits(parent1, parent2, 0.05)

	// Child intelligence should be close to one of the parents
	if child.Intelligence < 0.6 || child.Intelligence > 1.0 {
		t.Errorf("Child intelligence should be inherited from parents, got %.2f", child.Intelligence)
	}
}

func TestTakeSnapshot(t *testing.T) {
	pm := NewPersonalityMatrix()

	pm.TakeSnapshot(1.0)
	pm.TakeSnapshot(2.0)

	if len(pm.TraitHistory) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(pm.TraitHistory))
	}
}

func TestGetPersonalityDescription(t *testing.T) {
	pm := NewPersonalityMatrix()
	pm.Traits.Playfulness = 0.9
	pm.Traits.Affectionate = 0.8

	description := pm.GetPersonalityDescription()

	if description == "" {
		t.Error("Description should not be empty")
	}

	if len(description) < 10 {
		t.Error("Description should be meaningful")
	}
}
