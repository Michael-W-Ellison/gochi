package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Species is the kind of animal a pet is. Care differs between species,
// and the game teaches each one's real-world needs.
type Species string

const (
	SpeciesCat Species = "cat"
	SpeciesDog Species = "dog"
)

// defaultSpecies is the species of pets saved before species existed
const defaultSpecies = SpeciesCat

// Species are registered: everything that differs between species is in
// its SpeciesDef, so a new kind of pet is added with one definition (see
// species_cat.go and species_dog.go) registered with RegisterSpecies.

// ToiletStyle is how a species relieves itself
type ToiletStyle int

const (
	// ToiletLitterBox: the pet uses a box in the home that the owner
	// scoops, and avoids it when it is too dirty
	ToiletLitterBox ToiletStyle = iota
	// ToiletOutside: the pet goes outside on walks, and has an accident
	// in the home if it has to wait too long
	ToiletOutside
)

// SpeciesDef is everything that makes a species different
type SpeciesDef struct {
	ID     Species
	Name   string // "Cat"
	Plural string // "cats"

	// How the page shows it: an icon, sprites by state (idle, happy,
	// eating, playing, sleeping, sick, exploring, social, hibernating,
	// dead), and the label of the clean-up button
	Icon       string
	Sprites    map[string]string
	CleanLabel string

	// Body
	LifespanFactor float64    // Multiplies the lifespan
	SleepWindow    [2]float64 // Night sleep, from and to, in game hours
	TwilightActive bool       // Most active at dawn and dusk
	SelfGrooming   float64    // Added to how much it wants to groom itself
	GroomingRate   float64    // How well it cleans itself, relative to a cat

	// Home and toilet
	Toilet       ToiletStyle
	LivingArea   string  // What gets dirty, such as "Litter box"
	MessRequest  string  // The request when it is dirty; %s is the pet's name
	MessSymptom  Symptom // The warning sign when it is dirty
	CleanRefusal string  // Why cleaning is refused when there's nothing to clean
	DirtIllness  string  // The illness a dirty home causes
	Walks        bool    // Goes for walks
	NoWalks      string  // Why a walk is refused if it doesn't; %s is the pet's name

	// Teaching
	ExerciseHint            string // Advice when its heart is unfit
	StressSign              string // How long-term stress shows
	CareLessons             map[types.InteractionType][]string
	CareGuide               []GuideTopic
	DailyRoutine            []checklistItem
	IllnessNames            map[string]string  // By illness kind
	IllnessSigns            map[string]Symptom // By illness kind
	UnwantedAttentionLesson string

	// Care
	Liking             map[types.InteractionType]float64 // How much more it enjoys kinds of care
	ReactionTexts      map[types.InteractionType]map[ReactionLevel]string
	PetRegions         []petRegion // Favorite spots first
	EnvironmentChanges []environmentChange
	BasicFood          supply   // Free and always in the pantry
	Treats             string   // The supply it is trained with
	Supplies           []supply // Supplies for this species only, in shop order

	// Social
	SameSpeciesMeeting  float64            // How well two of them get along at first
	IntroductionLessons map[Species]string // How to introduce it to each species, its own included
	ThreatSignal        string             // "hissed at"
	FriendlySignals     []string
	WarySignals         []string
	CrowdSensitive      bool              // Territorial pets mind groups of more than two
	LeaderText          string            // Who leads a group; %s is its name
	GroupLessons        map[string]string // By group activity
	ConflictLesson      string
}

// SpriteStates are the states a species has a sprite for. Another
// player's game is told only which of these its pet shows.
var SpriteStates = []string{"idle", "happy", "eating", "playing", "sleeping", "sick", "exploring", "social",
	"hibernating", "dead"}

// Meetings between different species start this badly
const crossSpeciesMeeting = -0.15

// speciesRegistry holds the registered species
var speciesRegistry = map[Species]*SpeciesDef{}

// RegisterSpecies adds a species. It panics if the definition is
// incomplete or the species is already registered, since that is a
// programming error.
func RegisterSpecies(def *SpeciesDef) {
	switch {
	case def.ID == "" || def.ID != Species(strings.ToLower(string(def.ID))):
		panic("species ID must be lower case and not empty")
	case speciesRegistry[def.ID] != nil:
		panic(fmt.Sprintf("species %q registered twice", def.ID))
	case def.Name == "" || def.Plural == "" || def.Icon == "" || def.LivingArea == "" || def.BasicFood.item.ID == "":
		panic(fmt.Sprintf("species %q is missing its name, icon, living area or basic food", def.ID))
	case len(def.DailyRoutine) == 0 || len(def.PetRegions) == 0 || def.Treats == "":
		panic(fmt.Sprintf("species %q needs a daily routine, petting regions and treats", def.ID))
	}
	def.BasicFood.species = []Species{def.ID}
	for i := range def.Supplies {
		def.Supplies[i].species = []Species{def.ID}
	}
	speciesRegistry[def.ID] = def
}

// SpeciesList returns the registered species, the default first and the
// rest by ID
func SpeciesList() []*SpeciesDef {
	list := make([]*SpeciesDef, 0, len(speciesRegistry))
	for _, def := range speciesRegistry {
		list = append(list, def)
	}
	sort.Slice(list, func(i, j int) bool {
		if (list[i].ID == defaultSpecies) != (list[j].ID == defaultSpecies) {
			return list[i].ID == defaultSpecies
		}
		return list[i].ID < list[j].ID
	})
	return list
}

// Def returns the species' definition; unknown species get the default's
func (s Species) Def() *SpeciesDef {
	if def := speciesRegistry[s]; def != nil {
		return def
	}
	return speciesRegistry[defaultSpecies]
}

// Known reports whether the species is registered
func (s Species) Known() bool {
	return speciesRegistry[s] != nil
}

// ParseSpecies returns the species with the given name ("" means the
// default, a cat), or false if there is no such species
func ParseSpecies(name string) (Species, bool) {
	s := Species(strings.ToLower(strings.TrimSpace(name)))
	if s == "" {
		return defaultSpecies, true
	}
	return s, s.Known()
}

// speciesLocked returns the pet's species; pets saved before species
// existed, or of a species no longer registered, are cats. Caller must
// hold p.mu.
func (p *DigitalPet) speciesLocked() Species {
	if p.Species.Known() {
		return p.Species
	}
	return defaultSpecies
}

// defLocked returns the pet's species definition. Caller must hold p.mu.
func (p *DigitalPet) defLocked() *SpeciesDef {
	return p.speciesLocked().Def()
}

// GetSpecies returns the pet's species
func (p *DigitalPet) GetSpecies() Species {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.speciesLocked()
}

// Plural returns the species' plural name, such as "cats"
func (s Species) Plural() string {
	return s.Def().Plural
}

// speciesMeeting is how well a pairing of species tends to get along at
// first
func speciesMeeting(a, b Species) float64 {
	if a == b {
		return a.Def().SameSpeciesMeeting
	}
	return crossSpeciesMeeting
}

// introductionLesson teaches how to introduce a pairing of species
func introductionLesson(a, b Species) string {
	if lesson, ok := a.Def().IntroductionLessons[b]; ok {
		return lesson
	}
	if lesson, ok := b.Def().IntroductionLessons[a]; ok {
		return lesson
	}
	return "Introduce pets slowly, with short, supervised meetings and a way for each to retreat."
}

// withName puts a pet's name into a species text where it has %s
func withName(text, name string) string {
	return strings.ReplaceAll(text, "%s", name)
}

// capitalize returns text with its first letter in upper case
func capitalize(text string) string {
	if text == "" {
		return text
	}
	return strings.ToUpper(text[:1]) + text[1:]
}
