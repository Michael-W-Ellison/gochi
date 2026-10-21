package core

import (
	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

// Item effects. Equipment, decorations, seasonal items and house upgrades
// each change something the simulation already does: how fast a need
// builds, how quickly stress eases, how likely a trip injury is. Effects
// can depend on the pet (its personality, old age), the weather, the real
// season, and upkeep (a fountain filter, a flea collar, fresh bedding).
//
// Effects of the same kind combine, within limits, so care always
// matters: together they slow a need, or reduce stress or a risk, by at
// most half; decorations add at most 6% to any one thing; and nothing
// removes a need or prevents death.

// effectKind is what an effect changes
type effectKind int

const (
	// Rates, as multipliers: below 1.0 slows or reduces, above speeds
	effHunger            effectKind = iota // How fast hunger builds
	effThirst                              // How fast thirst builds
	effTiredness                           // How fast tiredness builds
	effEnergy                              // How fast energy runs down
	effCleanliness                         // How fast the coat gets dirty
	effHappinessDrop                       // How fast happiness falls
	effMess                                // How fast the living area gets dirty
	effAccidents                           // How likely a dog is to have an accident inside
	effDirtStress                          // Stress from a dirty living area
	effWeatherStress                       // Stress from the weather
	effNightStress                         // Stress from the surroundings at night
	effAloneStress                         // How fast stress builds when the pet is alone
	effConflictStress                      // Stress from clashes with other pets
	effTripStress                          // Stress from trips
	effParasites                           // How fast parasite exposure builds on trips
	effIllness                             // Chance of falling ill from dirt or foul air
	effFitnessLoss                         // How fast fitness fades
	effAirDamage                           // How fast foul air harms the airways
	effFoodUpset                           // How much unsuitable food upsets the stomach
	effExerciseNeed                        // How fast the needs for exercise and stimulation build
	effKidneyDecline                       // How fast the kidneys' reserve is lost
	effEyeIrritation                       // How fast prone eyes get sore
	effDentalDecline                       // How fast plaque builds
	effHeartAging                          // How fast the heart ages
	effStressRecovery                      // How fast stress eases
	effRest                                // How fast sleep restores
	effSleepDebt                           // How fast sleep debt is repaid
	effAirRecovery                         // How fast the airways heal
	effDigestiveRecovery                   // How fast the gut recovers
	effPlay                                // How much play does
	effGrooming                            // How much grooming does
	effTraining                            // How much training does
	effFitnessGain                         // Fitness from walks and play
	effEnrichment                          // How much environment changes do
	effPerch                               // How much the window perch and scratching post do
	effAffection                           // Happiness from play and petting
	effTreatTraining                       // How much training with food rewards does

	// Amounts
	effTripInjury      // Change to trip injury chance, in percentage points
	effSelfExercise    // Added to how much the pet wants to run around on its own
	effIdleStimulation // Mental stimulation per game day while idle
	effIdleExercise    // Exercise per game day while idle and awake
	effFeedStimulation // Mental stimulation from each meal
	effPlayStimulation // Mental stimulation from each play session
	effStressCap       // The most stress there can be (0 for no limit)
	effEarlyWarnings   // Warning signs show sooner (1 when on)
	effFrailLater      // Game days later the "Very old and frail" sign appears
	effDaylightMood    // Mood hormone from daylight per game day

	effectKinds // Number of kinds
)

// reduces reports whether an effect kind's multipliers reduce something
// (below 1.0), which effects together can do by at most half
func (k effectKind) reduces() bool {
	return k < effStressRecovery
}

// multiplier reports whether an effect kind is a multiplier
func (k effectKind) multiplier() bool {
	return k < effTripInjury
}

// Limits on effects together
const (
	maxReduction  = 0.5  // Reductions bring a rate to no less than half
	maxDecoration = 0.06 // Decorations add or take away at most 6%
)

// effectCondition is when an effect applies
type effectCondition struct {
	trait     string             // A personality trait the pet must have more than 0.6 of
	senior    bool               // Only for seniors
	below     float64            // Only below this temperature (°C); 0 for any
	above     float64            // Only above this temperature (°C); 0 for any
	season    environment.Season // Only in this real season, if inSeason
	inSeason  bool
	night     bool    // Only at night
	upkeep    string  // Only while this upkeep (Upkeep) is current
	lapsed    string  // Only while this upkeep has run out
	scooped   bool    // Only if the living area was cleaned today
	species   Species // Only for this species
	halfStale string  // Half as strong once this upkeep has run out (fresh bedding)
}

// itemEffect is one thing an item does
type itemEffect struct {
	kind     effectKind
	value    float64 // A multiplier, or an amount
	when     effectCondition
	uncapped bool // A decoration effect not limited to 6% (the night lamp's)
}

// eff makes an effect that always applies
func eff(kind effectKind, value float64) itemEffect {
	return itemEffect{kind: kind, value: value}
}

// effWhen makes an effect that applies under a condition
func effWhen(kind effectKind, value float64, when effectCondition) itemEffect {
	return itemEffect{kind: kind, value: value, when: when}
}

// traitLevel returns a personality trait by name
func traitLevel(t *ai.Traits, name string) float64 {
	switch name {
	case "openness":
		return t.Openness
	case "extraversion":
		return t.Extraversion
	case "neuroticism":
		return t.Neuroticism
	case "playfulness":
		return t.Playfulness
	case "curiosity":
		return t.Curiosity
	case "energy":
		return t.EnergyLevel
	case "affectionate":
		return t.Affectionate
	case "territoriality":
		return t.Territoriality
	}
	return 0
}

// appliesLocked reports whether an effect's condition holds now, and how
// strongly (1, or 0.5 when it is half as strong). Caller must hold p.mu.
func (p *DigitalPet) appliesLocked(when effectCondition) (bool, float64) {
	if when.trait != "" && traitLevel(p.Personality.Traits, when.trait) <= 0.6 {
		return false, 0
	}
	if when.senior && p.Biology.LifeStage() != biology.StageSenior {
		return false, 0
	}
	if when.species != "" && p.speciesLocked() != when.species {
		return false, 0
	}
	if when.below != 0 || when.above != 0 || when.night {
		p.ensureEnvironmentLocked()
		temp := p.Environment.Weather.Current.Temperature
		if (when.below != 0 && temp >= when.below) || (when.above != 0 && temp <= when.above) {
			return false, 0
		}
		if when.night && p.Environment.IsDaytime() {
			return false, 0
		}
	}
	if when.inSeason && RealSeason() != when.season {
		return false, 0
	}
	if when.upkeep != "" && !p.upkeepCurrentLocked(when.upkeep) {
		return false, 0
	}
	if when.lapsed != "" && p.upkeepCurrentLocked(when.lapsed) {
		return false, 0
	}
	if when.scooped && p.LastCleanedDay != today() {
		return false, 0
	}
	if when.halfStale != "" && !p.upkeepCurrentLocked(when.halfStale) {
		return true, 0.5
	}
	return true, 1
}

// scaled weakens an effect by strength (0.5 halves it)
func scaled(kind effectKind, value, strength float64) float64 {
	if kind.multiplier() {
		return 1 + (value-1)*strength
	}
	return value * strength
}

// effectsLocked combines the effects of everything the pet owns, its
// timed supplies and the house's upgrades. Caller must hold p.mu.
func (p *DigitalPet) effectsLocked() [effectKinds]float64 {
	var total, decor [effectKinds]float64
	for k := range total {
		if effectKind(k).multiplier() {
			total[k] = 1
		}
	}
	add := func(e itemEffect, decoration bool) {
		ok, strength := p.appliesLocked(e.when)
		if !ok {
			return
		}
		value := scaled(e.kind, e.value, strength)
		switch {
		case decoration && e.kind.multiplier() && !e.uncapped:
			decor[e.kind] += value - 1
		case e.kind == effStressCap:
			if total[e.kind] == 0 || value < total[e.kind] {
				total[e.kind] = value
			}
		case e.kind.multiplier():
			total[e.kind] *= value
		default:
			total[e.kind] += value
		}
	}

	for _, item := range p.effectiveItemsLocked() {
		for _, e := range item.effects {
			add(e, item.isDecoration())
		}
	}
	for _, e := range p.timedEffectsLocked() {
		add(e, false)
	}
	if shelfSetShown() {
		add(eff(effHappinessDrop, 0.98), true) // A full set on the shelf, as a decoration
	}
	upgrades, filterFresh := houseUpgrades()
	for _, id := range upgrades {
		if up, ok := findUpgrade(id); ok {
			for _, e := range up.effects {
				if e.when.upkeep == airFilterUpkeep {
					if !filterFresh {
						continue
					}
					e.when.upkeep = ""
				}
				add(e, false)
			}
		}
	}

	for k := range total {
		kind := effectKind(k)
		if kind.multiplier() {
			total[k] *= 1 + min(max(decor[k], -maxDecoration), maxDecoration)
			if kind.reduces() {
				total[k] = max(total[k], maxReduction)
			}
		}
	}

	// The breed's tendencies come on top: care can offset them, within
	// the limits above, but not remove them
	for _, e := range p.breedEffectsLocked() {
		if ok, strength := p.appliesLocked(e.when); ok {
			value := scaled(e.kind, e.value, strength)
			if e.kind.multiplier() {
				total[e.kind] *= value
			} else {
				total[e.kind] += value
			}
		}
	}
	return total
}

// effectLocked returns the combined effect of one kind. Caller must hold
// p.mu.
func (p *DigitalPet) effectLocked(kind effectKind) float64 {
	return p.effectsLocked()[kind]
}
