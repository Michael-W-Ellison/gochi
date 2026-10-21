package core

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/calendar"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// richPet returns a pet that can afford and has unlocked everything
func wealthyPet(t *testing.T, species Species) *DigitalPet {
	t.Helper()
	pet := NewDigitalPet("Mochi", "owner")
	pet.Species = species
	pet.FriendshipPoints, pet.PointsEarned = 100000, 100000
	return pet
}

func TestCatalogCounts(t *testing.T) {
	everyday := 0
	for _, c := range cosmeticCatalog {
		if c.Season == "" && c.Holiday == "" {
			everyday++
		}
	}
	if everyday != 46 {
		t.Errorf("Everyday cosmetics = %d, want 46", everyday)
	}
	perHoliday := map[string]int{}
	for _, g := range holidayGoods {
		perHoliday[g.holiday]++
		if _, ok := FindHoliday(g.holiday); !ok {
			t.Errorf("%s is for an unknown holiday %s", g.id, g.holiday)
		}
	}
	if len(holidays) != 20 || len(holidayGoods) != 60 {
		t.Errorf("%d holidays, %d holiday items; want 20 and 60", len(holidays), len(holidayGoods))
	}
	for _, h := range holidays {
		if perHoliday[h.ID] != 3 {
			t.Errorf("%s has %d items", h.ID, perHoliday[h.ID])
		}
	}
	total := 0
	for _, u := range HouseUpgrades() {
		total += u.Price
	}
	if total != 2400 {
		t.Errorf("House upgrades cost %d Home Stars, want 2,400", total)
	}
	ids := map[string]bool{}
	for _, id := range append(append(itemIDs(), cosmeticIDs()...), supplyIDs()...) {
		if ids[id] {
			t.Errorf("Duplicate ID %s", id)
		}
		ids[id] = true
	}
}

func itemIDs() (ids []string) {
	for _, i := range shopCatalog {
		ids = append(ids, i.ID)
	}
	return
}

func cosmeticIDs() (ids []string) {
	for _, c := range cosmeticCatalog {
		ids = append(ids, c.ID)
	}
	return
}

func supplyIDs() (ids []string) {
	for _, s := range supplyCatalog() {
		ids = append(ids, s.item.ID)
	}
	return
}

func TestEffectsAreCapped(t *testing.T) {
	pet := wealthyPet(t, SpeciesCat)
	for _, id := range []string{"litter_mat", "second_litter_box", "self_cleaning_box"} {
		if err := pet.BuyItem(id); err != nil {
			t.Fatal(err)
		}
	}
	pet.LastCleanedDay = today()
	// 0.85 × 0.75 × 0.6 is 0.38, but together they at most halve it
	if got := pet.effectLocked(effMess); got != maxReduction {
		t.Errorf("Mess = %v, want %v", got, maxReduction)
	}
	pet.LastCleanedDay = ""
	if got := pet.effectLocked(effMess); math.Abs(got-0.85*0.75) > 1e-9 {
		t.Errorf("Without a scoop today, the self-cleaning box doesn't help: %v", got)
	}

	// Decorations together add at most 6%
	pet.Personality.Traits.Playfulness, pet.Personality.Traits.Openness = 0.9, 0.9
	for _, id := range []string{"balloon", "painting", "rainbow"} {
		if err := pet.BuyItem(id); err != nil {
			t.Fatal(err)
		}
	}
	if got := pet.effectLocked(effHappinessDrop); math.Abs(got-0.94) > 1e-9 {
		t.Errorf("Three decorations: happiness drop %v, want 0.94", got)
	}
	// A decoration that doesn't suit the pet does nothing
	pet.Personality.Traits.Playfulness = 0.3
	if got := pet.effectLocked(effHappinessDrop); math.Abs(got-0.96) > 1e-9 {
		t.Errorf("Without a playful pet the balloon does nothing: %v", got)
	}
}

func TestUpkeep(t *testing.T) {
	pet := wealthyPet(t, SpeciesCat)
	if err := pet.BuyItem("water_fountain"); err != nil {
		t.Fatal(err)
	}
	if got := pet.effectLocked(effThirst); got != 0.6 {
		t.Errorf("A fresh fountain: %v", got)
	}
	pet.Biology.Processes.Age += fountainFilterDays + 1
	if got := pet.effectLocked(effThirst); got != 0.8 {
		t.Errorf("An old filter: %v", got)
	}
	if err := pet.BuySupply("fountain_filter"); err != nil {
		t.Fatal(err)
	}
	if _, err := pet.GiveCareSupply("fountain_filter"); err != nil {
		t.Fatal(err)
	}
	if got := pet.effectLocked(effThirst); got != 0.6 {
		t.Errorf("A new filter: %v", got)
	}

	// The first-aid kit treats injuries, three times
	if err := pet.BuyItem("first_aid_kit"); err != nil {
		t.Fatal(err)
	}
	if _, err := pet.UseFirstAid(); !errors.Is(err, ErrNotInjured) {
		t.Errorf("Not injured: %v", err)
	}
	for i := 0; i < 3; i++ {
		pet.Injured = true
		if _, err := pet.UseFirstAid(); err != nil {
			t.Fatal(err)
		}
	}
	pet.Injured = true
	if _, err := pet.UseFirstAid(); !errors.Is(err, ErrNoFirstAid) {
		t.Errorf("A used-up kit: %v", err)
	}

	// Once-a-day supplies
	if err := pet.BuySupply("probiotic"); err != nil {
		t.Fatal(err)
	}
	if _, err := pet.GiveCareSupply("probiotic"); err != nil {
		t.Fatal(err)
	}
	if _, err := pet.GiveCareSupply("probiotic"); !errors.Is(err, ErrNotNeeded) {
		t.Errorf("A second probiotic today: %v", err)
	}
	if got := pet.effectLocked(effFoodUpset); got != 0.5 {
		t.Errorf("Probiotic: food upset %v", got)
	}
	// A filter is only for a fountain
	dog := wealthyPet(t, SpeciesDog)
	if err := dog.BuySupply("fountain_filter"); err != nil {
		t.Fatal(err)
	}
	if _, err := dog.GiveCareSupply("fountain_filter"); !errors.Is(err, ErrNotNeeded) {
		t.Errorf("A filter without a fountain: %v", err)
	}
}

func TestSeasonalItemsFollowTheCalendar(t *testing.T) {
	clock := useClock(t, time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC))
	UseCalendar(environment.Northern)
	t.Cleanup(func() { environment.StopCalendar(); household.calendar, household.hemi = false, "" })

	pet := wealthyPet(t, SpeciesDog)
	if err := pet.BuyItem("knitted_sweater"); !errors.Is(err, ErrOutOfSeason) {
		t.Errorf("A sweater in a northern July: %v", err)
	}
	if err := pet.BuyItem("paddling_pool"); err != nil {
		t.Errorf("A pool in a northern July: %v", err)
	}
	UseCalendar(environment.Southern)
	if err := pet.BuyItem("knitted_sweater"); err != nil {
		t.Errorf("A sweater in a southern July: %v", err)
	}
	// Owned seasonal equipment works only in its season
	pet.ensureEnvironmentLocked()
	pet.Environment.Weather.Current.Temperature = 5
	if got := pet.effectLocked(effEnergy); got != 0.85 {
		t.Errorf("Sweater in the southern winter: %v", got)
	}
	*clock = time.Date(2026, 12, 10, 12, 0, 0, 0, time.UTC)
	if got := pet.effectLocked(effEnergy); got != 1 {
		t.Errorf("Sweater in the southern summer: %v", got)
	}
	// The weather's season follows the calendar too
	pet.Update(0.01)
	if pet.Environment.GetSeason() != environment.SeasonSummer {
		t.Errorf("December in the south is summer: %v", pet.Environment.GetSeason())
	}
}

func TestHolidayWindows(t *testing.T) {
	clock := useClock(t, time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	t.Cleanup(func() { SetHolidays(false, nil) })
	open := map[string]bool{}
	for _, h := range OpenHolidays() {
		open[h.ID] = true
	}
	if !open["halloween"] || !open["day_of_the_dead"] || open["christmas"] {
		t.Errorf("Open on October 25: %v", open)
	}

	pet := wealthyPet(t, SpeciesCat)
	if err := pet.BuyCosmetic("pumpkin_costume"); err != nil {
		t.Fatal(err)
	}
	if err := pet.Wear(SlotBody, "pumpkin_costume"); err != nil {
		t.Fatal(err)
	}
	if err := pet.BuyCosmetic("santa_hat"); !errors.Is(err, ErrNotHolidayTime) {
		t.Errorf("A Santa hat in October: %v", err)
	}

	// Turned off by a parent: no shop, and its cosmetics aren't shown
	SetHolidays(false, []string{"halloween"})
	if HolidayOpen("halloween") || pet.Worn()[SlotBody].ID != "" {
		t.Error("A holiday that is off has no shop and its cosmetics aren't shown")
	}
	if err := pet.Wear(SlotBody, "pumpkin_costume"); !errors.Is(err, ErrHolidayOff) {
		t.Errorf("Wearing a turned-off holiday's cosmetic: %v", err)
	}
	SetHolidays(true, nil)
	if len(OpenHolidays()) != 0 {
		t.Error("All holidays off")
	}
	SetHolidays(false, nil)
	if pet.Worn()[SlotBody].ID != "pumpkin_costume" {
		t.Error("Turning it back on restores it")
	}

	// Owned holiday items stay after the window
	*clock = time.Date(2027, 1, 20, 12, 0, 0, 0, time.UTC)
	if HolidayOpen("halloween") || pet.Worn()[SlotBody].ID != "pumpkin_costume" {
		t.Error("After the window the shop closes but the costume is kept")
	}
	// Day of the Dead lasts two days, Kwanzaa seven
	dotd, _ := FindHoliday("day_of_the_dead")
	if !dotd.OpenOn(calendar.Date{Year: 2026, Month: 11, Day: 9}) || dotd.OpenOn(calendar.Date{Year: 2026, Month: 11, Day: 10}) {
		t.Error("The Day of the Dead window runs to a week after November 2")
	}
	kwanzaa, _ := FindHoliday("kwanzaa")
	if !kwanzaa.OpenOn(calendar.Date{Year: 2027, Month: 1, Day: 8}) {
		t.Error("The Kwanzaa window runs to a week after January 1")
	}
}

func TestCosmetics(t *testing.T) {
	pet := wealthyPet(t, SpeciesDog)
	if err := pet.Wear(SlotHead, "beret"); !errors.Is(err, ErrNotOwned) {
		t.Errorf("Wearing what isn't owned: %v", err)
	}
	if err := pet.BuyCosmetic("beret"); err != nil {
		t.Fatal(err)
	}
	if err := pet.Wear(SlotNeck, "beret"); !errors.Is(err, ErrUnknownCosmetic) {
		t.Errorf("A beret round the neck: %v", err)
	}
	if err := pet.Wear(SlotHead, "beret"); err != nil {
		t.Fatal(err)
	}
	poor := NewDigitalPet("Poor", "owner")
	poor.FriendshipPoints = 1000
	if err := poor.BuyCosmetic("tiny_crown"); !errors.Is(err, ErrItemLocked) {
		t.Errorf("The tiny crown needs 600 lifetime points: %v", err)
	}
	if !ValidOutfit(pet.OutfitIDs()) || ValidOutfit(map[string]string{SlotHead: "collar_red"}) ||
		ValidOutfit(map[string]string{SlotHead: "anything"}) {
		t.Error("ValidOutfit")
	}
	if err := pet.Wear(SlotHead, ""); err != nil || len(pet.Worn()) != 0 {
		t.Errorf("Taking it off: %v %v", err, pet.Worn())
	}
	// Kept in the save
	if err := pet.Wear(SlotHead, "beret"); err != nil {
		t.Fatal(err)
	}
	data, _ := pet.Save()
	loaded, _ := Load(data)
	if loaded.Worn()[SlotHead].ID != "beret" {
		t.Error("The outfit should be saved")
	}
}

func TestHouseUpgradesApplyToEveryPet(t *testing.T) {
	t.Cleanup(func() { SetHouse(nil, false) })
	cat, dog := wealthyPet(t, SpeciesCat), wealthyPet(t, SpeciesDog)
	SetHouse([]string{"climate_control", "pet_door", "air_filtration"}, false)
	for _, pet := range []*DigitalPet{cat, dog} {
		if got := pet.effectLocked(effWeatherStress); got != 0.7 {
			t.Errorf("Climate control: %v", got)
		}
		if got := pet.effectLocked(effAirDamage); got != 1 {
			t.Errorf("Air filtration with an old filter: %v", got)
		}
	}
	if dog.effectLocked(effAccidents) != 0.7 || cat.effectLocked(effAccidents) != 1 {
		t.Error("The pet door helps dogs hold on")
	}
	SetHouse([]string{"air_filtration"}, true)
	if got := cat.effectLocked(effAirDamage); got != 0.5 {
		t.Errorf("Air filtration with a fresh filter: %v", got)
	}
}

func TestShopEffectsChangeTheSimulation(t *testing.T) {
	// A flea collar halves parasite exposure from trips
	pet := wealthyPet(t, SpeciesDog)
	if err := pet.BuyItem("flea_collar"); err != nil {
		t.Fatal(err)
	}
	if got := pet.effectLocked(effParasites); got != 0.5 {
		t.Errorf("Flea collar: %v", got)
	}
	// Freeze-dried treats make the next training count more, once
	if err := pet.BuySupply("freeze_dried_treats"); err != nil {
		t.Fatal(err)
	}
	if _, err := pet.GiveCareSupply("freeze_dried_treats"); err != nil || !pet.TrainingBoost {
		t.Fatalf("Freeze-dried treats: %v", err)
	}
	pet.Biology.Vitals.Energy, pet.Biology.Vitals.Fatigue = 1, 0
	if _, err := pet.TrainTrick("sit"); err != nil {
		t.Fatal(err)
	}
	if pet.TrainingBoost {
		t.Error("The treat is used by one session")
	}
	// Cleaning marks the day, for the self-cleaning box
	cat := wealthyPet(t, SpeciesCat)
	cat.LivingAreaWaste = 0.5
	cat.Interact(types.InteractionCleaning, 1)
	if cat.LastCleanedDay != today() {
		t.Error("Cleaning should be recorded")
	}
}
