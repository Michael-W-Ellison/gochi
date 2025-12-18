package interaction

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Helper function to create a default test context
func defaultTestContext() InteractionContext {
	return InteractionContext{
		Health:          0.8,
		Energy:          0.7,
		Happiness:       0.6,
		Hunger:          0.4,
		Thirst:          0.3,
		Cleanliness:     0.7,
		Stress:          0.2,
		Fatigue:         0.3,
		Playfulness:     0.5,
		Independence:    0.5,
		Affectionate:    0.5,
		Intelligence:    0.5,
		CurrentBehavior: types.BehaviorIdle,
		PetAge:          30.0,
		IsSleeping:      false,
		IsSick:          false,
		TimeOfDay:       12.0,
		IsOutdoors:      false,
	}
}

// TestNewInteractionProcessor tests processor creation
func TestNewInteractionProcessor(t *testing.T) {
	proc := NewInteractionProcessor()

	if proc == nil {
		t.Fatal("NewInteractionProcessor returned nil")
	}

	if proc.TotalInteractions != 0 {
		t.Errorf("Expected 0 total interactions, got %d", proc.TotalInteractions)
	}

	if proc.InteractionCounts == nil {
		t.Error("InteractionCounts map is nil")
	}

	if proc.CooldownTimers == nil {
		t.Error("CooldownTimers map is nil")
	}
}

// TestNewInteractionProcessorWithConfig tests custom config
func TestNewInteractionProcessorWithConfig(t *testing.T) {
	config := ProcessorConfig{
		BaseCooldownSeconds: 5.0,
		EffectMultiplier:    2.0,
		CooldownOverrides:   make(map[types.InteractionType]float64),
	}

	proc := NewInteractionProcessorWithConfig(config)

	if proc.Config.BaseCooldownSeconds != 5.0 {
		t.Errorf("Expected base cooldown 5.0, got %f", proc.Config.BaseCooldownSeconds)
	}

	if proc.Config.EffectMultiplier != 2.0 {
		t.Errorf("Expected effect multiplier 2.0, got %f", proc.Config.EffectMultiplier)
	}
}

// TestDefaultProcessorConfig tests default config values
func TestDefaultProcessorConfig(t *testing.T) {
	config := DefaultProcessorConfig()

	if config.BaseCooldownSeconds != 1.0 {
		t.Errorf("Expected base cooldown 1.0, got %f", config.BaseCooldownSeconds)
	}

	if config.EffectMultiplier != 1.0 {
		t.Errorf("Expected effect multiplier 1.0, got %f", config.EffectMultiplier)
	}

	// Check cooldown overrides exist
	if _, exists := config.CooldownOverrides[types.InteractionFeeding]; !exists {
		t.Error("Expected feeding cooldown override")
	}
}

// TestProcessFeeding tests feeding interaction
func TestProcessFeeding(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Hunger = 0.3 // Pet is hungry

	result := proc.Process(types.InteractionFeeding, 1.0, ctx)

	if !result.Success {
		t.Error("Feeding should succeed")
	}

	if result.Effects["hunger"] >= 0 {
		t.Error("Feeding should reduce hunger")
	}

	if result.Effects["energy"] <= 0 {
		t.Error("Feeding should provide energy")
	}

	if result.Feedback.Animation != "eating" {
		t.Errorf("Expected eating animation, got %s", result.Feedback.Animation)
	}
}

// TestProcessFeedingNotHungry tests feeding when pet is full
func TestProcessFeedingNotHungry(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Hunger = 0.9 // Pet is not hungry

	result := proc.Process(types.InteractionFeeding, 1.0, ctx)

	// Still succeeds but with reduced effects
	if !result.Success {
		t.Error("Feeding should still succeed")
	}

	// Message should indicate pet isn't hungry
	if result.Message != "Your pet isn't very hungry right now" {
		t.Errorf("Expected not hungry message, got: %s", result.Message)
	}
}

// TestProcessPetting tests petting interaction
func TestProcessPetting(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	result := proc.Process(types.InteractionPetting, 1.0, ctx)

	if !result.Success {
		t.Error("Petting should succeed")
	}

	if result.Effects["happiness"] <= 0 {
		t.Error("Petting should increase happiness")
	}

	if result.Effects["stress"] >= 0 {
		t.Error("Petting should reduce stress")
	}
}

// TestProcessPettingAffectionate tests petting with affectionate pet
func TestProcessPettingAffectionate(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Affectionate = 0.8

	result := proc.Process(types.InteractionPetting, 1.0, ctx)

	// Affectionate pets should get more happiness from petting
	ctxNormal := defaultTestContext()
	ctxNormal.Affectionate = 0.5
	proc.ResetCooldowns()
	resultNormal := proc.Process(types.InteractionPetting, 1.0, ctxNormal)

	if result.Effects["happiness"] <= resultNormal.Effects["happiness"] {
		t.Error("Affectionate pet should get more happiness from petting")
	}
}

// TestProcessPlaying tests playing interaction
func TestProcessPlaying(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Energy = 0.8 // Good energy

	result := proc.Process(types.InteractionPlaying, 1.0, ctx)

	if !result.Success {
		t.Error("Playing should succeed")
	}

	if result.Effects["happiness"] <= 0 {
		t.Error("Playing should increase happiness")
	}

	if result.Effects["energy"] >= 0 {
		t.Error("Playing should drain energy")
	}

	if result.SkillGain["agility"] <= 0 {
		t.Error("Playing should increase agility skill")
	}
}

// TestProcessPlayingTooTired tests playing when pet is tired
func TestProcessPlayingTooTired(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Energy = 0.1 // Too tired

	result := proc.Process(types.InteractionPlaying, 1.0, ctx)

	if result.Success {
		t.Error("Playing should fail when pet is too tired")
	}

	if result.Message != "Your pet is too tired to play" {
		t.Errorf("Expected tired message, got: %s", result.Message)
	}
}

// TestProcessTraining tests training interaction
func TestProcessTraining(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Energy = 0.6
	ctx.Stress = 0.2

	result := proc.Process(types.InteractionTraining, 1.0, ctx)

	if !result.Success {
		t.Error("Training should succeed")
	}

	if result.SkillGain["obedience"] <= 0 {
		t.Error("Training should increase obedience")
	}

	if result.SkillGain["intelligence"] <= 0 {
		t.Error("Training should increase intelligence")
	}
}

// TestProcessTrainingTooTired tests training when tired
func TestProcessTrainingTooTired(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Energy = 0.2 // Too tired

	result := proc.Process(types.InteractionTraining, 1.0, ctx)

	if result.Success {
		t.Error("Training should fail when pet is too tired")
	}
}

// TestProcessTrainingTooStressed tests training when stressed
func TestProcessTrainingTooStressed(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Stress = 0.8 // Too stressed

	result := proc.Process(types.InteractionTraining, 1.0, ctx)

	if result.Success {
		t.Error("Training should fail when pet is too stressed")
	}
}

// TestProcessGrooming tests grooming interaction
func TestProcessGrooming(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.Cleanliness = 0.3 // Dirty

	result := proc.Process(types.InteractionGrooming, 1.0, ctx)

	if !result.Success {
		t.Error("Grooming should succeed")
	}

	if result.Effects["cleanliness"] <= 0 {
		t.Error("Grooming should increase cleanliness")
	}

	if result.Feedback.Animation != "grooming" {
		t.Errorf("Expected grooming animation, got %s", result.Feedback.Animation)
	}
}

// TestProcessMedicalCare tests medical care interaction
func TestProcessMedicalCare(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	result := proc.Process(types.InteractionMedicalCare, 1.0, ctx)

	if !result.Success {
		t.Error("Medical care should succeed")
	}

	if result.Effects["health"] <= 0 {
		t.Error("Medical care should improve health")
	}

	// Medical care is stressful
	if result.Effects["stress"] <= 0 {
		t.Error("Medical care should cause stress")
	}
}

// TestProcessMedicalCareSick tests medical care on sick pet
func TestProcessMedicalCareSick(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.IsSick = true
	ctx.Health = 0.3

	result := proc.Process(types.InteractionMedicalCare, 1.0, ctx)

	// Sick pets should benefit more
	procNormal := NewInteractionProcessor()
	ctxNormal := defaultTestContext()
	ctxNormal.Health = 0.8
	resultNormal := procNormal.Process(types.InteractionMedicalCare, 1.0, ctxNormal)

	if result.Effects["health"] <= resultNormal.Effects["health"] {
		t.Error("Sick pet should benefit more from medical care")
	}
}

// TestProcessRewards tests reward interaction
func TestProcessRewards(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	result := proc.Process(types.InteractionRewards, 1.0, ctx)

	if !result.Success {
		t.Error("Rewards should succeed")
	}

	if result.Effects["happiness"] <= 0 {
		t.Error("Rewards should increase happiness")
	}

	if result.Effects["loyalty_bond"] <= 0 {
		t.Error("Rewards should increase loyalty bond")
	}
}

// TestProcessDiscipline tests discipline interaction
func TestProcessDiscipline(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	result := proc.Process(types.InteractionDiscipline, 1.0, ctx)

	if !result.Success {
		t.Error("Discipline should succeed")
	}

	if result.Effects["happiness"] >= 0 {
		t.Error("Discipline should reduce happiness")
	}

	if result.Effects["stress"] <= 0 {
		t.Error("Discipline should increase stress")
	}

	if result.SkillGain["obedience"] <= 0 {
		t.Error("Discipline should improve obedience")
	}
}

// TestProcessEnrichment tests environmental enrichment
func TestProcessEnrichment(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	result := proc.Process(types.InteractionEnvironmentalEnrichment, 1.0, ctx)

	if !result.Success {
		t.Error("Enrichment should succeed")
	}

	if result.Effects["mental_stimulation"] <= 0 {
		t.Error("Enrichment should provide mental stimulation")
	}
}

// TestProcessSocialIntroduction tests social introduction
func TestProcessSocialIntroduction(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	result := proc.Process(types.InteractionSocialIntroduction, 1.0, ctx)

	if !result.Success {
		t.Error("Social introduction should succeed")
	}

	if result.Effects["social_satisfaction"] <= 0 {
		t.Error("Social introduction should provide social satisfaction")
	}
}

// TestSleepingPreventsInteraction tests that sleeping prevents most interactions
func TestSleepingPreventsInteraction(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()
	ctx.IsSleeping = true

	// Playing should fail
	result := proc.Process(types.InteractionPlaying, 1.0, ctx)
	if result.Success {
		t.Error("Playing should fail when pet is sleeping")
	}

	// But medical care should work
	proc.ResetCooldowns()
	result = proc.Process(types.InteractionMedicalCare, 1.0, ctx)
	if !result.Success {
		t.Error("Medical care should work even when pet is sleeping")
	}
}

// TestCooldowns tests the cooldown system
func TestCooldowns(t *testing.T) {
	config := ProcessorConfig{
		BaseCooldownSeconds: 0.1, // Short cooldown for testing
		EffectMultiplier:    1.0,
		CooldownOverrides:   make(map[types.InteractionType]float64),
	}
	proc := NewInteractionProcessorWithConfig(config)
	ctx := defaultTestContext()

	// First interaction should succeed
	result := proc.Process(types.InteractionPetting, 1.0, ctx)
	if !result.Success {
		t.Error("First petting should succeed")
	}

	// Immediate second should fail
	result = proc.Process(types.InteractionPetting, 1.0, ctx)
	if result.Success {
		t.Error("Immediate second petting should fail due to cooldown")
	}

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)

	// Should succeed again
	result = proc.Process(types.InteractionPetting, 1.0, ctx)
	if !result.Success {
		t.Error("Petting should succeed after cooldown")
	}
}

// TestResetCooldowns tests cooldown reset
func TestResetCooldowns(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	proc.Process(types.InteractionPetting, 1.0, ctx)

	// Should fail due to cooldown
	result := proc.Process(types.InteractionPetting, 1.0, ctx)
	if result.Success {
		t.Error("Should fail due to cooldown")
	}

	// Reset cooldowns
	proc.ResetCooldowns()

	// Should succeed now
	result = proc.Process(types.InteractionPetting, 1.0, ctx)
	if !result.Success {
		t.Error("Should succeed after cooldown reset")
	}
}

// TestGetCooldownRemaining tests cooldown remaining calculation
func TestGetCooldownRemaining(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	// No cooldown initially
	remaining := proc.GetCooldownRemaining(types.InteractionPetting)
	if remaining != 0 {
		t.Errorf("Expected 0 remaining cooldown initially, got %f", remaining)
	}

	// Process an interaction
	proc.Process(types.InteractionPetting, 1.0, ctx)

	// Should have some cooldown remaining
	remaining = proc.GetCooldownRemaining(types.InteractionPetting)
	if remaining <= 0 {
		t.Error("Expected positive cooldown remaining after interaction")
	}
}

// TestSetCooldownOverride tests custom cooldown setting
func TestSetCooldownOverride(t *testing.T) {
	proc := NewInteractionProcessor()

	proc.SetCooldownOverride(types.InteractionPetting, 100.0)

	if proc.Config.CooldownOverrides[types.InteractionPetting] != 100.0 {
		t.Error("Cooldown override not set correctly")
	}
}

// TestGetAvailableInteractions tests getting available interactions
func TestGetAvailableInteractions(t *testing.T) {
	proc := NewInteractionProcessor()

	available := proc.GetAvailableInteractions()

	if len(available) == 0 {
		t.Error("Expected some available interactions")
	}

	// All should be available initially
	if len(available) != 10 {
		t.Errorf("Expected 10 available interactions initially, got %d", len(available))
	}
}

// TestGetInteractionStats tests statistics retrieval
func TestGetInteractionStats(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	proc.Process(types.InteractionPetting, 1.0, ctx)
	proc.ResetCooldowns()
	proc.Process(types.InteractionFeeding, 1.0, ctx)

	stats := proc.GetInteractionStats()

	total, ok := stats["total_interactions"].(int)
	if !ok || total != 2 {
		t.Errorf("Expected 2 total interactions, got %v", stats["total_interactions"])
	}
}

// TestInteractionStatistics tests that interactions update statistics
func TestInteractionStatistics(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	if proc.TotalInteractions != 0 {
		t.Errorf("Expected 0 initial interactions, got %d", proc.TotalInteractions)
	}

	proc.Process(types.InteractionPetting, 1.0, ctx)

	if proc.TotalInteractions != 1 {
		t.Errorf("Expected 1 interaction after petting, got %d", proc.TotalInteractions)
	}

	if proc.InteractionCounts[types.InteractionPetting] != 1 {
		t.Errorf("Expected 1 petting count, got %d", proc.InteractionCounts[types.InteractionPetting])
	}
}

// TestGetInteractionDescription tests description retrieval
func TestGetInteractionDescription(t *testing.T) {
	desc := GetInteractionDescription(types.InteractionFeeding)
	if desc == "" {
		t.Error("Expected non-empty description for feeding")
	}

	desc = GetInteractionDescription(types.InteractionType(999))
	if desc != "Interact with your pet" {
		t.Errorf("Expected default description for unknown type, got: %s", desc)
	}
}

// TestValidateInteraction tests interaction validation
func TestValidateInteraction(t *testing.T) {
	tests := []struct {
		name        string
		interaction types.InteractionType
		ctx         InteractionContext
		wantValid   bool
	}{
		{
			name:        "playing with energy",
			interaction: types.InteractionPlaying,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Energy = 0.5
				return c
			}(),
			wantValid: true,
		},
		{
			name:        "playing tired",
			interaction: types.InteractionPlaying,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Energy = 0.1
				return c
			}(),
			wantValid: false,
		},
		{
			name:        "training normal",
			interaction: types.InteractionTraining,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Energy = 0.5
				c.Stress = 0.3
				return c
			}(),
			wantValid: true,
		},
		{
			name:        "training stressed",
			interaction: types.InteractionTraining,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Stress = 0.9
				return c
			}(),
			wantValid: false,
		},
		{
			name:        "feeding hungry",
			interaction: types.InteractionFeeding,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Hunger = 0.3
				return c
			}(),
			wantValid: true,
		},
		{
			name:        "feeding full",
			interaction: types.InteractionFeeding,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Hunger = 0.95
				return c
			}(),
			wantValid: false,
		},
		{
			name:        "grooming dirty",
			interaction: types.InteractionGrooming,
			ctx:         defaultTestContext(),
			wantValid:   true,
		},
		{
			name:        "grooming clean",
			interaction: types.InteractionGrooming,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Cleanliness = 0.99
				return c
			}(),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := ValidateInteraction(tt.interaction, tt.ctx)
			if valid != tt.wantValid {
				t.Errorf("ValidateInteraction() valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

// TestCalculateEffectiveness tests effectiveness calculation
func TestCalculateEffectiveness(t *testing.T) {
	ctx := defaultTestContext()

	// Base effectiveness
	eff := CalculateEffectiveness(types.InteractionPetting, ctx)
	if eff <= 0 || eff > 1.5 {
		t.Errorf("Effectiveness out of expected range: %f", eff)
	}

	// Low energy should reduce effectiveness
	ctx.Energy = 0.1
	lowEnergyEff := CalculateEffectiveness(types.InteractionPetting, ctx)
	if lowEnergyEff >= eff {
		t.Error("Low energy should reduce effectiveness")
	}

	// High stress should reduce effectiveness
	ctx = defaultTestContext()
	ctx.Stress = 0.9
	highStressEff := CalculateEffectiveness(types.InteractionPetting, ctx)
	if highStressEff >= eff {
		t.Error("High stress should reduce effectiveness")
	}
}

// TestIntensityClamping tests that intensity is clamped to [0,1]
func TestIntensityClamping(t *testing.T) {
	proc := NewInteractionProcessor()
	ctx := defaultTestContext()

	// Test with intensity > 1
	result1 := proc.Process(types.InteractionPetting, 5.0, ctx)
	proc.ResetCooldowns()

	// Test with intensity = 1
	result2 := proc.Process(types.InteractionPetting, 1.0, ctx)

	// Effects should be the same (clamped to 1.0)
	if result1.Effects["happiness"] != result2.Effects["happiness"] {
		t.Error("Intensity should be clamped to 1.0")
	}
}

// TestEffectMultiplier tests global effect multiplier
func TestEffectMultiplier(t *testing.T) {
	config := ProcessorConfig{
		BaseCooldownSeconds: 0.0,
		EffectMultiplier:    2.0,
		CooldownOverrides:   make(map[types.InteractionType]float64),
	}
	procDouble := NewInteractionProcessorWithConfig(config)

	config.EffectMultiplier = 1.0
	procNormal := NewInteractionProcessorWithConfig(config)

	ctx := defaultTestContext()

	resultDouble := procDouble.Process(types.InteractionPetting, 1.0, ctx)
	resultNormal := procNormal.Process(types.InteractionPetting, 1.0, ctx)

	// With 2x multiplier, effects should be double
	if resultDouble.Effects["happiness"] != resultNormal.Effects["happiness"]*2 {
		t.Errorf("Expected 2x happiness, got %f vs %f", resultDouble.Effects["happiness"], resultNormal.Effects["happiness"])
	}
}

// =============== Item and Inventory Tests ===============

// TestNewInventory tests inventory creation
func TestNewInventory(t *testing.T) {
	inv := NewInventory(20)

	if inv == nil {
		t.Fatal("NewInventory returned nil")
	}

	if inv.MaxCapacity != 20 {
		t.Errorf("Expected capacity 20, got %d", inv.MaxCapacity)
	}

	if inv.Currency != 100 {
		t.Errorf("Expected starting currency 100, got %d", inv.Currency)
	}

	if inv.Items == nil {
		t.Error("Items map should not be nil")
	}
}

// TestAddItem tests adding items to inventory
func TestAddItem(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{
		ID:   "test_item",
		Name: "Test Item",
		Type: ItemTypeFood,
	}

	success := inv.AddItem(item, 5)
	if !success {
		t.Error("AddItem should succeed")
	}

	if inv.GetItemQuantity("test_item") != 5 {
		t.Errorf("Expected quantity 5, got %d", inv.GetItemQuantity("test_item"))
	}

	// Add more of same item
	success = inv.AddItem(item, 3)
	if !success {
		t.Error("Adding more of same item should succeed")
	}

	if inv.GetItemQuantity("test_item") != 8 {
		t.Errorf("Expected quantity 8, got %d", inv.GetItemQuantity("test_item"))
	}
}

// TestAddItemInventoryFull tests adding items when inventory is full
func TestAddItemInventoryFull(t *testing.T) {
	inv := NewInventory(2)

	item1 := &Item{ID: "item1", Name: "Item 1"}
	item2 := &Item{ID: "item2", Name: "Item 2"}
	item3 := &Item{ID: "item3", Name: "Item 3"}

	inv.AddItem(item1, 1)
	inv.AddItem(item2, 1)

	// Should fail - inventory full
	success := inv.AddItem(item3, 1)
	if success {
		t.Error("Adding item to full inventory should fail")
	}

	// But adding to existing stack should work
	success = inv.AddItem(item1, 1)
	if !success {
		t.Error("Adding to existing stack should succeed even when inventory is full")
	}
}

// TestRemoveItem tests removing items from inventory
func TestRemoveItem(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{ID: "test_item", Name: "Test Item"}
	inv.AddItem(item, 5)

	success := inv.RemoveItem("test_item", 3)
	if !success {
		t.Error("RemoveItem should succeed")
	}

	if inv.GetItemQuantity("test_item") != 2 {
		t.Errorf("Expected quantity 2, got %d", inv.GetItemQuantity("test_item"))
	}

	// Remove all remaining
	inv.RemoveItem("test_item", 2)
	if inv.HasItem("test_item") {
		t.Error("Item should be removed from inventory when quantity reaches 0")
	}
}

// TestRemoveItemNotEnough tests removing more items than available
func TestRemoveItemNotEnough(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{ID: "test_item", Name: "Test Item"}
	inv.AddItem(item, 2)

	success := inv.RemoveItem("test_item", 5)
	if success {
		t.Error("Removing more items than available should fail")
	}

	// Quantity should be unchanged
	if inv.GetItemQuantity("test_item") != 2 {
		t.Errorf("Quantity should be unchanged, got %d", inv.GetItemQuantity("test_item"))
	}
}

// TestUseItem tests using items
func TestUseItem(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{
		ID:       "food_item",
		Name:     "Food",
		Type:     ItemTypeFood,
		Cooldown: 0,
		Effects:  map[string]float64{"hunger": -0.3},
	}
	inv.AddItem(item, 3)

	usedItem, err := inv.UseItem("food_item")
	if err != nil {
		t.Errorf("UseItem failed: %v", err)
	}

	if usedItem == nil {
		t.Error("UseItem should return the item")
	}

	// Food is consumable - quantity should decrease
	if inv.GetItemQuantity("food_item") != 2 {
		t.Errorf("Expected quantity 2 after use, got %d", inv.GetItemQuantity("food_item"))
	}
}

// TestUseItemCooldown tests item cooldown
func TestUseItemCooldown(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{
		ID:       "toy_item",
		Name:     "Toy",
		Type:     ItemTypeToy,
		Cooldown: 60, // 60 second cooldown
		Uses:     -1, // Unlimited uses
	}
	inv.AddItem(item, 1)

	// First use should work
	_, err := inv.UseItem("toy_item")
	if err != nil {
		t.Errorf("First use should succeed: %v", err)
	}

	// Immediate second use should fail due to cooldown
	_, err = inv.UseItem("toy_item")
	if err == nil {
		t.Error("Second use should fail due to cooldown")
	}
}

// TestUseItemNotFound tests using non-existent item
func TestUseItemNotFound(t *testing.T) {
	inv := NewInventory(10)

	_, err := inv.UseItem("nonexistent")
	if err == nil {
		t.Error("Using non-existent item should fail")
	}
}

// TestHasItem tests item existence check
func TestHasItem(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{ID: "test_item", Name: "Test Item"}
	inv.AddItem(item, 1)

	if !inv.HasItem("test_item") {
		t.Error("HasItem should return true for existing item")
	}

	if inv.HasItem("nonexistent") {
		t.Error("HasItem should return false for non-existent item")
	}
}

// TestGetItemsByType tests filtering items by type
func TestGetItemsByType(t *testing.T) {
	inv := NewInventory(10)
	food1 := &Item{ID: "food1", Name: "Food 1", Type: ItemTypeFood}
	food2 := &Item{ID: "food2", Name: "Food 2", Type: ItemTypeFood}
	toy := &Item{ID: "toy1", Name: "Toy 1", Type: ItemTypeToy}

	inv.AddItem(food1, 1)
	inv.AddItem(food2, 1)
	inv.AddItem(toy, 1)

	foods := inv.GetItemsByType(ItemTypeFood)
	if len(foods) != 2 {
		t.Errorf("Expected 2 food items, got %d", len(foods))
	}

	toys := inv.GetItemsByType(ItemTypeToy)
	if len(toys) != 1 {
		t.Errorf("Expected 1 toy item, got %d", len(toys))
	}
}

// TestCurrency tests currency operations
func TestCurrency(t *testing.T) {
	inv := NewInventory(10)

	if !inv.CanAfford(50) {
		t.Error("Should be able to afford 50 with 100 starting currency")
	}

	if inv.CanAfford(200) {
		t.Error("Should not be able to afford 200 with 100 starting currency")
	}

	success := inv.SpendCurrency(30)
	if !success {
		t.Error("SpendCurrency should succeed")
	}
	if inv.Currency != 70 {
		t.Errorf("Expected 70 currency, got %d", inv.Currency)
	}

	success = inv.SpendCurrency(100)
	if success {
		t.Error("SpendCurrency should fail when insufficient funds")
	}

	inv.AddCurrency(50)
	if inv.Currency != 120 {
		t.Errorf("Expected 120 currency after adding 50, got %d", inv.Currency)
	}
}

// TestNewItemCatalog tests catalog creation
func TestNewItemCatalog(t *testing.T) {
	catalog := NewItemCatalog()

	if catalog == nil {
		t.Fatal("NewItemCatalog returned nil")
	}

	if len(catalog.Items) == 0 {
		t.Error("Catalog should have default items")
	}

	// Check for some expected items
	if _, exists := catalog.Items["basic_food"]; !exists {
		t.Error("Catalog should contain basic_food")
	}

	if _, exists := catalog.Items["ball"]; !exists {
		t.Error("Catalog should contain ball")
	}
}

// TestCatalogGetItem tests retrieving items from catalog
func TestCatalogGetItem(t *testing.T) {
	catalog := NewItemCatalog()

	item, exists := catalog.GetItem("basic_food")
	if !exists {
		t.Error("basic_food should exist in catalog")
	}
	if item.Name != "Basic Pet Food" {
		t.Errorf("Expected 'Basic Pet Food', got '%s'", item.Name)
	}

	_, exists = catalog.GetItem("nonexistent")
	if exists {
		t.Error("Nonexistent item should not exist")
	}
}

// TestCatalogGetItemsByType tests filtering catalog by type
func TestCatalogGetItemsByType(t *testing.T) {
	catalog := NewItemCatalog()

	foods := catalog.GetItemsByType(ItemTypeFood)
	if len(foods) == 0 {
		t.Error("Should have food items")
	}

	toys := catalog.GetItemsByType(ItemTypeToy)
	if len(toys) == 0 {
		t.Error("Should have toy items")
	}
}

// TestCatalogGetUnlockedItems tests getting unlocked items
func TestCatalogGetUnlockedItems(t *testing.T) {
	catalog := NewItemCatalog()

	unlocked := catalog.GetUnlockedItems()
	if len(unlocked) == 0 {
		t.Error("Should have some unlocked items by default")
	}

	// Verify all returned items are actually unlocked
	for _, item := range unlocked {
		if !item.Unlocked {
			t.Errorf("Item %s returned as unlocked but Unlocked=false", item.ID)
		}
	}
}

// TestCatalogUnlockItem tests unlocking items
func TestCatalogUnlockItem(t *testing.T) {
	catalog := NewItemCatalog()

	// premium_food starts locked
	item, _ := catalog.GetItem("premium_food")
	if item.Unlocked {
		t.Error("premium_food should start locked")
	}

	success := catalog.UnlockItem("premium_food")
	if !success {
		t.Error("UnlockItem should succeed")
	}

	item, _ = catalog.GetItem("premium_food")
	if !item.Unlocked {
		t.Error("premium_food should be unlocked now")
	}

	// Unlocking non-existent item should fail
	success = catalog.UnlockItem("nonexistent")
	if success {
		t.Error("Unlocking non-existent item should fail")
	}
}

// TestCatalogGetItemsByRarity tests filtering by rarity
func TestCatalogGetItemsByRarity(t *testing.T) {
	catalog := NewItemCatalog()

	commonItems := catalog.GetItemsByRarity(RarityCommon)
	if len(commonItems) == 0 {
		t.Error("Should have common items")
	}

	rareItems := catalog.GetItemsByRarity(RarityRare)
	if len(rareItems) == 0 {
		t.Error("Should have rare items")
	}
}

// TestShop tests shop functionality
func TestShop(t *testing.T) {
	catalog := NewItemCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	if shop == nil {
		t.Fatal("NewShop returned nil")
	}
}

// TestShopGetPrice tests price retrieval
func TestShopGetPrice(t *testing.T) {
	catalog := NewItemCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	price, err := shop.GetPrice("basic_food")
	if err != nil {
		t.Errorf("GetPrice failed: %v", err)
	}
	if price != 10 {
		t.Errorf("Expected price 10, got %d", price)
	}

	_, err = shop.GetPrice("nonexistent")
	if err == nil {
		t.Error("GetPrice should fail for non-existent item")
	}
}

// TestShopBuy tests purchasing items
func TestShopBuy(t *testing.T) {
	catalog := NewItemCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	// Buy basic_food (10 currency each)
	err := shop.Buy("basic_food", 2)
	if err != nil {
		t.Errorf("Buy failed: %v", err)
	}

	if inv.GetItemQuantity("basic_food") != 2 {
		t.Errorf("Expected 2 basic_food in inventory, got %d", inv.GetItemQuantity("basic_food"))
	}

	if inv.Currency != 80 { // 100 - 20
		t.Errorf("Expected 80 currency remaining, got %d", inv.Currency)
	}
}

// TestShopBuyNotEnoughMoney tests buying without sufficient funds
func TestShopBuyNotEnoughMoney(t *testing.T) {
	catalog := NewItemCatalog()
	inv := NewInventory(20)
	inv.Currency = 5 // Only 5 currency
	shop := NewShop(catalog, inv)

	err := shop.Buy("basic_food", 1) // Costs 10
	if err == nil {
		t.Error("Buy should fail with insufficient funds")
	}
}

// TestShopBuyLockedItem tests buying locked items
func TestShopBuyLockedItem(t *testing.T) {
	catalog := NewItemCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	// premium_food is locked by default
	err := shop.Buy("premium_food", 1)
	if err == nil {
		t.Error("Buy should fail for locked items")
	}
}

// TestShopDiscount tests discount system
func TestShopDiscount(t *testing.T) {
	catalog := NewItemCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	// Set 50% discount
	shop.SetDiscount("basic_food", 0.5)

	price, _ := shop.GetPrice("basic_food")
	if price != 5 { // 50% of 10
		t.Errorf("Expected discounted price 5, got %d", price)
	}

	// Clear discount
	shop.ClearDiscount("basic_food")
	price, _ = shop.GetPrice("basic_food")
	if price != 10 {
		t.Errorf("Expected full price 10 after clearing discount, got %d", price)
	}
}

// TestShopDiscountBounds tests discount clamping
func TestShopDiscountBounds(t *testing.T) {
	catalog := NewItemCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	// Negative discount should be clamped to 0
	shop.SetDiscount("basic_food", -0.5)
	if shop.Discounts["basic_food"] != 0 {
		t.Error("Negative discount should be clamped to 0")
	}

	// Over 100% discount should be clamped to 1
	shop.SetDiscount("basic_food", 1.5)
	if shop.Discounts["basic_food"] != 1 {
		t.Error("Discount over 1 should be clamped to 1")
	}
}

// TestShopGetAvailableItems tests getting purchasable items
func TestShopGetAvailableItems(t *testing.T) {
	catalog := NewItemCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	available := shop.GetAvailableItems()
	if len(available) == 0 {
		t.Error("Should have some available items")
	}

	// All available items should be unlocked
	for _, item := range available {
		if !item.Unlocked {
			t.Errorf("Available item %s should be unlocked", item.ID)
		}
	}
}

// TestItemTypeString tests item type string conversion
func TestItemTypeString(t *testing.T) {
	tests := []struct {
		itemType ItemType
		expected string
	}{
		{ItemTypeFood, "Food"},
		{ItemTypeToy, "Toy"},
		{ItemTypeMedicine, "Medicine"},
		{ItemTypeAccessory, "Accessory"},
		{ItemTypeGrooming, "Grooming"},
	}

	for _, tt := range tests {
		result := tt.itemType.String()
		if result != tt.expected {
			t.Errorf("ItemType.String() = %s, want %s", result, tt.expected)
		}
	}
}

// TestItemRarityString tests item rarity string conversion
func TestItemRarityString(t *testing.T) {
	tests := []struct {
		rarity   ItemRarity
		expected string
	}{
		{RarityCommon, "Common"},
		{RarityUncommon, "Uncommon"},
		{RarityRare, "Rare"},
		{RarityEpic, "Epic"},
		{RarityLegendary, "Legendary"},
	}

	for _, tt := range tests {
		result := tt.rarity.String()
		if result != tt.expected {
			t.Errorf("ItemRarity.String() = %s, want %s", result, tt.expected)
		}
	}
}
