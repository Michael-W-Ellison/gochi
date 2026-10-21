package core

import (
	"errors"
	"slices"

	"github.com/Michael-W-Ellison/gochi/internal/interaction"
)

// Cosmetics change only how a pet looks: collars, tags, bandanas, hats,
// glasses and coats, shown on the pet in the game, the pet list, AR and
// at playdates. They never change fur: dyes can harm a pet's skin. Each
// pet has six slots and wears at most one item in each; owned cosmetics
// are kept when taken off.

// Cosmetic slots, from the neck up and then the body
const (
	SlotCollar = "collar"
	SlotTag    = "tag"
	SlotNeck   = "neck"
	SlotHead   = "head"
	SlotEyes   = "eyes"
	SlotBody   = "body"
)

// CosmeticSlots lists the slots in display order
var CosmeticSlots = []string{SlotHead, SlotEyes, SlotNeck, SlotCollar, SlotTag, SlotBody}

// Cosmetic is something a pet can wear
type Cosmetic struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Slot    string `json:"slot"`
	Price   int    `json:"price"`
	Color   string `json:"color,omitempty"`   // How the page tints it, for colored items
	Lesson  string `json:"lesson,omitempty"`  // Shown when bought
	Season  string `json:"season,omitempty"`  // Sold only in this real season
	Holiday string `json:"holiday,omitempty"` // Sold only around this holiday

	rarity interaction.ItemRarity
}

// cosmetic makes an everyday cosmetic
func cosmetic(id, name, icon, slot string, price int, color string) Cosmetic {
	return Cosmetic{ID: id, Name: name, Icon: icon, Slot: slot, Price: price, Color: color}
}

// rare sets a cosmetic's rarity
func (c Cosmetic) rare(r interaction.ItemRarity) Cosmetic {
	c.rarity = r
	return c
}

// cosmeticCatalog lists every cosmetic; holiday ones are added from
// holidayGoods
var cosmeticCatalog = []Cosmetic{
	// Collars
	cosmetic("collar_red", "Red collar", "⭕", SlotCollar, 25, "#e53935"),
	cosmetic("collar_blue", "Blue collar", "⭕", SlotCollar, 25, "#1e88e5"),
	cosmetic("collar_green", "Green collar", "⭕", SlotCollar, 25, "#43a047"),
	cosmetic("collar_yellow", "Yellow collar", "⭕", SlotCollar, 25, "#fdd835"),
	cosmetic("collar_purple", "Purple collar", "⭕", SlotCollar, 25, "#8e24aa"),
	cosmetic("collar_pink", "Pink collar", "⭕", SlotCollar, 25, "#ec407a"),
	cosmetic("collar_orange", "Orange collar", "⭕", SlotCollar, 25, "#fb8c00"),
	cosmetic("collar_black", "Black collar", "⭕", SlotCollar, 25, "#212121"),
	cosmetic("collar_studded", "Studded collar", "✴️", SlotCollar, 60, "#5d4037"),
	{ID: "collar_bell", Name: "Collar with bell", Icon: "🔔", Slot: SlotCollar, Price: 45, Color: "#c62828",
		Lesson: "A cat's collar needs a breakaway buckle, so it can't get caught on something."},
	cosmetic("collar_leather", "Leather collar", "🟫", SlotCollar, 70, "#795548"),
	cosmetic("collar_glow", "Glow-in-the-dark collar", "✨", SlotCollar, 90, "#76ff03").rare(interaction.RarityRare),

	// Tags
	cosmetic("tag_bone", "Bone tag", "🦴", SlotTag, 20, ""),
	cosmetic("tag_fish", "Fish tag", "🐟", SlotTag, 20, ""),
	cosmetic("tag_heart", "Heart tag", "💖", SlotTag, 20, ""),
	cosmetic("tag_star", "Star tag", "⭐", SlotTag, 20, ""),
	cosmetic("tag_paw", "Paw tag", "🐾", SlotTag, 20, ""),
	cosmetic("tag_moon", "Moon tag", "🌙", SlotTag, 20, ""),
	cosmetic("tag_gold", "Gold tag", "🏅", SlotTag, 120, "").rare(interaction.RarityEpic),

	// Neck
	cosmetic("bandana_red", "Red paisley bandana", "🔻", SlotNeck, 35, "#d32f2f"),
	cosmetic("bandana_blue", "Blue paisley bandana", "🔻", SlotNeck, 35, "#1976d2"),
	cosmetic("bandana_dots", "Polka-dot bandana", "🔻", SlotNeck, 35, "#f06292"),
	cosmetic("bandana_plaid", "Plaid bandana", "🔻", SlotNeck, 35, "#6d4c41"),
	cosmetic("bandana_stars", "Star bandana", "🔻", SlotNeck, 35, "#3949ab"),
	cosmetic("bandana_flowers", "Flower bandana", "🔻", SlotNeck, 35, "#7cb342"),
	cosmetic("bow_tie", "Bow tie", "🎀", SlotNeck, 50, "#263238"),
	cosmetic("knitted_scarf", "Knitted scarf", "🧣", SlotNeck, 60, ""),
	cosmetic("ruffle_collar", "Ruffle collar", "🪭", SlotNeck, 75, "").rare(interaction.RarityRare),

	// Head
	cosmetic("bow_red", "Red bow", "🎀", SlotHead, 30, "#e53935"),
	cosmetic("bow_pink", "Pink bow", "🎀", SlotHead, 30, "#f48fb1"),
	cosmetic("bow_blue", "Blue bow", "🎀", SlotHead, 30, "#42a5f5"),
	cosmetic("bow_yellow", "Yellow bow", "🎀", SlotHead, 30, "#ffd54f"),
	cosmetic("flower_crown", "Flower crown", "💐", SlotHead, 80, ""),
	cosmetic("beret", "Beret", "🧢", SlotHead, 60, "#c62828"),
	cosmetic("sun_hat", "Sun hat", "👒", SlotHead, 60, ""),
	cosmetic("beanie", "Beanie", "🧶", SlotHead, 60, "#5c6bc0"),
	cosmetic("tiny_crown", "Tiny crown", "👑", SlotHead, 250, "").rare(interaction.RarityLegendary),

	// Eyes
	cosmetic("round_glasses", "Round glasses", "👓", SlotEyes, 55, ""),
	cosmetic("sunglasses", "Sunglasses", "🕶️", SlotEyes, 65, ""),
	cosmetic("star_glasses", "Star glasses", "🤩", SlotEyes, 90, "").rare(interaction.RarityRare),

	// Body
	cosmetic("raincoat", "Raincoat", "🧥", SlotBody, 90, "#fbc02d"),
	cosmetic("hoodie_grey", "Grey hoodie", "👕", SlotBody, 70, "#9e9e9e"),
	cosmetic("hoodie_red", "Red hoodie", "👕", SlotBody, 70, "#e53935"),
	cosmetic("hoodie_blue", "Blue hoodie", "👕", SlotBody, 70, "#1e88e5"),
	cosmetic("vest", "Vest", "🦺", SlotBody, 65, ""),
	cosmetic("superhero_cape", "Superhero cape", "🦸", SlotBody, 150, "#d32f2f").rare(interaction.RarityEpic),

	// Seasonal
	{ID: "daisy_crown", Name: "Daisy crown", Icon: "🌼", Slot: SlotHead, Price: 70, Season: "Spring"},
	{ID: "beach_hat", Name: "Beach hat and shades", Icon: "👒", Slot: SlotHead, Price: 80, Season: "Summer"},
	{ID: "plaid_scarf", Name: "Plaid scarf", Icon: "🧣", Slot: SlotNeck, Price: 60, Season: "Fall", Color: "#8d6e63"},
	{ID: "snowflake_bandana", Name: "Snowflake bandana", Icon: "❄️", Slot: SlotNeck, Price: 50, Season: "Winter", Color: "#90caf9"},
}

// FindCosmetic returns a cosmetic by ID
func FindCosmetic(id string) (Cosmetic, bool) {
	for _, c := range cosmeticCatalog {
		if c.ID == id {
			return c, true
		}
	}
	return Cosmetic{}, false
}

// ValidOutfit reports whether an outfit holds only known cosmetics, each
// in its own slot (the hub checks playdate cards with it)
func ValidOutfit(outfit map[string]string) bool {
	for slot, id := range outfit {
		c, ok := FindCosmetic(id)
		if !ok || c.Slot != slot {
			return false
		}
	}
	return true
}

// Cosmetic errors
var (
	ErrUnknownCosmetic = errors.New("unknown cosmetic")
	ErrNotOwned        = errors.New("not owned")
	ErrHolidayOff      = errors.New("that holiday is turned off")
)

// CosmeticOffer is a cosmetic as the shop and wardrobe show it
type CosmeticOffer struct {
	Cosmetic
	Rarity   string `json:"rarity"`
	Locked   bool   `json:"locked"`
	UnlockAt int    `json:"unlock_at"`
	Owned    bool   `json:"owned"`
	Worn     bool   `json:"worn"`
	OnSale   bool   `json:"on_sale"` // Can be bought now (in season, or around its holiday)
}

// CosmeticOffers lists the cosmetics the pet owns or can buy now
func (p *DigitalPet) CosmeticOffers() []CosmeticOffer {
	p.mu.Lock()
	defer p.mu.Unlock()
	offers := []CosmeticOffer{}
	for _, c := range cosmeticCatalog {
		owned := slices.Contains(p.Cosmetics, c.ID)
		onSale := (c.Season == "" || c.Season == RealSeason().String()) && (c.Holiday == "" || HolidayOpen(c.Holiday))
		if c.Holiday != "" && !HolidayOn(c.Holiday) || !owned && !onSale {
			continue
		}
		offers = append(offers, CosmeticOffer{Cosmetic: c, Rarity: c.rarity.String(),
			Locked: p.PointsEarned < rarityUnlocks[c.rarity], UnlockAt: rarityUnlocks[c.rarity],
			Owned: owned, Worn: p.Outfit[c.Slot] == c.ID, OnSale: onSale})
	}
	return offers
}

// BuyCosmetic spends Friendship Points on a cosmetic
func (p *DigitalPet) BuyCosmetic(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	c, ok := FindCosmetic(id)
	switch {
	case !ok:
		return ErrUnknownCosmetic
	case !p.Biology.IsAlive:
		return ErrPetNotAvailable
	case slices.Contains(p.Cosmetics, id):
		return ErrAlreadyOwned
	}
	if err := p.canBuyLocked(nil, c.rarity, c.Season, c.Holiday); err != nil {
		return err
	}
	if p.FriendshipPoints < c.Price {
		return ErrNotEnoughPoints
	}
	p.FriendshipPoints -= c.Price
	p.Cosmetics = append(p.Cosmetics, id)
	return nil
}

// Wear puts on a cosmetic the pet owns, or with id "" takes off what it
// wears in slot
func (p *DigitalPet) Wear(slot, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !slices.Contains(CosmeticSlots, slot) {
		return ErrUnknownCosmetic
	}
	if id == "" {
		delete(p.Outfit, slot)
		return nil
	}
	c, ok := FindCosmetic(id)
	switch {
	case !ok || c.Slot != slot:
		return ErrUnknownCosmetic
	case !slices.Contains(p.Cosmetics, id):
		return ErrNotOwned
	case c.Holiday != "" && !HolidayOn(c.Holiday):
		return ErrHolidayOff
	}
	if p.Outfit == nil {
		p.Outfit = make(map[string]string)
	}
	p.Outfit[slot] = id
	return nil
}

// Worn returns what the pet wears, by slot. Cosmetics of a holiday that is
// turned off aren't shown.
func (p *DigitalPet) Worn() map[string]Cosmetic {
	p.mu.Lock()
	defer p.mu.Unlock()
	worn := map[string]Cosmetic{}
	for slot, id := range p.Outfit {
		if c, ok := FindCosmetic(id); ok && c.Slot == slot && (c.Holiday == "" || HolidayOn(c.Holiday)) {
			worn[slot] = c
		}
	}
	return worn
}

// OutfitIDs returns what the pet wears as IDs, by slot (for playdate cards)
func (p *DigitalPet) OutfitIDs() map[string]string {
	ids := map[string]string{}
	for slot, c := range p.Worn() {
		ids[slot] = c.ID
	}
	return ids
}
