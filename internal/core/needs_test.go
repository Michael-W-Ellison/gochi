package core

import (
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
