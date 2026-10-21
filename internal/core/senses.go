package core

import (
	"fmt"
	"strings"
)

// The kidneys, eyes, teeth and hearing (modelled in the biology package).
// Some breeds are prone to problems with them, as real ones are: inherited
// kidney disease, sore prominent eyes, crowded teeth, and deafness. Each
// has warning signs, a care guide topic, and care that helps: water and a
// kidney-support diet, eye wipes, tooth brushing, and training with hand
// signals for a pet that can't hear well.

// Levels below which an eye infection or dental disease can take hold
const (
	illnessEyeLevel    = 0.35
	illnessDentalLevel = 0.3
)

// senseIllnessLocked returns the illness sore eyes or neglected teeth can
// lead to, and how likely it is per game day. Caller must hold p.mu.
func (p *DigitalPet) senseIllnessLocked(resistance float64) (string, float64) {
	ph := p.Biology.Physiology
	kind, rate := "", 0.0
	if ph.EyeHealth < illnessEyeLevel {
		kind, rate = IllnessEye, 0.3*(illnessEyeLevel-ph.EyeHealth)/illnessEyeLevel*resistance
	}
	if ph.DentalHealth < illnessDentalLevel {
		if r := 0.3 * (illnessDentalLevel - ph.DentalHealth) / illnessDentalLevel * resistance; r > rate {
			kind, rate = IllnessDental, r
		}
	}
	return kind, rate
}

// giveFluidsLocked is what medical care does for kidneys hurt by
// dehydration: fluids bring them back up to what they can still do.
// Caller must hold p.mu.
func (p *DigitalPet) giveFluidsLocked() bool {
	ph := p.Biology.Physiology
	if ph.KidneyHealth >= ph.KidneyReserve-0.05 {
		return false
	}
	ph.KidneyHealth = ph.KidneyReserve
	return true
}

// senseSymptomsLocked returns the warning signs of the kidneys, eyes,
// teeth and hearing. early is how much sooner signs show (a home vet
// corner). Caller must hold p.mu.
func (p *DigitalPet) senseSymptomsLocked(early float64) []Symptom {
	ph := p.Biology.Physiology
	var signs []Symptom
	ill := func(kind string) bool { return p.Illness != nil && p.Illness.Kind == kind }
	switch {
	case ph.KidneyReserve < 0.5+early:
		signs = append(signs, Symptom{"Kidney disease",
			p.Name + "'s kidneys are failing. It can't be cured, but plenty of water, wet food and a kidney-support " +
				"diet slow it down.", "kidneys", false})
	case ph.KidneyHealth < 0.7+early:
		signs = append(signs, Symptom{"Drinking and peeing more than usual",
			"An early sign the kidneys are struggling. Make sure water is always there; medical care (Heal) gives " +
				"fluids, and a vet can test the kidneys.", "kidneys", false})
	}
	if ph.EyeHealth < 0.7+early && !ill(IllnessEye) {
		signs = append(signs, Symptom{"Squinting, with runny eyes",
			"Sore eyes. Wipe them gently with eye wipes, and keep the air clean.", "eyes", false})
	}
	if ph.DentalHealth < 0.6+early && !ill(IllnessDental) {
		signs = append(signs, Symptom{"Bad breath and tartar on the teeth",
			"Plaque is building up. Brush the teeth with pet toothpaste every day.", "teeth", false})
	}
	if ph.Hearing < 0.5 {
		signs = append(signs, Symptom{"Doesn't come when called",
			p.Name + " can't hear well. Train with hand signals, and approach where it can see you so it isn't startled.",
			"hearing", false})
	}
	return signs
}

// senseGuide returns the care guide topics for the kidneys, eyes, teeth
// and hearing
func senseGuide(species Species) []GuideTopic {
	kind := strings.ToLower(species.Def().Name)
	kidneys := "Kidneys filter waste from the blood. They wear with age, and some breeds inherit kidney disease. " +
		"Signs are drinking and peeing more, weight loss and a poor appetite. It can't be cured, but plenty of " +
		"water, wet food and a kidney-support diet from a vet slow it down; regular checkups catch it early."
	if species == SpeciesCat {
		kidneys += " Kidney disease is one of the most common illnesses of older cats."
	}
	return []GuideTopic{
		{"kidneys", "Kidneys", kidneys},
		{"eyes", "Eyes", "Breeds with prominent eyes or short faces get sore eyes easily, and their eyes are easily " +
			"scratched. Wipe away discharge daily with a clean eye wipe, keep the air clean, and see a vet if an eye " +
			"is red, cloudy or held shut."},
		{"teeth", "Teeth", fmt.Sprintf("Most %s have dental disease by middle age. Plaque hardens into tartar and "+
			"inflames the gums, which is painful and can spread infection. Brush the teeth daily with toothpaste made "+
			"for pets (never human toothpaste); small and short-faced breeds, with crowded teeth, need it most. A vet "+
			"cleans teeth that have tartar.", species.Def().Plural)},
		{"hearing", "Hearing", fmt.Sprintf("Some %s are born deaf in one or both ears: white coats with blue eyes, "+
			"and merle coats, carry the risk. Hearing also fades in old age. A deaf %s can live a full life: train "+
			"it with hand signals instead of words or a clicker, approach where it can see you, and wake it with a "+
			"gentle touch or a stamp on the floor.", species.Def().Plural, kind)},
	}
}

// hearingFactor is how well training that relies on sound (words or a
// clicker) works for a pet with the given hearing
func hearingFactor(hearing float64) float64 {
	return 0.25 + 0.75*hearing
}
