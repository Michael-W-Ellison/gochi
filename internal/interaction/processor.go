package interaction

import (
	"fmt"
	"math"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// InteractionResult contains the outcome of an interaction
type InteractionResult struct {
	Success       bool                   `json:"success"`
	Message       string                 `json:"message"`
	Effects       map[string]float64     `json:"effects"`
	SkillGain     map[string]float64     `json:"skill_gain,omitempty"`
	MoodChange    string                 `json:"mood_change,omitempty"`
	ItemConsumed  string                 `json:"item_consumed,omitempty"`
	Feedback      InteractionFeedback    `json:"feedback"`
	Timestamp     time.Time              `json:"timestamp"`
}

// InteractionFeedback provides user-facing feedback
type InteractionFeedback struct {
	Animation     string   `json:"animation"`
	Sound         string   `json:"sound"`
	Particles     string   `json:"particles,omitempty"`
	PetReaction   string   `json:"pet_reaction"`
	Messages      []string `json:"messages"`
}

// InteractionContext provides context for processing interactions
type InteractionContext struct {
	// Pet state
	Health       float64
	Energy       float64
	Happiness    float64
	Hunger       float64
	Thirst       float64
	Cleanliness  float64
	Stress       float64
	Fatigue      float64

	// Personality traits
	Playfulness  float64
	Independence float64
	Affectionate float64
	Intelligence float64

	// Current state
	CurrentBehavior types.BehaviorState
	PetAge          float64
	IsSleeping      bool
	IsSick          bool

	// Environment
	TimeOfDay       float64
	IsOutdoors      bool
}

// InteractionProcessor handles all user interactions with pets
type InteractionProcessor struct {
	// Configuration
	Config ProcessorConfig

	// Statistics
	TotalInteractions   int
	InteractionCounts   map[types.InteractionType]int
	LastInteractionTime time.Time
	CooldownTimers      map[types.InteractionType]time.Time
}

// ProcessorConfig configures the interaction processor
type ProcessorConfig struct {
	BaseCooldownSeconds float64            // Base cooldown between interactions
	EffectMultiplier    float64            // Global effect multiplier
	CooldownOverrides   map[types.InteractionType]float64
}

// DefaultProcessorConfig returns default configuration
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		BaseCooldownSeconds: 1.0,
		EffectMultiplier:    1.0,
		CooldownOverrides: map[types.InteractionType]float64{
			types.InteractionFeeding:     5.0,
			types.InteractionMedicalCare: 30.0,
			types.InteractionTraining:    10.0,
		},
	}
}

// NewInteractionProcessor creates a new interaction processor
func NewInteractionProcessor() *InteractionProcessor {
	return &InteractionProcessor{
		Config:            DefaultProcessorConfig(),
		InteractionCounts: make(map[types.InteractionType]int),
		CooldownTimers:    make(map[types.InteractionType]time.Time),
	}
}

// NewInteractionProcessorWithConfig creates a processor with custom config
func NewInteractionProcessorWithConfig(config ProcessorConfig) *InteractionProcessor {
	return &InteractionProcessor{
		Config:            config,
		InteractionCounts: make(map[types.InteractionType]int),
		CooldownTimers:    make(map[types.InteractionType]time.Time),
	}
}

// Process handles an interaction and returns the result
func (ip *InteractionProcessor) Process(
	interactionType types.InteractionType,
	intensity float64,
	context InteractionContext,
) InteractionResult {
	result := InteractionResult{
		Timestamp: time.Now(),
		Effects:   make(map[string]float64),
		SkillGain: make(map[string]float64),
		Feedback: InteractionFeedback{
			Messages: make([]string, 0),
		},
	}

	// Check cooldown
	if !ip.canInteract(interactionType) {
		result.Success = false
		result.Message = "Please wait before doing that again"
		result.Feedback.PetReaction = "waiting"
		return result
	}

	// Check if pet is in a state that prevents interaction
	if context.IsSleeping && interactionType != types.InteractionMedicalCare {
		result.Success = false
		result.Message = "Your pet is sleeping"
		result.Feedback.PetReaction = "sleeping"
		return result
	}

	// Clamp intensity
	intensity = math.Max(0.0, math.Min(1.0, intensity))

	// Process based on interaction type
	switch interactionType {
	case types.InteractionFeeding:
		ip.processFeeding(&result, intensity, context)
	case types.InteractionPetting:
		ip.processPetting(&result, intensity, context)
	case types.InteractionPlaying:
		ip.processPlaying(&result, intensity, context)
	case types.InteractionTraining:
		ip.processTraining(&result, intensity, context)
	case types.InteractionGrooming:
		ip.processGrooming(&result, intensity, context)
	case types.InteractionMedicalCare:
		ip.processMedicalCare(&result, intensity, context)
	case types.InteractionRewards:
		ip.processRewards(&result, intensity, context)
	case types.InteractionDiscipline:
		ip.processDiscipline(&result, intensity, context)
	case types.InteractionEnvironmentalEnrichment:
		ip.processEnrichment(&result, intensity, context)
	case types.InteractionSocialIntroduction:
		ip.processSocialIntroduction(&result, intensity, context)
	default:
		result.Success = false
		result.Message = "Unknown interaction type"
		return result
	}

	// Apply global multiplier
	for key, value := range result.Effects {
		result.Effects[key] = value * ip.Config.EffectMultiplier
	}

	// Update statistics
	ip.TotalInteractions++
	ip.InteractionCounts[interactionType]++
	ip.LastInteractionTime = time.Now()
	ip.CooldownTimers[interactionType] = time.Now()

	return result
}

// canInteract checks if an interaction is allowed based on cooldown
func (ip *InteractionProcessor) canInteract(interactionType types.InteractionType) bool {
	lastTime, exists := ip.CooldownTimers[interactionType]
	if !exists {
		return true
	}

	cooldown := ip.Config.BaseCooldownSeconds
	if override, exists := ip.Config.CooldownOverrides[interactionType]; exists {
		cooldown = override
	}

	return time.Since(lastTime).Seconds() >= cooldown
}

// processFeeding handles feeding interactions
func (ip *InteractionProcessor) processFeeding(result *InteractionResult, intensity float64, ctx InteractionContext) {
	result.Success = true

	// Check if pet is hungry
	if ctx.Hunger > 0.8 {
		result.Message = "Your pet isn't very hungry right now"
		intensity *= 0.5
	} else {
		result.Message = "You fed your pet"
	}

	// Calculate effects based on hunger level
	hungerReduction := 0.3 * intensity
	energyGain := 0.15 * intensity
	happinessGain := 0.1 * intensity

	// Personality affects enjoyment
	if ctx.Affectionate > 0.6 {
		happinessGain *= 1.2
	}

	result.Effects["hunger"] = -hungerReduction
	result.Effects["energy"] = energyGain
	result.Effects["happiness"] = happinessGain
	result.Effects["nutrition"] = 0.25 * intensity

	// Feedback
	result.Feedback.Animation = "eating"
	result.Feedback.Sound = "munch"
	result.Feedback.PetReaction = "happy_eating"
	result.Feedback.Messages = append(result.Feedback.Messages, "Yum yum!")

	if ctx.Hunger < 0.3 {
		result.Feedback.Messages = append(result.Feedback.Messages, "That was delicious!")
		result.MoodChange = "satisfied"
	}
}

// processPetting handles petting interactions
func (ip *InteractionProcessor) processPetting(result *InteractionResult, intensity float64, ctx InteractionContext) {
	result.Success = true
	result.Message = "You petted your pet"

	// Base effects
	happinessGain := 0.15 * intensity
	stressReduction := 0.1 * intensity

	// Personality modifiers
	if ctx.Affectionate > 0.7 {
		happinessGain *= 1.5
		stressReduction *= 1.3
	} else if ctx.Independence > 0.7 {
		happinessGain *= 0.6
		result.Feedback.Messages = append(result.Feedback.Messages, "Your pet tolerates the attention")
	}

	// Stress affects receptiveness
	if ctx.Stress > 0.7 {
		stressReduction *= 1.5
		result.Feedback.Messages = append(result.Feedback.Messages, "That really helped calm them down")
	}

	result.Effects["happiness"] = happinessGain
	result.Effects["stress"] = -stressReduction
	result.Effects["affection_bond"] = 0.05 * intensity

	// Feedback
	result.Feedback.Animation = "being_petted"
	result.Feedback.Sound = "purr"
	result.Feedback.PetReaction = "content"
	result.MoodChange = "relaxed"
}

// processPlaying handles play interactions
func (ip *InteractionProcessor) processPlaying(result *InteractionResult, intensity float64, ctx InteractionContext) {
	// Check energy level
	if ctx.Energy < 0.2 {
		result.Success = false
		result.Message = "Your pet is too tired to play"
		result.Feedback.PetReaction = "tired"
		return
	}

	result.Success = true
	result.Message = "You played with your pet"

	// Calculate effects
	happinessGain := 0.2 * intensity
	energyDrain := 0.15 * intensity
	stressReduction := 0.08 * intensity

	// Playfulness affects enjoyment
	if ctx.Playfulness > 0.7 {
		happinessGain *= 1.4
		result.Feedback.Messages = append(result.Feedback.Messages, "Your pet is having a blast!")
	} else if ctx.Playfulness < 0.3 {
		happinessGain *= 0.7
		energyDrain *= 0.8
	}

	// Skill gains from play
	result.SkillGain["agility"] = 0.02 * intensity
	result.SkillGain["coordination"] = 0.01 * intensity

	result.Effects["happiness"] = happinessGain
	result.Effects["energy"] = -energyDrain
	result.Effects["stress"] = -stressReduction
	result.Effects["exercise"] = 0.2 * intensity

	// Feedback
	result.Feedback.Animation = "playing"
	result.Feedback.Sound = "playful_sounds"
	result.Feedback.Particles = "sparkles"
	result.Feedback.PetReaction = "excited"
	result.MoodChange = "playful"
}

// processTraining handles training interactions
func (ip *InteractionProcessor) processTraining(result *InteractionResult, intensity float64, ctx InteractionContext) {
	// Check energy and stress
	if ctx.Energy < 0.3 {
		result.Success = false
		result.Message = "Your pet is too tired for training"
		result.Feedback.PetReaction = "tired"
		return
	}

	if ctx.Stress > 0.7 {
		result.Success = false
		result.Message = "Your pet is too stressed for training"
		result.Feedback.PetReaction = "stressed"
		return
	}

	result.Success = true
	result.Message = "You trained your pet"

	// Calculate effects
	energyDrain := 0.2 * intensity
	stressGain := 0.05 * intensity

	// Intelligence affects learning speed
	learningMultiplier := 0.5 + ctx.Intelligence*0.5

	// Skill gains
	result.SkillGain["obedience"] = 0.03 * intensity * learningMultiplier
	result.SkillGain["intelligence"] = 0.02 * intensity * learningMultiplier
	result.SkillGain["focus"] = 0.02 * intensity * learningMultiplier

	result.Effects["energy"] = -energyDrain
	result.Effects["stress"] = stressGain
	result.Effects["mental_stimulation"] = 0.25 * intensity

	// Small happiness if successful
	if ctx.Intelligence > 0.5 {
		result.Effects["happiness"] = 0.05 * intensity
		result.Feedback.Messages = append(result.Feedback.Messages, "Good job! Quick learner!")
	}

	// Feedback
	result.Feedback.Animation = "training"
	result.Feedback.Sound = "training_complete"
	result.Feedback.PetReaction = "focused"
	result.MoodChange = "accomplished"
}

// processGrooming handles grooming interactions
func (ip *InteractionProcessor) processGrooming(result *InteractionResult, intensity float64, ctx InteractionContext) {
	result.Success = true
	result.Message = "You groomed your pet"

	// Calculate effects
	cleanlinessGain := 0.35 * intensity
	happinessChange := 0.05 * intensity

	// Some pets don't enjoy grooming
	if ctx.Independence > 0.7 {
		happinessChange *= 0.5
		result.Feedback.Messages = append(result.Feedback.Messages, "Your pet endures the grooming")
	} else if ctx.Affectionate > 0.6 {
		happinessChange *= 1.5
		result.Feedback.Messages = append(result.Feedback.Messages, "Your pet enjoys the attention")
	}

	result.Effects["cleanliness"] = cleanlinessGain
	result.Effects["happiness"] = happinessChange
	result.Effects["health"] = 0.02 * intensity // Slight health benefit

	// Feedback
	result.Feedback.Animation = "grooming"
	result.Feedback.Sound = "brush"
	result.Feedback.PetReaction = "being_groomed"
	result.MoodChange = "clean"
}

// processMedicalCare handles medical care interactions
func (ip *InteractionProcessor) processMedicalCare(result *InteractionResult, intensity float64, ctx InteractionContext) {
	result.Success = true
	result.Message = "You provided medical care"

	// Calculate effects
	healthGain := 0.25 * intensity
	stressGain := 0.1 * intensity // Medical care is stressful

	// Sick pets benefit more
	if ctx.IsSick || ctx.Health < 0.5 {
		healthGain *= 1.5
		result.Feedback.Messages = append(result.Feedback.Messages, "The treatment is helping!")
	}

	result.Effects["health"] = healthGain
	result.Effects["stress"] = stressGain
	result.Effects["happiness"] = -0.05 * intensity // Not enjoyable

	// Feedback
	result.Feedback.Animation = "medical_care"
	result.Feedback.Sound = "medical"
	result.Feedback.PetReaction = "uncomfortable"
	result.MoodChange = "recovering"
}

// processRewards handles reward/treat interactions
func (ip *InteractionProcessor) processRewards(result *InteractionResult, intensity float64, ctx InteractionContext) {
	result.Success = true
	result.Message = "You gave your pet a treat"

	// Calculate effects
	happinessGain := 0.25 * intensity
	hungerReduction := 0.05 * intensity

	// Rewards are especially effective after training
	if ctx.CurrentBehavior == types.BehaviorIdle {
		happinessGain *= 1.2
	}

	result.Effects["happiness"] = happinessGain
	result.Effects["hunger"] = -hungerReduction
	result.Effects["loyalty_bond"] = 0.03 * intensity

	// Feedback
	result.Feedback.Animation = "receiving_treat"
	result.Feedback.Sound = "happy_munch"
	result.Feedback.Particles = "hearts"
	result.Feedback.PetReaction = "delighted"
	result.Feedback.Messages = append(result.Feedback.Messages, "What a treat!")
	result.MoodChange = "happy"
}

// processDiscipline handles discipline interactions
func (ip *InteractionProcessor) processDiscipline(result *InteractionResult, intensity float64, ctx InteractionContext) {
	result.Success = true
	result.Message = "You disciplined your pet"

	// Calculate effects - discipline increases stress and reduces happiness
	stressGain := 0.15 * intensity
	happinessLoss := 0.1 * intensity

	// But can improve obedience
	result.SkillGain["obedience"] = 0.02 * intensity

	// Sensitive pets react more strongly
	if ctx.Affectionate > 0.7 {
		stressGain *= 1.3
		happinessLoss *= 1.3
		result.Feedback.Messages = append(result.Feedback.Messages, "Your pet looks hurt...")
	}

	result.Effects["stress"] = stressGain
	result.Effects["happiness"] = -happinessLoss
	result.Effects["loyalty_bond"] = -0.02 * intensity

	// Feedback
	result.Feedback.Animation = "scolded"
	result.Feedback.Sound = "sad_whimper"
	result.Feedback.PetReaction = "sad"
	result.MoodChange = "subdued"
}

// processEnrichment handles environmental enrichment
func (ip *InteractionProcessor) processEnrichment(result *InteractionResult, intensity float64, ctx InteractionContext) {
	result.Success = true
	result.Message = "You enriched your pet's environment"

	// Calculate effects
	happinessGain := 0.1 * intensity
	curiositySatisfaction := 0.2 * intensity

	// Curious pets benefit more
	curiosityBonus := 1.0
	if ctx.Intelligence > 0.6 {
		curiosityBonus = 1.3
		result.Feedback.Messages = append(result.Feedback.Messages, "Your pet is fascinated!")
	}

	result.Effects["happiness"] = happinessGain * curiosityBonus
	result.Effects["mental_stimulation"] = curiositySatisfaction * curiosityBonus
	result.Effects["exploration"] = 0.15 * intensity

	result.SkillGain["problem_solving"] = 0.02 * intensity
	result.SkillGain["curiosity"] = 0.01 * intensity

	// Feedback
	result.Feedback.Animation = "exploring"
	result.Feedback.Sound = "curious_sounds"
	result.Feedback.PetReaction = "curious"
	result.MoodChange = "interested"
}

// processSocialIntroduction handles introducing the pet to others
func (ip *InteractionProcessor) processSocialIntroduction(result *InteractionResult, intensity float64, ctx InteractionContext) {
	result.Success = true
	result.Message = "You introduced your pet to someone new"

	// Calculate effects based on personality
	socialSatisfaction := 0.15 * intensity
	stressChange := 0.0

	// Extroverted pets enjoy socialization more
	if ctx.Independence < 0.4 {
		socialSatisfaction *= 1.4
		result.Feedback.Messages = append(result.Feedback.Messages, "Your pet loves making new friends!")
	} else if ctx.Independence > 0.7 {
		socialSatisfaction *= 0.6
		stressChange = 0.05 * intensity
		result.Feedback.Messages = append(result.Feedback.Messages, "Your pet is a bit overwhelmed")
	}

	result.Effects["social_satisfaction"] = socialSatisfaction
	result.Effects["stress"] = stressChange
	result.Effects["happiness"] = 0.08 * intensity

	result.SkillGain["sociability"] = 0.02 * intensity

	// Feedback
	result.Feedback.Animation = "meeting"
	result.Feedback.Sound = "greeting"
	result.Feedback.PetReaction = "curious"
	result.MoodChange = "social"
}

// GetInteractionStats returns statistics about interactions
func (ip *InteractionProcessor) GetInteractionStats() map[string]interface{} {
	return map[string]interface{}{
		"total_interactions":    ip.TotalInteractions,
		"interaction_counts":    ip.InteractionCounts,
		"last_interaction_time": ip.LastInteractionTime,
	}
}

// GetCooldownRemaining returns remaining cooldown for an interaction type
func (ip *InteractionProcessor) GetCooldownRemaining(interactionType types.InteractionType) float64 {
	lastTime, exists := ip.CooldownTimers[interactionType]
	if !exists {
		return 0
	}

	cooldown := ip.Config.BaseCooldownSeconds
	if override, exists := ip.Config.CooldownOverrides[interactionType]; exists {
		cooldown = override
	}

	remaining := cooldown - time.Since(lastTime).Seconds()
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ResetCooldowns clears all cooldown timers
func (ip *InteractionProcessor) ResetCooldowns() {
	ip.CooldownTimers = make(map[types.InteractionType]time.Time)
}

// SetCooldownOverride sets a custom cooldown for an interaction type
func (ip *InteractionProcessor) SetCooldownOverride(interactionType types.InteractionType, seconds float64) {
	ip.Config.CooldownOverrides[interactionType] = seconds
}

// GetAvailableInteractions returns interactions not on cooldown
func (ip *InteractionProcessor) GetAvailableInteractions() []types.InteractionType {
	available := []types.InteractionType{
		types.InteractionFeeding,
		types.InteractionPetting,
		types.InteractionPlaying,
		types.InteractionTraining,
		types.InteractionGrooming,
		types.InteractionMedicalCare,
		types.InteractionRewards,
		types.InteractionDiscipline,
		types.InteractionEnvironmentalEnrichment,
		types.InteractionSocialIntroduction,
	}

	var result []types.InteractionType
	for _, it := range available {
		if ip.canInteract(it) {
			result = append(result, it)
		}
	}

	return result
}

// GetInteractionDescription returns a description of an interaction type
func GetInteractionDescription(interactionType types.InteractionType) string {
	descriptions := map[types.InteractionType]string{
		types.InteractionFeeding:                 "Feed your pet to reduce hunger and restore energy",
		types.InteractionPetting:                 "Pet your companion to increase happiness and reduce stress",
		types.InteractionPlaying:                 "Play with your pet to boost happiness and exercise",
		types.InteractionTraining:                "Train your pet to improve skills and obedience",
		types.InteractionGrooming:                "Groom your pet to maintain cleanliness and health",
		types.InteractionMedicalCare:             "Provide medical care to improve health",
		types.InteractionRewards:                 "Give treats to reward good behavior",
		types.InteractionDiscipline:              "Discipline to correct unwanted behavior",
		types.InteractionEnvironmentalEnrichment: "Add enrichment to stimulate curiosity",
		types.InteractionSocialIntroduction:      "Introduce your pet to others for socialization",
	}

	if desc, exists := descriptions[interactionType]; exists {
		return desc
	}
	return "Interact with your pet"
}

// ValidateInteraction checks if an interaction is valid for the current context
func ValidateInteraction(interactionType types.InteractionType, ctx InteractionContext) (bool, string) {
	switch interactionType {
	case types.InteractionPlaying:
		if ctx.Energy < 0.2 {
			return false, "Pet is too tired to play"
		}
		if ctx.IsSleeping {
			return false, "Pet is sleeping"
		}

	case types.InteractionTraining:
		if ctx.Energy < 0.3 {
			return false, "Pet is too tired for training"
		}
		if ctx.Stress > 0.7 {
			return false, "Pet is too stressed for training"
		}
		if ctx.IsSleeping {
			return false, "Pet is sleeping"
		}

	case types.InteractionFeeding:
		if ctx.Hunger > 0.9 {
			return false, "Pet is not hungry"
		}

	case types.InteractionGrooming:
		if ctx.Cleanliness > 0.95 {
			return false, "Pet is already clean"
		}
	}

	return true, ""
}

// CalculateEffectiveness returns how effective an interaction will be
func CalculateEffectiveness(interactionType types.InteractionType, ctx InteractionContext) float64 {
	effectiveness := 1.0

	// Energy affects most activities
	if ctx.Energy < 0.3 {
		effectiveness *= 0.7
	}

	// Stress reduces effectiveness
	if ctx.Stress > 0.6 {
		effectiveness *= 0.8
	}

	// Happiness affects receptiveness
	if ctx.Happiness < 0.3 {
		effectiveness *= 0.9
	}

	// Type-specific modifiers
	switch interactionType {
	case types.InteractionPetting:
		effectiveness *= (0.5 + ctx.Affectionate*0.5)

	case types.InteractionPlaying:
		effectiveness *= (0.5 + ctx.Playfulness*0.5)

	case types.InteractionTraining:
		effectiveness *= (0.5 + ctx.Intelligence*0.5)
	}

	return math.Max(0.1, math.Min(1.5, effectiveness))
}

// formatDuration formats a duration for display
func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.0f seconds", seconds)
	}
	return fmt.Sprintf("%.1f minutes", seconds/60)
}
