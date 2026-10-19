package core

import (
	"math"
	"sort"
	"strings"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

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

// careNames name each kind of care as something a pet can like or dislike
var careNames = []struct {
	interaction types.InteractionType
	name        string
}{
	{types.InteractionFeeding, "mealtime"},
	{types.InteractionDrinking, "fresh water"},
	{types.InteractionPetting, "being petted"},
	{types.InteractionPlaying, "playing"},
	{types.InteractionGrooming, "being groomed"},
	{types.InteractionTraining, "training"},
	{types.InteractionEnvironmentalEnrichment, "exploring"},
	{types.InteractionMedicalCare, "check-ups"},
	{types.InteractionRewards, "treats"},
	{types.InteractionDiscipline, "being scolded"},
}

// Learned affinity (weighted by confidence, -1.0 to 1.0) at which a pet
// shows that it likes or dislikes something
const (
	likeThreshold    = 0.25
	dislikeThreshold = -0.15
)

// LikesAndDislikes returns what the pet has learned it likes and dislikes,
// strongest first
func (p *DigitalPet) LikesAndDislikes() (likes, dislikes []string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	type rated struct {
		name     string
		affinity float64
	}
	var liked, disliked []rated
	for _, care := range careNames {
		affinity := p.Memory.GetLearnedAffinity(care.interaction)
		switch {
		case affinity >= likeThreshold:
			liked = append(liked, rated{care.name, affinity})
		case affinity <= dislikeThreshold:
			disliked = append(disliked, rated{care.name, affinity})
		}
	}

	strongestFirst := func(r []rated) []string {
		sort.SliceStable(r, func(i, j int) bool { return math.Abs(r[i].affinity) > math.Abs(r[j].affinity) })
		names := make([]string, len(r))
		for i, x := range r {
			names[i] = x.name
		}
		return names
	}
	return strongestFirst(liked), strongestFirst(disliked)
}

// PersonalityDescription describes the pet's personality using its name
func (p *DigitalPet) PersonalityDescription() string {
	return strings.Replace(p.Personality.GetPersonalityDescription(), "This pet", p.Name, 1)
}
