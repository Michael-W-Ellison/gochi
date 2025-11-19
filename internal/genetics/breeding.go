package genetics

import (
	"fmt"
	mathrand "math/rand"
	"sync"
	"time"
)

// BreedingCompatibility represents how compatible two pets are for breeding
type BreedingCompatibility struct {
	Compatible          bool
	CompatibilityScore  float64 // 0-1, higher is better
	GeneticSimilarity   float64 // 0-1, too high can be problematic
	Reasons             []string
	Warnings            []string
	RecommendedWaitTime float64 // Hours before breeding again
}

// BreedingPair represents two pets attempting to breed
type BreedingPair struct {
	Parent1Genome *Genome
	Parent2Genome *Genome
	BreedingTime  time.Time
}

// OffspringResult contains the outcome of breeding
type OffspringResult struct {
	Success        bool
	OffspringGenome *Genome
	MutationOccurred bool
	MutationCount   int
	InheritanceLog  []string
	Traits         string // Description of offspring traits
}

// GeneticSystem manages breeding and genetic operations
type GeneticSystem struct {
	mu sync.RWMutex

	// Configuration
	BaseMutationRate      float64 // Base chance of mutation per gene
	InbreedingPenalty     float64 // Reduced compatibility for similar genomes
	MinBreedingInterval   float64 // Hours between breeding attempts
	MaxOffspringPerPair   int     // Lifetime limit

	// Breeding history
	BreedingHistory       []BreedingPair
	OffspringCount        map[string]int // Count per genome ID
	LastBreedingTime      map[string]time.Time
}

// NewGeneticSystem creates a new genetics management system
func NewGeneticSystem() *GeneticSystem {
	return &GeneticSystem{
		BaseMutationRate:    0.05, // 5% mutation rate per gene
		InbreedingPenalty:   0.3,  // 30% penalty for similar genetics
		MinBreedingInterval: 48.0, // 48 hours between breeding
		MaxOffspringPerPair: 5,    // Maximum 5 offspring per pair
		BreedingHistory:     make([]BreedingPair, 0),
		OffspringCount:      make(map[string]int),
		LastBreedingTime:    make(map[string]time.Time),
	}
}

// CheckBreedingCompatibility determines if two pets can breed
func (gs *GeneticSystem) CheckBreedingCompatibility(parent1, parent2 *Genome) *BreedingCompatibility {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	compat := &BreedingCompatibility{
		Compatible:         true,
		CompatibilityScore: 1.0,
		Reasons:            make([]string, 0),
		Warnings:           make([]string, 0),
	}

	// Cannot breed with self
	if parent1.ID == parent2.ID {
		compat.Compatible = false
		compat.CompatibilityScore = 0.0
		compat.Reasons = append(compat.Reasons, "Cannot breed with self")
		return compat
	}

	// Cannot breed parent with offspring
	if parent1.Parent1ID == parent2.ID || parent1.Parent2ID == parent2.ID ||
		parent2.Parent1ID == parent1.ID || parent2.Parent2ID == parent1.ID {
		compat.Compatible = false
		compat.CompatibilityScore = 0.0
		compat.Reasons = append(compat.Reasons, "Cannot breed parent with direct offspring")
		return compat
	}

	// Check breeding cooldown
	if lastBreed, exists := gs.LastBreedingTime[parent1.ID]; exists {
		hoursSince := time.Since(lastBreed).Hours()
		if hoursSince < gs.MinBreedingInterval {
			compat.Compatible = false
			compat.RecommendedWaitTime = gs.MinBreedingInterval - hoursSince
			compat.Reasons = append(compat.Reasons,
				fmt.Sprintf("Parent 1 needs %.1f more hours before breeding again",
					gs.MinBreedingInterval-hoursSince))
			return compat
		}
	}

	if lastBreed, exists := gs.LastBreedingTime[parent2.ID]; exists {
		hoursSince := time.Since(lastBreed).Hours()
		if hoursSince < gs.MinBreedingInterval {
			compat.Compatible = false
			compat.RecommendedWaitTime = gs.MinBreedingInterval - hoursSince
			compat.Reasons = append(compat.Reasons,
				fmt.Sprintf("Parent 2 needs %.1f more hours before breeding again",
					gs.MinBreedingInterval-hoursSince))
			return compat
		}
	}

	// Check offspring limits
	pairKey := generatePairKey(parent1.ID, parent2.ID)
	if count, exists := gs.OffspringCount[pairKey]; exists && count >= gs.MaxOffspringPerPair {
		compat.Compatible = false
		compat.Reasons = append(compat.Reasons,
			fmt.Sprintf("This pair has already produced maximum offspring (%d)", gs.MaxOffspringPerPair))
		return compat
	}

	// Calculate genetic similarity
	similarity := parent1.CalculateGeneticSimilarity(parent2)
	compat.GeneticSimilarity = similarity

	// High genetic similarity is a concern (inbreeding)
	if similarity > 0.9 {
		compat.CompatibilityScore *= (1.0 - gs.InbreedingPenalty)
		compat.Warnings = append(compat.Warnings,
			"Very high genetic similarity - offspring may have reduced fitness")
	} else if similarity > 0.8 {
		compat.CompatibilityScore *= (1.0 - gs.InbreedingPenalty*0.5)
		compat.Warnings = append(compat.Warnings,
			"High genetic similarity - some inbreeding concerns")
	}

	// Optimal compatibility is in the sweet spot: not too similar, not too different
	if similarity >= 0.3 && similarity <= 0.7 {
		compat.CompatibilityScore *= 1.1 // Bonus for good genetic diversity
		compat.Reasons = append(compat.Reasons, "Good genetic diversity for healthy offspring")
	}

	// Check health traits - warn if both parents have poor health genes
	parent1Health := parent1.HealthTraits.ImmuneStrength
	parent2Health := parent2.HealthTraits.ImmuneStrength
	if parent1Health < 0.3 && parent2Health < 0.3 {
		compat.Warnings = append(compat.Warnings,
			"Both parents have weak immune systems - offspring may inherit this")
	}

	// Bonus for complementary traits
	complementaryScore := calculateComplementaryTraits(parent1, parent2)
	compat.CompatibilityScore *= (1.0 + complementaryScore*0.1)

	if complementaryScore > 0.5 {
		compat.Reasons = append(compat.Reasons, "Parents have complementary traits")
	}

	// Clamp final score
	if compat.CompatibilityScore > 1.0 {
		compat.CompatibilityScore = 1.0
	}

	return compat
}

// Breed creates offspring from two parent genomes
func (gs *GeneticSystem) Breed(parent1, parent2 *Genome) *OffspringResult {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	result := &OffspringResult{
		Success:          false,
		InheritanceLog:   make([]string, 0),
		MutationOccurred: false,
	}

	// Check compatibility (without lock since we're already locked)
	gs.mu.Unlock()
	compat := gs.CheckBreedingCompatibility(parent1, parent2)
	gs.mu.Lock()

	if !compat.Compatible {
		result.InheritanceLog = append(result.InheritanceLog,
			"Breeding failed: "+compat.Reasons[0])
		return result
	}

	// Create offspring genome
	offspring := &Genome{
		ID:               generateGenomeID(),
		Generation:       maxInt(parent1.Generation, parent2.Generation) + 1,
		Parent1ID:        parent1.ID,
		Parent2ID:        parent2.ID,
		MutationCount:    0,
		PhysicalGenes:    make(map[string]*Gene),
		HealthGenes:      make(map[string]*Gene),
		BehavioralGenes:  make(map[string]*Gene),
		PersonalityGenes: make(map[string]*Gene),
	}

	result.InheritanceLog = append(result.InheritanceLog,
		fmt.Sprintf("Generation %d offspring from parents (gen %d and %d)",
			offspring.Generation, parent1.Generation, parent2.Generation))

	// Perform genetic crossover for each gene category
	mutationRate := gs.BaseMutationRate

	// Increase mutation rate for inbred pairs
	if compat.GeneticSimilarity > 0.8 {
		mutationRate *= 1.5
		result.InheritanceLog = append(result.InheritanceLog,
			"Increased mutation rate due to genetic similarity")
	}

	// Crossover physical genes
	for name := range parent1.PhysicalGenes {
		offspring.PhysicalGenes[name] = gs.crossoverGene(
			parent1.PhysicalGenes[name],
			parent2.PhysicalGenes[name],
			mutationRate,
			&result.MutationCount,
		)
	}

	// Crossover health genes
	for name := range parent1.HealthGenes {
		offspring.HealthGenes[name] = gs.crossoverGene(
			parent1.HealthGenes[name],
			parent2.HealthGenes[name],
			mutationRate,
			&result.MutationCount,
		)
	}

	// Crossover behavioral genes
	for name := range parent1.BehavioralGenes {
		offspring.BehavioralGenes[name] = gs.crossoverGene(
			parent1.BehavioralGenes[name],
			parent2.BehavioralGenes[name],
			mutationRate,
			&result.MutationCount,
		)
	}

	// Crossover personality genes
	for name := range parent1.PersonalityGenes {
		offspring.PersonalityGenes[name] = gs.crossoverGene(
			parent1.PersonalityGenes[name],
			parent2.PersonalityGenes[name],
			mutationRate,
			&result.MutationCount,
		)
	}

	// Express all genes
	offspring.ExpressGenes()

	// Update breeding records
	pairKey := generatePairKey(parent1.ID, parent2.ID)
	gs.OffspringCount[pairKey]++
	gs.LastBreedingTime[parent1.ID] = time.Now()
	gs.LastBreedingTime[parent2.ID] = time.Now()
	gs.BreedingHistory = append(gs.BreedingHistory, BreedingPair{
		Parent1Genome: parent1,
		Parent2Genome: parent2,
		BreedingTime:  time.Now(),
	})

	result.Success = true
	result.OffspringGenome = offspring
	result.MutationOccurred = result.MutationCount > 0

	if result.MutationCount > 0 {
		result.InheritanceLog = append(result.InheritanceLog,
			fmt.Sprintf("%d mutations occurred during inheritance", result.MutationCount))
	}

	result.Traits = generateOffspringDescription(offspring)
	result.InheritanceLog = append(result.InheritanceLog, result.Traits)

	return result
}

// crossoverGene performs Mendelian inheritance with possible mutation
func (gs *GeneticSystem) crossoverGene(parent1Gene, parent2Gene *Gene, mutationRate float64, mutationCount *int) *Gene {
	newGene := &Gene{
		Name: parent1Gene.Name,
		Type: parent1Gene.Type,
	}

	// Randomly select one allele from each parent (Mendelian inheritance)
	if mathrand.Float64() < 0.5 {
		newGene.Allele1 = parent1Gene.Allele1
	} else {
		newGene.Allele1 = parent1Gene.Allele2
	}

	if mathrand.Float64() < 0.5 {
		newGene.Allele2 = parent2Gene.Allele1
	} else {
		newGene.Allele2 = parent2Gene.Allele2
	}

	// Inherit dominance (average of parents with small random variation)
	newGene.Dominance = (parent1Gene.Dominance + parent2Gene.Dominance) / 2.0
	newGene.Dominance += (mathrand.Float64() - 0.5) * 0.1 // Small random variation

	// Apply mutations
	if mathrand.Float64() < mutationRate {
		newGene.Allele1 = mutate(newGene.Allele1)
		*mutationCount++
	}

	if mathrand.Float64() < mutationRate {
		newGene.Allele2 = mutate(newGene.Allele2)
		*mutationCount++
	}

	if mathrand.Float64() < mutationRate*0.5 { // Dominance mutates less frequently
		newGene.Dominance = mutate(newGene.Dominance)
		*mutationCount++
	}

	// Clamp values
	newGene.Allele1 = clamp(newGene.Allele1, 0.0, 1.0)
	newGene.Allele2 = clamp(newGene.Allele2, 0.0, 1.0)
	newGene.Dominance = clamp(newGene.Dominance, 0.0, 1.0)

	return newGene
}

// mutate applies a random mutation to a gene value
func mutate(value float64) float64 {
	// Mutation can be small or large
	mutationSize := mathrand.Float64()

	var change float64
	if mutationSize < 0.7 {
		// Small mutation (70% chance)
		change = (mathrand.Float64() - 0.5) * 0.2
	} else if mutationSize < 0.95 {
		// Medium mutation (25% chance)
		change = (mathrand.Float64() - 0.5) * 0.5
	} else {
		// Large mutation (5% chance) - complete randomization
		return mathrand.Float64()
	}

	return value + change
}

// GetBreedingHistory returns recent breeding events
func (gs *GeneticSystem) GetBreedingHistory(limit int) []BreedingPair {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	if limit <= 0 || limit > len(gs.BreedingHistory) {
		limit = len(gs.BreedingHistory)
	}

	history := make([]BreedingPair, limit)
	copy(history, gs.BreedingHistory[len(gs.BreedingHistory)-limit:])

	return history
}

// GetOffspringCount returns how many offspring a genome has produced
func (gs *GeneticSystem) GetOffspringCount(genomeID string) int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	// Count as parent1 or parent2 in any pairing
	count := 0
	for key, c := range gs.OffspringCount {
		if containsGenomeID(key, genomeID) {
			count += c
		}
	}

	return count
}

// Helper functions

func calculateComplementaryTraits(parent1, parent2 *Genome) float64 {
	// Traits are complementary if they balance each other out
	score := 0.0
	count := 0

	// Check if size complements (one large, one small is okay)
	sizeDiff := parent1.PhysicalTraits.BaseSize - parent2.PhysicalTraits.BaseSize
	if sizeDiff > 0.3 && sizeDiff < 0.7 {
		score += 1.0
	}
	count++

	// Check if temperaments complement
	activity1 := parent1.BehavioralPredispositions.ActivityLevel
	activity2 := parent2.BehavioralPredispositions.ActivityLevel
	if (activity1 > 0.6 && activity2 < 0.4) || (activity1 < 0.4 && activity2 > 0.6) {
		score += 1.0
	}
	count++

	// Check if health traits complement
	if parent1.HealthTraits.ImmuneStrength > 0.6 && parent2.HealthTraits.MetabolicEfficiency > 0.6 {
		score += 1.0
	}
	count++

	return score / float64(count)
}

func generateOffspringDescription(offspring *Genome) string {
	desc := fmt.Sprintf("Offspring traits: ")

	// Physical description
	if offspring.PhysicalTraits.BaseSize > 0.7 {
		desc += "large, "
	} else if offspring.PhysicalTraits.BaseSize < 0.3 {
		desc += "small, "
	} else {
		desc += "medium-sized, "
	}

	if offspring.PhysicalTraits.Cuteness > 0.7 {
		desc += "adorable appearance, "
	}

	// Health description
	if offspring.HealthTraits.ImmuneStrength > 0.7 {
		desc += "strong immune system, "
	} else if offspring.HealthTraits.ImmuneStrength < 0.3 {
		desc += "delicate constitution, "
	}

	// Behavioral description
	if offspring.BehavioralPredispositions.ActivityLevel > 0.7 {
		desc += "highly energetic"
	} else if offspring.BehavioralPredispositions.ActivityLevel < 0.3 {
		desc += "calm and relaxed"
	} else {
		desc += "moderate energy"
	}

	return desc
}

func generatePairKey(id1, id2 string) string {
	// Ensure consistent key regardless of order
	if id1 < id2 {
		return id1 + "_" + id2
	}
	return id2 + "_" + id1
}

func containsGenomeID(pairKey, genomeID string) bool {
	return pairKey[0:len(genomeID)] == genomeID ||
		pairKey[len(pairKey)-len(genomeID):] == genomeID
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
