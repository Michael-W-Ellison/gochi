package genetics

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// BreedingRequirements defines minimum requirements for breeding
type BreedingRequirements struct {
	MinAge            float64 // Minimum age in days
	MaxAge            float64 // Maximum age in days
	MinHealth         float64 // Minimum health (0.0 to 1.0)
	MinHappiness      float64 // Minimum happiness (0.0 to 1.0)
	CooldownDays      float64 // Days between breeding attempts
	MinCompatibility  float64 // Minimum compatibility score
}

// DefaultBreedingRequirements returns standard breeding requirements
func DefaultBreedingRequirements() BreedingRequirements {
	return BreedingRequirements{
		MinAge:           7.0,   // At least 7 days old
		MaxAge:           365.0, // Not older than 1 year
		MinHealth:        0.6,
		MinHappiness:     0.5,
		CooldownDays:     3.0,
		MinCompatibility: 0.3,
	}
}

// BreedingPair represents two pets that may breed together
type BreedingPair struct {
	Parent1ID          string
	Parent2ID          string
	Parent1Genome      *Genome
	Parent2Genome      *Genome
	CompatibilityScore float64
	LastBreedingTime   time.Time
}

// BreedingResult contains the outcome of a breeding attempt
type BreedingResult struct {
	Success          bool
	OffspringGenome  *Genome
	Mutations        []Mutation
	InheritedTraits  map[TraitType]string // Which parent each trait came from
	CompatibilityScore float64
	FailureReason    string
}

// GeneticSystem manages all genetic and breeding operations
type GeneticSystem struct {
	MutationRate     float64
	CrossoverPoints  int
	Requirements     BreedingRequirements
	BreedingHistory  []BreedingResult
}

// NewGeneticSystem creates a new genetic system with default settings
func NewGeneticSystem() *GeneticSystem {
	return &GeneticSystem{
		MutationRate:    0.05,  // 5% base mutation rate
		CrossoverPoints: 3,     // Number of crossover points in chromosome
		Requirements:    DefaultBreedingRequirements(),
		BreedingHistory: make([]BreedingResult, 0),
	}
}

// NewGeneticSystemWithConfig creates a genetic system with custom settings
func NewGeneticSystemWithConfig(mutationRate float64, crossoverPoints int, req BreedingRequirements) *GeneticSystem {
	return &GeneticSystem{
		MutationRate:    mutationRate,
		CrossoverPoints: crossoverPoints,
		Requirements:    req,
		BreedingHistory: make([]BreedingResult, 0),
	}
}

// CalculateCompatibility computes breeding compatibility between two genomes
func (gs *GeneticSystem) CalculateCompatibility(genome1, genome2 *Genome) float64 {
	if genome1 == nil || genome2 == nil {
		return 0.0
	}

	var totalDifference float64
	var traitCount float64

	// Compare all shared traits
	for trait, allele1 := range genome1.Traits {
		if allele2, exists := genome2.Traits[trait]; exists {
			// Difference between expressed values
			valueDiff := math.Abs(allele1.Value - allele2.Value)

			// Genetic diversity bonus - some difference is good
			diversityBonus := 0.0
			if valueDiff > 0.1 && valueDiff < 0.5 {
				diversityBonus = 0.1
			}

			totalDifference += valueDiff - diversityBonus
			traitCount++
		}
	}

	if traitCount == 0 {
		return 0.5 // Default compatibility
	}

	// Average difference, inverted to get compatibility
	avgDifference := totalDifference / traitCount

	// Compatibility is higher when there's moderate genetic diversity
	// Too similar (inbreeding) or too different both reduce compatibility
	compatibility := 1.0 - math.Abs(avgDifference-0.3)*2

	// Factor in generation difference (closer generations are more compatible)
	genDiff := math.Abs(float64(genome1.Generation - genome2.Generation))
	genPenalty := genDiff * 0.05 // 5% penalty per generation difference
	compatibility -= genPenalty

	return clamp(compatibility, 0.0, 1.0)
}

// CanBreed checks if two pets meet breeding requirements
func (gs *GeneticSystem) CanBreed(age1, age2, health1, health2, happiness1, happiness2 float64,
	lastBreeding1, lastBreeding2 time.Time) (bool, string) {

	req := gs.Requirements
	now := time.Now()

	// Age checks
	if age1 < req.MinAge || age2 < req.MinAge {
		return false, "one or both pets are too young"
	}
	if age1 > req.MaxAge || age2 > req.MaxAge {
		return false, "one or both pets are too old"
	}

	// Health checks
	if health1 < req.MinHealth || health2 < req.MinHealth {
		return false, "one or both pets are not healthy enough"
	}

	// Happiness checks
	if happiness1 < req.MinHappiness || happiness2 < req.MinHappiness {
		return false, "one or both pets are not happy enough"
	}

	// Cooldown checks
	cooldownDuration := time.Duration(req.CooldownDays * 24 * float64(time.Hour))
	if !lastBreeding1.IsZero() && now.Sub(lastBreeding1) < cooldownDuration {
		return false, "parent 1 is still in cooldown"
	}
	if !lastBreeding2.IsZero() && now.Sub(lastBreeding2) < cooldownDuration {
		return false, "parent 2 is still in cooldown"
	}

	return true, ""
}

// Crossover performs genetic crossover between two parent genomes
func (gs *GeneticSystem) Crossover(parent1, parent2 *Genome) *Genome {
	offspring := &Genome{
		ID:               generateGenomeID(),
		Traits:           make(map[TraitType]GeneAllele),
		Mutations:        make([]Mutation, 0),
		GeneticDiversity: 0.0,
		Generation:       max(parent1.Generation, parent2.Generation) + 1,
		ParentIDs:        []string{parent1.ID, parent2.ID},
		CreatedAt:        time.Now(),
	}

	// Get all traits from both parents
	allTraits := make([]TraitType, 0)
	for trait := range parent1.Traits {
		allTraits = append(allTraits, trait)
	}

	// Generate crossover points
	crossoverIndices := make([]int, gs.CrossoverPoints)
	for i := 0; i < gs.CrossoverPoints; i++ {
		crossoverIndices[i] = rand.Intn(len(allTraits))
	}

	// Sort crossover points
	for i := 0; i < len(crossoverIndices)-1; i++ {
		for j := i + 1; j < len(crossoverIndices); j++ {
			if crossoverIndices[i] > crossoverIndices[j] {
				crossoverIndices[i], crossoverIndices[j] = crossoverIndices[j], crossoverIndices[i]
			}
		}
	}

	// Apply crossover
	currentParent := 1 // Start with parent 1
	crossoverIdx := 0

	for i, trait := range allTraits {
		// Check if we've reached a crossover point
		if crossoverIdx < len(crossoverIndices) && i >= crossoverIndices[crossoverIdx] {
			currentParent = 3 - currentParent // Switch between 1 and 2
			crossoverIdx++
		}

		// Get allele from current parent
		var parentAllele GeneAllele
		if currentParent == 1 {
			parentAllele = parent1.Traits[trait]
		} else {
			parentAllele = parent2.Traits[trait]
		}

		// Create offspring allele with some recombination
		offspringAllele := GeneAllele{
			Dominant:   parentAllele.Dominant,
			Recessive:  parentAllele.Recessive,
			Dominance:  parentAllele.Dominance,
			Expression: parentAllele.Expression,
		}

		// 50% chance to swap dominant/recessive
		if rand.Float64() < 0.5 {
			offspringAllele.Dominant, offspringAllele.Recessive = offspringAllele.Recessive, offspringAllele.Dominant
		}

		// Recalculate expressed value
		offspringAllele.Value = calculateExpression(
			offspringAllele.Dominant,
			offspringAllele.Recessive,
			offspringAllele.Dominance,
		)

		offspring.Traits[trait] = offspringAllele
	}

	return offspring
}

// Breed attempts to create offspring from two parent genomes
func (gs *GeneticSystem) Breed(parent1, parent2 *Genome) BreedingResult {
	result := BreedingResult{
		Success:         false,
		InheritedTraits: make(map[TraitType]string),
	}

	// Calculate compatibility
	result.CompatibilityScore = gs.CalculateCompatibility(parent1, parent2)

	// Check minimum compatibility
	if result.CompatibilityScore < gs.Requirements.MinCompatibility {
		result.FailureReason = fmt.Sprintf("compatibility too low (%.2f < %.2f)",
			result.CompatibilityScore, gs.Requirements.MinCompatibility)
		// Record failed attempt in history
		gs.BreedingHistory = append(gs.BreedingHistory, result)
		return result
	}

	// Breeding success chance based on compatibility
	successChance := 0.5 + result.CompatibilityScore*0.5
	if rand.Float64() > successChance {
		result.FailureReason = "breeding attempt failed (random chance)"
		// Record failed attempt in history
		gs.BreedingHistory = append(gs.BreedingHistory, result)
		return result
	}

	// Perform crossover
	offspring := gs.Crossover(parent1, parent2)

	// Apply mutations
	mutations := offspring.Mutate(gs.MutationRate)
	result.Mutations = mutations

	// Track trait inheritance
	for trait := range offspring.Traits {
		p1Val := parent1.GetTraitValue(trait)
		p2Val := parent2.GetTraitValue(trait)
		offVal := offspring.GetTraitValue(trait)

		// Determine which parent the trait is closer to
		if math.Abs(offVal-p1Val) < math.Abs(offVal-p2Val) {
			result.InheritedTraits[trait] = "parent1"
		} else {
			result.InheritedTraits[trait] = "parent2"
		}
	}

	// Calculate offspring's genetic diversity
	offspring.CalculateGeneticDiversity()

	result.Success = true
	result.OffspringGenome = offspring

	// Record in history
	gs.BreedingHistory = append(gs.BreedingHistory, result)

	return result
}

// CreateOffspring is a convenience method that handles the full breeding process
func (gs *GeneticSystem) CreateOffspring(parent1ID, parent2ID string,
	parent1Genome, parent2Genome *Genome,
	age1, age2, health1, health2, happiness1, happiness2 float64,
	lastBreeding1, lastBreeding2 time.Time) BreedingResult {

	result := BreedingResult{
		Success:         false,
		InheritedTraits: make(map[TraitType]string),
	}

	// Check breeding requirements
	canBreed, reason := gs.CanBreed(age1, age2, health1, health2, happiness1, happiness2,
		lastBreeding1, lastBreeding2)
	if !canBreed {
		result.FailureReason = reason
		return result
	}

	// Attempt breeding
	return gs.Breed(parent1Genome, parent2Genome)
}

// GetBreedingStats returns statistics about breeding history
func (gs *GeneticSystem) GetBreedingStats() map[string]interface{} {
	totalAttempts := len(gs.BreedingHistory)
	successfulBreedings := 0
	totalMutations := 0
	avgCompatibility := 0.0

	for _, result := range gs.BreedingHistory {
		if result.Success {
			successfulBreedings++
		}
		totalMutations += len(result.Mutations)
		avgCompatibility += result.CompatibilityScore
	}

	if totalAttempts > 0 {
		avgCompatibility /= float64(totalAttempts)
	}

	return map[string]interface{}{
		"total_attempts":     totalAttempts,
		"successful":         successfulBreedings,
		"success_rate":       float64(successfulBreedings) / float64(max(totalAttempts, 1)),
		"total_mutations":    totalMutations,
		"avg_compatibility":  avgCompatibility,
	}
}

// SetMutationRate updates the mutation rate
func (gs *GeneticSystem) SetMutationRate(rate float64) {
	gs.MutationRate = clamp(rate, 0.0, 1.0)
}

// SetCrossoverPoints updates the number of crossover points
func (gs *GeneticSystem) SetCrossoverPoints(points int) {
	if points < 1 {
		points = 1
	}
	gs.CrossoverPoints = points
}

// PredictOffspringTraits estimates likely offspring traits without breeding
func (gs *GeneticSystem) PredictOffspringTraits(parent1, parent2 *Genome) map[TraitType]struct {
	Min float64
	Max float64
	Avg float64
} {
	predictions := make(map[TraitType]struct {
		Min float64
		Max float64
		Avg float64
	})

	for trait := range parent1.Traits {
		p1Val := parent1.GetTraitValue(trait)
		p2Val := parent2.GetTraitValue(trait)

		minVal := math.Min(p1Val, p2Val) * 0.9 // Account for possible mutation decrease
		maxVal := math.Max(p1Val, p2Val) * 1.1 // Account for possible mutation increase
		avgVal := (p1Val + p2Val) / 2.0

		predictions[trait] = struct {
			Min float64
			Max float64
			Avg float64
		}{
			Min: clamp(minVal, 0.0, 1.0),
			Max: clamp(maxVal, 0.0, 1.0),
			Avg: avgVal,
		}
	}

	return predictions
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
