package core

import (
	"errors"
	"math"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Friendship Points are earned by caring for a pet and spent on items for
// that pet. Each kind of care earns points at most once per cooldown, so
// repeated clicking cares for the pet but does not earn more.
const (
	// PointsPerCare is the number of Friendship Points one rewarded
	// interaction earns
	PointsPerCare = 5

	// CareRewardCooldown is how long after a rewarded interaction the same
	// kind of interaction earns points again
	CareRewardCooldown = 3 * time.Minute
)

// now is the clock used for reward cooldowns (replaced in tests)
var now = time.Now

// rewardedCare lists the interactions that earn Friendship Points
var rewardedCare = map[types.InteractionType]bool{
	types.InteractionFeeding:     true,
	types.InteractionDrinking:    true,
	types.InteractionPetting:     true,
	types.InteractionPlaying:     true,
	types.InteractionGrooming:    true,
	types.InteractionMedicalCare: true,
	types.InteractionRewards:     true,
}

// ItemCategory groups shop items. Within a category with an effect, the
// strongest owned item applies.
type ItemCategory string

const (
	CategoryFoodBowl   ItemCategory = "food_bowl"
	CategoryWaterBowl  ItemCategory = "water_bowl"
	CategoryToy        ItemCategory = "toy"
	CategoryBed        ItemCategory = "bed"
	CategoryBrush      ItemCategory = "brush"
	CategoryDecoration ItemCategory = "decoration"
)

// ShopItem is something that can be bought with Friendship Points
type ShopItem struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Icon        string       `json:"icon"`
	Category    ItemCategory `json:"category"`
	Price       int          `json:"price"`
	Description string       `json:"description"`

	// Effect strength: a decay multiplier for bowls and beds (below 1.0
	// slows the need) or an effectiveness multiplier for toys and brushes
	// (above 1.0 strengthens the interaction). 0 for decorations.
	Effect float64 `json:"-"`
}

// shopCatalog lists every item in display order
var shopCatalog = []ShopItem{
	{ID: "food_bowl", Name: "Sturdy Food Bowl", Icon: "🥣", Category: CategoryFoodBowl, Price: 100, Effect: 0.8,
		Description: "Hunger builds 20% slower."},
	{ID: "auto_feeder", Name: "Automatic Feeder", Icon: "🍱", Category: CategoryFoodBowl, Price: 250, Effect: 0.6,
		Description: "Hunger builds 40% slower."},
	{ID: "water_bowl", Name: "Big Water Bowl", Icon: "🥛", Category: CategoryWaterBowl, Price: 100, Effect: 0.8,
		Description: "Thirst builds 20% slower."},
	{ID: "water_fountain", Name: "Water Fountain", Icon: "⛲", Category: CategoryWaterBowl, Price: 250, Effect: 0.6,
		Description: "Thirst builds 40% slower."},
	{ID: "squeaky_toy", Name: "Squeaky Toy", Icon: "🧸", Category: CategoryToy, Price: 80, Effect: 1.25,
		Description: "Playing is 25% more fun."},
	{ID: "puzzle_ball", Name: "Puzzle Ball", Icon: "⚽", Category: CategoryToy, Price: 200, Effect: 1.5,
		Description: "Playing is 50% more fun."},
	{ID: "cozy_bed", Name: "Cozy Bed", Icon: "🛏️", Category: CategoryBed, Price: 150, Effect: 0.75,
		Description: "Tiredness builds 25% slower."},
	{ID: "soft_brush", Name: "Soft Brush", Icon: "🪮", Category: CategoryBrush, Price: 80, Effect: 1.5,
		Description: "Grooming is 50% more effective."},
	{ID: "balloon", Name: "Balloon", Icon: "🎈", Category: CategoryDecoration, Price: 30,
		Description: "A cheerful decoration."},
	{ID: "plant", Name: "Potted Plant", Icon: "🪴", Category: CategoryDecoration, Price: 40,
		Description: "A leafy decoration."},
	{ID: "lamp", Name: "Night Lamp", Icon: "💡", Category: CategoryDecoration, Price: 50,
		Description: "A warm glow for your pet's room."},
	{ID: "painting", Name: "Painting", Icon: "🖼️", Category: CategoryDecoration, Price: 60,
		Description: "Art for your pet's wall."},
	{ID: "rainbow", Name: "Rainbow Mural", Icon: "🌈", Category: CategoryDecoration, Price: 120,
		Description: "A bright mural for a well-loved pet."},
}

// Shop errors
var (
	ErrUnknownItem     = errors.New("unknown item")
	ErrAlreadyOwned    = errors.New("item already owned")
	ErrNotEnoughPoints = errors.New("not enough Friendship Points")
	ErrPetNotAvailable = errors.New("pet is not available")
)

// ShopCatalog returns every item that can be bought
func ShopCatalog() []ShopItem {
	return append([]ShopItem(nil), shopCatalog...)
}

// findItem returns the catalog item with the given ID
func findItem(id string) (ShopItem, bool) {
	for _, item := range shopCatalog {
		if item.ID == id {
			return item, true
		}
	}
	return ShopItem{}, false
}

// InteractionResult reports the outcome of an interaction
type InteractionResult struct {
	Applied      bool         // False if the pet could not respond (hibernating or not alive)
	PointsEarned int          // Friendship Points earned by this interaction
	Changes      []StatChange // How the pet's stats changed
	BoostedBy    *ShopItem    // Item that made this interaction more effective, if any

	// How the pet reacted, which scales the interaction's benefits by
	// ReactionMultiplier (see reactionLocked)
	Reaction           ReactionLevel
	ReactionText       string
	ReactionMultiplier float64
}

// StatChange is the change in one of the pet's stats (-1.0 to 1.0)
type StatChange struct {
	Stat   string  `json:"stat"`
	Change float64 `json:"change"`
}

// minReportedChange is the smallest stat change worth reporting
const minReportedChange = 0.005

// statChanges lists the stats that changed between two snapshots, in
// display order. Hunger is reported as it is shown: the opposite of
// nutrition.
func statChanges(before, after biology.VitalStats) []StatChange {
	all := []StatChange{
		{"Health", after.Health - before.Health},
		{"Energy", after.Energy - before.Energy},
		{"Happiness", after.Happiness - before.Happiness},
		{"Hunger", before.Nutrition - after.Nutrition},
		{"Water", after.Hydration - before.Hydration},
		{"Cleanliness", after.Cleanliness - before.Cleanliness},
		{"Stress", after.Stress - before.Stress},
	}

	changes := []StatChange{}
	for _, c := range all {
		if math.Abs(c.Change) >= minReportedChange {
			changes = append(changes, c)
		}
	}
	return changes
}

// ActiveEffects returns the items whose effects currently apply to the
// pet: the best owned item in each category with an effect. Items that
// are outclassed by a better one, and decorations, are not included.
func (p *DigitalPet) ActiveEffects() []ShopItem {
	p.mu.Lock()
	defer p.mu.Unlock()

	effects := []ShopItem{}
	for _, c := range []struct {
		category ItemCategory
		lower    bool
	}{
		{CategoryFoodBowl, true},
		{CategoryWaterBowl, true},
		{CategoryToy, false},
		{CategoryBed, true},
		{CategoryBrush, false},
	} {
		if item, ok := p.bestItemLocked(c.category, c.lower); ok {
			effects = append(effects, item)
		}
	}
	return effects
}

// Interact applies an interaction like ProcessUserInteraction and reports
// the Friendship Points it earned
func (p *DigitalPet) Interact(interactionType types.InteractionType, intensity float64) InteractionResult {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.interactLocked(interactionType, intensity)
}

// awardCareLocked grants Friendship Points for an interaction if its
// cooldown has passed. Caller must hold p.mu.
func (p *DigitalPet) awardCareLocked(interactionType types.InteractionType) int {
	if !rewardedCare[interactionType] {
		return 0
	}

	t := now()
	if last, ok := p.LastCareReward[interactionType]; ok && t.Sub(last) < CareRewardCooldown {
		return 0
	}

	if p.LastCareReward == nil {
		p.LastCareReward = make(map[types.InteractionType]time.Time)
	}
	p.LastCareReward[interactionType] = t
	p.FriendshipPoints += PointsPerCare
	return PointsPerCare
}

// BuyItem spends Friendship Points on an item for this pet
func (p *DigitalPet) BuyItem(itemID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	item, ok := findItem(itemID)
	if !ok {
		return ErrUnknownItem
	}
	if !p.Biology.IsAlive {
		return ErrPetNotAvailable
	}
	if p.ownsLocked(itemID) {
		return ErrAlreadyOwned
	}
	if p.FriendshipPoints < item.Price {
		return ErrNotEnoughPoints
	}

	p.FriendshipPoints -= item.Price
	p.Inventory = append(p.Inventory, itemID)
	return nil
}

// Owns reports whether the pet has the item
func (p *DigitalPet) Owns(itemID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ownsLocked(itemID)
}

// ownsLocked reports whether the pet has the item. Caller must hold p.mu.
func (p *DigitalPet) ownsLocked(itemID string) bool {
	for _, owned := range p.Inventory {
		if owned == itemID {
			return true
		}
	}
	return false
}

// OwnedItems returns the catalog entries of the items the pet has
func (p *DigitalPet) OwnedItems() []ShopItem {
	p.mu.Lock()
	defer p.mu.Unlock()

	var items []ShopItem
	for _, item := range shopCatalog {
		if p.ownsLocked(item.ID) {
			items = append(items, item)
		}
	}
	return items
}

// bestEffectLocked returns the strongest effect among owned items in a
// category, or 1.0 if none is owned. lower selects the smallest effect
// (decay multipliers) instead of the largest (effectiveness multipliers).
// Caller must hold p.mu.
func (p *DigitalPet) bestEffectLocked(category ItemCategory, lower bool) float64 {
	if item, ok := p.bestItemLocked(category, lower); ok {
		return item.Effect
	}
	return 1.0
}

// bestItemLocked returns the owned item with the strongest effect in a
// category. Caller must hold p.mu.
func (p *DigitalPet) bestItemLocked(category ItemCategory, lower bool) (ShopItem, bool) {
	var best ShopItem
	found := false
	for _, item := range shopCatalog {
		if item.Category != category || !p.ownsLocked(item.ID) {
			continue
		}
		if !found || (lower && item.Effect < best.Effect) || (!lower && item.Effect > best.Effect) {
			best, found = item, true
		}
	}
	return best, found
}

// decayMultipliersLocked returns how the pet's belongings slow its needs.
// Caller must hold p.mu.
func (p *DigitalPet) decayMultipliersLocked() biology.DecayMultipliers {
	return biology.DecayMultipliers{
		Nutrition: p.bestEffectLocked(CategoryFoodBowl, true),
		Hydration: p.bestEffectLocked(CategoryWaterBowl, true),
		Fatigue:   p.bestEffectLocked(CategoryBed, true),
	}
}

// careBoostLocked returns the owned item that strengthens an interaction,
// if any. Caller must hold p.mu.
func (p *DigitalPet) careBoostLocked(interactionType types.InteractionType) (ShopItem, bool) {
	switch interactionType {
	case types.InteractionPlaying:
		return p.bestItemLocked(CategoryToy, false)
	case types.InteractionGrooming:
		return p.bestItemLocked(CategoryBrush, false)
	default:
		return ShopItem{}, false
	}
}
