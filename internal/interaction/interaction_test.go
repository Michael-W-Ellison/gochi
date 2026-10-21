package interaction

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Helper function to create a default test context
func defaultTestContext() InteractionContext {
	return InteractionContext{
		Health:          0.8,
		Energy:          0.7,
		Happiness:       0.6,
		Hunger:          0.4,
		Thirst:          0.3,
		Cleanliness:     0.7,
		Stress:          0.2,
		Fatigue:         0.3,
		Playfulness:     0.5,
		Independence:    0.5,
		Affectionate:    0.5,
		Intelligence:    0.5,
		CurrentBehavior: types.BehaviorIdle,
		PetAge:          30.0,
		IsSleeping:      false,
		IsSick:          false,
		TimeOfDay:       12.0,
		IsOutdoors:      false,
	}
}

// TestGetInteractionDescription tests description retrieval
func TestGetInteractionDescription(t *testing.T) {
	desc := GetInteractionDescription(types.InteractionFeeding)
	if desc == "" {
		t.Error("Expected non-empty description for feeding")
	}

	desc = GetInteractionDescription(types.InteractionType(999))
	if desc != "Interact with your pet" {
		t.Errorf("Expected default description for unknown type, got: %s", desc)
	}
}

// TestValidateInteraction tests interaction validation
func TestValidateInteraction(t *testing.T) {
	tests := []struct {
		name        string
		interaction types.InteractionType
		ctx         InteractionContext
		wantValid   bool
	}{
		{
			name:        "playing with energy",
			interaction: types.InteractionPlaying,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Energy = 0.5
				return c
			}(),
			wantValid: true,
		},
		{
			name:        "playing tired",
			interaction: types.InteractionPlaying,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Energy = 0.1
				return c
			}(),
			wantValid: false,
		},
		{
			name:        "training normal",
			interaction: types.InteractionTraining,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Energy = 0.5
				c.Stress = 0.3
				return c
			}(),
			wantValid: true,
		},
		{
			name:        "training stressed",
			interaction: types.InteractionTraining,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Stress = 0.9
				return c
			}(),
			wantValid: false,
		},
		{
			name:        "feeding hungry",
			interaction: types.InteractionFeeding,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Hunger = 0.3
				return c
			}(),
			wantValid: true,
		},
		{
			name:        "feeding full",
			interaction: types.InteractionFeeding,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Hunger = 0.95
				return c
			}(),
			wantValid: false,
		},
		{
			name:        "grooming dirty",
			interaction: types.InteractionGrooming,
			ctx:         defaultTestContext(),
			wantValid:   true,
		},
		{
			name:        "grooming clean",
			interaction: types.InteractionGrooming,
			ctx: func() InteractionContext {
				c := defaultTestContext()
				c.Cleanliness = 0.99
				return c
			}(),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := ValidateInteraction(tt.interaction, tt.ctx)
			if valid != tt.wantValid {
				t.Errorf("ValidateInteraction() valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

// =============== Item and Inventory Tests ===============

// TestNewInventory tests inventory creation
func TestNewInventory(t *testing.T) {
	inv := NewInventory(20)

	if inv == nil {
		t.Fatal("NewInventory returned nil")
	}

	if inv.MaxCapacity != 20 {
		t.Errorf("Expected capacity 20, got %d", inv.MaxCapacity)
	}

	if inv.Currency != 100 {
		t.Errorf("Expected starting currency 100, got %d", inv.Currency)
	}

	if inv.Items == nil {
		t.Error("Items map should not be nil")
	}
}

// TestAddItem tests adding items to inventory
func TestAddItem(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{
		ID:   "test_item",
		Name: "Test Item",
		Type: ItemTypeFood,
	}

	success := inv.AddItem(item, 5)
	if !success {
		t.Error("AddItem should succeed")
	}

	if inv.GetItemQuantity("test_item") != 5 {
		t.Errorf("Expected quantity 5, got %d", inv.GetItemQuantity("test_item"))
	}

	// Add more of same item
	success = inv.AddItem(item, 3)
	if !success {
		t.Error("Adding more of same item should succeed")
	}

	if inv.GetItemQuantity("test_item") != 8 {
		t.Errorf("Expected quantity 8, got %d", inv.GetItemQuantity("test_item"))
	}
}

// TestAddItemInventoryFull tests adding items when inventory is full
func TestAddItemInventoryFull(t *testing.T) {
	inv := NewInventory(2)

	item1 := &Item{ID: "item1", Name: "Item 1"}
	item2 := &Item{ID: "item2", Name: "Item 2"}
	item3 := &Item{ID: "item3", Name: "Item 3"}

	inv.AddItem(item1, 1)
	inv.AddItem(item2, 1)

	// Should fail - inventory full
	success := inv.AddItem(item3, 1)
	if success {
		t.Error("Adding item to full inventory should fail")
	}

	// But adding to existing stack should work
	success = inv.AddItem(item1, 1)
	if !success {
		t.Error("Adding to existing stack should succeed even when inventory is full")
	}
}

// TestRemoveItem tests removing items from inventory
func TestRemoveItem(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{ID: "test_item", Name: "Test Item"}
	inv.AddItem(item, 5)

	success := inv.RemoveItem("test_item", 3)
	if !success {
		t.Error("RemoveItem should succeed")
	}

	if inv.GetItemQuantity("test_item") != 2 {
		t.Errorf("Expected quantity 2, got %d", inv.GetItemQuantity("test_item"))
	}

	// Remove all remaining
	inv.RemoveItem("test_item", 2)
	if inv.HasItem("test_item") {
		t.Error("Item should be removed from inventory when quantity reaches 0")
	}
}

// TestRemoveItemNotEnough tests removing more items than available
func TestRemoveItemNotEnough(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{ID: "test_item", Name: "Test Item"}
	inv.AddItem(item, 2)

	success := inv.RemoveItem("test_item", 5)
	if success {
		t.Error("Removing more items than available should fail")
	}

	// Quantity should be unchanged
	if inv.GetItemQuantity("test_item") != 2 {
		t.Errorf("Quantity should be unchanged, got %d", inv.GetItemQuantity("test_item"))
	}
}

// TestUseItem tests using items
func TestUseItem(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{
		ID:       "food_item",
		Name:     "Food",
		Type:     ItemTypeFood,
		Cooldown: 0,
		Effects:  map[string]float64{"hunger": -0.3},
	}
	inv.AddItem(item, 3)

	usedItem, err := inv.UseItem("food_item")
	if err != nil {
		t.Errorf("UseItem failed: %v", err)
	}

	if usedItem == nil {
		t.Error("UseItem should return the item")
	}

	// Food is consumable - quantity should decrease
	if inv.GetItemQuantity("food_item") != 2 {
		t.Errorf("Expected quantity 2 after use, got %d", inv.GetItemQuantity("food_item"))
	}
}

// TestUseItemCooldown tests item cooldown
func TestUseItemCooldown(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{
		ID:       "toy_item",
		Name:     "Toy",
		Type:     ItemTypeToy,
		Cooldown: 60, // 60 second cooldown
		Uses:     -1, // Unlimited uses
	}
	inv.AddItem(item, 1)

	// First use should work
	_, err := inv.UseItem("toy_item")
	if err != nil {
		t.Errorf("First use should succeed: %v", err)
	}

	// Immediate second use should fail due to cooldown
	_, err = inv.UseItem("toy_item")
	if err == nil {
		t.Error("Second use should fail due to cooldown")
	}
}

// TestUseItemNotFound tests using non-existent item
func TestUseItemNotFound(t *testing.T) {
	inv := NewInventory(10)

	_, err := inv.UseItem("nonexistent")
	if err == nil {
		t.Error("Using non-existent item should fail")
	}
}

// TestHasItem tests item existence check
func TestHasItem(t *testing.T) {
	inv := NewInventory(10)
	item := &Item{ID: "test_item", Name: "Test Item"}
	inv.AddItem(item, 1)

	if !inv.HasItem("test_item") {
		t.Error("HasItem should return true for existing item")
	}

	if inv.HasItem("nonexistent") {
		t.Error("HasItem should return false for non-existent item")
	}
}

// TestGetItemsByType tests filtering items by type
func TestGetItemsByType(t *testing.T) {
	inv := NewInventory(10)
	food1 := &Item{ID: "food1", Name: "Food 1", Type: ItemTypeFood}
	food2 := &Item{ID: "food2", Name: "Food 2", Type: ItemTypeFood}
	toy := &Item{ID: "toy1", Name: "Toy 1", Type: ItemTypeToy}

	inv.AddItem(food1, 1)
	inv.AddItem(food2, 1)
	inv.AddItem(toy, 1)

	foods := inv.GetItemsByType(ItemTypeFood)
	if len(foods) != 2 {
		t.Errorf("Expected 2 food items, got %d", len(foods))
	}

	toys := inv.GetItemsByType(ItemTypeToy)
	if len(toys) != 1 {
		t.Errorf("Expected 1 toy item, got %d", len(toys))
	}
}

// TestCurrency tests currency operations
func TestCurrency(t *testing.T) {
	inv := NewInventory(10)

	if !inv.CanAfford(50) {
		t.Error("Should be able to afford 50 with 100 starting currency")
	}

	if inv.CanAfford(200) {
		t.Error("Should not be able to afford 200 with 100 starting currency")
	}

	success := inv.SpendCurrency(30)
	if !success {
		t.Error("SpendCurrency should succeed")
	}
	if inv.Currency != 70 {
		t.Errorf("Expected 70 currency, got %d", inv.Currency)
	}

	success = inv.SpendCurrency(100)
	if success {
		t.Error("SpendCurrency should fail when insufficient funds")
	}

	inv.AddCurrency(50)
	if inv.Currency != 120 {
		t.Errorf("Expected 120 currency after adding 50, got %d", inv.Currency)
	}
}

// TestCatalogGetItem tests retrieving items from catalog
func TestCatalogGetItem(t *testing.T) {
	catalog := testCatalog()

	item, exists := catalog.GetItem("basic_food")
	if !exists {
		t.Error("basic_food should exist in catalog")
	}
	if item.Name != "Basic Pet Food" {
		t.Errorf("Expected 'Basic Pet Food', got '%s'", item.Name)
	}

	_, exists = catalog.GetItem("nonexistent")
	if exists {
		t.Error("Nonexistent item should not exist")
	}
}

// TestShop tests shop functionality
func TestShop(t *testing.T) {
	catalog := testCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	if shop == nil {
		t.Fatal("NewShop returned nil")
	}
}

// TestShopGetPrice tests price retrieval
func TestShopGetPrice(t *testing.T) {
	catalog := testCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	price, err := shop.GetPrice("basic_food")
	if err != nil {
		t.Errorf("GetPrice failed: %v", err)
	}
	if price != 10 {
		t.Errorf("Expected price 10, got %d", price)
	}

	_, err = shop.GetPrice("nonexistent")
	if err == nil {
		t.Error("GetPrice should fail for non-existent item")
	}
}

// TestShopBuy tests purchasing items
func TestShopBuy(t *testing.T) {
	catalog := testCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	// Buy basic_food (10 currency each)
	err := shop.Buy("basic_food", 2)
	if err != nil {
		t.Errorf("Buy failed: %v", err)
	}

	if inv.GetItemQuantity("basic_food") != 2 {
		t.Errorf("Expected 2 basic_food in inventory, got %d", inv.GetItemQuantity("basic_food"))
	}

	if inv.Currency != 80 { // 100 - 20
		t.Errorf("Expected 80 currency remaining, got %d", inv.Currency)
	}
}

// TestShopBuyNotEnoughMoney tests buying without sufficient funds
func TestShopBuyNotEnoughMoney(t *testing.T) {
	catalog := testCatalog()
	inv := NewInventory(20)
	inv.Currency = 5 // Only 5 currency
	shop := NewShop(catalog, inv)

	err := shop.Buy("basic_food", 1) // Costs 10
	if err == nil {
		t.Error("Buy should fail with insufficient funds")
	}
}

// TestShopBuyLockedItem tests buying locked items
func TestShopBuyLockedItem(t *testing.T) {
	catalog := testCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	// premium_food is locked by default
	err := shop.Buy("premium_food", 1)
	if err == nil {
		t.Error("Buy should fail for locked items")
	}
}

// TestShopDiscount tests discount system
func TestShopDiscount(t *testing.T) {
	catalog := testCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	// Set 50% discount
	shop.SetDiscount("basic_food", 0.5)

	price, _ := shop.GetPrice("basic_food")
	if price != 5 { // 50% of 10
		t.Errorf("Expected discounted price 5, got %d", price)
	}
}

// TestShopDiscountBounds tests discount clamping
func TestShopDiscountBounds(t *testing.T) {
	catalog := testCatalog()
	inv := NewInventory(20)
	shop := NewShop(catalog, inv)

	// Negative discount should be clamped to 0
	shop.SetDiscount("basic_food", -0.5)
	if shop.Discounts["basic_food"] != 0 {
		t.Error("Negative discount should be clamped to 0")
	}

	// Over 100% discount should be clamped to 1
	shop.SetDiscount("basic_food", 1.5)
	if shop.Discounts["basic_food"] != 1 {
		t.Error("Discount over 1 should be clamped to 1")
	}
}

// TestItemTypeString tests item type string conversion
func TestItemTypeString(t *testing.T) {
	tests := []struct {
		itemType ItemType
		expected string
	}{
		{ItemTypeFood, "Food"},
		{ItemTypeToy, "Toy"},
		{ItemTypeMedicine, "Medicine"},
		{ItemTypeAccessory, "Accessory"},
		{ItemTypeGrooming, "Grooming"},
	}

	for _, tt := range tests {
		result := tt.itemType.String()
		if result != tt.expected {
			t.Errorf("ItemType.String() = %s, want %s", result, tt.expected)
		}
	}
}

// TestItemRarityString tests item rarity string conversion
func TestItemRarityString(t *testing.T) {
	tests := []struct {
		rarity   ItemRarity
		expected string
	}{
		{RarityCommon, "Common"},
		{RarityUncommon, "Uncommon"},
		{RarityRare, "Rare"},
		{RarityEpic, "Epic"},
		{RarityLegendary, "Legendary"},
	}

	for _, tt := range tests {
		result := tt.rarity.String()
		if result != tt.expected {
			t.Errorf("ItemRarity.String() = %s, want %s", result, tt.expected)
		}
	}
}

// testCatalog returns a catalog with an unlocked and a locked food
func testCatalog() *ItemCatalog {
	return &ItemCatalog{Items: map[string]*Item{
		"basic_food":   {ID: "basic_food", Name: "Basic Pet Food", Type: ItemTypeFood, Rarity: RarityCommon, Price: 10, Unlocked: true},
		"premium_food": {ID: "premium_food", Name: "Premium Pet Food", Type: ItemTypeFood, Rarity: RarityUncommon, Price: 25},
	}}
}
