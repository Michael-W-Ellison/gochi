package core

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewDigitalPet(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	if pet.Name != "TestPet" {
		t.Errorf("Expected name 'TestPet', got '%s'", pet.Name)
	}

	if pet.Owner != "user123" {
		t.Errorf("Expected owner 'user123', got '%s'", pet.Owner)
	}

	if pet.Biology == nil {
		t.Error("Biology should be initialized")
	}

	if pet.Personality == nil {
		t.Error("Personality should be initialized")
	}

	if pet.Memory == nil {
		t.Error("Memory should be initialized")
	}

	if pet.Emotions == nil {
		t.Error("Emotions should be initialized")
	}

	if pet.Relationships == nil {
		t.Error("Relationships should be initialized")
	}

	if !pet.IsAlive() {
		t.Error("New pet should be alive")
	}
}

func TestDigitalPetUpdate(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	initialAge := pet.GetAge()

	pet.Update(1.0)

	if pet.GetAge() <= initialAge {
		t.Error("Pet age should increase after update")
	}
}

func TestProcessUserInteraction(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	initialHappiness := pet.Biology.Vitals.Happiness
	initialInteractions := pet.TotalInteractions

	pet.ProcessUserInteraction(types.InteractionPetting, 1.0)

	if pet.Biology.Vitals.Happiness <= initialHappiness {
		t.Error("Happiness should increase after petting")
	}

	if pet.TotalInteractions != initialInteractions+1 {
		t.Error("Total interactions should increment")
	}
}

func TestGetCurrentStatus(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	status := pet.GetCurrentStatus()

	if status.Name != "TestPet" {
		t.Errorf("Expected status name 'TestPet', got '%s'", status.Name)
	}

	if !status.IsAlive {
		t.Error("Status should show pet is alive")
	}

	if status.Wellbeing < 0.0 || status.Wellbeing > 1.0 {
		t.Errorf("Wellbeing should be between 0 and 1, got %.2f", status.Wellbeing)
	}
}

func TestPetStatusString(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	status := pet.GetCurrentStatus()

	statusStr := status.String()

	if statusStr == "" {
		t.Error("Status string should not be empty")
	}

	if len(statusStr) < 50 {
		t.Error("Status string should be detailed")
	}
}

func TestSaveAndLoad(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)

	data, err := pet.Save()
	if err != nil {
		t.Fatalf("Failed to save pet: %v", err)
	}

	loadedPet, err := Load(data)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	if loadedPet.Name != pet.Name {
		t.Errorf("Loaded pet name mismatch: expected '%s', got '%s'", pet.Name, loadedPet.Name)
	}

	if loadedPet.TotalInteractions != pet.TotalInteractions {
		t.Errorf("Loaded pet interactions mismatch: expected %d, got %d",
			pet.TotalInteractions, loadedPet.TotalInteractions)
	}
}

func TestGetPersonalityDescription(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	description := pet.GetPersonalityDescription()

	if description == "" {
		t.Error("Personality description should not be empty")
	}
}

func TestBehaviorUpdate(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	// Make pet very tired
	pet.Biology.Vitals.Fatigue = 0.9
	pet.Update(0.1)

	if pet.CurrentBehavior != types.BehaviorSleeping {
		t.Errorf("Expected sleeping behavior when tired, got %s", pet.CurrentBehavior.String())
	}
}

func TestDeadPetNoUpdates(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	// Kill the pet
	pet.Biology.IsAlive = false
	pet.Biology.CauseOfDeath = "Test death"

	initialAge := pet.GetAge()
	pet.Update(1.0)

	if pet.GetAge() != initialAge {
		t.Error("Dead pet should not age")
	}

	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	// Should not panic or cause issues
}

// setCalmHealthyState puts the pet in a state where discretionary
// behavior is chosen (healthy, rested, fed, neutral mood)
func setCalmHealthyState(pet *DigitalPet) {
	v := pet.Biology.Vitals
	v.Health, v.Energy, v.Nutrition, v.Hydration = 1.0, 0.8, 0.8, 0.8
	v.Happiness, v.Cleanliness, v.Stress, v.Fatigue = 0.8, 0.8, 0.0, 0.0
	pet.Emotions = ai.NewEmotionState()
}

func TestBalancedPetIdlesWithoutLearning(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	setCalmHealthyState(pet)
	pet.updateBehavior()

	if pet.CurrentBehavior != types.BehaviorIdle {
		t.Errorf("Balanced pet with no history should idle, got %s", pet.CurrentBehavior)
	}
}

func TestEnjoyedPlayMakesPetPlayful(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	for i := 0; i < 10; i++ {
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
		setCalmHealthyState(pet)
	}
	pet.updateBehavior()

	if pet.CurrentBehavior != types.BehaviorPlaying {
		t.Errorf("Pet that enjoyed play should choose to play, got %s", pet.CurrentBehavior)
	}
}

func TestEnjoyedPettingMakesPetSocial(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	for i := 0; i < 10; i++ {
		pet.ProcessUserInteraction(types.InteractionPetting, 1.0)
		setCalmHealthyState(pet)
	}
	pet.updateBehavior()

	if pet.CurrentBehavior != types.BehaviorSocialInteraction {
		t.Errorf("Pet that enjoyed petting should seek interaction, got %s", pet.CurrentBehavior)
	}
}

func TestDisciplineMakesPetWithdrawn(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	for i := 0; i < 10; i++ {
		pet.ProcessUserInteraction(types.InteractionPetting, 1.0)
		setCalmHealthyState(pet)
	}
	for i := 0; i < 30; i++ {
		pet.ProcessUserInteraction(types.InteractionDiscipline, 1.0)
		setCalmHealthyState(pet)
	}
	pet.updateBehavior()

	if pet.CurrentBehavior == types.BehaviorSocialInteraction {
		t.Error("Frequent discipline should make the pet stop seeking interaction")
	}
}

func TestLearnedBehaviorPersistsWithoutInteraction(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	for i := 0; i < 10; i++ {
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
		setCalmHealthyState(pet)
	}
	before := pet.Memory.GetLearnedAffinity(types.InteractionPlaying)

	// Four weeks of real time with no interaction; the server advances one
	// time unit per real minute. Vitals are reset each step so only memory
	// and learning are under test.
	for minute := 0; minute < 28*24*60; minute++ {
		pet.Update(1)
		setCalmHealthyState(pet)
	}
	if !pet.IsAlive() {
		t.Fatal("Pet should be alive for this test to be meaningful")
	}
	pet.updateBehavior()

	if after := pet.Memory.GetLearnedAffinity(types.InteractionPlaying); after != before {
		t.Errorf("Learned preference should not change without interaction: %f -> %f", before, after)
	}
	if pet.CurrentBehavior != types.BehaviorPlaying {
		t.Errorf("Pet should still prefer play after four weeks, got %s", pet.CurrentBehavior)
	}
}

func TestLearnedPreferencesSurviveSaveLoad(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	for i := 0; i < 5; i++ {
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
	}
	want := pet.Memory.GetLearnedAffinity(types.InteractionPlaying)

	data, err := pet.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if got := loaded.Memory.GetLearnedAffinity(types.InteractionPlaying); got != want {
		t.Errorf("Learned affinity after load = %f, want %f", got, want)
	}
}
