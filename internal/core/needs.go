package core

import (
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// The pet's needs come from the simulation package's NeedsManager. Needs
// that the biology already tracks as vital stats (hunger, thirst, sleep,
// cleanliness, medical care) are copied from the stats so they are not
// tracked twice. The other needs decay over time and are met by matching
// care and activities.

// psychologicalNeeds are the needs not covered by vital stats
var psychologicalNeeds = []types.NeedType{
	types.NeedExercise,
	types.NeedSocial,
	types.NeedMentalStimulation,
	types.NeedAffection,
	types.NeedExploration,
}

// needsMet lists how much each kind of interaction meets each need (scaled
// by the interaction's intensity and the need's satisfaction rate)
var needsMet = map[types.InteractionType]map[types.NeedType]float64{
	types.InteractionPlaying:                 {types.NeedExercise: 1.0, types.NeedMentalStimulation: 0.5},
	types.InteractionPetting:                 {types.NeedAffection: 1.0, types.NeedSocial: 0.5},
	types.InteractionGrooming:                {types.NeedAffection: 0.3},
	types.InteractionTraining:                {types.NeedMentalStimulation: 1.0, types.NeedSocial: 0.3},
	types.InteractionEnvironmentalEnrichment: {types.NeedExploration: 2.0, types.NeedExercise: 1.0},
	types.InteractionRewards:                 {types.NeedAffection: 0.3},
	types.InteractionWalking:                 {types.NeedExercise: 1.0, types.NeedExploration: 0.5, types.NeedSocial: 0.3},
	types.InteractionSocialIntroduction:      {types.NeedSocial: 0.5}, // More when the meeting goes well (social.go)
}

// unmetNeedUnhappiness is how much happiness each critically unmet
// psychological need costs per game day
const unmetNeedUnhappiness = 0.02

// NeedLevel is one need's current level for display
type NeedLevel struct {
	Need  string  `json:"need"`
	Level float64 `json:"level"` // 0.0 (urgent) to 1.0 (met)
}

// ensureNeedsLocked creates the needs for pets saved before needs were
// tracked. Caller must hold p.mu.
func (p *DigitalPet) ensureNeedsLocked() {
	if p.Needs == nil {
		p.Needs = simulation.NewNeedsManager()
	}
}

// syncPhysicalNeedsLocked copies the needs tracked by vital stats into the
// needs manager. Caller must hold p.mu.
func (p *DigitalPet) syncPhysicalNeedsLocked() {
	v := p.Biology.Vitals
	medical := v.Health
	if p.Injured {
		medical = min(medical, 0.3)
	}
	p.Needs.SetNeedLevel(types.NeedHunger, v.Nutrition)
	p.Needs.SetNeedLevel(types.NeedThirst, v.Hydration)
	p.Needs.SetNeedLevel(types.NeedSleep, 1.0-v.Fatigue)
	p.Needs.SetNeedLevel(types.NeedCleanliness, v.Cleanliness)
	p.Needs.SetNeedLevel(types.NeedMedicalCare, medical)
}

// updateNeedsLocked decays needs over deltaTime game days; critically
// unmet psychological needs make the pet unhappier. Caller must hold p.mu.
func (p *DigitalPet) updateNeedsLocked(deltaTime float64) {
	p.ensureNeedsLocked()
	p.Needs.Update(deltaTime)
	p.syncPhysicalNeedsLocked()

	for _, need := range psychologicalNeeds {
		if p.Needs.GetNeed(need).IsCritical() {
			p.Biology.Vitals.Happiness -= unmetNeedUnhappiness * deltaTime
		}
	}
	p.Biology.Vitals.Clamp()
}

// meetNeedsLocked records the needs an interaction met. Caller must hold
// p.mu.
func (p *DigitalPet) meetNeedsLocked(interactionType types.InteractionType, intensity float64) {
	p.ensureNeedsLocked()
	for need, amount := range needsMet[interactionType] {
		p.Needs.SatisfyNeed(need, amount*intensity, p.Biology.GetAgeInDays())
	}
	p.syncPhysicalNeedsLocked()
}

// needLevelLocked returns a need's level. Caller must hold p.mu.
func (p *DigitalPet) needLevelLocked(need types.NeedType) float64 {
	p.ensureNeedsLocked()
	return p.Needs.GetNeedLevel(need)
}

// needWarningLocked reports whether a need is low enough to ask for.
// Caller must hold p.mu.
func (p *DigitalPet) needWarningLocked(need types.NeedType) bool {
	p.ensureNeedsLocked()
	return p.Needs.GetNeed(need).IsWarning()
}

// PsychologicalNeeds returns the levels of the needs that are not shown as
// vital stats, in display order
func (p *DigitalPet) PsychologicalNeeds() []NeedLevel {
	p.mu.Lock()
	defer p.mu.Unlock()

	levels := make([]NeedLevel, len(psychologicalNeeds))
	for i, need := range psychologicalNeeds {
		levels[i] = NeedLevel{Need: need.String(), Level: p.needLevelLocked(need)}
	}
	return levels
}

// metersPerWalkSession is how far the player must walk with the pet (for
// example, tracked by AR) to meet its exercise need as much as one play
// session does
const metersPerWalkSession = 500.0

// maxWalkSessions caps how much one walk counts
const maxWalkSessions = 2.0

// walkEnergyCost is the energy one walk session uses
const walkEnergyCost = 0.05

// Walk records a walk with the player and returns how much it counted, in
// play sessions of exercise. Returns the same errors as other activities
// if the pet can't go.
func (p *DigitalPet) Walk(meters float64) (float64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.checkActiveLocked(); err != nil {
		return 0, err
	}
	sessions := min(max(meters, 0)/metersPerWalkSession, maxWalkSessions)
	if sessions == 0 {
		return 0, nil
	}

	p.ensureNeedsLocked()
	p.Needs.SatisfyNeed(types.NeedExercise, sessions, p.Biology.GetAgeInDays())
	p.Biology.EnsurePhysiology()
	p.Biology.Exercise(sessions)
	p.Biology.Vitals.Energy = max(p.Biology.Vitals.Energy-walkEnergyCost*sessions, 0)
	p.syncPhysicalNeedsLocked()
	return sessions, nil
}
