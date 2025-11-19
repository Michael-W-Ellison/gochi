package ai

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// LearningSystem implements reinforcement learning for behavior adaptation
type LearningSystem struct {
	mu sync.RWMutex

	// Q-Learning parameters
	QTable            map[StateActionPair]float64 // Q-values for state-action pairs
	LearningRate      float64                     // Alpha: how much new info overrides old (0-1)
	DiscountFactor    float64                     // Gamma: importance of future rewards (0-1)
	ExplorationRate   float64                     // Epsilon: exploration vs exploitation (0-1)
	ExplorationDecay  float64                     // Rate at which exploration decreases
	MinExplorationRate float64                    // Minimum exploration rate

	// Experience tracking
	ExperienceBuffer  []Experience // Replay buffer for learning
	BufferSize        int          // Maximum size of experience buffer
	TotalExperiences  int          // Total experiences collected
	SuccessfulActions int          // Count of positively rewarded actions
	FailedActions     int          // Count of negatively rewarded actions

	// Learning statistics
	AverageReward     float64       // Moving average of rewards
	BestAction        map[types.BehaviorState]types.BehaviorState // Best learned action for each state
	ActionPreferences map[types.BehaviorState][]ActionPreference  // Learned preferences per state

	// Temporal difference learning
	LastState  types.BehaviorState
	LastAction types.BehaviorState
	LastReward float64
}

// StateActionPair represents a state-action combination for Q-learning
type StateActionPair struct {
	State  types.BehaviorState
	Action types.BehaviorState
}

// Experience represents a single learning experience
type Experience struct {
	State      types.BehaviorState
	Action     types.BehaviorState
	Reward     float64
	NextState  types.BehaviorState
	Timestamp  time.Time
}

// ActionPreference represents learned preference for an action in a state
type ActionPreference struct {
	Action      types.BehaviorState
	Probability float64
	AvgReward   float64
	TimesChosen int
}

// NewLearningSystem creates a new learning system with default parameters
func NewLearningSystem() *LearningSystem {
	return &LearningSystem{
		QTable:             make(map[StateActionPair]float64),
		LearningRate:       0.1,  // Standard Q-learning rate
		DiscountFactor:     0.9,  // Value future rewards highly
		ExplorationRate:    1.0,  // Start with full exploration
		ExplorationDecay:   0.995, // Gradual decay
		MinExplorationRate: 0.05, // Always keep some exploration
		ExperienceBuffer:   make([]Experience, 0, 1000),
		BufferSize:         1000,
		BestAction:         make(map[types.BehaviorState]types.BehaviorState),
		ActionPreferences:  make(map[types.BehaviorState][]ActionPreference),
		AverageReward:      0.0,
	}
}

// LearnFromInteraction processes a user interaction and updates Q-values
func (ls *LearningSystem) LearnFromInteraction(
	currentState types.BehaviorState,
	action types.BehaviorState,
	reward float64,
	nextState types.BehaviorState,
) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// Create experience
	exp := Experience{
		State:     currentState,
		Action:    action,
		Reward:    reward,
		NextState: nextState,
		Timestamp: time.Now(),
	}

	// Add to experience buffer
	ls.addExperience(exp)

	// Update Q-value using Q-learning formula:
	// Q(s,a) = Q(s,a) + α * [r + γ * max(Q(s',a')) - Q(s,a)]
	ls.updateQValue(currentState, action, reward, nextState)

	// Update statistics
	ls.updateStatistics(reward)

	// Update action preferences
	ls.updateActionPreferences(currentState, action, reward)

	// Decay exploration rate
	if ls.ExplorationRate > ls.MinExplorationRate {
		ls.ExplorationRate *= ls.ExplorationDecay
		if ls.ExplorationRate < ls.MinExplorationRate {
			ls.ExplorationRate = ls.MinExplorationRate
		}
	}

	ls.TotalExperiences++
	if reward > 0 {
		ls.SuccessfulActions++
	} else if reward < 0 {
		ls.FailedActions++
	}
}

// updateQValue updates the Q-value for a state-action pair
func (ls *LearningSystem) updateQValue(
	state types.BehaviorState,
	action types.BehaviorState,
	reward float64,
	nextState types.BehaviorState,
) {
	pair := StateActionPair{State: state, Action: action}
	currentQ := ls.QTable[pair]

	// Find maximum Q-value for next state
	maxNextQ := ls.getMaxQValue(nextState)

	// Q-learning update
	newQ := currentQ + ls.LearningRate*(reward+ls.DiscountFactor*maxNextQ-currentQ)
	ls.QTable[pair] = newQ
}

// getMaxQValue returns the maximum Q-value for all actions in a state
func (ls *LearningSystem) getMaxQValue(state types.BehaviorState) float64 {
	maxQ := -math.MaxFloat64

	// Check all possible actions
	for action := types.BehaviorIdle; action <= types.BehaviorExcited; action++ {
		pair := StateActionPair{State: state, Action: action}
		if q, exists := ls.QTable[pair]; exists {
			if q > maxQ {
				maxQ = q
			}
		}
	}

	if maxQ == -math.MaxFloat64 {
		return 0.0 // No Q-values yet
	}
	return maxQ
}

// ChooseAction selects an action using epsilon-greedy strategy
func (ls *LearningSystem) ChooseAction(currentState types.BehaviorState, validActions []types.BehaviorState) types.BehaviorState {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	if len(validActions) == 0 {
		return types.BehaviorIdle
	}

	// Epsilon-greedy: explore vs exploit
	if rand.Float64() < ls.ExplorationRate {
		// Explore: choose random action
		return validActions[rand.Intn(len(validActions))]
	}

	// Exploit: choose best action based on Q-values
	bestAction := validActions[0]
	bestQ := -math.MaxFloat64

	for _, action := range validActions {
		pair := StateActionPair{State: currentState, Action: action}
		q := ls.QTable[pair] // Default 0.0 if not exists
		if q > bestQ {
			bestQ = q
			bestAction = action
		}
	}

	return bestAction
}

// GetActionProbabilities returns learned probabilities for each action in a state
func (ls *LearningSystem) GetActionProbabilities(state types.BehaviorState) map[types.BehaviorState]float64 {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	probabilities := make(map[types.BehaviorState]float64)

	// Use softmax on Q-values to get probabilities
	qValues := make(map[types.BehaviorState]float64)
	sumExp := 0.0
	temperature := 1.0 // Temperature parameter for softmax

	for action := types.BehaviorIdle; action <= types.BehaviorExcited; action++ {
		pair := StateActionPair{State: state, Action: action}
		q := ls.QTable[pair]
		expQ := math.Exp(q / temperature)
		qValues[action] = expQ
		sumExp += expQ
	}

	// Normalize to probabilities
	if sumExp > 0 {
		for action, expQ := range qValues {
			probabilities[action] = expQ / sumExp
		}
	}

	return probabilities
}

// addExperience adds an experience to the replay buffer
func (ls *LearningSystem) addExperience(exp Experience) {
	if len(ls.ExperienceBuffer) >= ls.BufferSize {
		// Remove oldest experience
		ls.ExperienceBuffer = ls.ExperienceBuffer[1:]
	}
	ls.ExperienceBuffer = append(ls.ExperienceBuffer, exp)
}

// ReplayExperiences performs experience replay to reinforce learning
func (ls *LearningSystem) ReplayExperiences(batchSize int) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if len(ls.ExperienceBuffer) < batchSize {
		batchSize = len(ls.ExperienceBuffer)
	}

	if batchSize == 0 {
		return
	}

	// Sample random experiences
	for i := 0; i < batchSize; i++ {
		idx := rand.Intn(len(ls.ExperienceBuffer))
		exp := ls.ExperienceBuffer[idx]

		// Re-learn from this experience
		ls.updateQValue(exp.State, exp.Action, exp.Reward, exp.NextState)
	}
}

// updateStatistics updates learning statistics
func (ls *LearningSystem) updateStatistics(reward float64) {
	// Moving average of rewards (exponential moving average)
	alpha := 0.1
	ls.AverageReward = ls.AverageReward*(1-alpha) + reward*alpha
}

// updateActionPreferences updates learned preferences for actions
func (ls *LearningSystem) updateActionPreferences(state types.BehaviorState, action types.BehaviorState, reward float64) {
	prefs := ls.ActionPreferences[state]

	// Find existing preference or create new
	found := false
	for i := range prefs {
		if prefs[i].Action == action {
			// Update existing preference
			prefs[i].TimesChosen++
			prefs[i].AvgReward = (prefs[i].AvgReward*float64(prefs[i].TimesChosen-1) + reward) / float64(prefs[i].TimesChosen)
			found = true
			break
		}
	}

	if !found {
		// Add new preference
		prefs = append(prefs, ActionPreference{
			Action:      action,
			Probability: 0.0, // Will be calculated
			AvgReward:   reward,
			TimesChosen: 1,
		})
	}

	// Calculate probabilities using softmax on average rewards
	ls.ActionPreferences[state] = ls.calculatePreferenceProbabilities(prefs)

	// Update best action for this state
	bestAction := action
	bestReward := reward
	for _, pref := range prefs {
		if pref.AvgReward > bestReward {
			bestReward = pref.AvgReward
			bestAction = pref.Action
		}
	}
	ls.BestAction[state] = bestAction
}

// calculatePreferenceProbabilities calculates probabilities for action preferences
func (ls *LearningSystem) calculatePreferenceProbabilities(prefs []ActionPreference) []ActionPreference {
	if len(prefs) == 0 {
		return prefs
	}

	temperature := 1.0
	sumExp := 0.0

	for _, pref := range prefs {
		sumExp += math.Exp(pref.AvgReward / temperature)
	}

	for i := range prefs {
		if sumExp > 0 {
			prefs[i].Probability = math.Exp(prefs[i].AvgReward/temperature) / sumExp
		} else {
			prefs[i].Probability = 1.0 / float64(len(prefs))
		}
	}

	return prefs
}

// GetBestAction returns the best learned action for a state
func (ls *LearningSystem) GetBestAction(state types.BehaviorState) (types.BehaviorState, bool) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	action, exists := ls.BestAction[state]
	return action, exists
}

// GetQValue returns the Q-value for a state-action pair
func (ls *LearningSystem) GetQValue(state types.BehaviorState, action types.BehaviorState) float64 {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	pair := StateActionPair{State: state, Action: action}
	return ls.QTable[pair]
}

// GetStats returns learning statistics
func (ls *LearningSystem) GetStats() LearningStats {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	successRate := 0.0
	if ls.TotalExperiences > 0 {
		successRate = float64(ls.SuccessfulActions) / float64(ls.TotalExperiences)
	}

	return LearningStats{
		TotalExperiences:   ls.TotalExperiences,
		SuccessfulActions:  ls.SuccessfulActions,
		FailedActions:      ls.FailedActions,
		AverageReward:      ls.AverageReward,
		ExplorationRate:    ls.ExplorationRate,
		QTableSize:         len(ls.QTable),
		ExperienceBufferSize: len(ls.ExperienceBuffer),
		SuccessRate:        successRate,
	}
}

// Reset clears all learning data
func (ls *LearningSystem) Reset() {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.QTable = make(map[StateActionPair]float64)
	ls.ExperienceBuffer = make([]Experience, 0, ls.BufferSize)
	ls.BestAction = make(map[types.BehaviorState]types.BehaviorState)
	ls.ActionPreferences = make(map[types.BehaviorState][]ActionPreference)
	ls.TotalExperiences = 0
	ls.SuccessfulActions = 0
	ls.FailedActions = 0
	ls.AverageReward = 0.0
	ls.ExplorationRate = 1.0
}

// SetLearningRate adjusts the learning rate
func (ls *LearningSystem) SetLearningRate(rate float64) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if rate >= 0 && rate <= 1.0 {
		ls.LearningRate = rate
	}
}

// SetExplorationRate adjusts the exploration rate
func (ls *LearningSystem) SetExplorationRate(rate float64) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if rate >= 0 && rate <= 1.0 {
		ls.ExplorationRate = rate
	}
}

// LearningStats contains statistics about the learning system
type LearningStats struct {
	TotalExperiences     int
	SuccessfulActions    int
	FailedActions        int
	AverageReward        float64
	ExplorationRate      float64
	QTableSize           int
	ExperienceBufferSize int
	SuccessRate          float64
}
