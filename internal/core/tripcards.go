package core

import (
	"fmt"
	"slices"
	"strings"
)

// Trip cards: while a pet is out exploring, the page shows facts about it
// (where it is and what it has found there, its breed, personality, likes,
// tricks and age) and care tips (the outdoor lessons of the place it went,
// its breed's tendencies, the theme of the week and the care guide), so
// the player has something to read instead of only waiting.

// TripCard is one fact or tip
type TripCard struct {
	Kind  string `json:"kind"` // "fact" or "tip"
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// TripCards returns the cards to show while the pet is away: facts and
// tips in turn, starting with the place it went
func (p *DigitalPet) TripCards() []TripCard {
	p.mu.Lock()
	name, species := p.Name, p.speciesLocked()
	placeID, loc := p.tripPlaceLocked()
	placeName, placeText := placeID, ""
	if loc != nil {
		placeName, placeText = loc.Name, loc.Description
	}
	placeIcon := placeDetails[placeID].icon
	found := slices.Clone(p.Souvenirs)
	p.mu.Unlock()

	var facts, tips []TripCard
	fact := func(icon, title, text string) { facts = append(facts, TripCard{"fact", icon, title, text}) }
	tip := func(icon, title, text string) { tips = append(tips, TripCard{"tip", icon, title, text}) }

	// Where it is, and what it has found there
	var have, missing []string
	total := 0
	for _, s := range souvenirs {
		if s.Place != placeID {
			continue
		}
		total++
		if slices.Contains(found, s.ID) {
			have = append(have, s.Icon)
		} else {
			missing = append(missing, s.Name)
		}
		if s.Lesson != "" {
			tip(s.Icon, "Found at the "+strings.ToLower(placeName), s.Lesson)
		}
	}
	text := name + " is at the " + strings.ToLower(placeName) + "."
	if placeText != "" {
		text = strings.TrimSuffix(placeText, ".") + ". " + text
	}
	switch {
	case total == 0:
	case len(have) == 0:
		text += fmt.Sprintf(" There are %d souvenirs to find here, and %s hasn't found any yet.", total, name)
	case len(missing) == 0:
		text += fmt.Sprintf(" %s has found all %d souvenirs from here: %s.", name, total, strings.Join(have, " "))
	default:
		text += fmt.Sprintf(" %s has found %d of the %d souvenirs from here (%s); still to find: %s.", name,
			len(have), total, strings.Join(have, " "), strings.Join(missing, ", "))
	}
	fact(placeIcon, "At the "+strings.ToLower(placeName), text)

	// The place's outdoor lessons, from its trip events
	for _, e := range tripEvents {
		if slices.Contains(e.places, placeID) && (e.species == "" || e.species == species) && e.lesson != "" {
			tip(e.icon, e.title, e.lesson)
		}
	}

	// The pet itself
	info := p.BreedInfo()
	b := info.Breed
	if b.Mixed {
		fact(species.Def().Icon, "A mixed breed", name+" is a mixed breed, so its looks and temperament come from "+
			"its parents' breeds. Mixed breeds tend to have fewer inherited health problems.")
	} else if b.Name != "" {
		fact(species.Def().Icon, b.Name, b.About+" "+name+" is "+strings.ToLower(b.Temperament)+".")
	}
	for _, t := range info.Tendencies {
		tip("🩺", t.Name, t.Lesson)
	}
	if d := p.PersonalityDescription(); d != "" {
		fact("💭", "Personality", d)
	}
	if likes, dislikes := p.LikesAndDislikes(); len(likes) > 0 || len(dislikes) > 0 {
		var parts []string
		if len(likes) > 0 {
			parts = append(parts, name+" likes "+strings.Join(likes, ", ")+".")
		}
		if len(dislikes) > 0 {
			parts = append(parts, name+" doesn't like "+strings.Join(dislikes, ", ")+".")
		}
		fact("❤️", "Likes and dislikes", strings.Join(parts, " ")+" Care it likes cheers it up more.")
	}
	var learned []string
	next := ""
	for _, t := range p.Tricks() {
		switch {
		case t.Learned:
			learned = append(learned, t.Name)
		case next == "" && t.Locked == "":
			next = t.Name
			if t.Lesson != "" {
				tip(t.Icon, "Teaching "+t.Name, t.Lesson)
			}
		}
	}
	trickText := name + " hasn't learned a trick yet."
	if len(learned) > 0 {
		trickText = name + " knows " + strings.Join(learned, ", ") + "."
	}
	if next != "" {
		trickText += " Next to learn: " + next + ", with 🎓 Train when " + name + " is home."
	}
	fact("🎓", "Tricks", trickText)
	if life := p.LifeStatus(); life.Lifespan > 0 {
		fact("🎂", "Age", fmt.Sprintf("%s is %.0f game days old, %s. %s can expect to live about %.0f game days; "+
			"good care helps.", name, life.Age, aStage(life.Stage), name, life.Lifespan))
	}

	// Care tips
	theme := WeekTheme()
	tip(theme.Icon, "This week: "+theme.Name, theme.Text)
	for _, g := range CareGuide(species) {
		tip("📖", g.Title, g.Text)
	}

	// Facts and tips in turn
	cards := make([]TripCard, 0, len(facts)+len(tips))
	for i := 0; i < max(len(facts), len(tips)); i++ {
		if i < len(facts) {
			cards = append(cards, facts[i])
		}
		if i < len(tips) {
			cards = append(cards, tips[i])
		}
	}
	return cards
}

// aStage names a life stage in a sentence: "a young pet", "an adult", "a
// senior"
func aStage(stage string) string {
	switch strings.ToLower(stage) {
	case "young":
		return "still young"
	case "adult":
		return "an adult"
	case "senior":
		return "a senior"
	}
	return strings.ToLower(stage)
}
