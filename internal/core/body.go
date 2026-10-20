package core

import (
	"errors"
	"fmt"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/internal/genetics"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// The pet's body responds to its surroundings and its life: daylight and
// darkness set its sleep hormone, company and exercise its mood hormone,
// and its age, genes and care how long it lives. See the biology package.

// speciesLifespans are how long each species lives relative to
// biology.BaseLifespan: cats tend to live a little longer than dogs
var speciesLifespans = map[Species]float64{
	SpeciesCat: 1.05,
	SpeciesDog: 0.95,
}

// EnvironmentData is what the pet senses of its surroundings
type EnvironmentData struct {
	TimeOfDay  float64 // Hour, 0-24
	Daylight   float64 // 0.0 dark to 1.0 full daylight
	AirQuality float64 // 0.0 foul to 1.0 fresh
}

// overcastWeather dims the daylight
var overcastWeather = map[environment.WeatherType]float64{
	environment.WeatherCloudy: 0.7,
	environment.WeatherRainy:  0.6,
	environment.WeatherStormy: 0.4,
	environment.WeatherSnowy:  0.7,
	environment.WeatherFoggy:  0.6,
}

// environmentDataLocked is what the pet senses at home now. Caller must
// hold p.mu.
func (p *DigitalPet) environmentDataLocked() EnvironmentData {
	p.ensureEnvironmentLocked()
	env := p.Environment
	data := EnvironmentData{TimeOfDay: env.GetTimeOfDay(), AirQuality: 1 - p.airPollutionLocked()}
	if env.IsDaytime() {
		data.Daylight = 1
		if dim, ok := overcastWeather[env.GetWeather().Type]; ok {
			data.Daylight = dim
		}
	}
	return data
}

// ProcessEnvironmentalInput lets the pet's body respond to its
// surroundings: daylight and darkness set its sleep cycle and hormones,
// and foul air harms its lungs. The pet's own environment is applied this
// way on each update; other sources (such as a camera's light reading) can
// call it directly.
func (p *DigitalPet) ProcessEnvironmentalInput(data EnvironmentData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processEnvironmentalInputLocked(data)
}

// processEnvironmentalInputLocked applies what the pet senses. Caller must
// hold p.mu.
func (p *DigitalPet) processEnvironmentalInputLocked(data EnvironmentData) {
	b := p.Biology
	b.Inputs.TimeOfDay = data.TimeOfDay
	b.Inputs.Daylight = min(max(data.Daylight, 0), 1)
	b.AirPollution = 1 - min(max(data.AirQuality, 0), 1)
}

// prepareBodyLocked tells the body what it needs from the rest of the pet
// before it is updated: its surroundings, whether it sleeps, how well its
// needs for exercise and company are met, its species and genes, and
// whether it can die of old age (not in gentle mode). Caller must hold
// p.mu.
func (p *DigitalPet) prepareBodyLocked() {
	b := p.Biology
	b.EnsureHormones()
	b.EnsureCircadian()
	p.processEnvironmentalInputLocked(p.environmentDataLocked())

	p.ensureNeedsLocked()
	b.Inputs.Asleep = p.CurrentBehavior == types.BehaviorSleeping
	b.Inputs.Exercise = p.Needs.GetNeedLevel(types.NeedExercise)
	b.Inputs.Company = (p.Needs.GetNeedLevel(types.NeedSocial) + p.Needs.GetNeedLevel(types.NeedAffection)) / 2

	b.SpeciesLifespan = speciesLifespans[p.speciesLocked()]
	p.ensureGenomeLocked()
	b.Longevity = geneValue(p.Genome, genetics.TraitLongevity)
	b.OldAgeDeath = !p.HibernationEnabled
	p.setSleepWindowLocked()
}

// sleepWindows are when each species prefers to sleep (hours): cats are
// most active at dawn and dusk, dogs keep closer to their owners' hours
var sleepWindows = map[Species][2]float64{
	SpeciesCat: {23, 5},
	SpeciesDog: {22, 6},
}

// setSleepWindowLocked sets the sleep cycle's preferred hours for the
// pet's species. Caller must hold p.mu.
func (p *DigitalPet) setSleepWindowLocked() {
	window := sleepWindows[p.speciesLocked()]
	c := p.Biology.Circadian
	if c.OptimalSleepStart != window[0] || c.OptimalSleepEnd != window[1] {
		c.SetOptimalSleepWindow(window[0], window[1])
	}
}

// nightSleepiness is how much easier it is for the pet to fall asleep now,
// 0.0 (its waking hours) to 1.0 (the middle of its night). Caller must
// hold p.mu.
func (p *DigitalPet) nightSleepinessLocked() float64 {
	p.Biology.EnsureCircadian()
	p.ensureEnvironmentLocked()
	return p.Biology.Circadian.SleepWindowFactor(p.Environment.GetTimeOfDay())
}

// Hormone responses to care and events: affection releases oxytocin, and
// frights and scolding release cortisol
var careHormones = map[types.InteractionType]struct {
	hormone string
	amount  float64
}{
	types.InteractionPetting:     {biology.HormoneOxytocin, 0.25},
	types.InteractionGrooming:    {biology.HormoneOxytocin, 0.15},
	types.InteractionRewards:     {biology.HormoneOxytocin, 0.1},
	types.InteractionDiscipline:  {biology.HormoneCortisol, 0.2},
	types.InteractionMedicalCare: {biology.HormoneCortisol, 0.05},
}

// careHormonesLocked releases the hormones a kind of care does, scaled by
// how much the pet enjoyed it (boost: 0.6 to 1.4); unwanted affection
// releases cortisol instead. Caller must hold p.mu.
func (p *DigitalPet) careHormonesLocked(interactionType types.InteractionType, boost float64, unwanted bool) {
	response, ok := careHormones[interactionType]
	if !ok {
		return
	}
	p.Biology.EnsureHormones()
	if unwanted && response.hormone == biology.HormoneOxytocin {
		p.Biology.Hormones.Pulse(biology.HormoneCortisol, 0.1)
		return
	}
	p.Biology.Hormones.Pulse(response.hormone, response.amount*boost)
}

// causeOldAge is the cause of death of a pet that lived out its lifespan
const causeOldAge = "Old age"

// frailDays is how close to the end of its lifespan (in game days) a pet
// shows that it is very old: about 10 hours of play
const frailDays = 600.0

// Life extension
const (
	// LifePotionID is the life-extension potion's supply ID
	LifePotionID = "life_potion"

	// lifePotionDays is how many game days a potion adds to a pet's
	// lifespan: about 25 hours of play at the default pace
	lifePotionDays = 1500.0
)

// ErrNoPotion is returned when giving a potion with none in the pantry
var ErrNoPotion = errors.New("no life-extension potion in the pantry")

// LifeStatus describes the pet's age and how long it can expect to live
type LifeStatus struct {
	Stage       string           `json:"stage"`         // Young, Adult or Senior
	Age         float64          `json:"age"`           // Game days
	Lifespan    float64          `json:"lifespan"`      // Age it can expect to reach, in game days
	Remaining   float64          `json:"remaining"`     // Game days left
	CareQuality float64          `json:"care_quality"`  // Record of care, 0.0 to 1.0
	Extension   float64          `json:"extension"`     // Game days added by potions
	Potions     int              `json:"potions"`       // Potions in the pantry
	OldAgeDeath bool             `json:"old_age_death"` // False in gentle mode, where it lives on
	SleepDebt   float64          `json:"sleep_debt"`    // Hours of sleep it owes
	SleepPhase  string           `json:"sleep_phase"`   // Awake, Drowsy, Asleep, Deep Sleep, REM Sleep
	Hormones    biology.Hormones `json:"hormones"`
}

// LifeStatus returns the pet's age, life stage and expected lifespan
func (p *DigitalPet) LifeStatus() LifeStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prepareBodyLocked()
	b := p.Biology
	return LifeStatus{
		Stage:       b.LifeStage().String(),
		Age:         b.GetAgeInDays(),
		Lifespan:    b.ExpectedLifespan(),
		Remaining:   b.CalculateLifespan(),
		CareQuality: b.CareQuality,
		Extension:   b.LifeExtension,
		Potions:     p.potionsLocked(),
		OldAgeDeath: b.OldAgeDeath,
		SleepDebt:   b.Circadian.GetSleepDebt(),
		SleepPhase:  b.Circadian.GetCurrentPhase().String(),
		Hormones:    *b.Hormones,
	}
}

// potionsLocked returns how many life-extension potions are in the
// pantry. Caller must hold p.mu.
func (p *DigitalPet) potionsLocked() int {
	p.ensureSuppliesLocked()
	return p.Supplies.GetItemQuantity(LifePotionID)
}

// GiveLifePotion gives the pet a life-extension potion from the pantry,
// adding lifePotionDays to its lifespan
func (p *DigitalPet) GiveLifePotion() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Biology.IsAlive {
		return "", ErrPetNotAvailable
	}
	if p.useSupplyLocked(LifePotionID) != nil {
		return "", ErrNoPotion
	}
	p.Biology.LifeExtension += lifePotionDays
	p.Biology.Hormones.Pulse(biology.HormoneGrowth, 0.3) // A burst of renewal
	return fmt.Sprintf("%s drinks the potion and looks years younger. 🧪 It has about %.0f more game days to live.",
		p.Name, p.Biology.CalculateLifespan()), nil
}

// Self-care: a pet grooms itself and runs around on its own
const (
	selfGroomingRate = 0.05 // Cleanliness a cat restores per game day of grooming (a dog half as much)
	selfExerciseRate = 0.5  // Play sessions' worth of exercise per game day of running around
	selfExerciseCost = 0.08 // Energy running around uses per game day
)

// selfGroomingScoreLocked is how strongly the pet wants to groom itself
// (chosen above 0.6): more the less clean it is, much more for cats, and
// more for tidy (conscientious) pets. Caller must hold p.mu.
func (p *DigitalPet) selfGroomingScoreLocked() float64 {
	score := 0.6*(1-p.Biology.Vitals.Cleanliness) + 0.3*p.Personality.Traits.Conscientiousness
	if p.speciesLocked() == SpeciesCat {
		score += 0.3
	}
	return score
}

// selfExerciseScoreLocked is how strongly the pet wants to run around on
// its own (chosen above 0.6): energetic pets that haven't had enough
// exercise and have energy to spare. Caller must hold p.mu.
func (p *DigitalPet) selfExerciseScoreLocked() float64 {
	if p.Biology.Vitals.Energy < 0.6 {
		return 0
	}
	p.ensureNeedsLocked()
	return 0.5*(1-p.Needs.GetNeedLevel(types.NeedExercise)) + 0.5*p.Personality.Traits.EnergyLevel
}

// selfCareLocked applies what the pet does on its own over deltaTime game
// days: grooming itself cleans its coat and calms it, and running around
// exercises it. Caller must hold p.mu.
func (p *DigitalPet) selfCareLocked(deltaTime float64) {
	v := p.Biology.Vitals
	switch p.CurrentBehavior {
	case types.BehaviorGrooming:
		rate := selfGroomingRate
		if p.speciesLocked() == SpeciesDog {
			rate /= 2
		}
		v.Cleanliness += rate * deltaTime
		v.Stress -= 0.02 * deltaTime // Grooming is soothing
	case types.BehaviorExercising:
		sessions := selfExerciseRate * deltaTime
		p.Biology.ProcessExercise(biology.ExerciseRun, sessions)
		p.ensureNeedsLocked()
		p.Needs.SatisfyNeed(types.NeedExercise, sessions*0.5, p.Biology.GetAgeInDays())
		v.Energy -= selfExerciseCost * deltaTime
	default:
		return
	}
	v.Clamp()
}
