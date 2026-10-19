package core

import (
	"fmt"
	"strings"
)

// The pet's living area gets dirty as it relieves itself: a cat uses its
// litter box, and a dog that isn't taken outside in time has accidents in
// the house. A dirty area fouls the air and can make the pet ill.
// Illnesses get worse until the pet is given medical care.

const (
	// Waste each use of the litter box (or accident) adds to the living
	// area, 0.0 (clean) to 1.0 (filthy)
	stoolWaste = 0.15
	urineWaste = 0.08

	// catUseLevel is how full the bowel or bladder gets before a cat uses
	// its litter box
	catUseLevel = 0.3

	// catAvoidLevel is how dirty a litter box can be before a cat avoids
	// it and goes elsewhere
	catAvoidLevel = 0.8

	// dogAccidentLevel is how full a dog's bowel or bladder can get before
	// it has an accident in the house
	dogAccidentLevel = 0.8

	// waterToBladder is how much of a drink fills the bladder
	waterToBladder = 0.25

	// illnessAreaLevel is how dirty the living area must be before it can
	// make the pet ill
	illnessAreaLevel = 0.6

	// illnessLungLevel is the respiratory health below which an airway
	// infection can take hold
	illnessLungLevel = 0.4

	// illnessSicknessLevel is the severity at which an ill pet behaves sick
	illnessSicknessLevel = 0.3

	// recentHealthEvent is how long (game days) a health event stays among
	// the pet's warning signs
	recentHealthEvent = 3.0

	// maxHealthEvents is how many health events are kept
	maxHealthEvents = 5
)

// Kinds of illness
const (
	IllnessUrinary     = "urinary"     // Cats: from a dirty litter box
	IllnessParasites   = "parasites"   // Dogs: from feces left in the home
	IllnessRespiratory = "respiratory" // From airways damaged by foul air
)

// Illness is an illness the pet has. It gets worse over time until treated
// with medical care.
type Illness struct {
	Kind     string  `json:"kind"`
	Severity float64 `json:"severity"`  // 0.0 to 1.0
	SinceAge float64 `json:"since_age"` // Pet's age when it fell ill
}

// illnessNames name each illness for each species
var illnessNames = map[string]map[Species]string{
	IllnessUrinary:     {SpeciesCat: "Urinary tract infection", SpeciesDog: "Urinary tract infection"},
	IllnessParasites:   {SpeciesCat: "Intestinal parasites", SpeciesDog: "Intestinal parasites"},
	IllnessRespiratory: {SpeciesCat: "Upper respiratory infection", SpeciesDog: "Respiratory infection"},
}

// HealthEvent is something that happened to the pet's health, shown among
// its warning signs for a while
type HealthEvent struct {
	Text  string  `json:"text"`
	Topic string  `json:"topic"` // Care guide topic
	Age   float64 `json:"age"`   // Pet's age when it happened
	Good  bool    `json:"good"`  // Good news, such as a recovery
}

// illnessNameLocked names the pet's illness, or returns "". Caller must
// hold p.mu.
func (p *DigitalPet) illnessNameLocked() string {
	if p.Illness == nil {
		return ""
	}
	return illnessNames[p.Illness.Kind][p.speciesLocked()]
}

// IllnessName names the pet's illness, or returns "" if it is well
func (p *DigitalPet) IllnessName() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.illnessNameLocked()
}

// recordHealthEventLocked adds a health event. Caller must hold p.mu.
func (p *DigitalPet) recordHealthEventLocked(topic, format string, args ...interface{}) {
	p.HealthEvents = append(p.HealthEvents, HealthEvent{
		Text: fmt.Sprintf(format, args...), Topic: topic, Age: p.Biology.GetAgeInDays(),
	})
	if len(p.HealthEvents) > maxHealthEvents {
		p.HealthEvents = p.HealthEvents[len(p.HealthEvents)-maxHealthEvents:]
	}
}

// airPollutionLocked is how foul the air is from the dirty living area
// (ammonia and odor). Caller must hold p.mu.
func (p *DigitalPet) airPollutionLocked() float64 {
	return max(p.LivingAreaWaste-0.3, 0) / 0.7
}

// updateHealthLocked advances the pet's body care by deltaTime game days:
// relieving itself, the illness it may catch from a dirty home, and the
// course of an illness. Caller must hold p.mu.
func (p *DigitalPet) updateHealthLocked(deltaTime float64) {
	p.Biology.EnsurePhysiology()
	p.Biology.AirPollution = p.airPollutionLocked()
	p.relieveLocked()
	p.catchIllnessLocked(deltaTime)
	p.progressIllnessLocked(deltaTime)
}

// relieveLocked lets the pet relieve itself when it needs to. Caller must
// hold p.mu.
func (p *DigitalPet) relieveLocked() {
	ph := p.Biology.Physiology
	if p.speciesLocked() == SpeciesDog {
		// Dogs relieve themselves outside on walks; held too long, they
		// have an accident in the house
		if ph.Bowel >= dogAccidentLevel || ph.Bladder >= dogAccidentLevel {
			p.LivingAreaWaste = min(p.LivingAreaWaste+ph.Bowel/dogAccidentLevel*stoolWaste+
				ph.Bladder/dogAccidentLevel*urineWaste, 1)
			ph.Bowel, ph.Bladder = 0, 0
			p.Biology.Vitals.Stress += 0.1
			p.Biology.Vitals.Clamp()
			p.recordHealthEventLocked("bathroom", "%s couldn't wait any longer and had an accident in the house. "+
				"Dogs need to go outside several times a day.", p.Name)
		}
		return
	}

	// Cats use their litter box, unless it is too dirty
	for _, use := range []struct {
		level *float64
		waste float64
	}{{&ph.Bowel, stoolWaste}, {&ph.Bladder, urineWaste}} {
		if *use.level < catUseLevel {
			continue
		}
		*use.level = 0
		if p.LivingAreaWaste >= catAvoidLevel {
			p.Biology.Vitals.Stress += 0.15
			p.Biology.Vitals.Happiness -= 0.1
			p.Biology.Vitals.Clamp()
			p.recordHealthEventLocked("litter", "%s went outside the litter box because it was too dirty. "+
				"Cats avoid a dirty box.", p.Name)
		}
		p.LivingAreaWaste = min(p.LivingAreaWaste+use.waste, 1)
	}
}

// chance returns true with the given probability
func chance(probability float64) bool {
	return float64(randIntn(1_000_000)) < probability*1_000_000
}

// catchIllnessLocked lets a pet living in dirt, or with damaged airways,
// fall ill. A strong immune system lowers the risk. Caller must hold p.mu.
func (p *DigitalPet) catchIllnessLocked(deltaTime float64) {
	if p.Illness != nil {
		return
	}
	resistance := max(1.2-p.Biology.Processes.ImmuneStrength, 0.1)

	kind, rate := "", 0.0
	if p.LivingAreaWaste > illnessAreaLevel {
		kind = IllnessUrinary
		if p.speciesLocked() == SpeciesDog {
			kind = IllnessParasites
		}
		rate = 0.3 * (p.LivingAreaWaste - illnessAreaLevel) / (1 - illnessAreaLevel) * resistance
	}
	if lungs := p.Biology.Physiology.RespiratoryHealth; lungs < illnessLungLevel {
		if r := 0.3 * (illnessLungLevel - lungs) / illnessLungLevel * resistance; r > rate {
			kind, rate = IllnessRespiratory, r
		}
	}
	if kind == "" || !chance(rate*deltaTime) {
		return
	}

	p.Illness = &Illness{Kind: kind, Severity: 0.1, SinceAge: p.Biology.GetAgeInDays()}
	p.recordHealthEventLocked(illnessTopics[kind], "%s is coming down with an illness.", p.Name)
}

// illnessTopics are the care guide topics that explain each illness
var illnessTopics = map[string]string{
	IllnessUrinary:     "litter",
	IllnessParasites:   "bathroom",
	IllnessRespiratory: "breathing",
}

// progressIllnessLocked worsens an untreated illness, which costs the pet
// health and harms the organ system it affects. Caller must hold p.mu.
func (p *DigitalPet) progressIllnessLocked(deltaTime float64) {
	ill := p.Illness
	if ill == nil {
		return
	}
	ill.Severity = min(ill.Severity+0.04*deltaTime, 1)

	v, ph := p.Biology.Vitals, p.Biology.Physiology
	v.Health -= 0.04 * ill.Severity * deltaTime
	switch ill.Kind {
	case IllnessUrinary:
		ph.Bladder += 0.15 * deltaTime // Needs to go far more often
		v.Stress += 0.03 * ill.Severity * deltaTime
	case IllnessParasites:
		ph.DigestiveHealth -= 0.05 * ill.Severity * deltaTime
	case IllnessRespiratory:
		ph.RespiratoryHealth -= 0.05 * ill.Severity * deltaTime
	}
	v.Clamp()
	ph.Clamp()
}

// treatIllnessLocked cures the pet's illness with medical care and returns
// the kind of illness it had, or "" if it was well. Caller must hold p.mu.
func (p *DigitalPet) treatIllnessLocked() string {
	if p.Illness == nil {
		return ""
	}
	kind := p.Illness.Kind
	p.recordHealthEventLocked(illnessTopics[kind], "%s was treated for %s and is recovering.",
		p.Name, strings.ToLower(p.illnessNameLocked()))
	p.HealthEvents[len(p.HealthEvents)-1].Good = true
	p.Illness = nil
	return kind
}

// BodySystem is one of the pet's organ systems, for display
type BodySystem struct {
	Name  string  `json:"name"`
	Level float64 `json:"level"` // 0.0 (failing) to 1.0 (healthy)
}

// BodySystems returns the health of the pet's organ systems
func (p *DigitalPet) BodySystems() []BodySystem {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Biology.EnsurePhysiology()
	ph := p.Biology.Physiology
	return []BodySystem{
		{"Digestion", ph.DigestiveHealth},
		{"Heart", ph.CardiovascularHealth},
		{"Lungs", ph.RespiratoryHealth},
		{"Nerves", ph.NervousHealth},
	}
}

// LivingArea describes how clean the pet's living area is
type LivingArea struct {
	Name        string  `json:"name"`        // "Litter box" or "Home"
	Cleanliness float64 `json:"cleanliness"` // 0.0 (filthy) to 1.0 (clean)
}

// GetLivingArea returns how clean the pet's living area is
func (p *DigitalPet) GetLivingArea() LivingArea {
	p.mu.Lock()
	defer p.mu.Unlock()
	return LivingArea{Name: livingAreaNames[p.speciesLocked()], Cleanliness: 1 - p.LivingAreaWaste}
}
