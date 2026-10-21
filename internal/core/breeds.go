package core

import (
	"slices"
	"sort"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/genetics"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Breeds. Each species has breeds that differ in looks, temperament and
// care, as real ones do. A breed is a starting point: a new pet's
// personality is drawn around its breed's typical temperament, so two of
// a breed still differ. Its body type (ears, muzzle, coat, legs, tail,
// size) is the breed's, and its coat colors, pattern and eye color are
// picked from the breed's by its genes. Each breed has a few care
// tendencies (see tendencies) and a lifespan relative to its species.
//
// Breeds whose defining looks come from a condition that harms the animal
// (folded ears from a cartilage disorder, short legs from dwarfism) are
// left out.
//
// Easy breeds are available from the start; the others unlock as the
// player earns care achievements (BreedTier).

// Breed tiers: which breeds are available
const (
	TierEasy      = 1 // From the start
	TierModerate  = 2 // After "All Grown Up" and "Daily Carer"
	TierDemanding = 3 // After also "Nurse" and "Attentive Carer"
)

// TierAchievements are the achievements, by any pet, that unlock each tier
var TierAchievements = map[int][]string{
	TierModerate:  {"grown_up", "daily_carer"},
	TierDemanding: {"grown_up", "daily_carer", "nurse", "attentive"},
}

// UnlockedTier returns the highest breed tier the achievements unlock
func UnlockedTier(achieved []string) int {
	tier := TierEasy
	for _, t := range []int{TierModerate, TierDemanding} {
		for _, id := range TierAchievements[t] {
			if !slices.Contains(achieved, id) {
				return tier
			}
		}
		tier = t
	}
	return tier
}

// Body features, for drawing
const (
	// Builds
	BuildSlim   = "slim"
	BuildMedium = "medium"
	BuildStocky = "stocky"
	BuildLarge  = "large"
	BuildLean   = "lean" // Sighthounds

	// Ears
	EarsPointed = "pointed"
	EarsLarge   = "large"
	EarsTufted  = "tufted"
	EarsRounded = "rounded" // Small and rounded
	EarsErect   = "erect"
	EarsFloppy  = "floppy"
	EarsLong    = "long" // Long and hanging
	EarsBat     = "bat"
	EarsButton  = "button" // Folded forward at the tip
	EarsRose    = "rose"

	// Muzzles
	MuzzleFlat   = "flat"
	MuzzleShort  = "short"
	MuzzleMedium = "medium"
	MuzzleLong   = "long"

	// Coats
	CoatHairless = "hairless"
	CoatShort    = "short"
	CoatCurly    = "curly"
	CoatWavy     = "wavy"
	CoatSemi     = "semi" // Semi-long
	CoatLong     = "long"
	CoatShaggy   = "shaggy"

	// Legs
	LegsShort  = "short"
	LegsMedium = "medium"
	LegsLong   = "long"

	// Tails
	TailLong   = "long"
	TailPlume  = "plume"
	TailCurled = "curled"
	TailOtter  = "otter"
	TailThin   = "thin"
	TailStub   = "stub"
	TailBushy  = "bushy"
)

// Coat patterns
const (
	PatternSolid      = "solid"
	PatternBicolor    = "bicolor"
	PatternTuxedo     = "tuxedo"
	PatternTabby      = "tabby"
	PatternTicked     = "ticked"
	PatternSpotted    = "spotted"
	PatternPoint      = "point" // Colorpoint: darker face, ears, paws and tail
	PatternMitted     = "mitted"
	PatternTortie     = "tortie"
	PatternCalico     = "calico"
	PatternVan        = "van" // White with color on the head and tail
	PatternWhite      = "white"
	PatternMerle      = "merle"
	PatternDoubleMerl = "double_merle" // Mostly white; from two merle parents
	PatternBrindle    = "brindle"
	PatternSaddle     = "saddle"
	PatternTricolor   = "tricolor"
	PatternMask       = "mask"
	PatternTan        = "tan" // Tan points (black and tan)
	PatternPatches    = "patches"
)

// CoatColor is a coat or eye color
type CoatColor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Hex  string `json:"hex"`
}

// coatColors are the colors coats and eyes come in
var coatColors = []CoatColor{
	{"black", "black", "#2b2b2e"},
	{"blue", "blue-grey", "#8592a0"},
	{"red", "red", "#d9822b"},
	{"cream", "cream", "#f0dcae"},
	{"chocolate", "chocolate", "#6b4226"},
	{"lilac", "lilac", "#b9a9ab"},
	{"white", "white", "#f7f5f0"},
	{"brown", "brown", "#8a5a34"},
	{"silver", "silver", "#c3c6cc"},
	{"seal", "seal brown", "#4a3426"},
	{"fawn", "fawn", "#d2aa7c"},
	{"cinnamon", "cinnamon", "#a0602f"},
	{"gold", "gold", "#e0a94a"},
	{"yellow", "yellow", "#ecc985"},
	{"tan", "tan", "#c2884a"},
	{"grey", "grey", "#8d8d8d"},
	{"sable", "sable", "#7b5532"},
	{"apricot", "apricot", "#efb98a"},
	{"ruddy", "ruddy", "#b4693a"},
	{"rust", "rust", "#9c4a22"},
	// Eye colors
	{"eye_blue", "blue", "#5aa0e6"},
	{"eye_green", "green", "#6fae53"},
	{"eye_gold", "gold", "#d9a521"},
	{"eye_copper", "copper", "#c56a1c"},
	{"eye_brown", "brown", "#6b3f1f"},
	{"eye_amber", "amber", "#c98a2b"},
	{"eye_hazel", "hazel", "#97803c"},
}

// CoatColors returns the colors coats and eyes come in
func CoatColors() []CoatColor { return slices.Clone(coatColors) }

// Body is a breed's body type
type Body struct {
	Build  string     `json:"build"`
	Ears   string     `json:"ears"`
	Muzzle string     `json:"muzzle"`
	Coat   string     `json:"coat"`
	Legs   string     `json:"legs"`
	Tail   string     `json:"tail"`
	Size   [2]float64 `json:"size"` // Drawing scale, smallest to largest (1.0 is a typical cat or dog)
}

// BreedDef is a breed of a species
type BreedDef struct {
	ID          string   `json:"id"`
	Species     Species  `json:"species"`
	Name        string   `json:"name"`
	Tier        int      `json:"tier"`
	About       string   `json:"about"`       // Where it comes from and what it is like
	Temperament string   `json:"temperament"` // In a few words
	Body        Body     `json:"body"`
	Colors      []string `json:"colors"`   // Coat colors (IDs)
	Patterns    []string `json:"patterns"` // Coat patterns
	Eyes        []string `json:"eyes"`     // Eye colors (IDs)
	Tendencies  []string `json:"tendencies"`
	Lifespan    float64  `json:"lifespan"` // Relative to its species
	Mixed       bool     `json:"mixed"`    // A mixed breed (any pairing of different breeds)

	traits map[string]float64 // Typical temperament, by personality trait
}

// Default breeds: what pets from before breeds, and babies of different
// breeds, are
const (
	BreedDomesticShorthair = "domestic_shorthair"
	BreedDomesticLonghair  = "domestic_longhair"
	BreedMixed             = "mixed_breed"
)

var anyCatColor = []string{"black", "blue", "red", "cream", "chocolate", "white", "brown", "silver"}
var anyCatPattern = []string{PatternSolid, PatternTabby, PatternBicolor, PatternTuxedo, PatternTortie, PatternCalico,
	PatternWhite}
var catEyes = []string{"eye_green", "eye_gold", "eye_copper", "eye_hazel", "eye_blue"}

// breeds are every breed, cats then dogs, in the order shown
var breeds = []BreedDef{
	// Cats
	{ID: BreedDomesticShorthair, Species: SpeciesCat, Name: "Domestic Shorthair", Tier: TierEasy, Mixed: true,
		About:       "The everyday cat of mixed ancestry: most cats in homes are domestic shorthairs, in every color and pattern.",
		Temperament: "Varies widely", Lifespan: 1.0,
		Body:   Body{BuildMedium, EarsPointed, MuzzleMedium, CoatShort, LegsMedium, TailLong, [2]float64{0.9, 1.05}},
		Colors: anyCatColor, Patterns: anyCatPattern, Eyes: catEyes},
	{ID: BreedDomesticLonghair, Species: SpeciesCat, Name: "Domestic Longhair", Tier: TierEasy, Mixed: true,
		About:       "A long-haired cat of mixed ancestry. The long coat needs regular brushing to stop mats.",
		Temperament: "Varies widely", Lifespan: 1.0, Tendencies: []string{TendGrooming},
		Body:   Body{BuildMedium, EarsPointed, MuzzleMedium, CoatLong, LegsMedium, TailPlume, [2]float64{0.9, 1.05}},
		Colors: anyCatColor, Patterns: anyCatPattern, Eyes: catEyes},
	{ID: "british_shorthair", Species: SpeciesCat, Name: "British Shorthair", Tier: TierEasy,
		About:       "A round, sturdy cat with a dense plush coat, most famous in blue-grey with copper eyes.",
		Temperament: "Calm, independent, easygoing", Lifespan: 1.0, Tendencies: []string{TendLowEnergy},
		traits: map[string]float64{"energy": 0.3, "neuroticism": 0.25, "independence": 0.7, "affectionate": 0.5},
		Body:   Body{BuildStocky, EarsRounded, MuzzleShort, CoatShort, LegsMedium, TailLong, [2]float64{1.0, 1.15}},
		Colors: []string{"blue", "black", "cream", "lilac", "white"}, Patterns: []string{PatternSolid, PatternBicolor, PatternWhite},
		Eyes: []string{"eye_copper", "eye_gold"}},
	{ID: "russian_blue", Species: SpeciesCat, Name: "Russian Blue", Tier: TierEasy,
		About:       "A slim, graceful cat with a silvery blue-grey double coat and green eyes.",
		Temperament: "Gentle, reserved with strangers", Lifespan: 1.05,
		traits: map[string]float64{"extraversion": 0.35, "neuroticism": 0.45, "affectionate": 0.6, "intelligence": 0.65},
		Body:   Body{BuildSlim, EarsLarge, MuzzleMedium, CoatShort, LegsLong, TailLong, [2]float64{0.9, 1.0}},
		Colors: []string{"blue"}, Patterns: []string{PatternSolid}, Eyes: []string{"eye_green"}},
	{ID: "birman", Species: SpeciesCat, Name: "Birman", Tier: TierEasy,
		About:       "A semi-longhaired colorpoint cat with blue eyes and white 'gloves' on all four paws.",
		Temperament: "Gentle, affectionate, quiet", Lifespan: 1.05,
		traits: map[string]float64{"affectionate": 0.75, "agreeableness": 0.75, "energy": 0.45},
		Body:   Body{BuildMedium, EarsPointed, MuzzleMedium, CoatSemi, LegsMedium, TailPlume, [2]float64{0.95, 1.1}},
		Colors: []string{"seal", "blue", "chocolate", "lilac"}, Patterns: []string{PatternMitted}, Eyes: []string{"eye_blue"}},
	{ID: "bombay", Species: SpeciesCat, Name: "Bombay", Tier: TierEasy,
		About:       "A sleek, glossy black cat with copper eyes, bred to look like a small panther.",
		Temperament: "Affectionate, playful, outgoing", Lifespan: 1.0,
		traits: map[string]float64{"affectionate": 0.75, "playfulness": 0.65, "extraversion": 0.65},
		Body:   Body{BuildMedium, EarsRounded, MuzzleShort, CoatShort, LegsMedium, TailLong, [2]float64{0.9, 1.0}},
		Colors: []string{"black"}, Patterns: []string{PatternSolid}, Eyes: []string{"eye_copper", "eye_gold"}},
	{ID: "siamese", Species: SpeciesCat, Name: "Siamese", Tier: TierModerate,
		About:       "A slim colorpoint cat with large ears and blue eyes, known for its loud voice and need for company.",
		Temperament: "Very social, vocal, curious", Lifespan: 1.0, Tendencies: []string{TendSeparation},
		traits: map[string]float64{"extraversion": 0.8, "curiosity": 0.75, "playfulness": 0.7, "affectionate": 0.75, "intelligence": 0.7},
		Body:   Body{BuildSlim, EarsLarge, MuzzleLong, CoatShort, LegsLong, TailThin, [2]float64{0.85, 0.95}},
		Colors: []string{"seal", "blue", "chocolate", "lilac"}, Patterns: []string{PatternPoint}, Eyes: []string{"eye_blue"}},
	{ID: "ragdoll", Species: SpeciesCat, Name: "Ragdoll", Tier: TierModerate,
		About:       "A large, semi-longhaired colorpoint cat with blue eyes, named for going limp when held. Some carry an inherited heart disease (hypertrophic cardiomyopathy).",
		Temperament: "Very relaxed, affectionate", Lifespan: 1.0, Tendencies: []string{TendHeart},
		traits: map[string]float64{"neuroticism": 0.2, "affectionate": 0.8, "agreeableness": 0.8, "energy": 0.35},
		Body:   Body{BuildLarge, EarsPointed, MuzzleMedium, CoatSemi, LegsMedium, TailPlume, [2]float64{1.1, 1.25}},
		Colors: []string{"seal", "blue", "chocolate", "lilac"}, Patterns: []string{PatternPoint, PatternMitted, PatternBicolor},
		Eyes: []string{"eye_blue"}},
	{ID: "maine_coon", Species: SpeciesCat, Name: "Maine Coon", Tier: TierModerate,
		About:       "One of the largest cats, with a shaggy coat, tufted ears and a bushy tail. Prone to hip dysplasia and an inherited heart disease.",
		Temperament: "Gentle giant, sociable, playful", Lifespan: 0.95, Tendencies: []string{TendJoints, TendHeart},
		traits: map[string]float64{"agreeableness": 0.75, "extraversion": 0.65, "playfulness": 0.65, "intelligence": 0.65},
		Body:   Body{BuildLarge, EarsTufted, MuzzleMedium, CoatShaggy, LegsLong, TailBushy, [2]float64{1.25, 1.4}},
		Colors: []string{"brown", "black", "red", "silver", "cream", "white"}, Patterns: []string{PatternTabby, PatternSolid, PatternBicolor, PatternTortie},
		Eyes: []string{"eye_green", "eye_gold", "eye_copper"}},
	{ID: "abyssinian", Species: SpeciesCat, Name: "Abyssinian", Tier: TierModerate,
		About:       "A lithe cat with a ticked coat in which each hair is banded. Prone to gum disease, so its teeth need care.",
		Temperament: "Very active, curious, playful", Lifespan: 1.0, Tendencies: []string{TendHighEnergy, TendTeeth},
		traits: map[string]float64{"energy": 0.85, "curiosity": 0.85, "playfulness": 0.8, "intelligence": 0.7},
		Body:   Body{BuildSlim, EarsLarge, MuzzleMedium, CoatShort, LegsLong, TailLong, [2]float64{0.85, 0.95}},
		Colors: []string{"ruddy", "cinnamon", "blue", "fawn"}, Patterns: []string{PatternTicked}, Eyes: []string{"eye_gold", "eye_green", "eye_hazel"}},
	{ID: "norwegian_forest", Species: SpeciesCat, Name: "Norwegian Forest Cat", Tier: TierModerate,
		About:       "A big climbing cat from Norway with a thick, water-resistant double coat that needs brushing, especially when it sheds.",
		Temperament: "Calm, sociable, a climber", Lifespan: 1.0, Tendencies: []string{TendGrooming, TendFeelsHeat},
		traits: map[string]float64{"agreeableness": 0.7, "neuroticism": 0.3, "curiosity": 0.6},
		Body:   Body{BuildLarge, EarsTufted, MuzzleMedium, CoatLong, LegsLong, TailBushy, [2]float64{1.15, 1.3}},
		Colors: []string{"brown", "black", "white", "silver", "red", "blue"}, Patterns: []string{PatternTabby, PatternBicolor, PatternSolid, PatternWhite},
		Eyes: []string{"eye_green", "eye_gold", "eye_copper"}},
	{ID: "burmese", Species: SpeciesCat, Name: "Burmese", Tier: TierModerate,
		About:       "A compact cat with a glossy short coat and gold eyes that bonds closely with its people.",
		Temperament: "People-oriented, playful", Lifespan: 1.05, Tendencies: []string{TendSeparation},
		traits: map[string]float64{"affectionate": 0.85, "extraversion": 0.8, "playfulness": 0.7, "independence": 0.2},
		Body:   Body{BuildStocky, EarsRounded, MuzzleShort, CoatShort, LegsMedium, TailLong, [2]float64{0.9, 1.0}},
		Colors: []string{"sable", "blue", "chocolate", "lilac"}, Patterns: []string{PatternSolid}, Eyes: []string{"eye_gold"}},
	{ID: "devon_rex", Species: SpeciesCat, Name: "Devon Rex", Tier: TierModerate,
		About:       "A pixie-like cat with huge ears and a thin, wavy coat that keeps it less warm than other cats.",
		Temperament: "Playful, mischievous, social", Lifespan: 1.0, Tendencies: []string{TendFeelsCold},
		traits: map[string]float64{"playfulness": 0.85, "extraversion": 0.8, "curiosity": 0.75, "affectionate": 0.75},
		Body:   Body{BuildSlim, EarsLarge, MuzzleShort, CoatWavy, LegsLong, TailThin, [2]float64{0.8, 0.9}},
		Colors: []string{"black", "blue", "red", "cream", "white", "seal"}, Patterns: []string{PatternSolid, PatternTabby, PatternPoint, PatternBicolor, PatternWhite},
		Eyes: catEyes},
	{ID: "oriental_shorthair", Species: SpeciesCat, Name: "Oriental Shorthair", Tier: TierModerate,
		About:       "A Siamese relative in solid colors and patterns, usually with green eyes; just as talkative and social.",
		Temperament: "Very social, vocal", Lifespan: 1.0, Tendencies: []string{TendSeparation},
		traits: map[string]float64{"extraversion": 0.85, "curiosity": 0.75, "affectionate": 0.75, "independence": 0.2},
		Body:   Body{BuildSlim, EarsLarge, MuzzleLong, CoatShort, LegsLong, TailThin, [2]float64{0.85, 0.95}},
		Colors: []string{"black", "blue", "chocolate", "red", "cinnamon", "white"}, Patterns: []string{PatternSolid, PatternTabby, PatternBicolor, PatternWhite},
		Eyes: []string{"eye_green", "eye_blue"}},
	{ID: "turkish_van", Species: SpeciesCat, Name: "Turkish Van", Tier: TierModerate,
		About:       "A large, strong cat, white with color only on its head and tail, known for liking water.",
		Temperament: "Energetic, bold", Lifespan: 1.0, Tendencies: []string{TendHighEnergy},
		traits: map[string]float64{"energy": 0.8, "playfulness": 0.75, "independence": 0.6},
		Body:   Body{BuildLarge, EarsPointed, MuzzleMedium, CoatSemi, LegsLong, TailPlume, [2]float64{1.05, 1.2}},
		Colors: []string{"red", "cream", "black", "blue"}, Patterns: []string{PatternVan}, Eyes: []string{"eye_gold", "eye_blue", "eye_copper"}},
	{ID: "siberian", Species: SpeciesCat, Name: "Siberian", Tier: TierModerate,
		About:       "A powerful forest cat from Russia with a thick triple coat that needs brushing and makes hot weather hard.",
		Temperament: "Playful, affectionate", Lifespan: 1.0, Tendencies: []string{TendGrooming, TendFeelsHeat},
		traits: map[string]float64{"playfulness": 0.7, "affectionate": 0.7, "agreeableness": 0.7},
		Body:   Body{BuildLarge, EarsTufted, MuzzleMedium, CoatLong, LegsMedium, TailBushy, [2]float64{1.1, 1.25}},
		Colors: []string{"brown", "silver", "red", "black", "seal", "white"}, Patterns: []string{PatternTabby, PatternSolid, PatternPoint, PatternWhite},
		Eyes: []string{"eye_green", "eye_gold", "eye_blue"}},
	{ID: "persian", Species: SpeciesCat, Name: "Persian", Tier: TierDemanding,
		About:       "A flat-faced cat with a very long coat. The short face makes breathing, heat and sore eyes a problem; many carry inherited polycystic kidney disease. Needs daily grooming.",
		Temperament: "Calm, gentle, low energy", Lifespan: 0.95,
		Tendencies: []string{TendShortAirways, TendGrooming, TendKidneys, TendEyes},
		traits:     map[string]float64{"energy": 0.2, "neuroticism": 0.3, "affectionate": 0.65, "playfulness": 0.35},
		Body:       Body{BuildStocky, EarsRounded, MuzzleFlat, CoatLong, LegsShort, TailPlume, [2]float64{0.95, 1.1}},
		Colors:     []string{"white", "black", "blue", "cream", "red", "silver", "chocolate"}, Patterns: []string{PatternSolid, PatternBicolor, PatternTabby, PatternPoint, PatternWhite},
		Eyes: []string{"eye_copper", "eye_blue", "eye_green"}},
	{ID: "exotic_shorthair", Species: SpeciesCat, Name: "Exotic Shorthair", Tier: TierDemanding,
		About:       "A Persian with a short, plush coat: easier to groom, but the same flat face, eye and kidney risks.",
		Temperament: "Calm, affectionate", Lifespan: 0.95, Tendencies: []string{TendShortAirways, TendKidneys, TendEyes},
		traits: map[string]float64{"energy": 0.3, "neuroticism": 0.3, "affectionate": 0.7},
		Body:   Body{BuildStocky, EarsRounded, MuzzleFlat, CoatShort, LegsShort, TailLong, [2]float64{0.95, 1.1}},
		Colors: []string{"cream", "black", "blue", "red", "silver", "white"}, Patterns: []string{PatternSolid, PatternTabby, PatternBicolor, PatternWhite},
		Eyes: []string{"eye_copper", "eye_gold", "eye_blue"}},
	{ID: "sphynx", Species: SpeciesCat, Name: "Sphynx", Tier: TierDemanding,
		About:       "A hairless, wrinkled cat. Without fur it gets cold easily and its skin needs regular baths for oil; prone to an inherited heart disease.",
		Temperament: "Very affectionate, social, energetic", Lifespan: 0.85, Tendencies: []string{TendFeelsCold, TendSkinCare, TendHeart},
		traits: map[string]float64{"affectionate": 0.9, "extraversion": 0.85, "energy": 0.7, "playfulness": 0.75},
		Body:   Body{BuildMedium, EarsLarge, MuzzleMedium, CoatHairless, LegsLong, TailThin, [2]float64{0.9, 1.0}},
		Colors: []string{"cream", "lilac", "fawn", "black"}, Patterns: []string{PatternSolid, PatternBicolor}, Eyes: catEyes},
	{ID: "bengal", Species: SpeciesCat, Name: "Bengal", Tier: TierDemanding,
		About:       "A muscular cat with a spotted or marbled coat from a leopard cat ancestor, with a great need for activity.",
		Temperament: "Very energetic, curious, clever", Lifespan: 0.9, Tendencies: []string{TendHighEnergy},
		traits: map[string]float64{"energy": 0.9, "curiosity": 0.85, "intelligence": 0.8, "playfulness": 0.8},
		Body:   Body{BuildMedium, EarsRounded, MuzzleMedium, CoatShort, LegsLong, TailLong, [2]float64{1.0, 1.15}},
		Colors: []string{"brown", "silver", "seal"}, Patterns: []string{PatternSpotted, PatternTabby}, Eyes: []string{"eye_green", "eye_gold", "eye_blue"}},
	{ID: "cornish_rex", Species: SpeciesCat, Name: "Cornish Rex", Tier: TierDemanding,
		About:       "A slender cat with a tight, curly coat of down hair only, so it feels the cold; tireless at play.",
		Temperament: "Very active, playful", Lifespan: 1.0, Tendencies: []string{TendFeelsCold, TendHighEnergy},
		traits: map[string]float64{"energy": 0.85, "playfulness": 0.85, "extraversion": 0.75},
		Body:   Body{BuildSlim, EarsLarge, MuzzleLong, CoatCurly, LegsLong, TailThin, [2]float64{0.8, 0.9}},
		Colors: []string{"black", "white", "red", "cream", "blue"}, Patterns: []string{PatternSolid, PatternBicolor, PatternTortie, PatternWhite},
		Eyes: catEyes},

	// Dogs
	{ID: BreedMixed, Species: SpeciesDog, Name: "Mixed Breed", Tier: TierEasy, Mixed: true,
		About:       "A dog of mixed ancestry, in every shape, size and color. Looks and temperament vary widely.",
		Temperament: "Varies widely", Lifespan: 1.0,
		Body:   Body{BuildMedium, EarsButton, MuzzleMedium, CoatShort, LegsMedium, TailLong, [2]float64{0.8, 1.1}},
		Colors: []string{"black", "brown", "tan", "white", "cream", "fawn", "grey"}, Patterns: []string{PatternSolid, PatternBicolor, PatternPatches, PatternTan, PatternBrindle},
		Eyes: []string{"eye_brown", "eye_amber"}},
	{ID: "labrador", Species: SpeciesDog, Name: "Labrador Retriever", Tier: TierEasy,
		About:       "A friendly retriever with a short, water-resistant coat and an otter tail. Very food-motivated (many carry a gene that makes them hungrier), and prone to hip and elbow dysplasia.",
		Temperament: "Friendly, eager to please, playful", Lifespan: 1.0, Tendencies: []string{TendFoodMotivated, TendJoints},
		traits: map[string]float64{"agreeableness": 0.85, "extraversion": 0.8, "playfulness": 0.8, "intelligence": 0.7, "energy": 0.7},
		Body:   Body{BuildLarge, EarsFloppy, MuzzleMedium, CoatShort, LegsMedium, TailOtter, [2]float64{1.05, 1.2}},
		Colors: []string{"black", "yellow", "chocolate"}, Patterns: []string{PatternSolid}, Eyes: []string{"eye_brown", "eye_hazel"}},
	{ID: "golden_retriever", Species: SpeciesDog, Name: "Golden Retriever", Tier: TierEasy,
		About:       "A gentle retriever with a feathered golden coat that sheds and needs brushing; prone to hip dysplasia.",
		Temperament: "Very friendly, gentle, eager", Lifespan: 0.95, Tendencies: []string{TendGrooming, TendJoints},
		traits: map[string]float64{"agreeableness": 0.9, "affectionate": 0.8, "intelligence": 0.75, "playfulness": 0.75},
		Body:   Body{BuildLarge, EarsFloppy, MuzzleMedium, CoatLong, LegsMedium, TailPlume, [2]float64{1.05, 1.2}},
		Colors: []string{"gold", "cream", "red"}, Patterns: []string{PatternSolid}, Eyes: []string{"eye_brown"}},
	{ID: "beagle", Species: SpeciesDog, Name: "Beagle", Tier: TierEasy,
		About:       "A small scent hound with long ears and a tricolor coat, led by its nose and very fond of food.",
		Temperament: "Curious, friendly, merry", Lifespan: 1.05, Tendencies: []string{TendFoodMotivated},
		traits: map[string]float64{"curiosity": 0.85, "extraversion": 0.75, "agreeableness": 0.75, "independence": 0.6},
		Body:   Body{BuildMedium, EarsLong, MuzzleMedium, CoatShort, LegsShort, TailLong, [2]float64{0.75, 0.85}},
		Colors: []string{"tan", "black", "white"}, Patterns: []string{PatternTricolor, PatternBicolor}, Eyes: []string{"eye_brown", "eye_hazel"}},
	{ID: "poodle", Species: SpeciesDog, Name: "Poodle", Tier: TierEasy,
		About:       "A very clever dog with a curly coat that keeps growing and needs regular brushing and clipping.",
		Temperament: "Very intelligent, active, proud", Lifespan: 1.1, Tendencies: []string{TendGrooming},
		traits: map[string]float64{"intelligence": 0.9, "energy": 0.65, "playfulness": 0.7, "affectionate": 0.65},
		Body:   Body{BuildSlim, EarsLong, MuzzleLong, CoatCurly, LegsLong, TailLong, [2]float64{0.9, 1.1}},
		Colors: []string{"black", "white", "apricot", "chocolate", "grey", "cream"}, Patterns: []string{PatternSolid}, Eyes: []string{"eye_brown"}},
	{ID: "shiba_inu", Species: SpeciesDog, Name: "Shiba Inu", Tier: TierEasy,
		About:       "An ancient Japanese breed with a fox-like face, a curled tail and cat-like cleanliness.",
		Temperament: "Independent, alert, bold", Lifespan: 1.1,
		traits: map[string]float64{"independence": 0.85, "territoriality": 0.65, "affectionate": 0.4, "agreeableness": 0.45},
		Body:   Body{BuildMedium, EarsErect, MuzzleMedium, CoatShort, LegsMedium, TailCurled, [2]float64{0.8, 0.9}},
		Colors: []string{"red", "black", "cream"}, Patterns: []string{PatternTan, PatternMask}, Eyes: []string{"eye_brown"}},
	{ID: "german_shepherd", Species: SpeciesDog, Name: "German Shepherd", Tier: TierModerate,
		About:       "A large, intelligent working dog with erect ears and a black and tan saddle; needs lots of activity and is prone to hip and elbow dysplasia.",
		Temperament: "Intelligent, loyal, alert", Lifespan: 0.9, Tendencies: []string{TendJoints, TendHighEnergy},
		traits: map[string]float64{"intelligence": 0.9, "loyalty": 0.9, "territoriality": 0.75, "energy": 0.75},
		Body:   Body{BuildLarge, EarsErect, MuzzleLong, CoatShort, LegsLong, TailBushy, [2]float64{1.15, 1.3}},
		Colors: []string{"tan", "black", "sable"}, Patterns: []string{PatternSaddle, PatternSolid}, Eyes: []string{"eye_brown"}},
	{ID: "chihuahua", Species: SpeciesDog, Name: "Chihuahua", Tier: TierModerate,
		About:       "The smallest breed: tiny, big-eared and devoted. Feels the cold, and its crowded teeth need daily care.",
		Temperament: "Devoted, alert, wary of strangers", Lifespan: 1.15, Tendencies: []string{TendFeelsCold, TendTeeth},
		traits: map[string]float64{"loyalty": 0.85, "neuroticism": 0.6, "territoriality": 0.7, "agreeableness": 0.45},
		Body:   Body{BuildSlim, EarsLarge, MuzzleShort, CoatShort, LegsMedium, TailCurled, [2]float64{0.5, 0.58}},
		Colors: []string{"fawn", "cream", "black", "chocolate", "white"}, Patterns: []string{PatternSolid, PatternTan, PatternBicolor}, Eyes: []string{"eye_brown"}},
	{ID: "yorkshire_terrier", Species: SpeciesDog, Name: "Yorkshire Terrier", Tier: TierModerate,
		About:       "A tiny terrier with a long, silky blue and tan coat that needs daily brushing; small mouths mean dental care matters.",
		Temperament: "Bold, affectionate, lively", Lifespan: 1.1, Tendencies: []string{TendGrooming, TendFeelsCold, TendTeeth},
		traits: map[string]float64{"extraversion": 0.7, "territoriality": 0.6, "affectionate": 0.75},
		Body:   Body{BuildSlim, EarsErect, MuzzleShort, CoatLong, LegsShort, TailLong, [2]float64{0.5, 0.58}},
		Colors: []string{"blue", "black"}, Patterns: []string{PatternTan}, Eyes: []string{"eye_brown"}},
	{ID: "greyhound", Species: SpeciesDog, Name: "Greyhound", Tier: TierModerate,
		About:       "A lean sighthound built for sprinting but calm indoors. Its thin coat and little body fat make it feel the cold; prone to dental disease.",
		Temperament: "Gentle, calm, quiet", Lifespan: 1.0, Tendencies: []string{TendFeelsCold, TendLowEnergy, TendTeeth},
		traits: map[string]float64{"neuroticism": 0.4, "agreeableness": 0.8, "energy": 0.4, "affectionate": 0.7},
		Body:   Body{BuildLean, EarsRose, MuzzleLong, CoatShort, LegsLong, TailThin, [2]float64{1.1, 1.25}},
		Colors: []string{"fawn", "black", "grey", "white"}, Patterns: []string{PatternSolid, PatternBrindle, PatternBicolor}, Eyes: []string{"eye_brown"}},
	{ID: "jack_russell", Species: SpeciesDog, Name: "Jack Russell Terrier", Tier: TierModerate,
		About:       "A small, tough terrier, mostly white with tan or black patches, with far more energy than its size suggests.",
		Temperament: "Very energetic, bold, clever", Lifespan: 1.15, Tendencies: []string{TendHighEnergy},
		traits: map[string]float64{"energy": 0.9, "curiosity": 0.8, "territoriality": 0.65, "intelligence": 0.75},
		Body:   Body{BuildMedium, EarsButton, MuzzleMedium, CoatShort, LegsMedium, TailLong, [2]float64{0.65, 0.75}},
		Colors: []string{"tan", "black"}, Patterns: []string{PatternPatches, PatternTricolor}, Eyes: []string{"eye_brown"}},
	{ID: "boxer", Species: SpeciesDog, Name: "Boxer", Tier: TierModerate,
		About:       "A muscular, playful dog with a short muzzle, fawn or brindle; some are white, and about one white Boxer in five is deaf in at least one ear. Prone to inherited heart disease.",
		Temperament: "Playful, energetic, loyal", Lifespan: 0.85, Tendencies: []string{TendHighEnergy, TendHeart},
		traits: map[string]float64{"playfulness": 0.85, "energy": 0.8, "loyalty": 0.8, "extraversion": 0.75},
		Body:   Body{BuildLarge, EarsButton, MuzzleShort, CoatShort, LegsLong, TailStub, [2]float64{1.1, 1.2}},
		Colors: []string{"fawn", "red"}, Patterns: []string{PatternMask, PatternBrindle, PatternWhite}, Eyes: []string{"eye_brown"}},
	{ID: "cavalier", Species: SpeciesDog, Name: "Cavalier King Charles Spaniel", Tier: TierModerate,
		About:       "A small, gentle spaniel with long feathered ears. Most develop mitral valve heart disease as they age, and they hate being left alone.",
		Temperament: "Very affectionate, gentle", Lifespan: 0.9, Tendencies: []string{TendHeart, TendSeparation},
		traits: map[string]float64{"affectionate": 0.95, "agreeableness": 0.9, "independence": 0.15, "extraversion": 0.75},
		Body:   Body{BuildMedium, EarsLong, MuzzleShort, CoatSemi, LegsMedium, TailPlume, [2]float64{0.7, 0.78}},
		Colors: []string{"chocolate", "red", "black"}, Patterns: []string{PatternBicolor, PatternTricolor, PatternSolid, PatternTan},
		Eyes: []string{"eye_brown"}},
	{ID: "french_bulldog", Species: SpeciesDog, Name: "French Bulldog", Tier: TierDemanding,
		About:       "A small, flat-faced dog with bat ears. Short airways make heat and exercise hard; skin folds and prominent eyes need cleaning.",
		Temperament: "Playful, affectionate, easygoing", Lifespan: 0.85, Tendencies: []string{TendShortAirways, TendSkinCare, TendEyes},
		traits: map[string]float64{"affectionate": 0.8, "playfulness": 0.7, "energy": 0.45, "agreeableness": 0.75},
		Body:   Body{BuildStocky, EarsBat, MuzzleFlat, CoatShort, LegsShort, TailStub, [2]float64{0.7, 0.78}},
		Colors: []string{"fawn", "cream", "black", "white"}, Patterns: []string{PatternSolid, PatternBrindle, PatternPatches}, Eyes: []string{"eye_brown"}},
	{ID: "bulldog", Species: SpeciesDog, Name: "Bulldog", Tier: TierDemanding,
		About:       "A heavy, flat-faced dog with deep skin folds. Breathing, heat, skin, eyes and joints all need care, and its lifespan is short.",
		Temperament: "Calm, gentle, stubborn", Lifespan: 0.7, Tendencies: []string{TendShortAirways, TendSkinCare, TendJoints, TendEyes},
		traits: map[string]float64{"energy": 0.25, "agreeableness": 0.8, "neuroticism": 0.3, "independence": 0.6},
		Body:   Body{BuildStocky, EarsRose, MuzzleFlat, CoatShort, LegsShort, TailStub, [2]float64{0.9, 1.0}},
		Colors: []string{"white", "fawn", "red"}, Patterns: []string{PatternPatches, PatternBrindle, PatternSolid}, Eyes: []string{"eye_brown"}},
	{ID: "pug", Species: SpeciesDog, Name: "Pug", Tier: TierDemanding,
		About:       "A small, flat-faced dog with a curled tail. Short airways make heat dangerous, and its prominent eyes are easily hurt.",
		Temperament: "Affectionate, playful, sociable", Lifespan: 0.9, Tendencies: []string{TendShortAirways, TendEyes},
		traits: map[string]float64{"affectionate": 0.85, "extraversion": 0.75, "energy": 0.4, "playfulness": 0.7},
		Body:   Body{BuildStocky, EarsButton, MuzzleFlat, CoatShort, LegsShort, TailCurled, [2]float64{0.65, 0.72}},
		Colors: []string{"fawn", "black"}, Patterns: []string{PatternMask, PatternSolid}, Eyes: []string{"eye_brown"}},
	{ID: "shih_tzu", Species: SpeciesDog, Name: "Shih Tzu", Tier: TierDemanding,
		About:       "A small dog with a long flowing coat and a short face: daily brushing, eye cleaning and tooth care.",
		Temperament: "Affectionate, calm, friendly", Lifespan: 1.1, Tendencies: []string{TendGrooming, TendShortAirways, TendEyes, TendTeeth},
		traits: map[string]float64{"affectionate": 0.8, "agreeableness": 0.8, "energy": 0.4},
		Body:   Body{BuildStocky, EarsLong, MuzzleFlat, CoatLong, LegsShort, TailPlume, [2]float64{0.6, 0.68}},
		Colors: []string{"white", "gold", "black", "brown"}, Patterns: []string{PatternBicolor, PatternSolid}, Eyes: []string{"eye_brown"}},
	{ID: "border_collie", Species: SpeciesDog, Name: "Border Collie", Tier: TierDemanding,
		About:       "A herding dog thought to be the most intelligent breed, with a huge need for exercise and things to do.",
		Temperament: "Extremely intelligent, very energetic", Lifespan: 1.05, Tendencies: []string{TendHighEnergy},
		traits: map[string]float64{"intelligence": 0.95, "energy": 0.95, "curiosity": 0.8, "loyalty": 0.8},
		Body:   Body{BuildMedium, EarsButton, MuzzleLong, CoatSemi, LegsMedium, TailPlume, [2]float64{0.9, 1.0}},
		Colors: []string{"black", "red", "blue"}, Patterns: []string{PatternTuxedo, PatternTricolor, PatternMerle}, Eyes: []string{"eye_brown", "eye_amber", "eye_blue"}},
	{ID: "siberian_husky", Species: SpeciesDog, Name: "Siberian Husky", Tier: TierDemanding,
		About:       "A sled dog with a thick double coat, a mask and often blue eyes, bred to run for hours in the cold.",
		Temperament: "Very energetic, sociable, independent", Lifespan: 1.0, Tendencies: []string{TendHighEnergy, TendFeelsHeat, TendGrooming},
		traits: map[string]float64{"energy": 0.95, "extraversion": 0.8, "independence": 0.75, "agreeableness": 0.7},
		Body:   Body{BuildMedium, EarsErect, MuzzleMedium, CoatSemi, LegsMedium, TailBushy, [2]float64{1.0, 1.1}},
		Colors: []string{"grey", "black", "red", "white"}, Patterns: []string{PatternMask, PatternSolid}, Eyes: []string{"eye_blue", "eye_brown"}},
	{ID: "australian_shepherd", Species: SpeciesDog, Name: "Australian Shepherd", Tier: TierDemanding,
		About:       "A herding dog, often merle, with a great need for work and exercise. Two merle parents can have 'double merle' puppies, which are often deaf and have eye problems, so merle is never bred to merle.",
		Temperament: "Very intelligent, very energetic", Lifespan: 1.0, Tendencies: []string{TendHighEnergy, TendGrooming},
		traits: map[string]float64{"intelligence": 0.9, "energy": 0.9, "loyalty": 0.8, "territoriality": 0.6},
		Body:   Body{BuildMedium, EarsButton, MuzzleMedium, CoatSemi, LegsMedium, TailStub, [2]float64{0.95, 1.05}},
		Colors: []string{"blue", "red", "black"}, Patterns: []string{PatternMerle, PatternTricolor}, Eyes: []string{"eye_blue", "eye_brown", "eye_amber"}},
	{ID: "great_dane", Species: SpeciesDog, Name: "Great Dane", Tier: TierDemanding,
		About:       "A giant, gentle dog. Like most giant breeds it ages quickly, with a short lifespan, and is prone to joint problems and heart disease (dilated cardiomyopathy).",
		Temperament: "Gentle, calm, friendly", Lifespan: 0.65, Tendencies: []string{TendJoints, TendHeart, TendLowEnergy},
		traits: map[string]float64{"agreeableness": 0.85, "energy": 0.35, "affectionate": 0.75, "neuroticism": 0.3},
		Body:   Body{BuildLarge, EarsFloppy, MuzzleLong, CoatShort, LegsLong, TailThin, [2]float64{1.4, 1.5}},
		Colors: []string{"fawn", "black", "blue", "white"}, Patterns: []string{PatternMask, PatternBrindle, PatternSolid, PatternPatches},
		Eyes: []string{"eye_brown"}},
	{ID: "bernese", Species: SpeciesDog, Name: "Bernese Mountain Dog", Tier: TierDemanding,
		About:       "A giant tricolor farm dog with a long thick coat. Struggles in the heat, is prone to joint problems and has a short lifespan.",
		Temperament: "Calm, gentle, affectionate", Lifespan: 0.65, Tendencies: []string{TendJoints, TendFeelsHeat, TendGrooming},
		traits: map[string]float64{"agreeableness": 0.9, "affectionate": 0.85, "energy": 0.45, "loyalty": 0.85},
		Body:   Body{BuildLarge, EarsFloppy, MuzzleMedium, CoatLong, LegsMedium, TailBushy, [2]float64{1.3, 1.4}},
		Colors: []string{"black"}, Patterns: []string{PatternTricolor}, Eyes: []string{"eye_brown"}},
}

// Breeds returns the breeds of a species ("" for all)
func Breeds(species Species) []BreedDef {
	var list []BreedDef
	for _, b := range breeds {
		if species == "" || b.Species == species {
			list = append(list, b)
		}
	}
	return list
}

// FindBreed returns a breed
func FindBreed(id string) (BreedDef, bool) {
	for _, b := range breeds {
		if b.ID == id {
			return b, true
		}
	}
	return BreedDef{}, false
}

// defaultBreed is a species' breed for pets of mixed or unknown ancestry
func defaultBreed(species Species) string {
	if species == SpeciesDog {
		return BreedMixed
	}
	return BreedDomesticShorthair
}

// PetLook is how a pet looks: its breed's body, and the colors, pattern
// and size its genes chose
type PetLook struct {
	Body
	Size    float64 `json:"scale"`   // Drawing scale
	Color   string  `json:"color"`   // Main coat color (ID)
	Color2  string  `json:"color2"`  // Second coat color (ID)
	Pattern string  `json:"pattern"` // Coat pattern
	Eye     string  `json:"eye"`     // Eye color (ID)
	Eye2    string  `json:"eye2"`    // The other eye (odd eyes)
}

// pick returns the option a gene value chooses
func pick(options []string, gene float64) string {
	if len(options) == 0 {
		return ""
	}
	return options[min(int(gene*float64(len(options))), len(options)-1)]
}

// lookFromGenes works out a pet of a breed's looks from its genome
func lookFromGenes(b BreedDef, g *genetics.Genome) PetLook {
	value := func(t genetics.TraitType) float64 { return min(max(geneValue(g, t), 0), 0.999) }
	look := PetLook{Body: b.Body}
	look.Size = b.Body.Size[0] + (b.Body.Size[1]-b.Body.Size[0])*value(genetics.TraitSize)
	look.Color = pick(b.Colors, value(genetics.TraitColorPrimary))
	look.Color2 = pick(b.Colors, value(genetics.TraitColorSecondary))
	look.Pattern = pick(b.Patterns, value(genetics.TraitPattern))
	look.Eye = pick(b.Eyes, value(genetics.TraitEyeColor))
	look.Eye2 = look.Eye
	look.normalize(b.Species)
	return look
}

// normalize makes a look consistent: patterns that need a white or
// darker second color get one, and white cats may have odd eyes
func (l *PetLook) normalize(species Species) {
	switch l.Pattern {
	case PatternWhite:
		l.Color, l.Color2 = "white", "white"
	case PatternBicolor, PatternTuxedo, PatternMitted, PatternVan, PatternPatches, PatternDoubleMerl:
		l.Color2 = "white"
	case PatternTan, PatternSaddle, PatternTricolor:
		if l.Color == "tan" {
			l.Color = "black"
		}
		l.Color2 = "tan"
	case PatternMask:
		l.Color2 = "black"
		if l.Color == "black" {
			l.Color2 = "white"
		}
	case PatternPoint:
		l.Color2, l.Color = l.Color, "cream" // A pale body with darker points
	case PatternCalico, PatternTortie:
		l.Color, l.Color2 = "black", "red"
	default:
		if l.Color2 == l.Color {
			l.Color2 = ""
		}
	}
	if l.Eye2 == "" {
		l.Eye2 = l.Eye
	}
}

// Tendencies: what a breed makes a pet prone to, each changing something
// the simulation already models
const (
	TendHighEnergy    = "high_energy"
	TendLowEnergy     = "low_energy"
	TendGrooming      = "grooming"
	TendSkinCare      = "skin_care"
	TendShortAirways  = "short_airways"
	TendFeelsCold     = "feels_cold"
	TendFeelsHeat     = "feels_heat"
	TendJoints        = "joints"
	TendHeart         = "heart"
	TendFoodMotivated = "food_motivated"
	TendSeparation    = "separation"
	TendKidneys       = "kidneys"
	TendEyes          = "eyes"
	TendTeeth         = "teeth"
)

// Tendency is something a breed is prone to
type Tendency struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Effect string `json:"effect"` // What it does in the game
	Lesson string `json:"lesson"` // The real-world care lesson
	Topic  string `json:"topic"`  // Care guide topic

	effects []itemEffect
}

// tendencies are every tendency
var tendencies = []Tendency{
	{TendHighEnergy, "High energy", "Needs for exercise and things to do build 30% faster.",
		"Working and sporting breeds need hours of exercise and things to think about every day; bored, they get stressed and destructive.",
		"exercise", []itemEffect{eff(effExerciseNeed, 1.3)}},
	{TendLowEnergy, "Low energy", "The need for exercise builds 25% slower.",
		"Calmer breeds still need daily walks and play to stay fit, just less of it.",
		"exercise", []itemEffect{eff(effExerciseNeed, 0.75)}},
	{TendGrooming, "Grooming need", "The coat gets dirty 40% faster.",
		"Long and double coats mat and trap dirt without regular brushing; mats pull on the skin and hurt.",
		"grooming", []itemEffect{eff(effCleanliness, 1.4)}},
	{TendSkinCare, "Skin care", "The coat (or skin) gets dirty 30% faster.",
		"Hairless cats build up skin oil and need regular gentle baths; dogs with skin folds need the folds cleaned and dried.",
		"grooming", []itemEffect{eff(effCleanliness, 1.3)}},
	{TendShortAirways, "Short airways", "Above 27 °C weather stress is 50% higher; foul air harms the airways 30% faster; play tires 15% faster.",
		"Flat-faced (brachycephalic) breeds have narrowed airways. They overheat easily, so keep them cool and walks short in warm weather; noisy breathing isn't normal.",
		"breathing", []itemEffect{effWhen(effWeatherStress, 1.5, hot), eff(effAirDamage, 1.3), eff(effTiredness, 1.15)}},
	{TendFeelsCold, "Feels the cold", "Below 10 °C weather stress is 50% higher.",
		"Hairless, thin-coated and tiny pets lose heat fast: a warm bed, and a coat or sweater outside in winter.",
		"weather", []itemEffect{effWhen(effWeatherStress, 1.5, cold)}},
	{TendFeelsHeat, "Feels the heat", "Above 27 °C weather stress is 30% higher.",
		"Heavy double coats are made for snow. In summer keep these pets in the shade with plenty of water; never shave a double coat.",
		"weather", []itemEffect{effWhen(effWeatherStress, 1.3, hot)}},
	{TendJoints, "Joints", "Seniors lose fitness 25% faster; trip injuries 2 points likelier.",
		"Large and long-backed breeds are prone to hip and elbow dysplasia and back problems. Keep them lean, avoid hard jumping while young, and use ramps or steps when old.",
		"senior", []itemEffect{effWhen(effFitnessLoss, 1.25, seniors), eff(effTripInjury, 2)}},
	{TendHeart, "Heart", "The heart ages 20% faster.",
		"Some breeds inherit heart disease (cardiomyopathy or mitral valve disease). Vets listen for murmurs at checkups; tiring easily or fast breathing at rest are warning signs.",
		"exercise", nil},
	{TendFoodMotivated, "Food-motivated", "Hunger builds 15% faster; training with treats does 20% more.",
		"Some breeds are always hungry and gain weight easily. Measure meals, count treats as part of the day's food, and use that love of food for training.",
		"feeding", []itemEffect{eff(effHunger, 1.15), eff(effTreatTraining, 1.2)}},
	{TendSeparation, "Separation stress", "Stress builds 30% faster when left alone.",
		"People-oriented breeds can become anxious when left alone. Build up time alone slowly, and leave toys and something that smells of home.",
		"stress", []itemEffect{eff(effAloneStress, 1.3)}},
	{TendKidneys, "Kidney disease", "Inherited kidney disease slowly lowers what the kidneys can do from adulthood.",
		"Some breeds inherit polycystic kidney disease. A DNA test finds it; plenty of water, wet food and a kidney-support diet slow it.",
		"kidneys", nil},
	{TendEyes, "Eyes", "The eyes get sore a little every day and heal more slowly.",
		"Prominent eyes and eyelid problems make eyes sore, dry and easily scratched. Wipe them daily and see a vet for a red or cloudy eye.",
		"eyes", nil},
	{TendTeeth, "Teeth", "Plaque builds 50% faster.",
		"Small mouths crowd the teeth, and some breeds are prone to gum disease. Brush daily and have the teeth checked.",
		"teeth", nil},
}

// FindTendency returns a tendency
func FindTendency(id string) (Tendency, bool) {
	for _, t := range tendencies {
		if t.ID == id {
			return t, true
		}
	}
	return Tendency{}, false
}

// breedDefLocked returns the pet's breed. Caller must hold p.mu.
func (p *DigitalPet) breedDefLocked() BreedDef {
	if b, ok := FindBreed(p.Breed); ok && b.Species == p.speciesLocked() {
		return b
	}
	b, _ := FindBreed(defaultBreed(p.speciesLocked()))
	return b
}

// breedEffectsLocked returns the effects of the pet's tendencies, each
// scaled by how strongly it has it. Caller must hold p.mu.
func (p *DigitalPet) breedEffectsLocked() []itemEffect {
	var effects []itemEffect
	for _, id := range tendencyIDs(p.Tendencies) {
		t, ok := FindTendency(id)
		if !ok {
			continue
		}
		for _, e := range t.effects {
			e.value = scaled(e.kind, e.value, p.Tendencies[id])
			effects = append(effects, e)
		}
	}
	return effects
}

// tendencyIDs returns a pet's tendencies in order
func tendencyIDs(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// predispositionsLocked returns what the pet's species and tendencies
// make its body prone to. Caller must hold p.mu.
func (p *DigitalPet) predispositionsLocked() biology.Predispositions {
	pre := biology.Predispositions{KidneyAging: 0.6, Dental: 1}
	if p.speciesLocked() == SpeciesCat {
		pre.KidneyAging = 1 // Kidney disease is common in old cats
	}
	t := p.Tendencies
	pre.KidneyDisease = t[TendKidneys]
	pre.Eyes = t[TendEyes]
	pre.Dental += 0.5 * t[TendTeeth]
	pre.Heart = t[TendHeart]
	return pre
}

// lifespanFactorLocked is how long the pet can expect to live relative to
// BaseLifespan: its species' and its breed's. Caller must hold p.mu.
func (p *DigitalPet) lifespanFactorLocked() float64 {
	breed := p.BreedLifespan
	if breed <= 0 {
		breed = 1
	}
	return p.defLocked().LifespanFactor * breed
}

// traitSetters set personality traits by name
var traitSetters = map[string]func(*ai.Traits, float64){
	"openness":       func(t *ai.Traits, v float64) { t.Openness = v },
	"extraversion":   func(t *ai.Traits, v float64) { t.Extraversion = v },
	"agreeableness":  func(t *ai.Traits, v float64) { t.Agreeableness = v },
	"neuroticism":    func(t *ai.Traits, v float64) { t.Neuroticism = v },
	"playfulness":    func(t *ai.Traits, v float64) { t.Playfulness = v },
	"independence":   func(t *ai.Traits, v float64) { t.Independence = v },
	"loyalty":        func(t *ai.Traits, v float64) { t.Loyalty = v },
	"intelligence":   func(t *ai.Traits, v float64) { t.Intelligence = v },
	"energy":         func(t *ai.Traits, v float64) { t.EnergyLevel = v },
	"affectionate":   func(t *ai.Traits, v float64) { t.Affectionate = v },
	"curiosity":      func(t *ai.Traits, v float64) { t.Curiosity = v },
	"territoriality": func(t *ai.Traits, v float64) { t.Territoriality = v },
}

// breedSpread is how far a pet's traits can be from its breed's typical
// temperament
const breedSpread = 0.15

// NewPetOfBreed creates a pet of a breed: its temperament drawn around
// the breed's, its looks from its genes. An unknown breed, or one of
// another species, gives the species' default breed.
func NewPetOfBreed(name string, owner types.UserID, species Species, breedID string) *DigitalPet {
	b, ok := FindBreed(breedID)
	if !ok || b.Species != species {
		b, _ = FindBreed(defaultBreed(species))
	}
	pet := NewDigitalPetRandom(name, owner)
	pet.Species = species
	traits := pet.Personality.Traits
	for name, typical := range b.traits {
		if set := traitSetters[name]; set != nil {
			set(traits, typical+(float64(randIntn(2001))/1000-1)*breedSpread)
		}
	}
	traits.Clamp()
	pet.Genome = nil
	pet.ensureGenomeLocked() // Match the genome to the personality
	pet.setBreedLocked(b, pet.Genome)
	return pet
}

// setBreedLocked makes the pet a purebred of breed b, with looks from its
// genome. Caller must hold p.mu (or own the pet alone).
func (p *DigitalPet) setBreedLocked(b BreedDef, g *genetics.Genome) {
	p.Breed = b.ID
	look := lookFromGenes(b, g)
	p.Look = &look
	p.BreedLifespan = b.Lifespan
	p.Tendencies = map[string]float64{}
	for _, t := range b.Tendencies {
		p.Tendencies[t] = 1
	}
	p.Biology.EnsurePhysiology()
	p.Biology.Physiology.Hearing = congenitalHearing(look)
}

// congenitalHearing returns the hearing a pet is born with. White cats
// with blue eyes are often deaf (the gene for a white coat also affects
// the inner ear); so are double merle dogs and some white Boxers.
// Deafness can be in one ear (hearing 0.5) or both (0).
func congenitalHearing(l PetLook) float64 {
	risk := 0.0
	switch l.Pattern {
	case PatternWhite:
		blue := 0
		for _, e := range []string{l.Eye, l.Eye2} {
			if e == "eye_blue" {
				blue++
			}
		}
		risk = []float64{0.2, 0.4, 0.7}[blue]
		if l.Body.Ears == EarsButton { // A white Boxer
			risk = 0.18
		}
	case PatternDoubleMerl:
		risk = 0.5
	}
	if risk == 0 || !chance(risk) {
		return 1
	}
	if chance(0.5) {
		return 0.5
	}
	return 0
}

// ensureBreedLocked gives a pet from before breeds its species' default
// breed, with looks from the genes it already has. Caller must hold p.mu.
func (p *DigitalPet) ensureBreedLocked() {
	if p.Look != nil && p.Breed != "" {
		return
	}
	p.ensureGenomeLocked()
	b, _ := FindBreed(defaultBreed(p.speciesLocked()))
	hearing := 1.0
	if p.Biology.Physiology != nil && p.Biology.Physiology.SensesSet {
		hearing = p.Biology.Physiology.Hearing
	}
	p.setBreedLocked(b, p.Genome)
	p.Biology.Physiology.Hearing = hearing // A pet doesn't go deaf by being given a breed
}

// BreedInfo is a pet's breed, looks and tendencies, for the page
type BreedInfo struct {
	Breed      BreedDef           `json:"breed"`
	Look       PetLook            `json:"look"`
	Tendencies []TendencyStrength `json:"tendencies"`
	Lineage    *Lineage           `json:"lineage,omitempty"`
	Hearing    string             `json:"hearing,omitempty"` // "deaf in one ear", "deaf"
}

// TendencyStrength is a tendency a pet has, and how strongly
type TendencyStrength struct {
	Tendency
	Strength float64 `json:"strength"` // 1, or 0.5 when inherited by a mixed breed from both parents
}

// BreedInfo returns the pet's breed, looks and tendencies
func (p *DigitalPet) BreedInfo() BreedInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ensureBreedLocked()
	info := BreedInfo{Breed: p.breedDefLocked(), Look: *p.Look, Tendencies: []TendencyStrength{}, Lineage: cloneLineage(p.Lineage)}
	for _, id := range tendencyIDs(p.Tendencies) {
		if t, ok := FindTendency(id); ok {
			info.Tendencies = append(info.Tendencies, TendencyStrength{t, p.Tendencies[id]})
		}
	}
	switch h := p.Biology.Physiology.Hearing; {
	case h <= 0.05:
		info.Hearing = "deaf"
	case h <= 0.55:
		info.Hearing = "deaf in one ear"
	case h < 0.8:
		info.Hearing = "hard of hearing"
	}
	return info
}

// GetLook returns the pet's breed and how it looks
func (p *DigitalPet) GetLook() (string, PetLook) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ensureBreedLocked()
	return p.Breed, *p.Look
}

// TypicalLook returns how a typical pet of a breed looks (for a visiting
// pet at a playdate, of which only the breed is known)
func TypicalLook(breedID string) PetLook {
	b, ok := FindBreed(breedID)
	if !ok {
		return PetLook{}
	}
	look := PetLook{Body: b.Body, Size: (b.Body.Size[0] + b.Body.Size[1]) / 2,
		Color: pick(b.Colors, 0), Color2: pick(b.Colors, 0.5), Pattern: pick(b.Patterns, 0), Eye: pick(b.Eyes, 0)}
	look.normalize(b.Species)
	return look
}

// repairBreed checks a loaded pet's breed and looks: an unknown breed or
// one of another species becomes the species' default, and tendencies
// keep to known ones at no more than full strength
func (p *DigitalPet) repairBreed() {
	if b, ok := FindBreed(p.Breed); p.Breed != "" && (!ok || b.Species != p.speciesLocked()) {
		p.Breed, p.Look = "", nil
	}
	for id, strength := range p.Tendencies {
		if _, ok := FindTendency(id); !ok || !(strength > 0) {
			delete(p.Tendencies, id)
		} else {
			p.Tendencies[id] = min(strength, 1)
		}
	}
	if !(p.BreedLifespan >= 0.5 && p.BreedLifespan <= 1.2) {
		p.BreedLifespan = 0
	}
	if p.Look != nil {
		p.Look.Size = min(max(p.Look.Size, 0.4), 1.6)
	}
}
