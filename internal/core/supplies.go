package core

import (
	"errors"
	"hash/fnv"
	"slices"

	"github.com/Michael-W-Ellison/gochi/internal/interaction"
)

// Supplies are consumable items bought with Friendship Points and kept in
// the pet's pantry: food, treats and toys. They are built on the
// interaction package's items, inventory, catalog and shop. Rarer supplies
// unlock as the pet earns more points over its life, and a couple of them
// are discounted each day. Equipment and decorations (shop.go) are
// separate: they are bought once and kept.

// Kinds of supply
const (
	SupplyFood  = "food"
	SupplyTreat = "treat"
	SupplyToy   = "toy"
)

// supply describes one kind of supply
type supply struct {
	item    interaction.Item // Template for the catalog and pantry
	icon    string
	kind    string
	species []Species // Who it is for; nil for both
	pack    int       // Units bought together for the price
	food    *foodProfile
	toy     *toyProfile
}

// foodProfile is how a food feeds the pet and what it teaches. Amounts
// are relative to a normal meal of the species' basic food.
type foodProfile struct {
	nutrition float64 // Share of a normal meal's nutrition
	hydration float64 // Water it adds
	taste     float64 // How much more the pet enjoys it (reaction)
	digestion float64 // Effect on digestive health: positive helps, negative upsets
	toxic     bool    // Poisonous to the species: makes the pet ill
	warning   string  // Shown before feeding it
	lesson    string  // Shown after feeding it
}

// toyProfile is how a toy plays and what it teaches
type toyProfile struct {
	fit         float64 // How much the species enjoys it (reaction)
	exercise    float64 // Exercise per play session, relative to playing together
	stimulation float64 // Mental stimulation per session
	calming     float64 // Stress it relieves
	catnip      bool    // Only some cats respond
	lesson      string
}

// Basic foods are free and always in the pantry
var basicFoods = map[Species]supply{
	SpeciesCat: {item: interaction.Item{ID: "cat_dry", Name: "Dry cat food", Type: interaction.ItemTypeFood,
		Description: "Complete dry food for cats. Always in the pantry."}, icon: "🥣", kind: SupplyFood,
		species: []Species{SpeciesCat},
		food:    &foodProfile{nutrition: 1, lesson: "Complete cat food gives a cat everything it needs; measure each meal."}},
	SpeciesDog: {item: interaction.Item{ID: "dog_kibble", Name: "Dry dog food", Type: interaction.ItemTypeFood,
		Description: "Complete dry food for dogs. Always in the pantry."}, icon: "🥣", kind: SupplyFood,
		species: []Species{SpeciesDog},
		food:    &foodProfile{nutrition: 1, lesson: "Complete dog food gives a dog everything it needs; measure each meal."}},
}

// supplyCatalog lists every supply in shop order
var supplyCatalog = []supply{
	// Cat food
	{item: interaction.Item{ID: "cat_wet", Name: "Wet cat food", Price: 8, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeFood, Description: "Complete food with plenty of water in it. 3 meals."},
		icon: "🥫", kind: SupplyFood, species: []Species{SpeciesCat}, pack: 3,
		food: &foodProfile{nutrition: 1, hydration: 0.15, taste: 0.1,
			lesson: "Wet food adds water to a cat's diet, which helps cats that don't drink much."}},
	{item: interaction.Item{ID: "cat_fish", Name: "Plain cooked fish", Price: 12, Rarity: interaction.RarityUncommon,
		Type: interaction.ItemTypeFood, Description: "A tasty topper, not a complete meal. 2 servings."},
		icon: "🐟", kind: SupplyFood, species: []Species{SpeciesCat}, pack: 2,
		food: &foodProfile{nutrition: 0.6, taste: 0.25,
			lesson: "Plain cooked fish is a treat, not a diet: on its own it lacks nutrients cats need. Never feed raw fish regularly."}},
	{item: interaction.Item{ID: "cat_premium", Name: "Premium cat food", Price: 25, Rarity: interaction.RarityRare,
		Type: interaction.ItemTypeFood, Description: "High-quality complete food that's easy to digest. 3 meals."},
		icon: "🍽️", kind: SupplyFood, species: []Species{SpeciesCat}, pack: 3,
		food: &foodProfile{nutrition: 1.1, taste: 0.15, digestion: 0.05,
			lesson: "A good complete diet keeps digestion healthy; change foods gradually over a week to avoid upsets."}},
	// Dog food
	{item: interaction.Item{ID: "dog_wet", Name: "Wet dog food", Price: 8, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeFood, Description: "Complete food with extra moisture. 3 meals."},
		icon: "🥫", kind: SupplyFood, species: []Species{SpeciesDog}, pack: 3,
		food: &foodProfile{nutrition: 1, hydration: 0.1, taste: 0.1,
			lesson: "Wet food is tasty and adds water; many owners mix it with dry food."}},
	{item: interaction.Item{ID: "dog_chicken", Name: "Plain cooked chicken", Price: 12, Rarity: interaction.RarityUncommon,
		Type: interaction.ItemTypeFood, Description: "Boneless and unseasoned; a tasty topper. 2 servings."},
		icon: "🍗", kind: SupplyFood, species: []Species{SpeciesDog}, pack: 2,
		food: &foodProfile{nutrition: 0.6, taste: 0.25,
			lesson: "Plain, boneless cooked chicken is a safe topper; cooked bones splinter and seasoning can be toxic."}},
	{item: interaction.Item{ID: "dog_premium", Name: "Premium dog food", Price: 25, Rarity: interaction.RarityRare,
		Type: interaction.ItemTypeFood, Description: "High-quality complete food that's easy to digest. 3 meals."},
		icon: "🍽️", kind: SupplyFood, species: []Species{SpeciesDog}, pack: 3,
		food: &foodProfile{nutrition: 1.1, taste: 0.15, digestion: 0.05,
			lesson: "A good complete diet keeps digestion healthy; change foods gradually over a week to avoid upsets."}},
	// Foods to avoid, to learn what not to feed
	{item: interaction.Item{ID: "milk", Name: "Cow's milk", Price: 3, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeFood, Description: "Not recommended. 1 bowl."},
		icon: "🥛", kind: SupplyFood, pack: 1,
		food: &foodProfile{nutrition: 0.3, taste: 0.2, digestion: -0.3,
			warning: "Most adult cats and many dogs can't digest the sugar in milk (lactose). Feed it anyway?",
			lesson:  "Milk upset the stomach: most adult cats and many dogs are lactose intolerant. Water is the drink they need."}},
	{item: interaction.Item{ID: "table_scraps", Name: "Table scraps", Price: 3, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeFood, Description: "Leftovers from dinner. Not recommended. 1 plate."},
		icon: "🍖", kind: SupplyFood, pack: 1,
		food: &foodProfile{nutrition: 0.5, taste: 0.3, digestion: -0.35,
			warning: "Fatty, seasoned leftovers can upset the stomach, and onions and garlic are toxic to pets. Feed them anyway?",
			lesson:  "Rich, fatty leftovers upset the stomach and can cause pancreatitis; onions and garlic are toxic to cats and dogs."}},
	{item: interaction.Item{ID: "chocolate", Name: "Chocolate", Price: 3, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeFood, Description: "Toxic to dogs. 1 piece."},
		icon: "🍫", kind: SupplyFood, species: []Species{SpeciesDog}, pack: 1,
		food: &foodProfile{nutrition: 0.2, taste: 0.3, toxic: true,
			warning: "Chocolate is poisonous to dogs. Feed it anyway?",
			lesson:  "Chocolate contains theobromine, which poisons dogs; dark chocolate is the most dangerous. If a dog eats chocolate, call a vet."}},
	// Treats
	{item: interaction.Item{ID: "cat_treats", Name: "Cat treats", Price: 10, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeFood, Description: "Small rewards for training and good behavior. Bag of 10."},
		icon: "🍪", kind: SupplyTreat, species: []Species{SpeciesCat}, pack: 10},
	{item: interaction.Item{ID: "dog_treats", Name: "Dog treats", Price: 10, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeFood, Description: "Small rewards for training and good behavior. Bag of 10."},
		icon: "🦴", kind: SupplyTreat, species: []Species{SpeciesDog}, pack: 10},
	// Cat toys
	{item: interaction.Item{ID: "wand_toy", Name: "Feather wand", Price: 15, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeToy, Uses: 30, MaxUses: 30, Description: "A feather on a string to stalk and pounce on. Wears out."},
		icon: "🪶", kind: SupplyToy, species: []Species{SpeciesCat}, pack: 1,
		toy: &toyProfile{fit: 0.25, exercise: 1.3, stimulation: 0.6,
			lesson: "A wand toy lets a cat stalk, chase and pounce, the way it would hunt; let it catch the feather at the end."}},
	{item: interaction.Item{ID: "catnip_mouse", Name: "Catnip mouse", Price: 10, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeToy, Uses: 20, MaxUses: 20, Description: "A toy mouse filled with catnip. Wears out."},
		icon: "🐭", kind: SupplyToy, species: []Species{SpeciesCat}, pack: 1,
		toy: &toyProfile{fit: 0.1, exercise: 0.8, stimulation: 0.4, catnip: true}},
	{item: interaction.Item{ID: "laser_pointer", Name: "Laser pointer", Price: 20, Rarity: interaction.RarityUncommon,
		Type: interaction.ItemTypeToy, Description: "A red dot to chase."},
		icon: "🔴", kind: SupplyToy, species: []Species{SpeciesCat}, pack: 1,
		toy: &toyProfile{fit: 0.15, exercise: 1.4, stimulation: 0.3,
			lesson: "Chasing a dot that can never be caught can frustrate a cat; finish with a toy it can catch. Never shine it in eyes."}},
	// Dog toys
	{item: interaction.Item{ID: "tennis_ball", Name: "Tennis ball", Price: 10, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeToy, Uses: 40, MaxUses: 40, Description: "For fetch. Wears out."},
		icon: "🎾", kind: SupplyToy, species: []Species{SpeciesDog}, pack: 1,
		toy: &toyProfile{fit: 0.25, exercise: 1.4, stimulation: 0.3,
			lesson: "Fetch is great exercise; use a ball too big to swallow, and stop before your dog is exhausted."}},
	{item: interaction.Item{ID: "rope_toy", Name: "Rope toy", Price: 15, Rarity: interaction.RarityCommon,
		Type: interaction.ItemTypeToy, Uses: 30, MaxUses: 30, Description: "For tug of war. Wears out."},
		icon: "🪢", kind: SupplyToy, species: []Species{SpeciesDog}, pack: 1,
		toy: &toyProfile{fit: 0.2, exercise: 1.1, stimulation: 0.4,
			lesson: "Tug is good fun with rules: your dog lets go on cue, and the game stops if teeth touch skin."}},
	{item: interaction.Item{ID: "chew_toy", Name: "Chew toy", Price: 20, Rarity: interaction.RarityUncommon,
		Type: interaction.ItemTypeToy, Uses: 50, MaxUses: 50, Description: "Something safe to chew. Wears out."},
		icon: "🦷", kind: SupplyToy, species: []Species{SpeciesDog}, pack: 1,
		toy: &toyProfile{fit: 0.1, exercise: 0.2, stimulation: 0.3, calming: 0.1,
			lesson: "Chewing calms dogs and keeps them from chewing furniture; replace a chew toy before it breaks into pieces."}},
	// Toys for both
	{item: interaction.Item{ID: "puzzle_feeder", Name: "Puzzle feeder", Price: 35, Rarity: interaction.RarityRare,
		Type: interaction.ItemTypeToy, Description: "A toy that releases food as it's worked out."},
		icon: "🧩", kind: SupplyToy, pack: 1,
		toy: &toyProfile{fit: 0.1, exercise: 0.3, stimulation: 1.2, calming: 0.05,
			lesson: "Puzzle feeders make a pet work for its food, which keeps its mind busy and slows down fast eaters."}},
}

// Points a pet must have earned over its life to unlock each rarity
var rarityUnlocks = map[interaction.ItemRarity]int{
	interaction.RarityCommon:    0,
	interaction.RarityUncommon:  50,
	interaction.RarityRare:      150,
	interaction.RarityEpic:      300,
	interaction.RarityLegendary: 600,
}

// Daily discounts
const (
	dailyDiscounts = 2    // Supplies discounted each day
	discountRate   = 0.25 // Discount on them
)

// Supply errors
var (
	ErrUnknownSupply = errors.New("unknown supply")
	ErrSupplyLocked  = errors.New("supply not unlocked yet")
	ErrPantryFull    = errors.New("pantry is full")
	ErrNoSupply      = errors.New("none left in the pantry")
)

// pantryCapacity is how many kinds of supply the pantry holds
const pantryCapacity = 30

// findSupply returns a supply, including the basic foods
func findSupply(id string) (supply, bool) {
	for _, s := range supplyCatalog {
		if s.item.ID == id {
			return s, true
		}
	}
	for _, s := range basicFoods {
		if s.item.ID == id {
			return s, true
		}
	}
	return supply{}, false
}

// forSpecies reports whether a supply is meant for a species
func (s supply) forSpecies(species Species) bool {
	return s.species == nil || slices.Contains(s.species, species)
}

// ensureSuppliesLocked gives the pet an empty pantry, and makes sure its
// lifetime points include points earned before they were counted.
// Caller must hold p.mu.
func (p *DigitalPet) ensureSuppliesLocked() {
	if p.Supplies == nil {
		p.Supplies = interaction.NewInventory(pantryCapacity)
		p.Supplies.Currency = 0
	}
	if p.PointsEarned < p.FriendshipPoints {
		p.PointsEarned = p.FriendshipPoints
	}
}

// supplyShopLocked builds a shop for the pet: its species' supplies, with
// those its lifetime points unlock marked unlocked, today's discounts, and
// its pantry. Caller must hold p.mu.
func (p *DigitalPet) supplyShopLocked() *interaction.Shop {
	p.ensureSuppliesLocked()
	catalog := &interaction.ItemCatalog{Items: make(map[string]*interaction.Item)}
	var ids []string
	for _, s := range supplyCatalog {
		if !s.forSpecies(p.speciesLocked()) {
			continue
		}
		item := s.item
		item.Unlocked = p.PointsEarned >= rarityUnlocks[item.Rarity]
		catalog.Items[item.ID] = &item
		if item.Unlocked {
			ids = append(ids, item.ID)
		}
	}

	shop := interaction.NewShop(catalog, p.Supplies)
	for _, id := range discountedToday(ids) {
		shop.SetDiscount(id, discountRate)
	}
	return shop
}

// discountedToday picks the day's discounted supplies from ids
func discountedToday(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	h := fnv.New32a()
	h.Write([]byte(today()))
	start := int(h.Sum32() % uint32(len(ids)))
	var picked []string
	for i := 0; i < dailyDiscounts && i < len(ids); i++ {
		picked = append(picked, ids[(start+i*3)%len(ids)])
	}
	slices.Sort(picked)
	return slices.Compact(picked)
}

// SupplyOffer is a supply as the shop shows it
type SupplyOffer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Rarity      string `json:"rarity"`
	Price       int    `json:"price"`      // After any discount
	FullPrice   int    `json:"full_price"` // Before any discount
	Discounted  bool   `json:"discounted"`
	Locked      bool   `json:"locked"`
	UnlockAt    int    `json:"unlock_at"` // Lifetime points that unlock it
	Owned       int    `json:"owned"`     // Units in the pantry
	Warning     string `json:"warning,omitempty"`
}

// SupplyOffers lists the supplies the pet's species can use, in shop order
func (p *DigitalPet) SupplyOffers() []SupplyOffer {
	p.mu.Lock()
	defer p.mu.Unlock()

	shop := p.supplyShopLocked()
	offers := []SupplyOffer{}
	for _, s := range supplyCatalog {
		item, ok := shop.Catalog.GetItem(s.item.ID)
		if !ok {
			continue
		}
		price, _ := shop.GetPrice(item.ID)
		offer := SupplyOffer{
			ID: item.ID, Name: item.Name, Icon: s.icon, Kind: s.kind, Description: item.Description,
			Rarity: item.Rarity.String(), Price: price, FullPrice: item.Price, Discounted: price < item.Price,
			Locked: !item.Unlocked, UnlockAt: rarityUnlocks[item.Rarity], Owned: p.Supplies.GetItemQuantity(item.ID),
		}
		if s.food != nil {
			offer.Warning = s.food.warning
		}
		offers = append(offers, offer)
	}
	return offers
}

// BuySupply spends Friendship Points on a pack of a supply for the pantry
func (p *DigitalPet) BuySupply(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Biology.IsAlive {
		return ErrPetNotAvailable
	}
	s, ok := findSupply(id)
	if !ok || !s.forSpecies(p.speciesLocked()) || s.pack == 0 {
		return ErrUnknownSupply
	}

	shop := p.supplyShopLocked()
	item, _ := shop.Catalog.GetItem(id)
	switch {
	case !item.Unlocked:
		return ErrSupplyLocked
	case p.Supplies.GetItemQuantity(id) == 0 && len(p.Supplies.Items) >= p.Supplies.MaxCapacity:
		return ErrPantryFull
	}

	// The shop takes payment from the pantry's currency: the pet's points
	p.Supplies.Currency = p.FriendshipPoints
	defer func() { p.Supplies.Currency = 0 }()
	if err := shop.Buy(id, 1); err != nil {
		return ErrNotEnoughPoints
	}
	p.FriendshipPoints = p.Supplies.Currency
	if s.pack > 1 {
		p.Supplies.Items[id].Quantity += s.pack - 1
	}
	return nil
}

// useSupplyLocked takes one unit of a food or treat from the pantry; the
// species' basic food never runs out. Caller must hold p.mu.
func (p *DigitalPet) useSupplyLocked(id string) error {
	if basic := basicFoods[p.speciesLocked()]; id == basic.item.ID {
		return nil
	}
	p.ensureSuppliesLocked()
	if p.Supplies.GetItemQuantity(id) == 0 {
		return ErrNoSupply
	}
	_, err := p.Supplies.UseItem(id)
	return err
}

// wearToyLocked wears a toy by one play session. A worn-out toy is thrown
// away, or replaced by the next in the pantry. It returns true if the toy
// wore out. Caller must hold p.mu.
func (p *DigitalPet) wearToyLocked(id string) bool {
	owned, ok := p.Supplies.Items[id]
	if !ok || owned.Item.MaxUses <= 0 {
		return false // Toys without a limit don't wear out
	}
	owned.TotalUsed++
	owned.Item.Uses--
	if owned.Item.Uses > 0 {
		return false
	}
	p.Supplies.RemoveItem(id, 1)
	if next, ok := p.Supplies.Items[id]; ok {
		next.Item.Uses = next.Item.MaxUses
	}
	return true
}

// PantryItem is a food, treat or toy the pet can be given now
type PantryItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Kind     string `json:"kind"`
	Quantity int    `json:"quantity"`            // -1 for food that never runs out
	UsesLeft int    `json:"uses_left,omitempty"` // Play sessions left in the current toy
	Warning  string `json:"warning,omitempty"`
}

// Pantry lists what the pet can be given: its basic food first, then the
// supplies in its pantry, in shop order
func (p *DigitalPet) Pantry() []PantryItem {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ensureSuppliesLocked()

	basic := basicFoods[p.speciesLocked()]
	items := []PantryItem{{ID: basic.item.ID, Name: basic.item.Name, Icon: basic.icon, Kind: SupplyFood, Quantity: -1}}
	for _, s := range supplyCatalog {
		owned, ok := p.Supplies.Items[s.item.ID]
		if !ok || !s.forSpecies(p.speciesLocked()) {
			continue
		}
		item := PantryItem{ID: s.item.ID, Name: s.item.Name, Icon: s.icon, Kind: s.kind, Quantity: owned.Quantity}
		if s.toy != nil && owned.Item.MaxUses > 0 {
			item.UsesLeft = owned.Item.Uses
		}
		if s.food != nil {
			item.Warning = s.food.warning
		}
		items = append(items, item)
	}
	return items
}
