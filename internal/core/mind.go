package core

import (
	"math"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// A pet's mind: a behavior state machine chooses what it does, it learns
// which of its behaviors its owner rewards (reinforcement learning), and
// its personality slowly changes with its experiences.

// anyState is the state machine's wildcard: a transition from any state
const anyState = types.BehaviorState(-1)

// behaviorMachineLocked returns the pet's behavior state machine, built on
// first use. Its transitions are checked in priority order; the first
// whose condition holds sets the pet's behavior. Conditions read the pet's
// state and run with p.mu held. Caller must hold p.mu.
func (p *DigitalPet) behaviorMachineLocked() *ai.BehaviorStateMachine {
	if p.behavior != nil {
		return p.behavior
	}

	m := ai.NewBehaviorStateMachine()
	m.ForceState(p.CurrentBehavior)
	add := func(priority int, to types.BehaviorState, condition func() bool) {
		m.AddTransition(ai.StateTransition{FromState: anyState, ToState: to, Condition: condition,
			Probability: 1.0, Priority: priority})
	}
	v := func() *biology.VitalStats { return p.Biology.Vitals }
	positiveMood := func() bool { return p.Emotions.GetMoodScore() > 0.6 && v().Energy > 0.5 }

	add(90, types.BehaviorExploring, p.isExploringLocked)
	// Critical needs and illness override other behaviors
	add(80, types.BehaviorSick, func() bool {
		return v().Health < 0.3 || (p.Illness != nil && p.Illness.Severity >= illnessSicknessLevel)
	})
	add(70, types.BehaviorDistressed, func() bool { return v().GetOverallWellbeing() < 0.3 })
	// A pet falls asleep when tired and sleeps until rested
	add(60, types.BehaviorSleeping, func() bool {
		asleep := p.CurrentBehavior == types.BehaviorSleeping
		return v().Fatigue > 0.7 || v().Energy < 0.3 ||
			(asleep && (v().Fatigue > wakeFatigue || v().Energy < wakeEnergy))
	})
	// Emotions and personality decide the rest
	add(50, types.BehaviorExcited, func() bool { return positiveMood() && p.Emotions.Excitement > 0.7 })
	add(49, types.BehaviorHappy, positiveMood)
	add(40, types.BehaviorEating, func() bool { return v().Nutrition < 0.3 })
	for _, b := range discretionaryBehaviors {
		behavior := b
		add(30, behavior, func() bool { return v().Energy > 0.4 && p.chooseDiscretionaryBehavior() == behavior })
	}
	add(0, types.BehaviorIdle, func() bool { return true })

	p.behavior = m
	return m
}

// discretionaryBehaviors are what a content pet with energy chooses to do
// from its personality and what it has learned
var discretionaryBehaviors = []types.BehaviorState{
	types.BehaviorPlaying,
	types.BehaviorSocialInteraction,
	types.BehaviorInvestigating,
}

// learningContextLocked is the situation the pet learns in, from its mood:
// excited, happy, neutral or upset. What its owner rewards can differ
// between them. Caller must hold p.mu.
func (p *DigitalPet) learningContextLocked() types.BehaviorState {
	mood := p.Emotions.GetMoodScore()
	switch {
	case mood > 0.6 && p.Emotions.Excitement > 0.7:
		return types.BehaviorExcited
	case mood > 0.6:
		return types.BehaviorHappy
	case mood < 0.35:
		return types.BehaviorDistressed
	}
	return types.BehaviorIdle
}

// learnedRewardLocked is how rewarding the pet has learned a behavior is
// in its current situation, -1.0 to 1.0. Caller must hold p.mu.
func (p *DigitalPet) learnedRewardLocked(behavior types.BehaviorState) float64 {
	if p.Learning == nil {
		return 0
	}
	return math.Max(-1, math.Min(1, p.Learning.GetQValue(p.learningContextLocked(), behavior)))
}

// learnFromResponseLocked teaches the pet how its owner responded to what
// it was doing: care it enjoyed while seeking attention rewards seeking
// attention, and scolding while playing discourages play. Caller must
// hold p.mu.
func (p *DigitalPet) learnFromResponseLocked(behavior, context types.BehaviorState, reward float64) {
	isDiscretionary := false
	for _, b := range discretionaryBehaviors {
		isDiscretionary = isDiscretionary || b == behavior
	}
	if !isDiscretionary {
		return
	}
	if p.Learning == nil {
		p.Learning = ai.NewLearningSystem()
	}
	p.Learning.LearnFromInteraction(context, behavior, reward, p.learningContextLocked())
}

// Personality change. Each experience moves a trait by its intensity times
// the personality's evolution rate (0.01), so traits change over many
// experiences rather than at once.

// careExperiences are the traits each kind of enjoyed care develops, with
// their intensity per unit of enjoyment
var careExperiences = map[types.InteractionType][]ai.ExperienceData{
	types.InteractionPlaying: {{TraitAffected: "playfulness", Intensity: 0.2},
		{TraitAffected: "energy_level", Intensity: 0.1}},
	types.InteractionWalking: {{TraitAffected: "energy_level", Intensity: 0.15},
		{TraitAffected: "playfulness", Intensity: 0.05}},
	types.InteractionPetting: {{TraitAffected: "affectionate", Intensity: 0.2},
		{TraitAffected: "loyalty", Intensity: 0.1}},
	types.InteractionGrooming: {{TraitAffected: "affectionate", Intensity: 0.05}},
	types.InteractionTraining: {{TraitAffected: "conscientiousness", Intensity: 0.2},
		{TraitAffected: "intelligence", Intensity: 0.05}},
	types.InteractionEnvironmentalEnrichment: {{TraitAffected: "curiosity", Intensity: 0.2},
		{TraitAffected: "openness", Intensity: 0.1}},
}

// evolveLocked applies experiences to the pet's personality. Caller must
// hold p.mu.
func (p *DigitalPet) evolveLocked(experiences ...ai.ExperienceData) {
	p.ensureTraitBaselineLocked()
	p.Personality.EvolveTraits(experiences)
}

// evolveFromCareLocked changes the pet's personality from care it was
// given: enjoyed care develops matching traits and, given consistently,
// makes the pet calmer and more attached; scolding and unwelcome care
// make it more anxious. weight (0.0 to 1.0) scales the change, lower for
// care repeated in quick succession. Caller must hold p.mu.
func (p *DigitalPet) evolveFromCareLocked(interactionType types.InteractionType, valence, weight float64) {
	if interactionType == types.InteractionDiscipline {
		p.evolveLocked(ai.ExperienceData{TraitAffected: "neuroticism", Intensity: 0.3 * weight},
			ai.ExperienceData{TraitAffected: "agreeableness", Intensity: -0.2 * weight})
		return
	}
	if valence <= 0 {
		p.evolveLocked(ai.ExperienceData{TraitAffected: "neuroticism", Intensity: 0.05 * weight})
		return
	}

	var experiences []ai.ExperienceData
	for _, e := range careExperiences[interactionType] {
		e.Intensity *= valence * weight
		experiences = append(experiences, e)
	}
	experiences = append(experiences,
		ai.ExperienceData{TraitAffected: "neuroticism", Intensity: -0.05 * valence * weight},
		ai.ExperienceData{TraitAffected: "loyalty", Intensity: 0.05 * valence * weight})
	p.evolveLocked(experiences...)
}

// Repeated care. The same kind of care given again and again in quick
// succession matters less each time to how the pet learns and grows: the
// nth repeat counts 1/n as much. Recent repeats fade with a half-life of
// repeatHalfLife (real time, like reward cooldowns), so care spread out
// over time keeps its full effect. The care's other benefits (food,
// water, happiness) are not reduced.

const (
	// repeatHalfLife is how long it takes a run of repeated care to count
	// half as much
	repeatHalfLife = 3 * time.Minute

	// diminishedWeight is the weight below which the player is told that
	// more of the same care won't change the pet just now
	diminishedWeight = 0.5
)

// RecentCare counts how much of one kind of care the pet has had lately
type RecentCare struct {
	Count float64   `json:"count"` // Recent repeats, fading over time
	At    time.Time `json:"at"`    // When it was last counted
}

// recentCareLocked is how many times the pet has had a kind of care
// lately, with older care fading over repeatHalfLife. Caller must hold
// p.mu.
func (p *DigitalPet) recentCareLocked(interactionType types.InteractionType) float64 {
	recent := p.RecentCare[interactionType]
	if recent == nil {
		return 0
	}
	if elapsed := now().Sub(recent.At); elapsed > 0 {
		return recent.Count * math.Pow(0.5, elapsed.Seconds()/repeatHalfLife.Seconds())
	}
	return recent.Count
}

// repeatWeightLocked returns how much a kind of care counts towards
// learning and personality now (1.0 the first time, less when repeated
// in quick succession) and records it. Caller must hold p.mu.
func (p *DigitalPet) repeatWeightLocked(interactionType types.InteractionType) float64 {
	count := p.recentCareLocked(interactionType)
	if p.RecentCare == nil {
		p.RecentCare = make(map[types.InteractionType]*RecentCare)
	}
	p.RecentCare[interactionType] = &RecentCare{Count: count + 1, At: now()}
	return 1 / (1 + count)
}

// evolveFromHardshipLocked makes the pet more anxious after a frightening
// or painful event (an injury, illness or accident); intensity is how bad
// it was. Caller must hold p.mu.
func (p *DigitalPet) evolveFromHardshipLocked(intensity float64) {
	p.evolveLocked(ai.ExperienceData{TraitAffected: "neuroticism", Intensity: intensity})
}

// evolveFromNeglectLocked changes a lonely or neglected pet over deltaTime
// game days: it becomes more independent and anxious, and less attached.
// Caller must hold p.mu.
func (p *DigitalPet) evolveFromNeglectLocked(deltaTime float64) {
	v := p.Biology.Vitals
	neglected := p.Emotions.Loneliness > 0.6 || v.Nutrition < 0.2 || v.Hydration < 0.2 ||
		p.LivingAreaWaste > illnessAreaLevel
	if !neglected {
		return
	}
	p.evolveLocked(
		ai.ExperienceData{TraitAffected: "independence", Intensity: 0.1 * deltaTime},
		ai.ExperienceData{TraitAffected: "loyalty", Intensity: -0.1 * deltaTime},
		ai.ExperienceData{TraitAffected: "neuroticism", Intensity: 0.05 * deltaTime})
}

// ensureTraitBaselineLocked records the pet's personality before it first
// changes, to compare against later. Pets saved before personality could
// change get their current personality as the baseline. Caller must hold
// p.mu.
func (p *DigitalPet) ensureTraitBaselineLocked() {
	if len(p.Personality.TraitHistory) == 0 {
		baseline := *p.Personality.Traits
		p.Personality.TraitHistory = []ai.TraitSnapshot{{Timestamp: p.Biology.GetAgeInDays(), Traits: &baseline}}
	}
}

// personalityChangeLevel is how far a trait must move from the baseline
// before the change is described
const personalityChangeLevel = 0.05

// traitChanges describe each trait that can change, as it rises and falls
var traitChanges = []struct {
	name       string
	get        func(*ai.Traits) float64
	more, less string
}{
	{"playfulness", func(t *ai.Traits) float64 { return t.Playfulness }, "more playful", "less playful"},
	{"affectionate", func(t *ai.Traits) float64 { return t.Affectionate }, "more affectionate", "less affectionate"},
	{"loyalty", func(t *ai.Traits) float64 { return t.Loyalty }, "more attached to you", "less attached to you"},
	{"neuroticism", func(t *ai.Traits) float64 { return t.Neuroticism }, "more anxious", "calmer"},
	{"independence", func(t *ai.Traits) float64 { return t.Independence }, "more independent", "more dependent on you"},
	{"curiosity", func(t *ai.Traits) float64 { return t.Curiosity }, "more curious", "more cautious"},
	{"openness", func(t *ai.Traits) float64 { return t.Openness }, "more open to new things", "more set in its ways"},
	{"energy_level", func(t *ai.Traits) float64 { return t.EnergyLevel }, "more active", "less active"},
	{"conscientiousness", func(t *ai.Traits) float64 { return t.Conscientiousness }, "better behaved", "less well behaved"},
	{"agreeableness", func(t *ai.Traits) float64 { return t.Agreeableness }, "friendlier", "grumpier"},
	{"intelligence", func(t *ai.Traits) float64 { return t.Intelligence }, "quicker to learn", "slower to learn"},
}

// PersonalityChanges describes how the pet's personality has changed since
// it was adopted (or since personality could change), largest change
// first, such as "more playful" or "calmer"
func (p *DigitalPet) PersonalityChanges() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	changes := []string{}
	if len(p.Personality.TraitHistory) == 0 {
		return changes
	}
	baseline, now := p.Personality.TraitHistory[0].Traits, p.Personality.Traits

	type change struct {
		text string
		size float64
	}
	var found []change
	for _, t := range traitChanges {
		diff := t.get(now) - t.get(baseline)
		switch {
		case diff >= personalityChangeLevel:
			found = append(found, change{t.more, diff})
		case diff <= -personalityChangeLevel:
			found = append(found, change{t.less, -diff})
		}
	}
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && found[j].size > found[j-1].size; j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	for _, c := range found {
		changes = append(changes, c.text)
	}
	return changes
}
