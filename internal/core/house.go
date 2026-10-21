package core

// House upgrades are bought with Home Stars, which belong to the player
// rather than a pet, and apply to every pet. House cosmetics change how
// the home looks behind every pet. The game server keeps the Home Stars
// and what has been bought (household.json) and tells this package which
// upgrades are owned (SetHouse).

// HouseUpgrade is an upgrade to the home
type HouseUpgrade struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Price       int    `json:"price"` // Home Stars
	Description string `json:"description"`
	Lesson      string `json:"lesson"`

	effects []itemEffect
}

// AirFilterStars is what the air filtration's filter costs, in Home Stars,
// every AirFilterDays game days
const (
	AirFilterStars = 20
	AirFilterDays  = 300
)

// houseUpgrades lists every house upgrade
var houseUpgradeCatalog = []HouseUpgrade{
	{ID: "air_filtration", Name: "Whole-house Air Filtration", Icon: "🌬️", Price: 200,
		Description: "Airway damage from foul air halved, and airways heal 25% faster, for every pet. Needs a filter every 300 game days (20 Home Stars).",
		Lesson:      "Ammonia and dust hurt airways.",
		effects: []itemEffect{effWhen(effAirDamage, 0.5, effectCondition{upkeep: airFilterUpkeep}),
			effWhen(effAirRecovery, 1.25, effectCondition{upkeep: airFilterUpkeep})}},
	{ID: "climate_control", Name: "Climate Control", Icon: "🌡️", Price: 300,
		Description: "Weather stress indoors 30% lower for every pet.",
		Lesson:      "Keep the home comfortable; pets can't open a window.",
		effects:     []itemEffect{eff(effWeatherStress, 0.7)}},
	{ID: "laundry_room", Name: "Laundry Room", Icon: "🧺", Price: 150,
		Description: "Coats get dirty 10% slower for every pet.",
		Lesson:      "Washing bedding weekly keeps parasites and smells down.",
		effects:     []itemEffect{eff(effCleanliness, 0.9)}},
	{ID: "playroom", Name: "Playroom", Icon: "🎨", Price: 250,
		Description: "Play and environment changes do 10% more for every pet.",
		Lesson:      "A space to play in daily keeps pets fit and calm.",
		effects:     []itemEffect{eff(effPlay, 1.1), eff(effEnrichment, 1.1)}},
	{ID: "sunroom", Name: "Sunroom", Icon: "🌞", Price: 200,
		Description: "In winter, daylight lifts every pet's mood.",
		Lesson:      "Daylight helps mood in animals as in people.",
		effects:     []itemEffect{effWhen(effDaylightMood, 0.1, inWinter)}},
	{ID: "feeding_station", Name: "Multi-pet Feeding Station", Icon: "🍽️", Price: 150,
		Description: "Pets that meet argue less: clashes are 30% less stressful.",
		Lesson:      "Pets that share a home should each have their own bowl and space.",
		effects:     []itemEffect{eff(effConflictStress, 0.7)}},
	{ID: "pet_door", Name: "Pet Door to a Fenced Yard", Icon: "🚪", Price: 250,
		Description: "Dogs: indoor accidents 30% less likely. Every pet runs around on its own more.",
		Lesson:      "A secure yard gives freedom without the dangers of the road.",
		effects: []itemEffect{effWhen(effAccidents, 0.7, effectCondition{species: SpeciesDog}),
			eff(effSelfExercise, 0.1)}},
	{ID: "catio", Name: "Catio", Icon: "🏡", Price: 250,
		Description: "Cats: the window perch does 25% more, and a little stimulation every day.",
		Lesson:      "Outdoor air and sights, without traffic or predators.",
		effects: []itemEffect{effWhen(effPerch, 1.25, effectCondition{species: SpeciesCat}),
			effWhen(effIdleStimulation, 0.05, effectCondition{species: SpeciesCat})}},
	{ID: "vet_corner", Name: "Home Vet Corner", Icon: "🩺", Price: 250,
		Description: "Warning signs of illness show sooner, so there's longer to act.",
		Lesson:      "Checking your pet regularly catches illness early.",
		effects:     []itemEffect{eff(effEarlyWarnings, 1)}},
	{ID: "quiet_room", Name: "Quiet Room", Icon: "🕯️", Price: 400,
		Description: "Seniors: stress stays lower, and \"Very old and frail\" shows later (lifespan doesn't change).",
		Lesson:      "Old pets value routine and calm.",
		effects: []itemEffect{effWhen(effStressCap, 0.6, seniors),
			effWhen(effFrailLater, 300, seniors)}},
}

// HouseUpgrades returns every house upgrade
func HouseUpgrades() []HouseUpgrade {
	return append([]HouseUpgrade(nil), houseUpgradeCatalog...)
}

// findUpgrade returns a house upgrade by ID
func findUpgrade(id string) (HouseUpgrade, bool) {
	for _, u := range houseUpgradeCatalog {
		if u.ID == id {
			return u, true
		}
	}
	return HouseUpgrade{}, false
}

// FindUpgrade returns a house upgrade by ID
func FindUpgrade(id string) (HouseUpgrade, bool) { return findUpgrade(id) }

// HouseLook is a house cosmetic: how the home looks behind every pet
type HouseLook struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Slot  string `json:"slot"`  // wallpaper, floor, window or nameplate
	Price int    `json:"price"` // Home Stars
	Style string `json:"style"` // How the page draws it: a color, pattern or scene
}

// House look slots
const (
	LookWallpaper = "wallpaper"
	LookFloor     = "floor"
	LookWindow    = "window"
	LookNameplate = "nameplate"
)

// houseLooks lists every house cosmetic
var houseLooks = []HouseLook{
	{ID: "wall_cream", Name: "Cream wallpaper", Icon: "🟨", Slot: LookWallpaper, Price: 40, Style: "#f6efd9"},
	{ID: "wall_mint", Name: "Mint stripes", Icon: "🟩", Slot: LookWallpaper, Price: 40, Style: "stripes:#dff3e6:#cbead6"},
	{ID: "wall_sky", Name: "Sky blue", Icon: "🟦", Slot: LookWallpaper, Price: 40, Style: "#dceefb"},
	{ID: "wall_rose", Name: "Rose dots", Icon: "🟥", Slot: LookWallpaper, Price: 40, Style: "dots:#fbe3ea:#f3c2cf"},
	{ID: "wall_lavender", Name: "Lavender", Icon: "🟪", Slot: LookWallpaper, Price: 40, Style: "#ece4f7"},
	{ID: "wall_stars", Name: "Night stars", Icon: "⭐", Slot: LookWallpaper, Price: 40, Style: "dots:#2d3561:#f5d76e"},
	{ID: "floor_oak", Name: "Oak boards", Icon: "🪵", Slot: LookFloor, Price: 50, Style: "boards:#c89b6d:#b88a5c"},
	{ID: "floor_tiles", Name: "Checker tiles", Icon: "🏁", Slot: LookFloor, Price: 50, Style: "checks:#eeeeee:#cfcfcf"},
	{ID: "floor_rug", Name: "Woven rug", Icon: "🧶", Slot: LookFloor, Price: 50, Style: "stripes:#d9a679:#c4865a"},
	{ID: "floor_grass", Name: "Indoor grass", Icon: "🌿", Slot: LookFloor, Price: 50, Style: "#9ccc65"},
	{ID: "view_city", Name: "City view", Icon: "🏙️", Slot: LookWindow, Price: 60, Style: "🏙️"},
	{ID: "view_garden", Name: "Garden view", Icon: "🌷", Slot: LookWindow, Price: 60, Style: "🌷"},
	{ID: "view_beach", Name: "Beach view", Icon: "🏖️", Slot: LookWindow, Price: 60, Style: "🏖️"},
	{ID: "view_mountains", Name: "Mountain view", Icon: "🏔️", Slot: LookWindow, Price: 60, Style: "🏔️"},
	{ID: "plate_wood", Name: "Wooden nameplate", Icon: "🪧", Slot: LookNameplate, Price: 30, Style: "#8d6e63"},
	{ID: "plate_brass", Name: "Brass nameplate", Icon: "🥇", Slot: LookNameplate, Price: 30, Style: "#c9a227"},
	{ID: "plate_slate", Name: "Slate nameplate", Icon: "⬛", Slot: LookNameplate, Price: 30, Style: "#455a64"},
	{ID: "plate_flowers", Name: "Painted flowers", Icon: "🌼", Slot: LookNameplate, Price: 30, Style: "#f48fb1"},
}

// HouseLooks returns every house cosmetic
func HouseLooks() []HouseLook {
	return append([]HouseLook(nil), houseLooks...)
}

// FindHouseLook returns a house cosmetic by ID
func FindHouseLook(id string) (HouseLook, bool) {
	for _, l := range houseLooks {
		if l.ID == id {
			return l, true
		}
	}
	return HouseLook{}, false
}
