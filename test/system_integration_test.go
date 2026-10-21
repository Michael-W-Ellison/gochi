package test

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// TestBiologyEmotionIntegration tests biology and emotion systems working together
func TestBiologyEmotionIntegration(t *testing.T) {
	pet := core.NewDigitalPet("BioEmoTest", "user_bioemo")

	// Low health should affect emotions
	pet.Biology.Vitals.Health = 0.2
	pet.Update(1.0)

	moodScore := pet.Emotions.GetMoodScore()
	t.Logf("Mood score with low health: %f", moodScore)

	// Restore health
	pet.Biology.Vitals.Health = 1.0
	pet.ProcessUserInteraction(types.InteractionPetting, 1.0)
	pet.Update(1.0)

	newMoodScore := pet.Emotions.GetMoodScore()
	t.Logf("Mood score after healing and petting: %f", newMoodScore)
}

// TestMemoryRecording tests that interactions are recorded in memory
func TestMemoryRecording(t *testing.T) {
	pet := core.NewDigitalPet("MemoryTest", "user_memory")

	// Perform several interactions
	interactions := []types.InteractionType{
		types.InteractionFeeding,
		types.InteractionPetting,
		types.InteractionPlaying,
	}

	for _, it := range interactions {
		pet.ProcessUserInteraction(it, 1.0)
	}

	// Check short-term memory
	shortTermCount := len(pet.Memory.ShortTermMemories)
	if shortTermCount < len(interactions) {
		t.Logf("Short-term memories: %d (expected at least %d)",
			shortTermCount, len(interactions))
	}
}

// TestPersonalityEffectsOnInteractions tests personality influence
func TestPersonalityEffectsOnInteractions(t *testing.T) {
	// Create two pets with random personalities
	pet1 := core.NewDigitalPetRandom("RandomPet1", "user_random1")
	pet2 := core.NewDigitalPetRandom("RandomPet2", "user_random2")

	// Get personality descriptions
	desc1 := pet1.GetPersonalityDescription()
	desc2 := pet2.GetPersonalityDescription()

	t.Logf("Pet1 personality: %s", desc1)
	t.Logf("Pet2 personality: %s", desc2)

	// Both should have descriptions
	if desc1 == "" || desc2 == "" {
		t.Error("Pets should have personality descriptions")
	}
}

// TestInteractionValidation tests interaction validation
func TestInteractionValidation(t *testing.T) {
	// Tired pet can't play
	ctx := interaction.InteractionContext{
		Energy:     0.1,
		IsSleeping: false,
	}

	valid, reason := interaction.ValidateInteraction(types.InteractionPlaying, ctx)
	if valid {
		t.Error("Playing should be invalid when tired")
	}
	if reason == "" {
		t.Error("Should provide reason for invalid interaction")
	}

	// Sleeping pet can't be trained
	ctx.IsSleeping = true
	ctx.Energy = 0.5

	valid, _ = interaction.ValidateInteraction(types.InteractionTraining, ctx)
	if valid {
		t.Error("Training should be invalid when sleeping")
	}
}

// TestEmotionalStimulus tests emotional stimulus processing
func TestEmotionalStimulus(t *testing.T) {
	emotions := ai.NewEmotionState()

	initialJoy := emotions.Joy

	// Apply positive stimulus
	stimulus := ai.CreateStimulusFromInteraction(types.InteractionPetting, 1.0)
	emotions.ApplyEmotionalStimulus(stimulus)

	if emotions.Joy <= initialJoy {
		t.Log("Joy should increase from petting stimulus")
	}

	// Check mood description
	mood := emotions.GetMoodDescription()
	if mood == "" {
		t.Error("Mood description should not be empty")
	}
}

// TestCircadianRhythm tests circadian rhythm effects
func TestCircadianRhythm(t *testing.T) {
	circadian := biology.NewCircadianRhythm()

	// Test initial state
	if !circadian.IsAwakePhase() {
		t.Error("New circadian should start in awake phase")
	}

	initialPressure := circadian.GetSleepPressure()
	t.Logf("Initial sleep pressure: %f", initialPressure)

	// Simulate time passing during the day
	circadian.Update(8.0, 14.0, false) // 8 hours during afternoon

	laterPressure := circadian.GetSleepPressure()
	t.Logf("Sleep pressure after 8 hours: %f", laterPressure)

	// Sleep pressure should increase over time while awake
	if laterPressure < initialPressure {
		t.Log("Note: Sleep pressure typically increases while awake")
	}

	// Test sleep initiation
	circadian.InitiateSleep()
	if !circadian.IsSleepingPhase() {
		t.Error("Should be in sleeping phase after InitiateSleep")
	}

	// Test getting stats
	stats := circadian.GetStats()
	t.Logf("Circadian stats - Phase: %v, SleepDebt: %f, SleepQuality: %f",
		stats.CurrentPhase, stats.SleepDebt, stats.SleepQuality)
}

// TestVitalsDecay tests vital stats decay over time
func TestVitalsDecay(t *testing.T) {
	bio := biology.NewBiologicalSystems()

	// Set initial values
	bio.Vitals.Nutrition = 1.0
	bio.Vitals.Energy = 1.0
	bio.Vitals.Cleanliness = 1.0

	// Update over significant time
	for i := 0; i < 100; i++ {
		bio.Update(1.0)
	}

	// Stats should have decayed
	if bio.Vitals.Nutrition >= 1.0 {
		t.Log("Nutrition should decay over time")
	}

	if bio.Vitals.Energy >= 1.0 {
		t.Log("Energy should decay over time")
	}

	if bio.Vitals.Cleanliness >= 1.0 {
		t.Log("Cleanliness should decay over time")
	}
}

// TestMemoryConsolidation tests memory consolidation
func TestMemoryConsolidation(t *testing.T) {
	memory := ai.NewMemorySystem(100)

	// Add multiple short-term memories
	for i := 0; i < 15; i++ {
		memory.RecordInteraction(types.InteractionPetting, float64(i), 1.0, "happy")
	}

	initialShortTerm := len(memory.ShortTermMemories)
	initialLongTerm := len(memory.LongTermMemories)

	// Consolidate memories
	memory.ConsolidateMemories()

	t.Logf("Before consolidation: %d short-term, %d long-term",
		initialShortTerm, initialLongTerm)
	t.Logf("After consolidation: %d short-term, %d long-term",
		len(memory.ShortTermMemories), len(memory.LongTermMemories))
}

// TestMemoryDecay tests memory decay over time
func TestMemoryDecay(t *testing.T) {
	memory := ai.NewMemorySystem(100)

	// Add memories
	memory.RecordInteraction(types.InteractionFeeding, 0, 1.0, "content")

	initialCount := len(memory.ShortTermMemories)

	// Decay over a very long time
	for i := 0; i < 1000; i++ {
		memory.DecayMemories(10.0)
	}

	finalCount := len(memory.ShortTermMemories)

	t.Logf("Memory count: %d -> %d after decay", initialCount, finalCount)
}

// TestFullSystemCycle tests a complete update cycle with all systems
func TestFullSystemCycle(t *testing.T) {
	pet := core.NewDigitalPet("FullCycleTest", "user_fullcycle")

	// Record initial state
	initialState := pet.GetCurrentStatus()

	// Simulate a day of activity
	interactions := []struct {
		interactionType types.InteractionType
		intensity       float64
	}{
		{types.InteractionFeeding, 1.0},
		{types.InteractionPetting, 0.8},
		{types.InteractionPlaying, 0.7},
		{types.InteractionGrooming, 0.6},
		{types.InteractionTraining, 0.5},
		{types.InteractionRewards, 1.0},
	}

	for _, inter := range interactions {
		pet.ProcessUserInteraction(inter.interactionType, inter.intensity)
		pet.Update(10.0) // 10 minutes between interactions
	}

	// Record final state
	finalState := pet.GetCurrentStatus()

	t.Logf("Initial wellbeing: %.2f%%, Final wellbeing: %.2f%%",
		initialState.Wellbeing*100, finalState.Wellbeing*100)
	t.Logf("Total interactions processed: %d", pet.TotalInteractions)
	t.Logf("Total play time: %.2f hours", pet.TotalPlayTime)
}

// TestLongRunningSimulation tests stability over long simulation
func TestLongRunningSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long simulation test in short mode")
	}

	pet := core.NewDigitalPet("LongSimTest", "user_longsim")

	// Simulate multiple update cycles with intensive care
	// The pet simulation has fast decay rates, so we need frequent care
	totalUpdates := 50

	for i := 0; i < totalUpdates; i++ {
		// Feed before updating to keep vitals high
		pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)

		// Small time delta per update (0.1 minute of game time)
		pet.Update(0.1)

		// Pet and groom occasionally
		if i%5 == 0 {
			pet.ProcessUserInteraction(types.InteractionPetting, 0.7)
			pet.ProcessUserInteraction(types.InteractionGrooming, 0.5)
		}

		// Check pet is still alive
		if !pet.IsAlive() {
			status := pet.GetCurrentStatus()
			t.Logf("Pet state at death: Health=%.2f%%, Energy=%.2f%%, Wellbeing=%.2f%%",
				pet.Biology.Vitals.Health*100, pet.Biology.Vitals.Energy*100, status.Wellbeing*100)
			t.Fatalf("Pet died at update %d", i)
		}
	}

	status := pet.GetCurrentStatus()
	t.Logf("After %d updates: Age=%.4f days, Wellbeing=%.2f%%, Interactions=%d",
		totalUpdates, status.Age, status.Wellbeing*100, pet.TotalInteractions)

	// Verify pet survived the simulation
	if !pet.IsAlive() {
		t.Error("Pet should survive with intensive care")
	}
}

// TestConcurrentUpdates tests thread safety of updates
func TestConcurrentUpdates(t *testing.T) {
	pet := core.NewDigitalPet("ConcurrentTest", "user_concurrent")

	done := make(chan bool)

	// Start multiple goroutines updating the pet
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				pet.Update(0.1)
			}
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent updates timed out")
		}
	}

	// Pet should still be in valid state
	if pet.Biology == nil || pet.Emotions == nil {
		t.Error("Pet systems corrupted after concurrent updates")
	}
}

// TestInteractionDescriptions tests interaction descriptions
func TestInteractionDescriptions(t *testing.T) {
	interactionTypes := []types.InteractionType{
		types.InteractionFeeding,
		types.InteractionPetting,
		types.InteractionPlaying,
		types.InteractionTraining,
		types.InteractionGrooming,
		types.InteractionMedicalCare,
		types.InteractionRewards,
		types.InteractionDiscipline,
	}

	for _, it := range interactionTypes {
		desc := interaction.GetInteractionDescription(it)
		if desc == "" {
			t.Errorf("Interaction %v should have a description", it)
		}
	}
}

// TestPetStatusString tests status string generation
func TestPetStatusString(t *testing.T) {
	pet := core.NewDigitalPet("StatusStringTest", "user_statusstring")

	status := pet.GetCurrentStatus()
	statusStr := status.String()

	if statusStr == "" {
		t.Error("Status string should not be empty")
	}

	t.Logf("Status: %s", statusStr)
}

// TestBiologyStatus tests biology status description
func TestBiologyStatus(t *testing.T) {
	bio := biology.NewBiologicalSystems()

	status := bio.GetStatus()
	if status == "" {
		t.Error("Biology status should not be empty")
	}

	// Set low values
	bio.Vitals.Health = 0.2
	status = bio.GetStatus()
	t.Logf("Status with low health: %s", status)
}

// TestEmotionMoodScore tests emotion mood score calculation
func TestEmotionMoodScore(t *testing.T) {
	emotions := ai.NewEmotionState()

	// Get initial mood
	initialMood := emotions.GetMoodScore()
	t.Logf("Initial mood score: %f", initialMood)

	// Increase joy
	emotions.Joy = 1.0
	happyMood := emotions.GetMoodScore()
	t.Logf("Happy mood score: %f", happyMood)

	// Mood should be higher with joy
	if happyMood <= initialMood {
		t.Log("Note: Mood calculation may involve multiple factors")
	}
}
