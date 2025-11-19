package ai

import (
	"math"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewLearningSystem(t *testing.T) {
	ls := NewLearningSystem()

	if ls == nil {
		t.Fatal("NewLearningSystem returned nil")
	}

	if ls.LearningRate != 0.1 {
		t.Errorf("Expected learning rate 0.1, got %.2f", ls.LearningRate)
	}

	if ls.DiscountFactor != 0.9 {
		t.Errorf("Expected discount factor 0.9, got %.2f", ls.DiscountFactor)
	}

	if ls.ExplorationRate != 1.0 {
		t.Errorf("Expected initial exploration rate 1.0, got %.2f", ls.ExplorationRate)
	}

	if ls.QTable == nil {
		t.Error("QTable should be initialized")
	}

	if ls.ExperienceBuffer == nil {
		t.Error("ExperienceBuffer should be initialized")
	}
}

func TestLearnFromInteraction(t *testing.T) {
	ls := NewLearningSystem()

	// Learn from positive interaction
	ls.LearnFromInteraction(
		types.BehaviorIdle,
		types.BehaviorPlaying,
		1.0, // Positive reward
		types.BehaviorPlaying,
	)

	stats := ls.GetStats()
	if stats.TotalExperiences != 1 {
		t.Errorf("Expected 1 experience, got %d", stats.TotalExperiences)
	}

	if stats.SuccessfulActions != 1 {
		t.Errorf("Expected 1 successful action, got %d", stats.SuccessfulActions)
	}

	// Check Q-value was updated
	qValue := ls.GetQValue(types.BehaviorIdle, types.BehaviorPlaying)
	if qValue == 0 {
		t.Error("Q-value should be updated after learning")
	}
}

func TestQValueUpdate(t *testing.T) {
	ls := NewLearningSystem()

	// Initial Q-value should be 0
	initialQ := ls.GetQValue(types.BehaviorIdle, types.BehaviorPlaying)
	if initialQ != 0 {
		t.Errorf("Initial Q-value should be 0, got %.2f", initialQ)
	}

	// Learn with positive reward
	ls.LearnFromInteraction(
		types.BehaviorIdle,
		types.BehaviorPlaying,
		1.0,
		types.BehaviorPlaying,
	)

	// Q-value should increase
	updatedQ := ls.GetQValue(types.BehaviorIdle, types.BehaviorPlaying)
	if updatedQ <= initialQ {
		t.Errorf("Q-value should increase after positive reward, got %.2f", updatedQ)
	}

	// Learn with negative reward
	ls.LearnFromInteraction(
		types.BehaviorIdle,
		types.BehaviorEating,
		-0.5,
		types.BehaviorEating,
	)

	negativeQ := ls.GetQValue(types.BehaviorIdle, types.BehaviorEating)
	if negativeQ >= 0 {
		t.Errorf("Q-value should be negative after negative reward, got %.2f", negativeQ)
	}
}

func TestChooseAction(t *testing.T) {
	ls := NewLearningSystem()
	ls.SetExplorationRate(0.0) // Pure exploitation

	validActions := []types.BehaviorState{
		types.BehaviorIdle,
		types.BehaviorPlaying,
		types.BehaviorEating,
	}

	// Train with one clearly better action
	for i := 0; i < 10; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			1.0,
			types.BehaviorPlaying,
		)
	}

	for i := 0; i < 10; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorEating,
			-0.5,
			types.BehaviorEating,
		)
	}

	// Should consistently choose the better action
	chosenAction := ls.ChooseAction(types.BehaviorIdle, validActions)
	if chosenAction != types.BehaviorPlaying {
		t.Errorf("Expected to choose Playing (best action), got %v", chosenAction)
	}
}

func TestExploration(t *testing.T) {
	ls := NewLearningSystem()
	ls.SetExplorationRate(1.0) // Full exploration

	validActions := []types.BehaviorState{
		types.BehaviorIdle,
		types.BehaviorPlaying,
		types.BehaviorEating,
	}

	// With full exploration, should see variety in choices
	actionCounts := make(map[types.BehaviorState]int)
	for i := 0; i < 100; i++ {
		action := ls.ChooseAction(types.BehaviorIdle, validActions)
		actionCounts[action]++
	}

	// Should have chosen multiple different actions
	if len(actionCounts) < 2 {
		t.Error("With full exploration, should choose multiple different actions")
	}
}

func TestExplorationDecay(t *testing.T) {
	ls := NewLearningSystem()
	initialExploration := ls.ExplorationRate

	// Learn multiple times
	for i := 0; i < 50; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			1.0,
			types.BehaviorPlaying,
		)
	}

	stats := ls.GetStats()
	if stats.ExplorationRate >= initialExploration {
		t.Error("Exploration rate should decay over time")
	}

	if stats.ExplorationRate < ls.MinExplorationRate {
		t.Error("Exploration rate should not go below minimum")
	}
}

func TestExperienceBuffer(t *testing.T) {
	ls := NewLearningSystem()

	// Add experiences
	for i := 0; i < 10; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			float64(i),
			types.BehaviorPlaying,
		)
	}

	stats := ls.GetStats()
	if stats.ExperienceBufferSize != 10 {
		t.Errorf("Expected 10 experiences in buffer, got %d", stats.ExperienceBufferSize)
	}

	// Buffer should be capped at BufferSize
	for i := 0; i < 1500; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			1.0,
			types.BehaviorPlaying,
		)
	}

	stats = ls.GetStats()
	if stats.ExperienceBufferSize > ls.BufferSize {
		t.Errorf("Buffer size should be capped at %d, got %d", ls.BufferSize, stats.ExperienceBufferSize)
	}
}

func TestReplayExperiences(t *testing.T) {
	ls := NewLearningSystem()

	// Add some experiences
	for i := 0; i < 20; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			1.0,
			types.BehaviorPlaying,
		)
	}

	beforeQ := ls.GetQValue(types.BehaviorIdle, types.BehaviorPlaying)

	// Replay experiences
	ls.ReplayExperiences(10)

	afterQ := ls.GetQValue(types.BehaviorIdle, types.BehaviorPlaying)

	// Q-value should change (likely increase) after replay
	if afterQ == beforeQ {
		t.Error("Q-value should be reinforced after experience replay")
	}
}

func TestGetActionProbabilities(t *testing.T) {
	ls := NewLearningSystem()

	// Train with different rewards
	for i := 0; i < 10; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			1.0,
			types.BehaviorPlaying,
		)
	}

	for i := 0; i < 5; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorEating,
			0.5,
			types.BehaviorEating,
		)
	}

	probs := ls.GetActionProbabilities(types.BehaviorIdle)

	// Should have probabilities
	if len(probs) == 0 {
		t.Error("Should return action probabilities")
	}

	// Probabilities should sum to ~1.0
	sum := 0.0
	for _, prob := range probs {
		sum += prob
	}

	if math.Abs(sum-1.0) > 0.01 {
		t.Errorf("Probabilities should sum to 1.0, got %.2f", sum)
	}

	// Playing should have higher probability than eating
	if probs[types.BehaviorPlaying] <= probs[types.BehaviorEating] {
		t.Error("Action with higher reward should have higher probability")
	}
}

func TestGetBestAction(t *testing.T) {
	ls := NewLearningSystem()

	// Initially no best action
	_, exists := ls.GetBestAction(types.BehaviorIdle)
	if exists {
		t.Error("Should not have best action before learning")
	}

	// Learn some actions
	ls.LearnFromInteraction(
		types.BehaviorIdle,
		types.BehaviorPlaying,
		1.0,
		types.BehaviorPlaying,
	)

	ls.LearnFromInteraction(
		types.BehaviorIdle,
		types.BehaviorEating,
		0.2,
		types.BehaviorEating,
	)

	bestAction, exists := ls.GetBestAction(types.BehaviorIdle)
	if !exists {
		t.Error("Should have best action after learning")
	}

	if bestAction != types.BehaviorPlaying {
		t.Errorf("Best action should be Playing, got %v", bestAction)
	}
}

func TestAverageRewardTracking(t *testing.T) {
	ls := NewLearningSystem()

	// Learn with positive rewards
	for i := 0; i < 10; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			1.0,
			types.BehaviorPlaying,
		)
	}

	stats := ls.GetStats()
	if stats.AverageReward <= 0 {
		t.Error("Average reward should be positive after positive experiences")
	}

	// Add negative rewards
	for i := 0; i < 20; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorEating,
			-0.5,
			types.BehaviorEating,
		)
	}

	stats = ls.GetStats()
	// Average should decrease
	if stats.AverageReward >= 1.0 {
		t.Error("Average reward should decrease after negative experiences")
	}
}

func TestSuccessRate(t *testing.T) {
	ls := NewLearningSystem()

	// 70% positive, 30% negative
	for i := 0; i < 70; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			1.0,
			types.BehaviorPlaying,
		)
	}

	for i := 0; i < 30; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorEating,
			-0.5,
			types.BehaviorEating,
		)
	}

	stats := ls.GetStats()
	expectedRate := 0.7
	tolerance := 0.01

	if math.Abs(stats.SuccessRate-expectedRate) > tolerance {
		t.Errorf("Expected success rate ~%.2f, got %.2f", expectedRate, stats.SuccessRate)
	}
}

func TestLearningSystemReset(t *testing.T) {
	ls := NewLearningSystem()

	// Learn some things
	for i := 0; i < 50; i++ {
		ls.LearnFromInteraction(
			types.BehaviorIdle,
			types.BehaviorPlaying,
			1.0,
			types.BehaviorPlaying,
		)
	}

	stats := ls.GetStats()
	if stats.TotalExperiences == 0 {
		t.Error("Should have experiences before reset")
	}

	// Reset
	ls.Reset()

	stats = ls.GetStats()
	if stats.TotalExperiences != 0 {
		t.Error("Total experiences should be 0 after reset")
	}

	if stats.QTableSize != 0 {
		t.Error("Q-table should be empty after reset")
	}

	if stats.ExplorationRate != 1.0 {
		t.Error("Exploration rate should reset to 1.0")
	}
}

func TestSetLearningRate(t *testing.T) {
	ls := NewLearningSystem()

	ls.SetLearningRate(0.5)
	if ls.LearningRate != 0.5 {
		t.Errorf("Expected learning rate 0.5, got %.2f", ls.LearningRate)
	}

	// Invalid values should be ignored
	ls.SetLearningRate(-0.1)
	if ls.LearningRate != 0.5 {
		t.Error("Negative learning rate should be ignored")
	}

	ls.SetLearningRate(1.5)
	if ls.LearningRate != 0.5 {
		t.Error("Learning rate > 1.0 should be ignored")
	}
}

func TestSetExplorationRate(t *testing.T) {
	ls := NewLearningSystem()

	ls.SetExplorationRate(0.3)
	if ls.ExplorationRate != 0.3 {
		t.Errorf("Expected exploration rate 0.3, got %.2f", ls.ExplorationRate)
	}

	// Invalid values should be ignored
	ls.SetExplorationRate(-0.1)
	if ls.ExplorationRate != 0.3 {
		t.Error("Negative exploration rate should be ignored")
	}

	ls.SetExplorationRate(1.5)
	if ls.ExplorationRate != 0.3 {
		t.Error("Exploration rate > 1.0 should be ignored")
	}
}

func TestConcurrentLearning(t *testing.T) {
	ls := NewLearningSystem()

	done := make(chan bool)

	// Multiple goroutines learning simultaneously
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				ls.LearnFromInteraction(
					types.BehaviorIdle,
					types.BehaviorPlaying,
					1.0,
					types.BehaviorPlaying,
				)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := ls.GetStats()
	if stats.TotalExperiences != 1000 {
		t.Errorf("Expected 1000 total experiences, got %d", stats.TotalExperiences)
	}
}

func TestMultipleStates(t *testing.T) {
	ls := NewLearningSystem()

	// Learn different actions for different states
	ls.LearnFromInteraction(types.BehaviorIdle, types.BehaviorPlaying, 1.0, types.BehaviorPlaying)
	ls.LearnFromInteraction(types.BehaviorSleeping, types.BehaviorIdle, 0.8, types.BehaviorIdle)
	ls.LearnFromInteraction(types.BehaviorEating, types.BehaviorPlaying, 0.5, types.BehaviorPlaying)

	// Check different states have different best actions learned
	bestIdle, _ := ls.GetBestAction(types.BehaviorIdle)
	bestSleeping, _ := ls.GetBestAction(types.BehaviorSleeping)

	if bestIdle == types.BehaviorIdle {
		t.Error("Idle state should have learned a different best action")
	}

	if bestSleeping == types.BehaviorSleeping {
		t.Error("Sleeping state should have learned a different best action")
	}
}

func TestQTableGrowth(t *testing.T) {
	ls := NewLearningSystem()

	initialSize := ls.GetStats().QTableSize

	// Learn diverse state-action pairs
	states := []types.BehaviorState{
		types.BehaviorIdle,
		types.BehaviorSleeping,
		types.BehaviorEating,
		types.BehaviorPlaying,
	}

	actions := []types.BehaviorState{
		types.BehaviorPlaying,
		types.BehaviorEating,
		types.BehaviorSleeping,
	}

	for _, state := range states {
		for _, action := range actions {
			ls.LearnFromInteraction(state, action, 0.5, action)
		}
	}

	finalSize := ls.GetStats().QTableSize

	if finalSize <= initialSize {
		t.Error("Q-table should grow as new state-action pairs are learned")
	}

	expectedPairs := len(states) * len(actions)
	if finalSize != expectedPairs {
		t.Errorf("Expected Q-table size %d, got %d", expectedPairs, finalSize)
	}
}
