package core

import (
	"errors"
	"fmt"
	"math"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Upkeep: some equipment needs looking after (a fountain's filter, a flea
// collar, bedding), and some supplies have effects for a while (a
// vaccination) or for the day (a probiotic). Care supplies are bought for
// the pantry and given from it.

// SupplyCare is a care supply: upkeep and health care given from the pantry
const SupplyCare = "care"

// upkeepCurrentLocked reports whether an upkeep timer is still running.
// Caller must hold p.mu.
func (p *DigitalPet) upkeepCurrentLocked(id string) bool {
	return p.Upkeep[id] > p.Biology.GetAgeInDays()
}

// renewUpkeepLocked starts an upkeep timer afresh. Caller must hold p.mu.
func (p *DigitalPet) renewUpkeepLocked(id string, days float64) {
	if p.Upkeep == nil {
		p.Upkeep = make(map[string]float64)
	}
	p.Upkeep[id] = p.Biology.GetAgeInDays() + days
}

// usedTodayLocked reports whether a once-a-day supply was given today.
// Caller must hold p.mu.
func (p *DigitalPet) usedTodayLocked(id string) bool {
	return p.DailyUse[id] == today()
}

// timedEffectsLocked returns the effects of supplies given lately. Caller
// must hold p.mu.
func (p *DigitalPet) timedEffectsLocked() []itemEffect {
	var effects []itemEffect
	if p.upkeepCurrentLocked(vaccinationUpkeep) {
		effects = append(effects, eff(effIllness, 0.7))
	}
	if p.usedTodayLocked("probiotic") {
		effects = append(effects, eff(effDigestiveRecovery, 1.5), eff(effFoodUpset, 0.5))
	}
	if p.usedTodayLocked("joint_supplement") {
		effects = append(effects, effWhen(effFitnessLoss, 0.85, seniors))
	}
	if p.usedTodayLocked("kidney_diet") {
		effects = append(effects, eff(effKidneyDecline, 0.5))
	}
	if p.usedTodayLocked("eye_wipes") {
		effects = append(effects, eff(effEyeIrritation, 0.5))
	}
	if p.usedTodayLocked("pet_toothpaste") {
		effects = append(effects, eff(effDentalDecline, 0.5))
	}
	if p.usedTodayLocked("dental_chews") {
		effects = append(effects, eff(effDentalDecline, 0.8))
	}
	return effects
}

// careSupplies are the care supplies for every species (dental chews and
// cat grass are with each species' supplies)
var careSupplies = []supply{
	careSupply("fountain_filter", "Fountain filter", "🫧", 30, interaction.RarityUncommon, 1,
		"Another 300 game days of the Water Fountain working fully."),
	careSupply("flea_collar_refill", "Replacement flea collar", "🛡️", 95, interaction.RarityRare, 1,
		"Another 500 game days of the Flea and Tick Collar."),
	careSupply("first_aid_refill", "First-aid refill", "🩹", 80, interaction.RarityUncommon, 1,
		"3 more uses of the First-aid Kit."),
	careSupply("vaccination", "Vaccination", "💉", 190, interaction.RarityRare, 1,
		"Illness from dirt or foul air is 30% less likely for 1,000 game days; then it is due again."),
	careSupply("dewormer", "Dewormer", "🪱", 65, interaction.RarityUncommon, 1,
		"Clears parasites picked up on trips, and treats them if your pet has them."),
	careSupply("probiotic", "Probiotic", "💊", 40, interaction.RarityUncommon, 5,
		"For the rest of the day the gut recovers faster, and milk or scraps upset it half as much (they are still bad)."),
	careSupply("joint_supplement", "Joint supplement", "🦴", 65, interaction.RarityEpic, 10,
		"Seniors: fitness fades a little slower for the day. One a day counts."),
	careSupply("freeze_dried_treats", "Freeze-dried treats", "🍗", 50, interaction.RarityRare, 5,
		"A high-value reward: the next training session counts 50% more."),
	careSupply("pet_toothpaste", "Pet toothbrush and toothpaste", "🪥", 20, interaction.RarityCommon, 10,
		"Brush the teeth: removes plaque, and plaque builds half as fast for the rest of the day. One a day counts."),
	careSupply("eye_wipes", "Eye wipes", "👁️", 15, interaction.RarityCommon, 10,
		"Gently clean sore eyes: they feel better, and are irritated half as much for the rest of the day. One a day counts."),
	careSupply("kidney_diet", "Kidney-support diet", "🩺", 60, interaction.RarityRare, 5,
		"A vet's diet for pets with weak kidneys: kidney damage builds half as fast for the day. One a day counts."),
	careSupply("fresh_bedding", "Fresh bedding", "🧺", 25, interaction.RarityCommon, 1,
		"Beds work fully for another 200 game days."),
	forSpecies(careSupply("dental_chews", "Dental chews", "🦷", 30, interaction.RarityCommon, 7,
		"Slows plaque by a fifth for the day, helps the gut a little and eases stress. One a day counts."), SpeciesDog),
	forSpecies(careSupply("cat_grass", "Cat grass", "🌿", 20, interaction.RarityCommon, 5,
		"Helps with hairballs (digestive health)."), SpeciesCat),
}

// forSpecies limits a supply to one species
func forSpecies(s supply, species Species) supply {
	s.species = []Species{species}
	return s
}

// careSupply makes a care supply
func careSupply(id, name, icon string, price int, rarity interaction.ItemRarity, pack int, description string) supply {
	return supply{item: interaction.Item{ID: id, Name: name, Price: price, Rarity: rarity,
		Type: interaction.ItemTypeMedicine, Description: description}, icon: icon, kind: SupplyCare, pack: pack}
}

// Errors giving care supplies
var (
	ErrNotNeeded  = errors.New("not needed now")
	ErrNoFirstAid = errors.New("no first-aid kit uses")
	ErrNotInjured = errors.New("not injured")
)

// careLessons are taught when a care supply is given
var careLessons = map[string]string{
	"fountain_filter":     "A clogged filter lets bacteria grow; change it as the maker says.",
	"flea_collar_refill":  "Collars wear out; replace them on time, all year round.",
	"vaccination":         "Vaccinations protect against serious diseases; vets keep a schedule for each pet.",
	"dewormer":            "Pets that go outdoors pick up worms; regular deworming keeps them healthy.",
	"probiotic":           "Probiotics support the gut's good bacteria, but they don't make bad food safe.",
	"joint_supplement":    "Supplements can ease old joints; ask a vet which ones help.",
	"freeze_dried_treats": "Save the best treats for training: a pet works hardest for a reward it loves.",
	"fresh_bedding":       "Wash or change bedding weekly to keep fleas and smells away.",
	"dental_chews":        "Chewing scrapes plaque off teeth; brushing is best, chews help between brushes.",
	"cat_grass":           "Grass helps a cat bring up hairballs; it's safer than nibbling houseplants.",
	"calming_chew":        "Fireworks frighten pets: keep them indoors with a den during celebrations.",
}

// GiveCareSupply gives the pet a care supply from the pantry, and returns
// what happened and what it teaches
func (p *DigitalPet) GiveCareSupply(id string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Biology.IsAlive {
		return "", ErrPetNotAvailable
	}
	s, ok := findSupply(id)
	if !ok || s.kind != SupplyCare || !s.forSpecies(p.speciesLocked()) {
		return "", ErrUnknownSupply
	}
	p.ensureSuppliesLocked()
	if p.Supplies.GetItemQuantity(id) == 0 {
		return "", ErrNoSupply
	}
	text, err := p.applyCareSupplyLocked(id)
	if err != nil {
		return "", err
	}
	_ = p.useSupplyLocked(id) // Checked above, under the same lock
	if lesson := careLessons[id]; lesson != "" {
		text += " " + lesson
	}
	return text, nil
}

// applyCareSupplyLocked applies a care supply. Caller must hold p.mu.
func (p *DigitalPet) applyCareSupplyLocked(id string) (string, error) {
	v, ph := p.Biology.Vitals, p.Biology.Physiology
	p.Biology.EnsurePhysiology()
	once := func(text string, apply func()) (string, error) {
		if p.usedTodayLocked(id) {
			return "", fmt.Errorf("%w: %s already had one today", ErrNotNeeded, p.Name)
		}
		if p.DailyUse == nil {
			p.DailyUse = make(map[string]string)
		}
		p.DailyUse[id] = today()
		apply()
		v.Clamp()
		ph.Clamp()
		return text, nil
	}
	needs := func(itemID string) error {
		if !p.ownsLocked(itemID) {
			item, _ := findItem(itemID)
			return fmt.Errorf("%w: it's for the %s", ErrNotNeeded, item.Name)
		}
		return nil
	}

	switch id {
	case "fountain_filter":
		if err := needs("water_fountain"); err != nil {
			return "", err
		}
		p.renewUpkeepLocked(fountainFilterUpkeep, fountainFilterDays)
		p.goalEventLocked(GoalUpkeep, fountainFilterUpkeep)
		return "The fountain has a fresh filter.", nil
	case "flea_collar_refill":
		if err := needs("flea_collar"); err != nil {
			return "", err
		}
		p.renewUpkeepLocked(fleaCollarUpkeep, fleaCollarDays)
		p.goalEventLocked(GoalUpkeep, fleaCollarUpkeep)
		return p.Name + " has a fresh flea and tick collar.", nil
	case "first_aid_refill":
		if err := needs("first_aid_kit"); err != nil {
			return "", err
		}
		p.FirstAidUses += 3
		return fmt.Sprintf("The first-aid kit is refilled: %d uses.", p.FirstAidUses), nil
	case "fresh_bedding":
		if _, ok := p.bestInGroupLocked("bed"); !ok {
			return "", fmt.Errorf("%w: it's for a bed", ErrNotNeeded)
		}
		p.renewUpkeepLocked(beddingUpkeep, beddingDays)
		p.goalEventLocked(GoalUpkeep, beddingUpkeep)
		return p.Name + "'s bed has fresh bedding.", nil
	case "vaccination":
		p.renewUpkeepLocked(vaccinationUpkeep, vaccinationDays)
		p.goalEventLocked(GoalUpkeep, vaccinationUpkeep)
		return p.Name + " is vaccinated.", nil
	case "dewormer":
		p.ParasiteExposure = 0
		if p.Illness != nil && p.Illness.Kind == IllnessParasites {
			p.Illness = nil
			return p.Name + " is dewormed and its parasites are gone.", nil
		}
		return p.Name + " is dewormed.", nil
	case "probiotic":
		return once(p.Name+" had a probiotic.", func() {})
	case "joint_supplement":
		if p.Biology.LifeStage() != biology.StageSenior {
			return "", fmt.Errorf("%w: joint supplements are for seniors", ErrNotNeeded)
		}
		return once(p.Name+" had a joint supplement.", func() {})
	case "freeze_dried_treats":
		if p.TrainingBoost {
			return "", fmt.Errorf("%w: a reward is already waiting for the next training", ErrNotNeeded)
		}
		p.TrainingBoost = true
		return "A freeze-dried treat is ready for the next training session.", nil
	case "dental_chews":
		return once(p.Name+" enjoyed a dental chew.", func() {
			ph.DigestiveHealth += 0.05
			v.Stress -= 0.03
		})
	case "pet_toothpaste":
		return once(p.Name+"'s teeth are brushed.", func() {
			ph.DentalHealth += 0.1
			p.countDayLocked("teeth_days")
			p.goalEventLocked(GoalTeeth, "")
		})
	case "eye_wipes":
		return once(p.Name+"'s eyes are cleaned.", func() {
			ph.EyeHealth += 0.08
			p.goalEventLocked(GoalEyes, "")
		})
	case "kidney_diet":
		if ph.KidneyReserve >= 0.9 && p.predispositionsLocked().KidneyDisease == 0 {
			return "", fmt.Errorf("%w: %s's kidneys are healthy, so it doesn't need a special diet", ErrNotNeeded, p.Name)
		}
		return once(p.Name+" had its kidney-support diet.", func() { v.Hydration += 0.1 })
	case "cat_grass":
		ph.DigestiveHealth = min(ph.DigestiveHealth+0.05, 1)
		return p.Name + " nibbled some cat grass.", nil
	case "calming_chew":
		v.Stress = math.Max(v.Stress-0.1, 0)
		return p.Name + " had a calming chew.", nil
	}
	return "", ErrUnknownSupply
}

// UseFirstAid treats a trip injury with the first-aid kit
func (p *DigitalPet) UseFirstAid() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch {
	case !p.Biology.IsAlive:
		return "", ErrPetNotAvailable
	case p.FirstAidUses <= 0:
		return "", ErrNoFirstAid
	case !p.Injured:
		return "", ErrNotInjured
	}
	p.FirstAidUses--
	p.Injured = false
	return fmt.Sprintf("You cleaned and covered %s's wound. 🩹 (%d first-aid uses left.) %s", p.Name, p.FirstAidUses,
		"Clean and cover small wounds; see a vet for deep ones."), nil
}

// UpkeepStatus is an upkeep timer, as the page shows it
type UpkeepStatus struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	DaysLeft float64 `json:"days_left"` // Game days; 0 when it has run out
}

// UpkeepStatuses returns the pet's upkeep timers that matter to it
func (p *DigitalPet) UpkeepStatuses() []UpkeepStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	age := p.Biology.GetAgeInDays()
	var list []UpkeepStatus
	add := func(id, name string, relevant bool) {
		if relevant {
			list = append(list, UpkeepStatus{ID: id, Name: name, DaysLeft: math.Max(p.Upkeep[id]-age, 0)})
		}
	}
	_, hasBed := p.bestInGroupLocked("bed")
	add(fountainFilterUpkeep, "Fountain filter", p.ownsLocked("water_fountain"))
	add(fleaCollarUpkeep, "Flea and tick collar", p.ownsLocked("flea_collar"))
	add(beddingUpkeep, "Bedding", hasBed)
	add(vaccinationUpkeep, "Vaccination", p.Upkeep[vaccinationUpkeep] > 0)
	return list
}

// cleanedLocked records that the living area was cleaned today (the
// self-cleaning litter box needs a scoop each day). Caller must hold p.mu.
func (p *DigitalPet) cleanedLocked(interactionType types.InteractionType) {
	if interactionType == types.InteractionCleaning {
		p.LastCleanedDay = today()
	}
}

// idleEnrichmentLocked lets things the pet can enjoy on its own (an
// aquarium, a ball launcher) meet its needs over deltaTime game days while
// it is idle. Caller must hold p.mu.
func (p *DigitalPet) idleEnrichmentLocked(deltaTime float64) {
	if p.CurrentBehavior != types.BehaviorIdle {
		return
	}
	e := p.effectsLocked()
	if e[effIdleStimulation] == 0 && e[effIdleExercise] == 0 {
		return
	}
	p.ensureNeedsLocked()
	age := p.Biology.GetAgeInDays()
	if s := e[effIdleStimulation]; s > 0 {
		p.Needs.SatisfyNeed(types.NeedMentalStimulation, s*deltaTime, age)
	}
	if x := e[effIdleExercise]; x > 0 && p.Biology.Vitals.Fatigue < 0.6 {
		p.Needs.SatisfyNeed(types.NeedExercise, x*deltaTime, age)
	}
}

// careStimulationLocked lets belongings make care more interesting: a
// snuffle mat turns meals into a puzzle, a leaf pile makes play an
// adventure. Caller must hold p.mu.
func (p *DigitalPet) careStimulationLocked(interactionType types.InteractionType) {
	kind := effFeedStimulation
	switch interactionType {
	case types.InteractionFeeding:
	case types.InteractionPlaying:
		kind = effPlayStimulation
	default:
		return
	}
	if amount := p.effectLocked(kind); amount > 0 {
		p.ensureNeedsLocked()
		p.Needs.SatisfyNeed(types.NeedMentalStimulation, amount, p.Biology.GetAgeInDays())
	}
}
