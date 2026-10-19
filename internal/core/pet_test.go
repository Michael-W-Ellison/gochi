package core

import (
	"errors"
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

func TestNeglectedPetHibernatesInsteadOfDying(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.HibernationEnabled = true

	// Many small updates with no care
	for i := 0; i < 1000; i++ {
		pet.Update(1)
	}

	if !pet.IsAlive() {
		t.Fatalf("Neglected pet should hibernate, not die (cause: %s)", pet.Biology.CauseOfDeath)
	}
	if !pet.IsHibernating() || pet.CurrentBehavior != types.BehaviorHibernating {
		t.Errorf("Neglected pet should be hibernating, behavior %s", pet.CurrentBehavior)
	}
	if v := pet.Biology.Vitals; v.Hydration <= 0 || v.Health <= 0 {
		t.Errorf("Hibernation should start before a stat reaches zero: %+v", *v)
	}
}

func TestLargeUpdateCannotSkipHibernation(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.HibernationEnabled = true

	pet.Update(10000)

	if !pet.IsAlive() || !pet.IsHibernating() {
		t.Errorf("A single long update should end in hibernation, alive=%v hibernating=%v", pet.IsAlive(), pet.IsHibernating())
	}
}

func TestHibernationFreezesStats(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.HibernationEnabled = true
	pet.Update(10000)
	if !pet.IsHibernating() {
		t.Fatal("Pet should be hibernating")
	}

	vitals := *pet.Biology.Vitals
	age := pet.GetAge()
	emotions := *pet.Emotions

	pet.Update(500)

	if *pet.Biology.Vitals != vitals || pet.GetAge() != age {
		t.Error("Vitals and age should not change while hibernating")
	}
	if pet.Emotions.Joy != emotions.Joy || pet.Emotions.Sadness != emotions.Sadness {
		t.Error("Emotions should not change while hibernating")
	}
}

func TestHibernatingPetIgnoresInteractions(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.Update(10000)
	interactions := pet.TotalInteractions
	nutrition := pet.Biology.Vitals.Nutrition

	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)

	if pet.TotalInteractions != interactions || pet.Biology.Vitals.Nutrition != nutrition {
		t.Error("A hibernating pet should not respond to interactions")
	}
}

func TestReviveRestoresPet(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.HibernationEnabled = true

	if err := pet.Revive(); !errors.Is(err, ErrNotHibernating) {
		t.Errorf("Reviving an awake pet should return ErrNotHibernating, got %v", err)
	}

	pet.Update(10000)
	if err := pet.Revive(); err != nil {
		t.Fatalf("Revive failed: %v", err)
	}

	if pet.IsHibernating() || pet.CurrentBehavior == types.BehaviorHibernating {
		t.Error("Revived pet should be awake")
	}
	v := pet.Biology.Vitals
	if v.Health < reviveLevel || v.Hydration < reviveLevel || v.Nutrition < reviveLevel || v.Energy < reviveLevel {
		t.Errorf("Critical stats should be restored to at least %.1f: %+v", reviveLevel, *v)
	}

	// The pet can be cared for and lives on
	pet.ProcessUserInteraction(types.InteractionFeeding, 1.0)
	if pet.TotalInteractions != 1 {
		t.Error("Revived pet should accept interactions")
	}
	pet.Update(1)
	if pet.IsHibernating() {
		t.Error("Revived pet should not immediately hibernate again")
	}
}

func TestInteractionLeavingStatCriticalHibernates(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.HibernationEnabled = true
	pet.Biology.Vitals.Nutrition = 0.05
	pet.Biology.Vitals.Energy = 0.2 // Just enough to agree to play

	// Play uses energy, leaving food and energy both critical
	pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)

	if !pet.IsHibernating() || !pet.IsAlive() {
		t.Errorf("Pet should hibernate after play leaves it exhausted and hungry, hibernating=%v", pet.IsHibernating())
	}
}

func TestHibernationSurvivesSaveLoad(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.HibernationEnabled = true
	pet.Update(10000)

	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}

	if !loaded.IsHibernating() {
		t.Error("Hibernation should be restored from a save")
	}
}

func TestNeglectedPetDiesByDefault(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	if pet.HibernationEnabled {
		t.Fatal("Hibernation should be off by default")
	}

	for i := 0; i < 1000 && pet.IsAlive(); i++ {
		pet.Update(1)
	}

	if pet.IsAlive() || pet.IsHibernating() {
		t.Errorf("A neglected pet without hibernation should die, alive=%v hibernating=%v", pet.IsAlive(), pet.IsHibernating())
	}
	if pet.Biology.CauseOfDeath == "" {
		t.Error("Cause of death should be recorded")
	}
	if pet.TotalPlayTime == 0 {
		t.Error("The time the pet lived should be recorded")
	}
}

func TestTurningHibernationOffKeepsHibernatingPetAsleep(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.SetHibernationEnabled(true)
	pet.Update(10000)
	pet.SetHibernationEnabled(false)

	pet.Update(100)
	if !pet.IsAlive() || !pet.IsHibernating() {
		t.Error("Turning hibernation off should not wake or kill a hibernating pet")
	}
	if err := pet.Revive(); err != nil {
		t.Errorf("A hibernating pet can still be revived: %v", err)
	}
}
