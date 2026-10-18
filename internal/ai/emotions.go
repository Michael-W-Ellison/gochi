package ai

import (
	"math"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// EmotionState represents the multi-dimensional emotional state of a pet
type EmotionState struct {
	// Primary emotions (0.0 to 1.0)
	Joy        float64 `json:"joy"`        // Happiness, contentment
	Sadness    float64 `json:"sadness"`    // Unhappiness, depression
	Anger      float64 `json:"anger"`      // Frustration, irritation
	Fear       float64 `json:"fear"`       // Anxiety, worry
	Excitement float64 `json:"excitement"` // Enthusiasm, anticipation
	Contentment float64 `json:"contentment"` // Peaceful satisfaction
	Affection  float64 `json:"affection"`  // Love, attachment
	Loneliness float64 `json:"loneliness"` // Feeling isolated

	// Current dominant emotion
	DominantEmotion string `json:"dominant_emotion"`

	// Emotional temperament (0.0 to 1.0)
	Resilience float64 `json:"resilience"` // How quickly negative emotions fade
	Volatility float64 `json:"volatility"` // How strongly stimuli affect emotions

	// Emotion history for tracking mood patterns
	MoodHistory []MoodSnapshot `json:"mood_history"`
	LastUpdate  time.Time
}

// MoodSnapshot records the overall mood at a point in time
type MoodSnapshot struct {
	Timestamp       time.Time `json:"timestamp"`
	MoodScore       float64   `json:"mood_score"`
	DominantEmotion string    `json:"dominant_emotion"`
}

// maxMoodHistory is the maximum number of mood snapshots retained
const maxMoodHistory = 100

// NewEmotionState creates a new emotion state with neutral/positive defaults
func NewEmotionState() *EmotionState {
	return &EmotionState{
		Joy:             0.6,
		Sadness:         0.1,
		Anger:           0.1,
		Fear:            0.1,
		Excitement:      0.4,
		Contentment:     0.5,
		Affection:       0.5,
		Loneliness:      0.2,
		DominantEmotion: "content",
		Resilience:      0.5,
		Volatility:      0.5,
		MoodHistory:     make([]MoodSnapshot, 0),
		LastUpdate:      time.Now(),
	}
}

// NewEmotionStateWithPersonality creates an emotion state whose baselines
// and temperament are derived from personality traits
func NewEmotionStateWithPersonality(traits *Traits) *EmotionState {
	e := NewEmotionState()

	// Low neuroticism means faster recovery and calmer reactions
	e.Resilience = clamp(1.0-traits.Neuroticism, 0.0, 1.0)
	e.Volatility = clamp(traits.Neuroticism, 0.0, 1.0)

	// Extraverts are less lonely and more excitable
	e.Loneliness = clamp(0.3-0.2*traits.Extraversion, 0.0, 1.0)
	e.Excitement = clamp(0.3+0.2*traits.Extraversion, 0.0, 1.0)

	e.updateDominantEmotion()
	return e
}

// Update processes emotional changes based on current state and time
func (e *EmotionState) Update(deltaTime float64) {
	// Emotions naturally decay toward neutral over time
	decayRate := 0.05 * deltaTime

	// Resilience speeds recovery from negative emotions (1.0x at 0.5)
	negativeDecayRate := decayRate * (0.5 + e.Resilience)

	e.Joy = decay(e.Joy, 0.5, decayRate)
	e.Sadness = decay(e.Sadness, 0.1, negativeDecayRate)
	e.Anger = decay(e.Anger, 0.0, negativeDecayRate)
	e.Fear = decay(e.Fear, 0.1, negativeDecayRate)
	e.Excitement = decay(e.Excitement, 0.3, decayRate)
	e.Contentment = decay(e.Contentment, 0.5, decayRate)
	e.Affection = decay(e.Affection, 0.4, decayRate)
	e.Loneliness = decay(e.Loneliness, 0.2, negativeDecayRate)

	e.Clamp()
	e.updateDominantEmotion()
	e.LastUpdate = time.Now()
}

// UpdateWithSnapshot updates emotions and records a mood snapshot
func (e *EmotionState) UpdateWithSnapshot(deltaTime float64) {
	e.Update(deltaTime)
	e.RecordMoodSnapshot()
}

// ApplyEmotionalStimulus modifies emotions based on external events,
// scaled by volatility (0.5x at volatility 0.0, 1.0x at volatility 1.0)
func (e *EmotionState) ApplyEmotionalStimulus(stimulus EmotionalStimulus) {
	multiplier := 0.5 + 0.5*clamp(e.Volatility, 0.0, 1.0)

	e.Joy += stimulus.JoyDelta * multiplier
	e.Sadness += stimulus.SadnessDelta * multiplier
	e.Anger += stimulus.AngerDelta * multiplier
	e.Fear += stimulus.FearDelta * multiplier
	e.Excitement += stimulus.ExcitementDelta * multiplier
	e.Contentment += stimulus.ContentmentDelta * multiplier
	e.Affection += stimulus.AffectionDelta * multiplier
	e.Loneliness += stimulus.LonelinessDelta * multiplier

	e.Clamp()
	e.updateDominantEmotion()
}

// ApplyEmotionalStimulusRaw applies a stimulus without volatility scaling
func (e *EmotionState) ApplyEmotionalStimulusRaw(stimulus EmotionalStimulus) {
	e.Joy += stimulus.JoyDelta
	e.Sadness += stimulus.SadnessDelta
	e.Anger += stimulus.AngerDelta
	e.Fear += stimulus.FearDelta
	e.Excitement += stimulus.ExcitementDelta
	e.Contentment += stimulus.ContentmentDelta
	e.Affection += stimulus.AffectionDelta
	e.Loneliness += stimulus.LonelinessDelta

	e.Clamp()
	e.updateDominantEmotion()
}

// EmotionalStimulus represents a change in emotional state
type EmotionalStimulus struct {
	JoyDelta         float64
	SadnessDelta     float64
	AngerDelta       float64
	FearDelta        float64
	ExcitementDelta  float64
	ContentmentDelta float64
	AffectionDelta   float64
	LonelinessDelta  float64
	Source           string // What caused this stimulus
}

// GetMoodScore returns an overall mood rating (-1.0 to 1.0)
func (e *EmotionState) GetMoodScore() float64 {
	positive := (e.Joy + e.Excitement + e.Contentment + e.Affection) / 4.0
	negative := (e.Sadness + e.Anger + e.Fear + e.Loneliness) / 4.0
	return positive - negative
}

// GetMoodDescription returns a text description of the current mood
func (e *EmotionState) GetMoodDescription() string {
	score := e.GetMoodScore()

	switch {
	case score > 0.6:
		return "very happy"
	case score > 0.3:
		return "happy"
	case score > 0.0:
		return "content"
	case score > -0.3:
		return "neutral"
	case score > -0.6:
		return "unhappy"
	default:
		return "very distressed"
	}
}

// updateDominantEmotion determines which emotion is strongest
func (e *EmotionState) updateDominantEmotion() {
	emotions := map[string]float64{
		"joyful":    e.Joy,
		"sad":       e.Sadness,
		"angry":     e.Anger,
		"fearful":   e.Fear,
		"excited":   e.Excitement,
		"content":   e.Contentment,
		"affectionate": e.Affection,
		"lonely":    e.Loneliness,
	}

	maxEmotion := "content"
	maxValue := 0.0

	for emotion, value := range emotions {
		if value > maxValue {
			maxValue = value
			maxEmotion = emotion
		}
	}

	e.DominantEmotion = maxEmotion
}

// Clamp ensures all emotions stay within valid range
func (e *EmotionState) Clamp() {
	e.Joy = clamp(e.Joy, 0.0, 1.0)
	e.Sadness = clamp(e.Sadness, 0.0, 1.0)
	e.Anger = clamp(e.Anger, 0.0, 1.0)
	e.Fear = clamp(e.Fear, 0.0, 1.0)
	e.Excitement = clamp(e.Excitement, 0.0, 1.0)
	e.Contentment = clamp(e.Contentment, 0.0, 1.0)
	e.Affection = clamp(e.Affection, 0.0, 1.0)
	e.Loneliness = clamp(e.Loneliness, 0.0, 1.0)
}

// GetBehaviorInfluence returns how emotions affect behavior choices
func (e *EmotionState) GetBehaviorInfluence(behavior types.BehaviorState) float64 {
	switch behavior {
	case types.BehaviorPlaying:
		return (e.Joy + e.Excitement) / 2.0
	case types.BehaviorSleeping:
		return (e.Contentment + (1.0 - e.Excitement)) / 2.0
	case types.BehaviorDistressed:
		return (e.Sadness + e.Fear + e.Anger) / 3.0
	case types.BehaviorHappy:
		return e.Joy
	case types.BehaviorExcited:
		return e.Excitement
	case types.BehaviorSocialInteraction:
		return (e.Affection + (1.0 - e.Loneliness)) / 2.0
	default:
		return 0.5
	}
}

// CreateStimulusFromInteraction creates an emotional response to user interaction
func CreateStimulusFromInteraction(interactionType types.InteractionType, quality float64) EmotionalStimulus {
	stimulus := EmotionalStimulus{
		Source: interactionType.String(),
	}

	switch interactionType {
	case types.InteractionPetting:
		stimulus.JoyDelta = 0.1 * quality
		stimulus.AffectionDelta = 0.15 * quality
		stimulus.ContentmentDelta = 0.1 * quality
		stimulus.LonelinessDelta = -0.1 * quality

	case types.InteractionPlaying:
		stimulus.JoyDelta = 0.15 * quality
		stimulus.ExcitementDelta = 0.2 * quality
		stimulus.LonelinessDelta = -0.1 * quality

	case types.InteractionFeeding:
		stimulus.ContentmentDelta = 0.15 * quality
		stimulus.JoyDelta = 0.1 * quality

	case types.InteractionGrooming:
		stimulus.ContentmentDelta = 0.1 * quality
		stimulus.JoyDelta = 0.05 * quality

	case types.InteractionDiscipline:
		stimulus.FearDelta = 0.1 * quality
		stimulus.SadnessDelta = 0.05 * quality
		stimulus.AngerDelta = 0.05 * quality

	case types.InteractionRewards:
		stimulus.JoyDelta = 0.2 * quality
		stimulus.ExcitementDelta = 0.1 * quality
		stimulus.AffectionDelta = 0.1 * quality

	case types.InteractionMedicalCare:
		stimulus.FearDelta = -0.1 * quality
		stimulus.ContentmentDelta = 0.05 * quality
	}

	return stimulus
}

// RecordMoodSnapshot appends the current mood to the history,
// keeping at most maxMoodHistory entries
func (e *EmotionState) RecordMoodSnapshot() {
	e.MoodHistory = append(e.MoodHistory, MoodSnapshot{
		Timestamp:       time.Now(),
		MoodScore:       e.GetMoodScore(),
		DominantEmotion: e.DominantEmotion,
	})

	if len(e.MoodHistory) > maxMoodHistory {
		e.MoodHistory = e.MoodHistory[len(e.MoodHistory)-maxMoodHistory:]
	}
}

// GetAverageMood returns the average recorded mood score,
// or the current mood score if no history exists
func (e *EmotionState) GetAverageMood() float64 {
	if len(e.MoodHistory) == 0 {
		return e.GetMoodScore()
	}

	total := 0.0
	for _, snapshot := range e.MoodHistory {
		total += snapshot.MoodScore
	}
	return total / float64(len(e.MoodHistory))
}

// GetMoodTrend compares the older and newer halves of the most recent
// snapshots: 1 = improving, -1 = declining, 0 = stable or insufficient data
func (e *EmotionState) GetMoodTrend() int {
	const window = 10
	const minSnapshots = 4
	const threshold = 0.05

	recent := e.MoodHistory
	if len(recent) > window {
		recent = recent[len(recent)-window:]
	}
	if len(recent) < minSnapshots {
		return 0
	}

	half := len(recent) / 2
	older, newer := 0.0, 0.0
	for _, snapshot := range recent[:half] {
		older += snapshot.MoodScore
	}
	for _, snapshot := range recent[half:] {
		newer += snapshot.MoodScore
	}
	diff := newer/float64(len(recent)-half) - older/float64(half)

	switch {
	case diff > threshold:
		return 1
	case diff < -threshold:
		return -1
	default:
		return 0
	}
}

// GetMoodTrendDescription returns a text description of the mood trend
func (e *EmotionState) GetMoodTrendDescription() string {
	switch e.GetMoodTrend() {
	case 1:
		return "improving"
	case -1:
		return "declining"
	default:
		return "stable"
	}
}

// emotionValues returns all primary emotion values
func (e *EmotionState) emotionValues() []float64 {
	return []float64{
		e.Joy, e.Sadness, e.Anger, e.Fear,
		e.Excitement, e.Contentment, e.Affection, e.Loneliness,
	}
}

// GetEmotionalBalance returns how evenly distributed the emotions are
// (1.0 = all equal, lower = dominated by a few emotions)
func (e *EmotionState) GetEmotionalBalance() float64 {
	values := e.emotionValues()

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))

	// Standard deviation of values in [0, 1] is at most 0.5
	return clamp(1.0-2.0*math.Sqrt(variance), 0.0, 1.0)
}

// IsEmotionallyStable reports whether negative emotions are under control
func (e *EmotionState) IsEmotionallyStable() bool {
	negative := (e.Sadness + e.Anger + e.Fear + e.Loneliness) / 4.0
	return negative < 0.4 && e.GetMoodScore() > -0.2
}

// GetEmotionalNeeds returns the emotional needs implied by the current state
func (e *EmotionState) GetEmotionalNeeds() []string {
	needs := make([]string, 0)

	if e.Loneliness > 0.6 {
		needs = append(needs, "social_interaction")
	}
	if e.Joy < 0.3 {
		needs = append(needs, "happiness")
	}
	if e.Affection < 0.3 {
		needs = append(needs, "affection")
	}
	if e.Fear > 0.6 || e.Sadness > 0.6 {
		needs = append(needs, "comfort")
	}
	if e.Anger > 0.6 {
		needs = append(needs, "calming")
	}
	if e.Excitement < 0.2 {
		needs = append(needs, "stimulation")
	}

	return needs
}

// BlendEmotions moves each emotion toward another state's value by factor (0.0 to 1.0)
func (e *EmotionState) BlendEmotions(other *EmotionState, factor float64) {
	factor = clamp(factor, 0.0, 1.0)

	e.Joy += (other.Joy - e.Joy) * factor
	e.Sadness += (other.Sadness - e.Sadness) * factor
	e.Anger += (other.Anger - e.Anger) * factor
	e.Fear += (other.Fear - e.Fear) * factor
	e.Excitement += (other.Excitement - e.Excitement) * factor
	e.Contentment += (other.Contentment - e.Contentment) * factor
	e.Affection += (other.Affection - e.Affection) * factor
	e.Loneliness += (other.Loneliness - e.Loneliness) * factor

	e.Clamp()
	e.updateDominantEmotion()
}

// Copy returns a deep copy of the emotion state
func (e *EmotionState) Copy() *EmotionState {
	c := *e
	c.MoodHistory = make([]MoodSnapshot, len(e.MoodHistory))
	copy(c.MoodHistory, e.MoodHistory)
	return &c
}

// SetResilience sets resilience, clamped to 0.0 - 1.0
func (e *EmotionState) SetResilience(value float64) {
	e.Resilience = clamp(value, 0.0, 1.0)
}

// SetVolatility sets volatility, clamped to 0.0 - 1.0
func (e *EmotionState) SetVolatility(value float64) {
	e.Volatility = clamp(value, 0.0, 1.0)
}

// GetEmotionIntensity returns the strength of the strongest emotion
func (e *EmotionState) GetEmotionIntensity() float64 {
	intensity := 0.0
	for _, v := range e.emotionValues() {
		if v > intensity {
			intensity = v
		}
	}
	return intensity
}

// Helper function for decay toward target value
func decay(current, target, rate float64) float64 {
	if current > target {
		return current - rate
	} else if current < target {
		return current + rate
	}
	return current
}
