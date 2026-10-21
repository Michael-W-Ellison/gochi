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
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Type        ItemType           `json:"type"`
	Rarity      ItemRarity         `json:"rarity"`
	Effects     map[string]float64 `json:"effects"`
	Duration    float64            `json:"duration"` // For toys/accessories, how long effect lasts
	Uses        int                `json:"uses"`     // -1 for unlimited (toys)
	MaxUses     int                `json:"max_uses"`
	Cooldown    float64            `json:"cooldown"` // Seconds between uses
	Price       int                `json:"price"`
	Unlocked    bool               `json:"unlocked"`
}

// InventoryItem represents an item in the player's inventory
type InventoryItem struct {
	Item      *Item     `json:"item"`
	Quantity  int       `json:"quantity"`
	LastUsed  time.Time `json:"last_used"`
	TotalUsed int       `json:"total_used"`
}

// Inventory manages the player's items
type Inventory struct {
	Items       map[string]*InventoryItem `json:"items"`
	MaxCapacity int                       `json:"max_capacity"`
	Currency    int                       `json:"currency"`
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

// GetItem retrieves an item from the catalog
func (ic *ItemCatalog) GetItem(itemID string) (*Item, bool) {
	item, exists := ic.Items[itemID]
	return item, exists
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
