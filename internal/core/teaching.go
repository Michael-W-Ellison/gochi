package core

import (
	"fmt"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// The game teaches real pet care: each action explains why it matters,
// the pet shows warning signs before problems get serious, a care guide
// explains each need, and a checklist follows a real owner's daily
// routine.

// treatmentLessons explain how to keep each illness from coming back
var treatmentLessons = map[string]string{
	IllnessPoisoning:   "Treatment worked this time. Keep chocolate, grapes and raisins, onions, garlic and xylitol well out of reach.",
	IllnessUrinary:     "Treatment helps, but keep the litter box clean: a cat that avoids a dirty box is prone to urinary problems.",
	IllnessParasites:   "Treatment clears the parasites; picking up after your dog every time stops them coming back.",
	IllnessRespiratory: "Treatment helps the airways heal; a clean home with fresh air keeps them healthy.",
}

// lessonLocked returns what a kind of care teaches, or the lesson of the
// illness the care just treated. Caller must hold p.mu.
func (p *DigitalPet) lessonLocked(interactionType types.InteractionType, treatedKind string) string {
	if lesson, ok := treatmentLessons[treatedKind]; ok {
		return lesson
	}
	lessons := p.defLocked().CareLessons[interactionType]
	if len(lessons) == 0 {
		return ""
	}
	return lessons[p.TotalInteractions%len(lessons)]
}

// GuideTopic is one lesson in the care guide
type GuideTopic struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// CareGuide returns the care guide for a species
func CareGuide(species Species) []GuideTopic {
	return append([]GuideTopic(nil), species.Def().CareGuide...)
}

// Symptom is a warning sign the pet shows, with what it means and the
// care guide topic that explains it
type Symptom struct {
	Sign  string `json:"sign"`
	Hint  string `json:"hint"`
	Topic string `json:"topic"`
	Good  bool   `json:"good"` // Good news, such as a recovery
}

// Symptoms returns the warning signs the pet shows now, most serious
// first. Signs appear before problems become serious.
func (p *DigitalPet) Symptoms() []Symptom {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.symptomsLocked()
}

// symptomsLocked returns the pet's warning signs. Caller must hold p.mu.
func (p *DigitalPet) symptomsLocked() []Symptom {
	symptoms := []Symptom{}
	if !p.Biology.IsAlive {
		return symptoms
	}
	p.Biology.EnsurePhysiology()
	def := p.defLocked()
	ph := p.Biology.Physiology
	e := p.effectsLocked()
	early := 0.05 * e[effEarlyWarnings] // A home vet corner shows signs sooner

	if p.Illness != nil {
		symptoms = append(symptoms, def.IllnessSigns[p.Illness.Kind])
	}
	age := p.Biology.GetAgeInDays()
	for i := len(p.HealthEvents) - 1; i >= 0; i-- {
		if e := p.HealthEvents[i]; age-e.Age <= recentHealthEvent {
			symptoms = append(symptoms, Symptom{Sign: e.Text, Topic: e.Topic, Good: e.Good})
		}
	}

	if p.LivingAreaWaste >= 0.5-early {
		symptoms = append(symptoms, def.MessSymptom)
	}
	if def.Toilet == ToiletOutside && max(ph.Bowel, ph.Bladder) >= 0.5 {
		symptoms = append(symptoms, Symptom{"Waiting by the door",
			"Your " + strings.ToLower(def.Name) + " needs to go outside. Take it for a walk.", "bathroom", false})
	}
	if ph.RespiratoryHealth < 0.7+early && (p.Illness == nil || p.Illness.Kind != IllnessRespiratory) {
		symptoms = append(symptoms, Symptom{"Sneezing now and then",
			"Foul air irritates the airways. Keep the living area clean.", "breathing", false})
	}
	if ph.CardiovascularHealth < 0.5+early {
		symptoms = append(symptoms, Symptom{"Tires quickly", def.ExerciseHint, "exercise", false})
	}
	if ph.NervousHealth < 0.6+early {
		symptoms = append(symptoms, Symptom{def.StressSign, "Signs of long-term stress. A calm routine, play and rest help.", "stress", false})
	}
	// Old age
	if b := p.Biology; b.LifeStage() == biology.StageSenior {
		remaining := b.CalculateLifespan()
		switch {
		case b.OldAgeDeath && remaining < frailDays-e[effFrailLater]:
			symptoms = append(symptoms, Symptom{"Very old and frail",
				fmt.Sprintf("%s is near the end of its life (about %.0f game days left). A life-extension potion from the shop "+
					"gives it about %.0f more.", p.Name, remaining, lifePotionDays), "senior", false})
		case b.Senescence() > 0.3:
			symptoms = append(symptoms, Symptom{"Slowing down with age",
				"Older pets tire sooner and fall ill more easily. Gentle exercise, easy-to-digest food and regular checkups help.",
				"senior", false})
		}
	}
	if ph.DigestiveHealth < 0.6+early && (p.Illness == nil || p.Illness.Kind != IllnessParasites) {
		symptoms = append(symptoms, Symptom{"Upset stomach",
			"From too much food or too many treats at once. Smaller, measured meals help.", "feeding", false})
	}
	return symptoms
}

// CareChecklist tracks the day's care, following a real owner's routine.
// It counts care given on the current calendar day.
type CareChecklist struct {
	Day  string         `json:"day"` // YYYY-MM-DD
	Done map[string]int `json:"done"`
}

// checklistItem is one task in a species' daily routine
type checklistItem struct {
	ID     string
	Label  string
	Target int
	Care   types.InteractionType
}

// ChecklistItem is a task on today's checklist
type ChecklistItem struct {
	Label  string `json:"label"`
	Done   int    `json:"done"`
	Target int    `json:"target"`
}

// today returns the current calendar day
func today() string {
	return now().Format("2006-01-02")
}

// checkOffLocked records care on today's checklist. Caller must hold p.mu.
func (p *DigitalPet) checkOffLocked(interactionType types.InteractionType) {
	day := today()
	if p.Checklist == nil || p.Checklist.Day != day {
		p.Checklist = &CareChecklist{Day: day, Done: map[string]int{}}
	}
	for _, item := range p.defLocked().DailyRoutine {
		if item.Care == interactionType {
			p.Checklist.Done[item.ID]++
		}
	}
}

// DailyChecklist returns today's care checklist
func (p *DigitalPet) DailyChecklist() []ChecklistItem {
	p.mu.Lock()
	defer p.mu.Unlock()

	items := []ChecklistItem{}
	for _, item := range p.defLocked().DailyRoutine {
		done := 0
		if p.Checklist != nil && p.Checklist.Day == today() {
			done = min(p.Checklist.Done[item.ID], item.Target)
		}
		items = append(items, ChecklistItem{Label: item.Label, Done: done, Target: item.Target})
	}
	return items
}
