package core

import (
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Household time: while the game runs, the saved pets live on with the
// active one. They age at the same rate, and the household looks after
// them, keeping them fed, watered, rested, clean and content, so they
// don't fall ill from neglect and don't earn points. Pets that have met
// grow a little closer, living together. Their upkeep runs, since it is
// timed by age, and they reach the end of their lives on schedule. The
// player can keep a pet paused, as it is, instead.

const (
	// householdLevel is the least the household keeps a pet's food,
	// water, energy, cleanliness and needs at
	householdLevel = 0.8

	// householdBondGrowth is how much closer pets that have met grow each
	// game day living together, up to householdBondLimit
	householdBondGrowth = 0.002
	householdBondLimit  = 0.8
)

// HouseholdNews is what happened to a saved pet while the household
// looked after it
type HouseholdNews struct {
	Lived        bool // It lived on (it isn't paused, hibernating, away or dead)
	GrewUp       bool
	BecameSenior bool
	Died         bool
}

// IsHouseholdPaused reports whether the player keeps the pet as it is
// while it isn't the active pet
func (p *DigitalPet) IsHouseholdPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.HouseholdPaused
}

// SetHouseholdPaused keeps the pet as it is while it isn't the active pet,
// or lets it live on with the household
func (p *DigitalPet) SetHouseholdPaused(paused bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.HouseholdPaused = paused
}

// householdActiveLocked reports whether the pet lives on with the
// household. Caller must hold p.mu.
func (p *DigitalPet) householdActiveLocked() bool {
	return p.Biology.IsAlive && !p.Hibernating && !p.HouseholdPaused && !p.isExploringLocked()
}

// LiveInHousehold lets a saved pet live days game days in the household's
// care
func (p *DigitalPet) LiveInHousehold(days float64) HouseholdNews {
	p.mu.Lock()
	defer p.mu.Unlock()
	var news HouseholdNews
	b := p.Biology
	if !(days > 0) || !p.householdActiveLocked() {
		return news
	}
	news.Lived = true
	stage := b.LifeStage()
	p.prepareBodyLocked()
	b.Processes.Age += days

	// The household keeps it well looked after
	v := b.Vitals
	v.Nutrition = max(v.Nutrition, householdLevel)
	v.Hydration = max(v.Hydration, householdLevel)
	v.Energy = max(v.Energy, householdLevel)
	v.Cleanliness = max(v.Cleanliness, householdLevel)
	v.Happiness = max(v.Happiness, 0.7)
	v.Stress = min(v.Stress, 0.2)
	v.Fatigue = min(v.Fatigue, 0.2)
	v.Clamp()
	p.LivingAreaWaste = 0
	p.ensureNeedsLocked()
	for _, need := range append([]types.NeedType{types.NeedHunger, types.NeedThirst, types.NeedSleep,
		types.NeedCleanliness}, psychologicalNeeds...) {
		p.Needs.SetNeedLevel(need, max(p.Needs.GetNeedLevel(need), householdLevel))
	}
	p.LastInteractionAge = b.GetAgeInDays() // It has had company all along

	p.keepLifeStageLocked(stage)
	switch now := b.LifeStage(); {
	case stage == biology.StageYoung && now != biology.StageYoung:
		news.GrewUp = true
		p.addHistoryLocked("Grew up", p.Name+" grew up while the household looked after it", "")
	case stage == biology.StageAdult && now == biology.StageSenior:
		news.BecameSenior = true
	}
	b.CheckOldAge()
	if !b.IsAlive {
		news.Died = true
		p.addHistoryLocked("Passed away", p.Name+" died of old age at home, cared for by the household", "")
	}
	return news
}

// LiveTogether lets pets that live together grow a little closer over
// days game days: each pair that has met and is alive. active is the
// active pet; the others count only while they live on with the
// household (not paused).
func LiveTogether(days float64, active *DigitalPet, saved ...*DigitalPet) {
	if !(days > 0) {
		return
	}
	pets := saved
	if active != nil {
		pets = append([]*DigitalPet{active}, saved...)
	}
	for i, a := range pets {
		for _, b := range pets[i+1:] {
			if a == b {
				continue
			}
			unlock := lockPair(a, b)
			together := a.Biology.IsAlive && b.Biology.IsAlive && (a == active || !a.HouseholdPaused) && !b.HouseholdPaused
			if together {
				for _, pair := range [][2]*DigitalPet{{a, b}, {b, a}} {
					if rel, ok := pair[0].Relationships.GetRelationship(pair[1].ID); ok && rel.BondStrength < householdBondLimit {
						rel.BondStrength = min(rel.BondStrength+householdBondGrowth*days, householdBondLimit)
					}
				}
			}
			unlock()
		}
	}
}
