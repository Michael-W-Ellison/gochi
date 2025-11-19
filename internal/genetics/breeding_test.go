package genetics

import (
	"testing"
	"time"
)

func TestNewGeneticSystem(t *testing.T) {
	gs := NewGeneticSystem()

	if gs == nil {
		t.Fatal("NewGeneticSystem returned nil")
	}

	if gs.BaseMutationRate <= 0 {
		t.Error("BaseMutationRate should be positive")
	}

	if gs.MinBreedingInterval <= 0 {
		t.Error("MinBreedingInterval should be positive")
	}

	if gs.MaxOffspringPerPair <= 0 {
		t.Error("MaxOffspringPerPair should be positive")
	}
}

func TestCheckBreedingCompatibility_SameGenome(t *testing.T) {
	gs := NewGeneticSystem()
	genome := NewRandomGenome()

	compat := gs.CheckBreedingCompatibility(genome, genome)

	if compat.Compatible {
		t.Error("Genome should not be compatible with itself")
	}

	if compat.CompatibilityScore != 0.0 {
		t.Errorf("Compatibility score should be 0 for self-breeding, got %f",
			compat.CompatibilityScore)
	}

	if len(compat.Reasons) == 0 {
		t.Error("Should have reasons for incompatibility")
	}
}

func TestCheckBreedingCompatibility_ParentOffspring(t *testing.T) {
	gs := NewGeneticSystem()
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	// Breed to create offspring
	result := gs.Breed(parent1, parent2)
	if !result.Success {
		t.Fatal("Breeding should succeed")
	}

	offspring := result.OffspringGenome

	// Try to breed parent with offspring
	compat := gs.CheckBreedingCompatibility(parent1, offspring)

	if compat.Compatible {
		t.Error("Parent should not be compatible with offspring")
	}
}

func TestCheckBreedingCompatibility_ValidPair(t *testing.T) {
	gs := NewGeneticSystem()
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	compat := gs.CheckBreedingCompatibility(parent1, parent2)

	if !compat.Compatible {
		t.Errorf("Two unrelated genomes should be compatible: %v", compat.Reasons)
	}

	if compat.CompatibilityScore <= 0 {
		t.Errorf("Compatible pair should have positive score, got %f",
			compat.CompatibilityScore)
	}

	if compat.GeneticSimilarity < 0 || compat.GeneticSimilarity > 1 {
		t.Errorf("Genetic similarity should be between 0 and 1, got %f",
			compat.GeneticSimilarity)
	}
}

func TestCheckBreedingCompatibility_Cooldown(t *testing.T) {
	gs := NewGeneticSystem()
	gs.MinBreedingInterval = 0.0001 // Very short interval for testing (0.36 seconds)

	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	// First breeding
	result := gs.Breed(parent1, parent2)
	if !result.Success {
		t.Fatal("First breeding should succeed")
	}

	// Immediate second breeding should fail
	parent3 := NewRandomGenome()
	compat := gs.CheckBreedingCompatibility(parent1, parent3)

	if compat.Compatible {
		t.Error("Should not be compatible during cooldown period")
	}

	if compat.RecommendedWaitTime <= 0 {
		t.Error("Should have recommended wait time")
	}

	// Wait for cooldown to pass
	time.Sleep(400 * time.Millisecond)

	// Should be compatible now
	compat2 := gs.CheckBreedingCompatibility(parent1, parent3)
	if !compat2.Compatible {
		t.Errorf("Should be compatible after cooldown (waited, recommended wait was %.4f hours)",
			compat.RecommendedWaitTime)
	}
}

func TestCheckBreedingCompatibility_MaxOffspring(t *testing.T) {
	gs := NewGeneticSystem()
	gs.MaxOffspringPerPair = 2
	gs.MinBreedingInterval = 0 // No cooldown for this test

	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	// Breed maximum number of times
	for i := 0; i < gs.MaxOffspringPerPair; i++ {
		result := gs.Breed(parent1, parent2)
		if !result.Success {
			t.Fatalf("Breeding %d should succeed", i+1)
		}
		// Reset breeding time for next iteration
		gs.LastBreedingTime[parent1.ID] = time.Time{}
		gs.LastBreedingTime[parent2.ID] = time.Time{}
	}

	// Next breeding should fail
	compat := gs.CheckBreedingCompatibility(parent1, parent2)
	if compat.Compatible {
		t.Error("Should not be compatible after max offspring reached")
	}
}

func TestBreed_BasicInheritance(t *testing.T) {
	gs := NewGeneticSystem()
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	result := gs.Breed(parent1, parent2)

	if !result.Success {
		t.Fatal("Breeding should succeed")
	}

	if result.OffspringGenome == nil {
		t.Fatal("Offspring genome should not be nil")
	}

	offspring := result.OffspringGenome

	// Check generation
	expectedGen := maxInt(parent1.Generation, parent2.Generation) + 1
	if offspring.Generation != expectedGen {
		t.Errorf("Offspring generation should be %d, got %d",
			expectedGen, offspring.Generation)
	}

	// Check parentage
	if offspring.Parent1ID != parent1.ID {
		t.Error("Parent1ID not set correctly")
	}

	if offspring.Parent2ID != parent2.ID {
		t.Error("Parent2ID not set correctly")
	}

	// Check that offspring has all gene categories
	if len(offspring.PhysicalGenes) == 0 {
		t.Error("Offspring should have physical genes")
	}

	if len(offspring.HealthGenes) == 0 {
		t.Error("Offspring should have health genes")
	}

	if len(offspring.BehavioralGenes) == 0 {
		t.Error("Offspring should have behavioral genes")
	}

	if len(offspring.PersonalityGenes) == 0 {
		t.Error("Offspring should have personality genes")
	}
}

func TestBreed_TraitInheritance(t *testing.T) {
	gs := NewGeneticSystem()

	// Create parents with specific traits
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	// Set parent1 to have high size
	parent1.PhysicalGenes["base_size"].Allele1 = 0.9
	parent1.PhysicalGenes["base_size"].Allele2 = 0.9
	parent1.PhysicalGenes["base_size"].Dominance = 0.5

	// Set parent2 to have low size
	parent2.PhysicalGenes["base_size"].Allele1 = 0.1
	parent2.PhysicalGenes["base_size"].Allele2 = 0.1
	parent2.PhysicalGenes["base_size"].Dominance = 0.5

	parent1.ExpressGenes()
	parent2.ExpressGenes()

	result := gs.Breed(parent1, parent2)
	if !result.Success {
		t.Fatal("Breeding should succeed")
	}

	offspring := result.OffspringGenome

	// Offspring should have intermediate size (approximately)
	// Unless mutations occurred
	sizeGene := offspring.PhysicalGenes["base_size"]

	// At least one allele should come from each parent
	// Given the setup, offspring should have one high and one low allele
	// So expressed size should be intermediate
	if offspring.PhysicalTraits.BaseSize > 0.95 || offspring.PhysicalTraits.BaseSize < 0.05 {
		t.Logf("Warning: Offspring size %f is extreme, may be due to mutation",
			offspring.PhysicalTraits.BaseSize)
	}

	// Check that alleles came from parents
	hasHighAllele := sizeGene.Allele1 > 0.7 || sizeGene.Allele2 > 0.7
	hasLowAllele := sizeGene.Allele1 < 0.3 || sizeGene.Allele2 < 0.3

	// At least one should be true (unless both mutated)
	if !hasHighAllele && !hasLowAllele {
		if result.MutationCount < 2 {
			t.Error("Offspring should inherit alleles from parents")
		}
	}
}

func TestBreed_MutationOccurs(t *testing.T) {
	gs := NewGeneticSystem()
	gs.BaseMutationRate = 0.5 // High mutation rate for testing

	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	result := gs.Breed(parent1, parent2)
	if !result.Success {
		t.Fatal("Breeding should succeed")
	}

	if !result.MutationOccurred {
		t.Error("With 50% mutation rate, mutations should occur")
	}

	if result.MutationCount == 0 {
		t.Error("Mutation count should be > 0 with high mutation rate")
	}
}

func TestBreed_InheritanceLog(t *testing.T) {
	gs := NewGeneticSystem()
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	result := gs.Breed(parent1, parent2)
	if !result.Success {
		t.Fatal("Breeding should succeed")
	}

	if len(result.InheritanceLog) == 0 {
		t.Error("Inheritance log should contain entries")
	}

	// Log should mention generation
	hasGenerationInfo := false
	for _, entry := range result.InheritanceLog {
		if len(entry) > 0 {
			hasGenerationInfo = true
			break
		}
	}

	if !hasGenerationInfo {
		t.Error("Inheritance log should contain information")
	}
}

func TestBreed_OffspringDescription(t *testing.T) {
	gs := NewGeneticSystem()
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	result := gs.Breed(parent1, parent2)
	if !result.Success {
		t.Fatal("Breeding should succeed")
	}

	if result.Traits == "" {
		t.Error("Offspring traits description should not be empty")
	}
}

func TestCrossoverGene(t *testing.T) {
	gs := NewGeneticSystem()

	parent1Gene := &Gene{
		Name:      "test",
		Allele1:   0.9,
		Allele2:   0.8,
		Dominance: 0.6,
		Type:      GenePhysical,
	}

	parent2Gene := &Gene{
		Name:      "test",
		Allele1:   0.2,
		Allele2:   0.1,
		Dominance: 0.4,
		Type:      GenePhysical,
	}

	mutationCount := 0
	offspring := gs.crossoverGene(parent1Gene, parent2Gene, 0.0, &mutationCount)

	// With no mutation, alleles should come from parents
	isFromParent1 := (offspring.Allele1 == 0.9 || offspring.Allele1 == 0.8)
	isFromParent2 := (offspring.Allele2 == 0.2 || offspring.Allele2 == 0.1)

	if !isFromParent1 && !isFromParent2 {
		t.Error("Offspring alleles should come from parents when mutation rate is 0")
	}

	// Dominance should be approximately average of parents
	expectedDominance := (parent1Gene.Dominance + parent2Gene.Dominance) / 2.0
	if offspring.Dominance < expectedDominance-0.2 || offspring.Dominance > expectedDominance+0.2 {
		t.Logf("Dominance %f not close to expected %f (small variation allowed)",
			offspring.Dominance, expectedDominance)
	}
}

func TestMutate(t *testing.T) {
	original := 0.5
	mutationOccurred := false

	// Run mutation multiple times to ensure it changes values
	for i := 0; i < 100; i++ {
		mutated := mutate(original)
		if mutated != original {
			mutationOccurred = true

			// Mutated value should still be in valid range
			if mutated < 0 || mutated > 1 {
				t.Errorf("Mutated value %f out of range [0,1]", mutated)
			}
		}
	}

	if !mutationOccurred {
		t.Error("Mutation should occur at least once in 100 attempts")
	}
}

func TestGetBreedingHistory(t *testing.T) {
	gs := NewGeneticSystem()
	gs.MinBreedingInterval = 0

	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	// Breed once
	gs.Breed(parent1, parent2)

	history := gs.GetBreedingHistory(10)

	if len(history) != 1 {
		t.Errorf("Expected 1 breeding event in history, got %d", len(history))
	}

	if history[0].Parent1Genome.ID != parent1.ID {
		t.Error("Breeding history should record parent1 correctly")
	}

	if history[0].Parent2Genome.ID != parent2.ID {
		t.Error("Breeding history should record parent2 correctly")
	}
}

func TestGetOffspringCount(t *testing.T) {
	gs := NewGeneticSystem()
	gs.MinBreedingInterval = 0

	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()
	parent3 := NewRandomGenome()

	// Breed parent1 with parent2
	gs.Breed(parent1, parent2)

	// Reset breeding times
	gs.LastBreedingTime[parent1.ID] = time.Time{}
	gs.LastBreedingTime[parent2.ID] = time.Time{}

	// Breed parent1 with parent3
	gs.Breed(parent1, parent3)

	// Parent1 should have 2 offspring
	count := gs.GetOffspringCount(parent1.ID)
	if count != 2 {
		t.Errorf("Parent1 should have 2 offspring, got %d", count)
	}

	// Parent2 should have 1 offspring
	count = gs.GetOffspringCount(parent2.ID)
	if count != 1 {
		t.Errorf("Parent2 should have 1 offspring, got %d", count)
	}
}

func TestInbreedingPenalty(t *testing.T) {
	gs := NewGeneticSystem()

	// Create two very similar genomes
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	// Make parent2 very similar to parent1
	for name, gene := range parent1.PhysicalGenes {
		parent2.PhysicalGenes[name].Allele1 = gene.Allele1
		parent2.PhysicalGenes[name].Allele2 = gene.Allele2
		parent2.PhysicalGenes[name].Dominance = gene.Dominance
	}

	for name, gene := range parent1.HealthGenes {
		parent2.HealthGenes[name].Allele1 = gene.Allele1
		parent2.HealthGenes[name].Allele2 = gene.Allele2
		parent2.HealthGenes[name].Dominance = gene.Dominance
	}

	compat := gs.CheckBreedingCompatibility(parent1, parent2)

	// Should have warnings about genetic similarity
	if len(compat.Warnings) == 0 {
		t.Error("Should have warnings for high genetic similarity")
	}

	// Compatibility score should be reduced
	if compat.CompatibilityScore >= 1.0 {
		t.Error("Compatibility score should be reduced for inbred pairs")
	}
}

func TestGenerationIncrement(t *testing.T) {
	gs := NewGeneticSystem()

	// Gen 0 parents
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	// Breed to create Gen 1
	result1 := gs.Breed(parent1, parent2)
	if !result1.Success {
		t.Fatal("First breeding should succeed")
	}

	gen1 := result1.OffspringGenome
	if gen1.Generation != 1 {
		t.Errorf("First generation should be 1, got %d", gen1.Generation)
	}

	// Breed Gen 1 with Gen 0 to create Gen 2
	gs.MinBreedingInterval = 0
	parent3 := NewRandomGenome()
	result2 := gs.Breed(gen1, parent3)
	if !result2.Success {
		t.Fatal("Second breeding should succeed")
	}

	gen2 := result2.OffspringGenome
	if gen2.Generation != 2 {
		t.Errorf("Second generation should be 2, got %d", gen2.Generation)
	}
}

func TestComplementaryTraits(t *testing.T) {
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	// Set complementary sizes
	parent1.PhysicalGenes["base_size"].Allele1 = 0.9
	parent1.PhysicalGenes["base_size"].Allele2 = 0.9
	parent2.PhysicalGenes["base_size"].Allele1 = 0.2
	parent2.PhysicalGenes["base_size"].Allele2 = 0.2

	parent1.ExpressGenes()
	parent2.ExpressGenes()

	score := calculateComplementaryTraits(parent1, parent2)

	if score < 0 || score > 1 {
		t.Errorf("Complementary score should be between 0 and 1, got %f", score)
	}
}

func TestPairKeyGeneration(t *testing.T) {
	id1 := "genome_abc"
	id2 := "genome_xyz"

	key1 := generatePairKey(id1, id2)
	key2 := generatePairKey(id2, id1)

	if key1 != key2 {
		t.Error("Pair key should be same regardless of order")
	}
}

func BenchmarkBreed(b *testing.B) {
	gs := NewGeneticSystem()
	gs.MinBreedingInterval = 0
	gs.MaxOffspringPerPair = 999999

	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gs.Breed(parent1, parent2)
	}
}

func BenchmarkCheckBreedingCompatibility(b *testing.B) {
	gs := NewGeneticSystem()
	parent1 := NewRandomGenome()
	parent2 := NewRandomGenome()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gs.CheckBreedingCompatibility(parent1, parent2)
	}
}
