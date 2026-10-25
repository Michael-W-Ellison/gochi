package core

import (
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Care rules come from the interaction package: ValidateInteraction decides
// whether the pet accepts an interaction, and GetInteractionDescription
// explains each one. Its effectiveness rules are not used, because the
// pet's reactions (see reactions.go) already weigh personality, energy and
// mood.

// Sleep recovery: a sleeping pet wakes once tiredness is at or below
// wakeFatigue and energy is at least wakeEnergy
const (
	wakeFatigue = 0.2
	wakeEnergy  = 0.5
)

// interactionContextLocked describes the pet for the interaction rules.
// Caller must hold p.mu.
func (p *DigitalPet) interactionContextLocked() interaction.InteractionContext {
	v := p.Biology.Vitals
	traits := p.Personality.Traits
	ctx := interaction.InteractionContext{
		Health:          v.Health,
		Energy:          v.Energy,
		Happiness:       v.Happiness,
		Hunger:          v.Nutrition, // How full the pet is (above 0.9 is not hungry)
		Thirst:          v.Hydration,
		Cleanliness:     v.Cleanliness,
		Stress:          v.Stress,
		Fatigue:         v.Fatigue,
		Playfulness:     traits.Playfulness,
		Independence:    traits.Independence,
		Affectionate:    traits.Affectionate,
		Intelligence:    traits.Intelligence,
		CurrentBehavior: p.CurrentBehavior,
		PetAge:          p.Biology.GetAgeInDays(),
		IsSleeping:      p.CurrentBehavior == types.BehaviorSleeping,
		IsSick:          p.CurrentBehavior == types.BehaviorSick || p.Injured,
	}
	if p.Environment != nil {
		ctx.TimeOfDay = p.Environment.GetTimeOfDay()
	}
	return ctx
}

// sleepQualityLocked returns how restful the pet's sleep is: the
// environment's sleep quality, improved by a bed. Caller must hold p.mu.
func (p *DigitalPet) sleepQualityLocked() float64 {
	quality := 1.0
	if p.Environment != nil {
		quality = p.Environment.GetEffects().SleepQuality
	}
	if _, ok := p.bestInGroupLocked("bed"); ok {
		quality *= 1.25
	}
	return quality
}

// WaitsForWaking reports whether a kind of care waits until a sleeping pet
// wakes: play, training, walks, meeting other pets, petting and grooming.
// Food, water, healing and cleaning up don't.
func WaitsForWaking(interactionType types.InteractionType) bool {
	switch interactionType {
	case types.InteractionPlaying, types.InteractionTraining, types.InteractionWalking,
		types.InteractionSocialIntroduction, types.InteractionPetting, types.InteractionGrooming:
		return true
	}
	return false
}

// WakeIn returns about how many game days a sleeping pet will sleep on,
// or 0 if it is awake
func (p *DigitalPet) WakeIn() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.wakeInLocked()
}

// wakeInLocked estimates how long the pet sleeps on: until its tiredness
// falls to wakeFatigue and its energy rises to wakeEnergy, at the rates
// rest restores them (ProcessRest). Caller must hold p.mu.
func (p *DigitalPet) wakeInLocked() float64 {
	if p.CurrentBehavior != types.BehaviorSleeping {
		return 0
	}
	p.Biology.EnsurePhysiology()
	quality := max(p.sleepQualityLocked()*p.Biology.SleepQualityFactor(), 0.05)
	v := p.Biology.Vitals
	return max((v.Fatigue-wakeFatigue)/(0.5*quality), (wakeEnergy-v.Energy)/(0.15*quality), 0.05)
}

// careRefusalLocked returns why care can't be given for reasons of the
// pet's species or home, or "". Caller must hold p.mu.
func (p *DigitalPet) careRefusalLocked(interactionType types.InteractionType) string {
	switch interactionType {
	case types.InteractionWalking:
		if def := p.defLocked(); !def.Walks {
			return withName(def.NoWalks, p.Name)
		}
	case types.InteractionCleaning:
		if p.LivingAreaWaste < 0.05 {
			return p.defLocked().CleanRefusal
		}
	}
	return ""
}

// digestLocked has the pet eat a food; too much at once makes it throw
// up. Caller must hold p.mu.
func (p *DigitalPet) digestLocked(food biology.FoodItem) {
	if p.Biology.ProcessDigestion(food) {
		p.evolveFromHardshipLocked(0.1)
		p.recordHealthEventLocked("feeding", "%s ate too much and threw up. "+
			"Smaller, measured meals and fewer treats are easier on the stomach.", p.Name)
	}
}

// CareDescription explains what an interaction does
func CareDescription(interactionType types.InteractionType) string {
	return interaction.GetInteractionDescription(interactionType)
}
