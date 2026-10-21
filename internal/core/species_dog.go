package core

import (
	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// The dog: a social pack animal that needs walks to exercise and to
// relieve itself outside
func init() { RegisterSpecies(dogSpecies) }

var dogSpecies = &SpeciesDef{
	ID:     SpeciesDog,
	Name:   "Dog",
	Plural: "dogs",
	Icon:   "🐶",
	Sprites: map[string]string{
		"idle": "🐶", "happy": "🐕", "eating": "😋", "playing": "🐕", "sleeping": "😴", "sick": "🤒",
		"exploring": "🦮", "social": "🐶", "hibernating": "💤", "dead": "🐶",
	},
	CleanLabel:         "🧹 Clean up",
	SelfGrooming:       0,
	GroomingRate:       0.5,
	Toilet:             ToiletOutside,
	LivingArea:         "Home",
	MessRequest:        "There's a mess in the house that needs cleaning up.",
	MessSymptom:        Symptom{"There's a mess in the house", "Clean up promptly, and take your dog out more often.", "bathroom", false},
	CleanRefusal:       "There's nothing to clean up.",
	DirtIllness:        IllnessParasites,
	Walks:              true,
	ExerciseHint:       "Not enough exercise. Daily walks and play keep the heart fit.",
	StressSign:         "Pacing and restless",
	SameSpeciesMeeting: 0.1,
	IntroductionLessons: map[Species]string{
		SpeciesDog: "Introduce dogs on neutral ground, on loose leashes, with a walk side by side before they meet face to face.",
		SpeciesCat: catDogIntroduction,
	},
	ThreatSignal:   "growled at",
	LeaderText:     "%s leads the pack.",
	LifespanFactor: 0.95,
	SleepWindow:    [2]float64{22, 6},
	BasicFood: supply{item: interaction.Item{ID: "dog_kibble", Name: "Dry dog food", Type: interaction.ItemTypeFood,
		Description: "Complete dry food for dogs. Always in the pantry."}, icon: "🥣", kind: SupplyFood,
		species: []Species{SpeciesDog},
		food:    &foodProfile{nutrition: 1, lesson: "Complete dog food gives a dog everything it needs; measure each meal."}},
	Liking: map[types.InteractionType]float64{
		types.InteractionPetting: 0.1,
		types.InteractionPlaying: 0.05,
		types.InteractionWalking: 0.1,
	},
	UnwantedAttentionLesson: "Dogs show they want a break by turning away, licking their lips, yawning or showing the " +
		"whites of their eyes. Give them space and let them come to you.",
	ReactionTexts: map[types.InteractionType]map[ReactionLevel]string{
		types.InteractionPetting: {
			ReactionDelighted: "%s leans into your hand, tail wagging!",
			ReactionReluctant: "%s turns away and licks its lips. It wants some space.",
		},
	},
	CareLessons: map[types.InteractionType][]string{
		types.InteractionFeeding: {
			"Two measured meals a day suit most adult dogs; table scraps upset their stomach, and some foods are toxic to them.",
		},
		types.InteractionDrinking: {
			"Dogs need fresh water available all the time, especially after exercise and in hot weather.",
		},
		types.InteractionWalking: {
			"Dogs need to go outside several times a day; regular walks prevent accidents and keep them fit.",
			"Always pick up after your dog: feces spread parasites to other dogs and to people.",
		},
		types.InteractionCleaning: {
			"Clean up accidents promptly: feces in the home spread parasites, and lingering smells invite repeat accidents.",
		},
		types.InteractionPlaying: {
			"Daily play and exercise prevent the boredom behind chewing and barking.",
		},
		types.InteractionPetting: {
			"Most dogs like being stroked on the chest and shoulders; turning away or lip licking means 'that's enough'.",
		},
		types.InteractionGrooming: {
			"Check the ears, paws and nails while you brush.",
		},
		types.InteractionMedicalCare: {
			"Yearly check-ups, vaccinations and parasite prevention keep dogs healthy.",
		},
		types.InteractionRewards: {
			"Keep treats to about 10% of the day's food.",
		},
		types.InteractionTraining: {
			"Reward-based training in short, frequent sessions works best for dogs.",
		},
		types.InteractionEnvironmentalEnrichment: {
			"New places and smells exercise a dog's mind as well as its body.",
		},
		types.InteractionDiscipline: {
			"Punishment adds stress and fear; rewarding the behavior you want works better.",
		},
	},
	CareGuide: []GuideTopic{
		{"feeding", "Feeding", "Most adult dogs eat two measured meals a day of complete dog food; puppies eat " +
			"more often. Follow the label for your dog's weight and keep treats to about 10% of the day's food. " +
			"Never give chocolate, grapes or raisins, onions, garlic, or xylitol (a sweetener), which are toxic " +
			"to dogs. Eating too much too fast can upset the stomach."},
		{"water", "Water", "Dogs need fresh water available all the time, and more after exercise and in hot " +
			"weather. Wash the bowl daily."},
		{"bathroom", "Bathroom breaks", "Adult dogs need to go outside to relieve themselves at least three to " +
			"five times a day, and puppies far more often. A dog left too long will have an accident in the house; " +
			"that isn't bad behavior. Always pick up after your dog: feces spread intestinal parasites such as " +
			"roundworms to other dogs and to people. Clean up accidents promptly with an enzyme cleaner."},
		{"exercise", "Walks and exercise", "Dogs need daily walks and play; how much depends on breed, age and " +
			"health, from about 30 minutes to two hours a day. Exercise keeps the heart and muscles fit and " +
			"prevents boredom, which often leads to chewing and barking."},
		{"grooming", "Grooming", "Brush your dog regularly, from weekly for short coats to daily for long ones, " +
			"and check the ears, paws, teeth and nails as you go."},
		{"vet", "Vet care", "Take your dog for a check-up at least once a year and keep vaccinations and parasite " +
			"prevention up to date. See a vet for diarrhea or vomiting that lasts, coughing, not eating, or low " +
			"energy. Illnesses get worse without treatment."},
		{"breathing", "Breathing", "Coughing, sneezing and nasal discharge are signs of a respiratory infection. " +
			"Kennel cough spreads where dogs are kept close together, and dusty or dirty air irritates the airways. " +
			"See a vet if your dog coughs or breathes with effort."},
		{"stress", "Stress", "Stressed dogs may pace, pant, chew, bark or have accidents. Regular walks, a " +
			"predictable routine and reward-based training help; punishment makes stress and fear worse."},
		{"senior", "Older dogs", "Dogs become seniors at about 7 to 10 years old, large breeds sooner. Older dogs " +
			"slow down, sleep more and may get stiff joints. Keep walks shorter and more frequent, keep your dog " +
			"lean to spare its joints, and take it to the vet twice a year: early checks catch arthritis, dental " +
			"disease and heart or kidney problems."},
	},
	DailyRoutine: []checklistItem{
		{"meals", "Feed two meals", 2, types.InteractionFeeding},
		{"water", "Give fresh water", 1, types.InteractionDrinking},
		{"walks", "Take three walks", 3, types.InteractionWalking},
		{"play", "Play together", 1, types.InteractionPlaying},
	},
	PetRegions: []petRegion{
		{ID: "chest", Name: "Chest", liking: 0.2,
			lesson: "Most dogs enjoy a scratch on the chest; reaching from the side and below feels friendlier than from above."},
		{ID: "ears", Name: "Behind the ears", liking: 0.1,
			lesson: "Many dogs love a scratch behind the ears; be gentle, ears are sensitive."},
		{ID: "back", Name: "Shoulders and back", liking: 0.1,
			lesson: "Strokes along the shoulders and back are welcome to most dogs."},
		{ID: "head", Name: "Top of the head", liking: -0.1, sensitive: 1,
			lesson: "Patting a dog on top of the head means reaching over it, which can feel threatening to nervous dogs."},
		{ID: "belly", Name: "Belly", liking: 0, sensitive: 2,
			lesson: "Some dogs love belly rubs, but rolling over can also mean 'please stop'; stop if your dog stiffens or licks its lips."},
		{ID: "paws", Name: "Paws", liking: -0.25, sensitive: 1,
			lesson: "Many dogs dislike having their paws handled; get them used to it slowly with treats, which helps with nail trims."},
	},
	Treats: "dog_treats",
	EnvironmentChanges: []environmentChange{
		{ID: "sniff_game", Name: "Hide treats to sniff out", Icon: "👃", stimulation: 0.9, exploration: 0.3,
			liking: func(t *ai.Traits) float64 { return 0.2 + (t.Curiosity-0.5)*0.3 },
			lesson: "Nose work tires a dog's mind; a short sniffing game can tire a dog out as well as a walk."},
		{ID: "new_smells", Name: "Bring in new things to sniff", Icon: "🍂", stimulation: 0.4, exploration: 0.6,
			liking: func(t *ai.Traits) float64 { return 0.1 + (t.Curiosity-0.5)*0.4 },
			lesson: "New smells are news to a dog; its sense of smell is far keener than ours."},
		{ID: "den", Name: "Make a quiet den", Icon: "🏠", stimulation: 0.1, calming: 0.15,
			liking: func(t *ai.Traits) float64 { return 0.1 + (t.Neuroticism-0.3)*0.4 },
			lesson: "A quiet den where nobody disturbs them helps dogs feel safe, especially nervous ones."},
		{ID: "frozen_toy", Name: "Give a frozen food toy", Icon: "🧊", stimulation: 0.6, calming: 0.1,
			liking: func(t *ai.Traits) float64 { return 0.2 },
			lesson: "A food toy stuffed and frozen keeps a dog busy and calm for a long time."},
		{ID: "rotate_toys", Name: "Swap out the toys", Icon: "🔄", stimulation: 0.5,
			liking: func(t *ai.Traits) float64 { return (t.Openness-0.5)*0.3 - (t.Neuroticism-0.3)*0.2 },
			lesson: "Putting some toys away and bringing them back later makes them feel new again."},
	},
	ConflictLesson:  "Interrupt a scuffle calmly with a noise or a distraction, never with your hands. Separate the dogs to calm down, and keep later meetings short and positive.",
	FriendlySignals: []string{"play-bowed to", "wagged its tail at", "licked"},
	WarySignals:     []string{"sniffed", "circled"},
	IllnessNames: map[string]string{
		IllnessUrinary:     "Urinary tract infection",
		IllnessParasites:   "Intestinal parasites",
		IllnessRespiratory: "Respiratory infection",
		IllnessPoisoning:   "Chocolate poisoning",
		IllnessEye:         "Eye infection",
		IllnessDental:      "Dental disease",
	},
	IllnessSigns: map[string]Symptom{
		IllnessUrinary: {"Needing to go out far more often",
			"Signs of a urinary tract infection, which gets worse until treated. Give medical care (Heal).", "vet", false},
		IllnessParasites: {"Diarrhea and losing weight",
			"Signs of intestinal parasites, often picked up from feces. Give medical care (Heal), then keep the " +
				"home clean.", "bathroom", false},
		IllnessPoisoning: {"Vomiting, restlessness and a racing heart",
			"Signs of chocolate poisoning, which gets worse until treated. Give medical care (Heal) right away.", "feeding", false},
		IllnessRespiratory: {"Coughing and sneezing",
			"Signs of a respiratory infection, which gets worse until treated. Give medical care (Heal).",
			"breathing", false},
		IllnessEye: {"Pawing at a red, runny eye",
			"Signs of an eye infection, which gets worse until treated. Give medical care (Heal).", "eyes", false},
		IllnessDental: {"Dropping food and pawing at the mouth",
			"Signs of painful dental disease. Give medical care (Heal): a vet cleans the teeth. Then brush them daily.",
			"teeth", false},
	},
	GroupLessons: map[string]string{
		GroupPlay: "Watch dogs at play: play bows, loose bodies and taking turns chasing are good signs; stiff bodies and pinning mean it's time for a break.",
		GroupWalk: "Walking dogs together is good exercise and builds their bond; walk them side by side and keep leashes from tangling.",
		GroupNap:  "Dogs that rest together trust each other. Give each dog its own bed too, so it can choose.",
	},
	Supplies: []supply{
		{item: interaction.Item{ID: "dog_wet", Name: "Wet dog food", Price: 8, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeFood, Description: "Complete food with extra moisture. 3 meals."},
			icon: "🥫", kind: SupplyFood, pack: 3,
			food: &foodProfile{nutrition: 1, hydration: 0.1, taste: 0.1,
				lesson: "Wet food is tasty and adds water; many owners mix it with dry food."}},
		{item: interaction.Item{ID: "dog_chicken", Name: "Plain cooked chicken", Price: 12, Rarity: interaction.RarityUncommon,
			Type: interaction.ItemTypeFood, Description: "Boneless and unseasoned; a tasty topper. 2 servings."},
			icon: "🍗", kind: SupplyFood, pack: 2,
			food: &foodProfile{nutrition: 0.6, taste: 0.25,
				lesson: "Plain, boneless cooked chicken is a safe topper; cooked bones splinter and seasoning can be toxic."}},
		{item: interaction.Item{ID: "dog_premium", Name: "Premium dog food", Price: 25, Rarity: interaction.RarityRare,
			Type: interaction.ItemTypeFood, Description: "High-quality complete food that's easy to digest. 3 meals."},
			icon: "🍽️", kind: SupplyFood, pack: 3,
			food: &foodProfile{nutrition: 1.1, taste: 0.15, digestion: 0.05,
				lesson: "A good complete diet keeps digestion healthy; change foods gradually over a week to avoid upsets."}},
		{item: interaction.Item{ID: "chocolate", Name: "Chocolate", Price: 3, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeFood, Description: "Toxic to dogs. 1 piece."},
			icon: "🍫", kind: SupplyFood, pack: 1,
			food: &foodProfile{nutrition: 0.2, taste: 0.3, toxic: true,
				warning: "Chocolate is poisonous to dogs. Feed it anyway?",
				lesson:  "Chocolate contains theobromine, which poisons dogs; dark chocolate is the most dangerous. If a dog eats chocolate, call a vet."}},
		{item: interaction.Item{ID: "dog_treats", Name: "Dog treats", Price: 10, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeFood, Description: "Small rewards for training and good behavior. Bag of 10."},
			icon: "🦴", kind: SupplyTreat, pack: 10},
		{item: interaction.Item{ID: "tennis_ball", Name: "Tennis ball", Price: 10, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeToy, Uses: 40, MaxUses: 40, Description: "For fetch. Wears out."},
			icon: "🎾", kind: SupplyToy, pack: 1,
			toy: &toyProfile{fit: 0.25, exercise: 1.4, stimulation: 0.3,
				lesson: "Fetch is great exercise; use a ball too big to swallow, and stop before your dog is exhausted."}},
		{item: interaction.Item{ID: "rope_toy", Name: "Rope toy", Price: 15, Rarity: interaction.RarityCommon,
			Type: interaction.ItemTypeToy, Uses: 30, MaxUses: 30, Description: "For tug of war. Wears out."},
			icon: "🪢", kind: SupplyToy, pack: 1,
			toy: &toyProfile{fit: 0.2, exercise: 1.1, stimulation: 0.4,
				lesson: "Tug is good fun with rules: your dog lets go on cue, and the game stops if teeth touch skin."}},
		{item: interaction.Item{ID: "chew_toy", Name: "Chew toy", Price: 20, Rarity: interaction.RarityUncommon,
			Type: interaction.ItemTypeToy, Uses: 50, MaxUses: 50, Description: "Something safe to chew. Wears out."},
			icon: "🦷", kind: SupplyToy, pack: 1,
			toy: &toyProfile{fit: 0.1, exercise: 0.2, stimulation: 0.3, calming: 0.1,
				lesson: "Chewing calms dogs and keeps them from chewing furniture; replace a chew toy before it breaks into pieces."}},
	},
}
