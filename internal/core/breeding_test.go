package core

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/genetics"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// adultPet returns a healthy, happy pet old enough to breed
func adultPet(name string) *DigitalPet {
	pet := NewDigitalPetRandom(name, "user123")
	pet.Biology.Processes.Age = 100 // An adult
	pet.Biology.Vitals.Health = 1.0
	pet.Biology.Vitals.Happiness = 0.9
	return pet
}

// breedUntilSuccess retries breeding, which can fail by chance
func breedUntilSuccess(t *testing.T, a, b *DigitalPet, name string) *DigitalPet {
	t.Helper()
	for i := 0; i < 50; i++ {
		child, err := Breed(a, b, name)
		if err == nil {
			return child
		}
		var be BreedingError
		if !errors.As(err, &be) || be.Reason != genetics.FailureChance {
			t.Fatalf("Breed failed for a reason other than chance: %v", err)
		}
	}
	t.Fatal("Breeding never succeeded")
	return nil
}

func TestGenomeMatchesPersonality(t *testing.T) {
	pet := NewDigitalPetRandom("Biscuit", "user123")
	// Genes carry a little natural variation (up to 0.05) around the trait
	if got, want := geneValue(pet.Genome, genetics.TraitBasePlayfulness), pet.Personality.Traits.Playfulness; math.Abs(got-want) > 0.05 {
		t.Errorf("Playfulness gene %f should match personality %f", got, want)
	}
	if pet.Generation() != 1 {
		t.Errorf("A new pet is generation 1, got %d", pet.Generation())
	}
}

func TestBreedingRequirements(t *testing.T) {
	young := NewDigitalPet("Young", "user123")
	adult := adultPet("Adult")
	if _, reason := BreedingCompatibility(young, adult); !strings.HasPrefix(reason, "Young is too young (0.0 days old; must be at least 60)") || strings.Contains(reason, "Adult") {
		t.Errorf("Young pet: %q", reason)
	}

	senior := adultPet("Senior")
	senior.Biology.Processes.Age = 7000
	if _, reason := BreedingCompatibility(senior, adult); !strings.HasPrefix(reason, "Senior is too old (7000 days old; the limit is 6000)") {
		t.Errorf("Senior pet: %q", reason)
	}

	sleepy := adultPet("Sleepy")
	sleepy.HibernationEnabled = true
	sleepy.Hibernating = true
	if _, err := Breed(adult, sleepy, "Baby"); err == nil || err.Error() != "Sleepy is hibernating" {
		t.Errorf("Hibernating partner: %v", err)
	}

	if _, err := Breed(adult, adult, "Baby"); !errors.Is(err, ErrSamePet) {
		t.Errorf("Breeding with itself: %v", err)
	}
}

func TestBreedingMakesBaby(t *testing.T) {
	a, b := adultPet("Biscuit"), adultPet("Mochi")
	if compatibility, reason := BreedingCompatibility(a, b); reason != "" || compatibility <= 0 {
		t.Fatalf("Adult pets should be able to breed: %f %q", compatibility, reason)
	}

	baby := breedUntilSuccess(t, a, b, "Pip")
	if baby.Name != "Pip" || baby.Generation() != 2 || baby.GetAge() != 0 {
		t.Errorf("Unexpected baby: %s generation %d age %f", baby.Name, baby.Generation(), baby.GetAge())
	}
	for _, link := range genePersonality {
		if got, want := link.get(baby.Personality.Traits), geneValue(baby.Genome, link.gene); math.Abs(got-want) > 1e-9 {
			t.Errorf("Baby's %s trait %f should come from its genome %f", link.gene, got, want)
		}
	}

	// Family relationships
	if rel, ok := baby.Relationships.GetRelationship(a.ID); !ok || rel.Type != types.RelationshipParent {
		t.Error("The baby should know its parents")
	}
	if rel, ok := a.Relationships.GetRelationship(baby.ID); !ok || rel.Type != types.RelationshipOffspring {
		t.Error("A parent should know its baby")
	}

	// Cooldown, counted in game days
	want := "Biscuit had a baby recently and can breed again in 3.0 days; Mochi had a baby recently and can breed again in 3.0 days"
	if _, reason := BreedingCompatibility(a, b); reason != want {
		t.Errorf("Parents should need a rest after breeding: %q", reason)
	}
	a.Biology.Processes.Age += 3.1
	b.Biology.Processes.Age += 3.1
	if _, reason := BreedingCompatibility(a, b); reason != "" {
		t.Errorf("The cooldown should end after 3 game days: %q", reason)
	}
}

func TestGenomeSurvivesSaveLoad(t *testing.T) {
	baby := breedUntilSuccess(t, adultPet("Biscuit"), adultPet("Mochi"), "Pip")
	data, _ := baby.Save()
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Generation() != 2 || loaded.Genome.ID != baby.Genome.ID {
		t.Error("The genome should be restored from a save")
	}

	// Pets saved before genomes existed get one from their personality
	loaded.Genome = nil
	if loaded.Generation() != 1 {
		t.Error("A missing genome should be recreated")
	}
}

func TestTraitsDoNotDriftAcrossGenerations(t *testing.T) {
	playful := func(name string) *DigitalPet {
		pet := adultPet(name)
		pet.Personality.Traits.Playfulness = 0.8
		pet.Genome = nil
		pet.Generation() // Rebuild the genome from the personality
		return pet
	}

	// Each generation, the newest baby grows up and breeds with a fresh
	// partner that has the same playfulness
	a := playful("A")
	for gen := 0; gen < 5; gen++ {
		child := breedUntilSuccess(t, a, playful("Partner"), "Child")
		child.Biology.Processes.Age = 100
		child.Biology.Vitals.Health, child.Biology.Vitals.Happiness = 1, 0.9
		a = child
	}

	if p := a.Personality.Traits.Playfulness; p < 0.6 || a.Generation() < 6 {
		t.Errorf("Playfulness should stay near 0.8 over generations, got %f in generation %d", p, a.Generation())
	}
}
