package ai

import (
	"encoding/json"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestPredictRewardByMood(t *testing.T) {
	ls := NewLearningSystem()
	if r, confidence := ls.PredictReward(types.InteractionPlaying, types.BehaviorHappy); r != 0 || confidence != 0 {
		t.Errorf("No experience: %v, %v", r, confidence)
	}
	for i := 0; i < 5; i++ {
		ls.RecordExperience(ExperienceRecord{Action: types.InteractionPlaying, Context: types.BehaviorHappy, Reward: 0.8})
		ls.RecordExperience(ExperienceRecord{Action: types.InteractionPlaying, Context: types.BehaviorDistressed, Reward: -0.4})
	}
	happy, sure := ls.PredictReward(types.InteractionPlaying, types.BehaviorHappy)
	upset, _ := ls.PredictReward(types.InteractionPlaying, types.BehaviorDistressed)
	if happy < 0.7 || upset > -0.3 || sure < 0.4 {
		t.Errorf("Play should be enjoyed when happy (%.2f, sure %.2f) and not when upset (%.2f)", happy, sure, upset)
	}

	// A mood with little experience falls back on the others, less surely
	neutral, unsure := ls.PredictReward(types.InteractionPlaying, types.BehaviorIdle)
	if neutral < 0 || neutral > 0.4 || unsure >= sure {
		t.Errorf("Fallback prediction: %.2f (sure %.2f)", neutral, unsure)
	}
}

func TestRepeatedCareCountsLessInPredictions(t *testing.T) {
	ls := NewLearningSystem()
	for i := 0; i < 10; i++ {
		ls.RecordExperience(ExperienceRecord{Action: types.InteractionPetting, Context: types.BehaviorHappy, Reward: 0.5, Weight: 1})
	}
	ls.RecordExperience(ExperienceRecord{Action: types.InteractionPetting, Context: types.BehaviorHappy, Reward: -1, Weight: 0.1})
	if r, _ := ls.PredictReward(types.InteractionPetting, types.BehaviorHappy); r < 0.4 {
		t.Errorf("A barely-counting repeat should hardly move the prediction, got %.2f", r)
	}
}

func TestUpdateBehaviorWeights(t *testing.T) {
	ls := NewLearningSystem()
	ls.QTable[StateActionPair{types.BehaviorHappy, types.BehaviorPlaying}] = 0.6
	ls.QTable[StateActionPair{types.BehaviorHappy, types.BehaviorSocialInteraction}] = 0.2
	ls.QTable[StateActionPair{types.BehaviorHappy, types.BehaviorInvestigating}] = -0.5
	ls.UpdateBehaviorWeights()

	if best, ok := ls.BestRewardedBehavior(types.BehaviorHappy); !ok || best != types.BehaviorPlaying {
		t.Errorf("Best rewarded when happy: %v, %v", best, ok)
	}
	if _, ok := ls.BestRewardedBehavior(types.BehaviorDistressed); ok {
		t.Error("Nothing has been rewarded when upset")
	}
	prefs := ls.ActionPreferences[types.BehaviorHappy]
	if len(prefs) != 2 {
		t.Fatalf("Only rewarded behaviors get a weight: %+v", prefs)
	}
	total := 0.0
	for _, p := range prefs {
		total += p.Probability
	}
	if total < 0.999 || total > 1.001 {
		t.Errorf("Weights should add up to 1, got %.3f", total)
	}
}

func TestGetPersonalityShift(t *testing.T) {
	ls := NewLearningSystem()
	effects := map[types.InteractionType][]ExperienceData{
		types.InteractionPlaying: {{TraitAffected: "playfulness", Intensity: 0.2}},
	}
	for i := 0; i < 10; i++ {
		ls.RecordExperience(ExperienceRecord{Action: types.InteractionPlaying, Reward: 1})
	}
	ls.RecordExperience(ExperienceRecord{Action: types.InteractionDiscipline, Reward: -1})

	shift := ls.GetPersonalityShift(effects)
	if shift["playfulness"] < 1.99 || shift["neuroticism"] <= 0 {
		t.Errorf("Shift = %v", shift)
	}
}

func TestExperiencesSurviveSaveLoad(t *testing.T) {
	ls := NewLearningSystem()
	ls.QTable[StateActionPair{types.BehaviorHappy, types.BehaviorPlaying}] = 0.6
	for i := 0; i < 4; i++ {
		ls.RecordExperience(ExperienceRecord{Action: types.InteractionPlaying, Context: types.BehaviorHappy, Reward: 0.7})
	}
	data, err := json.Marshal(ls)
	if err != nil {
		t.Fatal(err)
	}
	var loaded LearningSystem
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	want, _ := ls.PredictReward(types.InteractionPlaying, types.BehaviorHappy)
	if got, _ := loaded.PredictReward(types.InteractionPlaying, types.BehaviorHappy); got != want || len(loaded.Records) != 4 {
		t.Errorf("Loaded prediction %.2f, want %.2f; %d records", got, want, len(loaded.Records))
	}
	if best, ok := loaded.BestRewardedBehavior(types.BehaviorHappy); !ok || best != types.BehaviorPlaying {
		t.Error("Behavior weights should be rebuilt on load")
	}
}

func TestExperienceRecordsAreBounded(t *testing.T) {
	ls := NewLearningSystem()
	for i := 0; i < maxExperienceRecords+50; i++ {
		ls.RecordExperience(ExperienceRecord{Action: types.InteractionFeeding, Reward: 0.1})
	}
	if len(ls.Records) != maxExperienceRecords {
		t.Errorf("Records = %d, want %d", len(ls.Records), maxExperienceRecords)
	}
}
