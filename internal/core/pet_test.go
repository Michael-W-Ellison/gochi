package core

import (
	"testing"

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

func TestNewDigitalPetRandom(t *testing.T) {
	pet := NewDigitalPetRandom("RandomPet", "user456")

	if pet.Name != "RandomPet" {
		t.Errorf("Expected name 'RandomPet', got '%s'", pet.Name)
	}

	if pet.Owner != "user456" {
		t.Errorf("Expected owner 'user456', got '%s'", pet.Owner)
	}

	if pet.Biology == nil {
		t.Error("Biology should be initialized")
	}

	if pet.Personality == nil {
		t.Error("Personality should be initialized")
	}

	if !pet.IsAlive() {
		t.Error("New random pet should be alive")
	}

	// Random pet should have random traits
	if pet.Personality.Traits.Playfulness == 0.5 && pet.Personality.Traits.Affectionate == 0.5 {
		t.Error("Random pet should have randomized personality traits")
	}
}

func TestGetName(t *testing.T) {
	pet := NewDigitalPet("Fluffy", "user123")

	if pet.GetName() != "Fluffy" {
		t.Errorf("Expected name 'Fluffy', got '%s'", pet.GetName())
	}
}

func TestApplyInteractionEffects(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	// Test feeding interaction when nutrition is low
	pet.Biology.Vitals.Nutrition = 0.5
	initialNutrition := pet.Biology.Vitals.Nutrition
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	if pet.Biology.Vitals.Nutrition <= initialNutrition {
		t.Error("Feeding should increase nutrition when below max")
	}

	// Test playing interaction
	initialHappiness := pet.Biology.Vitals.Happiness
	pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
	if pet.Biology.Vitals.Happiness <= initialHappiness {
		t.Error("Playing should increase happiness")
	}

	// Test training interaction
	pet.ProcessUserInteraction(types.InteractionTraining, 1.0)
	// Training should affect the pet (no specific assertion needed)

	// Test grooming interaction
	pet.Biology.Vitals.Cleanliness = 0.5
	pet.ProcessUserInteraction(types.InteractionGrooming, 1.0)
	if pet.Biology.Vitals.Cleanliness <= 0.5 {
		t.Error("Grooming should increase cleanliness")
	}

	// Test medical care
	pet.Biology.Vitals.Health = 0.5
	initialHealth := pet.Biology.Vitals.Health
	pet.ProcessUserInteraction(types.InteractionMedicalCare, 1.0)
	if pet.Biology.Vitals.Health <= initialHealth {
		t.Error("Medical care should increase health")
	}
}

func TestUpdateBehaviorStates(t *testing.T) {
	// Test sick behavior when health is low (highest priority)
	pet := NewDigitalPet("TestPet", "user123")
	pet.Biology.Vitals.Health = 0.15
	pet.Update(0.1)
	if pet.CurrentBehavior != types.BehaviorSick {
		t.Errorf("Expected sick when health is low, got %s", pet.CurrentBehavior.String())
	}

	// Test sleeping behavior when very tired
	pet2 := NewDigitalPet("TestPet2", "user123")
	pet2.Biology.Vitals.Fatigue = 0.95
	pet2.Update(0.1)
	if pet2.CurrentBehavior != types.BehaviorSleeping {
		t.Errorf("Expected sleeping when tired, got %s", pet2.CurrentBehavior.String())
	}

	// Test eating behavior when hungry (low nutrition)
	pet3 := NewDigitalPet("TestPet3", "user123")
	pet3.Biology.Vitals.Nutrition = 0.2
	pet3.Biology.Vitals.Energy = 0.5
	pet3.Biology.Vitals.Fatigue = 0.0
	pet3.Update(0.1)
	if pet3.CurrentBehavior != types.BehaviorEating {
		t.Errorf("Expected eating when hungry, got %s", pet3.CurrentBehavior.String())
	}

	// Test that behavior updates (not stuck on initial state)
	pet4 := NewDigitalPet("TestPet4", "user123")
	initialBehavior := pet4.CurrentBehavior
	pet4.Biology.Vitals.Fatigue = 0.9
	pet4.Update(0.1)
	if pet4.CurrentBehavior == initialBehavior && pet4.CurrentBehavior != types.BehaviorSleeping {
		t.Error("Behavior should change when conditions change")
	}
}

func TestAllInteractionTypes(t *testing.T) {
	interactionTypes := []types.InteractionType{
		types.InteractionFeeding,
		types.InteractionPetting,
		types.InteractionPlaying,
		types.InteractionTraining,
		types.InteractionGrooming,
		types.InteractionMedicalCare,
		types.InteractionEnvironmentalEnrichment,
		types.InteractionSocialIntroduction,
		types.InteractionDiscipline,
		types.InteractionRewards,
	}

	for _, interactionType := range interactionTypes {
		pet := NewDigitalPet("TestPet", "user123")
		initialInteractions := pet.TotalInteractions

		pet.ProcessUserInteraction(interactionType, 1.0)

		if pet.TotalInteractions != initialInteractions+1 {
			t.Errorf("Interaction %s should increment total interactions", interactionType.String())
		}
	}
}
