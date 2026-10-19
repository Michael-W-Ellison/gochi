package core

import (
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
	if _, ok := p.bestItemLocked(CategoryBed, true); ok {
		quality *= 1.25
	}
	return quality
}

// CareDescription explains what an interaction does
func CareDescription(interactionType types.InteractionType) string {
	return interaction.GetInteractionDescription(interactionType)
}
