package interaction

import (
	"fmt"
	"time"
)

// ItemType represents different categories of items
type ItemType int

const (
	ItemTypeFood ItemType = iota
	ItemTypeToy
	ItemTypeMedicine
	ItemTypeAccessory
	ItemTypeGrooming
)

// String returns the string representation of ItemType
func (it ItemType) String() string {
	return [...]string{"Food", "Toy", "Medicine", "Accessory", "Grooming"}[it]
}

// ItemRarity represents item rarity levels
type ItemRarity int

const (
	RarityCommon ItemRarity = iota
	RarityUncommon
	RarityRare
	RarityEpic
	RarityLegendary
)

// String returns the string representation of ItemRarity
func (ir ItemRarity) String() string {
	return [...]string{"Common", "Uncommon", "Rare", "Epic", "Legendary"}[ir]
}

// Item represents an interactable item
type Item struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Type        ItemType            `json:"type"`
	Rarity      ItemRarity          `json:"rarity"`
	Effects     map[string]float64  `json:"effects"`
	Duration    float64             `json:"duration"`     // For toys/accessories, how long effect lasts
	Uses        int                 `json:"uses"`         // -1 for unlimited (toys)
	MaxUses     int                 `json:"max_uses"`
	Cooldown    float64             `json:"cooldown"`     // Seconds between uses
	Price       int                 `json:"price"`
	Unlocked    bool                `json:"unlocked"`
}

// InventoryItem represents an item in the player's inventory
type InventoryItem struct {
	Item        *Item     `json:"item"`
	Quantity    int       `json:"quantity"`
	LastUsed    time.Time `json:"last_used"`
	TotalUsed   int       `json:"total_used"`
}

// Inventory manages the player's items
type Inventory struct {
	Items       map[string]*InventoryItem `json:"items"`
	MaxCapacity int                        `json:"max_capacity"`
	Currency    int                        `json:"currency"`
}

// NewInventory creates a new inventory
func NewInventory(capacity int) *Inventory {
	return &Inventory{
		Items:       make(map[string]*InventoryItem),
		MaxCapacity: capacity,
		Currency:    100, // Starting currency
	}
}

// AddItem adds an item to the inventory
func (inv *Inventory) AddItem(item *Item, quantity int) bool {
	if len(inv.Items) >= inv.MaxCapacity {
		// Check if we're adding to existing stack
		if existing, exists := inv.Items[item.ID]; exists {
			existing.Quantity += quantity
			return true
		}
		return false // Inventory full
	}

	if existing, exists := inv.Items[item.ID]; exists {
		existing.Quantity += quantity
	} else {
		inv.Items[item.ID] = &InventoryItem{
			Item:     item,
			Quantity: quantity,
		}
	}

	return true
}

// RemoveItem removes items from inventory
func (inv *Inventory) RemoveItem(itemID string, quantity int) bool {
	item, exists := inv.Items[itemID]
	if !exists {
		return false
	}

	if item.Quantity < quantity {
		return false
	}

	item.Quantity -= quantity
	if item.Quantity <= 0 {
		delete(inv.Items, itemID)
	}

	return true
}

// UseItem uses an item and returns its effects
func (inv *Inventory) UseItem(itemID string) (*Item, error) {
	invItem, exists := inv.Items[itemID]
	if !exists {
		return nil, fmt.Errorf("item not found in inventory")
	}

	item := invItem.Item

	// Check cooldown
	if time.Since(invItem.LastUsed).Seconds() < item.Cooldown {
		remaining := item.Cooldown - time.Since(invItem.LastUsed).Seconds()
		return nil, fmt.Errorf("item on cooldown (%.1f seconds remaining)", remaining)
	}

	// Check uses
	if item.MaxUses > 0 && item.Uses <= 0 {
		return nil, fmt.Errorf("item has no uses remaining")
	}

	// Consume item if it has limited uses
	if item.Type == ItemTypeFood || item.Type == ItemTypeMedicine {
		invItem.Quantity--
		if invItem.Quantity <= 0 {
			delete(inv.Items, itemID)
		}
	} else if item.MaxUses > 0 {
		item.Uses--
	}

	invItem.LastUsed = time.Now()
	invItem.TotalUsed++

	return item, nil
}

// HasItem checks if inventory contains an item
func (inv *Inventory) HasItem(itemID string) bool {
	_, exists := inv.Items[itemID]
	return exists
}

// GetItemQuantity returns the quantity of an item
func (inv *Inventory) GetItemQuantity(itemID string) int {
	if item, exists := inv.Items[itemID]; exists {
		return item.Quantity
	}
	return 0
}

// GetItemsByType returns all items of a specific type
func (inv *Inventory) GetItemsByType(itemType ItemType) []*InventoryItem {
	var result []*InventoryItem
	for _, item := range inv.Items {
		if item.Item.Type == itemType {
			result = append(result, item)
		}
	}
	return result
}

// CanAfford checks if player can afford an item
func (inv *Inventory) CanAfford(price int) bool {
	return inv.Currency >= price
}

// SpendCurrency reduces currency
func (inv *Inventory) SpendCurrency(amount int) bool {
	if inv.Currency < amount {
		return false
	}
	inv.Currency -= amount
	return true
}

// AddCurrency adds currency
func (inv *Inventory) AddCurrency(amount int) {
	inv.Currency += amount
}

// ItemCatalog contains all available items
type ItemCatalog struct {
	Items map[string]*Item
}

// NewItemCatalog creates a catalog with default items
func NewItemCatalog() *ItemCatalog {
	catalog := &ItemCatalog{
		Items: make(map[string]*Item),
	}

	// Add default items
	catalog.initializeDefaultItems()

	return catalog
}

// initializeDefaultItems adds all default items to the catalog
func (ic *ItemCatalog) initializeDefaultItems() {
	// Foods
	ic.Items["basic_food"] = &Item{
		ID:          "basic_food",
		Name:        "Basic Pet Food",
		Description: "Standard nutritious pet food",
		Type:        ItemTypeFood,
		Rarity:      RarityCommon,
		Effects: map[string]float64{
			"hunger":    -0.3,
			"energy":    0.1,
			"happiness": 0.05,
		},
		Price:    10,
		Unlocked: true,
	}

	ic.Items["premium_food"] = &Item{
		ID:          "premium_food",
		Name:        "Premium Pet Food",
		Description: "High-quality gourmet pet food",
		Type:        ItemTypeFood,
		Rarity:      RarityUncommon,
		Effects: map[string]float64{
			"hunger":    -0.4,
			"energy":    0.15,
			"happiness": 0.1,
			"health":    0.02,
		},
		Price:    25,
		Unlocked: false,
	}

	ic.Items["treat"] = &Item{
		ID:          "treat",
		Name:        "Tasty Treat",
		Description: "A delicious snack for rewards",
		Type:        ItemTypeFood,
		Rarity:      RarityCommon,
		Effects: map[string]float64{
			"hunger":    -0.1,
			"happiness": 0.15,
		},
		Price:    5,
		Unlocked: true,
	}

	ic.Items["gourmet_meal"] = &Item{
		ID:          "gourmet_meal",
		Name:        "Gourmet Meal",
		Description: "An exquisite meal fit for royalty",
		Type:        ItemTypeFood,
		Rarity:      RarityRare,
		Effects: map[string]float64{
			"hunger":    -0.5,
			"energy":    0.2,
			"happiness": 0.2,
			"health":    0.05,
		},
		Price:    50,
		Unlocked: false,
	}

	// Toys
	ic.Items["ball"] = &Item{
		ID:          "ball",
		Name:        "Bouncy Ball",
		Description: "A simple but fun ball to play with",
		Type:        ItemTypeToy,
		Rarity:      RarityCommon,
		Effects: map[string]float64{
			"happiness": 0.15,
			"energy":    -0.1,
			"exercise":  0.2,
		},
		Uses:     -1, // Unlimited
		Cooldown: 30,
		Price:    15,
		Unlocked: true,
	}

	ic.Items["rope_toy"] = &Item{
		ID:          "rope_toy",
		Name:        "Rope Toy",
		Description: "Great for tug-of-war games",
		Type:        ItemTypeToy,
		Rarity:      RarityCommon,
		Effects: map[string]float64{
			"happiness": 0.2,
			"energy":    -0.15,
			"strength":  0.05,
		},
		Uses:     -1,
		Cooldown: 30,
		Price:    20,
		Unlocked: false,
	}

	ic.Items["puzzle_toy"] = &Item{
		ID:          "puzzle_toy",
		Name:        "Puzzle Toy",
		Description: "Stimulates mental activity",
		Type:        ItemTypeToy,
		Rarity:      RarityUncommon,
		Effects: map[string]float64{
			"happiness":          0.1,
			"mental_stimulation": 0.3,
			"intelligence":       0.02,
		},
		Uses:     -1,
		Cooldown: 60,
		Price:    35,
		Unlocked: false,
	}

	ic.Items["squeaky_toy"] = &Item{
		ID:          "squeaky_toy",
		Name:        "Squeaky Toy",
		Description: "Makes exciting noises!",
		Type:        ItemTypeToy,
		Rarity:      RarityCommon,
		Effects: map[string]float64{
			"happiness": 0.25,
			"energy":    -0.05,
		},
		Uses:     50, // Wears out eventually
		MaxUses:  50,
		Cooldown: 20,
		Price:    12,
		Unlocked: true,
	}

	// Medicine
	ic.Items["basic_medicine"] = &Item{
		ID:          "basic_medicine",
		Name:        "Basic Medicine",
		Description: "Treats minor ailments",
		Type:        ItemTypeMedicine,
		Rarity:      RarityCommon,
		Effects: map[string]float64{
			"health": 0.2,
			"stress": 0.05, // Medicine is a bit stressful
		},
		Price:    30,
		Unlocked: true,
	}

	ic.Items["vitamin_supplement"] = &Item{
		ID:          "vitamin_supplement",
		Name:        "Vitamin Supplement",
		Description: "Boosts overall health",
		Type:        ItemTypeMedicine,
		Rarity:      RarityUncommon,
		Effects: map[string]float64{
			"health":   0.1,
			"energy":   0.1,
			"immunity": 0.05,
		},
		Cooldown: 3600, // Once per hour
		Price:    40,
		Unlocked: false,
	}

	ic.Items["advanced_medicine"] = &Item{
		ID:          "advanced_medicine",
		Name:        "Advanced Medicine",
		Description: "Powerful treatment for serious illness",
		Type:        ItemTypeMedicine,
		Rarity:      RarityRare,
		Effects: map[string]float64{
			"health": 0.4,
		},
		Price:    75,
		Unlocked: false,
	}

	// Grooming items
	ic.Items["basic_brush"] = &Item{
		ID:          "basic_brush",
		Name:        "Basic Brush",
		Description: "For regular grooming",
		Type:        ItemTypeGrooming,
		Rarity:      RarityCommon,
		Effects: map[string]float64{
			"cleanliness": 0.3,
			"happiness":   0.05,
		},
		Uses:     -1,
		Cooldown: 60,
		Price:    15,
		Unlocked: true,
	}

	ic.Items["deluxe_grooming_kit"] = &Item{
		ID:          "deluxe_grooming_kit",
		Name:        "Deluxe Grooming Kit",
		Description: "Complete grooming solution",
		Type:        ItemTypeGrooming,
		Rarity:      RarityUncommon,
		Effects: map[string]float64{
			"cleanliness": 0.5,
			"happiness":   0.1,
			"health":      0.02,
		},
		Uses:     -1,
		Cooldown: 120,
		Price:    45,
		Unlocked: false,
	}

	// Accessories
	ic.Items["collar"] = &Item{
		ID:          "collar",
		Name:        "Basic Collar",
		Description: "A simple but stylish collar",
		Type:        ItemTypeAccessory,
		Rarity:      RarityCommon,
		Effects: map[string]float64{
			"style": 0.1,
		},
		Duration: -1, // Permanent while equipped
		Price:    20,
		Unlocked: true,
	}

	ic.Items["fancy_bow"] = &Item{
		ID:          "fancy_bow",
		Name:        "Fancy Bow",
		Description: "An adorable decorative bow",
		Type:        ItemTypeAccessory,
		Rarity:      RarityUncommon,
		Effects: map[string]float64{
			"style":     0.2,
			"happiness": 0.05,
		},
		Duration: -1,
		Price:    35,
		Unlocked: false,
	}
}

// GetItem retrieves an item from the catalog
func (ic *ItemCatalog) GetItem(itemID string) (*Item, bool) {
	item, exists := ic.Items[itemID]
	return item, exists
}

// GetItemsByType returns all items of a specific type
func (ic *ItemCatalog) GetItemsByType(itemType ItemType) []*Item {
	var result []*Item
	for _, item := range ic.Items {
		if item.Type == itemType {
			result = append(result, item)
		}
	}
	return result
}

// GetUnlockedItems returns all unlocked items
func (ic *ItemCatalog) GetUnlockedItems() []*Item {
	var result []*Item
	for _, item := range ic.Items {
		if item.Unlocked {
			result = append(result, item)
		}
	}
	return result
}

// UnlockItem unlocks an item in the catalog
func (ic *ItemCatalog) UnlockItem(itemID string) bool {
	if item, exists := ic.Items[itemID]; exists {
		item.Unlocked = true
		return true
	}
	return false
}

// GetItemsByRarity returns items of a specific rarity
func (ic *ItemCatalog) GetItemsByRarity(rarity ItemRarity) []*Item {
	var result []*Item
	for _, item := range ic.Items {
		if item.Rarity == rarity {
			result = append(result, item)
		}
	}
	return result
}

// Shop provides item purchasing functionality
type Shop struct {
	Catalog   *ItemCatalog
	Inventory *Inventory
	Discounts map[string]float64 // Item ID -> discount percentage
}

// NewShop creates a new shop
func NewShop(catalog *ItemCatalog, inventory *Inventory) *Shop {
	return &Shop{
		Catalog:   catalog,
		Inventory: inventory,
		Discounts: make(map[string]float64),
	}
}

// GetPrice returns the price of an item (with any discounts)
func (s *Shop) GetPrice(itemID string) (int, error) {
	item, exists := s.Catalog.GetItem(itemID)
	if !exists {
		return 0, fmt.Errorf("item not found")
	}

	price := float64(item.Price)
	if discount, exists := s.Discounts[itemID]; exists {
		price *= (1.0 - discount)
	}

	return int(price), nil
}

// Buy purchases an item
func (s *Shop) Buy(itemID string, quantity int) error {
	item, exists := s.Catalog.GetItem(itemID)
	if !exists {
		return fmt.Errorf("item not found")
	}

	if !item.Unlocked {
		return fmt.Errorf("item not unlocked")
	}

	price, _ := s.GetPrice(itemID)
	totalCost := price * quantity

	if !s.Inventory.CanAfford(totalCost) {
		return fmt.Errorf("not enough currency")
	}

	if !s.Inventory.AddItem(item, quantity) {
		return fmt.Errorf("inventory full")
	}

	s.Inventory.SpendCurrency(totalCost)
	return nil
}

// SetDiscount sets a discount for an item
func (s *Shop) SetDiscount(itemID string, discountPercent float64) {
	if discountPercent < 0 {
		discountPercent = 0
	}
	if discountPercent > 1 {
		discountPercent = 1
	}
	s.Discounts[itemID] = discountPercent
}

// ClearDiscount removes a discount
func (s *Shop) ClearDiscount(itemID string) {
	delete(s.Discounts, itemID)
}

// GetAvailableItems returns items that can be purchased
func (s *Shop) GetAvailableItems() []*Item {
	return s.Catalog.GetUnlockedItems()
}
