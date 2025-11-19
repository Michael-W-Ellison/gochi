package genetics

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	mathrand "math/rand"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
)

// GeneType represents different categories of genes
type GeneType int

const (
	GenePhysical GeneType = iota
	GeneHealth
	GeneBehavioral
	GenePersonality
)

// Gene represents a single genetic trait with dominant/recessive alleles
type Gene struct {
	Name      string
	Allele1   float64 // First allele value (0-1)
	Allele2   float64 // Second allele value (0-1)
	Dominance float64 // How dominant allele1 is (0-1, 0.5 = co-dominant)
	Type      GeneType
}

// Express calculates the expressed value of a gene based on dominance
func (g *Gene) Express() float64 {
	// Complete dominance if dominance > 0.8
	if g.Dominance > 0.8 {
		return g.Allele1
	}
	// Complete recessiveness if dominance < 0.2
	if g.Dominance < 0.2 {
		return g.Allele2
	}
	// Co-dominance/incomplete dominance - blend based on dominance value
	return g.Allele1*g.Dominance + g.Allele2*(1.0-g.Dominance)
}

// PhysicalTraits represents visible physical characteristics
type PhysicalTraits struct {
	// Size and build
	BaseSize        float64 // 0-1, affects max health and energy
	BodyShape       float64 // 0-1, stocky to slender
	LegLength       float64 // 0-1, short to long

	// Appearance
	PrimaryColor    float64 // 0-1, hue value
	SecondaryColor  float64 // 0-1, hue value
	PatternType     float64 // 0-1, solid to complex patterns
	FurLength       float64 // 0-1, short to long
	EyeColor        float64 // 0-1, hue value

	// Features
	EarShape        float64 // 0-1, small/folded to large/pointed
	TailLength      float64 // 0-1, short to long
	Cuteness        float64 // 0-1, subjective appeal factor
}

// HealthTraits represents genetic health predispositions
type HealthTraits struct {
	LifespanModifier      float64 // 0.5-1.5, affects natural lifespan
	MetabolicEfficiency   float64 // 0-1, affects energy usage
	ImmuneStrength        float64 // 0-1, resistance to illness
	StressResilience      float64 // 0-1, how well pet handles stress
	RecoveryRate          float64 // 0-1, how fast pet recovers from illness
	AppetiteLevel         float64 // 0-1, affects hunger rate
	SleepQuality          float64 // 0-1, affects rest efficiency
	PhysicalStamina       float64 // 0-1, affects fatigue accumulation
}

// BehavioralPredispositions represents inherited behavioral tendencies
type BehavioralPredispositions struct {
	SocialPreference      float64 // 0-1, solitary to social
	ActivityLevel         float64 // 0-1, lazy to hyperactive
	Trainability          float64 // 0-1, stubborn to eager to learn
	FearlLevel            float64 // 0-1, fearless to timid
	AggessionTendency     float64 // 0-1, passive to aggressive
	VocalizationFrequency float64 // 0-1, quiet to vocal
	TerritorialInstinct   float64 // 0-1, relaxed to possessive
	PlayDrive             float64 // 0-1, low to high play motivation
}

// Genome represents the complete genetic makeup of a digital pet
type Genome struct {
	ID            string
	Generation    int
	Parent1ID     string
	Parent2ID     string
	MutationCount int

	// Physical genes
	PhysicalGenes map[string]*Gene

	// Health genes
	HealthGenes map[string]*Gene

	// Behavioral genes
	BehavioralGenes map[string]*Gene

	// Personality genes (used to initialize PersonalityMatrix)
	PersonalityGenes map[string]*Gene

	// Expressed traits (calculated from genes)
	PhysicalTraits            *PhysicalTraits
	HealthTraits              *HealthTraits
	BehavioralPredispositions *BehavioralPredispositions
}

// NewRandomGenome creates a genome with random genetic values
func NewRandomGenome() *Genome {
	genome := &Genome{
		ID:               generateGenomeID(),
		Generation:       0,
		MutationCount:    0,
		PhysicalGenes:    make(map[string]*Gene),
		HealthGenes:      make(map[string]*Gene),
		BehavioralGenes:  make(map[string]*Gene),
		PersonalityGenes: make(map[string]*Gene),
	}

	// Initialize physical genes
	genome.PhysicalGenes["base_size"] = randomGene("base_size", GenePhysical)
	genome.PhysicalGenes["body_shape"] = randomGene("body_shape", GenePhysical)
	genome.PhysicalGenes["leg_length"] = randomGene("leg_length", GenePhysical)
	genome.PhysicalGenes["primary_color"] = randomGene("primary_color", GenePhysical)
	genome.PhysicalGenes["secondary_color"] = randomGene("secondary_color", GenePhysical)
	genome.PhysicalGenes["pattern_type"] = randomGene("pattern_type", GenePhysical)
	genome.PhysicalGenes["fur_length"] = randomGene("fur_length", GenePhysical)
	genome.PhysicalGenes["eye_color"] = randomGene("eye_color", GenePhysical)
	genome.PhysicalGenes["ear_shape"] = randomGene("ear_shape", GenePhysical)
	genome.PhysicalGenes["tail_length"] = randomGene("tail_length", GenePhysical)
	genome.PhysicalGenes["cuteness"] = randomGene("cuteness", GenePhysical)

	// Initialize health genes
	genome.HealthGenes["lifespan_modifier"] = randomGene("lifespan_modifier", GeneHealth)
	genome.HealthGenes["metabolic_efficiency"] = randomGene("metabolic_efficiency", GeneHealth)
	genome.HealthGenes["immune_strength"] = randomGene("immune_strength", GeneHealth)
	genome.HealthGenes["stress_resilience"] = randomGene("stress_resilience", GeneHealth)
	genome.HealthGenes["recovery_rate"] = randomGene("recovery_rate", GeneHealth)
	genome.HealthGenes["appetite_level"] = randomGene("appetite_level", GeneHealth)
	genome.HealthGenes["sleep_quality"] = randomGene("sleep_quality", GeneHealth)
	genome.HealthGenes["physical_stamina"] = randomGene("physical_stamina", GeneHealth)

	// Initialize behavioral genes
	genome.BehavioralGenes["social_preference"] = randomGene("social_preference", GeneBehavioral)
	genome.BehavioralGenes["activity_level"] = randomGene("activity_level", GeneBehavioral)
	genome.BehavioralGenes["trainability"] = randomGene("trainability", GeneBehavioral)
	genome.BehavioralGenes["fear_level"] = randomGene("fear_level", GeneBehavioral)
	genome.BehavioralGenes["aggression_tendency"] = randomGene("aggression_tendency", GeneBehavioral)
	genome.BehavioralGenes["vocalization_frequency"] = randomGene("vocalization_frequency", GeneBehavioral)
	genome.BehavioralGenes["territorial_instinct"] = randomGene("territorial_instinct", GeneBehavioral)
	genome.BehavioralGenes["play_drive"] = randomGene("play_drive", GeneBehavioral)

	// Initialize personality genes (Big Five + pet traits)
	genome.PersonalityGenes["openness"] = randomGene("openness", GenePersonality)
	genome.PersonalityGenes["conscientiousness"] = randomGene("conscientiousness", GenePersonality)
	genome.PersonalityGenes["extraversion"] = randomGene("extraversion", GenePersonality)
	genome.PersonalityGenes["agreeableness"] = randomGene("agreeableness", GenePersonality)
	genome.PersonalityGenes["neuroticism"] = randomGene("neuroticism", GenePersonality)
	genome.PersonalityGenes["playfulness"] = randomGene("playfulness", GenePersonality)
	genome.PersonalityGenes["independence"] = randomGene("independence", GenePersonality)
	genome.PersonalityGenes["loyalty"] = randomGene("loyalty", GenePersonality)
	genome.PersonalityGenes["intelligence"] = randomGene("intelligence", GenePersonality)
	genome.PersonalityGenes["energy_level"] = randomGene("energy_level", GenePersonality)
	genome.PersonalityGenes["affectionate"] = randomGene("affectionate", GenePersonality)
	genome.PersonalityGenes["curiosity"] = randomGene("curiosity", GenePersonality)
	genome.PersonalityGenes["adaptability"] = randomGene("adaptability", GenePersonality)
	genome.PersonalityGenes["vocalization"] = randomGene("vocalization", GenePersonality)
	genome.PersonalityGenes["territoriality"] = randomGene("territoriality", GenePersonality)

	// Express all genes to traits
	genome.ExpressGenes()

	return genome
}

// ExpressGenes calculates the expressed traits from the genetic code
func (g *Genome) ExpressGenes() {
	// Express physical traits
	g.PhysicalTraits = &PhysicalTraits{
		BaseSize:       g.PhysicalGenes["base_size"].Express(),
		BodyShape:      g.PhysicalGenes["body_shape"].Express(),
		LegLength:      g.PhysicalGenes["leg_length"].Express(),
		PrimaryColor:   g.PhysicalGenes["primary_color"].Express(),
		SecondaryColor: g.PhysicalGenes["secondary_color"].Express(),
		PatternType:    g.PhysicalGenes["pattern_type"].Express(),
		FurLength:      g.PhysicalGenes["fur_length"].Express(),
		EyeColor:       g.PhysicalGenes["eye_color"].Express(),
		EarShape:       g.PhysicalGenes["ear_shape"].Express(),
		TailLength:     g.PhysicalGenes["tail_length"].Express(),
		Cuteness:       g.PhysicalGenes["cuteness"].Express(),
	}

	// Express health traits (some need special scaling)
	lifespanVal := g.HealthGenes["lifespan_modifier"].Express()
	// Scale lifespan from 0.5 to 1.5
	lifespanModifier := 0.5 + lifespanVal

	g.HealthTraits = &HealthTraits{
		LifespanModifier:    lifespanModifier,
		MetabolicEfficiency: g.HealthGenes["metabolic_efficiency"].Express(),
		ImmuneStrength:      g.HealthGenes["immune_strength"].Express(),
		StressResilience:    g.HealthGenes["stress_resilience"].Express(),
		RecoveryRate:        g.HealthGenes["recovery_rate"].Express(),
		AppetiteLevel:       g.HealthGenes["appetite_level"].Express(),
		SleepQuality:        g.HealthGenes["sleep_quality"].Express(),
		PhysicalStamina:     g.HealthGenes["physical_stamina"].Express(),
	}

	// Express behavioral predispositions
	g.BehavioralPredispositions = &BehavioralPredispositions{
		SocialPreference:      g.BehavioralGenes["social_preference"].Express(),
		ActivityLevel:         g.BehavioralGenes["activity_level"].Express(),
		Trainability:          g.BehavioralGenes["trainability"].Express(),
		FearlLevel:            g.BehavioralGenes["fear_level"].Express(),
		AggessionTendency:     g.BehavioralGenes["aggression_tendency"].Express(),
		VocalizationFrequency: g.BehavioralGenes["vocalization_frequency"].Express(),
		TerritorialInstinct:   g.BehavioralGenes["territorial_instinct"].Express(),
		PlayDrive:             g.BehavioralGenes["play_drive"].Express(),
	}
}

// GetPersonalityTraits extracts personality traits from genes
func (g *Genome) GetPersonalityTraits() *ai.Traits {
	return &ai.Traits{
		Openness:          g.PersonalityGenes["openness"].Express(),
		Conscientiousness: g.PersonalityGenes["conscientiousness"].Express(),
		Extraversion:      g.PersonalityGenes["extraversion"].Express(),
		Agreeableness:     g.PersonalityGenes["agreeableness"].Express(),
		Neuroticism:       g.PersonalityGenes["neuroticism"].Express(),
		Playfulness:       g.PersonalityGenes["playfulness"].Express(),
		Independence:      g.PersonalityGenes["independence"].Express(),
		Loyalty:           g.PersonalityGenes["loyalty"].Express(),
		Intelligence:      g.PersonalityGenes["intelligence"].Express(),
		EnergyLevel:       g.PersonalityGenes["energy_level"].Express(),
		Affectionate:      g.PersonalityGenes["affectionate"].Express(),
		Curiosity:         g.PersonalityGenes["curiosity"].Express(),
		Adaptability:      g.PersonalityGenes["adaptability"].Express(),
		Vocalization:      g.PersonalityGenes["vocalization"].Express(),
		Territoriality:    g.PersonalityGenes["territoriality"].Express(),
	}
}

// CalculateGeneticSimilarity returns similarity between two genomes (0-1)
func (g *Genome) CalculateGeneticSimilarity(other *Genome) float64 {
	totalDiff := 0.0
	count := 0

	// Compare all gene categories
	for name, gene := range g.PhysicalGenes {
		if otherGene, exists := other.PhysicalGenes[name]; exists {
			diff1 := math.Abs(gene.Allele1 - otherGene.Allele1)
			diff2 := math.Abs(gene.Allele2 - otherGene.Allele2)
			totalDiff += (diff1 + diff2) / 2.0
			count++
		}
	}

	for name, gene := range g.HealthGenes {
		if otherGene, exists := other.HealthGenes[name]; exists {
			diff1 := math.Abs(gene.Allele1 - otherGene.Allele1)
			diff2 := math.Abs(gene.Allele2 - otherGene.Allele2)
			totalDiff += (diff1 + diff2) / 2.0
			count++
		}
	}

	for name, gene := range g.BehavioralGenes {
		if otherGene, exists := other.BehavioralGenes[name]; exists {
			diff1 := math.Abs(gene.Allele1 - otherGene.Allele1)
			diff2 := math.Abs(gene.Allele2 - otherGene.Allele2)
			totalDiff += (diff1 + diff2) / 2.0
			count++
		}
	}

	for name, gene := range g.PersonalityGenes {
		if otherGene, exists := other.PersonalityGenes[name]; exists {
			diff1 := math.Abs(gene.Allele1 - otherGene.Allele1)
			diff2 := math.Abs(gene.Allele2 - otherGene.Allele2)
			totalDiff += (diff1 + diff2) / 2.0
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	avgDiff := totalDiff / float64(count)
	// Convert difference to similarity (0 diff = 1 similarity, 1 diff = 0 similarity)
	return 1.0 - avgDiff
}

// Helper functions

func randomGene(name string, geneType GeneType) *Gene {
	return &Gene{
		Name:      name,
		Allele1:   mathrand.Float64(),
		Allele2:   mathrand.Float64(),
		Dominance: mathrand.Float64(),
		Type:      geneType,
	}
}

func generateGenomeID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "genome_" + hex.EncodeToString(bytes)[:16]
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func init() {
	mathrand.Seed(time.Now().UnixNano())
}
