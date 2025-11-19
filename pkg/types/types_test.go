package types

import "testing"

func TestBehaviorStateString(t *testing.T) {
	tests := []struct {
		state    BehaviorState
		expected string
	}{
		{BehaviorIdle, "Idle"},
		{BehaviorSleeping, "Sleeping"},
		{BehaviorEating, "Eating"},
		{BehaviorPlaying, "Playing"},
		{BehaviorHappy, "Happy"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.expected {
				t.Errorf("BehaviorState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInteractionTypeString(t *testing.T) {
	tests := []struct {
		interaction InteractionType
		expected    string
	}{
		{InteractionFeeding, "Feeding"},
		{InteractionPetting, "Petting"},
		{InteractionPlaying, "Playing"},
		{InteractionTraining, "Training"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.interaction.String()
			if got != tt.expected {
				t.Errorf("InteractionType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNeedTypeString(t *testing.T) {
	tests := []struct {
		need     NeedType
		expected string
	}{
		{NeedHunger, "Hunger"},
		{NeedThirst, "Thirst"},
		{NeedSleep, "Sleep"},
		{NeedSocial, "Social"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.need.String()
			if got != tt.expected {
				t.Errorf("NeedType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTimeScaleString(t *testing.T) {
	tests := []struct {
		scale    TimeScale
		expected string
	}{
		{TimeScaleRealTime, "RealTime"},
		{TimeScaleAccelerated4X, "Accelerated4X"},
		{TimeScaleAccelerated24X, "Accelerated24X"},
		{TimeScalePaused, "Paused"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.scale.String()
			if got != tt.expected {
				t.Errorf("TimeScale.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRelationshipTypeString(t *testing.T) {
	tests := []struct {
		relType  RelationshipType
		expected string
	}{
		{RelationshipFriend, "Friend"},
		{RelationshipRival, "Rival"},
		{RelationshipMate, "Mate"},
		{RelationshipOffspring, "Offspring"},
		{RelationshipParent, "Parent"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.relType.String()
			if got != tt.expected {
				t.Errorf("RelationshipType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAllBehaviorStates(t *testing.T) {
	allStates := []struct {
		state    BehaviorState
		expected string
	}{
		{BehaviorIdle, "Idle"},
		{BehaviorSleeping, "Sleeping"},
		{BehaviorEating, "Eating"},
		{BehaviorPlaying, "Playing"},
		{BehaviorExploring, "Exploring"},
		{BehaviorSocialInteraction, "Social Interaction"},
		{BehaviorGrooming, "Grooming"},
		{BehaviorExercising, "Exercising"},
		{BehaviorSick, "Sick"},
		{BehaviorDistressed, "Distressed"},
		{BehaviorHappy, "Happy"},
		{BehaviorExcited, "Excited"},
	}

	for _, tt := range allStates {
		got := tt.state.String()
		if got != tt.expected {
			t.Errorf("BehaviorState(%d).String() = %v, want %v", tt.state, got, tt.expected)
		}
	}
}

func TestAllInteractionTypes(t *testing.T) {
	allTypes := []struct {
		interaction InteractionType
		expected    string
	}{
		{InteractionFeeding, "Feeding"},
		{InteractionPetting, "Petting"},
		{InteractionPlaying, "Playing"},
		{InteractionTraining, "Training"},
		{InteractionGrooming, "Grooming"},
		{InteractionMedicalCare, "Medical Care"},
		{InteractionEnvironmentalEnrichment, "Environmental Enrichment"},
		{InteractionSocialIntroduction, "Social Introduction"},
		{InteractionDiscipline, "Discipline"},
		{InteractionRewards, "Rewards"},
	}

	for _, tt := range allTypes {
		got := tt.interaction.String()
		if got != tt.expected {
			t.Errorf("InteractionType(%d).String() = %v, want %v", tt.interaction, got, tt.expected)
		}
	}
}

func TestAllNeedTypes(t *testing.T) {
	allNeeds := []struct {
		need     NeedType
		expected string
	}{
		{NeedHunger, "Hunger"},
		{NeedThirst, "Thirst"},
		{NeedSleep, "Sleep"},
		{NeedExercise, "Exercise"},
		{NeedSocial, "Social"},
		{NeedMentalStimulation, "Mental Stimulation"},
		{NeedAffection, "Affection"},
		{NeedCleanliness, "Cleanliness"},
		{NeedMedicalCare, "Medical Care"},
		{NeedExploration, "Exploration"},
	}

	for _, tt := range allNeeds {
		got := tt.need.String()
		if got != tt.expected {
			t.Errorf("NeedType(%d).String() = %v, want %v", tt.need, got, tt.expected)
		}
	}
}

func TestPetIDType(t *testing.T) {
	id := PetID("test-pet-123")
	if string(id) != "test-pet-123" {
		t.Errorf("PetID conversion failed")
	}
}

func TestUserIDType(t *testing.T) {
	id := UserID("test-user-456")
	if string(id) != "test-user-456" {
		t.Errorf("UserID conversion failed")
	}
}

func TestPriorityConstants(t *testing.T) {
	// Test that priority constants are distinct
	priorities := []Priority{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityCritical,
	}

	seen := make(map[Priority]bool)
	for _, p := range priorities {
		if seen[p] {
			t.Errorf("Duplicate priority value: %d", p)
		}
		seen[p] = true
	}

	if len(priorities) != 4 {
		t.Errorf("Expected 4 priority levels, got %d", len(priorities))
	}
}
