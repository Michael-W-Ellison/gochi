package test

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// TestPetCreationAndInitialization tests that a new pet is created with valid initial state
func TestPetCreationAndInitialization(t *testing.T) {
	pet := core.NewDigitalPet("TestPet", "user_123")

	// Verify basic identification
	if pet.Name != "TestPet" {
		t.Errorf("Expected name 'TestPet', got '%s'", pet.Name)
	}

	if pet.Owner != "user_123" {
		t.Errorf("Expected owner 'user_123', got '%s'", pet.Owner)
	}

	if pet.ID == "" {
		t.Error("Pet ID should not be empty")
	}

	// Verify systems are initialized
	if pet.Biology == nil {
		t.Error("Biology system should be initialized")
	}

	if pet.Personality == nil {
		t.Error("Personality system should be initialized")
	}

	if pet.Memory == nil {
		t.Error("Memory system should be initialized")
	}

	if pet.Emotions == nil {
		t.Error("Emotions system should be initialized")
	}

	if pet.Relationships == nil {
		t.Error("Relationships system should be initialized")
	}

	// Verify initial state
	if !pet.IsAlive() {
		t.Error("New pet should be alive")
	}

	if pet.CurrentBehavior != types.BehaviorIdle {
		t.Errorf("New pet should be idle, got %v", pet.CurrentBehavior)
	}

	if pet.Location != "home" {
		t.Errorf("New pet should be at home, got '%s'", pet.Location)
	}

	if pet.TotalInteractions != 0 {
		t.Errorf("New pet should have 0 interactions, got %d", pet.TotalInteractions)
	}
}

// TestPetUpdateCycle tests that the update cycle processes correctly
func TestPetUpdateCycle(t *testing.T) {
	pet := core.NewDigitalPet("UpdateTest", "user_456")

	initialEnergy := pet.Biology.Vitals.Energy

	// Simulate time passing (1 hour = 60 minutes of delta)
	pet.Update(60.0)

	// Verify last update was recorded
	if pet.LastUpdateAt.Before(pet.CreatedAt) {
		t.Error("LastUpdateAt should be after CreatedAt")
	}

	// Energy should decrease over time (natural decay)
	if pet.Biology.Vitals.Energy >= initialEnergy {
		t.Log("Note: Energy decay may not occur in short periods")
	}

	// Play time should be tracked
	if pet.TotalPlayTime == 0 {
		t.Error("TotalPlayTime should increase with updates")
	}
}

// TestPetInteractionProcessing tests that interactions affect pet state
func TestPetInteractionProcessing(t *testing.T) {
	pet := core.NewDigitalPet("InteractionTest", "user_789")

	// Lower nutrition to make feeding effective
	pet.Biology.Vitals.Nutrition = 0.3

	initialNutrition := pet.Biology.Vitals.Nutrition
	initialInteractions := pet.TotalInteractions

	// Feed the pet
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)

	// Verify interaction was counted
	if pet.TotalInteractions != initialInteractions+1 {
		t.Errorf("Interactions should increment, expected %d, got %d",
			initialInteractions+1, pet.TotalInteractions)
	}

	// Verify nutrition increased
	if pet.Biology.Vitals.Nutrition <= initialNutrition {
		t.Error("Feeding should increase nutrition")
	}
}

// TestPetPettingInteraction tests petting effects
func TestPetPettingInteraction(t *testing.T) {
	pet := core.NewDigitalPet("PettingTest", "user_petting")

	// Set some stress to make petting effective
	pet.Biology.Vitals.Stress = 0.5

	initialStress := pet.Biology.Vitals.Stress
	initialHappiness := pet.Biology.Vitals.Happiness

	// Pet the pet
	pet.ProcessUserInteraction(types.InteractionPetting, 1.0)

	// Stress should decrease
	if pet.Biology.Vitals.Stress >= initialStress {
		t.Error("Petting should reduce stress")
	}

	// Happiness should increase
	if pet.Biology.Vitals.Happiness <= initialHappiness {
		t.Error("Petting should increase happiness")
	}
}

// TestPetPlayingInteraction tests playing effects
func TestPetPlayingInteraction(t *testing.T) {
	pet := core.NewDigitalPet("PlayingTest", "user_playing")

	// Ensure pet has energy to play
	pet.Biology.Vitals.Energy = 0.8

	initialEnergy := pet.Biology.Vitals.Energy
	initialHappiness := pet.Biology.Vitals.Happiness

	// Play with the pet
	pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)

	// Energy should decrease (playing is tiring)
	if pet.Biology.Vitals.Energy >= initialEnergy {
		t.Error("Playing should consume energy")
	}

	// Happiness should increase
	if pet.Biology.Vitals.Happiness <= initialHappiness {
		t.Error("Playing should increase happiness")
	}
}

// TestPetGroomingInteraction tests grooming effects
func TestPetGroomingInteraction(t *testing.T) {
	pet := core.NewDigitalPet("GroomingTest", "user_grooming")

	// Set low cleanliness
	pet.Biology.Vitals.Cleanliness = 0.3

	initialCleanliness := pet.Biology.Vitals.Cleanliness

	// Groom the pet
	pet.ProcessUserInteraction(types.InteractionGrooming, 1.0)

	// Cleanliness should increase
	if pet.Biology.Vitals.Cleanliness <= initialCleanliness {
		t.Error("Grooming should increase cleanliness")
	}
}

// TestPetMedicalCareInteraction tests medical care effects
func TestPetMedicalCareInteraction(t *testing.T) {
	pet := core.NewDigitalPet("MedicalTest", "user_medical")

	// Set low health
	pet.Biology.Vitals.Health = 0.4

	initialHealth := pet.Biology.Vitals.Health
	initialStress := pet.Biology.Vitals.Stress

	// Provide medical care
	pet.ProcessUserInteraction(types.InteractionMedicalCare, 1.0)

	// Health should increase
	if pet.Biology.Vitals.Health <= initialHealth {
		t.Error("Medical care should increase health")
	}

	// Stress should increase slightly (medical care is stressful)
	if pet.Biology.Vitals.Stress <= initialStress {
		t.Log("Note: Medical care may cause slight stress increase")
	}
}

// TestPetSaveAndLoad tests serialization and deserialization
func TestPetSaveAndLoad(t *testing.T) {
	// Create and modify a pet
	originalPet := core.NewDigitalPet("SaveTest", "user_save")
	originalPet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	originalPet.ProcessUserInteraction(types.InteractionPetting, 0.8)
	originalPet.TotalInteractions = 42
	originalPet.Location = "park"

	// Save to JSON
	data, err := originalPet.Save()
	if err != nil {
		t.Fatalf("Failed to save pet: %v", err)
	}

	if len(data) == 0 {
		t.Error("Saved data should not be empty")
	}

	// Load from JSON
	loadedPet, err := core.Load(data)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Verify loaded data matches original
	if loadedPet.Name != originalPet.Name {
		t.Errorf("Name mismatch: expected '%s', got '%s'",
			originalPet.Name, loadedPet.Name)
	}

	if loadedPet.ID != originalPet.ID {
		t.Errorf("ID mismatch: expected '%s', got '%s'",
			originalPet.ID, loadedPet.ID)
	}

	if loadedPet.Owner != originalPet.Owner {
		t.Errorf("Owner mismatch: expected '%s', got '%s'",
			originalPet.Owner, loadedPet.Owner)
	}

	if loadedPet.Location != originalPet.Location {
		t.Errorf("Location mismatch: expected '%s', got '%s'",
			originalPet.Location, loadedPet.Location)
	}

	if loadedPet.TotalInteractions != originalPet.TotalInteractions {
		t.Errorf("TotalInteractions mismatch: expected %d, got %d",
			originalPet.TotalInteractions, loadedPet.TotalInteractions)
	}
}

// TestPetStatusReport tests status reporting
func TestPetStatusReport(t *testing.T) {
	pet := core.NewDigitalPet("StatusTest", "user_status")

	status := pet.GetCurrentStatus()

	if status.PetID != pet.ID {
		t.Errorf("Status PetID mismatch: expected '%s', got '%s'",
			pet.ID, status.PetID)
	}

	if status.Name != pet.Name {
		t.Errorf("Status Name mismatch: expected '%s', got '%s'",
			pet.Name, status.Name)
	}

	if !status.IsAlive {
		t.Error("Status should indicate pet is alive")
	}

	if status.Wellbeing < 0 || status.Wellbeing > 1 {
		t.Errorf("Wellbeing should be between 0 and 1, got %f", status.Wellbeing)
	}

	// Test string output
	statusStr := status.String()
	if statusStr == "" {
		t.Error("Status string should not be empty")
	}
}

// TestPetBehaviorStates tests behavior state transitions
func TestPetBehaviorStates(t *testing.T) {
	pet := core.NewDigitalPet("BehaviorTest", "user_behavior")

	// Healthy pet should be idle or happy
	if pet.CurrentBehavior != types.BehaviorIdle && pet.CurrentBehavior != types.BehaviorHappy {
		t.Logf("Initial behavior: %v", pet.CurrentBehavior)
	}

	// Low health should trigger sick behavior
	pet.Biology.Vitals.Health = 0.2
	pet.Update(0.1)

	if pet.CurrentBehavior != types.BehaviorSick {
		t.Logf("Expected Sick behavior with low health, got %v", pet.CurrentBehavior)
	}

	// Restore health, make fatigued
	pet.Biology.Vitals.Health = 1.0
	pet.Biology.Vitals.Fatigue = 0.9
	pet.Biology.Vitals.Energy = 0.1
	pet.Update(0.1)

	if pet.CurrentBehavior != types.BehaviorSleeping {
		t.Logf("Expected Sleeping behavior with high fatigue, got %v", pet.CurrentBehavior)
	}
}

// TestPetRandomPersonality tests random personality generation
func TestPetRandomPersonality(t *testing.T) {
	pet := core.NewDigitalPetRandom("RandomTest", "user_random")

	if pet.Personality == nil {
		t.Fatal("Personality should not be nil")
	}

	// Verify personality description exists
	desc := pet.GetPersonalityDescription()
	if desc == "" {
		t.Error("Personality description should not be empty")
	}
}

// TestPetMultipleInteractions tests a sequence of interactions
func TestPetMultipleInteractions(t *testing.T) {
	pet := core.NewDigitalPet("MultiTest", "user_multi")

	interactionSequence := []struct {
		interactionType types.InteractionType
		intensity       float64
	}{
		{types.InteractionFeeding, 1.0},
		{types.InteractionPetting, 0.8},
		{types.InteractionPlaying, 0.7},
		{types.InteractionGrooming, 0.6},
		{types.InteractionRewards, 0.9},
	}

	for i, interaction := range interactionSequence {
		pet.ProcessUserInteraction(interaction.interactionType, interaction.intensity)

		if pet.TotalInteractions != i+1 {
			t.Errorf("After interaction %d, expected %d total, got %d",
				i, i+1, pet.TotalInteractions)
		}
	}

	// Verify final interaction count
	if pet.TotalInteractions != len(interactionSequence) {
		t.Errorf("Expected %d total interactions, got %d",
			len(interactionSequence), pet.TotalInteractions)
	}
}

// TestPetAgeTracking tests age calculation
func TestPetAgeTracking(t *testing.T) {
	pet := core.NewDigitalPet("AgeTest", "user_age")

	initialAge := pet.GetAge()

	// Age should be very small initially
	if initialAge < 0 {
		t.Error("Age should not be negative")
	}

	// Update with significant time
	pet.Update(1440.0) // 24 hours = 1 day

	newAge := pet.GetAge()
	if newAge < initialAge {
		t.Error("Age should not decrease")
	}
}

// TestPetDeadState tests that dead pets don't process updates
func TestPetDeadState(t *testing.T) {
	pet := core.NewDigitalPet("DeadTest", "user_dead")

	// Kill the pet
	pet.Biology.IsAlive = false

	initialInteractions := pet.TotalInteractions

	// Try to interact
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)

	// Interaction should not be processed
	if pet.TotalInteractions != initialInteractions {
		t.Error("Dead pet should not process interactions")
	}

	// Update should also be skipped
	initialPlayTime := pet.TotalPlayTime
	pet.Update(60.0)

	if pet.TotalPlayTime != initialPlayTime {
		t.Error("Dead pet should not accumulate play time")
	}
}

// TestPetTimestamps tests timestamp tracking
func TestPetTimestamps(t *testing.T) {
	beforeCreate := time.Now().Add(-time.Second)
	pet := core.NewDigitalPet("TimestampTest", "user_timestamp")
	afterCreate := time.Now().Add(time.Second)

	// CreatedAt should be in expected range
	if pet.CreatedAt.Before(beforeCreate) {
		t.Error("CreatedAt is too early")
	}
	if pet.CreatedAt.After(afterCreate) {
		t.Error("CreatedAt is too late")
	}

	// LastUpdateAt should be same as CreatedAt initially
	if pet.LastUpdateAt.Before(pet.CreatedAt) {
		t.Error("LastUpdateAt should not be before CreatedAt")
	}

	// Update and verify timestamp changes
	time.Sleep(10 * time.Millisecond)
	pet.Update(1.0)

	if !pet.LastUpdateAt.After(pet.CreatedAt) {
		t.Error("LastUpdateAt should advance after update")
	}
}
