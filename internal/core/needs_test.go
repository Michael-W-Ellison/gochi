package core

import (
	"errors"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func needOf(pet *DigitalPet, need types.NeedType) float64 {
	pet.mu.Lock()
	defer pet.mu.Unlock()
	return pet.needLevelLocked(need)
}

func TestPsychologicalNeedsDecayAndAreMet(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.Update(2)

	exercise := needOf(pet, types.NeedExercise)
	if exercise >= 1.0 {
		t.Fatal("Exercise need should decay over time")
	}

	pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
	if needOf(pet, types.NeedExercise) <= exercise {
		t.Error("Playing should meet the exercise need")
	}

	affection := needOf(pet, types.NeedAffection)
	pet.ProcessUserInteraction(types.InteractionPetting, 1.0)
	if needOf(pet, types.NeedAffection) <= affection {
		t.Error("Petting should meet the affection need")
	}

	if levels := pet.PsychologicalNeeds(); len(levels) != 5 || levels[0].Need != "Exercise" {
		t.Errorf("PsychologicalNeeds = %+v", levels)
	}
}

func TestPhysicalNeedsFollowVitalStats(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Biology.Vitals.Hydration = 0.3
	pet.Update(0.01)

	if got := needOf(pet, types.NeedThirst); got != pet.Biology.Vitals.Hydration {
		t.Errorf("Thirst need %f should match hydration %f", got, pet.Biology.Vitals.Hydration)
	}

	pet.Injured = true
	pet.Update(0.01)
	if needOf(pet, types.NeedMedicalCare) > 0.3 {
		t.Error("An injury should make medical care urgent")
	}
}

func TestUnmetNeedsMakePetUnhappy(t *testing.T) {
	cared := NewDigitalPet("Cared", "user123")
	ignored := NewDigitalPet("Ignored", "user123")
	for _, pet := range []*DigitalPet{cared, ignored} {
		setCalmHealthyState(pet)
	}
	for _, need := range psychologicalNeeds {
		ignored.Needs.SetNeedLevel(need, 0.0)
	}

	cared.Update(1)
	ignored.Update(1)

	if ignored.Biology.Vitals.Happiness >= cared.Biology.Vitals.Happiness {
		t.Error("A pet with unmet needs should become unhappier")
	}
}

func TestNeedsDriveRequests(t *testing.T) {
	tests := []struct {
		need   types.NeedType
		action string
	}{
		{types.NeedExercise, "play"},
		{types.NeedMentalStimulation, "train"},
		{types.NeedExploration, "explore"},
		{types.NeedAffection, "pet"},
	}
	for _, tt := range tests {
		pet := NewDigitalPet("Biscuit", "user123")
		setCalmHealthyState(pet)
		pet.Needs.SetNeedLevel(tt.need, 0.1)

		if r := pet.CurrentRequest(); r == nil || r.Action != tt.action {
			t.Errorf("Low %s should ask for %q, got %+v", tt.need, tt.action, r)
		}
	}
}

func TestNeedsSurviveSaveLoad(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Needs.SetNeedLevel(types.NeedSocial, 0.25)

	data, _ := pet.Save()
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := needOf(loaded, types.NeedSocial); got != 0.25 {
		t.Errorf("Social need after load = %f, want 0.25", got)
	}

	// Pets saved before needs were tracked get fresh needs
	loaded.Needs = nil
	if levels := loaded.PsychologicalNeeds(); levels[0].Level != 1.0 {
		t.Errorf("Missing needs should start met, got %+v", levels[0])
	}
}

func TestCareRulesCanRefuse(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Biology.Vitals.Nutrition = 0.95

	r := pet.Interact(types.InteractionFeeding, 1.0)
	if r.Applied || r.Refusal != "Biscuit is not hungry." || pet.TotalInteractions != 0 || r.PointsEarned != 0 {
		t.Errorf("A full pet should turn food down with nothing applied: %+v", r)
	}

	pet.Biology.Vitals.Cleanliness = 1.0
	if r := pet.Interact(types.InteractionGrooming, 1.0); r.Applied || r.Refusal != "Biscuit is already clean." {
		t.Errorf("A clean pet should not need grooming: %+v", r)
	}
}

func TestSleepingPetRestsAndWakes(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.Biology.Vitals.Fatigue = 0.9
	pet.Update(0.01)
	if pet.CurrentBehavior != types.BehaviorSleeping {
		t.Fatalf("A tired pet should sleep, got %s", pet.CurrentBehavior)
	}

	// Play, petting and grooming wait until it wakes, and say so
	for _, care := range []types.InteractionType{types.InteractionPlaying, types.InteractionPetting,
		types.InteractionGrooming, types.InteractionTraining} {
		if r := pet.Interact(care, 1.0); r.Applied || !r.Asleep || r.Refusal != "Biscuit is asleep." || r.WakeIn <= 0 {
			t.Errorf("A sleeping pet should turn down %s: %+v", care, r)
		}
	}
	if r, err := pet.PetOn("cheeks"); err != nil || r.Applied || !r.Asleep {
		t.Errorf("Petting a sleeping pet on its cheeks: %+v %v", r, err)
	}
	if err := pet.StartHideAndSeek(); err != ErrSleeping {
		t.Errorf("A sleeping pet should not play hide and seek: %v", err)
	}
	// Food and water still work
	pet.Biology.Vitals.Nutrition, pet.Biology.Vitals.Hydration = 0.3, 0.3
	for _, care := range []types.InteractionType{types.InteractionFeeding, types.InteractionDrinking} {
		if r := pet.Interact(care, 1.0); !r.Applied || r.Asleep {
			t.Errorf("A sleeping pet should still take %s: %+v", care, r)
		}
	}

	// Sleep continues until rested, then the pet wakes, about when the
	// estimate said
	estimate := pet.WakeIn()
	slept := 0
	for pet.CurrentBehavior == types.BehaviorSleeping && slept < 100 {
		pet.Update(0.1)
		pet.Biology.Vitals.Hydration, pet.Biology.Vitals.Nutrition = 1, 1
		slept++
	}
	if pet.CurrentBehavior == types.BehaviorSleeping {
		t.Fatal("A sleeping pet should eventually wake up")
	}
	if slept < 5 {
		t.Errorf("The pet should sleep until rested, but woke after %d updates", slept)
	}
	if pet.Biology.Vitals.Fatigue > wakeFatigue {
		t.Errorf("A rested pet should have low fatigue, got %f", pet.Biology.Vitals.Fatigue)
	}
	t.Logf("Woke after %.1f game days; estimated %.2f", float64(slept)*0.1, estimate)
	if actual := float64(slept) * 0.1; actual < estimate*0.5 || actual > estimate*2 {
		t.Errorf("The pet woke after %.1f game days; the estimate was %.1f", actual, estimate)
	}
	if pet.WakeIn() != 0 {
		t.Error("An awake pet has no time to wake in")
	}
}

func TestBedImprovesSleep(t *testing.T) {
	rest := func(withBed bool) float64 {
		pet := NewDigitalPet("Biscuit", "user123")
		if withBed {
			pet.FriendshipPoints = 1000
			if err := pet.BuyItem("cozy_bed"); err != nil {
				t.Fatal(err)
			}
		}
		pet.CurrentBehavior = types.BehaviorSleeping
		pet.Biology.Vitals.Fatigue = 0.9
		pet.mu.Lock()
		pet.Biology.ProcessRest(0.1, pet.sleepQualityLocked())
		pet.mu.Unlock()
		return 0.9 - pet.Biology.Vitals.Fatigue
	}
	if rest(true) <= rest(false) {
		t.Error("A bed should make sleep more restful")
	}
}

func TestRefusedTrainingTeachesNothing(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Biology.Vitals.Stress = 0.9

	_, err := pet.TrainTrick("sit")
	var refused RefusedError
	if !errors.As(err, &refused) || refused.Reason != "Biscuit is too stressed for training." {
		t.Errorf("A stressed pet should refuse training: %v", err)
	}
	if pet.Tricks()[0].Skill != 0 {
		t.Error("Refused training should not improve the trick")
	}
}

func TestWalkMeetsExerciseNeed(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	pet.Update(2)
	exercise := needOf(pet, types.NeedExercise)
	energy := pet.Biology.Vitals.Energy

	sessions, err := pet.Walk(250)
	if err != nil || sessions != 0.5 {
		t.Fatalf("A 250 m walk should count as half a play session, got %f, %v", sessions, err)
	}
	if needOf(pet, types.NeedExercise) <= exercise || pet.Biology.Vitals.Energy >= energy {
		t.Error("A walk should meet the exercise need and use energy")
	}
	if sessions, _ := pet.Walk(10000); sessions != maxWalkSessions {
		t.Errorf("A long walk should be capped at %f sessions, got %f", maxWalkSessions, sessions)
	}

	pet.CurrentBehavior = types.BehaviorSleeping
	if _, err := pet.Walk(500); !errors.Is(err, ErrSleeping) {
		t.Errorf("A sleeping pet can't walk, got %v", err)
	}
}
