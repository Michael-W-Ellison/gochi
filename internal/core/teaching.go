package core

import (
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// The game teaches real pet care: each action explains why it matters,
// the pet shows warning signs before problems get serious, a care guide
// explains each need, and a checklist follows a real owner's daily
// routine.

// careLessons explain why each kind of care matters, per species. One is
// shown after each action, taking turns when there are several.
var careLessons = map[Species]map[types.InteractionType][]string{
	SpeciesCat: {
		types.InteractionFeeding: {
			"Two measured meals a day keep most adult cats at a healthy weight.",
			"Many adult cats can't digest milk well; water is the best drink.",
		},
		types.InteractionDrinking: {
			"Fresh water every day, away from the food bowl, helps cats drink enough to protect their kidneys and bladder.",
		},
		types.InteractionCleaning: {
			"Scooping the litter box every day keeps your cat using it and helps prevent urinary problems.",
			"Ammonia from a dirty litter box irritates a cat's airways; a clean box means cleaner air.",
		},
		types.InteractionPlaying: {
			"Short daily play sessions keep an indoor cat fit and prevent boredom.",
		},
		types.InteractionPetting: {
			"Most cats prefer gentle strokes on the head and cheeks; let your cat decide when it has had enough.",
		},
		types.InteractionGrooming: {
			"Brushing removes loose hair, which means fewer hairballs, and lets you check the skin and coat.",
		},
		types.InteractionMedicalCare: {
			"A check-up at least once a year catches problems early.",
		},
		types.InteractionRewards: {
			"Treats should be no more than about 10% of what your cat eats in a day.",
		},
		types.InteractionTraining: {
			"Cats learn best in short sessions with rewards; punishment doesn't work.",
		},
		types.InteractionEnvironmentalEnrichment: {
			"New things to explore keep a curious cat's mind busy; indoors, window perches and new toys do the same.",
		},
		types.InteractionDiscipline: {
			"Punishment adds stress and fear; rewarding the behavior you want works better.",
		},
	},
	SpeciesDog: {
		types.InteractionFeeding: {
			"Two measured meals a day suit most adult dogs; table scraps upset their stomach, and some foods are toxic to them.",
		},
		types.InteractionDrinking: {
			"Dogs need fresh water available all the time, especially after exercise and in hot weather.",
		},
		types.InteractionWalking: {
			"Dogs need to go outside several times a day; regular walks prevent accidents and keep them fit.",
			"Always pick up after your dog: feces spread parasites to other dogs and to people.",
		},
		types.InteractionCleaning: {
			"Clean up accidents promptly: feces in the home spread parasites, and lingering smells invite repeat accidents.",
		},
		types.InteractionPlaying: {
			"Daily play and exercise prevent the boredom behind chewing and barking.",
		},
		types.InteractionPetting: {
			"Most dogs like being stroked on the chest and shoulders; turning away or lip licking means 'that's enough'.",
		},
		types.InteractionGrooming: {
			"Check the ears, paws and nails while you brush.",
		},
		types.InteractionMedicalCare: {
			"Yearly check-ups, vaccinations and parasite prevention keep dogs healthy.",
		},
		types.InteractionRewards: {
			"Keep treats to about 10% of the day's food.",
		},
		types.InteractionTraining: {
			"Reward-based training in short, frequent sessions works best for dogs.",
		},
		types.InteractionEnvironmentalEnrichment: {
			"New places and smells exercise a dog's mind as well as its body.",
		},
		types.InteractionDiscipline: {
			"Punishment adds stress and fear; rewarding the behavior you want works better.",
		},
	},
}

// treatmentLessons explain how to keep each illness from coming back
var treatmentLessons = map[string]string{
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
	lessons := careLessons[p.speciesLocked()][interactionType]
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

// careGuide is the care guide for each species
var careGuide = map[Species][]GuideTopic{
	SpeciesCat: {
		{"feeding", "Feeding", "Adult cats do well on two measured meals a day of complete cat food. Follow the " +
			"portion on the label for your cat's weight; free-feeding and extra treats lead to weight gain. Treats " +
			"should be no more than about 10% of the day's food, and eating too much at once can make a cat throw " +
			"up. Never give onions, garlic or chocolate, and remember that many adult cats can't digest milk well."},
		{"water", "Water", "Cats often don't drink enough. Give fresh water every day in a clean bowl, away from " +
			"the food and the litter box. Wet food and water fountains help cats drink more, which protects the " +
			"kidneys and bladder."},
		{"litter", "Litter box", "Scoop the litter box at least once a day, and empty and wash it regularly. A " +
			"common rule is one box per cat plus one, in quiet spots. Cats avoid a dirty box and may go elsewhere " +
			"or hold their urine, which is linked to urinary problems. Ammonia from an unclean box also irritates " +
			"the airways."},
		{"exercise", "Play and exercise", "Indoor cats need daily play to stay fit and avoid boredom. A few short " +
			"sessions with a wand toy, letting the cat stalk, chase and catch, imitate hunting. A cat that doesn't " +
			"exercise loses fitness and gains weight."},
		{"grooming", "Grooming", "Brushing removes loose hair, which means fewer hairballs, and is a chance to " +
			"check the skin, ears and coat. Short-haired cats need brushing about once a week, long-haired cats " +
			"daily."},
		{"vet", "Vet care", "Take your cat for a check-up at least once a year and keep vaccinations up to date. " +
			"See a vet promptly for straining in the litter box or going outside it (an emergency in male cats), " +
			"sneezing with runny eyes, repeated vomiting, not eating, or hiding. Illnesses get worse without " +
			"treatment."},
		{"breathing", "Breathing", "Sneezing and runny eyes and nose are signs of an upper respiratory " +
			"infection, which is common in cats. Dust, smoke and ammonia from a dirty litter box irritate a cat's " +
			"airways. Keep the litter box clean and the room aired, and see a vet if the signs don't clear up."},
		{"stress", "Stress", "Cats under long-term stress may hide, over-groom, stop eating or go outside the " +
			"litter box. Keep a predictable routine, give your cat places to hide and climb, and avoid punishment, " +
			"which only adds to stress."},
	},
	SpeciesDog: {
		{"feeding", "Feeding", "Most adult dogs eat two measured meals a day of complete dog food; puppies eat " +
			"more often. Follow the label for your dog's weight and keep treats to about 10% of the day's food. " +
			"Never give chocolate, grapes or raisins, onions, garlic, or xylitol (a sweetener), which are toxic " +
			"to dogs. Eating too much too fast can upset the stomach."},
		{"water", "Water", "Dogs need fresh water available all the time, and more after exercise and in hot " +
			"weather. Wash the bowl daily."},
		{"bathroom", "Bathroom breaks", "Adult dogs need to go outside to relieve themselves at least three to " +
			"five times a day, and puppies far more often. A dog left too long will have an accident in the house; " +
			"that isn't bad behavior. Always pick up after your dog: feces spread intestinal parasites such as " +
			"roundworms to other dogs and to people. Clean up accidents promptly with an enzyme cleaner."},
		{"exercise", "Walks and exercise", "Dogs need daily walks and play; how much depends on breed, age and " +
			"health, from about 30 minutes to two hours a day. Exercise keeps the heart and muscles fit and " +
			"prevents boredom, which often leads to chewing and barking."},
		{"grooming", "Grooming", "Brush your dog regularly, from weekly for short coats to daily for long ones, " +
			"and check the ears, paws, teeth and nails as you go."},
		{"vet", "Vet care", "Take your dog for a check-up at least once a year and keep vaccinations and parasite " +
			"prevention up to date. See a vet for diarrhea or vomiting that lasts, coughing, not eating, or low " +
			"energy. Illnesses get worse without treatment."},
		{"breathing", "Breathing", "Coughing, sneezing and nasal discharge are signs of a respiratory infection. " +
			"Kennel cough spreads where dogs are kept close together, and dusty or dirty air irritates the airways. " +
			"See a vet if your dog coughs or breathes with effort."},
		{"stress", "Stress", "Stressed dogs may pace, pant, chew, bark or have accidents. Regular walks, a " +
			"predictable routine and reward-based training help; punishment makes stress and fear worse."},
	},
}

// CareGuide returns the care guide for a species
func CareGuide(species Species) []GuideTopic {
	return append([]GuideTopic(nil), careGuide[species]...)
}

// Symptom is a warning sign the pet shows, with what it means and the
// care guide topic that explains it
type Symptom struct {
	Sign  string `json:"sign"`
	Hint  string `json:"hint"`
	Topic string `json:"topic"`
	Good  bool   `json:"good"` // Good news, such as a recovery
}

// illnessSigns describe each illness as the owner would see it
var illnessSigns = map[string]map[Species]Symptom{
	IllnessUrinary: {
		SpeciesCat: {"Straining and making frequent trips to the litter box",
			"Signs of a urinary tract infection, which gets worse until treated. Give medical care (Heal).", "litter", false},
		SpeciesDog: {"Needing to go out far more often",
			"Signs of a urinary tract infection, which gets worse until treated. Give medical care (Heal).", "vet", false},
	},
	IllnessParasites: {
		SpeciesCat: {"Diarrhea and losing weight",
			"Signs of intestinal parasites. Give medical care (Heal), then keep the litter box clean.", "vet", false},
		SpeciesDog: {"Diarrhea and losing weight",
			"Signs of intestinal parasites, often picked up from feces. Give medical care (Heal), then keep the " +
				"home clean.", "bathroom", false},
	},
	IllnessRespiratory: {
		SpeciesCat: {"Sneezing, with runny eyes and nose",
			"Signs of an upper respiratory infection, which gets worse until treated. Give medical care (Heal).",
			"breathing", false},
		SpeciesDog: {"Coughing and sneezing",
			"Signs of a respiratory infection, which gets worse until treated. Give medical care (Heal).",
			"breathing", false},
	},
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
	species := p.speciesLocked()
	ph := p.Biology.Physiology

	if p.Illness != nil {
		symptoms = append(symptoms, illnessSigns[p.Illness.Kind][species])
	}
	age := p.Biology.GetAgeInDays()
	for i := len(p.HealthEvents) - 1; i >= 0; i-- {
		if e := p.HealthEvents[i]; age-e.Age <= recentHealthEvent {
			symptoms = append(symptoms, Symptom{Sign: e.Text, Topic: e.Topic, Good: e.Good})
		}
	}

	if p.LivingAreaWaste >= 0.5 {
		if species == SpeciesDog {
			symptoms = append(symptoms, Symptom{"There's a mess in the house",
				"Clean up promptly, and take your dog out more often.", "bathroom", false})
		} else {
			symptoms = append(symptoms, Symptom{"The litter box smells",
				"Scoop the litter box at least once a day. A dirty box can make your cat ill.", "litter", false})
		}
	}
	if species == SpeciesDog && max(ph.Bowel, ph.Bladder) >= 0.5 {
		symptoms = append(symptoms, Symptom{"Waiting by the door",
			"Your dog needs to go outside. Take it for a walk.", "bathroom", false})
	}
	if ph.RespiratoryHealth < 0.7 && (p.Illness == nil || p.Illness.Kind != IllnessRespiratory) {
		symptoms = append(symptoms, Symptom{"Sneezing now and then",
			"Foul air irritates the airways. Keep the living area clean.", "breathing", false})
	}
	if ph.CardiovascularHealth < 0.5 {
		hint := "Not enough exercise. Daily play keeps the heart fit."
		if species == SpeciesDog {
			hint = "Not enough exercise. Daily walks and play keep the heart fit."
		}
		symptoms = append(symptoms, Symptom{"Tires quickly", hint, "exercise", false})
	}
	if ph.NervousHealth < 0.6 {
		sign := "Hiding and over-grooming"
		if species == SpeciesDog {
			sign = "Pacing and restless"
		}
		symptoms = append(symptoms, Symptom{sign, "Signs of long-term stress. A calm routine, play and rest help.", "stress", false})
	}
	if ph.DigestiveHealth < 0.6 && (p.Illness == nil || p.Illness.Kind != IllnessParasites) {
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

// dailyRoutines are what an owner does each day for each species
var dailyRoutines = map[Species][]checklistItem{
	SpeciesCat: {
		{"meals", "Feed two meals", 2, types.InteractionFeeding},
		{"water", "Give fresh water", 1, types.InteractionDrinking},
		{"litter", "Scoop the litter box", 1, types.InteractionCleaning},
		{"play", "Play together", 1, types.InteractionPlaying},
		{"brush", "Brush the coat", 1, types.InteractionGrooming},
	},
	SpeciesDog: {
		{"meals", "Feed two meals", 2, types.InteractionFeeding},
		{"water", "Give fresh water", 1, types.InteractionDrinking},
		{"walks", "Take three walks", 3, types.InteractionWalking},
		{"play", "Play together", 1, types.InteractionPlaying},
	},
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
	for _, item := range dailyRoutines[p.speciesLocked()] {
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
	for _, item := range dailyRoutines[p.speciesLocked()] {
		done := 0
		if p.Checklist != nil && p.Checklist.Day == today() {
			done = min(p.Checklist.Done[item.ID], item.Target)
		}
		items = append(items, ChecklistItem{Label: item.Label, Done: done, Target: item.Target})
	}
	return items
}
