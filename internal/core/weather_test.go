package core

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// withWeather sets a fine summer middday, then the given weather
func withWeather(pet *DigitalPet, weather environment.WeatherType, comfort float64) {
	setCalmHealthyState(pet)
	pet.Environment.ForceWeather(weather)
	pet.Environment.Weather.Current.Comfort = comfort
}

func TestWeatherChangesHowActivePetsAre(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	withWeather(pet, environment.WeatherClear, 0.9)
	clear := pet.activityDriveLocked()
	withWeather(pet, environment.WeatherHot, 0.4)
	if hot := pet.activityDriveLocked(); hot >= clear {
		t.Errorf("Heat should make a pet less active: %.2f vs %.2f", hot, clear)
	}

	// Cats are most active at dawn and dusk; dogs slow down after dark
	withWeather(pet, environment.WeatherClear, 0.9)
	pet.Environment.SetTimeOfDay(19.5)
	if pet.Environment.GetDayPhase() == "Dusk" && pet.activityDriveLocked() <= clear {
		t.Error("A cat should be livelier at dusk")
	}
	dog := NewDigitalPet("Rex", "user123")
	dog.Species = SpeciesDog
	withWeather(dog, environment.WeatherClear, 0.9)
	day := dog.activityDriveLocked()
	dog.Environment.SetTimeOfDay(2)
	if dog.activityDriveLocked() >= day {
		t.Error("A dog should be less active at night")
	}

	// A playful pet plays in fine weather but not in a heatwave
	pet.Personality.Traits.Playfulness, pet.Personality.Traits.EnergyLevel = 0.75, 0.6
	withWeather(pet, environment.WeatherClear, 0.9)
	pet.Environment.ForceSeason(environment.SeasonSpring, 10)
	fine := pet.Personality.GetTraitInfluence("play") * pet.activityDriveLocked()
	withWeather(pet, environment.WeatherHot, 0.3)
	hot := pet.Personality.GetTraitInfluence("play") * pet.activityDriveLocked()
	if hot >= fine {
		t.Errorf("Play should appeal less in the heat: %.2f vs %.2f", hot, fine)
	}
}

func TestStormsDrawNervousPetsToTheirOwners(t *testing.T) {
	nervous, calm := NewDigitalPet("A", "u"), NewDigitalPet("B", "u")
	nervous.Personality.Traits.Neuroticism, calm.Personality.Traits.Neuroticism = 0.9, 0.1
	for _, pet := range []*DigitalPet{nervous, calm} {
		withWeather(pet, environment.WeatherStormy, 0.2)
	}
	if nervous.comfortSeekingLocked() <= calm.comfortSeekingLocked() || calm.comfortSeekingLocked() <= 0 {
		t.Errorf("Storms: nervous %.2f, calm %.2f", nervous.comfortSeekingLocked(), calm.comfortSeekingLocked())
	}
	withWeather(calm, environment.WeatherClear, 0.9)
	if calm.comfortSeekingLocked() != 0 {
		t.Error("Fine weather doesn't drive pets to company")
	}
}

func TestBadWeatherKeepsPetsIn(t *testing.T) {
	pet := NewDigitalPet("Rex", "user123")
	pet.Species = SpeciesDog
	withWeather(pet, environment.WeatherStormy, 0.2)
	pet.Needs.SetNeedLevel(types.NeedExploration, 0.1)
	if r := pet.CurrentRequest(); r != nil && r.Action == "explore" {
		t.Error("A pet shouldn't ask to go exploring in a storm")
	}

	walk := func(weather environment.WeatherType, comfort float64) float64 {
		withWeather(pet, weather, comfort)
		return pet.innateReactionLocked(types.InteractionWalking)
	}
	if walk(environment.WeatherRainy, 0.2) >= walk(environment.WeatherClear, 0.9) {
		t.Error("A walk in the rain should be less enjoyable")
	}

	withWeather(pet, environment.WeatherClear, 0.9)
	fineInjury, _ := pet.explorationConditionsLocked()
	withWeather(pet, environment.WeatherStormy, 0.2)
	stormInjury, _ := pet.explorationConditionsLocked()
	if fineInjury != 0 || stormInjury <= fineInjury {
		t.Errorf("Storms should make trips riskier: %d vs %d", stormInjury, fineInjury)
	}
}

func TestPlacesOfferCompanyAndRisk(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Personality.Traits.Extraversion, pet.Personality.Traits.Agreeableness = 0.9, 0.9
	pet.ensureNeedsLocked()
	pet.Needs.SetNeedLevel(types.NeedSocial, 0.2)
	pet.ExploringTo = "park"
	if note := pet.visitPlaceLocked(); note != " Biscuit made some friends along the way." {
		t.Errorf("A sociable pet at the park: %q", note)
	}
	if pet.Needs.GetNeedLevel(types.NeedSocial) <= 0.2 {
		t.Error("The park's company should meet the social need")
	}

	// Risky places build up parasite exposure until the pet catches them
	pet.ExploringTo = "mountain"
	visits := 0
	for pet.Illness == nil && visits < 100 {
		pet.visitPlaceLocked()
		visits++
	}
	if pet.Illness == nil || pet.Illness.Kind != IllnessParasites || visits < 5 {
		t.Errorf("Parasites after %d mountain trips: %+v", visits, pet.Illness)
	}
}

func TestFoodAvailabilitySpeedsRegrowth(t *testing.T) {
	regrown := func(season environment.Season) float64 {
		pet := NewDigitalPet("Biscuit", "user123")
		pet.Environment.ForceSeason(season, 10)
		loc, _ := pet.placeLocked("park")
		loc.Deplete(1)
		pet.Environment.Update(24 * 5)
		return loc.ResourceLevel()
	}
	if summer, winter := regrown(environment.SeasonSummer), regrown(environment.SeasonWinter); summer <= winter {
		t.Errorf("Places should grow back faster in summer: %.2f vs %.2f", summer, winter)
	}
}
