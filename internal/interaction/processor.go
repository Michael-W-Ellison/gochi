package interaction

import "github.com/Michael-W-Ellison/gochi/pkg/types"

// InteractionContext provides context for processing interactions
type InteractionContext struct {
	// Pet state
	Health      float64
	Energy      float64
	Happiness   float64
	Hunger      float64
	Thirst      float64
	Cleanliness float64
	Stress      float64
	Fatigue     float64

	// Personality traits
	Playfulness  float64
	Independence float64
	Affectionate float64
	Intelligence float64

	// Current state
	CurrentBehavior types.BehaviorState
	PetAge          float64
	IsSleeping      bool
	IsSick          bool

	// Environment
	TimeOfDay  float64
	IsOutdoors bool
}

// GetInteractionDescription returns a description of an interaction type
func GetInteractionDescription(interactionType types.InteractionType) string {
	descriptions := map[types.InteractionType]string{
		types.InteractionFeeding:                 "Feed your pet to reduce hunger and restore energy",
		types.InteractionPetting:                 "Pet your companion to increase happiness and reduce stress",
		types.InteractionPlaying:                 "Play with your pet to boost happiness and exercise",
		types.InteractionTraining:                "Train your pet to improve skills and obedience",
		types.InteractionGrooming:                "Groom your pet to maintain cleanliness and health",
		types.InteractionMedicalCare:             "Provide medical care to improve health",
		types.InteractionRewards:                 "Give treats to reward good behavior",
		types.InteractionDiscipline:              "Discipline to correct unwanted behavior",
		types.InteractionEnvironmentalEnrichment: "Add enrichment to stimulate curiosity",
		types.InteractionSocialIntroduction:      "Introduce your pet to others for socialization",
		types.InteractionDrinking:                "Give your pet fresh water to quench its thirst",
		types.InteractionCleaning:                "Clean your pet's living area to keep it healthy",
		types.InteractionWalking:                 "Take your dog outside for exercise and a bathroom break",
		types.InteractionEnvironmentChange:       "Change something at home to keep your pet's mind busy",
	}

	if desc, exists := descriptions[interactionType]; exists {
		return desc
	}
	return "Interact with your pet"
}

// ValidateInteraction checks if an interaction is valid for the current context
func ValidateInteraction(interactionType types.InteractionType, ctx InteractionContext) (bool, string) {
	switch interactionType {
	case types.InteractionPlaying:
		if ctx.Energy < 0.2 {
			return false, "Pet is too tired to play"
		}
		if ctx.IsSleeping {
			return false, "Pet is sleeping"
		}

	case types.InteractionTraining:
		if ctx.Energy < 0.3 {
			return false, "Pet is too tired for training"
		}
		if ctx.Stress > 0.7 {
			return false, "Pet is too stressed for training"
		}
		if ctx.IsSleeping {
			return false, "Pet is sleeping"
		}

	case types.InteractionSocialIntroduction:
		if ctx.IsSleeping {
			return false, "Pet is sleeping"
		}

	case types.InteractionWalking:
		if ctx.Energy < 0.2 {
			return false, "Pet is too tired for a walk"
		}
		if ctx.IsSleeping {
			return false, "Pet is sleeping"
		}

	case types.InteractionFeeding:
		if ctx.Hunger > 0.9 {
			return false, "Pet is not hungry"
		}

	case types.InteractionGrooming:
		if ctx.IsSleeping {
			return false, "Pet is sleeping"
		}
		if ctx.Cleanliness > 0.95 {
			return false, "Pet is already clean"
		}

	case types.InteractionPetting:
		if ctx.IsSleeping {
			return false, "Pet is sleeping"
		}
	}

	return true, ""
}
