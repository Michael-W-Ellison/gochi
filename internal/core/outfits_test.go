package core

import (
	"errors"
	"slices"
	"testing"
	"time"
)

// Outfits: every cosmetic, for both kinds of pet, can be bought on a day
// it is sold, worn in its slot, kept in the save and taken off, and a pet
// can wear one item in each of its six slots at once.

// onSaleDay sets the clock to a day the cosmetic is sold (in its season,
// or while its holiday's items are on sale), searching two years from the
// start of 2026
func onSaleDay(t *testing.T, c Cosmetic) {
	t.Helper()
	day := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 730; i++ {
		d := day.AddDate(0, 0, i)
		UseClock(func() time.Time { return d })
		if (c.Season == "" || RealSeason().String() == c.Season) && (c.Holiday == "" || HolidayOpen(c.Holiday)) {
			return
		}
	}
	t.Fatalf("%s is never on sale", c.ID)
}

func TestEveryCosmeticCanBeWorn(t *testing.T) {
	t.Cleanup(func() { UseClock(nil) })
	if len(cosmeticCatalog) < 50 {
		t.Fatalf("Only %d cosmetics: the holiday ones should be in the catalog too", len(cosmeticCatalog))
	}
	for _, species := range []Species{SpeciesCat, SpeciesDog} {
		for _, c := range cosmeticCatalog {
			onSaleDay(t, c)
			pet := wealthyPet(t, species)
			if err := pet.BuyCosmetic(c.ID); err != nil {
				t.Errorf("%s: buying %s: %v", species, c.ID, err)
				continue
			}
			if err := pet.Wear(c.Slot, c.ID); err != nil {
				t.Errorf("%s: wearing %s: %v", species, c.ID, err)
				continue
			}
			if worn := pet.Worn(); len(worn) != 1 || worn[c.Slot].ID != c.ID {
				t.Errorf("%s: %s should be worn in the %s slot: %v", species, c.ID, c.Slot, worn)
			}
			offered := false
			for _, o := range pet.CosmeticOffers() {
				if o.ID == c.ID {
					offered = o.Owned && o.Worn
				}
			}
			if !offered {
				t.Errorf("%s: the wardrobe should show %s owned and worn", species, c.ID)
			}
			if !ValidOutfit(pet.OutfitIDs()) {
				t.Errorf("%s: %s should make a valid outfit for playdates", species, c.ID)
			}
			for _, slot := range CosmeticSlots {
				if slot != c.Slot && pet.Wear(slot, c.ID) == nil {
					t.Errorf("%s: %s shouldn't be worn in the %s slot", species, c.ID, slot)
				}
			}

			data, err := pet.Save()
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := Load(data)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Worn()[c.Slot].ID != c.ID {
				t.Errorf("%s: %s should still be worn after loading", species, c.ID)
			}
			if err := loaded.Wear(c.Slot, ""); err != nil || len(loaded.Worn()) != 0 {
				t.Errorf("%s: taking off %s: %v %v", species, c.ID, err, loaded.Worn())
			}
			if !slices.Contains(loaded.Cosmetics, c.ID) {
				t.Errorf("%s: %s should be kept when taken off", species, c.ID)
			}
		}
	}
}

// One item in each slot, worn at once; another item in a slot replaces the
// first, which is kept
func TestAFullOutfit(t *testing.T) {
	full := map[string]string{SlotHead: "beret", SlotEyes: "sunglasses", SlotNeck: "bandana_red",
		SlotCollar: "collar_blue", SlotTag: "tag_paw", SlotBody: "raincoat"}
	if len(full) != len(CosmeticSlots) {
		t.Fatal("The outfit should fill every slot")
	}
	for _, species := range []Species{SpeciesCat, SpeciesDog} {
		pet := wealthyPet(t, species)
		for slot, id := range full {
			if err := pet.BuyCosmetic(id); err != nil {
				t.Fatal(err)
			}
			if err := pet.Wear(slot, id); err != nil {
				t.Fatal(err)
			}
		}
		if got := pet.OutfitIDs(); len(got) != 6 || !ValidOutfit(got) {
			t.Fatalf("%s: all six should be worn: %v", species, got)
		}
		for slot, id := range full {
			if pet.Worn()[slot].ID != id {
				t.Errorf("%s: %s should be worn in the %s slot", species, id, slot)
			}
		}

		if err := pet.BuyCosmetic("tiny_crown"); err != nil {
			t.Fatal(err)
		}
		if err := pet.Wear(SlotHead, "tiny_crown"); err != nil {
			t.Fatal(err)
		}
		if worn := pet.Worn(); worn[SlotHead].ID != "tiny_crown" || len(worn) != 6 {
			t.Errorf("%s: the crown should replace the beret, and the rest stay: %v", species, pet.OutfitIDs())
		}
		if !slices.Contains(pet.Cosmetics, "beret") {
			t.Errorf("%s: the beret should be kept", species)
		}
		if err := pet.Wear(SlotHead, "beret"); err != nil {
			t.Errorf("%s: wearing the beret again: %v", species, err)
		}
		if err := pet.BuyCosmetic("beret"); !errors.Is(err, ErrAlreadyOwned) {
			t.Errorf("%s: buying the beret twice: %v", species, err)
		}
	}
}

// What can't be done: wear what isn't owned, buy what's locked, out of
// season or unaffordable, or put something on an unknown slot
func TestOutfitRules(t *testing.T) {
	t.Cleanup(func() { UseClock(nil); SetHolidays(false, nil) })
	pet := wealthyPet(t, SpeciesCat)
	if err := pet.Wear(SlotHead, "beret"); !errors.Is(err, ErrNotOwned) {
		t.Errorf("Wearing what isn't owned: %v", err)
	}
	if err := pet.Wear("tail", ""); !errors.Is(err, ErrUnknownCosmetic) {
		t.Errorf("An unknown slot: %v", err)
	}
	poor := NewDigitalPet("Poor", "owner")
	if err := poor.BuyCosmetic("collar_red"); !errors.Is(err, ErrNotEnoughPoints) {
		t.Errorf("A collar with no points: %v", err)
	}

	// A seasonal item bought in its season stays wearable after it
	daisy, _ := FindCosmetic("daisy_crown")
	onSaleDay(t, daisy)
	if err := pet.BuyCosmetic("daisy_crown"); err != nil {
		t.Fatal(err)
	}
	later := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) // Autumn in the north
	UseClock(func() time.Time { return later })
	if err := pet.Wear(SlotHead, "daisy_crown"); err != nil {
		t.Errorf("A daisy crown out of season: %v", err)
	}
	other := wealthyPet(t, SpeciesCat)
	if err := other.BuyCosmetic("daisy_crown"); !errors.Is(err, ErrOutOfSeason) {
		t.Errorf("Buying a daisy crown out of season: %v", err)
	}

	// A holiday item is hidden, and can't be put on, while a parent has
	// turned its holiday off
	ears, _ := FindCosmetic("bunny_ears")
	onSaleDay(t, ears)
	if err := pet.BuyCosmetic("bunny_ears"); err != nil {
		t.Fatal(err)
	}
	if err := pet.Wear(SlotHead, "bunny_ears"); err != nil {
		t.Fatal(err)
	}
	SetHolidays(false, []string{ears.Holiday})
	if _, shown := pet.Worn()[SlotHead]; shown {
		t.Error("Bunny ears shouldn't be shown with Easter turned off")
	}
	if err := pet.Wear(SlotHead, "bunny_ears"); !errors.Is(err, ErrHolidayOff) {
		t.Errorf("Putting bunny ears on with Easter turned off: %v", err)
	}
	SetHolidays(false, nil)
	if pet.Worn()[SlotHead].ID != "bunny_ears" {
		t.Error("Bunny ears should be shown again with Easter back on")
	}
}
