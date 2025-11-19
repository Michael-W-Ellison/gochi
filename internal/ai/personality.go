package ai

import (
	"math"
	"math/rand"
)

// Traits represents the personality characteristics of a digital pet
// Based on the Big Five personality model plus pet-specific traits
type Traits struct {
	// Big Five personality traits (0.0 to 1.0)
	Openness          float64 `json:"openness"`          // Curiosity, creativity, openness to experience
	Conscientiousness float64 `json:"conscientiousness"` // Discipline, organization, reliability
	Extraversion      float64 `json:"extraversion"`      // Social energy, assertiveness, enthusiasm
	Agreeableness     float64 `json:"agreeableness"`     // Cooperation, trust, compassion
	Neuroticism       float64 `json:"neuroticism"`       // Emotional stability (higher = less stable)

	// Pet-specific traits (0.0 to 1.0)
	Playfulness   float64 `json:"playfulness"`   // Desire to play and have fun
	Independence  float64 `json:"independence"`  // Self-reliance vs. neediness
	Loyalty       float64 `json:"loyalty"`       // Attachment to caregiver
	Intelligence  float64 `json:"intelligence"`  // Learning speed and problem-solving
	EnergyLevel   float64 `json:"energy_level"`  // Natural activity level
	Affectionate  float64 `json:"affectionate"`  // Desire for physical affection
	Curiosity     float64 `json:"curiosity"`     // Exploration drive
	Adaptability  float64 `json:"adaptability"`  // Ability to handle change
	Vocalization  float64 `json:"vocalization"`  // Tendency to make sounds/communicate
	Territoriality float64 `json:"territoriality"` // Protective of space/resources
}

// NewRandomTraits generates a random personality
func NewRandomTraits() *Traits {
	return &Traits{
		Openness:          rand.Float64(),
		Conscientiousness: rand.Float64(),
		Extraversion:      rand.Float64(),
		Agreeableness:     rand.Float64(),
		Neuroticism:       rand.Float64(),
		Playfulness:       rand.Float64(),
		Independence:      rand.Float64(),
		Loyalty:           0.5 + rand.Float64()*0.5, // Bias toward loyal
		Intelligence:      0.3 + rand.Float64()*0.7, // Minimum intelligence
		EnergyLevel:       rand.Float64(),
		Affectionate:      rand.Float64(),
		Curiosity:         rand.Float64(),
		Adaptability:      rand.Float64(),
		Vocalization:      rand.Float64(),
		Territoriality:    rand.Float64(),
	}
}

// NewBalancedTraits creates a balanced personality (all traits at 0.5)
func NewBalancedTraits() *Traits {
	return &Traits{
		Openness:          0.5,
		Conscientiousness: 0.5,
		Extraversion:      0.5,
		Agreeableness:     0.5,
		Neuroticism:       0.3, // Lower neuroticism is better
		Playfulness:       0.6,
		Independence:      0.4,
		Loyalty:           0.7,
		Intelligence:      0.6,
		EnergyLevel:       0.5,
		Affectionate:      0.6,
		Curiosity:         0.6,
		Adaptability:      0.6,
		Vocalization:      0.5,
		Territoriality:    0.3,
	}
}

// Clamp ensures all traits remain within valid range [0.0, 1.0]
func (t *Traits) Clamp() {
	t.Openness = clamp(t.Openness, 0.0, 1.0)
	t.Conscientiousness = clamp(t.Conscientiousness, 0.0, 1.0)
	t.Extraversion = clamp(t.Extraversion, 0.0, 1.0)
	t.Agreeableness = clamp(t.Agreeableness, 0.0, 1.0)
	t.Neuroticism = clamp(t.Neuroticism, 0.0, 1.0)
	t.Playfulness = clamp(t.Playfulness, 0.0, 1.0)
	t.Independence = clamp(t.Independence, 0.0, 1.0)
	t.Loyalty = clamp(t.Loyalty, 0.0, 1.0)
	t.Intelligence = clamp(t.Intelligence, 0.0, 1.0)
	t.EnergyLevel = clamp(t.EnergyLevel, 0.0, 1.0)
	t.Affectionate = clamp(t.Affectionate, 0.0, 1.0)
	t.Curiosity = clamp(t.Curiosity, 0.0, 1.0)
	t.Adaptability = clamp(t.Adaptability, 0.0, 1.0)
	t.Vocalization = clamp(t.Vocalization, 0.0, 1.0)
	t.Territoriality = clamp(t.Territoriality, 0.0, 1.0)
}

// ExperienceData represents an experience that can influence personality
type ExperienceData struct {
	ExperienceType string  // Type of experience (positive, negative, neutral)
	Intensity      float64 // How strong the experience was
	TraitAffected  string  // Which trait is primarily affected
	Duration       float64 // How long the experience lasted
}

// PersonalityMatrix manages the personality traits and their evolution
type PersonalityMatrix struct {
	Traits          *Traits
	EvolutionRate   float64            // How quickly traits change
	TraitHistory    []TraitSnapshot    // Historical trait values
	ExperienceCount int                // Total experiences processed
	TraitBias       map[string]float64 // Genetic/innate biases for traits
}

// TraitSnapshot captures trait values at a point in time
type TraitSnapshot struct {
	Timestamp float64 // Game time when snapshot was taken
	Traits    *Traits
}

// NewPersonalityMatrix creates a new personality system with balanced traits
func NewPersonalityMatrix() *PersonalityMatrix {
	return &PersonalityMatrix{
		Traits:        NewBalancedTraits(),
		EvolutionRate: 0.01,
		TraitHistory:  make([]TraitSnapshot, 0),
		TraitBias:     make(map[string]float64),
	}
}

// NewPersonalityMatrixRandom creates a personality system with random traits
func NewPersonalityMatrixRandom() *PersonalityMatrix {
	return &PersonalityMatrix{
		Traits:        NewRandomTraits(),
		EvolutionRate: 0.01,
		TraitHistory:  make([]TraitSnapshot, 0),
		TraitBias:     make(map[string]float64),
	}
}

// EvolveTraits modifies personality traits based on experiences
func (p *PersonalityMatrix) EvolveTraits(experiences []ExperienceData) {
	for _, exp := range experiences {
		p.applyExperience(exp)
		p.ExperienceCount++
	}

	p.Traits.Clamp()
}

// applyExperience applies a single experience to modify traits
func (p *PersonalityMatrix) applyExperience(exp ExperienceData) {
	changeAmount := exp.Intensity * p.EvolutionRate

	switch exp.TraitAffected {
	case "openness":
		p.Traits.Openness += changeAmount
	case "conscientiousness":
		p.Traits.Conscientiousness += changeAmount
	case "extraversion":
		p.Traits.Extraversion += changeAmount
	case "agreeableness":
		p.Traits.Agreeableness += changeAmount
	case "neuroticism":
		p.Traits.Neuroticism += changeAmount
	case "playfulness":
		p.Traits.Playfulness += changeAmount
	case "independence":
		p.Traits.Independence += changeAmount
	case "loyalty":
		p.Traits.Loyalty += changeAmount
	case "curiosity":
		p.Traits.Curiosity += changeAmount
	case "adaptability":
		p.Traits.Adaptability += changeAmount
	}
}

// GetTraitInfluence returns how much a specific trait influences a behavior
// behaviorType could be "play", "social", "explore", etc.
func (p *PersonalityMatrix) GetTraitInfluence(behaviorType string) float64 {
	switch behaviorType {
	case "play":
		return (p.Traits.Playfulness + p.Traits.EnergyLevel + p.Traits.Extraversion) / 3.0
	case "social":
		return (p.Traits.Extraversion + p.Traits.Agreeableness + (1.0 - p.Traits.Independence)) / 3.0
	case "explore":
		return (p.Traits.Curiosity + p.Traits.Openness + p.Traits.EnergyLevel) / 3.0
	case "rest":
		return (1.0 - p.Traits.EnergyLevel) / 2.0
	case "affection":
		return (p.Traits.Affectionate + p.Traits.Loyalty + (1.0 - p.Traits.Independence)) / 3.0
	case "learning":
		return (p.Traits.Intelligence + p.Traits.Curiosity + p.Traits.Openness) / 3.0
	case "routine":
		return (p.Traits.Conscientiousness + (1.0 - p.Traits.Openness)) / 2.0
	default:
		return 0.5
	}
}

// InheritTraits creates child traits from two parents with genetic variation
func (p *PersonalityMatrix) InheritTraits(parent1, parent2 *PersonalityMatrix, mutationRate float64) *Traits {
	child := &Traits{}

	// Each trait is inherited from one parent randomly, with possible mutation
	child.Openness = inheritTrait(parent1.Traits.Openness, parent2.Traits.Openness, mutationRate)
	child.Conscientiousness = inheritTrait(parent1.Traits.Conscientiousness, parent2.Traits.Conscientiousness, mutationRate)
	child.Extraversion = inheritTrait(parent1.Traits.Extraversion, parent2.Traits.Extraversion, mutationRate)
	child.Agreeableness = inheritTrait(parent1.Traits.Agreeableness, parent2.Traits.Agreeableness, mutationRate)
	child.Neuroticism = inheritTrait(parent1.Traits.Neuroticism, parent2.Traits.Neuroticism, mutationRate)
	child.Playfulness = inheritTrait(parent1.Traits.Playfulness, parent2.Traits.Playfulness, mutationRate)
	child.Independence = inheritTrait(parent1.Traits.Independence, parent2.Traits.Independence, mutationRate)
	child.Loyalty = inheritTrait(parent1.Traits.Loyalty, parent2.Traits.Loyalty, mutationRate)
	child.Intelligence = inheritTrait(parent1.Traits.Intelligence, parent2.Traits.Intelligence, mutationRate)
	child.EnergyLevel = inheritTrait(parent1.Traits.EnergyLevel, parent2.Traits.EnergyLevel, mutationRate)
	child.Affectionate = inheritTrait(parent1.Traits.Affectionate, parent2.Traits.Affectionate, mutationRate)
	child.Curiosity = inheritTrait(parent1.Traits.Curiosity, parent2.Traits.Curiosity, mutationRate)
	child.Adaptability = inheritTrait(parent1.Traits.Adaptability, parent2.Traits.Adaptability, mutationRate)
	child.Vocalization = inheritTrait(parent1.Traits.Vocalization, parent2.Traits.Vocalization, mutationRate)
	child.Territoriality = inheritTrait(parent1.Traits.Territoriality, parent2.Traits.Territoriality, mutationRate)

	child.Clamp()
	return child
}

// TakeSnapshot saves the current trait values for history
func (p *PersonalityMatrix) TakeSnapshot(gameTime float64) {
	// Create a copy of current traits
	traitsCopy := *p.Traits

	snapshot := TraitSnapshot{
		Timestamp: gameTime,
		Traits:    &traitsCopy,
	}

	p.TraitHistory = append(p.TraitHistory, snapshot)

	// Keep only last 100 snapshots to manage memory
	if len(p.TraitHistory) > 100 {
		p.TraitHistory = p.TraitHistory[1:]
	}
}

// GetPersonalityDescription returns a text description of the personality
func (p *PersonalityMatrix) GetPersonalityDescription() string {
	traits := p.Traits

	// Determine dominant traits
	description := "This pet is "

	if traits.Playfulness > 0.7 {
		description += "very playful, "
	} else if traits.Playfulness < 0.3 {
		description += "calm and reserved, "
	}

	if traits.Affectionate > 0.7 {
		description += "affectionate, "
	} else if traits.Independence > 0.7 {
		description += "independent, "
	}

	if traits.Intelligence > 0.7 {
		description += "highly intelligent, "
	}

	if traits.EnergyLevel > 0.7 {
		description += "energetic, "
	} else if traits.EnergyLevel < 0.3 {
		description += "low-energy, "
	}

	if traits.Loyalty > 0.7 {
		description += "loyal, "
	}

	if traits.Curiosity > 0.7 {
		description += "curious, "
	}

	// Clean up trailing comma
	if len(description) > 13 {
		description = description[:len(description)-2] + "."
	} else {
		description += "well-balanced."
	}

	return description
}

// Helper function to inherit a single trait with mutation
func inheritTrait(parent1Val, parent2Val, mutationRate float64) float64 {
	// Randomly choose one parent's trait
	var baseValue float64
	if rand.Float64() < 0.5 {
		baseValue = parent1Val
	} else {
		baseValue = parent2Val
	}

	// Apply mutation
	if rand.Float64() < mutationRate {
		mutation := (rand.Float64() - 0.5) * 0.2 // +/- 10% mutation
		baseValue += mutation
	}

	return clamp(baseValue, 0.0, 1.0)
}

// Helper function to clamp values
func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}
