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
	SupplyFood   = "food"
	SupplyTreat  = "treat"
	SupplyToy    = "toy"
	SupplyHealth = "health" // Medicine the owner gives (see GiveLifePotion)
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
	holiday string // Sold only around this holiday
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

// sharedSupplies are the supplies for every species
var sharedSupplies = []supply{
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
	// Toys for both
	{item: interaction.Item{ID: "puzzle_feeder", Name: "Puzzle feeder", Price: 35, Rarity: interaction.RarityRare,
		Type: interaction.ItemTypeToy, Description: "A toy that releases food as it's worked out."},
		icon: "🧩", kind: SupplyToy, pack: 1,
		toy: &toyProfile{fit: 0.1, exercise: 0.3, stimulation: 1.2, calming: 0.05,
			lesson: "Puzzle feeders make a pet work for its food, which keeps its mind busy and slows down fast eaters."}},
	// Health
	{item: interaction.Item{ID: LifePotionID, Name: "Life-extension potion", Price: 1000, Rarity: interaction.RarityLegendary,
		Type: interaction.ItemTypeMedicine, Description: "Adds about 1,500 game days to your pet's life. Give it from the Body panel."},
		icon: "🧪", kind: SupplyHealth, pack: 1},
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
const pantryCapacity = 40

// supplyKinds order the shop
var supplyKinds = []string{SupplyFood, SupplyTreat, SupplyToy, SupplyCare, SupplyHealth}

// supplyCatalog lists every supply in shop order: by kind, each species'
// own before the shared ones
func supplyCatalog() []supply {
	var all []supply
	for _, def := range SpeciesList() {
		all = append(all, def.Supplies...)
	}
	all = append(all, sharedSupplies...)
	all = append(all, careSupplies...)
	all = append(all, holidaySupplies...)
	catalog := make([]supply, 0, len(all))
	for _, kind := range supplyKinds {
		for _, s := range all {
			if s.kind == kind {
				catalog = append(catalog, s)
			}
		}
	}
	return catalog
}

// findSupply returns a supply, including the basic foods
func findSupply(id string) (supply, bool) {
	for _, s := range supplyCatalog() {
		if s.item.ID == id {
			return s, true
		}
	}
	for _, def := range SpeciesList() {
		if def.BasicFood.item.ID == id {
			return def.BasicFood, true
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
	p.Supplies.MaxCapacity = max(p.Supplies.MaxCapacity, pantryCapacity) // Pantries grew with more supplies
	if earned := p.FriendshipPoints - p.GiftedPoints; p.PointsEarned < earned {
		p.PointsEarned = earned
	}
}

// supplyShopLocked builds a shop for the pet: its species' supplies, with
// those its lifetime points unlock marked unlocked, today's discounts, and
// its pantry. Caller must hold p.mu.
func (p *DigitalPet) supplyShopLocked() *interaction.Shop {
	p.ensureSuppliesLocked()
	catalog := &interaction.ItemCatalog{Items: make(map[string]*interaction.Item)}
	var ids []string
	for _, s := range supplyCatalog() {
		if !s.forSpecies(p.speciesLocked()) || (s.holiday != "" && !HolidayOpen(s.holiday)) {
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
	Holiday     string `json:"holiday,omitempty"` // Sold only around this holiday
}

// SupplyOffers lists the supplies the pet's species can use, in shop order
func (p *DigitalPet) SupplyOffers() []SupplyOffer {
	p.mu.Lock()
	defer p.mu.Unlock()

	shop := p.supplyShopLocked()
	offers := []SupplyOffer{}
	for _, s := range supplyCatalog() {
		item, ok := shop.Catalog.GetItem(s.item.ID)
		if !ok {
			continue
		}
		price, _ := shop.GetPrice(item.ID)
		offer := SupplyOffer{
			ID: item.ID, Name: item.Name, Icon: s.icon, Kind: s.kind, Description: item.Description,
			Rarity: item.Rarity.String(), Price: price, FullPrice: item.Price, Discounted: price < item.Price,
			Locked: !item.Unlocked, UnlockAt: rarityUnlocks[item.Rarity], Owned: p.Supplies.GetItemQuantity(item.ID),
			Holiday: s.holiday,
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
	if basic := p.defLocked().BasicFood; id == basic.item.ID {
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

	basic := p.defLocked().BasicFood
	items := []PantryItem{{ID: basic.item.ID, Name: basic.item.Name, Icon: basic.icon, Kind: SupplyFood, Quantity: -1}}
	for _, s := range supplyCatalog() {
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
