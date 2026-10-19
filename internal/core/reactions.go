package core

import (
	"fmt"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// ReactionLevel is how much a pet enjoys an interaction
type ReactionLevel string

const (
	ReactionDelighted ReactionLevel = "delighted"
	ReactionHappy     ReactionLevel = "happy"
	ReactionOkay      ReactionLevel = "okay"
	ReactionReluctant ReactionLevel = "reluctant"
)

// Reaction scores (roughly -1.0 to 1.0) at which each level starts
const (
	delightedScore = 0.3
	happyScore     = 0.1
	okayScore      = -0.15
)

// reactionTexts describe each reaction; %s is the pet's name
var reactionTexts = map[types.InteractionType]map[ReactionLevel]string{
	types.InteractionFeeding: {
		ReactionDelighted: "%s gobbles up every bite!",
		ReactionHappy:     "%s eats happily.",
		ReactionOkay:      "%s nibbles at the food.",
		ReactionReluctant: "%s isn't hungry and barely eats.",
	},
	types.InteractionDrinking: {
		ReactionDelighted: "%s laps up the water eagerly!",
		ReactionHappy:     "%s takes a nice long drink.",
		ReactionOkay:      "%s takes a small sip.",
		ReactionReluctant: "%s isn't thirsty.",
	},
	types.InteractionPetting: {
		ReactionDelighted: "%s melts into your hand and purrs!",
		ReactionHappy:     "%s leans into the pets.",
		ReactionOkay:      "%s lets you pet it.",
		ReactionReluctant: "%s wriggles away. It wants some space.",
	},
	types.InteractionPlaying: {
		ReactionDelighted: "%s bounces around, overjoyed!",
		ReactionHappy:     "%s plays happily.",
		ReactionOkay:      "%s plays for a little while.",
		ReactionReluctant: "%s isn't in the mood to play.",
	},
	types.InteractionGrooming: {
		ReactionDelighted: "%s loves being brushed and looks so shiny!",
		ReactionHappy:     "%s enjoys the grooming.",
		ReactionOkay:      "%s sits still for grooming.",
		ReactionReluctant: "%s squirms through the grooming.",
	},
	types.InteractionMedicalCare: {
		ReactionDelighted: "%s feels much better!",
		ReactionHappy:     "%s takes the medicine bravely.",
		ReactionOkay:      "%s puts up with the check-up.",
		ReactionReluctant: "%s tries to hide from the check-up.",
	},
	types.InteractionRewards: {
		ReactionDelighted: "%s does a happy dance for the treat!",
		ReactionHappy:     "%s enjoys the treat.",
		ReactionOkay:      "%s eats the treat.",
		ReactionReluctant: "%s isn't interested in a treat right now.",
	},
	types.InteractionTraining: {
		ReactionDelighted: "%s is focused and eager to learn!",
		ReactionHappy:     "%s pays attention and tries hard.",
		ReactionOkay:      "%s gets a little distracted.",
		ReactionReluctant: "%s would rather do something else.",
	},
	types.InteractionEnvironmentalEnrichment: {
		ReactionDelighted: "%s had the best adventure ever!",
		ReactionHappy:     "%s enjoyed exploring.",
		ReactionOkay:      "%s had a quiet walk around.",
		ReactionReluctant: "%s is glad to be home.",
	},
	types.InteractionDiscipline: {
		ReactionDelighted: "%s looks sad and lowers its ears.",
		ReactionHappy:     "%s looks sad and lowers its ears.",
		ReactionOkay:      "%s looks sad and lowers its ears.",
		ReactionReluctant: "%s looks sad and lowers its ears.",
	},
}

// reactionLocked works out how much the pet enjoys an interaction right
// now, from what it has learned, its personality and its current state.
// It returns the reaction level and a multiplier (0.6 to 1.4) for the
// interaction's benefits. Caller must hold p.mu.
func (p *DigitalPet) reactionLocked(interactionType types.InteractionType) (ReactionLevel, float64) {
	traits := p.Personality.Traits
	v := p.Biology.Vitals

	// What the pet has learned about this kind of care
	score := 0.5 * p.Memory.GetLearnedAffinity(interactionType)

	// Personality and current needs
	switch interactionType {
	case types.InteractionFeeding:
		score += (0.5 - v.Nutrition) * 0.8
	case types.InteractionDrinking:
		score += (0.5 - v.Hydration) * 0.8
	case types.InteractionPetting:
		score += (traits.Affectionate-0.5)*0.6 - (traits.Independence-0.5)*0.4
		score += (p.Emotions.Loneliness - 0.2) * 0.8
	case types.InteractionPlaying:
		score += (traits.Playfulness - 0.5) * 0.6
		score += (v.Energy - 0.5) * 0.6
	case types.InteractionGrooming:
		score += (traits.Conscientiousness - 0.5) * 0.4
		score += (0.5 - v.Cleanliness) * 0.6
	case types.InteractionMedicalCare:
		score -= 0.1 + (traits.Neuroticism-0.5)*0.4
		score += (0.6 - v.Health) * 0.8
		if p.Injured {
			score += 0.3 // Relief at being treated
		}
	case types.InteractionRewards:
		score += 0.2
	case types.InteractionTraining:
		score += (traits.Intelligence - 0.5) * 0.6
		score += (v.Energy - 0.5) * 0.4
	case types.InteractionEnvironmentalEnrichment:
		score += (traits.Curiosity - 0.5) * 0.6
		score += (v.Energy - 0.5) * 0.4
	case types.InteractionDiscipline:
		// Discipline is never enjoyed and has no benefits to scale
		return ReactionReluctant, 1.0
	}

	level := ReactionReluctant
	switch {
	case score >= delightedScore:
		level = ReactionDelighted
	case score >= happyScore:
		level = ReactionHappy
	case score >= okayScore:
		level = ReactionOkay
	}

	return level, min(max(1.0+0.5*score, 0.6), 1.4)
}

// reactionText describes a reaction, or returns "" for interactions
// without reaction texts
func reactionText(interactionType types.InteractionType, level ReactionLevel, name string) string {
	texts, ok := reactionTexts[interactionType]
	if !ok {
		return ""
	}
	return fmt.Sprintf(texts[level], name)
}
