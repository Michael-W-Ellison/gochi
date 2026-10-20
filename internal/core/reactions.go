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
	types.InteractionWalking: {
		ReactionDelighted: "%s pulls happily on the leash, sniffing everything!",
		ReactionHappy:     "%s trots along beside you.",
		ReactionOkay:      "%s takes a short stroll.",
		ReactionReluctant: "%s drags its paws and wants to go home.",
	},
	types.InteractionEnvironmentChange: {
		ReactionDelighted: "%s rushes over to investigate!",
		ReactionHappy:     "%s explores the change with interest.",
		ReactionOkay:      "%s gives it a quick sniff.",
		ReactionReluctant: "%s eyes the change warily.",
	},
	types.InteractionSocialIntroduction: {
		ReactionDelighted: "%s is thrilled to have company!",
		ReactionHappy:     "%s seems glad of the company.",
		ReactionOkay:      "%s watches carefully.",
		ReactionReluctant: "%s wants to get away.",
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
	if interactionType == types.InteractionDiscipline {
		// Discipline is never enjoyed and has no benefits to scale
		return ReactionReluctant, 1.0
	}

	// What the pet has learned about this kind of care, plus how it takes
	// it by nature and in its current state
	score := 0.5*p.Memory.GetLearnedAffinity(interactionType) + p.innateReactionLocked(interactionType)

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

// overstimulation is how much less welcome each recent petting or grooming
// makes the next one
const overstimulation = 0.08

// innateReactionLocked is how the pet takes an interaction apart from what
// it has learned: from its personality, its species and its current state
// (roughly -1.0 to 1.0). Caller must hold p.mu.
func (p *DigitalPet) innateReactionLocked(interactionType types.InteractionType) float64 {
	traits := p.Personality.Traits
	v := p.Biology.Vitals
	score := p.choiceReactionLocked() // The food, toy, spot or method chosen

	switch interactionType {
	case types.InteractionFeeding:
		score += (0.5 - v.Nutrition) * 0.8
	case types.InteractionDrinking:
		score += (0.5 - v.Hydration) * 0.8
	case types.InteractionPetting:
		score += p.attentionLikingLocked()
		score += (p.Emotions.Loneliness - 0.2) * 0.8
		score -= overstimulation * p.recentCareLocked(types.InteractionPetting)
	case types.InteractionPlaying:
		score += (traits.Playfulness - 0.5) * 0.6
		score += (v.Energy - 0.5) * 0.6
		score += p.defLocked().Liking[interactionType]
	case types.InteractionGrooming:
		score += (traits.Conscientiousness - 0.5) * 0.4
		score += (0.5 - v.Cleanliness) * 0.6
		score += 0.5 * p.attentionLikingLocked() // Being handled
		score -= overstimulation * p.recentCareLocked(types.InteractionGrooming)
	case types.InteractionMedicalCare:
		score -= 0.1 + (traits.Neuroticism-0.5)*0.4
		score += (0.6 - v.Health) * 0.8
		if p.Injured || p.Illness != nil {
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
	case types.InteractionWalking:
		score += (traits.EnergyLevel - 0.5) * 0.6
		score += (v.Energy - 0.5) * 0.4
		ph := p.Biology.Physiology
		score += max(ph.Bowel, ph.Bladder) * 0.4 // Relief at going outside
		if suitability := p.outdoorSuitabilityLocked(); suitability < goodOutdoorLevel {
			score -= (goodOutdoorLevel - suitability) * 0.5 // Rain, storms and darkness spoil a walk
		}
		score += p.defLocked().Liking[interactionType]
	case types.InteractionDiscipline:
		score -= 0.5
	}
	return score
}

// attentionLikingLocked is how much the pet enjoys being handled and given
// attention by nature, when it isn't especially lonely (roughly -0.6 to
// 0.5): affectionate, outgoing pets love it; most tolerate it; independent
// and nervous ones would rather be left alone. Caller must hold p.mu.
func (p *DigitalPet) attentionLikingLocked() float64 {
	traits := p.Personality.Traits
	return (traits.Affectionate-0.5)*0.6 - (traits.Independence-0.5)*0.4 +
		(traits.Extraversion-0.5)*0.2 - (traits.Neuroticism-0.5)*0.2 - 0.05 +
		p.defLocked().Liking[types.InteractionPetting]
}

// experiencedValence is how the pet took an interaction (-1.0 to 1.0),
// from its innate reaction: delighted is 1.0, merely putting up with it
// is near 0, and reluctance is negative. Learned preferences come from
// this, so they follow the pet's nature and when care is given rather
// than reinforcing themselves.
func experiencedValence(innate float64) float64 {
	return max(-1, min(1, innate/delightedScore))
}

// Attention styles, from how much a pet enjoys attention by nature
const (
	AttentionLoves     = "loves attention"
	AttentionTolerates = "tolerates attention, in moderation"
	AttentionAvoids    = "prefers its own space"
)

// AttentionStyle describes how much the pet enjoys attention by nature
func (p *DigitalPet) AttentionStyle() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	// The same levels as a happy and a reluctant reaction to petting
	switch liking := p.attentionLikingLocked(); {
	case liking >= happyScore:
		return AttentionLoves
	case liking >= okayScore:
		return AttentionTolerates
	}
	return AttentionAvoids
}

// attentionCare is care that is attention to the pet itself, which an
// unwilling pet finds stressful
var attentionCare = map[types.InteractionType]bool{
	types.InteractionPetting:  true,
	types.InteractionGrooming: true,
	types.InteractionPlaying:  true,
	types.InteractionTraining: true,
	types.InteractionWalking:  true,
}

// reactionText describes a species' reaction, or returns "" for
// interactions without reaction texts
func reactionText(species Species, interactionType types.InteractionType, level ReactionLevel, name string) string {
	if text, ok := species.Def().ReactionTexts[interactionType][level]; ok {
		return fmt.Sprintf(text, name)
	}
	texts, ok := reactionTexts[interactionType]
	if !ok {
		return ""
	}
	return fmt.Sprintf(texts[level], name)
}
