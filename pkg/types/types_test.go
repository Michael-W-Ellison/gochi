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
