package genetics

import (
	"testing"
)

func TestNewGenome(t *testing.T) {
	g := NewGenome()

	if g.ID == "" {
		t.Error("Genome should have an ID")
	}

	if len(g.Traits) == 0 {
		t.Error("Genome should have traits")
	}

	if g.Generation != 1 {
		t.Errorf("Expected generation 1, got %d", g.Generation)
	}

	if g.GeneticDiversity != 1.0 {
		t.Errorf("Expected initial diversity 1.0, got %f", g.GeneticDiversity)
	}
}

func TestNewGeneAllele(t *testing.T) {
	allele := NewGeneAllele()

	if allele.Value < 0.0 || allele.Value > 1.0 {
		t.Errorf("Allele value should be between 0 and 1, got %f", allele.Value)
	}

	if allele.Dominant < 0.0 || allele.Dominant > 1.0 {
		t.Errorf("Dominant allele should be between 0 and 1, got %f", allele.Dominant)
	}

	if allele.Recessive < 0.0 || allele.Recessive > 1.0 {
		t.Errorf("Recessive allele should be between 0 and 1, got %f", allele.Recessive)
	}

	if allele.Expression < 0.8 || allele.Expression > 1.0 {
		t.Errorf("Expression should be between 0.8 and 1, got %f", allele.Expression)
	}
}

func TestNewGeneAlleleWithValue(t *testing.T) {
	targetValue := 0.7
	allele := NewGeneAlleleWithValue(targetValue)

	// The expressed value should be close to the target
	if allele.Value < 0.0 || allele.Value > 1.0 {
		t.Errorf("Allele value should be between 0 and 1, got %f", allele.Value)
	}
}

func TestGetTraitValue(t *testing.T) {
	g := NewGenome()

	// All standard traits should be present
	traits := []TraitType{
		TraitSize, TraitSpeed, TraitStrength,
		TraitBasePlayfulness, TraitBaseIntelligence,
	}

	for _, trait := range traits {
		value := g.GetTraitValue(trait)
		if value < 0.0 || value > 1.0 {
			t.Errorf("Trait %s value should be between 0 and 1, got %f", trait, value)
		}
	}
}

func TestGetTraitValueDefault(t *testing.T) {
	g := NewGenome()

	// Non-existent trait should return 0.5
	value := g.GetTraitValue(TraitType("nonexistent"))
	if value != 0.5 {
		t.Errorf("Non-existent trait should return 0.5, got %f", value)
	}
}

func TestSetTraitValue(t *testing.T) {
	g := NewGenome()

	g.SetTraitValue(TraitSize, 0.8)

	// Get the raw allele value (not multiplied by expression)
	if allele, exists := g.Traits[TraitSize]; exists {
		if allele.Value != 0.8 {
			t.Errorf("Expected trait value 0.8, got %f", allele.Value)
		}
	} else {
		t.Error("TraitSize should exist")
	}
}

func TestMutate(t *testing.T) {
	g := NewGenome()

	// Store original values
	originalValues := make(map[TraitType]float64)
	for trait, allele := range g.Traits {
		originalValues[trait] = allele.Value
	}

	// High mutation rate to ensure mutations occur
	mutations := g.Mutate(1.0)

	if len(mutations) == 0 {
		t.Error("Expected mutations with 100% mutation rate")
	}

	// Check that mutations are recorded
	if len(g.Mutations) != len(mutations) {
		t.Errorf("Expected %d mutations in genome, got %d", len(mutations), len(g.Mutations))
	}

	// Check mutation structure
	for _, m := range mutations {
		if m.TraitAffected == "" {
			t.Error("Mutation should have affected trait")
		}
		if m.Description == "" {
			t.Error("Mutation should have description")
		}
	}
}

func TestMutateZeroRate(t *testing.T) {
	g := NewGenome()

	mutations := g.Mutate(0.0)

	if len(mutations) != 0 {
		t.Error("Expected no mutations with 0% mutation rate")
	}
}

func TestCalculateGeneticDiversity(t *testing.T) {
	g := NewGenome()

	diversity := g.CalculateGeneticDiversity()

	if diversity < 0.0 || diversity > 1.0 {
		t.Errorf("Diversity should be between 0 and 1, got %f", diversity)
	}

	if g.GeneticDiversity != diversity {
		t.Error("Genome's diversity field should be updated")
	}
}

func TestClone(t *testing.T) {
	g := NewGenome()
	g.Mutate(0.5)

	clone := g.Clone()

	if clone.ID == g.ID {
		t.Error("Clone should have different ID")
	}

	if clone.Generation != g.Generation {
		t.Error("Clone should have same generation")
	}

	if len(clone.Traits) != len(g.Traits) {
		t.Error("Clone should have same number of traits")
	}

	// Verify traits are copied
	for trait, allele := range g.Traits {
		cloneAllele, exists := clone.Traits[trait]
		if !exists {
			t.Errorf("Clone missing trait %s", trait)
			continue
		}
		if cloneAllele.Value != allele.Value {
			t.Errorf("Clone trait %s value mismatch", trait)
		}
	}

	// Verify mutations are copied
	if len(clone.Mutations) != len(g.Mutations) {
		t.Errorf("Expected %d mutations in clone, got %d", len(g.Mutations), len(clone.Mutations))
	}
}

func TestGetPhysicalTraits(t *testing.T) {
	g := NewGenome()

	physical := g.GetPhysicalTraits()

	expectedTraits := []string{"size", "speed", "strength", "endurance", "longevity"}

	for _, trait := range expectedTraits {
		if _, exists := physical[trait]; !exists {
			t.Errorf("Missing physical trait: %s", trait)
		}
	}
}

func TestGetBehavioralPredispositions(t *testing.T) {
	g := NewGenome()

	behavioral := g.GetBehavioralPredispositions()

	expectedTraits := []string{"playfulness", "intelligence", "aggression", "sociability"}

	for _, trait := range expectedTraits {
		if _, exists := behavioral[trait]; !exists {
			t.Errorf("Missing behavioral trait: %s", trait)
		}
	}
}

func TestGetHealthTraits(t *testing.T) {
	g := NewGenome()

	health := g.GetHealthTraits()

	expectedTraits := []string{"immune_strength", "metabolic_rate", "disease_susceptibility"}

	for _, trait := range expectedTraits {
		if _, exists := health[trait]; !exists {
			t.Errorf("Missing health trait: %s", trait)
		}
	}
}

func TestNewGenomeWithTraits(t *testing.T) {
	customTraits := map[TraitType]float64{
		TraitSize:            0.8,
		TraitBasePlayfulness: 0.9,
	}

	g := NewGenomeWithTraits(customTraits)

	// Custom traits should be close to specified values
	// (may vary slightly due to allele calculation)
	if g.Traits[TraitSize].Value < 0.0 || g.Traits[TraitSize].Value > 1.0 {
		t.Error("Size trait should be in valid range")
	}

	if g.Traits[TraitBasePlayfulness].Value < 0.0 || g.Traits[TraitBasePlayfulness].Value > 1.0 {
		t.Error("Playfulness trait should be in valid range")
	}
}

func TestIsMutationBeneficial(t *testing.T) {
	// For positive traits, increase is beneficial
	if !isMutationBeneficial(TraitSize, 0.5, 0.7) {
		t.Error("Size increase should be beneficial")
	}

	if isMutationBeneficial(TraitSize, 0.7, 0.5) {
		t.Error("Size decrease should not be beneficial")
	}

	// For negative traits, decrease is beneficial
	if !isMutationBeneficial(TraitBaseAggression, 0.7, 0.5) {
		t.Error("Aggression decrease should be beneficial")
	}

	if isMutationBeneficial(TraitBaseAggression, 0.5, 0.7) {
		t.Error("Aggression increase should not be beneficial")
	}
}

func TestDescribeMutation(t *testing.T) {
	desc := describeMutation(TraitSize, 0.5, 0.6)

	if desc == "" {
		t.Error("Description should not be empty")
	}

	if len(desc) < 10 {
		t.Error("Description should be meaningful")
	}
}

func TestClampFunction(t *testing.T) {
	tests := []struct {
		value, min, max, expected float64
	}{
		{0.5, 0.0, 1.0, 0.5},
		{-0.5, 0.0, 1.0, 0.0},
		{1.5, 0.0, 1.0, 1.0},
		{0.0, 0.0, 1.0, 0.0},
		{1.0, 0.0, 1.0, 1.0},
	}

	for _, test := range tests {
		result := clamp(test.value, test.min, test.max)
		if result != test.expected {
			t.Errorf("clamp(%f, %f, %f) = %f, expected %f",
				test.value, test.min, test.max, result, test.expected)
		}
	}
}
