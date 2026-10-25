package core

import (
	"slices"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
)

// Themes of the week: a care theme each real week, in a fixed rotation of
// 12. Each highlights a care guide topic, makes the week's goal match it
// when it can, and brings one home happening about it. Hydration week and
// paw care week swap places when that puts hydration in summer or paw care
// out of summer and into winter, for the household's hemisphere.

// Theme is a theme of the week
type Theme struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Topic string `json:"topic"` // A care guide topic
	Text  string `json:"text"`

	weekly     string   // The weekly goal kind it prefers, if any
	happenings []string // Its happenings, the first that fits comes
}

// themes are the themes, in rotation order
var themes = []Theme{
	{ID: "dental", Name: "Dental week", Icon: "🦷", Topic: "teeth", weekly: WeeklyTeethDays, happenings: []string{"wont_eat"},
		Text: "Most pets have some dental disease by middle age. Brush a little every day."},
	{ID: "enrichment", Name: "Enrichment week", Icon: "🧩", Topic: "exercise", weekly: WeeklyPlays, happenings: []string{"new_toy", "box", "birds"},
		Text: "New things to explore, sniff and solve keep a pet's mind healthy."},
	{ID: "grooming", Name: "Grooming week", Icon: "🪮", Topic: "grooming", happenings: []string{"nails", "tangle", "hairball"},
		Text: "Regular brushing keeps the coat healthy and lets you spot lumps, ticks and sore skin early."},
	{ID: "hydration", Name: "Hydration week", Icon: "💧", Topic: "water", happenings: []string{"water_bowl"},
		Text: "Fresh water every day, in a clean bowl, keeps the kidneys healthy."},
	{ID: "training", Name: "Training week", Icon: "🎓", Topic: "training", weekly: WeeklyTricks, happenings: []string{"doorbell", "jumping", "barking"},
		Text: "Short, rewarding training sessions build trust and good habits."},
	{ID: "senior", Name: "Senior care week", Icon: "🧓", Topic: "senior", happenings: []string{"senior_stairs"},
		Text: "Older pets need gentler exercise, softer beds and more frequent checkups."},
	{ID: "calm", Name: "Calm & comfort week", Icon: "🕯️", Topic: "stress", happenings: []string{"thunderstorm", "hiding", "sunbeam"},
		Text: "A safe hiding place, routine and quiet help pets cope with stress."},
	{ID: "explorer", Name: "Explorer week", Icon: "🧭", Topic: "outdoors", weekly: WeeklyTrips, happenings: []string{"bee", "gift"},
		Text: "Time outdoors is good for body and mind, done safely."},
	{ID: "friendship", Name: "Friendship week", Icon: "🤝", Topic: "company", happenings: []string{"visitor_pet", "kneading"},
		Text: "Pets need company. Introduce new friends slowly."},
	{ID: "nutrition", Name: "Nutrition week", Icon: "🥣", Topic: "feeding", weekly: WeeklyGoalDays, happenings: []string{"counter", "sick"},
		Text: "Measured meals of complete food, and few treats, keep a pet at a healthy weight."},
	{ID: "safety", Name: "Safety at home week", Icon: "🏠", Topic: "safety", happenings: []string{"chocolate", "string", "plant"},
		Text: "Many everyday things are dangerous to pets. Check your home for them."},
	{ID: "paws", Name: "Paw care week", Icon: "🐾", Topic: "paws", happenings: []string{"mud", "nails", "salt"},
		Text: "Check paws, trim nails and protect pads from heat, salt and ice."},
}

// themeGuide are the care guide topics the themes add
func themeGuide(species Species) []GuideTopic {
	cat := species == SpeciesCat
	training := "Reward the behavior you want straight away, with a treat, praise or play, so your pet knows exactly " +
		"what earned it. Keep sessions short (five to ten minutes) and end on a success. Punishment makes pets " +
		"fearful and slows learning."
	if cat {
		training += " Cats learn too: use tiny treats and a clicker, and train before meals."
	}
	outdoors := "Keep dogs on a lead near roads, farm animals and wildlife, and teach a reliable recall before " +
		"letting them off. Bring water, avoid the midday heat, check for ticks after walks in long grass, and keep " +
		"vaccinations and parasite treatment up to date."
	if cat {
		outdoors = "Most vets advise keeping cats indoors or giving them supervised time outside, on a harness or " +
			"in an enclosed garden or catio. Get your cat used to the harness indoors first, and check for ticks " +
			"after time in long grass."
	}
	company := "Dogs are social and suffer when left alone for long hours; plan company, walks and play every day."
	if cat {
		company = "Cats are more independent than dogs but still need daily attention and play, and some are " +
			"lonely when left alone for long."
	}
	company += " Introduce pets to each other slowly, with separate food, water and resting places, and let them " +
		"get used to each other's smell first."
	return []GuideTopic{
		{"training", "Training", training},
		{"outdoors", "Outdoors safely", outdoors},
		{"company", "Company and friends", company},
		{"safety", "A safe home", "Keep chocolate, grapes and raisins, onions, garlic, xylitol and medicines out of " +
			"reach. Choose pet-safe plants (lilies are deadly to cats). Put away string, elastic bands and small things " +
			"that can be swallowed, keep cleaning products shut away, and keep the vet's number handy."},
		{"paws", "Paw care", "Check paws after walks for cuts, thorns and grit. Trim nails before they touch the " +
			"floor, avoiding the pink quick. Wipe off salt and de-icers in winter, and keep off hot pavement in summer."},
	}
}

// themeIndex returns a theme's place in the rotation
func themeIndex(id string) int {
	return slices.IndexFunc(themes, func(t Theme) bool { return t.ID == id })
}

// themeFor returns the theme of the week holding t
func themeFor(t time.Time) Theme {
	_, n := t.ISOWeek()
	i := n % len(themes)
	th := themes[i]
	if th.ID != "hydration" && th.ID != "paws" {
		return th
	}
	// Where the other of the two falls in this round of the rotation
	h, p := themeIndex("hydration"), themeIndex("paws")
	at := func(j int) environment.Season {
		return environment.SeasonOn(t.AddDate(0, 0, 7*(j-i)), Hemisphere())
	}
	if at(h) == environment.SeasonWinter || at(p) == environment.SeasonSummer {
		if th.ID == "hydration" {
			return themes[p]
		}
		return themes[h]
	}
	return th
}

// WeekTheme returns this week's theme
func WeekTheme() Theme {
	return themeFor(now())
}

// ThemeWeek identifies the current week, as households record the week
// they had its themed happening in
func ThemeWeek() string {
	return weekID(now())
}

// themeHappeningLocked returns the week's themed happening if neither the
// pet nor (themeDone) another pet of its player has had it this week and
// one fits. Caller must hold p.mu.
func (p *DigitalPet) themeHappeningLocked(themeDone bool) (happeningDef, bool) {
	week := weekID(now())
	if themeDone || p.Marks["theme_happening"] == week {
		return happeningDef{}, false
	}
	for _, id := range WeekTheme().happenings {
		if h, ok := findHappening(id); ok && p.happeningFitsLocked(h) {
			if p.Marks == nil {
				p.Marks = make(map[string]string)
			}
			p.Marks["theme_happening"] = week
			return h, true
		}
	}
	return happeningDef{}, false
}
