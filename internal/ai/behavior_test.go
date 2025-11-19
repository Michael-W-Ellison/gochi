package ai

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewBehaviorStateMachine(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	if bsm == nil {
		t.Fatal("BehaviorStateMachine should not be nil")
	}

	if bsm.CurrentState != types.BehaviorIdle {
		t.Errorf("Expected initial state Idle, got %v", bsm.CurrentState)
	}

	if bsm.StateChangeCount != 0 {
		t.Errorf("Expected 0 state changes, got %d", bsm.StateChangeCount)
	}
}

func TestForceState(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	bsm.ForceState(types.BehaviorPlaying)

	if bsm.GetCurrentState() != types.BehaviorPlaying {
		t.Errorf("Expected state Playing, got %v", bsm.GetCurrentState())
	}

	if bsm.GetPreviousState() != types.BehaviorIdle {
		t.Errorf("Expected previous state Idle, got %v", bsm.GetPreviousState())
	}

	if bsm.GetTransitionCount() != 1 {
		t.Errorf("Expected 1 transition, got %d", bsm.GetTransitionCount())
	}
}

func TestGetTimeInState(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	time.Sleep(50 * time.Millisecond)

	timeInState := bsm.GetTimeInState()

	if timeInState < 40*time.Millisecond {
		t.Errorf("Time in state should be at least 40ms, got %v", timeInState)
	}
}

func TestAddSimpleTransition(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	conditionMet := false
	condition := func() bool {
		return conditionMet
	}

	bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorPlaying, condition)

	// Condition not met - should not transition
	bsm.Update()
	if bsm.GetCurrentState() != types.BehaviorIdle {
		t.Error("Should not transition when condition not met")
	}

	// Condition met - should transition
	conditionMet = true
	bsm.Update()
	if bsm.GetCurrentState() != types.BehaviorPlaying {
		t.Error("Should transition when condition met")
	}
}

func TestAddTransitionWithProbability(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	// Add transition with 0.0 probability (never transitions)
	bsm.AddTransition(StateTransition{
		FromState:   types.BehaviorIdle,
		ToState:     types.BehaviorPlaying,
		Condition:   func() bool { return true },
		Probability: 0.0,
		Priority:    0,
	})

	// Try many times - should never transition
	for i := 0; i < 100; i++ {
		bsm.Update()
		if bsm.GetCurrentState() != types.BehaviorIdle {
			t.Error("Should not transition with 0.0 probability")
		}
	}
}

func TestTransitionPriority(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	// Add low priority transition
	bsm.AddTransition(StateTransition{
		FromState:   types.BehaviorIdle,
		ToState:     types.BehaviorPlaying,
		Condition:   func() bool { return true },
		Probability: 1.0,
		Priority:    1,
	})

	// Add high priority transition
	bsm.AddTransition(StateTransition{
		FromState:   types.BehaviorIdle,
		ToState:     types.BehaviorSleeping,
		Condition:   func() bool { return true },
		Probability: 1.0,
		Priority:    10,
	})

	// Higher priority should be checked first
	bsm.Update()

	if bsm.GetCurrentState() != types.BehaviorSleeping {
		t.Errorf("Expected Sleeping (higher priority), got %v", bsm.GetCurrentState())
	}
}

func TestRemoveTransition(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorPlaying, func() bool { return true })

	// Remove the transition
	bsm.RemoveTransition(types.BehaviorIdle, types.BehaviorPlaying)

	// Should not transition anymore
	bsm.Update()
	if bsm.GetCurrentState() != types.BehaviorIdle {
		t.Error("Transition should be removed")
	}
}

func TestClearTransitions(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorPlaying, func() bool { return true })
	bsm.AddSimpleTransition(types.BehaviorPlaying, types.BehaviorSleeping, func() bool { return true })

	bsm.ClearTransitions()

	// No transitions should exist
	bsm.Update()
	if bsm.GetCurrentState() != types.BehaviorIdle {
		t.Error("All transitions should be cleared")
	}
}

func TestGetTransitionHistory(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	// Make several transitions
	bsm.ForceState(types.BehaviorPlaying)
	bsm.ForceState(types.BehaviorSleeping)
	bsm.ForceState(types.BehaviorEating)

	history := bsm.GetTransitionHistory(10)

	if len(history) != 3 {
		t.Errorf("Expected 3 transitions in history, got %d", len(history))
	}

	// Check order (most recent last)
	if history[0].ToState != types.BehaviorPlaying {
		t.Errorf("First transition should be to Playing, got %v", history[0].ToState)
	}

	if history[2].ToState != types.BehaviorEating {
		t.Errorf("Last transition should be to Eating, got %v", history[2].ToState)
	}
}

func TestIsInState(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	if !bsm.IsInState(types.BehaviorIdle) {
		t.Error("Should be in Idle state")
	}

	bsm.ForceState(types.BehaviorPlaying)

	if !bsm.IsInState(types.BehaviorPlaying) {
		t.Error("Should be in Playing state")
	}

	if bsm.IsInState(types.BehaviorIdle) {
		t.Error("Should not be in Idle state")
	}
}

func TestWasInState(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	bsm.ForceState(types.BehaviorPlaying)

	if !bsm.WasInState(types.BehaviorIdle) {
		t.Error("Previous state should be Idle")
	}

	bsm.ForceState(types.BehaviorSleeping)

	if !bsm.WasInState(types.BehaviorPlaying) {
		t.Error("Previous state should be Playing")
	}
}

func TestSetGetStateData(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	testData := map[string]int{"energy": 100}
	bsm.SetStateData(types.BehaviorPlaying, testData)

	retrieved := bsm.GetStateData(types.BehaviorPlaying)

	if retrieved == nil {
		t.Fatal("State data should not be nil")
	}

	data, ok := retrieved.(map[string]int)
	if !ok {
		t.Fatal("State data should be map[string]int")
	}

	if data["energy"] != 100 {
		t.Errorf("Expected energy 100, got %d", data["energy"])
	}
}

func TestGetStats(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	bsm.ForceState(types.BehaviorPlaying)

	stats := bsm.GetStats()

	if stats.CurrentState != types.BehaviorPlaying {
		t.Errorf("Expected current state Playing, got %v", stats.CurrentState)
	}

	if stats.PreviousState != types.BehaviorIdle {
		t.Errorf("Expected previous state Idle, got %v", stats.PreviousState)
	}

	if stats.StateChangeCount != 1 {
		t.Errorf("Expected 1 state change, got %d", stats.StateChangeCount)
	}
}

func TestReset(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	// Make changes
	bsm.ForceState(types.BehaviorPlaying)
	bsm.ForceState(types.BehaviorSleeping)
	bsm.SetStateData(types.BehaviorPlaying, "test")

	// Reset
	bsm.Reset()

	if bsm.GetCurrentState() != types.BehaviorIdle {
		t.Error("State should be reset to Idle")
	}

	if bsm.GetTransitionCount() != 0 {
		t.Error("Transition count should be reset to 0")
	}

	history := bsm.GetTransitionHistory(10)
	if len(history) != 0 {
		t.Error("History should be cleared")
	}

	if bsm.GetStateData(types.BehaviorPlaying) != nil {
		t.Error("State data should be cleared")
	}
}

func TestCanTransitionTo(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	// Add transition
	bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorPlaying, func() bool { return true })

	if !bsm.CanTransitionTo(types.BehaviorPlaying) {
		t.Error("Should be able to transition to Playing")
	}

	if bsm.CanTransitionTo(types.BehaviorSleeping) {
		t.Error("Should not be able to transition to Sleeping")
	}
}

func TestCanTransitionToWithCondition(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	conditionMet := false

	bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorPlaying, func() bool { return conditionMet })

	if bsm.CanTransitionTo(types.BehaviorPlaying) {
		t.Error("Should not be able to transition when condition not met")
	}

	conditionMet = true

	if !bsm.CanTransitionTo(types.BehaviorPlaying) {
		t.Error("Should be able to transition when condition met")
	}
}

func TestGetPossibleTransitions(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	// Add multiple transitions
	bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorPlaying, func() bool { return true })
	bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorSleeping, func() bool { return true })
	bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorEating, func() bool { return false })

	possible := bsm.GetPossibleTransitions()

	// Should have 2 possible transitions (Eating condition is false)
	if len(possible) != 2 {
		t.Errorf("Expected 2 possible transitions, got %d", len(possible))
	}

	// Check that Playing and Sleeping are in the list
	hasPlaying := false
	hasSleeping := false
	for _, state := range possible {
		if state == types.BehaviorPlaying {
			hasPlaying = true
		}
		if state == types.BehaviorSleeping {
			hasSleeping = true
		}
	}

	if !hasPlaying || !hasSleeping {
		t.Error("Possible transitions should include Playing and Sleeping")
	}
}

func TestNoTransitionToSameState(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	initialCount := bsm.GetTransitionCount()

	// Force same state
	bsm.ForceState(types.BehaviorIdle)

	if bsm.GetTransitionCount() != initialCount {
		t.Error("Should not count transition to same state")
	}
}

func TestTransitionHistoryLimit(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	// Make more than 50 transitions
	for i := 0; i < 60; i++ {
		if i%2 == 0 {
			bsm.ForceState(types.BehaviorPlaying)
		} else {
			bsm.ForceState(types.BehaviorIdle)
		}
	}

	history := bsm.GetTransitionHistory(100)

	// Should be limited to 50
	if len(history) > 50 {
		t.Errorf("History should be limited to 50, got %d", len(history))
	}
}

func TestConcurrentAccess(t *testing.T) {
	bsm := NewBehaviorStateMachine()
	done := make(chan bool, 3)

	// Goroutine 1: Force states
	go func() {
		for i := 0; i < 50; i++ {
			bsm.ForceState(types.BehaviorPlaying)
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Add transitions
	go func() {
		for i := 0; i < 50; i++ {
			bsm.AddSimpleTransition(types.BehaviorIdle, types.BehaviorSleeping, func() bool { return true })
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 3: Read stats
	go func() {
		for i := 0; i < 50; i++ {
			_ = bsm.GetStats()
			_ = bsm.GetCurrentState()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should complete without panicking
}

func TestUpdateWithMultipleTransitions(t *testing.T) {
	bsm := NewBehaviorStateMachine()

	transitionOccurred := false

	// Add two transitions with different priorities
	bsm.AddTransition(StateTransition{
		FromState: types.BehaviorIdle,
		ToState:   types.BehaviorPlaying,
		Condition: func() bool {
			transitionOccurred = true
			return true
		},
		Probability: 1.0,
		Priority:    5,
	})

	bsm.AddTransition(StateTransition{
		FromState: types.BehaviorIdle,
		ToState:   types.BehaviorSleeping,
		Condition: func() bool {
			return true
		},
		Probability: 1.0,
		Priority:    1,
	})

	bsm.Update()

	// Higher priority (5) should execute
	if bsm.GetCurrentState() != types.BehaviorPlaying {
		t.Error("Higher priority transition should execute")
	}

	if !transitionOccurred {
		t.Error("Transition condition should have been checked")
	}
}
