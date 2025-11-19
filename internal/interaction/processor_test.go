package interaction

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Helper function to create a test context
func createTestContext() *InteractionContext {
	return &InteractionContext{
		Vitals:         biology.NewVitalStats(),
		Emotions:       ai.NewEmotionState(),
		Personality:    ai.NewPersonalityMatrix(),
		Needs:          simulation.NewNeedsManager(),
		Memory:         ai.NewMemorySystem(100),
		Learning:       ai.NewLearningSystem(),
		Energy:         0.7,
		CurrentMood:    0.6,
		Fatigue:        0.3,
		Age:            30,
		UserID:         "test_user",
		UserSkillLevel: 0.5,
		BondStrength:   0.6,
	}
}

func TestNewInteractionProcessor(t *testing.T) {
	ip := NewInteractionProcessor()
	if ip == nil {
		t.Fatal("NewInteractionProcessor returned nil")
	}
	if ip.skills == nil {
		t.Error("skills map not initialized")
	}
	if ip.rand == nil {
		t.Error("random source not initialized")
	}
}

func TestProcessFeeding(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name     string
		foodType FoodType
		intensity float64
		wantSuccess bool
	}{
		{"Basic Kibble", FoodTypeBasicKibble, 1.0, true},
		{"Premium Kibble", FoodTypePremiumKibble, 1.0, true},
		{"Treat", FoodTypeTreat, 0.5, true},
		{"Special Meal", FoodTypeSpecialMeal, 1.0, true},
		{"Junk Food", FoodTypeJunkFood, 0.8, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ip.ProcessInteraction(
				types.InteractionFeeding,
				tt.intensity,
				ctx,
				tt.foodType,
			)

			if result.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if result.Feedback == "" {
				t.Error("Expected feedback message")
			}

			// Check that nutrition was affected
			if result.VitalChanges["Nutrition"] <= 0 {
				t.Error("Expected positive nutrition change")
			}

			// Special meal should provide maximum benefits
			if tt.foodType == FoodTypeSpecialMeal {
				if result.VitalChanges["Happiness"] <= 0 {
					t.Error("Special meal should increase happiness")
				}
			}

			// Junk food should have warnings
			if tt.foodType == FoodTypeJunkFood {
				if len(result.Warnings) == 0 {
					t.Error("Expected warning for junk food")
				}
			}
		})
	}
}

func TestProcessPetting(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name string
		intensity float64
		affectionate float64
		independent float64
	}{
		{"Normal Petting", 1.0, 0.5, 0.5},
		{"Affectionate Pet", 1.0, 0.9, 0.2},
		{"Independent Pet", 1.0, 0.2, 0.9},
		{"Light Petting", 0.3, 0.5, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.Personality.Traits.Affectionate = tt.affectionate
			ctx.Personality.Traits.Independence = tt.independent

			result := ip.ProcessInteraction(
				types.InteractionPetting,
				tt.intensity,
				ctx,
				nil,
			)

			if !result.Success {
				t.Error("Petting should succeed")
			}

			// Petting should reduce stress
			if result.VitalChanges["Stress"] >= 0 {
				t.Error("Petting should reduce stress")
			}

			// Should increase affection need
			if result.NeedChanges["Affection"] <= 0 {
				t.Error("Petting should satisfy affection need")
			}

			// Affectionate pets should respond better
			if tt.affectionate > 0.8 && result.VitalChanges["Happiness"] <= 0 {
				t.Error("Affectionate pet should be happier from petting")
			}
		})
	}
}

func TestProcessPlaying(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name string
		toyType ToyType
		intensity float64
	}{
		{"Ball Play", ToyTypeBall, 1.0},
		{"Puzzle Toy", ToyTypePuzzle, 0.8},
		{"Interactive Toy", ToyTypeInteractive, 1.0},
		{"Laser Pointer", ToyTypeLaser, 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set high energy for play
			ctx.Energy = 0.8

			result := ip.ProcessInteraction(
				types.InteractionPlaying,
				tt.intensity,
				ctx,
				tt.toyType,
			)

			if !result.Success {
				t.Error("Playing should succeed when pet has energy")
			}

			// Playing should drain energy
			if result.VitalChanges["Energy"] >= 0 {
				t.Error("Playing should drain energy")
			}

			// Should increase happiness
			if result.VitalChanges["Happiness"] <= 0 {
				t.Error("Playing should increase happiness")
			}

			// Should satisfy exercise need
			if result.NeedChanges["Exercise"] <= 0 {
				t.Error("Playing should satisfy exercise need")
			}

			// Puzzle toys should provide more mental stimulation
			if tt.toyType == ToyTypePuzzle {
				if result.NeedChanges["MentalStimulation"] <= 0 {
					t.Error("Puzzle toy should provide mental stimulation")
				}
			}
		})
	}
}

func TestProcessPlayingLowEnergy(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()
	ctx.Energy = 0.1 // Too low for play

	result := ip.ProcessInteraction(
		types.InteractionPlaying,
		1.0,
		ctx,
		ToyTypeBall,
	)

	if result.Success {
		t.Error("Playing should fail when pet has low energy")
	}

	if len(result.Warnings) == 0 {
		t.Error("Expected warning about low energy")
	}
}

func TestProcessTraining(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name string
		skillType SkillType
		intensity float64
	}{
		{"Obedience Training", SkillObedience, 1.0},
		{"Agility Training", SkillAgility, 0.8},
		{"Trick Training", SkillTricks, 0.9},
		{"Socialization", SkillSocialization, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure good conditions for training
			ctx.Energy = 0.7
			ctx.Fatigue = 0.3
			ctx.CurrentMood = 0.7

			result := ip.ProcessInteraction(
				types.InteractionTraining,
				tt.intensity,
				ctx,
				tt.skillType,
			)

			if !result.Success {
				t.Error("Training should succeed with good conditions")
			}

			if result.SkillProgress == nil {
				t.Fatal("Expected skill progress data")
			}

			if result.SkillProgress.Skill != tt.skillType {
				t.Errorf("Skill type = %v, want %v", result.SkillProgress.Skill, tt.skillType)
			}

			// Check that skill was created/updated
			skill := ip.getOrCreateSkill(tt.skillType)
			if skill == nil {
				t.Error("Skill should be created after training")
			}

			if skill.Experience <= 0 {
				t.Error("Skill should gain experience from training")
			}

			// Training should drain energy
			if result.VitalChanges["Energy"] >= 0 {
				t.Error("Training should drain energy")
			}

			// Should provide mental stimulation
			if result.NeedChanges["MentalStimulation"] <= 0 {
				t.Error("Training should provide mental stimulation")
			}
		})
	}
}

func TestProcessTrainingBadConditions(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()
	ctx.Energy = 0.2 // Too tired
	ctx.Fatigue = 0.8

	result := ip.ProcessInteraction(
		types.InteractionTraining,
		1.0,
		ctx,
		SkillObedience,
	)

	if result.Success {
		t.Error("Training should fail when pet is too tired")
	}
}

func TestProcessGrooming(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name string
		toolType GroomingToolType
		userSkill float64
	}{
		{"Brush", GroomingToolBrush, 0.5},
		{"Shampoo", GroomingToolShampoo, 0.6},
		{"Nail Clipper - Skilled", GroomingToolNailClipper, 0.9},
		{"Nail Clipper - Unskilled", GroomingToolNailClipper, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.UserSkillLevel = tt.userSkill

			result := ip.ProcessInteraction(
				types.InteractionGrooming,
				1.0,
				ctx,
				tt.toolType,
			)

			if !result.Success {
				t.Error("Grooming should succeed")
			}

			// Grooming should improve cleanliness
			if result.VitalChanges["Cleanliness"] <= 0 {
				t.Error("Grooming should improve cleanliness")
			}

			if result.NeedChanges["Cleanliness"] <= 0 {
				t.Error("Grooming should satisfy cleanliness need")
			}

			// Check skill warnings for difficult tools
			props := GetGroomingToolProperties(tt.toolType)
			if tt.userSkill < props.SkillLevelRequired {
				if len(result.Warnings) == 0 {
					t.Error("Expected warning for insufficient skill")
				}
			}
		})
	}
}

func TestProcessMedicalCare(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name string
		itemType MedicalItemType
		userSkill float64
		shouldSucceed bool
	}{
		{"Vitamins - Anyone", MedicalItemVitamins, 0.3, true},
		{"Antibiotics - Skilled", MedicalItemAntibiotics, 0.8, true},
		{"Antibiotics - Unskilled", MedicalItemAntibiotics, 0.3, false},
		{"Bandage", MedicalItemBandage, 0.5, true},
		{"Vaccine - Skilled", MedicalItemVaccine, 0.9, true},
		{"Vaccine - Unskilled", MedicalItemVaccine, 0.4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.UserSkillLevel = tt.userSkill

			result := ip.ProcessInteraction(
				types.InteractionMedicalCare,
				1.0,
				ctx,
				tt.itemType,
			)

			if result.Success != tt.shouldSucceed {
				t.Errorf("Success = %v, want %v", result.Success, tt.shouldSucceed)
			}

			if tt.shouldSucceed {
				// Medical care should improve health
				if result.VitalChanges["Health"] <= 0 {
					t.Error("Medical care should improve health")
				}

				if result.NeedChanges["MedicalCare"] <= 0 {
					t.Error("Medical care should satisfy medical need")
				}
			} else {
				// Should have warning about skill
				if len(result.Warnings) == 0 {
					t.Error("Expected warning for insufficient skill")
				}
			}
		})
	}
}

func TestProcessEnvironmentalEnrichment(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name string
		curiosity float64
		intelligence float64
	}{
		{"Normal Pet", 0.5, 0.5},
		{"Curious Pet", 0.9, 0.7},
		{"Less Curious", 0.2, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.Personality.Traits.Curiosity = tt.curiosity
			ctx.Personality.Traits.Intelligence = tt.intelligence

			result := ip.ProcessInteraction(
				types.InteractionEnvironmentalEnrichment,
				1.0,
				ctx,
				nil,
			)

			if !result.Success {
				t.Error("Environmental enrichment should succeed")
			}

			// Should provide mental stimulation
			if result.NeedChanges["MentalStimulation"] <= 0 {
				t.Error("Should provide mental stimulation")
			}

			// Should satisfy exploration
			if result.NeedChanges["Exploration"] <= 0 {
				t.Error("Should satisfy exploration need")
			}

			// Curious pets should benefit more
			if tt.curiosity > 0.8 {
				if result.NeedChanges["MentalStimulation"] < 0.5 {
					t.Error("Curious pets should benefit more from enrichment")
				}
			}
		})
	}
}

func TestProcessSocialIntroduction(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name string
		extraversion float64
		adaptability float64
	}{
		{"Extraverted Pet", 0.8, 0.7},
		{"Introverted Pet", 0.2, 0.5},
		{"Adaptable Pet", 0.5, 0.9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.Personality.Traits.Extraversion = tt.extraversion
			ctx.Personality.Traits.Adaptability = tt.adaptability

			result := ip.ProcessInteraction(
				types.InteractionSocialIntroduction,
				1.0,
				ctx,
				nil,
			)

			if !result.Success {
				t.Error("Social introduction should succeed")
			}

			// Should satisfy social need
			if result.NeedChanges["Social"] <= 0 {
				t.Error("Should satisfy social need")
			}

			// Introverted pets should be more stressed
			if tt.extraversion < 0.3 {
				if result.VitalChanges["Stress"] <= 0 {
					t.Error("Introverted pet should be stressed by social interaction")
				}
			}
		})
	}
}

func TestProcessDiscipline(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	tests := []struct {
		name string
		intensity float64
		userSkill float64
	}{
		{"Appropriate Discipline", 0.5, 0.7},
		{"Too Harsh", 0.9, 0.5},
		{"Unskilled", 0.6, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.UserSkillLevel = tt.userSkill

			result := ip.ProcessInteraction(
				types.InteractionDiscipline,
				tt.intensity,
				ctx,
				nil,
			)

			if !result.Success {
				t.Error("Discipline should succeed (though may not be effective)")
			}

			// Discipline should increase stress
			if result.VitalChanges["Stress"] <= 0 {
				t.Error("Discipline should increase stress")
			}

			// Discipline should decrease happiness
			if result.VitalChanges["Happiness"] >= 0 {
				t.Error("Discipline should decrease happiness")
			}

			// Too harsh should have warnings
			if tt.intensity > 0.8 {
				if len(result.Warnings) == 0 {
					t.Error("Expected warning for harsh discipline")
				}
			}
		})
	}
}

func TestProcessRewards(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	// First, train a skill
	ip.ProcessInteraction(
		types.InteractionTraining,
		1.0,
		ctx,
		SkillObedience,
	)

	// Now reward
	result := ip.ProcessInteraction(
		types.InteractionRewards,
		1.0,
		ctx,
		nil,
	)

	if !result.Success {
		t.Error("Rewards should succeed")
	}

	// Rewards should increase happiness
	if result.VitalChanges["Happiness"] <= 0 {
		t.Error("Rewards should increase happiness")
	}

	// Should reduce stress
	if result.VitalChanges["Stress"] >= 0 {
		t.Error("Rewards should reduce stress")
	}

	// Should satisfy affection
	if result.NeedChanges["Affection"] <= 0 {
		t.Error("Rewards should satisfy affection need")
	}

	// Should reinforce recent training
	skill := ip.getSkillLevel(SkillObedience)
	if skill <= 0 {
		t.Error("Reward should reinforce trained skill")
	}
}

func TestCalculateEffectiveness(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	result := &InteractionResult{
		Success: true,
		VitalChanges: map[string]float64{
			"Happiness": 0.5,
			"Stress": -0.3,
		},
		EmotionChanges: map[string]float64{
			"Joy": 0.4,
		},
		NeedChanges: map[string]float64{
			"Affection": 0.6,
		},
	}

	effectiveness := ip.calculateEffectiveness(ctx, types.InteractionPetting, result)

	if effectiveness <= 0 || effectiveness > 1 {
		t.Errorf("Effectiveness %f should be between 0 and 1", effectiveness)
	}

	// Higher mood should increase effectiveness
	ctx.CurrentMood = 0.9
	highMoodEffectiveness := ip.calculateEffectiveness(ctx, types.InteractionPetting, result)

	ctx.CurrentMood = 0.2
	lowMoodEffectiveness := ip.calculateEffectiveness(ctx, types.InteractionPetting, result)

	if highMoodEffectiveness <= lowMoodEffectiveness {
		t.Error("Higher mood should result in better effectiveness")
	}
}

func TestSkillProgression(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	skillType := SkillObedience
	initialLevel := ip.getSkillLevel(skillType)

	if initialLevel != 0 {
		t.Error("Initial skill level should be 0")
	}

	// Train multiple times
	for i := 0; i < 10; i++ {
		ctx.Energy = 0.8 // Reset energy
		ctx.Fatigue = 0.2
		ip.ProcessInteraction(
			types.InteractionTraining,
			1.0,
			ctx,
			skillType,
		)
	}

	finalLevel := ip.getSkillLevel(skillType)

	if finalLevel <= initialLevel {
		t.Error("Skill level should increase with training")
	}

	skill := ip.getOrCreateSkill(skillType)
	if skill.Experience <= 0 {
		t.Error("Skill should have accumulated experience")
	}
}

func TestSkillDecay(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	// Train a skill
	skillType := SkillObedience
	ip.ProcessInteraction(
		types.InteractionTraining,
		1.0,
		ctx,
		skillType,
	)

	initialLevel := ip.getSkillLevel(skillType)

	// Simulate time passing
	ip.UpdateSkillDecay(1000.0) // Large time delta

	decayedLevel := ip.getSkillLevel(skillType)

	if decayedLevel > initialLevel {
		t.Error("Skill level should decay or stay same, not increase")
	}
}

func TestMemoryCreation(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	result := ip.ProcessInteraction(
		types.InteractionPetting,
		1.0,
		ctx,
		nil,
	)

	if !result.MemoryCreated {
		t.Error("High effectiveness interaction should create memory")
	}

	// Check that memory was actually created
	memories := ctx.Memory.RecallMemories(ai.MemoryInteraction, 10)
	if len(memories) == 0 {
		t.Error("Memory should be present in memory system")
	}
}

func TestInteractionHistory(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	// Perform several interactions
	ip.ProcessInteraction(types.InteractionFeeding, 1.0, ctx, FoodTypeBasicKibble)
	ip.ProcessInteraction(types.InteractionPetting, 1.0, ctx, nil)
	ip.ProcessInteraction(types.InteractionPlaying, 0.8, ctx, ToyTypeBall)

	history := ip.GetInteractionHistory()

	if len(history) != 3 {
		t.Errorf("History length = %d, want 3", len(history))
	}

	// Check that history contains expected types
	foundFeeding := false
	foundPetting := false
	foundPlaying := false

	for _, record := range history {
		switch record.Type {
		case types.InteractionFeeding:
			foundFeeding = true
		case types.InteractionPetting:
			foundPetting = true
		case types.InteractionPlaying:
			foundPlaying = true
		}
	}

	if !foundFeeding || !foundPetting || !foundPlaying {
		t.Error("History should contain all interaction types")
	}
}

func TestGetAllSkills(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	// Train several skills
	ip.ProcessInteraction(types.InteractionTraining, 1.0, ctx, SkillObedience)
	ip.ProcessInteraction(types.InteractionTraining, 1.0, ctx, SkillAgility)
	ip.ProcessInteraction(types.InteractionTraining, 1.0, ctx, SkillTricks)

	skills := ip.GetAllSkills()

	if len(skills) != 3 {
		t.Errorf("Skills count = %d, want 3", len(skills))
	}

	if _, exists := skills[SkillObedience]; !exists {
		t.Error("Should have obedience skill")
	}

	if _, exists := skills[SkillAgility]; !exists {
		t.Error("Should have agility skill")
	}

	if _, exists := skills[SkillTricks]; !exists {
		t.Error("Should have tricks skill")
	}
}

func TestExhaustedPetInteraction(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()
	ctx.Fatigue = 0.95 // Extremely tired

	result := ip.ProcessInteraction(
		types.InteractionPlaying,
		1.0,
		ctx,
		ToyTypeBall,
	)

	if result.Success {
		t.Error("Exhausted pet should not be able to interact")
	}

	if len(result.Warnings) == 0 {
		t.Error("Should have warning about exhaustion")
	}
}

func TestConcurrentAccess(t *testing.T) {
	ip := NewInteractionProcessor()
	ctx := createTestContext()

	// Test that concurrent access doesn't cause race conditions
	done := make(chan bool)

	// Multiple goroutines performing interactions
	for i := 0; i < 10; i++ {
		go func() {
			ip.ProcessInteraction(types.InteractionPetting, 1.0, ctx, nil)
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should be able to get history without panic
	history := ip.GetInteractionHistory()
	if len(history) != 10 {
		t.Errorf("Expected 10 interactions, got %d", len(history))
	}
}
