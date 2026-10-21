package core

import "github.com/Michael-W-Ellison/gochi/internal/interaction"

// The items for each holiday: decorations (kept, no effect), cosmetics,
// and treats or toys for the pantry. They are sold only around their
// holiday; what has been bought stays usable afterwards.

// Kinds of holiday good
const (
	goodDecor = "decor"
	goodTreat = "treat"
	goodToy   = "toy"
	goodCare  = "care"
)

// holidayGood is one item for a holiday
type holidayGood struct {
	holiday string
	id      string
	name    string
	icon    string
	kind    string // goodDecor, goodTreat, goodToy, goodCare, or a cosmetic slot
	price   int
	color   string
	species Species // "" for both
}

// holidayGoods lists every holiday's items
var holidayGoods = []holidayGood{
	{"new_year", "party_hat", "Party hat", "🥳", SlotHead, 60, "", ""},
	{"new_year", "confetti_garland", "Confetti garland", "🎊", goodDecor, 90, "", ""},
	{"new_year", "calming_chew", "Calming chew", "🍬", goodCare, 40, "", ""},
	{"lunar_new_year", "red_lantern", "Red lantern", "🏮", goodDecor, 110, "", ""},
	{"lunar_new_year", "lion_dance_costume", "Lion-dance costume", "🦁", SlotBody, 90, "#e53935", ""},
	{"lunar_new_year", "dumpling_biscuit", "Dumpling-shaped biscuits", "🥟", goodTreat, 40, "", ""},
	{"valentines", "heart_bandana", "Heart bandana", "💗", SlotNeck, 50, "#ec407a", ""},
	{"valentines", "heart_cushion", "Heart cushion", "💝", goodDecor, 100, "", ""},
	{"valentines", "heart_biscuits", "Heart biscuits", "🍪", goodTreat, 40, "", ""},
	{"nowruz", "blossom_garland", "Spring blossom garland", "🌸", goodDecor, 100, "", ""},
	{"nowruz", "nowruz_flower_crown", "Spring flower crown", "🌺", SlotHead, 70, "", ""},
	{"nowruz", "herb_biscuit", "Herb biscuits", "🌿", goodTreat, 40, "", ""},
	{"holi", "rainbow_bandana", "Rainbow bandana", "🌈", SlotNeck, 50, "#ab47bc", ""},
	{"holi", "color_mural", "Color-splash mural", "🎨", goodDecor, 120, "", ""},
	{"holi", "sweet_potato_chew", "Sweet-potato chews", "🍠", goodTreat, 40, "", ""},
	{"easter", "bunny_ears", "Bunny ears", "🐰", SlotHead, 60, "", ""},
	{"easter", "spring_basket", "Spring basket", "🧺", goodDecor, 100, "", ""},
	{"easter", "carrot_biscuit", "Carrot biscuits", "🥕", goodTreat, 40, "", ""},
	{"eid_al_fitr", "crescent_lantern", "Crescent lantern", "🌙", goodDecor, 110, "", ""},
	{"eid_al_fitr", "festive_bow_tie", "Festive bow tie", "🎀", SlotNeck, 50, "#00897b", ""},
	{"eid_al_fitr", "fruit_treat", "Fruit treats (date-free)", "🍏", goodTreat, 40, "", ""},
	{"earth_day", "recycled_toy", "Recycled toy", "♻️", goodToy, 60, "", ""},
	{"earth_day", "leaf_crown", "Leaf crown", "🍃", SlotHead, 60, "", ""},
	{"earth_day", "garden_decoration", "Garden decoration", "🪴", goodDecor, 90, "", ""},
	{"vesak", "lotus_lanterns", "Paper lotus lanterns", "🪷", goodDecor, 110, "", ""},
	{"vesak", "lotus_charm", "Lotus collar charm", "🪷", SlotTag, 40, "", ""},
	{"vesak", "rice_biscuit", "Rice biscuits", "🍘", goodTreat, 40, "", ""},
	{"eid_al_adha", "festival_lights", "Festival lights", "✨", goodDecor, 120, "", ""},
	{"eid_al_adha", "embroidered_vest", "Embroidered vest", "🦺", SlotBody, 70, "#3949ab", ""},
	{"eid_al_adha", "lamb_biscuit", "Lamb-flavored biscuits", "🍖", goodTreat, 40, "", ""},
	{"cat_day", "fish_bandana", "Fish-pattern bandana", "🐟", SlotNeck, 50, "#26a69a", ""},
	{"cat_day", "cat_art", "Cat silhouette art", "🐈‍⬛", goodDecor, 100, "", ""},
	{"cat_day", "catnip_pillow", "Catnip pillow", "🛋️", goodToy, 45, "", SpeciesCat},
	{"dog_day", "bone_bandana", "Bone-pattern bandana", "🦴", SlotNeck, 50, "#8d6e63", ""},
	{"dog_day", "paw_rug", "Paw-print rug", "🐾", goodDecor, 100, "", ""},
	{"dog_day", "pb_biscuit", "Peanut-butter biscuits (xylitol-free)", "🥜", goodTreat, 40, "", SpeciesDog},
	{"mid_autumn", "rabbit_lantern", "Rabbit lantern", "🐇", goodDecor, 110, "", ""},
	{"mid_autumn", "moon_rabbit_ears", "Moon-rabbit ears", "🐰", SlotHead, 60, "", ""},
	{"mid_autumn", "pumpkin_biscuit", "Pumpkin biscuits", "🎃", goodTreat, 40, "", ""},
	{"world_animal_day", "globe_tag", "Globe tag", "🌍", SlotTag, 40, "", ""},
	{"world_animal_day", "animal_mural", "Animal mural", "🦒", goodDecor, 150, "", ""},
	{"world_animal_day", "shelter_badge", "Shelter badge", "🏠", SlotTag, 60, "", ""},
	{"diwali", "diya_lights", "Electric diya lights", "🪔", goodDecor, 120, "", ""},
	{"diwali", "rangoli_mat", "Rangoli mat", "🔷", goodDecor, 100, "", ""},
	{"diwali", "festive_collar", "Festive collar", "🟠", SlotCollar, 60, "#ff9800", ""},
	{"halloween", "pumpkin_costume", "Pumpkin costume", "🎃", SlotBody, 90, "#fb8c00", ""},
	{"halloween", "bat_wings", "Bat wings", "🦇", SlotBody, 80, "#424242", ""},
	{"halloween", "pumpkin_lantern", "Pumpkin lantern", "🎃", goodDecor, 110, "", ""},
	{"day_of_the_dead", "marigold_garland", "Marigold garland", "🏵️", goodDecor, 100, "", ""},
	{"day_of_the_dead", "sugar_skull_bandana", "Sugar-skull bandana", "💀", SlotNeck, 50, "#7e57c2", ""},
	{"day_of_the_dead", "remembrance_candle", "Remembrance candle (electric)", "🕯️", goodDecor, 80, "", ""},
	{"hanukkah", "blue_white_lights", "Blue-and-white lights", "💙", goodDecor, 120, "", ""},
	{"hanukkah", "hanukkah_scarf", "Knitted blue scarf", "🧣", SlotNeck, 60, "#1e88e5", ""},
	{"hanukkah", "potato_biscuit", "Potato biscuits", "🥔", goodTreat, 40, "", ""},
	{"christmas", "santa_hat", "Santa hat", "🎅", SlotHead, 60, "", ""},
	{"christmas", "reindeer_antlers", "Reindeer antlers", "🦌", SlotHead, 70, "", ""},
	{"christmas", "twinkle_lights", "Twinkle lights", "🌟", goodDecor, 130, "", ""},
	{"kwanzaa", "harvest_basket", "Woven harvest basket", "🧺", goodDecor, 100, "", ""},
	{"kwanzaa", "kente_bandana", "Kente-pattern bandana", "🟨", SlotNeck, 50, "#f9a825", ""},
	{"kwanzaa", "kwanzaa_chew", "Sweet-potato chews", "🍠", goodTreat, 40, "", ""},
}

// holidaySupplies are the holiday treats, toys and care supplies
var holidaySupplies []supply

// HolidayOf returns the holiday an item, supply or cosmetic belongs to, or
// "" if it isn't a holiday item
func HolidayOf(id string) string {
	for _, g := range holidayGoods {
		if g.id == id {
			return g.holiday
		}
	}
	return ""
}

func init() {
	for _, g := range holidayGoods {
		lesson := ""
		if h, ok := FindHoliday(g.holiday); ok {
			lesson = h.Lesson
		}
		var species []Species
		if g.species != "" {
			species = []Species{g.species}
		}
		switch g.kind {
		case goodDecor:
			shopCatalog = append(shopCatalog, ShopItem{ID: g.id, Name: g.name, Icon: g.icon, Category: CategoryHoliday,
				Price: g.price, Holiday: g.holiday, Description: "A holiday decoration.", Lesson: lesson, species: species})
		case goodTreat, goodToy, goodCare:
			s := supply{icon: g.icon, kind: g.kind, pack: 3, species: species, holiday: g.holiday,
				item: interaction.Item{ID: g.id, Name: g.name, Price: g.price, Rarity: interaction.RarityCommon,
					Type: interaction.ItemTypeFood, Description: "A holiday treat that's safe for pets. Pack of 3."}}
			switch g.kind {
			case goodToy:
				s.pack, s.item.Type = 1, interaction.ItemTypeToy
				s.item.Uses, s.item.MaxUses = 30, 30
				s.item.Description = "A holiday toy. Wears out."
				s.toy = &toyProfile{fit: 0.15, exercise: 1.0, stimulation: 0.5, catnip: g.id == "catnip_pillow", lesson: lesson}
			case goodCare:
				s.item.Type = interaction.ItemTypeMedicine
				s.item.Description = "Helps a pet stay calm during fireworks. Pack of 3."
			}
			holidaySupplies = append(holidaySupplies, s)
		default:
			cosmeticCatalog = append(cosmeticCatalog, Cosmetic{ID: g.id, Name: g.name, Icon: g.icon, Slot: g.kind,
				Price: g.price, Color: g.color, Holiday: g.holiday, Lesson: lesson})
		}
	}
}
