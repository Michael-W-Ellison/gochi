package core

import (
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// spriteStates are the sprites the page needs for each species
var spriteStates = []string{"idle", "happy", "eating", "playing", "sleeping", "sick", "exploring", "social", "hibernating", "dead"}

func TestRegisteredSpeciesAreComplete(t *testing.T) {
	list := SpeciesList()
	if len(list) < 2 || list[0].ID != defaultSpecies {
		t.Fatalf("The default species should be listed first: %v", list)
	}
	for _, def := range list {
		for _, kind := range []string{IllnessUrinary, IllnessParasites, IllnessRespiratory, IllnessPoisoning} {
			if def.IllnessNames[kind] == "" || def.IllnessSigns[kind].Sign == "" {
				t.Errorf("%s has no name or sign for illness %s", def.ID, kind)
			}
		}
		for _, state := range spriteStates {
			if def.Sprites[state] == "" {
				t.Errorf("%s has no %s sprite", def.ID, state)
			}
		}
		if s, ok := findSupply(def.Treats); !ok || !s.forSpecies(def.ID) || s.kind != SupplyTreat {
			t.Errorf("%s's treats %q aren't a treat for it", def.ID, def.Treats)
		}
		if s, ok := findSupply(def.BasicFood.item.ID); !ok || s.food == nil {
			t.Errorf("%s's basic food can't be found", def.ID)
		}
		if def.IntroductionLessons[def.ID] == "" {
			t.Errorf("%s has no lesson for meeting its own kind", def.ID)
		}
		for _, activity := range []string{GroupPlay, GroupNap} {
			if def.GroupLessons[activity] == "" {
				t.Errorf("%s has no lesson for %s", def.ID, activity)
			}
		}
		if def.Walks && def.GroupLessons[GroupWalk] == "" {
			t.Errorf("%s goes for walks but has no group walk lesson", def.ID)
		}
		if !def.Walks && def.NoWalks == "" {
			t.Errorf("%s doesn't go for walks but doesn't say why", def.ID)
		}
		if w := def.SleepWindow; w[0] < 0 || w[0] >= 24 || w[1] < 0 || w[1] >= 24 {
			t.Errorf("%s's sleep window %v isn't in hours", def.ID, w)
		}
		if def.MessRequest == "" || def.CleanRefusal == "" || def.ConflictLesson == "" || def.LeaderText == "" ||
			len(def.FriendlySignals) == 0 || len(def.WarySignals) == 0 || def.CleanLabel == "" {
			t.Errorf("%s is missing text", def.ID)
		}
	}
}

// registerRabbit registers a test species made from the cat's
// definition, for the duration of a test
func registerRabbit(t *testing.T) {
	t.Helper()
	def := *speciesRegistry[SpeciesCat]
	def.ID, def.Name, def.Plural, def.Icon = "rabbit", "Rabbit", "rabbits", "🐰"
	def.LivingArea, def.MessRequest = "Hutch", "%s's hutch needs cleaning."
	def.SameSpeciesMeeting = 0.2
	def.IntroductionLessons = map[Species]string{"rabbit": "Introduce rabbits on neutral ground."}
	def.BasicFood = supply{item: interaction.Item{ID: "rabbit_hay", Name: "Hay", Type: interaction.ItemTypeFood},
		icon: "🌾", kind: SupplyFood, food: &foodProfile{nutrition: 1, lesson: "Rabbits need hay all day."}}
	def.Treats = "rabbit_treats"
	def.Supplies = []supply{{item: interaction.Item{ID: "rabbit_treats", Name: "Rabbit treats", Price: 5,
		Rarity: interaction.RarityCommon, Type: interaction.ItemTypeFood}, icon: "🥕", kind: SupplyTreat, pack: 10}}
	RegisterSpecies(&def)
	t.Cleanup(func() { delete(speciesRegistry, "rabbit") })
}

func TestNewSpeciesFromOneDefinition(t *testing.T) {
	registerRabbit(t)
	if s, ok := ParseSpecies(" Rabbit "); !ok || s != "rabbit" {
		t.Fatalf("ParseSpecies: %q %v", s, ok)
	}

	pet := NewDigitalPet("Thumper", "user123")
	pet.Species = "rabbit"
	if pet.GetSpecies() != "rabbit" || pet.GetLivingArea().Name != "Hutch" {
		t.Errorf("Species %q, living area %+v", pet.GetSpecies(), pet.GetLivingArea())
	}
	pet.LivingAreaWaste = 0.6
	if r := pet.CurrentRequest(); r == nil || r.Message != "Thumper's hutch needs cleaning." {
		t.Errorf("Request: %+v", r)
	}

	// Its own supplies, not the cat's
	var ids []string
	for _, offer := range pet.SupplyOffers() {
		ids = append(ids, offer.ID)
	}
	joined := strings.Join(ids, " ")
	if !strings.Contains(joined, "rabbit_treats") || strings.Contains(joined, "cat_treats") {
		t.Errorf("A rabbit's shop: %v", ids)
	}
	pet.Biology.Vitals.Nutrition = 0.3
	if r, err := pet.Feed("rabbit_hay"); err != nil || !r.Applied || !strings.Contains(r.Lesson, "hay") {
		t.Errorf("Feeding hay: %+v %v", r, err)
	}

	// Meetings with its own kind and with others
	if speciesMeeting("rabbit", "rabbit") != 0.2 || speciesMeeting("rabbit", SpeciesDog) != crossSpeciesMeeting {
		t.Error("Meeting odds should come from the definition")
	}
	if introductionLesson("rabbit", SpeciesDog) == "" || introductionLesson("rabbit", "rabbit") != "Introduce rabbits on neutral ground." {
		t.Error("Introduction lessons should come from the definition, with a general one between other species")
	}
	other := NewDigitalPet("Flopsy", "user123")
	other.Species = "rabbit"
	setCalmHealthyState(pet)
	setCalmHealthyState(other)
	if _, err := Introduce(pet, other); err != nil {
		t.Errorf("Two rabbits should be able to meet: %v", err)
	}
	if r := pet.Interact(types.InteractionWalking, 1.0); r.Applied || !strings.Contains(r.Refusal, "Thumper") {
		t.Errorf("A rabbit based on a cat doesn't go for walks: %+v", r)
	}
}

func TestUnknownSpeciesActsAsDefault(t *testing.T) {
	pet := NewDigitalPet("Mystery", "user123")
	pet.Species = "dragon"
	if pet.GetSpecies() != defaultSpecies {
		t.Errorf("A species that isn't registered should act as the default, got %q", pet.GetSpecies())
	}
	if _, ok := ParseSpecies("dragon"); ok {
		t.Error("ParseSpecies should reject unknown species")
	}
}

func TestRegisterSpeciesRejectsBadDefinitions(t *testing.T) {
	for name, def := range map[string]*SpeciesDef{
		"duplicate":  speciesRegistry[SpeciesCat],
		"incomplete": {ID: "fish", Name: "Fish"},
		"upper case": {ID: "Fish"},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Registering a %s species should panic", name)
				}
			}()
			RegisterSpecies(def)
		}()
	}
}

func TestValidPetName(t *testing.T) {
	for _, good := range []string{"Mochi", "Mr. Biscuit", " Rex ", "Ünïcödé", strings.Repeat("a", MaxPetNameLength)} {
		if !ValidPetName(good) {
			t.Errorf("%q should be valid", good)
		}
	}
	for _, bad := range []string{"", "   ", strings.Repeat("a", MaxPetNameLength+1), "line\nbreak", "tab\there", "​zero"} {
		if ValidPetName(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
