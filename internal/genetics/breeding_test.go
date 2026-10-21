package genetics

import (
	"testing"
)

func TestNewGeneticSystem(t *testing.T) {
	gs := NewGeneticSystem()

	if gs.MutationRate != 0.05 {
		t.Errorf("Expected mutation rate 0.05, got %f", gs.MutationRate)
	}

	if gs.CrossoverPoints != 3 {
		t.Errorf("Expected 3 crossover points, got %d", gs.CrossoverPoints)
	}

	if gs.BreedingHistory == nil {
		t.Error("Breeding history should be initialized")
	}
}

func TestDefaultBreedingRequirements(t *testing.T) {
	req := DefaultBreedingRequirements()

	if req.MinAge != 7.0 {
		t.Errorf("Expected MinAge 7.0, got %f", req.MinAge)
	}

	if req.MaxAge != 365.0 {
		t.Errorf("Expected MaxAge 365.0, got %f", req.MaxAge)
	}

	if req.MinHealth != 0.6 {
		t.Errorf("Expected MinHealth 0.6, got %f", req.MinHealth)
	}
}

func TestCalculateCompatibility(t *testing.T) {
	gs := NewGeneticSystem()

	genome1 := NewGenome()
	genome2 := NewGenome()

	compatibility := gs.CalculateCompatibility(genome1, genome2)

	if compatibility < 0.0 || compatibility > 1.0 {
		t.Errorf("Compatibility should be between 0 and 1, got %f", compatibility)
	}
}

func TestCalculateCompatibilityNilGenomes(t *testing.T) {
	gs := NewGeneticSystem()

	compatibility := gs.CalculateCompatibility(nil, nil)
	if compatibility != 0.0 {
		t.Errorf("Nil genomes should have 0 compatibility, got %f", compatibility)
	}

	genome := NewGenome()
	compatibility = gs.CalculateCompatibility(genome, nil)
	if compatibility != 0.0 {
		t.Errorf("One nil genome should have 0 compatibility, got %f", compatibility)
	}
}

func TestCalculateCompatibilitySameGenome(t *testing.T) {
	gs := NewGeneticSystem()

	genome := NewGenome()

	compatibility := gs.CalculateCompatibility(genome, genome)

	// Same genome should have high compatibility but not perfect
	// (too similar can reduce compatibility in our model)
	if compatibility < 0.0 || compatibility > 1.0 {
		t.Errorf("Self-compatibility should be between 0 and 1, got %f", compatibility)
	}
}

func TestCheckRequirementsNamesEachPet(t *testing.T) {
	gs := NewGeneticSystem()

	reason := gs.CheckRequirements(
		BreedingCandidate{Name: "Pip", Age: 2, Health: 0.4, Happiness: 0.9},
		BreedingCandidate{Name: "Biscuit", Age: 10, Health: 1, Happiness: 0.9},
	)
	want := "Pip is too young (2.0 days old; must be at least 7); Pip isn't healthy enough (health 40%; needs at least 60%)"
	if reason != want {
		t.Errorf("CheckRequirements = %q, want %q", reason, want)
	}

	if problem := gs.CompatibilityProblem(0.25); problem != "they aren't compatible enough (25% compatible; need at least 30%)" {
		t.Errorf("CompatibilityProblem = %q", problem)
	}
	if problem := gs.CompatibilityProblem(0.5); problem != "" {
		t.Errorf("A compatible pair should have no problem, got %q", problem)
	}
}

func TestCrossover(t *testing.T) {
	gs := NewGeneticSystem()

	parent1 := NewGenome()
	parent2 := NewGenome()

	offspring := gs.Crossover(parent1, parent2)

	if offspring == nil {
		t.Fatal("Offspring should not be nil")
	}

	if offspring.ID == parent1.ID || offspring.ID == parent2.ID {
		t.Error("Offspring should have unique ID")
	}

	if offspring.Generation != 2 {
		t.Errorf("Offspring generation should be 2, got %d", offspring.Generation)
	}

	if len(offspring.ParentIDs) != 2 {
		t.Errorf("Offspring should have 2 parent IDs, got %d", len(offspring.ParentIDs))
	}

	if offspring.ParentIDs[0] != parent1.ID || offspring.ParentIDs[1] != parent2.ID {
		t.Error("Offspring should reference correct parents")
	}

	// Offspring should have all traits
	if len(offspring.Traits) != len(parent1.Traits) {
		t.Errorf("Offspring should have same number of traits as parents")
	}
}

func TestBreed(t *testing.T) {
	gs := NewGeneticSystem()
	gs.Requirements.MinCompatibility = 0.0 // Allow all compatibility levels

	parent1 := NewGenome()
	parent2 := NewGenome()

	// Try multiple times as breeding has randomness
	successCount := 0
	attempts := 10

	for i := 0; i < attempts; i++ {
		result := gs.Breed(parent1, parent2)
		if result.Success {
			successCount++

			if result.OffspringGenome == nil {
				t.Error("Successful breeding should produce offspring")
			}

			if result.InheritedTraits == nil {
				t.Error("Result should have inherited traits map")
			}
		}
	}

	if successCount == 0 {
		t.Error("Expected at least one successful breeding in 10 attempts")
	}
}

func TestBreedLowCompatibility(t *testing.T) {
	gs := NewGeneticSystem()
	gs.Requirements.MinCompatibility = 0.99 // Very high requirement

	parent1 := NewGenome()
	parent2 := NewGenome()

	result := gs.Breed(parent1, parent2)

	// Most random genome pairs won't meet 99% compatibility
	// This test may occasionally pass if random genomes happen to be very similar
	if result.Success {
		// It's possible but unlikely
		t.Log("Warning: breeding succeeded despite high compatibility requirement")
	}
}

func TestBreedingHistoryGrows(t *testing.T) {
	gs := NewGeneticSystem()
	gs.Requirements.MinCompatibility = 0.0

	parent1 := NewGenome()
	parent2 := NewGenome()

	initialLen := len(gs.BreedingHistory)

	gs.Breed(parent1, parent2)
	gs.Breed(parent1, parent2)

	// Only successful breedings are recorded
	// But we should see some change
	if len(gs.BreedingHistory) <= initialLen {
		// This could happen if all breedings failed
		t.Log("No successful breedings recorded (may be due to randomness)")
	}
}

func TestMaxFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 2},
		{5, 3, 5},
		{4, 4, 4},
		{-1, 0, 0},
	}

	for _, test := range tests {
		result := max(test.a, test.b)
		if result != test.expected {
			t.Errorf("max(%d, %d) = %d, expected %d",
				test.a, test.b, result, test.expected)
		}
	}
}

func TestSeasonModifierScalesSuccess(t *testing.T) {
	successes := func(modifier float64) int {
		gs := NewGeneticSystem()
		gs.SeasonModifier = modifier
		count := 0
		for i := 0; i < 200; i++ {
			a, b := NewGenome(), NewGenome()
			if gs.CompatibilityProblem(gs.CalculateCompatibility(a, b)) != "" {
				continue
			}
			if gs.Breed(a, b).Success {
				count++
			}
		}
		return count
	}

	// At most 1% of attempts succeed with a tiny modifier; every
	// compatible attempt succeeds with a large one
	if n := successes(0.01); n > 20 {
		t.Errorf("A season modifier of 0.01 should make babies rare, got %d of 200", n)
	}
	if n := successes(0); n < 50 {
		t.Errorf("A season modifier of 0 should mean no change, got %d of 200", n)
	}
	gs := NewGeneticSystem()
	gs.SeasonModifier = 10
	for i := 0; i < 50; i++ {
		a, b := NewGenome(), NewGenome()
		if gs.CompatibilityProblem(gs.CalculateCompatibility(a, b)) != "" {
			continue
		}
		if result := gs.Breed(a, b); !result.Success {
			t.Fatalf("Success should be certain with a large modifier: %s", result.FailureReason)
		}
	}
}
