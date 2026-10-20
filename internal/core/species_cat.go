package core

import (
	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// The cat: a territorial, independent hunter that keeps itself clean,
// uses a litter box and is most active at dawn and dusk
func init() { RegisterSpecies(catSpecies) }

var catSpecies = &SpeciesDef{
	ID:     SpeciesCat,
	Name:   "Cat",
	Plural: "cats",
	Icon:   "🐱",
	Sprites: map[string]string{
		"idle": "🐱", "happy": "😺", "eating": "😋", "playing": "😸", "sleeping": "😴", "sick": "🤒",
		"exploring": "🐈", "social": "😻", "hibernating": "💤", "dead": "😿",
	},
	CleanLabel:         "🧹 Scoop litter",
	TwilightActive:     true,
	SelfGrooming:       0.3,
	GroomingRate:       1,
	Toilet:             ToiletLitterBox,
	LivingArea:         "Litter box",
	MessRequest:        "%s's litter box needs cleaning.",
	MessSymptom:        Symptom{"The litter box smells", "Scoop the litter box at least once a day. A dirty box can make your cat ill.", "litter", false},
	CleanRefusal:       "The litter box is already clean.",
	DirtIllness:        IllnessUrinary,
	NoWalks:            "Cats don't go for walks; play with %s indoors instead.",
	ExerciseHint:       "Not enough exercise. Daily play keeps the heart fit.",
	StressSign:         "Hiding and over-grooming",
	SameSpeciesMeeting: -0.1,
	IntroductionLessons: map[Species]string{
		SpeciesCat: "Introduce cats slowly: keep them apart at first, swap bedding so they learn each other's scent, " +
			"then let them see each other before short, supervised meetings.",
		SpeciesDog: catDogIntroduction,
	},
	ThreatSignal:   "hissed at",
	CrowdSensitive: true,
	LeaderText:     "%s is top cat.",
	LifespanFactor: 1.05,
	SleepWindow:    [2]float64{23, 5},
	BasicFood: supply{item: interaction.Item{ID: "cat_dry", Name: "Dry cat food", Type: interaction.ItemTypeFood,
		Description: "Complete dry food for cats. Always in the pantry."}, icon: "🥣", kind: SupplyFood,
		species: []Species{SpeciesCat},
		food:    &foodProfile{nutrition: 1, lesson: "Complete cat food gives a cat everything it needs; measure each meal."}},
	UnwantedAttentionLesson: "Cats show they've had enough with a twitching tail, flattened ears or rippling skin. " +
		"Stop before they have to swat; let your cat come to you.",
	CareLessons: map[types.InteractionType][]string{
		types.InteractionFeeding: {
			"Two measured meals a day keep most adult cats at a healthy weight.",
			"Many adult cats can't digest milk well; water is the best drink.",
		},
		types.InteractionDrinking: {
			"Fresh water every day, away from the food bowl, helps cats drink enough to protect their kidneys and bladder.",
		},
		types.InteractionCleaning: {
			"Scooping the litter box every day keeps your cat using it and helps prevent urinary problems.",
			"Ammonia from a dirty litter box irritates a cat's airways; a clean box means cleaner air.",
		},
		types.InteractionPlaying: {
			"Short daily play sessions keep an indoor cat fit and prevent boredom.",
		},
		types.InteractionPetting: {
			"Most cats prefer gentle strokes on the head and cheeks; let your cat decide when it has had enough.",
		},
		types.InteractionGrooming: {
			"Brushing removes loose hair, which means fewer hairballs, and lets you check the skin and coat.",
		},
		types.InteractionMedicalCare: {
			"A check-up at least once a year catches problems early.",
		},
		types.InteractionRewards: {
			"Treats should be no more than about 10% of what your cat eats in a day.",
		},
		types.InteractionTraining: {
			"Cats learn best in short sessions with rewards; punishment doesn't work.",
		},
		types.InteractionEnvironmentalEnrichment: {
			"New things to explore keep a curious cat's mind busy; indoors, window perches and new toys do the same.",
		},
		types.InteractionDiscipline: {
			"Punishment adds stress and fear; rewarding the behavior you want works better.",
		},
	},
	CareGuide: []GuideTopic{
		{"feeding", "Feeding", "Adult cats do well on two measured meals a day of complete cat food. Follow the " +
			"portion on the label for your cat's weight; free-feeding and extra treats lead to weight gain. Treats " +
			"should be no more than about 10% of the day's food, and eating too much at once can make a cat throw " +
			"up. Never give onions, garlic or chocolate, and remember that many adult cats can't digest milk well."},
		{"water", "Water", "Cats often don't drink enough. Give fresh water every day in a clean bowl, away from " +
			"the food and the litter box. Wet food and water fountains help cats drink more, which protects the " +
			"kidneys and bladder."},
		{"litter", "Litter box", "Scoop the litter box at least once a day, and empty and wash it regularly. A " +
			"common rule is one box per cat plus one, in quiet spots. Cats avoid a dirty box and may go elsewhere " +
			"or hold their urine, which is linked to urinary problems. Ammonia from an unclean box also irritates " +
			"the airways."},
		{"exercise", "Play and exercise", "Indoor cats need daily play to stay fit and avoid boredom. A few short " +
			"sessions with a wand toy, letting the cat stalk, chase and catch, imitate hunting. A cat that doesn't " +
			"exercise loses fitness and gains weight."},
		{"grooming", "Grooming", "Brushing removes loose hair, which means fewer hairballs, and is a chance to " +
			"check the skin, ears and coat. Short-haired cats need brushing about once a week, long-haired cats " +
			"daily."},
		{"vet", "Vet care", "Take your cat for a check-up at least once a year and keep vaccinations up to date. " +
			"See a vet promptly for straining in the litter box or going outside it (an emergency in male cats), " +
			"sneezing with runny eyes, repeated vomiting, not eating, or hiding. Illnesses get worse without " +
			"treatment."},
		{"breathing", "Breathing", "Sneezing and runny eyes and nose are signs of an upper respiratory " +
			"infection, which is common in cats. Dust, smoke and ammonia from a dirty litter box irritate a cat's " +
			"airways. Keep the litter box clean and the room aired, and see a vet if the signs don't clear up."},
		{"stress", "Stress", "Cats under long-term stress may hide, over-groom, stop eating or go outside the " +
			"litter box. Keep a predictable routine, give your cat places to hide and climb, and avoid punishment, " +
			"which only adds to stress."},
		{"senior", "Older cats", "Cats are seniors from about 11 years old and often live into their late teens. " +
			"Older cats sleep more, may lose muscle and find high places harder to reach. Take a senior cat to the " +
			"vet twice a year, keep it active with gentle play, and watch for weight loss, drinking more, or changes " +
			"in litter box habits, which can be early signs of kidney disease, diabetes or an overactive thyroid."},
	},
	DailyRoutine: []checklistItem{
		{"meals", "Feed two meals", 2, types.InteractionFeeding},
		{"water", "Give fresh water", 1, types.InteractionDrinking},
		{"litter", "Scoop the litter box", 1, types.InteractionCleaning},
		{"play", "Play together", 1, types.InteractionPlaying},
		{"brush", "Brush the coat", 1, types.InteractionGrooming},
	},
	PetRegions: []petRegion{
		{ID: "cheeks", Name: "Cheeks and chin", liking: 0.2,
			lesson: "Cats have scent glands in their cheeks and chin; rubbing there is how they mark you as their own."},
		{ID: "head", Name: "Top of the head", liking: 0.1,
			lesson: "Most cats enjoy gentle strokes on the head; let your cat push into your hand."},
		{ID: "back", Name: "Back", liking: 0,
			lesson: "Long strokes along the back are fine for many cats; watch the tail for twitching."},
		{ID: "tail", Name: "Base of the tail", liking: -0.1, sensitive: 1,
			lesson: "Many cats dislike having their tail touched, though some enjoy a scratch at its base."},
		{ID: "belly", Name: "Belly", liking: -0.35, sensitive: 1,
			lesson: "A cat rolling over is showing trust, not asking for a belly rub; many grab and kick if touched there."},
	},
	Treats: "cat_treats",
	EnvironmentChanges: []environmentChange{
		{ID: "box", Name: "Put out a cardboard box", Icon: "📦", stimulation: 0.3, calming: 0.1,
			liking: func(t *ai.Traits) float64 { return 0.2 + (t.Neuroticism-0.3)*0.3 },
			lesson: "Boxes give cats a safe place to hide; hiding helps a stressed cat cope."},
		{ID: "perch", Name: "Set up a window perch", Icon: "🪟", stimulation: 0.6, exploration: 0.5,
			liking: func(t *ai.Traits) float64 { return 0.1 + (t.Curiosity-0.5)*0.4 },
			lesson: "A window perch lets an indoor cat watch birds and passers-by, which keeps its mind busy."},
		{ID: "forage", Name: "Hide food to find", Icon: "🔍", stimulation: 0.8, exercise: 0.2,
			liking: func(t *ai.Traits) float64 { return 0.15 + (t.Curiosity-0.5)*0.3 },
			lesson: "Hiding small portions of food lets a cat 'hunt' for its meal, which keeps indoor cats active."},
		{ID: "scratching_post", Name: "Add a scratching post", Icon: "🪵", stimulation: 0.2, calming: 0.05,
			liking: func(t *ai.Traits) float64 { return 0.15 },
			lesson: "Scratching is natural: it keeps claws healthy and marks territory. A sturdy post saves the furniture."},
		{ID: "rotate_toys", Name: "Swap out the toys", Icon: "🔄", stimulation: 0.5,
			liking: func(t *ai.Traits) float64 { return (t.Openness-0.5)*0.3 - (t.Neuroticism-0.3)*0.2 },
			lesson: "Putting some toys away and bringing them back later makes them feel new again."},
	},
	ConflictLesson:  "Cats that clash need space: separate them, give each its own food, litter box and high places, and reintroduce them slowly. Never punish hissing; it's a warning.",
	FriendlySignals: []string{"slow-blinked at", "rubbed against", "touched noses with"},
	WarySignals:     []string{"watched", "kept its distance from"},
	IllnessNames: map[string]string{
		IllnessUrinary:     "Urinary tract infection",
		IllnessParasites:   "Intestinal parasites",
		IllnessRespiratory: "Upper respiratory infection",
		IllnessPoisoning:   "Poisoning",
	},
	IllnessSigns: map[string]Symptom{
		IllnessUrinary: {"Straining and making frequent trips to the litter box",
			"Signs of a urinary tract infection, which gets worse until treated. Give medical care (Heal).", "litter", false},
		IllnessParasites: {"Diarrhea and losing weight",
			"Signs of intestinal parasites. Give medical care (Heal), then keep the litter box clean.", "vet", false},
		IllnessPoisoning: {"Vomiting and restlessness",
			"Signs of poisoning, which gets worse until treated. Give medical care (Heal) right away.", "feeding", false},
		IllnessRespiratory: {"Sneezing, with runny eyes and nose",
			"Signs of an upper respiratory infection, which gets worse until treated. Give medical care (Heal).",
			"breathing", false},
	},
	GroupLessons: map[string]string{
		GroupPlay: "Cats play best with cats they know. Give each cat its own toy, and stop the game if ears go flat or tails lash.",
		GroupNap:  "Cats that sleep curled up together are bonded; cats that don't get along need separate beds and resting places.",
	},
	Supplies: []supply{
		{item: interaction.Item{ID: "cat_wet", Name: "Wet cat food", Price: 8, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeFood, Description: "Complete food with plenty of water in it. 3 meals."},
			icon: "🥫", kind: SupplyFood, pack: 3,
			food: &foodProfile{nutrition: 1, hydration: 0.15, taste: 0.1,
				lesson: "Wet food adds water to a cat's diet, which helps cats that don't drink much."}},
		{item: interaction.Item{ID: "cat_fish", Name: "Plain cooked fish", Price: 12, Rarity: interaction.RarityUncommon,
			Type: interaction.ItemTypeFood, Description: "A tasty topper, not a complete meal. 2 servings."},
			icon: "🐟", kind: SupplyFood, pack: 2,
			food: &foodProfile{nutrition: 0.6, taste: 0.25,
				lesson: "Plain cooked fish is a treat, not a diet: on its own it lacks nutrients cats need. Never feed raw fish regularly."}},
		{item: interaction.Item{ID: "cat_premium", Name: "Premium cat food", Price: 25, Rarity: interaction.RarityRare,
			Type: interaction.ItemTypeFood, Description: "High-quality complete food that's easy to digest. 3 meals."},
			icon: "🍽️", kind: SupplyFood, pack: 3,
			food: &foodProfile{nutrition: 1.1, taste: 0.15, digestion: 0.05,
				lesson: "A good complete diet keeps digestion healthy; change foods gradually over a week to avoid upsets."}},
		{item: interaction.Item{ID: "cat_treats", Name: "Cat treats", Price: 10, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeFood, Description: "Small rewards for training and good behavior. Bag of 10."},
			icon: "🍪", kind: SupplyTreat, pack: 10},
		{item: interaction.Item{ID: "wand_toy", Name: "Feather wand", Price: 15, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeToy, Uses: 30, MaxUses: 30, Description: "A feather on a string to stalk and pounce on. Wears out."},
			icon: "🪶", kind: SupplyToy, pack: 1,
			toy: &toyProfile{fit: 0.25, exercise: 1.3, stimulation: 0.6,
				lesson: "A wand toy lets a cat stalk, chase and pounce, the way it would hunt; let it catch the feather at the end."}},
		{item: interaction.Item{ID: "catnip_mouse", Name: "Catnip mouse", Price: 10, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeToy, Uses: 20, MaxUses: 20, Description: "A toy mouse filled with catnip. Wears out."},
			icon: "🐭", kind: SupplyToy, pack: 1,
			toy: &toyProfile{fit: 0.1, exercise: 0.8, stimulation: 0.4, catnip: true}},
		{item: interaction.Item{ID: "laser_pointer", Name: "Laser pointer", Price: 20, Rarity: interaction.RarityUncommon,
			Type: interaction.ItemTypeToy, Description: "A red dot to chase."},
			icon: "🔴", kind: SupplyToy, pack: 1,
			toy: &toyProfile{fit: 0.15, exercise: 1.4, stimulation: 0.3,
				lesson: "Chasing a dot that can never be caught can frustrate a cat; finish with a toy it can catch. Never shine it in eyes."}},
	},
}

// catDogIntroduction teaches how to introduce a cat and a dog
const catDogIntroduction = "Keep the dog on a leash for early meetings and give the cat high places and a way out; " +
	"never let the dog chase the cat."
