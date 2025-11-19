package ai

import (
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

	// Emotion history for tracking mood patterns
	LastUpdate time.Time
}

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
		LastUpdate:      time.Now(),
	}
}

// Update processes emotional changes based on current state and time
func (e *EmotionState) Update(deltaTime float64) {
	// Emotions naturally decay toward neutral over time
	decayRate := 0.05 * deltaTime

	e.Joy = decay(e.Joy, 0.5, decayRate)
	e.Sadness = decay(e.Sadness, 0.1, decayRate)
	e.Anger = decay(e.Anger, 0.0, decayRate)
	e.Fear = decay(e.Fear, 0.1, decayRate)
	e.Excitement = decay(e.Excitement, 0.3, decayRate)
	e.Contentment = decay(e.Contentment, 0.5, decayRate)
	e.Affection = decay(e.Affection, 0.4, decayRate)
	e.Loneliness = decay(e.Loneliness, 0.2, decayRate)

	e.Clamp()
	e.updateDominantEmotion()
	e.LastUpdate = time.Now()
}

// ApplyEmotionalStimulus modifies emotions based on external events
func (e *EmotionState) ApplyEmotionalStimulus(stimulus EmotionalStimulus) {
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

// Helper function for decay toward target value
func decay(current, target, rate float64) float64 {
	if current > target {
		return current - rate
	} else if current < target {
		return current + rate
	}
	return current
}
