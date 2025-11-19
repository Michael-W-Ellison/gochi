package ai

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewEmotionState(t *testing.T) {
	es := NewEmotionState()

	if es == nil {
		t.Fatal("NewEmotionState returned nil")
	}

	if es.Joy != 0.6 {
		t.Errorf("Expected Joy 0.6, got %f", es.Joy)
	}

	if es.Sadness != 0.1 {
		t.Errorf("Expected Sadness 0.1, got %f", es.Sadness)
	}

	if es.DominantEmotion != "content" {
		t.Errorf("Expected DominantEmotion 'content', got %s", es.DominantEmotion)
	}

	if es.LastUpdate.IsZero() {
		t.Error("LastUpdate should not be zero")
	}
}

func TestEmotionStateUpdate(t *testing.T) {
	es := NewEmotionState()

	// Set some extreme values
	es.Joy = 1.0
	es.Anger = 0.8

	initialJoy := es.Joy
	initialAnger := es.Anger

	// Update with delta time
	es.Update(1.0)

	// Joy should decay toward 0.5
	if es.Joy >= initialJoy {
		t.Error("Joy should decay toward neutral")
	}

	// Anger should decay toward 0.0
	if es.Anger >= initialAnger {
		t.Error("Anger should decay toward zero")
	}

	// Values should be clamped
	if es.Joy < 0 || es.Joy > 1 {
		t.Errorf("Joy should be clamped between 0 and 1, got %f", es.Joy)
	}
}

func TestApplyEmotionalStimulus(t *testing.T) {
	es := NewEmotionState()

	initialJoy := es.Joy
	initialSadness := es.Sadness

	stimulus := EmotionalStimulus{
		JoyDelta:     0.2,
		SadnessDelta: -0.1,
		AngerDelta:   0.1,
	}

	es.ApplyEmotionalStimulus(stimulus)

	if es.Joy != initialJoy+0.2 {
		t.Errorf("Expected Joy %f, got %f", initialJoy+0.2, es.Joy)
	}

	if es.Sadness != initialSadness-0.1 {
		t.Errorf("Expected Sadness %f, got %f", initialSadness-0.1, es.Sadness)
	}
}

func TestClamp(t *testing.T) {
	es := NewEmotionState()

	// Set values out of bounds
	es.Joy = 1.5
	es.Sadness = -0.5
	es.Anger = 2.0
	es.Fear = -1.0

	es.Clamp()

	if es.Joy != 1.0 {
		t.Errorf("Joy should be clamped to 1.0, got %f", es.Joy)
	}

	if es.Sadness != 0.0 {
		t.Errorf("Sadness should be clamped to 0.0, got %f", es.Sadness)
	}

	if es.Anger != 1.0 {
		t.Errorf("Anger should be clamped to 1.0, got %f", es.Anger)
	}

	if es.Fear != 0.0 {
		t.Errorf("Fear should be clamped to 0.0, got %f", es.Fear)
	}
}

func TestGetMoodScore(t *testing.T) {
	es := NewEmotionState()

	// High positive emotions
	es.Joy = 0.9
	es.Contentment = 0.8
	es.Affection = 0.9
	es.Sadness = 0.1
	es.Anger = 0.0
	es.Fear = 0.1

	score := es.GetMoodScore()

	if score <= 0 {
		t.Errorf("Expected positive mood score, got %f", score)
	}

	// High negative emotions
	es.Joy = 0.1
	es.Sadness = 0.9
	es.Anger = 0.8
	es.Fear = 0.7

	score = es.GetMoodScore()

	if score >= 0 {
		t.Errorf("Expected negative mood score, got %f", score)
	}
}

func TestGetMoodDescription(t *testing.T) {
	es := NewEmotionState()

	// Set high positive emotions
	es.Joy = 0.9
	es.Contentment = 0.8
	es.Affection = 0.8
	es.Sadness = 0.1
	es.Anger = 0.0
	es.Fear = 0.1
	desc := es.GetMoodDescription()
	if desc != "very happy" && desc != "happy" {
		t.Errorf("Expected 'very happy' or 'happy', got %s", desc)
	}

	// Set moderate positive emotions
	es.Joy = 0.5
	es.Contentment = 0.5
	es.Sadness = 0.2
	desc = es.GetMoodDescription()
	if desc != "happy" && desc != "content" {
		t.Errorf("Expected 'happy' or 'content', got %s", desc)
	}

	// Set high negative emotions
	es.Joy = 0.1
	es.Contentment = 0.1
	es.Sadness = 0.9
	es.Anger = 0.8
	es.Fear = 0.7
	desc = es.GetMoodDescription()
	if desc != "very distressed" && desc != "unhappy" {
		t.Errorf("Expected 'very distressed' or 'unhappy', got %s", desc)
	}
}

func TestGetBehaviorInfluence(t *testing.T) {
	es := NewEmotionState()

	// Test different behaviors
	behaviors := []types.BehaviorState{
		types.BehaviorPlaying,
		types.BehaviorSleeping,
		types.BehaviorHappy,
		types.BehaviorExcited,
	}

	for _, behavior := range behaviors {
		influence := es.GetBehaviorInfluence(behavior)

		if influence < 0 || influence > 1 {
			t.Errorf("Behavior influence should be between 0 and 1, got %f for %v", influence, behavior)
		}
	}
}

func TestCreateStimulusFromInteraction(t *testing.T) {
	tests := []struct {
		interactionType types.InteractionType
		checkPositive   bool
	}{
		{types.InteractionFeeding, true},
		{types.InteractionPlaying, true},
		{types.InteractionPetting, true},
		{types.InteractionTraining, false},
		{types.InteractionGrooming, true},
	}

	for _, test := range tests {
		stimulus := CreateStimulusFromInteraction(test.interactionType, 1.0)

		if test.checkPositive {
			// Positive interactions should increase joy
			if stimulus.JoyDelta <= 0 {
				t.Errorf("Expected positive JoyDelta for %v, got %f", test.interactionType, stimulus.JoyDelta)
			}
		}
	}
}

func TestUpdateDominantEmotion(t *testing.T) {
	es := NewEmotionState()

	// Set joy as highest
	es.Joy = 0.9
	es.Sadness = 0.1
	es.Anger = 0.1
	es.Fear = 0.1
	es.Excitement = 0.2
	es.Contentment = 0.3
	es.Affection = 0.3
	es.Loneliness = 0.1

	es.updateDominantEmotion()

	if es.DominantEmotion != "joyful" {
		t.Errorf("Expected 'joyful', got %s", es.DominantEmotion)
	}

	// Set sadness as highest
	es.Joy = 0.1
	es.Sadness = 0.9

	es.updateDominantEmotion()

	if es.DominantEmotion != "sad" {
		t.Errorf("Expected 'sad', got %s", es.DominantEmotion)
	}
}

func TestEmotionDecay(t *testing.T) {
	es := NewEmotionState()

	es.Joy = 1.0
	es.Anger = 0.8
	es.LastUpdate = time.Now().Add(-1 * time.Hour)

	// Multiple updates should cause decay
	for i := 0; i < 10; i++ {
		es.Update(1.0)
	}

	// After many updates, emotions should be closer to neutral
	if es.Joy > 0.9 {
		t.Error("Joy should have decayed from 1.0")
	}

	if es.Anger > 0.5 {
		t.Error("Anger should have decayed significantly")
	}
}

func TestEmotionalStimulus(t *testing.T) {
	es := NewEmotionState()

	// Apply multiple stimuli
	stimuli := []EmotionalStimulus{
		{JoyDelta: 0.1, SadnessDelta: -0.1},
		{ExcitementDelta: 0.2, FearDelta: -0.05},
		{AffectionDelta: 0.15, LonelinessDelta: -0.2},
	}

	for _, stimulus := range stimuli {
		es.ApplyEmotionalStimulus(stimulus)
	}

	// All values should still be clamped
	if es.Joy < 0 || es.Joy > 1 {
		t.Errorf("Joy out of bounds: %f", es.Joy)
	}
	if es.Excitement < 0 || es.Excitement > 1 {
		t.Errorf("Excitement out of bounds: %f", es.Excitement)
	}
}
