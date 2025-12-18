package ai

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// =============== EmotionState Tests ===============

// TestNewEmotionState tests emotion state creation
func TestNewEmotionState(t *testing.T) {
	e := NewEmotionState()

	if e == nil {
		t.Fatal("NewEmotionState returned nil")
	}

	// Check default values
	if e.Joy != 0.6 {
		t.Errorf("Expected Joy 0.6, got %f", e.Joy)
	}
	if e.Sadness != 0.1 {
		t.Errorf("Expected Sadness 0.1, got %f", e.Sadness)
	}
	if e.Anger != 0.1 {
		t.Errorf("Expected Anger 0.1, got %f", e.Anger)
	}
	if e.Fear != 0.1 {
		t.Errorf("Expected Fear 0.1, got %f", e.Fear)
	}
	if e.Excitement != 0.4 {
		t.Errorf("Expected Excitement 0.4, got %f", e.Excitement)
	}
	if e.Contentment != 0.5 {
		t.Errorf("Expected Contentment 0.5, got %f", e.Contentment)
	}
	if e.Affection != 0.5 {
		t.Errorf("Expected Affection 0.5, got %f", e.Affection)
	}
	if e.Loneliness != 0.2 {
		t.Errorf("Expected Loneliness 0.2, got %f", e.Loneliness)
	}

	// Check new fields
	if e.Resilience != 0.5 {
		t.Errorf("Expected Resilience 0.5, got %f", e.Resilience)
	}
	if e.Volatility != 0.5 {
		t.Errorf("Expected Volatility 0.5, got %f", e.Volatility)
	}
	if e.MoodHistory == nil {
		t.Error("MoodHistory should not be nil")
	}
}

// TestNewEmotionStateWithPersonality tests personality-influenced creation
func TestNewEmotionStateWithPersonality(t *testing.T) {
	traits := NewBalancedTraits()
	e := NewEmotionStateWithPersonality(traits)

	if e == nil {
		t.Fatal("NewEmotionStateWithPersonality returned nil")
	}

	// With balanced traits (Neuroticism=0.3), resilience should be high
	if e.Resilience < 0.5 {
		t.Errorf("Expected higher resilience with low neuroticism, got %f", e.Resilience)
	}
}

// TestNewEmotionStateWithHighNeuroticism tests high neuroticism effects
func TestNewEmotionStateWithHighNeuroticism(t *testing.T) {
	traits := NewBalancedTraits()
	traits.Neuroticism = 0.9 // High neuroticism

	e := NewEmotionStateWithPersonality(traits)

	// High neuroticism should mean low resilience
	if e.Resilience > 0.3 {
		t.Errorf("Expected low resilience with high neuroticism, got %f", e.Resilience)
	}

	// And high volatility
	if e.Volatility < 0.6 {
		t.Errorf("Expected high volatility with high neuroticism, got %f", e.Volatility)
	}
}

// TestUpdate tests emotional decay over time
func TestUpdate(t *testing.T) {
	e := NewEmotionState()

	// Set emotions to extreme values
	e.Joy = 1.0
	e.Sadness = 0.8
	e.Anger = 0.9

	initialJoy := e.Joy
	initialSadness := e.Sadness
	initialAnger := e.Anger

	e.Update(1.0)

	// Joy should decay toward 0.5
	if e.Joy >= initialJoy {
		t.Error("Joy should decay toward target")
	}

	// Sadness should decay toward 0.1
	if e.Sadness >= initialSadness {
		t.Error("Sadness should decay toward target")
	}

	// Anger should decay toward 0.0
	if e.Anger >= initialAnger {
		t.Error("Anger should decay toward target")
	}
}

// TestUpdateResilienceAffectsDecay tests resilience impact on decay
func TestUpdateResilienceAffectsDecay(t *testing.T) {
	// High resilience
	e1 := NewEmotionState()
	e1.Resilience = 1.0
	e1.Sadness = 0.8

	// Low resilience
	e2 := NewEmotionState()
	e2.Resilience = 0.0
	e2.Sadness = 0.8

	e1.Update(1.0)
	e2.Update(1.0)

	// High resilience should recover from sadness faster
	if e1.Sadness >= e2.Sadness {
		t.Error("High resilience should recover from negative emotions faster")
	}
}

// TestUpdateWithSnapshot tests snapshot recording
func TestUpdateWithSnapshot(t *testing.T) {
	e := NewEmotionState()

	if len(e.MoodHistory) != 0 {
		t.Error("Should start with empty history")
	}

	e.UpdateWithSnapshot(1.0)

	if len(e.MoodHistory) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(e.MoodHistory))
	}
}

// TestApplyEmotionalStimulus tests stimulus application
func TestApplyEmotionalStimulus(t *testing.T) {
	e := NewEmotionState()
	e.Volatility = 1.0 // Max volatility for predictable results

	stimulus := EmotionalStimulus{
		JoyDelta:    0.2,
		SadnessDelta: -0.1,
	}

	initialJoy := e.Joy
	e.ApplyEmotionalStimulus(stimulus)

	// With max volatility (multiplier = 1.0), full delta should apply
	if e.Joy <= initialJoy {
		t.Error("Joy should increase from positive stimulus")
	}
}

// TestApplyEmotionalStimulusVolatility tests volatility effect
func TestApplyEmotionalStimulusVolatility(t *testing.T) {
	// High volatility
	e1 := NewEmotionState()
	e1.Volatility = 1.0
	e1.Joy = 0.5

	// Low volatility
	e2 := NewEmotionState()
	e2.Volatility = 0.0
	e2.Joy = 0.5

	stimulus := EmotionalStimulus{JoyDelta: 0.2}

	e1.ApplyEmotionalStimulus(stimulus)
	e2.ApplyEmotionalStimulus(stimulus)

	// High volatility should have stronger reaction
	if e1.Joy <= e2.Joy {
		t.Error("High volatility should produce stronger emotional response")
	}
}

// TestApplyEmotionalStimulusRaw tests raw stimulus application
func TestApplyEmotionalStimulusRaw(t *testing.T) {
	e := NewEmotionState()
	e.Volatility = 0.0 // Low volatility
	e.Joy = 0.5

	stimulus := EmotionalStimulus{JoyDelta: 0.2}

	e.ApplyEmotionalStimulusRaw(stimulus)

	// Raw should apply full delta regardless of volatility
	expected := 0.7
	if e.Joy != expected {
		t.Errorf("Expected Joy %f, got %f", expected, e.Joy)
	}
}

// TestGetMoodScore tests mood score calculation
func TestGetMoodScore(t *testing.T) {
	e := NewEmotionState()

	// With default values, should be positive
	score := e.GetMoodScore()
	if score <= 0 {
		t.Errorf("Default mood should be positive, got %f", score)
	}

	// Set to very negative state
	e.Joy = 0.0
	e.Excitement = 0.0
	e.Contentment = 0.0
	e.Affection = 0.0
	e.Sadness = 1.0
	e.Anger = 1.0
	e.Fear = 1.0
	e.Loneliness = 1.0

	score = e.GetMoodScore()
	if score >= 0 {
		t.Errorf("Negative mood should be negative score, got %f", score)
	}

	// Score should be between -1 and 1
	if score < -1 || score > 1 {
		t.Errorf("Mood score should be between -1 and 1, got %f", score)
	}
}

// TestGetMoodDescription tests mood descriptions
func TestGetMoodDescription(t *testing.T) {
	e := NewEmotionState()

	tests := []struct {
		joy        float64
		sadness    float64
		expected   string
	}{
		{1.0, 0.0, "very happy"},
		{0.7, 0.1, "happy"},
		{0.5, 0.2, "content"},
		{0.3, 0.4, "neutral"},
		{0.1, 0.6, "unhappy"},
		{0.0, 1.0, "very distressed"},
	}

	for _, tt := range tests {
		e.Joy = tt.joy
		e.Excitement = tt.joy
		e.Contentment = tt.joy
		e.Affection = tt.joy
		e.Sadness = tt.sadness
		e.Anger = tt.sadness
		e.Fear = tt.sadness
		e.Loneliness = tt.sadness

		desc := e.GetMoodDescription()
		if desc != tt.expected {
			t.Errorf("With joy=%f sadness=%f, expected '%s', got '%s' (score=%f)",
				tt.joy, tt.sadness, tt.expected, desc, e.GetMoodScore())
		}
	}
}

// TestUpdateDominantEmotion tests dominant emotion detection
func TestUpdateDominantEmotion(t *testing.T) {
	e := NewEmotionState()

	// Set joy as highest
	e.Joy = 1.0
	e.Sadness = 0.1
	e.Anger = 0.1
	e.Fear = 0.1
	e.Excitement = 0.1
	e.Contentment = 0.1
	e.Affection = 0.1
	e.Loneliness = 0.1
	e.updateDominantEmotion()

	if e.DominantEmotion != "joyful" {
		t.Errorf("Expected dominant emotion 'joyful', got '%s'", e.DominantEmotion)
	}

	// Set sadness as highest
	e.Joy = 0.1
	e.Sadness = 1.0
	e.updateDominantEmotion()

	if e.DominantEmotion != "sad" {
		t.Errorf("Expected dominant emotion 'sad', got '%s'", e.DominantEmotion)
	}
}

// TestClamp tests value clamping
func TestClamp(t *testing.T) {
	e := NewEmotionState()

	e.Joy = 1.5
	e.Sadness = -0.5
	e.Anger = 2.0
	e.Fear = -1.0

	e.Clamp()

	if e.Joy != 1.0 {
		t.Errorf("Joy should be clamped to 1.0, got %f", e.Joy)
	}
	if e.Sadness != 0.0 {
		t.Errorf("Sadness should be clamped to 0.0, got %f", e.Sadness)
	}
	if e.Anger != 1.0 {
		t.Errorf("Anger should be clamped to 1.0, got %f", e.Anger)
	}
	if e.Fear != 0.0 {
		t.Errorf("Fear should be clamped to 0.0, got %f", e.Fear)
	}
}

// TestGetBehaviorInfluence tests behavior influence calculation
func TestGetBehaviorInfluence(t *testing.T) {
	e := NewEmotionState()
	e.Joy = 1.0
	e.Excitement = 1.0

	playInfluence := e.GetBehaviorInfluence(types.BehaviorPlaying)
	if playInfluence != 1.0 {
		t.Errorf("Expected play influence 1.0 with max joy/excitement, got %f", playInfluence)
	}

	e.Sadness = 1.0
	e.Fear = 1.0
	e.Anger = 1.0
	distressInfluence := e.GetBehaviorInfluence(types.BehaviorDistressed)
	if distressInfluence != 1.0 {
		t.Errorf("Expected distress influence 1.0 with max negative emotions, got %f", distressInfluence)
	}
}

// TestCreateStimulusFromInteraction tests interaction stimuli
func TestCreateStimulusFromInteraction(t *testing.T) {
	// Test petting
	stimulus := CreateStimulusFromInteraction(types.InteractionPetting, 1.0)
	if stimulus.JoyDelta <= 0 {
		t.Error("Petting should increase joy")
	}
	if stimulus.AffectionDelta <= 0 {
		t.Error("Petting should increase affection")
	}
	if stimulus.LonelinessDelta >= 0 {
		t.Error("Petting should decrease loneliness")
	}

	// Test discipline
	stimulus = CreateStimulusFromInteraction(types.InteractionDiscipline, 1.0)
	if stimulus.FearDelta <= 0 {
		t.Error("Discipline should increase fear")
	}
	if stimulus.SadnessDelta <= 0 {
		t.Error("Discipline should increase sadness")
	}

	// Test rewards
	stimulus = CreateStimulusFromInteraction(types.InteractionRewards, 1.0)
	if stimulus.JoyDelta <= 0 {
		t.Error("Rewards should increase joy")
	}
	if stimulus.ExcitementDelta <= 0 {
		t.Error("Rewards should increase excitement")
	}
}

// =============== Mood History Tests ===============

// TestRecordMoodSnapshot tests snapshot recording
func TestRecordMoodSnapshot(t *testing.T) {
	e := NewEmotionState()

	e.RecordMoodSnapshot()

	if len(e.MoodHistory) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(e.MoodHistory))
	}

	snapshot := e.MoodHistory[0]
	if snapshot.MoodScore != e.GetMoodScore() {
		t.Error("Snapshot mood score should match current mood")
	}
	if snapshot.DominantEmotion != e.DominantEmotion {
		t.Error("Snapshot dominant emotion should match current")
	}
}

// TestRecordMoodSnapshotLimit tests snapshot limit
func TestRecordMoodSnapshotLimit(t *testing.T) {
	e := NewEmotionState()

	for i := 0; i < 150; i++ {
		e.RecordMoodSnapshot()
	}

	if len(e.MoodHistory) != 100 {
		t.Errorf("Mood history should be capped at 100, got %d", len(e.MoodHistory))
	}
}

// TestGetAverageMood tests average mood calculation
func TestGetAverageMood(t *testing.T) {
	e := NewEmotionState()

	// No history - should return current mood
	avg := e.GetAverageMood()
	if avg != e.GetMoodScore() {
		t.Error("With no history, average should equal current mood")
	}

	// Add some history
	e.RecordMoodSnapshot()
	e.Joy = 1.0
	e.Sadness = 0.0
	e.RecordMoodSnapshot()

	avg = e.GetAverageMood()
	// Average should be between the two recorded moods
	if avg <= 0 || avg >= 1 {
		t.Errorf("Average mood seems incorrect: %f", avg)
	}
}

// TestGetMoodTrend tests mood trend detection
func TestGetMoodTrend(t *testing.T) {
	e := NewEmotionState()

	// Not enough history
	if e.GetMoodTrend() != 0 {
		t.Error("Should return 0 with insufficient history")
	}

	// Record improving mood
	e.Joy = 0.3
	e.Sadness = 0.5
	for i := 0; i < 5; i++ {
		e.RecordMoodSnapshot()
	}

	e.Joy = 0.9
	e.Sadness = 0.1
	for i := 0; i < 5; i++ {
		e.RecordMoodSnapshot()
	}

	trend := e.GetMoodTrend()
	if trend != 1 {
		t.Errorf("Expected improving trend (1), got %d", trend)
	}
}

// TestGetMoodTrendDescription tests trend descriptions
func TestGetMoodTrendDescription(t *testing.T) {
	e := NewEmotionState()

	desc := e.GetMoodTrendDescription()
	if desc != "stable" {
		t.Errorf("Expected 'stable' with no history, got '%s'", desc)
	}
}

// =============== Emotional Analysis Tests ===============

// TestGetEmotionalBalance tests balance calculation
func TestGetEmotionalBalance(t *testing.T) {
	e := NewEmotionState()

	// Set all emotions to same value - should be very balanced
	e.Joy = 0.5
	e.Sadness = 0.5
	e.Anger = 0.5
	e.Fear = 0.5
	e.Excitement = 0.5
	e.Contentment = 0.5
	e.Affection = 0.5
	e.Loneliness = 0.5

	balance := e.GetEmotionalBalance()
	if balance != 1.0 {
		t.Errorf("Expected balance 1.0 with uniform emotions, got %f", balance)
	}

	// Set one emotion extremely high - should be unbalanced
	e.Joy = 1.0
	e.Sadness = 0.0
	e.Anger = 0.0
	e.Fear = 0.0
	e.Excitement = 0.0
	e.Contentment = 0.0
	e.Affection = 0.0
	e.Loneliness = 0.0

	balance = e.GetEmotionalBalance()
	if balance >= 1.0 {
		t.Errorf("Expected lower balance with extreme emotions, got %f", balance)
	}
}

// TestIsEmotionallyStable tests stability check
func TestIsEmotionallyStable(t *testing.T) {
	e := NewEmotionState()

	// Default should be stable
	if !e.IsEmotionallyStable() {
		t.Error("Default state should be stable")
	}

	// High negative emotions should be unstable
	e.Sadness = 0.5
	e.Anger = 0.5
	e.Fear = 0.5
	e.Loneliness = 0.5

	if e.IsEmotionallyStable() {
		t.Error("High negative emotions should be unstable")
	}
}

// TestGetEmotionalNeeds tests needs detection
func TestGetEmotionalNeeds(t *testing.T) {
	e := NewEmotionState()

	// Default state should have minimal needs
	needs := e.GetEmotionalNeeds()
	initialNeedsCount := len(needs)

	// Set high loneliness
	e.Loneliness = 0.8
	needs = e.GetEmotionalNeeds()
	found := false
	for _, n := range needs {
		if n == "social_interaction" {
			found = true
			break
		}
	}
	if !found {
		t.Error("High loneliness should indicate social_interaction need")
	}

	// Set low joy
	e.Joy = 0.1
	needs = e.GetEmotionalNeeds()
	found = false
	for _, n := range needs {
		if n == "happiness" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Low joy should indicate happiness need")
	}

	// More needs should exist now
	if len(needs) <= initialNeedsCount {
		t.Error("Should have more needs with negative state")
	}
}

// TestBlendEmotions tests emotion blending
func TestBlendEmotions(t *testing.T) {
	e1 := NewEmotionState()
	e1.Joy = 0.0
	e1.Sadness = 1.0

	e2 := NewEmotionState()
	e2.Joy = 1.0
	e2.Sadness = 0.0

	// Blend 50%
	e1.BlendEmotions(e2, 0.5)

	if e1.Joy != 0.5 {
		t.Errorf("Expected Joy 0.5 after 50%% blend, got %f", e1.Joy)
	}
	if e1.Sadness != 0.5 {
		t.Errorf("Expected Sadness 0.5 after 50%% blend, got %f", e1.Sadness)
	}

	// Blend 100% - should become other
	e1.Joy = 0.0
	e1.BlendEmotions(e2, 1.0)

	if e1.Joy != 1.0 {
		t.Errorf("Expected Joy 1.0 after 100%% blend, got %f", e1.Joy)
	}
}

// TestBlendEmotionsClamping tests blend factor clamping
func TestBlendEmotionsClamping(t *testing.T) {
	e1 := NewEmotionState()
	e1.Joy = 0.0

	e2 := NewEmotionState()
	e2.Joy = 1.0

	// Blend factor > 1 should be clamped
	e1.BlendEmotions(e2, 2.0)

	if e1.Joy != 1.0 {
		t.Errorf("Blend factor should be clamped, expected Joy 1.0, got %f", e1.Joy)
	}
}

// TestCopy tests deep copying
func TestCopy(t *testing.T) {
	e := NewEmotionState()
	e.Joy = 0.8
	e.Resilience = 0.9
	e.RecordMoodSnapshot()

	copy := e.Copy()

	// Verify values are copied
	if copy.Joy != e.Joy {
		t.Error("Joy not copied correctly")
	}
	if copy.Resilience != e.Resilience {
		t.Error("Resilience not copied correctly")
	}
	if len(copy.MoodHistory) != len(e.MoodHistory) {
		t.Error("MoodHistory not copied correctly")
	}

	// Verify it's a deep copy - modifying copy shouldn't affect original
	copy.Joy = 0.1
	if e.Joy == copy.Joy {
		t.Error("Copy is not independent - Joy was modified")
	}
}

// TestSetResilience tests resilience setting
func TestSetResilience(t *testing.T) {
	e := NewEmotionState()

	e.SetResilience(0.8)
	if e.Resilience != 0.8 {
		t.Errorf("Expected resilience 0.8, got %f", e.Resilience)
	}

	// Test clamping
	e.SetResilience(1.5)
	if e.Resilience != 1.0 {
		t.Errorf("Resilience should be clamped to 1.0, got %f", e.Resilience)
	}

	e.SetResilience(-0.5)
	if e.Resilience != 0.0 {
		t.Errorf("Resilience should be clamped to 0.0, got %f", e.Resilience)
	}
}

// TestSetVolatility tests volatility setting
func TestSetVolatility(t *testing.T) {
	e := NewEmotionState()

	e.SetVolatility(0.8)
	if e.Volatility != 0.8 {
		t.Errorf("Expected volatility 0.8, got %f", e.Volatility)
	}

	// Test clamping
	e.SetVolatility(1.5)
	if e.Volatility != 1.0 {
		t.Errorf("Volatility should be clamped to 1.0, got %f", e.Volatility)
	}
}

// TestGetEmotionIntensity tests intensity calculation
func TestGetEmotionIntensity(t *testing.T) {
	e := NewEmotionState()

	// Set all to low
	e.Joy = 0.2
	e.Sadness = 0.2
	e.Anger = 0.2
	e.Fear = 0.2
	e.Excitement = 0.2
	e.Contentment = 0.2
	e.Affection = 0.2
	e.Loneliness = 0.2

	intensity := e.GetEmotionIntensity()
	if intensity != 0.2 {
		t.Errorf("Expected intensity 0.2, got %f", intensity)
	}

	// Set one high
	e.Joy = 0.9
	intensity = e.GetEmotionIntensity()
	if intensity != 0.9 {
		t.Errorf("Expected intensity 0.9, got %f", intensity)
	}
}

// TestDecay tests decay helper function
func TestDecay(t *testing.T) {
	tolerance := 0.0001

	// Decay toward lower target
	result := decay(0.8, 0.5, 0.1)
	expected := 0.7
	if result < expected-tolerance || result > expected+tolerance {
		t.Errorf("Expected %f, got %f", expected, result)
	}

	// Decay toward higher target
	result = decay(0.2, 0.5, 0.1)
	expected = 0.3
	if result < expected-tolerance || result > expected+tolerance {
		t.Errorf("Expected %f, got %f", expected, result)
	}

	// Already at target
	result = decay(0.5, 0.5, 0.1)
	expected = 0.5
	if result < expected-tolerance || result > expected+tolerance {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

// =============== Integration Tests ===============

// TestEmotionalLifecycle tests a complete emotional lifecycle
func TestEmotionalLifecycle(t *testing.T) {
	e := NewEmotionState()

	// Initial state should be positive
	initialMood := e.GetMoodScore()
	if initialMood < 0 {
		t.Error("Initial mood should be positive")
	}

	// Apply negative stimulus
	negativeStimulus := EmotionalStimulus{
		SadnessDelta: 0.3,
		AngerDelta:   0.2,
		FearDelta:    0.2,
	}
	e.ApplyEmotionalStimulus(negativeStimulus)
	e.RecordMoodSnapshot()

	afterNegative := e.GetMoodScore()
	if afterNegative >= initialMood {
		t.Error("Mood should decrease after negative stimulus")
	}

	// Update over time - should recover
	for i := 0; i < 10; i++ {
		e.Update(1.0)
		e.RecordMoodSnapshot()
	}

	afterRecovery := e.GetMoodScore()
	if afterRecovery <= afterNegative {
		t.Error("Mood should improve after decay toward baseline")
	}

	// Verify history was recorded
	if len(e.MoodHistory) < 10 {
		t.Error("Mood history should have been recorded")
	}
}

// TestEmotionalInteractionCycle tests interaction effects
func TestEmotionalInteractionCycle(t *testing.T) {
	e := NewEmotionState()

	// Apply multiple positive interactions
	for i := 0; i < 5; i++ {
		stimulus := CreateStimulusFromInteraction(types.InteractionPetting, 1.0)
		e.ApplyEmotionalStimulus(stimulus)
	}

	// Joy and affection should be high
	if e.Joy < 0.7 {
		t.Errorf("Joy should be high after positive interactions, got %f", e.Joy)
	}
	if e.Affection < 0.7 {
		t.Errorf("Affection should be high after petting, got %f", e.Affection)
	}

	// Loneliness should be low
	if e.Loneliness > 0.2 {
		t.Errorf("Loneliness should be low after interactions, got %f", e.Loneliness)
	}
}

// TestPersonalityEmotionIntegration tests personality affecting emotions
func TestPersonalityEmotionIntegration(t *testing.T) {
	// Create two different personality types
	extrovertTraits := NewBalancedTraits()
	extrovertTraits.Extraversion = 0.9
	extrovertTraits.Neuroticism = 0.2

	introvertTraits := NewBalancedTraits()
	introvertTraits.Extraversion = 0.1
	introvertTraits.Neuroticism = 0.8

	extrovertEmotions := NewEmotionStateWithPersonality(extrovertTraits)
	introvertEmotions := NewEmotionStateWithPersonality(introvertTraits)

	// Extrovert should have lower loneliness baseline
	if extrovertEmotions.Loneliness >= introvertEmotions.Loneliness {
		t.Error("Extrovert should have lower loneliness baseline")
	}

	// Extrovert should have higher excitement baseline
	if extrovertEmotions.Excitement <= introvertEmotions.Excitement {
		t.Error("Extrovert should have higher excitement baseline")
	}

	// Low neuroticism should mean higher resilience
	if extrovertEmotions.Resilience <= introvertEmotions.Resilience {
		t.Error("Low neuroticism should mean higher resilience")
	}
}

// TestMoodSnapshotTimestamp tests timestamp recording
func TestMoodSnapshotTimestamp(t *testing.T) {
	e := NewEmotionState()

	before := time.Now()
	e.RecordMoodSnapshot()
	after := time.Now()

	if len(e.MoodHistory) != 1 {
		t.Fatal("Snapshot not recorded")
	}

	timestamp := e.MoodHistory[0].Timestamp
	if timestamp.Before(before) || timestamp.After(after) {
		t.Error("Snapshot timestamp is incorrect")
	}
}
