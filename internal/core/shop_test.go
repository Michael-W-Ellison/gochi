package core

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// useClock replaces the reward clock for one test
func useClock(t *testing.T, start time.Time) *time.Time {
	t.Helper()
	current := start
	now = func() time.Time { return current }
	t.Cleanup(func() { now = time.Now })
	return &current
}

func TestDrinkRestoresHydration(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.Biology.Vitals.Hydration = 0.3

	r := pet.Interact(types.InteractionDrinking, 1.0)

	// A drink restores 0.4, scaled by how much the pet enjoys it
	want := 0.3 + 0.4*r.ReactionMultiplier
	if got := pet.Biology.Vitals.Hydration; math.Abs(got-want) > 1e-9 {
		t.Errorf("Hydration after drinking = %f, want %f", got, want)
	}
}

func TestCareEarnsPointsWithCooldown(t *testing.T) {
	clock := useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("TestPet", "user123")
	hungry := func() { pet.Biology.Vitals.Nutrition = 0.3 }

	hungry()
	if r := pet.Interact(types.InteractionFeeding, 1.0); r.PointsEarned != PointsPerCare || !r.Applied {
		t.Errorf("First feeding should earn %d points, got %+v", PointsPerCare, r)
	}
	hungry()
	if r := pet.Interact(types.InteractionFeeding, 1.0); r.PointsEarned != 0 || !r.Applied {
		t.Errorf("Feeding again within the cooldown should still apply but earn nothing, got %+v", r)
	}

	// A different kind of care has its own cooldown
	if r := pet.Interact(types.InteractionDrinking, 1.0); r.PointsEarned != PointsPerCare {
		t.Errorf("First drink should earn points, got %+v", r)
	}

	*clock = clock.Add(CareRewardCooldown)
	hungry()
	if r := pet.Interact(types.InteractionFeeding, 1.0); r.PointsEarned != PointsPerCare {
		t.Errorf("Feeding after the cooldown should earn points, got %+v", r)
	}

	if pet.FriendshipPoints != 3*PointsPerCare {
		t.Errorf("FriendshipPoints = %d, want %d", pet.FriendshipPoints, 3*PointsPerCare)
	}
}

func TestDisciplineEarnsNoPoints(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	if r := pet.Interact(types.InteractionDiscipline, 1.0); r.PointsEarned != 0 {
		t.Errorf("Discipline should not earn points, got %+v", r)
	}
}

func TestHibernatingPetEarnsNoPoints(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.Update(10000)

	if r := pet.Interact(types.InteractionFeeding, 1.0); r.Applied || r.PointsEarned != 0 {
		t.Errorf("Hibernating pet should not respond or earn points, got %+v", r)
	}
}

func TestBuyItem(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")

	if err := pet.BuyItem("no_such_item"); !errors.Is(err, ErrUnknownItem) {
		t.Errorf("Unknown item: got %v", err)
	}
	if err := pet.BuyItem("balloon"); !errors.Is(err, ErrNotEnoughPoints) {
		t.Errorf("Buying without points: got %v", err)
	}

	pet.FriendshipPoints = 100
	if err := pet.BuyItem("balloon"); err != nil {
		t.Fatalf("BuyItem failed: %v", err)
	}
	if pet.FriendshipPoints != 70 || !pet.Owns("balloon") {
		t.Errorf("After buying: %d points, owns=%v; want 70, true", pet.FriendshipPoints, pet.Owns("balloon"))
	}
	if err := pet.BuyItem("balloon"); !errors.Is(err, ErrAlreadyOwned) {
		t.Errorf("Buying twice: got %v", err)
	}
	if pet.FriendshipPoints != 70 {
		t.Error("A failed purchase should not spend points")
	}
}

// hydrationDrop returns how much hydration a pet loses over one update
func hydrationDrop(pet *DigitalPet) float64 {
	pet.Biology.Vitals.Hydration = 1.0
	pet.Update(1)
	return 1.0 - pet.Biology.Vitals.Hydration
}

func TestContainersSlowNeeds(t *testing.T) {
	plain := NewDigitalPet("Plain", "user123")
	equipped := NewDigitalPet("Equipped", "user123")
	equipped.FriendshipPoints = 1000
	if err := equipped.BuyItem("water_bowl"); err != nil {
		t.Fatal(err)
	}
	if err := equipped.BuyItem("water_fountain"); err != nil {
		t.Fatal(err)
	}

	// The fountain (40% slower) is the best owned water container
	ratio := hydrationDrop(equipped) / hydrationDrop(plain)
	if math.Abs(ratio-0.6) > 1e-9 {
		t.Errorf("Thirst with a fountain should build at 0.6x, got %fx", ratio)
	}
}

func TestBedSlowsTiredness(t *testing.T) {
	plain := NewDigitalPet("Plain", "user123")
	equipped := NewDigitalPet("Equipped", "user123")
	equipped.FriendshipPoints = 1000
	if err := equipped.BuyItem("cozy_bed"); err != nil {
		t.Fatal(err)
	}

	plain.Update(1)
	equipped.Update(1)

	ratio := equipped.Biology.Vitals.Fatigue / plain.Biology.Vitals.Fatigue
	if math.Abs(ratio-0.75) > 1e-9 {
		t.Errorf("Tiredness with a bed should build at 0.75x, got %fx", ratio)
	}
}

func TestToyMakesPlayMoreEffective(t *testing.T) {
	happinessGain := func(pet *DigitalPet) float64 {
		pet.Biology.Vitals.Happiness = 0.3
		pet.ProcessUserInteraction(types.InteractionPlaying, 1.0)
		return pet.Biology.Vitals.Happiness - 0.3
	}

	plain := NewDigitalPet("Plain", "user123")
	equipped := NewDigitalPet("Equipped", "user123")
	equipped.FriendshipPoints = 1000
	if err := equipped.BuyItem("puzzle_ball"); err != nil {
		t.Fatal(err)
	}

	ratio := happinessGain(equipped) / happinessGain(plain)
	if math.Abs(ratio-1.5) > 1e-9 {
		t.Errorf("Play with a puzzle ball should be 1.5x as effective, got %fx", ratio)
	}
}

func TestDecorationsAreCosmetic(t *testing.T) {
	plain := NewDigitalPet("Plain", "user123")
	decorated := NewDigitalPet("Decorated", "user123")
	decorated.FriendshipPoints = 1000
	if err := decorated.BuyItem("rainbow"); err != nil {
		t.Fatal(err)
	}

	if hydrationDrop(decorated) != hydrationDrop(plain) {
		t.Error("Decorations should not change needs")
	}
	if items := decorated.OwnedItems(); len(items) != 1 || items[0].Icon == "" {
		t.Errorf("OwnedItems should list the decoration with its icon, got %+v", items)
	}
}

func TestPointsAndItemsSurviveSaveLoad(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.FriendshipPoints = 200
	if err := pet.BuyItem("plant"); err != nil {
		t.Fatal(err)
	}
	pet.Interact(types.InteractionPetting, 1.0)

	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.FriendshipPoints != pet.FriendshipPoints || !loaded.Owns("plant") {
		t.Errorf("Loaded %d points, owns plant=%v; want %d, true", loaded.FriendshipPoints, loaded.Owns("plant"), pet.FriendshipPoints)
	}
	// The cooldown is kept, so reloading does not allow earning again early
	if r := loaded.Interact(types.InteractionPetting, 1.0); r.PointsEarned != 0 {
		t.Error("Reward cooldown should survive save and load")
	}
}

func TestCatalogItemsAreValid(t *testing.T) {
	seen := map[string]bool{}
	for _, item := range ShopCatalog() {
		if seen[item.ID] {
			t.Errorf("Duplicate item ID %q", item.ID)
		}
		seen[item.ID] = true
		if item.Name == "" || item.Icon == "" || item.Price <= 0 || item.Description == "" {
			t.Errorf("Incomplete item: %+v", item)
		}
		if item.Category != CategoryDecoration && item.Effect == 0 {
			t.Errorf("Item %q should have an effect", item.ID)
		}
	}
}

func changeOf(changes []StatChange, stat string) (float64, bool) {
	for _, c := range changes {
		if c.Stat == stat {
			return c.Change, true
		}
	}
	return 0, false
}

func TestInteractionReportsStatChanges(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	v := pet.Biology.Vitals
	v.Happiness, v.Energy, v.Stress = 0.3, 0.8, 0.5

	r := pet.Interact(types.InteractionPlaying, 1.0)

	// Benefits scale with the pet's reaction; the energy cost does not
	m := r.ReactionMultiplier
	want := map[string]float64{"Happiness": 0.2 * m, "Energy": -0.1, "Stress": -0.05 * m}
	for stat, delta := range want {
		got, ok := changeOf(r.Changes, stat)
		if !ok || math.Abs(got-delta) > 1e-9 {
			t.Errorf("%s change = %f (reported %v), want %f", stat, got, ok, delta)
		}
	}
	if _, ok := changeOf(r.Changes, "Water"); ok {
		t.Error("Unchanged stats should not be reported")
	}
	if r.BoostedBy != nil {
		t.Error("Play without a toy should not report a boost")
	}
}

func TestHungerChangeIsReportedAsShown(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.Biology.Vitals.Nutrition = 0.4

	r := pet.Interact(types.InteractionFeeding, 1.0)

	// Feeding raises nutrition by 0.3 (scaled by the reaction), so the
	// Hunger bar drops by the same amount
	want := -0.3 * r.ReactionMultiplier
	if got, _ := changeOf(r.Changes, "Hunger"); math.Abs(got-want) > 1e-9 {
		t.Errorf("Hunger change = %f, want %f", got, want)
	}
}

func TestInteractionReportsBoostingItem(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	pet.FriendshipPoints = 1000
	pet.BuyItem("squeaky_toy")
	pet.BuyItem("puzzle_ball")
	pet.Biology.Vitals.Happiness = 0.3

	r := pet.Interact(types.InteractionPlaying, 1.0)

	if r.BoostedBy == nil || r.BoostedBy.ID != "puzzle_ball" {
		t.Fatalf("Play should report the best toy as the boost, got %+v", r.BoostedBy)
	}
	want := 0.2 * 1.5 * r.ReactionMultiplier
	if got, _ := changeOf(r.Changes, "Happiness"); math.Abs(got-want) > 1e-9 {
		t.Errorf("Boosted happiness change = %f, want %f", got, want)
	}
	// A better toy makes play more fun, not more tiring
	if got, _ := changeOf(r.Changes, "Energy"); math.Abs(got+0.1) > 1e-9 {
		t.Errorf("Boosted energy change = %f, want -0.1", got)
	}

	if r := pet.Interact(types.InteractionFeeding, 1.0); r.BoostedBy != nil {
		t.Error("A toy should not boost feeding")
	}
}

func TestActiveEffects(t *testing.T) {
	pet := NewDigitalPet("TestPet", "user123")
	if effects := pet.ActiveEffects(); len(effects) != 0 {
		t.Errorf("A new pet should have no active effects, got %+v", effects)
	}

	pet.FriendshipPoints = 1000
	for _, id := range []string{"water_bowl", "water_fountain", "cozy_bed", "balloon"} {
		if err := pet.BuyItem(id); err != nil {
			t.Fatal(err)
		}
	}

	var ids []string
	for _, item := range pet.ActiveEffects() {
		ids = append(ids, item.ID)
	}
	// The fountain outclasses the bowl; decorations have no effect
	if len(ids) != 2 || ids[0] != "water_fountain" || ids[1] != "cozy_bed" {
		t.Errorf("Active effects = %v, want [water_fountain cozy_bed]", ids)
	}
}
