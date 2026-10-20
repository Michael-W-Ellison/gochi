package core

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/internal/genetics"
)

func TestFavoriteSeasonFollowsPersonality(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	traits := pet.Personality.Traits
	traits.Curiosity, traits.EnergyLevel, traits.Adaptability, traits.Affectionate = 0.2, 0.9, 0.3, 0.3
	if got := pet.EnvironmentStatus().FavoriteSeason; got != "Summer" {
		t.Errorf("An energetic pet should love summer, got %s", got)
	}
	traits.EnergyLevel, traits.Affectionate = 0.1, 0.95
	if got := pet.EnvironmentStatus().FavoriteSeason; got != "Winter" {
		t.Errorf("A cuddly, low-energy pet should love winter, got %s", got)
	}
}

func TestSeasonPreference(t *testing.T) {
	summer := environment.SeasonSummer
	for season, want := range map[environment.Season]float64{
		environment.SeasonSummer: favoriteSeasonMood,
		environment.SeasonWinter: oppositeSeasonMood,
		environment.SeasonSpring: 0,
		environment.SeasonFall:   0,
	} {
		if got := seasonPreference(summer, season); got != want {
			t.Errorf("A summer lover in %s: %v, want %v", season, got, want)
		}
	}
}

func TestFavoriteSeasonLiftsMood(t *testing.T) {
	summerLover, springLover := NewDigitalPet("Biscuit", "user123"), NewDigitalPet("Mochi", "user123")
	t1, t2 := summerLover.Personality.Traits, springLover.Personality.Traits
	t1.Curiosity, t1.EnergyLevel, t1.Adaptability, t1.Affectionate = 0.2, 0.9, 0.3, 0.3
	t2.Curiosity, t2.EnergyLevel, t2.Adaptability, t2.Affectionate = 0.9, 0.5, 0.3, 0.3

	// The same surroundings, so only the pets' feelings about summer differ
	summerLover.Environment.ForceSeason(environment.SeasonSummer, 10)
	springLover.Environment = summerLover.Environment
	lover, other := summerLover.moodModifierLocked(), springLover.moodModifierLocked()
	if math.Abs(lover-other-favoriteSeasonMood) > 1e-9 {
		t.Errorf("Summer should lift a summer lover's mood by %v more: %.3f vs %.3f", favoriteSeasonMood, lover, other)
	}

	// A summer lover feels winter more than a spring lover does
	summerLover.Environment.ForceSeason(environment.SeasonWinter, 10)
	lover, other = summerLover.moodModifierLocked(), springLover.moodModifierLocked()
	if math.Abs(lover-other-oppositeSeasonMood) > 1e-9 {
		t.Errorf("Winter should lower a summer lover's mood by %v more: %.3f vs %.3f", -oppositeSeasonMood, lover, other)
	}
}

func TestBreedingSeason(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Environment.ForceSeason(environment.SeasonSpring, 10)
	if !pet.EnvironmentStatus().BreedingSeason || pet.seasonalBreedingLocked() <= 1 {
		t.Errorf("Spring should be breeding season, modifier %.2f", pet.seasonalBreedingLocked())
	}
	pet.Environment.ForceSeason(environment.SeasonWinter, 10)
	if pet.EnvironmentStatus().BreedingSeason || pet.seasonalBreedingLocked() >= 1 {
		t.Errorf("Winter should not be breeding season, modifier %.2f", pet.seasonalBreedingLocked())
	}
}

func TestBabiesPerPairAreLimited(t *testing.T) {
	a, b := adultPet("Biscuit"), adultPet("Mochi")
	for i := 0; i < maxBabiesPerPair; i++ {
		breedUntilSuccess(t, a, b, "Baby")
		a.LastBredAge, b.LastBredAge = 0, 0 // Skip the cooldown
	}
	if a.BabiesWith[b.ID] != maxBabiesPerPair || b.BabiesWith[a.ID] != maxBabiesPerPair {
		t.Fatalf("Both parents should count their babies: %v, %v", a.BabiesWith, b.BabiesWith)
	}

	want := "Biscuit and Mochi have already had 5 babies together"
	if _, reason := BreedingCompatibility(a, b); reason != want {
		t.Errorf("Compatibility reason = %q", reason)
	}
	var be BreedingError
	if _, err := Breed(a, b, "Baby"); !errors.As(err, &be) || be.Reason != want {
		t.Errorf("Breed = %v", err)
	}

	// Either parent's record is enough, and other partners are unaffected
	b.BabiesWith = nil
	if _, err := Breed(a, b, "Baby"); err == nil || !strings.Contains(err.Error(), "5 babies") {
		t.Errorf("One parent's record should be enough: %v", err)
	}
	c := adultPet("Pepper")
	if _, reason := BreedingCompatibility(a, c); strings.Contains(reason, "babies") {
		t.Errorf("Another partner should not be limited: %q", reason)
	}
}

func TestBreedingFailsMoreInWinter(t *testing.T) {
	successes := func(season environment.Season) int {
		count := 0
		for i := 0; i < 100; i++ {
			a, b := adultPet("A"), adultPet("B")
			a.Environment.ForceSeason(season, 10)
			b.Environment.ForceSeason(season, 10)
			_, err := Breed(a, b, "Baby")
			var be BreedingError
			switch {
			case err == nil:
				count++
			case errors.As(err, &be) && be.Reason == genetics.FailureChance:
			case errors.As(err, &be): // Incompatible pair; not counted
			default:
				t.Fatal(err)
			}
		}
		return count
	}
	if spring, winter := successes(environment.SeasonSpring), successes(environment.SeasonWinter); spring <= winter {
		t.Errorf("More babies should come in spring (%d) than winter (%d)", spring, winter)
	}
}
