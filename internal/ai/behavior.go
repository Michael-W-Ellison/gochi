package ai

import (
	"math/rand"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// StateTransitionCondition is a function that evaluates whether a transition should occur
type StateTransitionCondition func() bool

// StateTransition represents a possible state transition
type StateTransition struct {
	FromState   types.BehaviorState
	ToState     types.BehaviorState
	Condition   StateTransitionCondition
	Probability float64 // 0.0 to 1.0, chance of transition if condition is met
	Priority    int     // Higher priority transitions are checked first
}

// BehaviorStateMachine manages the behavioral state of a digital pet
type BehaviorStateMachine struct {
	mu sync.RWMutex

	CurrentState      types.BehaviorState
	PreviousState     types.BehaviorState
	StateEnteredAt    time.Time
	TimeInState       time.Duration
	StateChangeCount  int
	Transitions       []StateTransition
	TransitionHistory []StateTransitionRecord

	// State-specific data
	StateData map[types.BehaviorState]interface{}
}

// StateTransitionRecord records a state change for history
type StateTransitionRecord struct {
	FromState   types.BehaviorState
	ToState     types.BehaviorState
	Timestamp   time.Time
	TimeInPrevState time.Duration
}

// NewBehaviorStateMachine creates a new behavior state machine
func NewBehaviorStateMachine() *BehaviorStateMachine {
	now := time.Now()
	return &BehaviorStateMachine{
		CurrentState:      types.BehaviorIdle,
		PreviousState:     types.BehaviorIdle,
		StateEnteredAt:    now,
		TimeInState:       0,
		StateChangeCount:  0,
		Transitions:       make([]StateTransition, 0),
		TransitionHistory: make([]StateTransitionRecord, 0),
		StateData:         make(map[types.BehaviorState]interface{}),
	}
}

// Update processes potential state transitions
func (bsm *BehaviorStateMachine) Update() {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	// Update time in current state
	bsm.TimeInState = time.Since(bsm.StateEnteredAt)

	// Sort transitions by priority (higher first)
	// We'll check them in priority order
	for i := 0; i < len(bsm.Transitions); i++ {
		for j := i + 1; j < len(bsm.Transitions); j++ {
			if bsm.Transitions[j].Priority > bsm.Transitions[i].Priority {
				bsm.Transitions[i], bsm.Transitions[j] = bsm.Transitions[j], bsm.Transitions[i]
			}
		}
	}

	// Check each transition
	for _, transition := range bsm.Transitions {
		// Only check transitions from current state or wildcard (any state)
		if transition.FromState != bsm.CurrentState && transition.FromState != types.BehaviorState(-1) {
			continue
		}

		// Check condition
		if transition.Condition != nil && !transition.Condition() {
			continue
		}

		// Check probability
		if rand.Float64() > transition.Probability {
			continue
		}

		// Transition is valid - change state
		bsm.transitionTo(transition.ToState)
		return // Only one transition per update
	}
}

// transitionTo changes to a new state (must be called with lock held)
func (bsm *BehaviorStateMachine) transitionTo(newState types.BehaviorState) {
	if newState == bsm.CurrentState {
		return
	}

	// Record transition
	record := StateTransitionRecord{
		FromState:       bsm.CurrentState,
		ToState:         newState,
		Timestamp:       time.Now(),
		TimeInPrevState: bsm.TimeInState,
	}

	bsm.TransitionHistory = append(bsm.TransitionHistory, record)

	// Keep only last 50 transitions
	if len(bsm.TransitionHistory) > 50 {
		bsm.TransitionHistory = bsm.TransitionHistory[1:]
	}

	// Update state
	bsm.PreviousState = bsm.CurrentState
	bsm.CurrentState = newState
	bsm.StateEnteredAt = time.Now()
	bsm.TimeInState = 0
	bsm.StateChangeCount++
}

// ForceState immediately changes to a new state
func (bsm *BehaviorStateMachine) ForceState(newState types.BehaviorState) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	bsm.transitionTo(newState)
}

// GetCurrentState returns the current behavior state
func (bsm *BehaviorStateMachine) GetCurrentState() types.BehaviorState {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()
	return bsm.CurrentState
}

// GetPreviousState returns the previous behavior state
func (bsm *BehaviorStateMachine) GetPreviousState() types.BehaviorState {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()
	return bsm.PreviousState
}

// GetTimeInState returns how long the pet has been in the current state
func (bsm *BehaviorStateMachine) GetTimeInState() time.Duration {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()
	return time.Since(bsm.StateEnteredAt)
}

// AddTransition adds a state transition rule
func (bsm *BehaviorStateMachine) AddTransition(transition StateTransition) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	bsm.Transitions = append(bsm.Transitions, transition)
}

// AddSimpleTransition adds a transition with default probability (1.0) and priority (0)
func (bsm *BehaviorStateMachine) AddSimpleTransition(from, to types.BehaviorState, condition StateTransitionCondition) {
	bsm.AddTransition(StateTransition{
		FromState:   from,
		ToState:     to,
		Condition:   condition,
		Probability: 1.0,
		Priority:    0,
	})
}

// RemoveTransition removes a specific transition
func (bsm *BehaviorStateMachine) RemoveTransition(from, to types.BehaviorState) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	filtered := make([]StateTransition, 0)
	for _, t := range bsm.Transitions {
		if t.FromState != from || t.ToState != to {
			filtered = append(filtered, t)
		}
	}
	bsm.Transitions = filtered
}

// ClearTransitions removes all transitions
func (bsm *BehaviorStateMachine) ClearTransitions() {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	bsm.Transitions = make([]StateTransition, 0)
}

// GetTransitionCount returns the number of state changes
func (bsm *BehaviorStateMachine) GetTransitionCount() int {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()
	return bsm.StateChangeCount
}

// GetTransitionHistory returns the recent transition history
func (bsm *BehaviorStateMachine) GetTransitionHistory(limit int) []StateTransitionRecord {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()

	if limit <= 0 || limit > len(bsm.TransitionHistory) {
		limit = len(bsm.TransitionHistory)
	}

	start := len(bsm.TransitionHistory) - limit
	if start < 0 {
		start = 0
	}

	history := make([]StateTransitionRecord, limit)
	copy(history, bsm.TransitionHistory[start:])
	return history
}

// IsInState checks if currently in a specific state
func (bsm *BehaviorStateMachine) IsInState(state types.BehaviorState) bool {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()
	return bsm.CurrentState == state
}

// WasInState checks if the previous state was a specific state
func (bsm *BehaviorStateMachine) WasInState(state types.BehaviorState) bool {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()
	return bsm.PreviousState == state
}

// SetStateData stores arbitrary data associated with a state
func (bsm *BehaviorStateMachine) SetStateData(state types.BehaviorState, data interface{}) {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()
	bsm.StateData[state] = data
}

// GetStateData retrieves data associated with a state
func (bsm *BehaviorStateMachine) GetStateData(state types.BehaviorState) interface{} {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()
	return bsm.StateData[state]
}

// GetStats returns statistics about the state machine
func (bsm *BehaviorStateMachine) GetStats() BehaviorStats {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()

	return BehaviorStats{
		CurrentState:     bsm.CurrentState,
		PreviousState:    bsm.PreviousState,
		TimeInState:      time.Since(bsm.StateEnteredAt),
		StateChangeCount: bsm.StateChangeCount,
		TransitionCount:  len(bsm.Transitions),
	}
}

// BehaviorStats contains statistics about behavior state
type BehaviorStats struct {
	CurrentState     types.BehaviorState
	PreviousState    types.BehaviorState
	TimeInState      time.Duration
	StateChangeCount int
	TransitionCount  int
}

// Reset resets the state machine to initial state
func (bsm *BehaviorStateMachine) Reset() {
	bsm.mu.Lock()
	defer bsm.mu.Unlock()

	now := time.Now()
	bsm.CurrentState = types.BehaviorIdle
	bsm.PreviousState = types.BehaviorIdle
	bsm.StateEnteredAt = now
	bsm.TimeInState = 0
	bsm.StateChangeCount = 0
	bsm.TransitionHistory = make([]StateTransitionRecord, 0)
	bsm.StateData = make(map[types.BehaviorState]interface{})
}

// CanTransitionTo checks if a transition to a specific state is possible
func (bsm *BehaviorStateMachine) CanTransitionTo(toState types.BehaviorState) bool {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()

	for _, transition := range bsm.Transitions {
		if (transition.FromState == bsm.CurrentState || transition.FromState == types.BehaviorState(-1)) &&
			transition.ToState == toState {
			if transition.Condition == nil || transition.Condition() {
				return true
			}
		}
	}

	return false
}

// GetPossibleTransitions returns all states that can be transitioned to from current state
func (bsm *BehaviorStateMachine) GetPossibleTransitions() []types.BehaviorState {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()

	possible := make(map[types.BehaviorState]bool)

	for _, transition := range bsm.Transitions {
		if transition.FromState == bsm.CurrentState || transition.FromState == types.BehaviorState(-1) {
			if transition.Condition == nil || transition.Condition() {
				possible[transition.ToState] = true
			}
		}
	}

	result := make([]types.BehaviorState, 0, len(possible))
	for state := range possible {
		result = append(result, state)
	}

	return result
}
