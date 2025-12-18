package genetics

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// TraitType represents different genetic traits that can be inherited
type TraitType string

const (
	// Physical traits
	TraitSize       TraitType = "size"
	TraitSpeed      TraitType = "speed"
	TraitStrength   TraitType = "strength"
	TraitEndurance  TraitType = "endurance"
	TraitLongevity  TraitType = "longevity"

	// Appearance traits
	TraitColorPrimary   TraitType = "color_primary"
	TraitColorSecondary TraitType = "color_secondary"
	TraitPattern        TraitType = "pattern"
	TraitEyeColor       TraitType = "eye_color"

	// Behavioral predispositions (genetic basis for personality)
	TraitBasePlayfulness    TraitType = "base_playfulness"
	TraitBaseIntelligence   TraitType = "base_intelligence"
	TraitBaseAggression     TraitType = "base_aggression"
	TraitBaseSociability    TraitType = "base_sociability"
	TraitBaseAnxiety        TraitType = "base_anxiety"
	TraitBaseEnergy         TraitType = "base_energy"
	TraitBaseCuriosity      TraitType = "base_curiosity"
	TraitBaseAffection      TraitType = "base_affection"

	// Health traits
	TraitImmuneStrength    TraitType = "immune_strength"
	TraitMetabolicRate     TraitType = "metabolic_rate"
	TraitDiseaseSusceptibility TraitType = "disease_susceptibility"
)

// GeneAllele represents a single gene's allele information
type GeneAllele struct {
	Value      float64 `json:"value"`       // The expressed value (0.0 to 1.0)
	Dominant   float64 `json:"dominant"`    // Dominant allele value
	Recessive  float64 `json:"recessive"`   // Recessive allele value
	Dominance  float64 `json:"dominance"`   // How dominant the dominant allele is (0.0 to 1.0)
	Expression float64 `json:"expression"`  // How strongly this gene is expressed
}

// NewGeneAllele creates a new gene allele with random values
func NewGeneAllele() GeneAllele {
	dominant := rand.Float64()
	recessive := rand.Float64()
	dominance := 0.5 + rand.Float64()*0.5 // Bias toward dominant expression

	return GeneAllele{
		Dominant:   dominant,
		Recessive:  recessive,
		Dominance:  dominance,
		Expression: 0.8 + rand.Float64()*0.2, // High expression by default
		Value:      calculateExpression(dominant, recessive, dominance),
	}
}

// NewGeneAlleleWithValue creates a gene allele with a specific target value
func NewGeneAlleleWithValue(targetValue float64) GeneAllele {
	dominance := 0.5 + rand.Float64()*0.5
	// Work backwards from target to set alleles
	dominant := targetValue + (rand.Float64()-0.5)*0.1
	recessive := targetValue - (rand.Float64()-0.5)*0.1

	dominant = clamp(dominant, 0.0, 1.0)
	recessive = clamp(recessive, 0.0, 1.0)

	return GeneAllele{
		Dominant:   dominant,
		Recessive:  recessive,
		Dominance:  dominance,
		Expression: 0.8 + rand.Float64()*0.2,
		Value:      calculateExpression(dominant, recessive, dominance),
	}
}

// calculateExpression computes the expressed value from alleles
func calculateExpression(dominant, recessive, dominance float64) float64 {
	// Weighted average based on dominance
	return dominant*dominance + recessive*(1.0-dominance)
}

// Mutation represents a genetic mutation
type Mutation struct {
	TraitAffected TraitType `json:"trait_affected"`
	OriginalValue float64   `json:"original_value"`
	MutatedValue  float64   `json:"mutated_value"`
	Generation    int       `json:"generation"`
	Beneficial    bool      `json:"beneficial"`
	Description   string    `json:"description"`
}

// Genome represents the complete genetic code of a digital pet
type Genome struct {
	ID               string                   `json:"id"`
	Traits           map[TraitType]GeneAllele `json:"traits"`
	Mutations        []Mutation               `json:"mutations"`
	GeneticDiversity float64                  `json:"genetic_diversity"`
	Generation       int                      `json:"generation"`
	ParentIDs        []string                 `json:"parent_ids"`
	CreatedAt        time.Time                `json:"created_at"`
}

// NewGenome creates a new genome with random genetic values
func NewGenome() *Genome {
	g := &Genome{
		ID:               generateGenomeID(),
		Traits:           make(map[TraitType]GeneAllele),
		Mutations:        make([]Mutation, 0),
		GeneticDiversity: 1.0,
		Generation:       1,
		ParentIDs:        make([]string, 0),
		CreatedAt:        time.Now(),
	}

	// Initialize all trait types with random values
	allTraits := []TraitType{
		TraitSize, TraitSpeed, TraitStrength, TraitEndurance, TraitLongevity,
		TraitColorPrimary, TraitColorSecondary, TraitPattern, TraitEyeColor,
		TraitBasePlayfulness, TraitBaseIntelligence, TraitBaseAggression,
		TraitBaseSociability, TraitBaseAnxiety, TraitBaseEnergy,
		TraitBaseCuriosity, TraitBaseAffection,
		TraitImmuneStrength, TraitMetabolicRate, TraitDiseaseSusceptibility,
	}

	for _, trait := range allTraits {
		g.Traits[trait] = NewGeneAllele()
	}

	return g
}

// NewGenomeWithTraits creates a genome with specific trait values
func NewGenomeWithTraits(traitValues map[TraitType]float64) *Genome {
	g := NewGenome()

	for trait, value := range traitValues {
		g.Traits[trait] = NewGeneAlleleWithValue(value)
	}

	return g
}

// GetTraitValue returns the expressed value of a specific trait
func (g *Genome) GetTraitValue(trait TraitType) float64 {
	if allele, exists := g.Traits[trait]; exists {
		return allele.Value * allele.Expression
	}
	return 0.5 // Default middle value
}

// SetTraitValue updates the expressed value of a trait
func (g *Genome) SetTraitValue(trait TraitType, value float64) {
	if allele, exists := g.Traits[trait]; exists {
		allele.Value = clamp(value, 0.0, 1.0)
		g.Traits[trait] = allele
	}
}

// Mutate applies random mutations to the genome
func (g *Genome) Mutate(mutationRate float64) []Mutation {
	newMutations := make([]Mutation, 0)

	for trait, allele := range g.Traits {
		if rand.Float64() < mutationRate {
			originalValue := allele.Value

			// Determine mutation magnitude (usually small)
			mutationMagnitude := (rand.Float64() - 0.5) * 0.3 // +/- 15%

			// Apply mutation to both alleles
			allele.Dominant = clamp(allele.Dominant+mutationMagnitude*rand.Float64(), 0.0, 1.0)
			allele.Recessive = clamp(allele.Recessive+mutationMagnitude*rand.Float64(), 0.0, 1.0)
			allele.Value = calculateExpression(allele.Dominant, allele.Recessive, allele.Dominance)

			g.Traits[trait] = allele

			// Record the mutation
			mutation := Mutation{
				TraitAffected: trait,
				OriginalValue: originalValue,
				MutatedValue:  allele.Value,
				Generation:    g.Generation,
				Beneficial:    isMutationBeneficial(trait, originalValue, allele.Value),
				Description:   describeMutation(trait, originalValue, allele.Value),
			}

			newMutations = append(newMutations, mutation)
			g.Mutations = append(g.Mutations, mutation)
		}
	}

	return newMutations
}

// CalculateGeneticDiversity computes how diverse the genome is
func (g *Genome) CalculateGeneticDiversity() float64 {
	if len(g.Traits) == 0 {
		return 0.0
	}

	// Calculate variance of trait values
	var sum, sumSq float64
	count := float64(len(g.Traits))

	for _, allele := range g.Traits {
		sum += allele.Value
		sumSq += allele.Value * allele.Value
	}

	mean := sum / count
	variance := (sumSq / count) - (mean * mean)

	// Also consider heterozygosity (difference between alleles)
	var heterozygosity float64
	for _, allele := range g.Traits {
		heterozygosity += math.Abs(allele.Dominant - allele.Recessive)
	}
	heterozygosity /= count

	g.GeneticDiversity = (variance + heterozygosity) / 2.0
	return g.GeneticDiversity
}

// Clone creates a deep copy of the genome
func (g *Genome) Clone() *Genome {
	clone := &Genome{
		ID:               generateGenomeID(),
		Traits:           make(map[TraitType]GeneAllele),
		Mutations:        make([]Mutation, len(g.Mutations)),
		GeneticDiversity: g.GeneticDiversity,
		Generation:       g.Generation,
		ParentIDs:        make([]string, len(g.ParentIDs)),
		CreatedAt:        time.Now(),
	}

	for trait, allele := range g.Traits {
		clone.Traits[trait] = allele
	}

	copy(clone.Mutations, g.Mutations)
	copy(clone.ParentIDs, g.ParentIDs)

	return clone
}

// GetPhysicalTraits returns a summary of physical characteristics
func (g *Genome) GetPhysicalTraits() map[string]float64 {
	return map[string]float64{
		"size":      g.GetTraitValue(TraitSize),
		"speed":     g.GetTraitValue(TraitSpeed),
		"strength":  g.GetTraitValue(TraitStrength),
		"endurance": g.GetTraitValue(TraitEndurance),
		"longevity": g.GetTraitValue(TraitLongevity),
	}
}

// GetBehavioralPredispositions returns genetic tendencies for behavior
func (g *Genome) GetBehavioralPredispositions() map[string]float64 {
	return map[string]float64{
		"playfulness":  g.GetTraitValue(TraitBasePlayfulness),
		"intelligence": g.GetTraitValue(TraitBaseIntelligence),
		"aggression":   g.GetTraitValue(TraitBaseAggression),
		"sociability":  g.GetTraitValue(TraitBaseSociability),
		"anxiety":      g.GetTraitValue(TraitBaseAnxiety),
		"energy":       g.GetTraitValue(TraitBaseEnergy),
		"curiosity":    g.GetTraitValue(TraitBaseCuriosity),
		"affection":    g.GetTraitValue(TraitBaseAffection),
	}
}

// GetHealthTraits returns health-related genetic traits
func (g *Genome) GetHealthTraits() map[string]float64 {
	return map[string]float64{
		"immune_strength":        g.GetTraitValue(TraitImmuneStrength),
		"metabolic_rate":         g.GetTraitValue(TraitMetabolicRate),
		"disease_susceptibility": g.GetTraitValue(TraitDiseaseSusceptibility),
	}
}

// Helper functions

func generateGenomeID() string {
	return fmt.Sprintf("genome_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func isMutationBeneficial(trait TraitType, original, mutated float64) bool {
	// For most traits, higher is better, except for negative traits
	negativeTraits := map[TraitType]bool{
		TraitBaseAggression:        true,
		TraitBaseAnxiety:           true,
		TraitDiseaseSusceptibility: true,
	}

	if negativeTraits[trait] {
		return mutated < original
	}
	return mutated > original
}

func describeMutation(trait TraitType, original, mutated float64) string {
	change := mutated - original
	direction := "increased"
	if change < 0 {
		direction = "decreased"
		change = -change
	}

	magnitude := "slightly"
	if change > 0.1 {
		magnitude = "significantly"
	} else if change > 0.2 {
		magnitude = "dramatically"
	}

	return fmt.Sprintf("%s %s %s (%.2f -> %.2f)", string(trait), magnitude, direction, original, mutated)
}
