package core

import "github.com/Michael-W-Ellison/gochi/pkg/types"

// activityDescriptions describe each behavior as something the pet is
// doing, to follow its name ("Biscuit is playing")
var activityDescriptions = map[types.BehaviorState]string{
	types.BehaviorIdle:              "is relaxing",
	types.BehaviorSleeping:          "is sleeping",
	types.BehaviorEating:            "is looking for food",
	types.BehaviorPlaying:           "is playing",
	types.BehaviorExploring:         "is exploring",
	types.BehaviorSocialInteraction: "wants your attention",
	types.BehaviorGrooming:          "is grooming itself",
	types.BehaviorExercising:        "is exercising",
	types.BehaviorSick:              "is feeling sick",
	types.BehaviorDistressed:        "is upset",
	types.BehaviorHappy:             "is happy",
	types.BehaviorExcited:           "is excited",
	types.BehaviorHibernating:       "is hibernating",
}

// ActivityDescription describes what a pet with the given behavior is
// doing, to follow its name
func ActivityDescription(behavior types.BehaviorState) string {
	if description, ok := activityDescriptions[behavior]; ok {
		return description
	}
	return "is relaxing"
}
