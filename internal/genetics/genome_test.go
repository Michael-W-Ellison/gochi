package genetics

import (
	"testing"
)

func TestNewRandomGenome(t *testing.T) {
	genome := NewRandomGenome()

	if genome == nil {
		t.Fatal("NewRandomGenome returned nil")
	}

	if genome.ID == "" {
		t.Error("Genome ID should not be empty")
	}

	if genome.Generation != 0 {
		t.Errorf("New genome should be generation 0, got %d", genome.Generation)
	}

	// Check that all gene maps are initialized
	if len(genome.PhysicalGenes) == 0 {
		t.Error("PhysicalGenes should be initialized")
	}

	if len(genome.HealthGenes) == 0 {
		t.Error("HealthGenes should be initialized")
	}

	if len(genome.BehavioralGenes) == 0 {
		t.Error("BehavioralGenes should be initialized")
	}

	if len(genome.PersonalityGenes) == 0 {
		t.Error("PersonalityGenes should be initialized")
	}

	// Check that traits are expressed
	if genome.PhysicalTraits == nil {
		t.Error("PhysicalTraits should be expressed")
	}

	if genome.HealthTraits == nil {
		t.Error("HealthTraits should be expressed")
	}

	if genome.BehavioralPredispositions == nil {
		t.Error("BehavioralPredispositions should be expressed")
	}
}

func TestGeneExpression(t *testing.T) {
	tests := []struct {
		name      string
		allele1   float64
		allele2   float64
		dominance float64
		wantRange [2]float64 // min, max expected value
	}{
		{"Complete Dominance", 0.8, 0.2, 0.9, [2]float64{0.75, 0.85}},
		{"Complete Recessiveness", 0.8, 0.2, 0.1, [2]float64{0.15, 0.25}},
		{"Co-Dominance", 0.8, 0.2, 0.5, [2]float64{0.45, 0.55}},
		{"High Dominance", 1.0, 0.0, 1.0, [2]float64{0.95, 1.0}},
		{"Low Dominance", 1.0, 0.0, 0.0, [2]float64{0.0, 0.05}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gene := &Gene{
				Name:      "test",
				Allele1:   tt.allele1,
				Allele2:   tt.allele2,
				Dominance: tt.dominance,
				Type:      GenePhysical,
			}

			expressed := gene.Express()

			if expressed < tt.wantRange[0] || expressed > tt.wantRange[1] {
				t.Errorf("Expressed value %f not in expected range [%f, %f]",
					expressed, tt.wantRange[0], tt.wantRange[1])
			}
		})
	}
}

func TestPhysicalGenes(t *testing.T) {
	genome := NewRandomGenome()

	// Check that all physical genes exist
	requiredGenes := []string{
		"base_size", "body_shape", "leg_length", "primary_color",
		"secondary_color", "pattern_type", "fur_length", "eye_color",
		"ear_shape", "tail_length", "cuteness",
	}

	for _, geneName := range requiredGenes {
		if _, exists := genome.PhysicalGenes[geneName]; !exists {
			t.Errorf("Missing required physical gene: %s", geneName)
		}
	}

	// Check that traits are in valid range
	pt := genome.PhysicalTraits
	if pt.BaseSize < 0 || pt.BaseSize > 1 {
		t.Errorf("BaseSize %f out of range [0,1]", pt.BaseSize)
	}

	if pt.Cuteness < 0 || pt.Cuteness > 1 {
		t.Errorf("Cuteness %f out of range [0,1]", pt.Cuteness)
	}
}

func TestHealthGenes(t *testing.T) {
	genome := NewRandomGenome()

	// Check that all health genes exist
	requiredGenes := []string{
		"lifespan_modifier", "metabolic_efficiency", "immune_strength",
		"stress_resilience", "recovery_rate", "appetite_level",
		"sleep_quality", "physical_stamina",
	}

	for _, geneName := range requiredGenes {
		if _, exists := genome.HealthGenes[geneName]; !exists {
			t.Errorf("Missing required health gene: %s", geneName)
		}
	}

	// Check that health traits are in valid range
	ht := genome.HealthTraits
	if ht.LifespanModifier < 0.5 || ht.LifespanModifier > 1.5 {
		t.Errorf("LifespanModifier %f out of range [0.5,1.5]", ht.LifespanModifier)
	}

	if ht.ImmuneStrength < 0 || ht.ImmuneStrength > 1 {
		t.Errorf("ImmuneStrength %f out of range [0,1]", ht.ImmuneStrength)
	}

	if ht.MetabolicEfficiency < 0 || ht.MetabolicEfficiency > 1 {
		t.Errorf("MetabolicEfficiency %f out of range [0,1]", ht.MetabolicEfficiency)
	}
}

func TestBehavioralGenes(t *testing.T) {
	genome := NewRandomGenome()

	// Check that all behavioral genes exist
	requiredGenes := []string{
		"social_preference", "activity_level", "trainability",
		"fear_level", "aggression_tendency", "vocalization_frequency",
		"territorial_instinct", "play_drive",
	}

	for _, geneName := range requiredGenes {
		if _, exists := genome.BehavioralGenes[geneName]; !exists {
			t.Errorf("Missing required behavioral gene: %s", geneName)
		}
	}

	// Check that behavioral traits are in valid range
	bp := genome.BehavioralPredispositions
	if bp.ActivityLevel < 0 || bp.ActivityLevel > 1 {
		t.Errorf("ActivityLevel %f out of range [0,1]", bp.ActivityLevel)
	}

	if bp.Trainability < 0 || bp.Trainability > 1 {
		t.Errorf("Trainability %f out of range [0,1]", bp.Trainability)
	}
}

func TestPersonalityGenes(t *testing.T) {
	genome := NewRandomGenome()

	// Check that all personality genes exist (Big Five + pet traits)
	requiredGenes := []string{
		"openness", "conscientiousness", "extraversion",
		"agreeableness", "neuroticism", "playfulness",
		"independence", "loyalty", "intelligence",
		"energy_level", "affectionate", "curiosity",
		"adaptability", "vocalization", "territoriality",
	}

	for _, geneName := range requiredGenes {
		if _, exists := genome.PersonalityGenes[geneName]; !exists {
			t.Errorf("Missing required personality gene: %s", geneName)
		}
	}

	// Get personality traits
	traits := genome.GetPersonalityTraits()
	if traits == nil {
		t.Fatal("GetPersonalityTraits returned nil")
	}

	// Check ranges
	if traits.Openness < 0 || traits.Openness > 1 {
		t.Errorf("Openness %f out of range [0,1]", traits.Openness)
	}

	if traits.Intelligence < 0 || traits.Intelligence > 1 {
		t.Errorf("Intelligence %f out of range [0,1]", traits.Intelligence)
	}
}

func TestCalculateGeneticSimilarity(t *testing.T) {
	tests := []struct {
		name             string
		genome1          *Genome
		genome2          *Genome
		expectedSimilar  float64 // Approximate expected similarity
		expectHighSimilarity bool
	}{
		{
			name:            "Same Genome",
			genome1:         NewRandomGenome(),
			genome2:         nil, // Will be set to genome1
			expectedSimilar: 1.0,
			expectHighSimilarity: true,
		},
		{
			name:            "Different Random Genomes",
			genome1:         NewRandomGenome(),
			genome2:         NewRandomGenome(),
			expectedSimilar: 0.5,
			expectHighSimilarity: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.genome2 == nil {
				tt.genome2 = tt.genome1
			}

			similarity := tt.genome1.CalculateGeneticSimilarity(tt.genome2)

			if similarity < 0 || similarity > 1 {
				t.Errorf("Similarity %f should be between 0 and 1", similarity)
			}

			if tt.expectHighSimilarity && similarity < 0.9 {
				t.Errorf("Expected high similarity, got %f", similarity)
			}

			if !tt.expectHighSimilarity && similarity > 0.9 {
				t.Errorf("Expected low similarity, got %f", similarity)
			}
		})
	}
}

func TestGeneSelfSimilarity(t *testing.T) {
	genome := NewRandomGenome()
	similarity := genome.CalculateGeneticSimilarity(genome)

	if similarity < 0.99 {
		t.Errorf("Genome should be nearly 100%% similar to itself, got %f", similarity)
	}
}

func TestRandomGenomesAreDifferent(t *testing.T) {
	genome1 := NewRandomGenome()
	genome2 := NewRandomGenome()

	if genome1.ID == genome2.ID {
		t.Error("Two random genomes should have different IDs")
	}

	// They should have some differences
	similarity := genome1.CalculateGeneticSimilarity(genome2)
	if similarity > 0.95 {
		t.Errorf("Two random genomes are too similar (%f), expected more diversity", similarity)
	}
}

func TestExpressGenes(t *testing.T) {
	genome := NewRandomGenome()

	// Modify a gene
	genome.PhysicalGenes["base_size"].Allele1 = 0.9
	genome.PhysicalGenes["base_size"].Allele2 = 0.1
	genome.PhysicalGenes["base_size"].Dominance = 1.0 // Complete dominance of allele1

	// Re-express genes
	genome.ExpressGenes()

	// Should express mostly the dominant allele
	if genome.PhysicalTraits.BaseSize < 0.85 {
		t.Errorf("Expected high BaseSize due to dominant allele, got %f",
			genome.PhysicalTraits.BaseSize)
	}
}

func TestLifespanModifierScaling(t *testing.T) {
	genome := NewRandomGenome()

	// Lifespan should be scaled to 0.5-1.5 range
	if genome.HealthTraits.LifespanModifier < 0.5 {
		t.Errorf("LifespanModifier %f should be >= 0.5", genome.HealthTraits.LifespanModifier)
	}

	if genome.HealthTraits.LifespanModifier > 1.5 {
		t.Errorf("LifespanModifier %f should be <= 1.5", genome.HealthTraits.LifespanModifier)
	}
}

func TestGenomeIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		genome := NewRandomGenome()
		if ids[genome.ID] {
			t.Errorf("Duplicate genome ID generated: %s", genome.ID)
		}
		ids[genome.ID] = true
	}

	if len(ids) != iterations {
		t.Errorf("Expected %d unique IDs, got %d", iterations, len(ids))
	}
}

func TestAlleleValues(t *testing.T) {
	genome := NewRandomGenome()

	// Check all genes have valid allele values
	checkAlleles := func(genes map[string]*Gene, category string) {
		for name, gene := range genes {
			if gene.Allele1 < 0 || gene.Allele1 > 1 {
				t.Errorf("%s gene %s has invalid Allele1: %f", category, name, gene.Allele1)
			}
			if gene.Allele2 < 0 || gene.Allele2 > 1 {
				t.Errorf("%s gene %s has invalid Allele2: %f", category, name, gene.Allele2)
			}
			if gene.Dominance < 0 || gene.Dominance > 1 {
				t.Errorf("%s gene %s has invalid Dominance: %f", category, name, gene.Dominance)
			}
		}
	}

	checkAlleles(genome.PhysicalGenes, "Physical")
	checkAlleles(genome.HealthGenes, "Health")
	checkAlleles(genome.BehavioralGenes, "Behavioral")
	checkAlleles(genome.PersonalityGenes, "Personality")
}

func TestGeneTypes(t *testing.T) {
	genome := NewRandomGenome()

	// Check that genes have correct types
	for _, gene := range genome.PhysicalGenes {
		if gene.Type != GenePhysical {
			t.Error("Physical gene should have type GenePhysical")
		}
	}

	for _, gene := range genome.HealthGenes {
		if gene.Type != GeneHealth {
			t.Error("Health gene should have type GeneHealth")
		}
	}

	for _, gene := range genome.BehavioralGenes {
		if gene.Type != GeneBehavioral {
			t.Error("Behavioral gene should have type GeneBehavioral")
		}
	}

	for _, gene := range genome.PersonalityGenes {
		if gene.Type != GenePersonality {
			t.Error("Personality gene should have type GenePersonality")
		}
	}
}

func BenchmarkNewRandomGenome(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewRandomGenome()
	}
}

func BenchmarkExpressGenes(b *testing.B) {
	genome := NewRandomGenome()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		genome.ExpressGenes()
	}
}

func BenchmarkCalculateGeneticSimilarity(b *testing.B) {
	genome1 := NewRandomGenome()
	genome2 := NewRandomGenome()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		genome1.CalculateGeneticSimilarity(genome2)
	}
}
