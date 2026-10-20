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
	types.BehaviorPlaying:           "is playing by itself",
	types.BehaviorExploring:         "is out exploring",
	types.BehaviorInvestigating:     "is sniffing around the house",
	types.BehaviorSocialInteraction: "wants your attention",
	types.BehaviorGrooming:          "is grooming itself",
	types.BehaviorExercising:        "is running around",
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

// Activity describes what the pet is doing right now, to follow its name
// ("Biscuit is out exploring"). Trips, games and hibernation take
// precedence over the pet's behavior.
func (p *DigitalPet) Activity() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch {
	case !p.Biology.IsAlive:
		return "has passed away"
	case p.Hibernating:
		return ActivityDescription(types.BehaviorHibernating)
	case p.isExploringLocked():
		return ActivityDescription(types.BehaviorExploring)
	case p.game != nil:
		return "is hiding somewhere"
	}
	return ActivityDescription(p.CurrentBehavior)
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
	{types.InteractionWalking, "walks"},
	{types.InteractionEnvironmentChange, "new things at home"},
	{types.InteractionSocialIntroduction, "meeting other pets"},
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

// toleratedExperience is how much experience of a kind of care the pet
// needs before it is said to tolerate it (neither liking nor disliking it)
const toleratedExperience = 5

// Tolerates returns the kinds of care the pet has had often enough to
// know, but neither likes nor dislikes: it puts up with them
func (p *DigitalPet) Tolerates() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	tolerated := []string{}
	for _, care := range careNames {
		pref, ok := p.Memory.Preferences[care.interaction]
		affinity := p.Memory.GetLearnedAffinity(care.interaction)
		if ok && pref.Experience >= toleratedExperience && affinity < likeThreshold && affinity > dislikeThreshold {
			tolerated = append(tolerated, care.name)
		}
	}
	return tolerated
}

// PersonalityDescription describes the pet's personality using its name
func (p *DigitalPet) PersonalityDescription() string {
	return strings.Replace(p.Personality.GetPersonalityDescription(), "This pet", p.Name, 1)
}

// PetRequest is something the pet is asking its owner for
type PetRequest struct {
	Action  string `json:"action"` // Page action that fulfils it: feed, drink, groom, pet, play, medic, train, explore, clean, walk or revive
	Icon    string `json:"icon"`
	Message string `json:"message"`
}

const (
	// needRequestLevel is the level below which the pet asks for food,
	// water or grooming
	needRequestLevel = 0.35

	// lonelyAfter is how long (in game days of the pet's age) without any
	// interaction before the pet asks for attention
	lonelyAfter = 3.0
)

// CurrentRequest returns what the pet is asking for, most urgent first, or
// nil if it is content
func (p *DigitalPet) CurrentRequest() *PetRequest {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Biology.IsAlive {
		return nil
	}
	if p.Hibernating {
		return &PetRequest{"revive", "💤", p.Name + " is hibernating and needs you to revive it."}
	}
	if p.isExploringLocked() {
		return nil
	}
	if p.Injured {
		return &PetRequest{"medic", "🩹", p.Name + " is hurt and needs to be healed."}
	}
	if p.Illness != nil {
		return &PetRequest{"medic", "🤒", p.Name + " is ill and needs medical care."}
	}
	p.Biology.EnsurePhysiology()
	if ph := p.Biology.Physiology; p.speciesLocked() == SpeciesDog && max(ph.Bowel, ph.Bladder) >= 0.5 {
		return &PetRequest{"walk", "🦮", p.Name + " is waiting by the door and needs to go outside!"}
	}

	v := p.Biology.Vitals
	switch {
	case v.Hydration < needRequestLevel:
		return &PetRequest{"drink", "💧", p.Name + " is thirsty!"}
	case v.Nutrition < needRequestLevel:
		return &PetRequest{"feed", "🍖", p.Name + " is hungry!"}
	}

	if p.LivingAreaWaste >= 0.5 {
		if p.speciesLocked() == SpeciesDog {
			return &PetRequest{"clean", "🧹", "There's a mess in the house that needs cleaning up."}
		}
		return &PetRequest{"clean", "🧹", p.Name + "'s litter box needs cleaning."}
	}

	// A sleeping pet only wakes for food and water
	if p.CurrentBehavior == types.BehaviorSleeping {
		return nil
	}

	if v.Cleanliness < needRequestLevel {
		return &PetRequest{"groom", "🧼", p.Name + " feels messy and wants to be groomed."}
	}

	alone := p.Biology.GetAgeInDays()-p.LastInteractionAge > lonelyAfter
	if alone || p.Emotions.Loneliness > 0.6 ||
		p.needWarningLocked(types.NeedAffection) || p.needWarningLocked(types.NeedSocial) {
		// Ask for whichever kind of attention the pet expects to enjoy more
		// in its current mood
		if v.Energy > 0.4 && p.expectedEnjoymentLocked(types.InteractionPlaying) > p.expectedEnjoymentLocked(types.InteractionPetting) {
			return &PetRequest{"play", "🎾", p.Name + " misses you and wants to play!"}
		}
		return &PetRequest{"pet", "💕", p.Name + " misses you and wants to be petted."}
	}

	// Needs the pet's stats do not show
	if p.needWarningLocked(types.NeedExercise) && v.Energy > 0.4 {
		return &PetRequest{"play", "🎾", p.Name + " has energy to burn and wants to play!"}
	}
	if p.needWarningLocked(types.NeedMentalStimulation) && v.Energy > minTrainingEnergy {
		return &PetRequest{"train", "🎓", p.Name + " is bored and wants to learn something new."}
	}
	// Pets ask to go exploring only when it's good to be outdoors
	if p.needWarningLocked(types.NeedExploration) && !p.Injured && p.outdoorSuitabilityLocked() >= goodOutdoorLevel &&
		v.Energy >= minExploreLevel && v.Hydration >= minExploreLevel && v.Nutrition >= minExploreLevel {
		return &PetRequest{"explore", "🧭", p.Name + " is curious about the world and wants to go exploring!"}
	}

	if v.Happiness < 0.45 && v.Energy > 0.5 {
		return &PetRequest{"play", "🎾", p.Name + " is bored and wants to play!"}
	}
	return nil
}
