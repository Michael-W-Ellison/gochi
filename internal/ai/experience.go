package ai

import (
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Experience records: the learning system remembers each kind of care the
// owner gives with the pet's mood at the time and how the pet took it.
// From them it predicts how the pet will take care in a given mood
// (PredictReward) and which way its personality is heading
// (GetPersonalityShift). The Q-learning above learns which of the pet's
// own behaviors its owner rewards; UpdateBehaviorWeights turns those
// values into the pet's best-rewarded behavior for each mood.

// ExperienceRecord is one kind of care and how the pet took it
type ExperienceRecord struct {
	Action    types.InteractionType `json:"action"`  // The care given
	Context   types.BehaviorState   `json:"context"` // The pet's mood: excited, happy, idle (neutral) or distressed
	Emotion   string                `json:"emotion"` // Its dominant emotion
	Reward    float64               `json:"reward"`  // How it took the care, -1.0 to 1.0
	Weight    float64               `json:"weight"`  // How much it counts (less for care repeated in quick succession)
	Timestamp time.Time             `json:"timestamp"`
}

// rewardEstimate is the running estimate of how the pet takes a kind of
// care in one mood
type rewardEstimate struct {
	Mean  float64 `json:"mean"`
	Count float64 `json:"count"` // Weighted experiences behind the estimate
}

// maxExperienceRecords is how many recent records are kept
const maxExperienceRecords = 200

// estimateWindow is how many weighted experiences the reward estimates
// average over before newer ones count for more
const estimateWindow = 20.0

// actionContext keys reward estimates
type actionContext struct {
	Action  types.InteractionType
	Context types.BehaviorState
}

// RecordExperience remembers a kind of care and how the pet took it, and
// updates the estimate PredictReward uses
func (ls *LearningSystem) RecordExperience(record ExperienceRecord) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if record.Weight <= 0 {
		record.Weight = 1
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}
	ls.Records = append(ls.Records, record)
	if over := len(ls.Records) - maxExperienceRecords; over > 0 {
		ls.Records = append(ls.Records[:0:0], ls.Records[over:]...)
	}

	if ls.Estimates == nil {
		ls.Estimates = make(map[actionContext]*rewardEstimate)
	}
	key := actionContext{record.Action, record.Context}
	estimate, ok := ls.Estimates[key]
	if !ok {
		estimate = &rewardEstimate{}
		ls.Estimates[key] = estimate
	}
	estimate.Count += record.Weight
	estimate.Mean += (record.Reward - estimate.Mean) * record.Weight / min(estimate.Count, estimateWindow)
}

// PredictReward returns how the pet is expected to take a kind of care in
// a mood (-1.0 to 1.0) and how sure the prediction is (0.0 with no
// experience, approaching 1.0 with many). With little experience in that
// mood it falls back on the care's estimates across all moods.
func (ls *LearningSystem) PredictReward(action types.InteractionType, context types.BehaviorState) (float64, float64) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	if estimate, ok := ls.Estimates[actionContext{action, context}]; ok && estimate.Count >= 3 {
		return estimate.Mean, estimate.Count / (estimate.Count + 5)
	}
	var sum, count float64
	for key, estimate := range ls.Estimates {
		if key.Action == action {
			sum += estimate.Mean * estimate.Count
			count += estimate.Count
		}
	}
	if count == 0 {
		return 0, 0
	}
	return sum / count, count / (count + 20) // Less sure: another mood's experience
}

// UpdateBehaviorWeights works out, for each mood, which of the pet's own
// behaviors its owner rewards most (from the learned Q-values) and the
// relative weight of each behavior. Only behaviors with a positive learned
// value count.
func (ls *LearningSystem) UpdateBehaviorWeights() {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	best := make(map[types.BehaviorState]types.BehaviorState)
	bestValue := make(map[types.BehaviorState]float64)
	weights := make(map[types.BehaviorState][]ActionPreference)
	totals := make(map[types.BehaviorState]float64)
	for pair, q := range ls.QTable {
		if q <= 0 {
			continue
		}
		if q > bestValue[pair.State] || (q == bestValue[pair.State] && pair.Action < best[pair.State]) {
			best[pair.State], bestValue[pair.State] = pair.Action, q
		}
		weights[pair.State] = append(weights[pair.State], ActionPreference{Action: pair.Action, AvgReward: q})
		totals[pair.State] += q
	}
	for state, prefs := range weights {
		for i := range prefs {
			prefs[i].Probability = prefs[i].AvgReward / totals[state]
		}
	}
	ls.BestAction = best
	ls.ActionPreferences = weights
}

// BestRewardedBehavior returns the behavior the pet has learned its owner
// rewards most in a mood (see UpdateBehaviorWeights)
func (ls *LearningSystem) BestRewardedBehavior(context types.BehaviorState) (types.BehaviorState, bool) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	behavior, ok := ls.BestAction[context]
	return behavior, ok
}

// PersonalityAdjustment is how much each trait is moving, by trait name
type PersonalityAdjustment map[string]float64

// personalityShiftRecords is how many recent records GetPersonalityShift
// looks at
const personalityShiftRecords = 50

// GetPersonalityShift returns which way the pet's recent experiences are
// moving its personality: for each record, the traits its care develops
// (traitEffects, per unit of reward) scaled by how the pet took it and how
// much it counted. Care the pet disliked makes it more anxious.
func (ls *LearningSystem) GetPersonalityShift(traitEffects map[types.InteractionType][]ExperienceData) PersonalityAdjustment {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	shift := PersonalityAdjustment{}
	start := max(len(ls.Records)-personalityShiftRecords, 0)
	for _, record := range ls.Records[start:] {
		if record.Reward <= 0 {
			shift["neuroticism"] += 0.05 * -record.Reward * record.Weight
			continue
		}
		for _, effect := range traitEffects[record.Action] {
			shift[effect.TraitAffected] += effect.Intensity * record.Reward * record.Weight
		}
	}
	return shift
}
