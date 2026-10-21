package core

import (
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Care comes in variants, each with its own effect and lesson: the food
// given, the toy played with, where the pet is petted, how it is trained,
// and what is changed in its surroundings.

// careChoice is the variant of the care being given (set while an
// interaction is applied)
type careChoice struct {
	food    *supply
	toy     *supply
	region  *petRegion
	method  *trainingMethod
	change  *environmentChange
	meeting *meetingOutcome // Social introductions (social.go)
	novelty float64         // Environment changes: 1.0 new, less if done recently
}

// Choice errors
var (
	ErrUnknownChoice = errors.New("unknown choice")
	ErrWrongSpecies  = errors.New("not for this species")
	ErrNoTreats      = errors.New("no treats in the pantry")
	ErrNoClicker     = errors.New("no clicker")
)

// interactWithLocked applies care with a chosen variant. Caller must hold
// p.mu.
func (p *DigitalPet) interactWithLocked(interactionType types.InteractionType, choice careChoice) InteractionResult {
	p.choice = &choice
	defer func() { p.choice = nil }()
	return p.interactLocked(interactionType, 1.0)
}

// Feed gives the pet a food from its pantry (its basic food is always
// there). Unsuitable foods have consequences.
func (p *DigitalPet) Feed(foodID string) (InteractionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	s, err := p.pantrySupplyLocked(foodID, SupplyFood)
	if err != nil {
		return InteractionResult{}, err
	}
	result := p.interactWithLocked(types.InteractionFeeding, careChoice{food: &s})
	if result.Applied {
		p.useSupplyLocked(foodID)
	}
	return result, nil
}

// GiveTreat gives the pet a treat from its pantry
func (p *DigitalPet) GiveTreat(treatID string) (InteractionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	s, err := p.pantrySupplyLocked(treatID, SupplyTreat)
	if err != nil {
		return InteractionResult{}, err
	}
	result := p.interactWithLocked(types.InteractionRewards, careChoice{food: &s})
	if result.Applied {
		p.useSupplyLocked(treatID)
	}
	return result, nil
}

// PlayWith plays with the pet using a toy from its pantry; "" plays
// together without a toy
func (p *DigitalPet) PlayWith(toyID string) (InteractionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	choice := careChoice{}
	if toyID != "" {
		s, err := p.pantrySupplyLocked(toyID, SupplyToy)
		if err != nil {
			return InteractionResult{}, err
		}
		choice.toy = &s
	}
	result := p.interactWithLocked(types.InteractionPlaying, choice)
	if result.Applied && choice.toy != nil && p.wearToyLocked(toyID) {
		result.Note = fmt.Sprintf("The %s is worn out.", lowerFirst(choice.toy.item.Name))
	}
	return result, nil
}

// PetOn pets the pet on a region of its body ("" anywhere)
func (p *DigitalPet) PetOn(regionID string) (InteractionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	choice := careChoice{}
	if regionID != "" {
		r, ok := findRegion(p.speciesLocked(), regionID)
		if !ok {
			return InteractionResult{}, ErrUnknownChoice
		}
		choice.region = &r
	}
	return p.interactWithLocked(types.InteractionPetting, choice), nil
}

// pantrySupplyLocked returns a supply of a kind the pet can be given now.
// Caller must hold p.mu.
func (p *DigitalPet) pantrySupplyLocked(id, kind string) (supply, error) {
	s, ok := findSupply(id)
	switch {
	case !ok || s.kind != kind:
		return supply{}, ErrUnknownChoice
	case !s.forSpecies(p.speciesLocked()):
		return supply{}, ErrWrongSpecies
	}
	if basic := p.defLocked().BasicFood; id == basic.item.ID {
		return s, nil
	}
	if p.ensureSuppliesLocked(); p.Supplies.GetItemQuantity(id) == 0 {
		return supply{}, ErrNoSupply
	}
	return s, nil
}

// lowerFirst lowercases the first letter of a name
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]|0x20) + s[1:]
}

// reactionScoreFor is the pet's reaction score to care with a variant,
// without giving it (for tests and comparisons)
func (p *DigitalPet) reactionScoreFor(interactionType types.InteractionType, choice careChoice) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.choice = &choice
	defer func() { p.choice = nil }()
	return 0.5*p.Memory.GetLearnedAffinity(interactionType) + p.innateReactionLocked(interactionType)
}

// Petting regions

// petRegion is a part of the body to pet
type petRegion struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	liking    float64 // How much most of the species enjoy it
	sensitive float64 // How strongly personality changes that (sensitive spots)
	lesson    string
}

// findRegion returns a species' petting region
func findRegion(species Species, id string) (petRegion, bool) {
	for _, r := range species.Def().PetRegions {
		if r.ID == id {
			return r, true
		}
	}
	return petRegion{}, false
}

// regionLikingLocked is how much the pet enjoys being petted on a region:
// affectionate, calm pets accept sensitive spots, nervous ones less.
// Caller must hold p.mu.
func (p *DigitalPet) regionLikingLocked(r petRegion) float64 {
	traits := p.Personality.Traits
	return r.liking + r.sensitive*((traits.Affectionate-0.5)*0.3-(traits.Neuroticism-0.3)*0.4)
}

// Training methods

// trainingMethod is a way of training
type trainingMethod struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	progress float64 // Learning speed, relative to praise
	liking   float64 // How much pets enjoy it
	lesson   string
}

// trainingMethods are the ways to train, most effective first
var trainingMethods = []trainingMethod{
	{ID: "clicker", Name: "Clicker and reward", liking: 0.15,
		lesson: "The click marks the exact moment your pet does it right, so it learns faster; always follow it with a reward."},
	{ID: "treats", Name: "Treat rewards", progress: 1.3, liking: 0.2,
		lesson: "A treat straight after the behavior tells your pet exactly what earned it; use tiny pieces."},
	{ID: "praise", Name: "Praise", progress: 1.0, liking: 0,
		lesson: "Praise motivates pets that love attention; for others, food rewards work better."},
	{ID: "punishment", Name: "Scolding", progress: 0.4, liking: -0.6,
		lesson: "Punishment makes pets fearful and slows learning: it shows what not to do, not what to do. Reward the behavior you want instead."},
}

// findMethod returns a training method
func findMethod(id string) (trainingMethod, bool) {
	for _, m := range trainingMethods {
		if m.ID == id {
			return m, true
		}
	}
	return trainingMethod{}, false
}

// methodAvailableLocked reports whether the pet can be trained with a
// method now, or why not. Caller must hold p.mu.
func (p *DigitalPet) methodAvailableLocked(m trainingMethod) error {
	p.ensureSuppliesLocked()
	treats := p.Supplies.GetItemQuantity(p.defLocked().Treats) > 0
	switch m.ID {
	case "treats":
		if !treats {
			return ErrNoTreats
		}
	case "clicker":
		if !p.ownsLocked("clicker") {
			return ErrNoClicker
		}
		if !treats {
			return ErrNoTreats
		}
	}
	return nil
}

// methodProgressLocked is how fast a method teaches. Caller must hold p.mu.
func (p *DigitalPet) methodProgressLocked(m trainingMethod) float64 {
	if m.ID == "clicker" {
		return clickerProgress
	}
	return m.progress
}

// methodLikingLocked is how much the pet enjoys a method; praise means
// more to pets that love attention. Caller must hold p.mu.
func (p *DigitalPet) methodLikingLocked(m trainingMethod) float64 {
	liking := m.liking
	if m.ID == "praise" {
		liking += p.attentionLikingLocked() * 0.5
	}
	return liking
}

// Environment changes

// environmentChange is something changed in the pet's surroundings
type environmentChange struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Icon        string  `json:"icon"`
	stimulation float64 // Mental stimulation it gives
	exploration float64 // Exploration need it meets
	exercise    float64
	calming     float64 // Stress relieved, more for nervous pets
	liking      func(t *ai.Traits) float64
	lesson      string
}

// noveltyDays is how long it takes a change to feel new again
const noveltyDays = 3.0

// findChange returns a species' environment change
func findChange(species Species, id string) (environmentChange, bool) {
	for _, c := range species.Def().EnvironmentChanges {
		if c.ID == id {
			return c, true
		}
	}
	return environmentChange{}, false
}

// ChangeEnvironment makes a change to the pet's surroundings at home. A
// change done recently feels less new.
func (p *DigitalPet) ChangeEnvironment(changeID string) (InteractionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	c, ok := findChange(p.speciesLocked(), changeID)
	if !ok {
		return InteractionResult{}, ErrUnknownChoice
	}
	age := p.Biology.GetAgeInDays()
	novelty := 1.0
	if last, ok := p.EnrichmentUsed[changeID]; ok {
		novelty = min(max(age-last, 0)/noveltyDays, 1)
	}

	result := p.interactWithLocked(types.InteractionEnvironmentChange, careChoice{change: &c, novelty: novelty})
	if result.Applied {
		if p.EnrichmentUsed == nil {
			p.EnrichmentUsed = make(map[string]float64)
		}
		p.EnrichmentUsed[changeID] = age
		if novelty < 0.5 {
			result.Note = fmt.Sprintf("%s has seen this recently; it will feel new again in a few days.", p.Name)
		}
	}
	return result, nil
}

// Effects of the choices, applied while care is given

// catnipResponderLocked reports whether the pet responds to catnip, which
// is inherited: about two cats in three do. Caller must hold p.mu.
func (p *DigitalPet) catnipResponderLocked() bool {
	h := fnv.New32a()
	h.Write([]byte(p.ID))
	return h.Sum32()%3 != 0
}

// choiceReactionLocked is how the chosen variant changes how much the pet
// enjoys care. Caller must hold p.mu.
func (p *DigitalPet) choiceReactionLocked() float64 {
	c := p.choice
	if c == nil {
		return 0
	}
	switch {
	case c.food != nil && c.food.food != nil:
		return c.food.food.taste
	case c.toy != nil:
		if c.toy.toy.catnip {
			if p.catnipResponderLocked() {
				return 0.4
			}
			return -0.1
		}
		return c.toy.toy.fit
	case c.region != nil:
		return p.regionLikingLocked(*c.region)
	case c.method != nil:
		return p.methodLikingLocked(*c.method)
	case c.change != nil:
		return c.change.liking(p.Personality.Traits) * c.novelty
	case c.meeting != nil:
		return c.meeting.reaction
	}
	return 0
}

// applyChoiceLocked applies the effects of the chosen variant, after the
// care's usual effects; boost is the care's effectiveness. It returns the
// variant's lesson. Caller must hold p.mu.
func (p *DigitalPet) applyChoiceLocked(boost float64) string {
	c := p.choice
	if c == nil {
		return ""
	}
	v, ph := p.Biology.Vitals, p.Biology.Physiology
	age := p.Biology.GetAgeInDays()
	p.ensureNeedsLocked()
	defer v.Clamp()
	defer ph.Clamp()

	switch {
	case c.food != nil && c.food.food != nil:
		f := c.food.food
		v.Hydration += f.hydration
		if f.digestion != 0 {
			upset := f.digestion
			if upset < 0 {
				upset *= p.effectLocked(effFoodUpset) // Less with a probiotic
			}
			ph.DigestiveHealth += upset
			if f.digestion < 0 {
				v.Happiness -= 0.05
				p.recordHealthEventLocked("feeding", "%s has an upset stomach after the %s.", p.Name, lowerFirst(c.food.item.Name))
			}
		}
		if f.toxic {
			p.poisonLocked()
		}
		return f.lesson

	case c.toy != nil:
		t := c.toy.toy
		p.Biology.ProcessExercise(biology.ExercisePlay, t.exercise-1) // Playing already counted one session
		p.Needs.SatisfyNeed(types.NeedExercise, t.exercise-1, age)
		p.Needs.SatisfyNeed(types.NeedMentalStimulation, t.stimulation, age)
		v.Stress -= t.calming
		if t.catnip {
			if p.catnipResponderLocked() {
				return "Catnip sends this cat into a happy frenzy for a few minutes. About two cats in three react to it; it's inherited."
			}
			return "This cat doesn't react to catnip: about one cat in three doesn't, because the response is inherited."
		}
		return t.lesson

	case c.region != nil:
		return c.region.lesson

	case c.method != nil:
		if c.method.ID == "punishment" {
			v.Stress += 0.1
			p.evolveLocked(ai.ExperienceData{TraitAffected: "neuroticism", Intensity: 0.2})
		}
		return c.method.lesson

	case c.change != nil:
		ch, n := c.change, c.novelty*p.effectLocked(effEnrichment)
		if ch.ID == "perch" || ch.ID == "scratching_post" {
			n *= p.effectLocked(effPerch)
		}
		p.Needs.SatisfyNeed(types.NeedMentalStimulation, ch.stimulation*n, age)
		p.Needs.SatisfyNeed(types.NeedExploration, ch.exploration*n, age)
		p.Needs.SatisfyNeed(types.NeedExercise, ch.exercise*n, age)
		p.Biology.ProcessExercise(biology.ExerciseRun, ch.exercise*n)
		v.Stress -= ch.calming * n * (0.5 + p.Personality.Traits.Neuroticism)
		v.Happiness += 0.08 * n * boost
		return ch.lesson

	case c.meeting != nil:
		return c.meeting.lesson
	}
	return ""
}

// CareOptions lists the variants of care the pet can be given now
type CareOptions struct {
	Foods   []PantryItem        `json:"foods"`
	Treats  []PantryItem        `json:"treats"`
	Toys    []PantryItem        `json:"toys"`
	Regions []petRegion         `json:"regions"`
	Methods []MethodOption      `json:"methods"`
	Changes []environmentChange `json:"changes"`
}

// MethodOption is a training method and whether it can be used now
type MethodOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Unavailable string `json:"unavailable,omitempty"` // Why it can't be used now
}

// GetCareOptions returns the variants of care the pet can be given now
func (p *DigitalPet) GetCareOptions() CareOptions {
	pantry := p.Pantry()

	p.mu.Lock()
	defer p.mu.Unlock()
	species := p.speciesLocked()
	options := CareOptions{Foods: []PantryItem{}, Treats: []PantryItem{}, Toys: []PantryItem{},
		Regions: species.Def().PetRegions, Changes: species.Def().EnvironmentChanges}
	for _, item := range pantry {
		switch item.Kind {
		case SupplyFood:
			options.Foods = append(options.Foods, item)
		case SupplyTreat:
			options.Treats = append(options.Treats, item)
		case SupplyToy:
			options.Toys = append(options.Toys, item)
		}
	}
	for _, m := range trainingMethods {
		option := MethodOption{ID: m.ID, Name: m.Name}
		switch err := p.methodAvailableLocked(m); {
		case errors.Is(err, ErrNoTreats):
			option.Unavailable = "Needs treats from the shop"
		case errors.Is(err, ErrNoClicker):
			option.Unavailable = "Needs a clicker from the shop"
		}
		options.Methods = append(options.Methods, option)
	}
	return options
}
