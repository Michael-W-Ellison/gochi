package interaction

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// InteractionContext contains all the state needed to process an interaction
type InteractionContext struct {
	// Core systems
	Vitals      *biology.VitalStats
	Emotions    *ai.EmotionState
	Personality *ai.PersonalityMatrix
	Needs       *simulation.NeedsManager
	Memory      *ai.MemorySystem
	Learning    *ai.LearningSystem

	// Current pet state
	Energy      float64
	CurrentMood float64
	Fatigue     float64
	Age         int // Age in days

	// User state
	UserID         types.UserID
	UserSkillLevel float64 // 0-1, how skilled the user is
	BondStrength   float64 // 0-1, relationship strength
}

// InteractionResult contains the outcome of processing an interaction
type InteractionResult struct {
	Success        bool
	Effectiveness  float64 // 0-1, how effective the interaction was
	Feedback       string  // Message to display to user
	VitalChanges   map[string]float64
	EmotionChanges map[string]float64
	NeedChanges    map[string]float64
	SkillProgress  *TrainingSession
	MemoryCreated  bool
	Warnings       []string // Any warnings (e.g., "Pet is too tired")
}

// InteractionProcessor handles all user interactions with the pet
type InteractionProcessor struct {
	mu sync.RWMutex

	// Skill tracking for training
	skills map[SkillType]*Skill

	// Interaction history for learning
	recentInteractions []InteractionRecord
	maxHistorySize     int

	// Random source for variability
	rand *rand.Rand
}

// InteractionRecord tracks past interactions for pattern learning
type InteractionRecord struct {
	Type      types.InteractionType
	Timestamp time.Time
	Success   bool
	Context   map[string]float64 // Relevant state at time of interaction
}

// NewInteractionProcessor creates a new interaction processor
func NewInteractionProcessor() *InteractionProcessor {
	return &InteractionProcessor{
		skills:             make(map[SkillType]*Skill),
		recentInteractions: make([]InteractionRecord, 0),
		maxHistorySize:     100,
		rand:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ProcessInteraction is the main entry point for handling user interactions
func (ip *InteractionProcessor) ProcessInteraction(
	interactionType types.InteractionType,
	intensity float64,
	context *InteractionContext,
	itemData interface{}, // Type depends on interaction (FoodType, ToyType, etc.)
) *InteractionResult {
	ip.mu.Lock()
	defer ip.mu.Unlock()

	result := &InteractionResult{
		Success:        true,
		VitalChanges:   make(map[string]float64),
		EmotionChanges: make(map[string]float64),
		NeedChanges:    make(map[string]float64),
		Warnings:       make([]string, 0),
	}

	// Check if pet is in a state to receive interaction
	if !ip.canInteract(context, interactionType, result) {
		result.Success = false
		return result
	}

	// Process the specific interaction type
	switch interactionType {
	case types.InteractionFeeding:
		ip.processFeeding(context, intensity, itemData, result)
	case types.InteractionPetting:
		ip.processPetting(context, intensity, result)
	case types.InteractionPlaying:
		ip.processPlaying(context, intensity, itemData, result)
	case types.InteractionTraining:
		ip.processTraining(context, intensity, itemData, result)
	case types.InteractionGrooming:
		ip.processGrooming(context, intensity, itemData, result)
	case types.InteractionMedicalCare:
		ip.processMedicalCare(context, intensity, itemData, result)
	case types.InteractionEnvironmentalEnrichment:
		ip.processEnvironmentalEnrichment(context, intensity, result)
	case types.InteractionSocialIntroduction:
		ip.processSocialIntroduction(context, intensity, result)
	case types.InteractionDiscipline:
		ip.processDiscipline(context, intensity, result)
	case types.InteractionRewards:
		ip.processRewards(context, intensity, result)
	}

	// Calculate overall effectiveness based on personality and mood
	result.Effectiveness = ip.calculateEffectiveness(context, interactionType, result)

	// Record memory of the interaction
	if result.Success && result.Effectiveness > 0.3 {
		ip.createMemory(context, interactionType, result)
		result.MemoryCreated = true
	}

	// Record for learning system
	ip.recordInteraction(interactionType, result.Success, context)

	return result
}

// canInteract checks if the pet can receive this type of interaction
func (ip *InteractionProcessor) canInteract(
	context *InteractionContext,
	interactionType types.InteractionType,
	result *InteractionResult,
) bool {
	// Check fatigue
	if context.Fatigue > 0.9 {
		result.Warnings = append(result.Warnings, "Pet is too exhausted for interaction")
		result.Feedback = "Your pet is too tired and needs rest."
		return false
	}

	// Check specific conditions for certain interactions
	switch interactionType {
	case types.InteractionPlaying:
		if context.Energy < 0.2 {
			result.Warnings = append(result.Warnings, "Pet has too little energy to play")
			result.Feedback = "Your pet is too low on energy to play right now."
			return false
		}
	case types.InteractionTraining:
		if context.Energy < 0.3 || context.Fatigue > 0.7 {
			result.Warnings = append(result.Warnings, "Pet is not in optimal state for training")
			result.Feedback = "Your pet needs to be well-rested and energized for training."
			return false
		}
	}

	return true
}

// processFeeding handles feeding interactions
func (ip *InteractionProcessor) processFeeding(
	context *InteractionContext,
	intensity float64,
	itemData interface{},
	result *InteractionResult,
) {
	var foodType FoodType
	if itemData != nil {
		if ft, ok := itemData.(FoodType); ok {
			foodType = ft
		} else {
			foodType = FoodTypeBasicKibble // Default
		}
	} else {
		foodType = FoodTypeBasicKibble
	}

	props := GetFoodProperties(foodType)

	// Apply nutritional effects
	result.VitalChanges["Nutrition"] = props.NutritionValue * intensity
	result.VitalChanges["Hydration"] = props.HydrationValue * intensity
	result.VitalChanges["Energy"] = props.EnergyBoost * intensity
	result.VitalChanges["Health"] = props.HealthImpact * intensity * 0.1 // Long-term effect
	result.VitalChanges["Happiness"] = props.HappinessBoost * intensity * 0.5

	// Update needs
	result.NeedChanges["Hunger"] = props.NutritionValue * intensity * 0.8
	result.NeedChanges["Thirst"] = props.HydrationValue * intensity * 0.8

	// Emotional response based on food quality and personality
	if props.HappinessBoost > 0.6 {
		result.EmotionChanges["Joy"] = 0.3 * intensity
		result.EmotionChanges["Contentment"] = 0.2 * intensity
	}

	// Generate feedback
	if props.HappinessBoost > 0.7 {
		result.Feedback = fmt.Sprintf("Your pet absolutely loves the %s! They're eating with gusto.", props.Name)
	} else if props.HappinessBoost > 0.4 {
		result.Feedback = fmt.Sprintf("Your pet enjoys the %s and eats contentedly.", props.Name)
	} else {
		result.Feedback = fmt.Sprintf("Your pet eats the %s, though they seem unexcited.", props.Name)
	}

	// Warning for unhealthy food
	if props.HealthImpact < 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s may have negative long-term health effects", props.Name))
	}
}

// processPetting handles petting/affection interactions
func (ip *InteractionProcessor) processPetting(
	context *InteractionContext,
	intensity float64,
	result *InteractionResult,
) {
	// Petting has strong effects on affection and stress
	affectionModifier := 1.0

	// Personality affects response to petting
	if context.Personality != nil {
		// Affectionate pets respond better
		affectionModifier += context.Personality.Traits.Affectionate * 0.5
		// Independent pets respond less
		affectionModifier -= context.Personality.Traits.Independence * 0.3
	}

	result.VitalChanges["Happiness"] = 0.3 * intensity * affectionModifier
	result.VitalChanges["Stress"] = -0.4 * intensity * affectionModifier

	result.NeedChanges["Affection"] = 0.6 * intensity * affectionModifier
	result.NeedChanges["Social"] = 0.3 * intensity

	result.EmotionChanges["Affection"] = 0.5 * intensity * affectionModifier
	result.EmotionChanges["Contentment"] = 0.3 * intensity
	result.EmotionChanges["Joy"] = 0.2 * intensity

	// Bond strength affects effectiveness
	bondBonus := context.BondStrength * 0.3

	// Generate feedback based on personality
	if context.Personality != nil && context.Personality.Traits.Affectionate > 0.7 {
		result.Feedback = "Your pet melts into your touch, purring with contentment. They clearly adore the affection!"
	} else if context.Personality != nil && context.Personality.Traits.Independence > 0.7 {
		result.Feedback = "Your pet tolerates the petting, enjoying it in their own reserved way."
	} else {
		result.Feedback = "Your pet enjoys the affection, leaning into your hand."
	}

	// Apply bond bonus
	for key := range result.VitalChanges {
		result.VitalChanges[key] *= (1.0 + bondBonus)
	}
}

// processPlaying handles play interactions
func (ip *InteractionProcessor) processPlaying(
	context *InteractionContext,
	intensity float64,
	itemData interface{},
	result *InteractionResult,
) {
	var toyType ToyType
	if itemData != nil {
		if tt, ok := itemData.(ToyType); ok {
			toyType = tt
		} else {
			toyType = ToyTypeBall // Default
		}
	} else {
		toyType = ToyTypeBall
	}

	props := GetToyProperties(toyType)

	// Personality affects play enjoyment
	playModifier := 1.0
	if context.Personality != nil {
		playModifier += context.Personality.Traits.Playfulness * 0.5
		playModifier += context.Personality.Traits.EnergyLevel * 0.3
	}

	// Play drains energy but increases happiness
	result.VitalChanges["Energy"] = -props.EnergyDrain * intensity
	result.VitalChanges["Happiness"] = props.FunFactor * intensity * playModifier
	result.VitalChanges["Fatigue"] = props.PhysicalExercise * intensity * 0.5

	// Update needs
	result.NeedChanges["Exercise"] = props.PhysicalExercise * intensity * 0.7
	result.NeedChanges["MentalStimulation"] = props.MentalStimulation * intensity * 0.6
	result.NeedChanges["Social"] = props.SocialEngagement * intensity * 0.5
	result.NeedChanges["Affection"] = props.SocialEngagement * intensity * 0.3

	// Emotional effects
	result.EmotionChanges["Joy"] = props.FunFactor * intensity * 0.6
	result.EmotionChanges["Excitement"] = props.PhysicalExercise * intensity * 0.5
	result.EmotionChanges["Contentment"] = 0.2 * intensity

	// Generate feedback based on toy and engagement
	funLevel := props.FunFactor * playModifier
	if funLevel > 0.8 {
		result.Feedback = fmt.Sprintf("Your pet is having a blast with the %s! They're full of energy and joy!", props.Name)
	} else if funLevel > 0.5 {
		result.Feedback = fmt.Sprintf("Your pet plays happily with the %s, clearly enjoying themselves.", props.Name)
	} else {
		result.Feedback = fmt.Sprintf("Your pet plays with the %s, but seems only mildly entertained.", props.Name)
	}

	// Warn if pet is getting too tired
	if context.Energy-props.EnergyDrain*intensity < 0.2 {
		result.Warnings = append(result.Warnings, "Pet is getting exhausted from play")
	}
}

// processTraining handles training interactions
func (ip *InteractionProcessor) processTraining(
	context *InteractionContext,
	intensity float64,
	itemData interface{},
	result *InteractionResult,
) {
	var skillType SkillType
	if itemData != nil {
		if st, ok := itemData.(SkillType); ok {
			skillType = st
		} else {
			skillType = SkillObedience // Default
		}
	} else {
		skillType = SkillObedience
	}

	// Get or create skill
	skill := ip.getOrCreateSkill(skillType)

	// Calculate training success based on multiple factors
	baseSuccess := 0.5

	// Intelligence and focus improve training
	if context.Personality != nil {
		baseSuccess += context.Personality.Traits.Intelligence * 0.3
		baseSuccess -= context.Personality.Traits.Independence * 0.1 // Independent pets harder to train
	}

	// Mood and energy affect training
	baseSuccess += (context.CurrentMood - 0.5) * 0.2
	baseSuccess += (context.Energy - 0.5) * 0.1

	// User skill level matters
	baseSuccess += context.UserSkillLevel * 0.2

	// Bond strength helps
	baseSuccess += context.BondStrength * 0.15

	// Add some randomness
	successRoll := ip.rand.Float64()
	success := successRoll < baseSuccess

	// Calculate experience gain
	experienceGain := 0.1 * intensity
	if success {
		experienceGain *= 1.5
	}

	// Update skill
	skill.Experience += experienceGain
	oldLevel := skill.Level
	skill.Level = ip.calculateSkillLevel(skill.Experience)
	skill.LastTrained = time.Now().Unix()

	// Training costs energy and mental effort
	result.VitalChanges["Energy"] = -0.2 * intensity
	result.VitalChanges["Fatigue"] = 0.15 * intensity
	result.NeedChanges["MentalStimulation"] = 0.5 * intensity

	if success {
		result.VitalChanges["Happiness"] = 0.2 * intensity
		result.EmotionChanges["Joy"] = 0.3 * intensity
		result.EmotionChanges["Contentment"] = 0.2 * intensity
		result.NeedChanges["Affection"] = 0.2 * intensity

		if skill.Level > oldLevel {
			result.Feedback = fmt.Sprintf("Success! Your pet has improved their %s skill! (Level %.1f%%)", skillType.String(), skill.Level*100)
		} else {
			result.Feedback = fmt.Sprintf("Great job! Your pet successfully completed the %s training.", skillType.String())
		}
	} else {
		result.VitalChanges["Stress"] = 0.1 * intensity
		result.EmotionChanges["Frustration"] = 0.2 * intensity
		result.Feedback = fmt.Sprintf("Your pet struggled with the %s training but learned a little. Keep practicing!", skillType.String())
	}

	// Create training session result
	result.SkillProgress = &TrainingSession{
		Skill:          skillType,
		Success:        success,
		ExperienceGain: experienceGain,
		LevelChange:    skill.Level - oldLevel,
		EffortRequired: intensity,
		PetEngagement:  baseSuccess,
	}

	// Recommend next steps
	if skill.Level > 0.8 {
		result.SkillProgress.RecommendedNext = fmt.Sprintf("Your pet has mastered %s! Try a new skill.", skillType.String())
	} else if success {
		result.SkillProgress.RecommendedNext = "Keep practicing this skill to improve mastery."
	} else {
		result.SkillProgress.RecommendedNext = "Try when your pet is more energized and in a better mood."
	}
}

// processGrooming handles grooming interactions
func (ip *InteractionProcessor) processGrooming(
	context *InteractionContext,
	intensity float64,
	itemData interface{},
	result *InteractionResult,
) {
	var toolType GroomingToolType
	if itemData != nil {
		if gt, ok := itemData.(GroomingToolType); ok {
			toolType = gt
		} else {
			toolType = GroomingToolBrush // Default
		}
	} else {
		toolType = GroomingToolBrush
	}

	props := GetGroomingToolProperties(toolType)

	// Apply grooming effects
	result.VitalChanges["Cleanliness"] = props.CleanlinessBoost * intensity
	result.VitalChanges["Health"] = props.HealthBoost * intensity * 0.5
	result.VitalChanges["Stress"] = props.StressImpact * intensity

	// Comfort level affects happiness
	if props.ComfortLevel > 0.6 {
		result.VitalChanges["Happiness"] = props.ComfortLevel * intensity * 0.4
		result.EmotionChanges["Contentment"] = 0.3 * intensity
	} else {
		result.VitalChanges["Happiness"] = -0.1 * intensity
		result.EmotionChanges["Discomfort"] = 0.2 * intensity
	}

	result.NeedChanges["Cleanliness"] = props.CleanlinessBoost * intensity * 0.8
	result.NeedChanges["Affection"] = 0.2 * intensity // Grooming is bonding time

	// User skill affects outcome
	skillModifier := 0.5 + (context.UserSkillLevel * 0.5)
	if context.UserSkillLevel < props.SkillLevelRequired {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Using %s requires more skill for best results", props.Name))
		skillModifier *= 0.7
	}

	// Apply skill modifier
	for key := range result.VitalChanges {
		result.VitalChanges[key] *= skillModifier
	}

	// Generate feedback
	if props.ComfortLevel > 0.7 {
		result.Feedback = fmt.Sprintf("Your pet relaxes as you groom them with the %s.", props.Name)
	} else if props.StressImpact > 0.4 {
		result.Feedback = fmt.Sprintf("Your pet endures the %s, though they seem uncomfortable.", props.Name)
	} else {
		result.Feedback = fmt.Sprintf("You groom your pet with the %s. They're now much cleaner!", props.Name)
	}
}

// processMedicalCare handles medical care interactions
func (ip *InteractionProcessor) processMedicalCare(
	context *InteractionContext,
	intensity float64,
	itemData interface{},
	result *InteractionResult,
) {
	var itemType MedicalItemType
	if itemData != nil {
		if mt, ok := itemData.(MedicalItemType); ok {
			itemType = mt
		} else {
			itemType = MedicalItemVitamins // Default
		}
	} else {
		itemType = MedicalItemVitamins
	}

	props := GetMedicalItemProperties(itemType)

	// Check if user has required skill
	if props.RequiresSkill && context.UserSkillLevel < 0.6 {
		result.Success = false
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s requires professional knowledge to use safely", props.Name))
		result.Feedback = fmt.Sprintf("You're not confident enough to use %s safely. Consult a professional.", props.Name)
		return
	}

	// Apply medical effects
	result.VitalChanges["Health"] = props.HealthBoost * intensity
	result.VitalChanges["Stress"] = props.StressImpact * intensity

	result.NeedChanges["MedicalCare"] = props.HealthBoost * intensity * 0.9

	// Emotional response
	if props.StressImpact > 0.3 {
		result.EmotionChanges["Fear"] = 0.3 * intensity
		result.EmotionChanges["Discomfort"] = 0.2 * intensity
	} else {
		result.EmotionChanges["Relief"] = 0.4 * intensity
		result.EmotionChanges["Contentment"] = 0.2 * intensity
	}

	// Generate feedback
	if props.HealthBoost > 0.5 {
		result.Feedback = fmt.Sprintf("You administer %s to your pet. They should feel better soon.", props.Name)
	} else if props.Preventive {
		result.Feedback = fmt.Sprintf("You give your pet %s to help keep them healthy.", props.Name)
	} else {
		result.Feedback = fmt.Sprintf("You provide %s care to your pet.", props.Name)
	}

	if props.StressImpact > 0.4 {
		result.Warnings = append(result.Warnings, "Medical treatment was stressful for your pet. Extra comfort needed.")
	}
}

// processEnvironmentalEnrichment handles environmental enrichment
func (ip *InteractionProcessor) processEnvironmentalEnrichment(
	context *InteractionContext,
	intensity float64,
	result *InteractionResult,
) {
	// Environmental enrichment provides mental stimulation and exploration
	curiosityModifier := 1.0
	if context.Personality != nil {
		curiosityModifier += context.Personality.Traits.Curiosity * 0.5
		curiosityModifier += context.Personality.Traits.Intelligence * 0.3
	}

	result.VitalChanges["Happiness"] = 0.3 * intensity * curiosityModifier
	result.VitalChanges["Stress"] = -0.2 * intensity

	result.NeedChanges["MentalStimulation"] = 0.7 * intensity * curiosityModifier
	result.NeedChanges["Exploration"] = 0.6 * intensity * curiosityModifier
	result.NeedChanges["Exercise"] = 0.3 * intensity

	result.EmotionChanges["Excitement"] = 0.4 * intensity
	result.EmotionChanges["Curiosity"] = 0.5 * intensity
	result.EmotionChanges["Joy"] = 0.3 * intensity

	if curiosityModifier > 1.3 {
		result.Feedback = "Your pet explores the enriched environment with great enthusiasm and curiosity!"
	} else {
		result.Feedback = "Your pet investigates the new environmental features with interest."
	}
}

// processSocialIntroduction handles social introductions
func (ip *InteractionProcessor) processSocialIntroduction(
	context *InteractionContext,
	intensity float64,
	result *InteractionResult,
) {
	// Social introductions satisfy social needs but can be stressful
	socializationModifier := 1.0
	if context.Personality != nil {
		// Extraverted pets enjoy it more
		socializationModifier += (context.Personality.Traits.Extraversion - 0.5) * 0.6
		// Adaptable pets handle it better
		socializationModifier += context.Personality.Traits.Adaptability * 0.3
	}

	// Social skill level (from training)
	socialSkill := ip.getSkillLevel(SkillSocialization)
	socializationModifier += socialSkill * 0.4

	result.NeedChanges["Social"] = 0.7 * intensity * socializationModifier
	result.NeedChanges["MentalStimulation"] = 0.3 * intensity

	if socializationModifier > 1.0 {
		// Positive social interaction
		result.VitalChanges["Happiness"] = 0.4 * intensity
		result.VitalChanges["Stress"] = -0.1 * intensity
		result.EmotionChanges["Joy"] = 0.4 * intensity
		result.EmotionChanges["Excitement"] = 0.3 * intensity
		result.Feedback = "Your pet has a wonderful time meeting new friends! They're naturally social."
	} else {
		// More stressful social interaction
		result.VitalChanges["Happiness"] = 0.2 * intensity
		result.VitalChanges["Stress"] = 0.3 * intensity
		result.EmotionChanges["Anxiety"] = 0.4 * intensity
		result.EmotionChanges["Curiosity"] = 0.2 * intensity
		result.Feedback = "Your pet meets someone new. They seem a bit nervous but curious."
		result.Warnings = append(result.Warnings, "Social interactions can be stressful. Monitor your pet's stress levels.")
	}
}

// processDiscipline handles discipline interactions
func (ip *InteractionProcessor) processDiscipline(
	context *InteractionContext,
	intensity float64,
	result *InteractionResult,
) {
	// Discipline should be used sparingly and appropriately
	obedienceSkill := ip.getSkillLevel(SkillObedience)

	// Discipline is less stressful if pet understands obedience
	stressModifier := 1.0 - (obedienceSkill * 0.5)

	result.VitalChanges["Happiness"] = -0.2 * intensity
	result.VitalChanges["Stress"] = 0.3 * intensity * stressModifier

	result.EmotionChanges["Fear"] = 0.3 * intensity * stressModifier
	result.EmotionChanges["Sadness"] = 0.2 * intensity

	// Discipline can improve obedience if done right
	if context.UserSkillLevel > 0.6 && intensity < 0.7 {
		// Appropriate discipline
		result.Feedback = "Your pet understands the correction and will learn from this."

		// Small boost to obedience training
		if skill := ip.getOrCreateSkill(SkillObedience); skill != nil {
			skill.Experience += 0.05 * intensity
			skill.Level = ip.calculateSkillLevel(skill.Experience)
		}
	} else if intensity > 0.8 {
		// Too harsh
		result.Warnings = append(result.Warnings, "Discipline was too harsh and may damage your bond")
		result.Feedback = "Your pet seems frightened and upset. Gentler methods work better."
		result.VitalChanges["Stress"] *= 1.5
		result.EmotionChanges["Fear"] *= 1.5
	} else {
		// Ineffective
		result.Feedback = "Your pet doesn't seem to understand the discipline."
	}
}

// processRewards handles reward interactions
func (ip *InteractionProcessor) processRewards(
	context *InteractionContext,
	intensity float64,
	result *InteractionResult,
) {
	// Rewards are positive reinforcement
	result.VitalChanges["Happiness"] = 0.4 * intensity
	result.VitalChanges["Stress"] = -0.2 * intensity

	result.NeedChanges["Affection"] = 0.4 * intensity
	result.NeedChanges["Social"] = 0.2 * intensity

	result.EmotionChanges["Joy"] = 0.5 * intensity
	result.EmotionChanges["Excitement"] = 0.4 * intensity
	result.EmotionChanges["Affection"] = 0.3 * intensity

	// Rewards reinforce recent training
	mostRecentSkill := ip.getMostRecentlyTrainedSkill()
	if mostRecentSkill != nil {
		mostRecentSkill.Experience += 0.08 * intensity
		mostRecentSkill.Level = ip.calculateSkillLevel(mostRecentSkill.Experience)
		result.Feedback = fmt.Sprintf("Your pet is thrilled with the reward! It reinforces their %s training.", mostRecentSkill.Type.String())
	} else {
		result.Feedback = "Your pet happily accepts the reward, tail wagging with joy!"
	}
}

// calculateEffectiveness determines how effective an interaction was
func (ip *InteractionProcessor) calculateEffectiveness(
	context *InteractionContext,
	interactionType types.InteractionType,
	result *InteractionResult,
) float64 {
	if !result.Success {
		return 0.0
	}

	// Base effectiveness from changes
	totalChange := 0.0
	for _, change := range result.VitalChanges {
		totalChange += math.Abs(change)
	}
	for _, change := range result.EmotionChanges {
		totalChange += math.Abs(change) * 0.5 // Emotions count for less
	}
	for _, change := range result.NeedChanges {
		totalChange += math.Abs(change) * 0.7
	}

	effectiveness := math.Min(totalChange/3.0, 1.0)

	// Mood affects perceived effectiveness
	if context.CurrentMood > 0.6 {
		effectiveness *= 1.2
	} else if context.CurrentMood < 0.4 {
		effectiveness *= 0.8
	}

	// Bond strength helps
	effectiveness *= (0.8 + context.BondStrength*0.4)

	return math.Min(effectiveness, 1.0)
}

// createMemory creates a memory of the interaction
func (ip *InteractionProcessor) createMemory(
	context *InteractionContext,
	interactionType types.InteractionType,
	result *InteractionResult,
) {
	if context.Memory == nil {
		return
	}

	// Determine emotional valence of memory
	emotionalImpact := 0.0
	for emotion, change := range result.EmotionChanges {
		if emotion == "Joy" || emotion == "Excitement" || emotion == "Contentment" || emotion == "Affection" {
			emotionalImpact += change
		} else {
			emotionalImpact -= change
		}
	}

	strength := result.Effectiveness * (0.5 + math.Abs(emotionalImpact)*0.5)

	// Convert interaction type to types.InteractionType for memory
	context.Memory.RecordInteraction(
		interactionType,
		0.0, // gameTime - would need to be passed in context
		strength,
		fmt.Sprintf("%.2f", emotionalImpact), // emotion as string
	)
}

// recordInteraction saves interaction for pattern learning
func (ip *InteractionProcessor) recordInteraction(
	interactionType types.InteractionType,
	success bool,
	context *InteractionContext,
) {
	record := InteractionRecord{
		Type:      interactionType,
		Timestamp: time.Now(),
		Success:   success,
		Context: map[string]float64{
			"energy":  context.Energy,
			"mood":    context.CurrentMood,
			"fatigue": context.Fatigue,
		},
	}

	ip.recentInteractions = append(ip.recentInteractions, record)

	// Keep history size manageable
	if len(ip.recentInteractions) > ip.maxHistorySize {
		ip.recentInteractions = ip.recentInteractions[1:]
	}
}

// Skill management methods

func (ip *InteractionProcessor) getOrCreateSkill(skillType SkillType) *Skill {
	if skill, exists := ip.skills[skillType]; exists {
		return skill
	}

	skill := &Skill{
		Type:       skillType,
		Level:      0.0,
		Experience: 0.0,
		Decay:      0.001, // Skills slowly decay without practice
	}
	ip.skills[skillType] = skill
	return skill
}

func (ip *InteractionProcessor) getSkillLevel(skillType SkillType) float64 {
	if skill, exists := ip.skills[skillType]; exists {
		return skill.Level
	}
	return 0.0
}

func (ip *InteractionProcessor) getMostRecentlyTrainedSkill() *Skill {
	var mostRecent *Skill
	var latestTime int64 = 0

	for _, skill := range ip.skills {
		if skill.LastTrained > latestTime {
			latestTime = skill.LastTrained
			mostRecent = skill
		}
	}

	return mostRecent
}

func (ip *InteractionProcessor) calculateSkillLevel(experience float64) float64 {
	// Logarithmic progression: harder to level up at higher levels
	// Level 0.5 requires ~5 exp, 0.8 requires ~15 exp, 1.0 requires ~30 exp
	level := math.Log1p(experience) / math.Log1p(30.0)
	return math.Min(level, 1.0)
}

// UpdateSkillDecay should be called periodically to decay skills over time
func (ip *InteractionProcessor) UpdateSkillDecay(deltaTime float64) {
	ip.mu.Lock()
	defer ip.mu.Unlock()

	for _, skill := range ip.skills {
		skill.Experience -= skill.Decay * deltaTime
		if skill.Experience < 0 {
			skill.Experience = 0
		}
		skill.Level = ip.calculateSkillLevel(skill.Experience)
	}
}

// GetAllSkills returns a copy of all skills for inspection
func (ip *InteractionProcessor) GetAllSkills() map[SkillType]*Skill {
	ip.mu.RLock()
	defer ip.mu.RUnlock()

	skillsCopy := make(map[SkillType]*Skill)
	for k, v := range ip.skills {
		skillCopy := *v
		skillsCopy[k] = &skillCopy
	}
	return skillsCopy
}

// GetInteractionHistory returns recent interaction history
func (ip *InteractionProcessor) GetInteractionHistory() []InteractionRecord {
	ip.mu.RLock()
	defer ip.mu.RUnlock()

	history := make([]InteractionRecord, len(ip.recentInteractions))
	copy(history, ip.recentInteractions)
	return history
}
