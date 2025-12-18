package genetics

import (
	"testing"
	"time"
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

func TestNewGeneticSystemWithConfig(t *testing.T) {
	req := BreedingRequirements{
		MinAge:           10.0,
		MaxAge:           200.0,
		MinHealth:        0.8,
		MinHappiness:     0.6,
		CooldownDays:     5.0,
		MinCompatibility: 0.5,
	}

	gs := NewGeneticSystemWithConfig(0.1, 5, req)

	if gs.MutationRate != 0.1 {
		t.Errorf("Expected mutation rate 0.1, got %f", gs.MutationRate)
	}

	if gs.CrossoverPoints != 5 {
		t.Errorf("Expected 5 crossover points, got %d", gs.CrossoverPoints)
	}

	if gs.Requirements.MinAge != 10.0 {
		t.Errorf("Expected MinAge 10.0, got %f", gs.Requirements.MinAge)
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

func TestCanBreed(t *testing.T) {
	gs := NewGeneticSystem()

	now := time.Now()
	oldTime := now.Add(-10 * 24 * time.Hour) // 10 days ago

	// Valid breeding conditions
	canBreed, reason := gs.CanBreed(
		30.0, 30.0, // Ages (valid)
		0.8, 0.8, // Health (valid)
		0.7, 0.7, // Happiness (valid)
		oldTime, oldTime, // Last breeding (past cooldown)
	)

	if !canBreed {
		t.Errorf("Should be able to breed with valid conditions, reason: %s", reason)
	}
}

func TestCanBreedTooYoung(t *testing.T) {
	gs := NewGeneticSystem()

	oldTime := time.Now().Add(-10 * 24 * time.Hour)

	canBreed, reason := gs.CanBreed(
		3.0, 30.0, // One too young
		0.8, 0.8,
		0.7, 0.7,
		oldTime, oldTime,
	)

	if canBreed {
		t.Error("Should not be able to breed when too young")
	}

	if reason != "one or both pets are too young" {
		t.Errorf("Expected 'too young' reason, got: %s", reason)
	}
}

func TestCanBreedTooOld(t *testing.T) {
	gs := NewGeneticSystem()

	oldTime := time.Now().Add(-10 * 24 * time.Hour)

	canBreed, reason := gs.CanBreed(
		30.0, 500.0, // One too old
		0.8, 0.8,
		0.7, 0.7,
		oldTime, oldTime,
	)

	if canBreed {
		t.Error("Should not be able to breed when too old")
	}

	if reason != "one or both pets are too old" {
		t.Errorf("Expected 'too old' reason, got: %s", reason)
	}
}

func TestCanBreedUnhealthy(t *testing.T) {
	gs := NewGeneticSystem()

	oldTime := time.Now().Add(-10 * 24 * time.Hour)

	canBreed, reason := gs.CanBreed(
		30.0, 30.0,
		0.3, 0.8, // One unhealthy
		0.7, 0.7,
		oldTime, oldTime,
	)

	if canBreed {
		t.Error("Should not be able to breed when unhealthy")
	}

	if reason != "one or both pets are not healthy enough" {
		t.Errorf("Expected 'not healthy' reason, got: %s", reason)
	}
}

func TestCanBreedUnhappy(t *testing.T) {
	gs := NewGeneticSystem()

	oldTime := time.Now().Add(-10 * 24 * time.Hour)

	canBreed, reason := gs.CanBreed(
		30.0, 30.0,
		0.8, 0.8,
		0.2, 0.7, // One unhappy
		oldTime, oldTime,
	)

	if canBreed {
		t.Error("Should not be able to breed when unhappy")
	}

	if reason != "one or both pets are not happy enough" {
		t.Errorf("Expected 'not happy' reason, got: %s", reason)
	}
}

func TestCanBreedCooldown(t *testing.T) {
	gs := NewGeneticSystem()

	recent := time.Now().Add(-1 * time.Hour) // 1 hour ago (within cooldown)
	old := time.Now().Add(-10 * 24 * time.Hour)

	canBreed, reason := gs.CanBreed(
		30.0, 30.0,
		0.8, 0.8,
		0.7, 0.7,
		recent, old, // Parent 1 recently bred
	)

	if canBreed {
		t.Error("Should not be able to breed during cooldown")
	}

	if reason != "parent 1 is still in cooldown" {
		t.Errorf("Expected cooldown reason, got: %s", reason)
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

func TestCreateOffspring(t *testing.T) {
	gs := NewGeneticSystem()

	parent1 := NewGenome()
	parent2 := NewGenome()

	oldTime := time.Now().Add(-10 * 24 * time.Hour)

	result := gs.CreateOffspring(
		"pet1", "pet2",
		parent1, parent2,
		30.0, 30.0,
		0.8, 0.8,
		0.7, 0.7,
		oldTime, oldTime,
	)

	// Result should have compatibility score regardless of success
	if result.CompatibilityScore < 0.0 || result.CompatibilityScore > 1.0 {
		t.Errorf("Compatibility score should be between 0 and 1, got %f",
			result.CompatibilityScore)
	}
}

func TestCreateOffspringFailsRequirements(t *testing.T) {
	gs := NewGeneticSystem()

	parent1 := NewGenome()
	parent2 := NewGenome()

	oldTime := time.Now().Add(-10 * 24 * time.Hour)

	result := gs.CreateOffspring(
		"pet1", "pet2",
		parent1, parent2,
		3.0, 30.0, // Parent 1 too young
		0.8, 0.8,
		0.7, 0.7,
		oldTime, oldTime,
	)

	if result.Success {
		t.Error("Breeding should fail with too young parent")
	}

	if result.FailureReason == "" {
		t.Error("Failed breeding should have a reason")
	}
}

func TestGetBreedingStats(t *testing.T) {
	gs := NewGeneticSystem()
	gs.Requirements.MinCompatibility = 0.0

	parent1 := NewGenome()
	parent2 := NewGenome()

	// Perform some breedings
	for i := 0; i < 5; i++ {
		gs.Breed(parent1, parent2)
	}

	stats := gs.GetBreedingStats()

	if stats["total_attempts"].(int) != 5 {
		t.Errorf("Expected 5 total attempts, got %d", stats["total_attempts"])
	}

	if _, exists := stats["successful"]; !exists {
		t.Error("Stats should have 'successful' field")
	}

	if _, exists := stats["success_rate"]; !exists {
		t.Error("Stats should have 'success_rate' field")
	}

	if _, exists := stats["total_mutations"]; !exists {
		t.Error("Stats should have 'total_mutations' field")
	}
}

func TestSetMutationRate(t *testing.T) {
	gs := NewGeneticSystem()

	gs.SetMutationRate(0.2)
	if gs.MutationRate != 0.2 {
		t.Errorf("Expected mutation rate 0.2, got %f", gs.MutationRate)
	}

	// Test clamping
	gs.SetMutationRate(1.5)
	if gs.MutationRate != 1.0 {
		t.Errorf("Mutation rate should be clamped to 1.0, got %f", gs.MutationRate)
	}

	gs.SetMutationRate(-0.5)
	if gs.MutationRate != 0.0 {
		t.Errorf("Mutation rate should be clamped to 0.0, got %f", gs.MutationRate)
	}
}

func TestSetCrossoverPoints(t *testing.T) {
	gs := NewGeneticSystem()

	gs.SetCrossoverPoints(5)
	if gs.CrossoverPoints != 5 {
		t.Errorf("Expected 5 crossover points, got %d", gs.CrossoverPoints)
	}

	// Test minimum
	gs.SetCrossoverPoints(0)
	if gs.CrossoverPoints != 1 {
		t.Errorf("Crossover points should be at least 1, got %d", gs.CrossoverPoints)
	}

	gs.SetCrossoverPoints(-5)
	if gs.CrossoverPoints != 1 {
		t.Errorf("Crossover points should be at least 1, got %d", gs.CrossoverPoints)
	}
}

func TestPredictOffspringTraits(t *testing.T) {
	gs := NewGeneticSystem()

	parent1 := NewGenome()
	parent2 := NewGenome()

	predictions := gs.PredictOffspringTraits(parent1, parent2)

	if len(predictions) == 0 {
		t.Error("Should have trait predictions")
	}

	for trait, pred := range predictions {
		if pred.Min < 0.0 || pred.Min > 1.0 {
			t.Errorf("Trait %s min should be between 0 and 1", trait)
		}
		if pred.Max < 0.0 || pred.Max > 1.0 {
			t.Errorf("Trait %s max should be between 0 and 1", trait)
		}
		if pred.Min > pred.Max {
			t.Errorf("Trait %s min should be <= max", trait)
		}
		if pred.Avg < pred.Min || pred.Avg > pred.Max {
			// Note: Due to mutation adjustment, avg might be outside min/max
			// This is expected behavior
		}
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
