package core

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// richPet returns a pet with points to spend and every supply unlocked
func richPet(species Species) *DigitalPet {
	pet := NewDigitalPet("Biscuit", "u")
	pet.Species = species
	pet.FriendshipPoints, pet.PointsEarned = 500, 500
	return pet
}

func offerOf(pet *DigitalPet, id string) (SupplyOffer, bool) {
	for _, o := range pet.SupplyOffers() {
		if o.ID == id {
			return o, true
		}
	}
	return SupplyOffer{}, false
}

func TestSuppliesUnlockByRarity(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "u")
	if o, ok := offerOf(pet, "cat_wet"); !ok || o.Locked {
		t.Errorf("Common supplies should be unlocked from the start: %+v", o)
	}
	if o, _ := offerOf(pet, "cat_premium"); !o.Locked || o.UnlockAt != 150 {
		t.Errorf("Rare supplies should unlock at 150 lifetime points: %+v", o)
	}
	if _, ok := offerOf(pet, "dog_kibble"); ok {
		t.Error("A cat's shop shouldn't sell dog food")
	}
	if _, ok := offerOf(pet, "chocolate"); ok {
		t.Error("Chocolate is offered to dog owners only")
	}
	pet.FriendshipPoints = 100
	if err := pet.BuySupply("cat_premium"); !errors.Is(err, ErrSupplyLocked) {
		t.Errorf("Buying a locked supply: %v", err)
	}
	pet.PointsEarned = 150
	if err := pet.BuySupply("cat_premium"); err != nil {
		t.Fatalf("Buying an unlocked supply: %v", err)
	}
	if pet.Supplies.GetItemQuantity("cat_premium") != 3 || pet.FriendshipPoints >= 100 {
		t.Errorf("A pack should hold 3 meals and cost points: %d left, %d points", pet.Supplies.GetItemQuantity("cat_premium"), pet.FriendshipPoints)
	}
}

func TestDailyDiscounts(t *testing.T) {
	clock := useClock(t, time.Date(2026, 10, 19, 9, 0, 0, 0, time.Local))
	pet := richPet(SpeciesCat)
	count := func() (n int) {
		for _, o := range pet.SupplyOffers() {
			if o.Discounted {
				n++
				if o.Price != int(float64(o.FullPrice)*(1-discountRate)) {
					t.Errorf("Discounted price wrong: %+v", o)
				}
			}
		}
		return n
	}
	if n := count(); n != dailyDiscounts {
		t.Errorf("%d supplies should be discounted each day, got %d", dailyDiscounts, n)
	}
	before := discountedToday([]string{"a", "b", "c", "d", "e", "f", "g"})
	*clock = clock.Add(24 * time.Hour)
	if after := discountedToday([]string{"a", "b", "c", "d", "e", "f", "g"}); slices.Equal(before, after) && before[0] == after[0] {
		t.Logf("Same discounts two days running: %v (allowed, but unlikely)", after)
	}
}

func TestFoodTypes(t *testing.T) {
	pet := richPet(SpeciesCat)
	if _, err := pet.Feed("cat_wet"); !errors.Is(err, ErrNoSupply) {
		t.Errorf("Feeding food not in the pantry: %v", err)
	}
	pet.BuySupply("cat_wet")

	setCalmHealthyState(pet)
	pet.Biology.Vitals.Nutrition, pet.Biology.Vitals.Hydration = 0.3, 0.5
	r, err := pet.Feed("cat_wet")
	if err != nil || !r.Applied || pet.Biology.Vitals.Hydration <= 0.5 {
		t.Fatalf("Wet food should feed and add water: %+v %v", r, err)
	}
	if pet.Supplies.GetItemQuantity("cat_wet") != 2 || !strings.Contains(r.Lesson, "Wet food") {
		t.Errorf("A meal should be used and the lesson shown: %d left, %q", pet.Supplies.GetItemQuantity("cat_wet"), r.Lesson)
	}

	// Basic food never runs out
	for i := 0; i < 3; i++ {
		pet.Biology.Vitals.Nutrition = 0.3
		if r, err := pet.Feed("cat_dry"); err != nil || !r.Applied {
			t.Fatalf("Basic food should always be available: %+v %v", r, err)
		}
	}
	if _, err := pet.Feed("dog_kibble"); !errors.Is(err, ErrWrongSpecies) {
		t.Errorf("Feeding dog food to a cat: %v", err)
	}
}

func TestBadFoods(t *testing.T) {
	useRandom(t, 999_999)
	cat := richPet(SpeciesCat)
	cat.BuySupply("milk")
	digestion := cat.Biology.Physiology.DigestiveHealth
	cat.Biology.Vitals.Nutrition = 0.3
	r, _ := cat.Feed("milk")
	if cat.Biology.Physiology.DigestiveHealth >= digestion || !strings.Contains(r.Lesson, "lactose") {
		t.Errorf("Milk should upset a cat's stomach and explain why: %+v", r)
	}
	if !hasSymptom(cat, "upset stomach") {
		t.Errorf("An upset stomach should be a warning sign: %+v", cat.Symptoms())
	}

	dog := richPet(SpeciesDog)
	if o, _ := offerOf(dog, "chocolate"); o.Warning == "" {
		t.Error("Chocolate should carry a warning")
	}
	dog.BuySupply("chocolate")
	dog.Biology.Vitals.Nutrition = 0.3
	r, _ = dog.Feed("chocolate")
	if dog.IllnessName() != "Chocolate poisoning" || !strings.Contains(r.Lesson, "theobromine") {
		t.Fatalf("Chocolate should poison a dog: %q, %q", dog.IllnessName(), r.Lesson)
	}
	r = dog.Interact(types.InteractionMedicalCare, 1.0)
	if dog.Illness != nil || !strings.Contains(r.Lesson, "chocolate") {
		t.Errorf("A vet should treat the poisoning and teach prevention: %+v", r)
	}
}

func TestToys(t *testing.T) {
	pet := richPet(SpeciesCat)
	if _, err := pet.PlayWith("wand_toy"); !errors.Is(err, ErrNoSupply) {
		t.Errorf("Playing with a toy not owned: %v", err)
	}
	pet.BuySupply("wand_toy")

	setCalmHealthyState(pet)
	plain := pet.reactionScoreFor(types.InteractionPlaying, careChoice{})
	wand, _ := findSupply("wand_toy")
	if pet.reactionScoreFor(types.InteractionPlaying, careChoice{toy: &wand}) <= plain {
		t.Error("A cat should enjoy a wand toy more than plain play")
	}

	r, err := pet.PlayWith("wand_toy")
	if err != nil || !r.Applied || !strings.Contains(r.Lesson, "stalk") {
		t.Fatalf("Playing with the wand: %+v %v", r, err)
	}
	if uses := pet.Supplies.Items["wand_toy"].Item.Uses; uses != 29 {
		t.Errorf("The wand should wear by one use, got %d left", uses)
	}

	// Toys wear out
	pet.Supplies.Items["wand_toy"].Item.Uses = 1
	setCalmHealthyState(pet)
	pet.RecentCare = nil
	r, _ = pet.PlayWith("wand_toy")
	if pet.Supplies.HasItem("wand_toy") || !strings.Contains(r.Note, "worn out") {
		t.Errorf("A worn-out toy should be thrown away: %+v", r)
	}

	// Catnip depends on the cat's genes
	pet.BuySupply("catnip_mouse")
	setCalmHealthyState(pet)
	r, _ = pet.PlayWith("catnip_mouse")
	if !strings.Contains(strings.ToLower(r.Lesson), "catnip") {
		t.Errorf("Catnip should explain who reacts to it: %q", r.Lesson)
	}
}

func TestPettingRegions(t *testing.T) {
	cat := NewDigitalPet("Biscuit", "u")
	setCalmHealthyState(cat)
	cheeks, _ := findRegion(SpeciesCat, "cheeks")
	belly, _ := findRegion(SpeciesCat, "belly")
	if cat.reactionScoreFor(types.InteractionPetting, careChoice{region: &cheeks}) <=
		cat.reactionScoreFor(types.InteractionPetting, careChoice{region: &belly}) {
		t.Error("Most cats prefer their cheeks stroked to their belly")
	}
	r, err := cat.PetOn("belly")
	if err != nil || r.Reaction != ReactionReluctant {
		t.Errorf("A typical cat shouldn't like a belly rub: %+v %v", r, err)
	}

	// A cuddly, calm dog loves belly rubs; a nervous one doesn't
	cuddly := withAttentionStyle(NewDigitalPet("Rex", "u"), 0.9, 0.2, 0.7, 0.1)
	cuddly.Species = SpeciesDog
	nervous := withAttentionStyle(NewDigitalPet("Shy", "u"), 0.4, 0.5, 0.4, 0.9)
	nervous.Species = SpeciesDog
	dogBelly, _ := findRegion(SpeciesDog, "belly")
	if cuddly.regionLikingLocked(dogBelly) <= 0 || nervous.regionLikingLocked(dogBelly) >= 0 {
		t.Errorf("Belly rubs: cuddly dog %f, nervous dog %f", cuddly.regionLikingLocked(dogBelly), nervous.regionLikingLocked(dogBelly))
	}
	if _, err := cat.PetOn("paws"); !errors.Is(err, ErrUnknownChoice) {
		t.Errorf("Cats have no 'paws' region in the list: %v", err)
	}
}

func TestTrainingMethods(t *testing.T) {
	pet := richPet(SpeciesDog)
	if _, err := pet.TrainTrickWith("sit", "treats"); !errors.Is(err, ErrNoTreats) {
		t.Errorf("Treat training without treats: %v", err)
	}
	pet.BuySupply("dog_treats")
	if _, err := pet.TrainTrickWith("sit", "clicker"); !errors.Is(err, ErrNoClicker) {
		t.Errorf("Clicker training without a clicker: %v", err)
	}

	gain := func(method string) float64 {
		p := richPet(SpeciesDog)
		p.BuySupply("dog_treats")
		p.BuyItem("clicker")
		setCalmHealthyState(p)
		r, err := p.TrainTrickWith("sit", method)
		if err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		return r.Skill
	}
	clicker, treats, praise, scolding := gain("clicker"), gain("treats"), gain("praise"), gain("punishment")
	if !(clicker > treats && treats > praise && praise > scolding) {
		t.Errorf("Learning speed should be clicker > treats > praise > scolding: %f %f %f %f", clicker, treats, praise, scolding)
	}

	setCalmHealthyState(pet)
	r, _ := pet.TrainTrickWith("sit", "treats")
	if pet.Supplies.GetItemQuantity("dog_treats") != 9 {
		t.Errorf("Treat training should use a treat, %d left", pet.Supplies.GetItemQuantity("dog_treats"))
	}
	if !strings.Contains(r.Lesson, "treat") {
		t.Errorf("The method's lesson should be shown: %q", r.Lesson)
	}

	scolded := richPet(SpeciesDog)
	setCalmHealthyState(scolded)
	neuroticism := scolded.Personality.Traits.Neuroticism
	r, _ = scolded.TrainTrickWith("sit", "punishment")
	if scolded.Personality.Traits.Neuroticism <= neuroticism || !strings.Contains(r.Lesson, "fearful") {
		t.Errorf("Scolding should make the pet more anxious and teach why it doesn't work: %+v", r)
	}
}

func TestEnvironmentChanges(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "u")
	setCalmHealthyState(pet)
	pet.Needs.Needs[types.NeedMentalStimulation].CurrentLevel = 0.3
	r, err := pet.ChangeEnvironment("perch")
	if err != nil || !r.Applied || !strings.Contains(r.Lesson, "window perch") {
		t.Fatalf("A window perch: %+v %v", r, err)
	}
	if needOf(pet, types.NeedMentalStimulation) <= 0.3 {
		t.Error("A window perch should stimulate the cat's mind")
	}

	// The same change again soon feels less new
	pet.RecentCare = nil
	r, _ = pet.ChangeEnvironment("perch")
	if !strings.Contains(r.Note, "feel new again") {
		t.Errorf("A repeated change should lose its novelty: %+v", r)
	}

	if _, err := pet.ChangeEnvironment("sniff_game"); !errors.Is(err, ErrUnknownChoice) {
		t.Errorf("A dog's game isn't offered for a cat: %v", err)
	}

	// Nervous pets are calmed by a hiding place
	nervous := withAttentionStyle(NewDigitalPet("Shy", "u"), 0.5, 0.5, 0.3, 0.9)
	setCalmHealthyState(nervous)
	nervous.Biology.Vitals.Stress = 0.6
	nervous.ChangeEnvironment("box")
	if nervous.Biology.Vitals.Stress >= 0.6 {
		t.Error("A box should calm a nervous cat")
	}
}

func TestCareOptions(t *testing.T) {
	pet := richPet(SpeciesDog)
	pet.BuySupply("tennis_ball")
	options := pet.GetCareOptions()
	if len(options.Foods) == 0 || options.Foods[0].ID != "dog_kibble" || options.Foods[0].Quantity != -1 {
		t.Errorf("Basic food should come first and never run out: %+v", options.Foods)
	}
	if len(options.Toys) != 1 || options.Toys[0].UsesLeft != 40 {
		t.Errorf("Owned toys should be listed with their uses: %+v", options.Toys)
	}
	if len(options.Regions) == 0 || options.Regions[0].ID != "chest" {
		t.Errorf("A dog's petting spots should be listed: %+v", options.Regions)
	}
	if options.Methods[0].Unavailable == "" {
		t.Errorf("Clicker training should be unavailable without a clicker: %+v", options.Methods[0])
	}
}
