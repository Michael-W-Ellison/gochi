package core

import (
	"errors"
	"math"
	"slices"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/internal/interaction"
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
	types.InteractionTraining:    true,

	types.InteractionEnvironmentalEnrichment: true,
	types.InteractionCleaning:                true,
	types.InteractionWalking:                 true,
	types.InteractionEnvironmentChange:       true,
	types.InteractionSocialIntroduction:      true,
}

// ItemCategory groups shop items for display
type ItemCategory string

const (
	CategoryFoodBowl   ItemCategory = "food_bowl"
	CategoryWaterBowl  ItemCategory = "water_bowl"
	CategoryToy        ItemCategory = "toy"
	CategoryBed        ItemCategory = "bed"
	CategoryBrush      ItemCategory = "brush"
	CategoryTraining   ItemCategory = "training"
	CategoryHome       ItemCategory = "home"       // Hygiene
	CategoryComfort    ItemCategory = "comfort"    // Comfort and weather
	CategoryHealth     ItemCategory = "health"     // Safety and health
	CategoryEnrichment ItemCategory = "enrichment" // Things to do
	CategoryOldAge     ItemCategory = "old_age"
	CategoryDecoration ItemCategory = "decoration"
	CategorySeasonal   ItemCategory = "seasonal"
	CategoryHoliday    ItemCategory = "holiday" // Holiday decorations
)

// ShopItem is something that can be bought with Friendship Points and is
// kept: equipment, decorations, and seasonal and holiday decorations
type ShopItem struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Icon        string       `json:"icon"`
	Category    ItemCategory `json:"category"`
	Price       int          `json:"price"`
	Description string       `json:"description"`
	Lesson      string       `json:"lesson,omitempty"`  // Shown when bought
	Suits       string       `json:"suits,omitempty"`   // The personality a decoration suits
	Season      string       `json:"season,omitempty"`  // Sold only in this real season
	Holiday     string       `json:"holiday,omitempty"` // Sold only around this holiday

	species    []Species // Who it is for; nil for both
	rarity     interaction.ItemRarity
	group      string // Items in a group replace each other: the dearest owned applies
	effects    []itemEffect
	upkeep     map[string]float64 // Upkeep it starts when bought: game days each lasts
	firstAid   int                // First-aid uses it comes with
	decoration bool               // A decoration outside the Decoration category (effects limited to 6%)
}

// isDecoration reports whether an item's effects count as a decoration's
func (item ShopItem) isDecoration() bool {
	return item.decoration || item.Category == CategoryDecoration || item.Category == CategoryHoliday
}

// clickerProgress is how much faster training with a clicker teaches
const clickerProgress = 1.4

// Upkeep timers (game days each lasts once renewed)
const (
	fountainFilterUpkeep = "fountain_filter"
	fleaCollarUpkeep     = "flea_collar"
	beddingUpkeep        = "bedding"
	vaccinationUpkeep    = "vaccination"
	airFilterUpkeep      = "air_filter" // The house's, kept with Home Stars

	fountainFilterDays = 300
	fleaCollarDays     = 500
	beddingDays        = 200
	vaccinationDays    = 1000
)

// Conditions used in the catalog
var (
	cold      = effectCondition{below: 10}
	hot       = effectCondition{above: 27}
	seniors   = effectCondition{senior: true}
	inSpring  = effectCondition{inSeason: true, season: environment.SeasonSpring}
	inSummer  = effectCondition{inSeason: true, season: environment.SeasonSummer}
	inAutumn  = effectCondition{inSeason: true, season: environment.SeasonFall}
	inWinter  = effectCondition{inSeason: true, season: environment.SeasonWinter}
	freshBeds = effectCondition{halfStale: beddingUpkeep}
)

// suits returns a condition on a personality trait
func suits(trait string) effectCondition { return effectCondition{trait: trait} }

var (
	cats = []Species{SpeciesCat}
	dogs = []Species{SpeciesDog}
)

// shopCatalog lists every item in display order
var shopCatalog = []ShopItem{
	// Food and water
	{ID: "food_bowl", Name: "Sturdy Food Bowl", Icon: "🥣", Category: CategoryFoodBowl, Price: 160, group: "food_bowl",
		Description: "Hunger builds 20% slower.", effects: []itemEffect{eff(effHunger, 0.8)},
		Lesson: "Measured meals at regular times keep a pet's weight and digestion steady."},
	{ID: "auto_feeder", Name: "Automatic Feeder", Icon: "🍱", Category: CategoryFoodBowl, Price: 400, group: "food_bowl",
		Description: "Hunger builds 40% slower.", effects: []itemEffect{eff(effHunger, 0.6)}, rarity: interaction.RarityUncommon,
		Lesson: "A feeder keeps mealtimes regular, but check that your pet is actually eating."},
	{ID: "water_bowl", Name: "Big Water Bowl", Icon: "🥛", Category: CategoryWaterBowl, Price: 160, group: "water_bowl",
		Description: "Thirst builds 20% slower.", effects: []itemEffect{eff(effThirst, 0.8)},
		Lesson: "Fresh water should always be available; change it every day."},
	{ID: "water_fountain", Name: "Water Fountain", Icon: "⛲", Category: CategoryWaterBowl, Price: 400, group: "water_bowl",
		Description: "Thirst builds 40% slower while its filter is fresh (every 300 game days), otherwise 20%.",
		rarity:      interaction.RarityUncommon,
		upkeep:      map[string]float64{fountainFilterUpkeep: fountainFilterDays},
		Lesson:      "Many cats drink more from moving water. A fountain's filter must be changed to keep the water clean.",
		effects: []itemEffect{effWhen(effThirst, 0.6, effectCondition{upkeep: fountainFilterUpkeep}),
			effWhen(effThirst, 0.8, effectCondition{lapsed: fountainFilterUpkeep})}},

	// Play, grooming and training
	{ID: "squeaky_toy", Name: "Squeaky Toy", Icon: "🧸", Category: CategoryToy, Price: 130, group: "toy",
		Description: "Playing is 25% more fun.", effects: []itemEffect{eff(effPlay, 1.25)},
		Lesson: "Check toys for loose parts that could be swallowed."},
	{ID: "puzzle_ball", Name: "Puzzle Ball", Icon: "⚽", Category: CategoryToy, Price: 320, group: "toy",
		Description: "Playing is 50% more fun, and more for a curious pet.", rarity: interaction.RarityUncommon,
		effects: []itemEffect{eff(effPlay, 1.5), effWhen(effPlay, 1.25, suits("curiosity"))},
		Lesson:  "Toys that make a pet think tire its mind as well as its body."},
	{ID: "soft_brush", Name: "Soft Brush", Icon: "🪮", Category: CategoryBrush, Price: 130,
		Description: "Grooming is 50% more effective.", effects: []itemEffect{eff(effGrooming, 1.5)},
		Lesson: "Regular brushing removes loose fur, prevents mats and lets you check the skin."},
	{ID: "clicker", Name: "Training Clicker", Icon: "🔘", Category: CategoryTraining, Price: 65,
		Description: "Marks the moment your pet gets it right: training with it is 40% faster.",
		Lesson:      "The click tells a pet exactly what it did right; follow every click with a reward."},

	// Beds (the dearest owned applies; fresh bedding keeps them working fully)
	{ID: "cozy_bed", Name: "Cozy Bed", Icon: "🛏️", Category: CategoryBed, Price: 240, group: "bed",
		Description: "Tiredness builds 25% slower; half as well once its bedding needs changing (every 200 game days).",
		upkeep:      map[string]float64{beddingUpkeep: beddingDays},
		effects:     []itemEffect{effWhen(effTiredness, 0.75, freshBeds)},
		Lesson:      "A pet sleeps better in a quiet, warm spot of its own."},
	{ID: "orthopedic_bed", Name: "Orthopedic Bed", Icon: "🦴", Category: CategoryBed, Price: 560, group: "bed",
		rarity:      interaction.RarityEpic,
		Description: "A bed that also helps seniors: fitness fades 25% slower and sleep debt clears 20% faster. Replaces the Cozy Bed.",
		upkeep:      map[string]float64{beddingUpkeep: beddingDays},
		effects: []itemEffect{effWhen(effTiredness, 0.75, freshBeds),
			effWhen(effFitnessLoss, 0.75, seniors), effWhen(effSleepDebt, 1.2, seniors)},
		Lesson: "Old joints need firmer support."},

	// Hygiene
	{ID: "litter_mat", Name: "Litter Mat", Icon: "🧽", Category: CategoryHome, Price: 95, species: cats,
		Description: "The living area gets dirty 15% slower.", effects: []itemEffect{eff(effMess, 0.85)},
		Lesson: "A mat catches litter from paws."},
	{ID: "second_litter_box", Name: "Second Litter Box", Icon: "🚪", Category: CategoryHome, Price: 290, species: cats,
		rarity:      interaction.RarityUncommon,
		Description: "Mess builds 25% slower, and a dirty area is half as stressful.",
		effects:     []itemEffect{eff(effMess, 0.75), eff(effDirtStress, 0.5)},
		Lesson:      "One box per cat, plus one."},
	{ID: "self_cleaning_box", Name: "Self-cleaning Litter Box", Icon: "🤖", Category: CategoryHome, Price: 720, species: cats,
		rarity:      interaction.RarityEpic,
		Description: "Mess builds 40% slower, on days the box has been scooped.",
		effects:     []itemEffect{effWhen(effMess, 0.6, effectCondition{scooped: true})},
		Lesson:      "Automatic boxes still need checking: changes in the box are an early sign of illness."},
	{ID: "potty_patch", Name: "Potty Patch", Icon: "🌱", Category: CategoryHome, Price: 190, species: dogs,
		rarity:      interaction.RarityUncommon,
		Description: "Indoor accidents are 25% less likely.", effects: []itemEffect{eff(effAccidents, 0.75)},
		Lesson: "Somewhere acceptable to go when a walk is late."},

	// Comfort and weather
	{ID: "heated_bed", Name: "Heated Bed", Icon: "🔥", Category: CategoryComfort, Price: 350, rarity: interaction.RarityUncommon,
		Description: "Below 10 °C: weather stress halved, and sleep restores 15% faster.",
		effects:     []itemEffect{effWhen(effWeatherStress, 0.5, cold), effWhen(effRest, 1.15, cold)},
		Lesson:      "Small, short-haired and old pets lose heat quickly."},
	{ID: "cooling_mat", Name: "Cooling Mat", Icon: "❄️", Category: CategoryComfort, Price: 350, rarity: interaction.RarityUncommon,
		Description: "Above 27 °C: weather stress halved, and thirst builds 15% slower.",
		effects:     []itemEffect{effWhen(effWeatherStress, 0.5, hot), effWhen(effThirst, 0.85, hot)},
		Lesson:      "Pets can't sweat like people; they need shade and water."},
	{ID: "hiding_cave", Name: "Hiding Cave", Icon: "🕳️", Category: CategoryComfort, Price: 190, species: cats,
		Description: "Stress eases 20% faster, 40% for a nervous cat.",
		effects:     []itemEffect{eff(effStressRecovery, 1.2), effWhen(effStressRecovery, 1.4/1.2, suits("neuroticism"))},
		Lesson:      "A safe den lets an anxious pet calm down."},
	{ID: "covered_crate", Name: "Covered Crate", Icon: "🏠", Category: CategoryComfort, Price: 190, species: dogs,
		Description: "Stress eases 20% faster, 40% for a nervous dog.",
		effects:     []itemEffect{eff(effStressRecovery, 1.2), effWhen(effStressRecovery, 1.4/1.2, suits("neuroticism"))},
		Lesson:      "A safe den lets an anxious pet calm down."},

	// Safety and health
	{ID: "first_aid_kit", Name: "First-aid Kit", Icon: "🩹", Category: CategoryHealth, Price: 240, firstAid: 3,
		rarity:      interaction.RarityUncommon,
		Description: "Treats a trip injury at home. 3 uses; refills are in the pantry shop.",
		Lesson:      "Clean and cover small wounds; see a vet for deep ones."},
	{ID: "travel_harness", Name: "Travel Harness", Icon: "🦮", Category: CategoryHealth, Price: 290, species: dogs,
		rarity:      interaction.RarityUncommon,
		Description: "Trip injury chance 4 points lower.", effects: []itemEffect{eff(effTripInjury, -4)},
		Lesson: "A harness spares the throat and is hard to slip."},
	{ID: "cat_carrier", Name: "Cat Carrier", Icon: "🐱", Category: CategoryHealth, Price: 290, species: cats,
		rarity:      interaction.RarityUncommon,
		Description: "Trip injury chance 4 points lower, and trips are less stressful.",
		effects:     []itemEffect{eff(effTripInjury, -4), eff(effTripStress, 0.5)},
		Lesson:      "A familiar carrier makes travel less frightening."},
	{ID: "flea_collar", Name: "Flea and Tick Collar", Icon: "🛡️", Category: CategoryHealth, Price: 240,
		rarity:      interaction.RarityRare,
		Description: "Parasites from trips build up 50% slower. Lasts 500 game days; replacements are in the pantry shop.",
		upkeep:      map[string]float64{fleaCollarUpkeep: fleaCollarDays},
		effects:     []itemEffect{effWhen(effParasites, 0.5, effectCondition{upkeep: fleaCollarUpkeep})},
		Lesson:      "Prevention is easier than treatment."},

	// Things to do
	{ID: "cat_tree", Name: "Cat Tree", Icon: "🌳", Category: CategoryEnrichment, Price: 510, species: cats,
		rarity:      interaction.RarityRare,
		Description: "The window perch and scratching post do 25% more, and the cat runs around more (more if curious).",
		effects: []itemEffect{eff(effPerch, 1.25), eff(effSelfExercise, 0.1),
			effWhen(effSelfExercise, 0.1, suits("curiosity"))},
		Lesson: "Cats feel safest up high."},
	{ID: "aquarium", Name: "Aquarium to Watch", Icon: "🐟", Category: CategoryEnrichment, Price: 640, species: cats,
		rarity:      interaction.RarityEpic,
		Description: "Mental stimulation slowly rises while the cat is idle.",
		effects:     []itemEffect{eff(effIdleStimulation, 0.3)},
		Lesson:      "Movement keeps hunting instincts busy; keep the lid closed."},
	{ID: "agility_set", Name: "Agility Set", Icon: "🏁", Category: CategoryEnrichment, Price: 510, species: dogs,
		rarity:      interaction.RarityRare,
		Description: "Training does 20% more and walks and play build fitness 20% faster (more for an energetic dog).",
		effects: []itemEffect{eff(effTraining, 1.2), eff(effFitnessGain, 1.2),
			effWhen(effTraining, 1.1, suits("energy")), effWhen(effFitnessGain, 1.1, suits("energy"))},
		Lesson: "Jumps and tunnels exercise body and mind."},
	{ID: "ball_launcher", Name: "Automatic Ball Launcher", Icon: "🎾", Category: CategoryEnrichment, Price: 640, species: dogs,
		rarity:      interaction.RarityEpic,
		Description: "Exercise slowly rises while the dog is idle and awake, unless it is tired.",
		effects:     []itemEffect{eff(effIdleExercise, 0.3)},
		Lesson:      "Play on its own helps, but doesn't replace play with you."},
	{ID: "snuffle_mat", Name: "Snuffle Mat", Icon: "🧠", Category: CategoryEnrichment, Price: 145, species: dogs,
		Description: "Each meal also gives a little mental stimulation.", effects: []itemEffect{eff(effFeedStimulation, 0.15)},
		Lesson: "A few minutes of sniffing tires a dog like a short walk."},

	// Old age
	{ID: "pet_steps", Name: "Pet Steps", Icon: "🪜", Category: CategoryOldAge, Price: 960, rarity: interaction.RarityLegendary,
		Description: "Seniors: tiredness from moving 20% lower, and they keep running around on their own longer.",
		effects:     []itemEffect{effWhen(effTiredness, 0.8, seniors), effWhen(effSelfExercise, 0.1, seniors)},
		Lesson:      "Steps spare old joints from jumping."},

	// Decorations: small effects on a pet whose personality suits them
	{ID: "balloon", Name: "Balloon", Icon: "🎈", Category: CategoryDecoration, Price: 50, Suits: "playful pets",
		Description: "Happiness falls 2% slower for a playful pet.", effects: []itemEffect{effWhen(effHappinessDrop, 0.98, suits("playfulness"))}},
	{ID: "plant", Name: "Potted Plant", Icon: "🪴", Category: CategoryDecoration, Price: 65, Suits: "open pets",
		Description: "Stress eases 2% faster for an open pet.", effects: []itemEffect{effWhen(effStressRecovery, 1.02, suits("openness"))},
		Lesson: "Choose pet-safe plants: lilies, sago palms and many others are poisonous."},
	{ID: "lamp", Name: "Night Lamp", Icon: "💡", Category: CategoryDecoration, Price: 80, Suits: "nervous pets",
		Description: "Night-time stress halved for a nervous pet.",
		effects:     []itemEffect{{kind: effNightStress, value: 0.5, when: suits("neuroticism"), uncapped: true}}},
	{ID: "painting", Name: "Painting", Icon: "🖼️", Category: CategoryDecoration, Price: 95, Suits: "open pets",
		Description: "Happiness falls 2% slower for an open pet.", effects: []itemEffect{effWhen(effHappinessDrop, 0.98, suits("openness"))}},
	{ID: "rainbow", Name: "Rainbow Mural", Icon: "🌈", Category: CategoryDecoration, Price: 190, Suits: "any pet",
		rarity:      interaction.RarityUncommon,
		Description: "Happiness falls 2% slower.", effects: []itemEffect{eff(effHappinessDrop, 0.98)}},
	{ID: "wind_chime", Name: "Wind Chime", Icon: "🎐", Category: CategoryDecoration, Price: 110, Suits: "curious pets",
		rarity:      interaction.RarityUncommon,
		Description: "A little mental stimulation while a curious pet is idle.",
		effects:     []itemEffect{effWhen(effIdleStimulation, 0.02, suits("curiosity"))}},
	{ID: "scent_blanket", Name: "Blanket in the Family's Scent", Icon: "🧶", Category: CategoryDecoration, Price: 145,
		Suits: "affectionate pets", rarity: interaction.RarityUncommon,
		Description: "Stress builds 3% slower when an affectionate pet is alone.",
		effects:     []itemEffect{effWhen(effAloneStress, 0.97, suits("affectionate"))}},
	{ID: "paper_lanterns", Name: "Paper Lanterns", Icon: "🏮", Category: CategoryDecoration, Price: 175, Suits: "outgoing pets",
		rarity:      interaction.RarityRare,
		Description: "Play and petting make an outgoing pet 2% happier.",
		effects:     []itemEffect{effWhen(effAffection, 1.02, suits("extraversion"))}},
	{ID: "curtains", Name: "Window Curtains", Icon: "🪟", Category: CategoryDecoration, Price: 130, Suits: "territorial pets",
		rarity:      interaction.RarityUncommon,
		Description: "Less stress from things moving outside: a territorial pet's stress eases 2% faster.",
		effects:     []itemEffect{effWhen(effStressRecovery, 1.02, suits("territoriality"))}},

	// Seasonal: sold only in their real season, and work only then
	{ID: "flower_box", Name: "Flower Box", Icon: "🌸", Category: CategorySeasonal, Price: 160, Season: "Spring",
		Description: "Happiness falls 3% slower in spring.",
		effects:     []itemEffect{effWhen(effHappinessDrop, 0.97, inSpring)}, decoration: true,
		Lesson: "Choose pet-safe plants: lilies are deadly to cats."},
	{ID: "paddling_pool", Name: "Paddling Pool", Icon: "🏊", Category: CategorySeasonal, Price: 190, Season: "Summer", species: dogs,
		Description: "In summer, above 27 °C: thirst builds 15% slower.",
		effects: []itemEffect{effWhen(effThirst, 0.85, effectCondition{above: 27, inSeason: true,
			season: environment.SeasonSummer})},
		Lesson: "Shallow, cool water lets a dog cool down; never leave it alone near water."},
	{ID: "ice_treat_mold", Name: "Ice Treat Mold", Icon: "🧊", Category: CategorySeasonal, Price: 190, Season: "Summer", species: cats,
		Description: "In summer, above 27 °C: thirst builds 15% slower.",
		effects: []itemEffect{effWhen(effThirst, 0.85, effectCondition{above: 27, inSeason: true,
			season: environment.SeasonSummer})},
		Lesson: "Frozen broth cubes help a cat drink more in hot weather."},
	{ID: "leaf_pile", Name: "Leaf Pile", Icon: "🍂", Category: CategorySeasonal, Price: 130, Season: "Fall",
		Description: "In autumn, play also gives mental stimulation.",
		effects:     []itemEffect{effWhen(effPlayStimulation, 0.1, inAutumn)},
		Lesson:      "Check leaf piles for sticks and mold before your pet dives in."},
	{ID: "knitted_sweater", Name: "Knitted Sweater", Icon: "🧥", Category: CategorySeasonal, Price: 190, Season: "Winter",
		Description: "In winter, below 10 °C: energy runs down 15% slower (25% for seniors).",
		effects: []itemEffect{
			effWhen(effEnergy, 0.85, effectCondition{below: 10, inSeason: true, season: environment.SeasonWinter}),
			effWhen(effEnergy, 0.75/0.85, effectCondition{below: 10, inSeason: true, season: environment.SeasonWinter, senior: true})},
		Lesson: "Not every pet needs one: thick-coated pets can overheat."},
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

	// Why the pet turned the interaction down, if it did (Applied is false)
	Refusal string

	// What this care teaches about looking after a real pet
	Lesson string

	// Anything else worth telling the player, such as a toy wearing out
	Note string

	// Set when the same care was repeated so often just now that it
	// barely counts towards the pet's learning and personality
	Repeated string

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
// pet: owned items with an effect that applies now. Items outclassed by a
// better one in the same group are not included.
func (p *DigitalPet) ActiveEffects() []ShopItem {
	p.mu.Lock()
	defer p.mu.Unlock()

	effects := []ShopItem{}
	for _, item := range p.effectiveItemsLocked() {
		for _, e := range item.effects {
			if ok, _ := p.appliesLocked(e.when); ok {
				effects = append(effects, item)
				break
			}
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

	t := p.playNowLocked()
	if last, ok := p.LastCareReward[interactionType]; ok && t.Sub(last) < CareRewardCooldown {
		return 0
	}

	if p.LastCareReward == nil {
		p.LastCareReward = make(map[types.InteractionType]time.Time)
	}
	p.LastCareReward[interactionType] = t
	p.FriendshipPoints += PointsPerCare
	p.PointsEarned += PointsPerCare
	return PointsPerCare
}

// Errors buying items
var (
	ErrItemLocked     = errors.New("item not unlocked yet")
	ErrOutOfSeason    = errors.New("item not sold in this season")
	ErrNotHolidayTime = errors.New("item sold only around its holiday")
)

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
	if err := p.canBuyLocked(item.species, item.rarity, item.Season, item.Holiday); err != nil {
		return err
	}
	if p.ownsLocked(itemID) {
		return ErrAlreadyOwned
	}
	if p.FriendshipPoints < item.Price {
		return ErrNotEnoughPoints
	}

	p.FriendshipPoints -= item.Price
	p.Inventory = append(p.Inventory, itemID)
	p.startUpkeepLocked(item)
	return nil
}

// canBuyLocked checks that something can be bought now: it is for the
// pet's species, the pet has earned enough points to unlock it, and it is
// in season or its holiday. Caller must hold p.mu.
func (p *DigitalPet) canBuyLocked(species []Species, rarity interaction.ItemRarity, season, holiday string) error {
	switch {
	case species != nil && !slices.Contains(species, p.speciesLocked()):
		return ErrWrongSpecies
	case p.PointsEarned < rarityUnlocks[rarity]:
		return ErrItemLocked
	case season != "" && season != RealSeason().String():
		return ErrOutOfSeason
	case holiday != "" && !HolidayOpen(holiday):
		return ErrNotHolidayTime
	}
	return nil
}

// startUpkeepLocked starts an item's upkeep timers and adds its first-aid
// uses. Caller must hold p.mu.
func (p *DigitalPet) startUpkeepLocked(item ShopItem) {
	for id, days := range item.upkeep {
		p.renewUpkeepLocked(id, days)
	}
	p.FirstAidUses += item.firstAid
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

// OwnedItems returns the catalog entries of the items the pet has.
// Decorations for a holiday that is turned off aren't shown.
func (p *DigitalPet) OwnedItems() []ShopItem {
	p.mu.Lock()
	defer p.mu.Unlock()

	var items []ShopItem
	for _, item := range shopCatalog {
		if p.ownsLocked(item.ID) && (item.Holiday == "" || HolidayOn(item.Holiday)) {
			items = append(items, item)
		}
	}
	return items
}

// ShopOffer is an item as the shop shows it
type ShopOffer struct {
	ShopItem
	Rarity   string `json:"rarity"`
	Locked   bool   `json:"locked"`    // The pet hasn't earned enough points yet
	UnlockAt int    `json:"unlock_at"` // Lifetime points that unlock it
	Owned    bool   `json:"owned"`
}

// ShopOffers lists the items for the pet's species that can be bought now
// or are owned: seasonal items only in their season, holiday items only
// around their holiday (and not if a parent turned it off)
func (p *DigitalPet) ShopOffers() []ShopOffer {
	p.mu.Lock()
	defer p.mu.Unlock()
	season := RealSeason().String()
	offers := []ShopOffer{}
	for _, item := range shopCatalog {
		if item.species != nil && !slices.Contains(item.species, p.speciesLocked()) {
			continue
		}
		owned := p.ownsLocked(item.ID)
		onSale := (item.Season == "" || item.Season == season) && (item.Holiday == "" || HolidayOpen(item.Holiday))
		if (item.Holiday != "" && !HolidayOn(item.Holiday)) || (!owned && !onSale) {
			continue
		}
		offers = append(offers, ShopOffer{ShopItem: item, Rarity: item.rarity.String(),
			Locked: p.PointsEarned < rarityUnlocks[item.rarity], UnlockAt: rarityUnlocks[item.rarity], Owned: owned})
	}
	return offers
}

// effectiveItemsLocked returns the owned items whose effects count: in
// each group, only the dearest owned. Caller must hold p.mu.
func (p *DigitalPet) effectiveItemsLocked() []ShopItem {
	best := map[string]ShopItem{}
	var items []ShopItem
	for _, item := range shopCatalog {
		if !p.ownsLocked(item.ID) || (item.species != nil && !slices.Contains(item.species, p.speciesLocked())) {
			continue
		}
		if item.group == "" {
			items = append(items, item)
		} else if b, ok := best[item.group]; !ok || item.Price > b.Price {
			best[item.group] = item
		}
	}
	for _, item := range shopCatalog { // In catalog order
		if b, ok := best[item.group]; ok && b.ID == item.ID {
			items = append(items, item)
		}
	}
	return items
}

// bestInGroupLocked returns the owned item that counts in a group.
// Caller must hold p.mu.
func (p *DigitalPet) bestInGroupLocked(group string) (ShopItem, bool) {
	for _, item := range p.effectiveItemsLocked() {
		if item.group == group {
			return item, true
		}
	}
	return ShopItem{}, false
}

// decayMultipliersLocked returns how the pet's belongings, home and
// surroundings change the rates its body works at. Caller must hold p.mu.
func (p *DigitalPet) decayMultipliersLocked() biology.DecayMultipliers {
	hunger, thirst, energy := p.environmentRatesLocked()
	e := p.effectsLocked()
	stressGain := 1.0
	if p.Biology.GetAgeInDays()-p.LastInteractionAge > lonelyAfter {
		stressGain = e[effAloneStress] // A blanket that smells of home helps
	}
	return biology.DecayMultipliers{
		StressGain:        stressGain,
		Nutrition:         e[effHunger] * hunger,
		Hydration:         e[effThirst] * thirst,
		Fatigue:           e[effTiredness],
		Energy:            e[effEnergy] * energy,
		Cleanliness:       e[effCleanliness],
		HappinessDrop:     e[effHappinessDrop],
		StressRecovery:    e[effStressRecovery],
		StressCap:         e[effStressCap],
		FitnessLoss:       e[effFitnessLoss],
		FitnessGain:       e[effFitnessGain],
		AirDamage:         e[effAirDamage],
		AirRecovery:       e[effAirRecovery],
		Rest:              e[effRest],
		SleepDebt:         e[effSleepDebt],
		DigestiveRecovery: e[effDigestiveRecovery],
		KidneyDecline:     e[effKidneyDecline],
		EyeIrritation:     e[effEyeIrritation],
		DentalDecline:     e[effDentalDecline],
		HeartAging:        e[effHeartAging],
	}
}

// careBoostLocked returns how much belongings strengthen an interaction,
// and the item that does most of it (for the page), if any. Caller must
// hold p.mu.
func (p *DigitalPet) careBoostLocked(interactionType types.InteractionType) (float64, *ShopItem) {
	e := p.effectsLocked()
	boost, group := 1.0, ""
	switch interactionType {
	case types.InteractionPlaying:
		boost, group = e[effPlay]*e[effAffection], "toy"
	case types.InteractionPetting:
		boost = e[effAffection]
	case types.InteractionGrooming:
		boost = e[effGrooming]
		if item, ok := findItem("soft_brush"); ok && p.ownsLocked(item.ID) {
			return boost, &item
		}
		return boost, nil
	}
	if item, ok := p.bestInGroupLocked(group); ok && group != "" {
		return boost, &item
	}
	return boost, nil
}
