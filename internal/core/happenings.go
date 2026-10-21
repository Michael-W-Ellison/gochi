package core

import (
	"errors"
	"hash/fnv"
	"slices"
	"strings"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Home happenings: unexpected events at home, each with two or three
// choices. At most one comes every 20 minutes of play, only while the pet
// is home, awake and well (the game server asks only while the player is
// there). The right choice cheers the pet up and earns a few points; a
// wrong one has a mild effect, except that letting a pet eat something
// toxic poisons it (treated with Heal). Each choice explains why. Closing
// the pop-up without choosing does nothing. Each happening comes at most
// once a week; some come only for one species, in one season, around a
// holiday, or when something is wrong (a dirty home, sore teeth).

const (
	// happeningGap is the least play time between happenings
	happeningGap = 20 * time.Minute

	// happeningChance is the chance a happening starts, checked once a
	// minute of play after the gap
	happeningChance = 0.1

	// happeningExpiry is how long a happening waits for a choice
	happeningExpiry = 10 * time.Minute

	// happeningRepeat is how many days pass before a happening can come
	// again
	happeningRepeat = 7

	// happeningPoints is the reward for a right choice, and
	// happeningPointsPerDay the most a calendar day
	happeningPoints       = 5
	happeningPointsPerDay = 25
)

// Errors choosing in a happening
var (
	ErrNoHappening = errors.New("nothing is happening now")
	ErrBadChoice   = errors.New("that isn't one of the choices")
)

// PendingHappening is a happening waiting for the player's choice
type PendingHappening struct {
	ID    string    `json:"id"`
	Since time.Time `json:"since"` // On the play clock
}

// Happening is a happening as the player sees it
type Happening struct {
	ID      string            `json:"id"`
	Icon    string            `json:"icon"`
	Title   string            `json:"title"`
	Text    string            `json:"text"`
	Choices []HappeningChoice `json:"choices"`
}

// HappeningChoice is one of a happening's choices
type HappeningChoice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// HappeningResult is what a choice led to
type HappeningResult struct {
	Right  bool   `json:"right"`
	Text   string `json:"text"`
	Lesson string `json:"lesson"`
	Points int    `json:"points"`
}

// happeningChoice is a choice and its outcome. Texts use {pet} for the
// pet's name.
type happeningChoice struct {
	id, label string
	right     bool
	text      string
	effect    func(p *DigitalPet)
}

// happeningDef is a happening. Texts use {pet} for the pet's name.
type happeningDef struct {
	id, icon, title, text, lesson string
	species                       Species              // "" for both
	seasons                       []environment.Season // Only in these seasons, if any
	holidays                      []string             // Only around these holidays, if any
	places                        []string             // For a trip event, the place it happens
	when                          func(p *DigitalPet) bool
	choices                       []happeningChoice
}

// Outcomes of choices. Each runs with p.mu held.

// calmed settles the pet
func calmed(p *DigitalPet) {
	p.Biology.Vitals.Stress -= 0.08
	p.Biology.Vitals.Happiness += 0.03
}

// scared frightens the pet
func scared(p *DigitalPet) {
	p.Biology.Vitals.Stress += 0.12
	p.Biology.Vitals.Happiness -= 0.05
}

// frustrated leaves the pet a little stressed
func frustrated(p *DigitalPet) {
	p.Biology.Vitals.Stress += 0.06
}

// upset gives the pet an upset stomach
func upset(p *DigitalPet) {
	v := p.Biology.Vitals
	v.Health -= 0.02
	v.Happiness -= 0.05
	v.Stress += 0.05
}

// poisoned poisons the pet
func poisoned(p *DigitalPet) { p.poisonLocked() }

// hurt hurts the pet a little
func hurt(p *DigitalPet) {
	p.Biology.Vitals.Health -= 0.05
	p.Biology.Vitals.Stress += 0.1
}

// played lets the pet play
func played(p *DigitalPet) {
	p.Biology.Vitals.Happiness += 0.06
	p.meetNeedsLocked(types.InteractionPlaying, 0.5)
}

// stimulated gives the pet something new to think about
func stimulated(p *DigitalPet) {
	p.ensureNeedsLocked()
	p.Needs.SatisfyNeed(types.NeedMentalStimulation, 0.3, p.Biology.GetAgeInDays())
}

// cleaned cleans the living area
func cleaned(p *DigitalPet) {
	p.LivingAreaWaste = 0
	p.Biology.Vitals.Stress -= 0.03
}

// groomed tidies the pet's coat
func groomed(p *DigitalPet) {
	p.Biology.Vitals.Cleanliness += 0.1
}

// Conditions for happenings. Each runs with p.mu held.

// homeDirty is true when the living area needs cleaning
func homeDirty(p *DigitalPet) bool { return p.LivingAreaWaste > 0.3 }

// soreTeeth is true when the pet's teeth need care
func soreTeeth(p *DigitalPet) bool {
	return p.Biology.Physiology != nil && p.Biology.Physiology.DentalHealth < 0.6
}

// longCoat is true for a pet with a long or shaggy coat
func longCoat(p *DigitalPet) bool {
	return p.Look != nil && slices.Contains([]string{CoatLong, CoatShaggy, CoatSemi}, p.Look.Coat)
}

// showsSigns is true when the pet shows a warning sign
func showsSigns(p *DigitalPet) bool {
	return slices.ContainsFunc(p.symptomsLocked(), func(s Symptom) bool { return !s.Good })
}

// happenings is every happening
var happenings = []happeningDef{
	// Both species
	{id: "thunderstorm", icon: "⛈️", title: "Thunderstorm",
		text:   "Thunder rumbles outside and {pet} is trembling.",
		lesson: "Comforting a frightened pet doesn't reward fear. Stay calm, let it hide somewhere safe, and never punish fear.",
		choices: []happeningChoice{
			{id: "comfort", label: "Comfort it calmly and let it hide", right: true, text: "{pet} curls up in its hiding place and settles.", effect: calmed},
			{id: "scold", label: "Tell it off for being silly", text: "Being scolded makes {pet} more frightened.", effect: scared},
			{id: "ignore", label: "Ignore it", text: "{pet} stays frightened until the storm passes.", effect: frustrated},
		}},
	{id: "zoomies", icon: "💨", title: "Zoomies!",
		text:   "{pet} suddenly races around the room at top speed.",
		lesson: "Zoomies are a normal burst of energy. Make sure the space is safe and let them run it off.",
		choices: []happeningChoice{
			{id: "let", label: "Clear the way and let it run", right: true, text: "{pet} zooms about, then flops down happily.", effect: played},
			{id: "stop", label: "Make it stop", text: "{pet} is confused and still full of energy.", effect: frustrated},
		}},
	{id: "sunbeam", icon: "☀️", title: "Sunbeam nap",
		text:   "{pet} has found a warm patch of sunlight and is dozing in it.",
		lesson: "Pets love warmth and rest a lot. Let a resting pet be; waking it often makes it grumpy.",
		choices: []happeningChoice{
			{id: "leave", label: "Leave it be", right: true, text: "{pet} enjoys a peaceful nap.", effect: calmed},
			{id: "wake", label: "Wake it up to play", text: "{pet} is grumpy at being woken.", effect: frustrated},
		}},
	{id: "smell", icon: "👃", title: "Something smells",
		text:   "There's a bad smell coming from {pet}'s area.",
		lesson: "A dirty living area causes stress and illness. Clean up after a pet every day.",
		when:   homeDirty,
		choices: []happeningChoice{
			{id: "clean", label: "Check the living area and clean it", right: true, text: "You clean up, and {pet} is much more comfortable.", effect: cleaned},
			{id: "spray", label: "Spray air freshener", text: "The smell is hidden, not gone, and strong scents bother a pet's sensitive nose.", effect: frustrated},
		}},
	{id: "sick", icon: "🤢", title: "Sick on the floor",
		text:   "{pet} has been sick on the floor.",
		lesson: "Being sick once, then acting normally, is usually minor. See a vet if it happens again, if the pet won't eat or drink, or if it seems weak or in pain.",
		choices: []happeningChoice{
			{id: "watch", label: "Clean up and keep an eye on it", right: true, text: "{pet} seems fine and is soon back to normal.", effect: cleaned},
			{id: "feed", label: "Give it a big meal straight away", text: "A big meal so soon upsets {pet}'s stomach again.", effect: upset},
		}},
	{id: "chocolate", icon: "🍫", title: "Chocolate left out",
		text:   "Someone left a bar of chocolate on the low table, right where {pet} can reach.",
		lesson: "Chocolate contains theobromine, which poisons dogs and cats; dark chocolate is the most dangerous. Keep it out of reach, and call a vet if a pet eats some.",
		choices: []happeningChoice{
			{id: "away", label: "Put it away", right: true, text: "The chocolate goes in a high cupboard.", effect: nil},
			{id: "leave", label: "Leave it there", text: "{pet} eats some of the chocolate and becomes ill.", effect: poisoned},
		}},
	{id: "plant", icon: "🪴", title: "Plant nibbling",
		text:   "{pet} is chewing the leaves of a houseplant.",
		lesson: "Many houseplants are toxic. Every part of a lily can kill a cat, and sago palm is deadly to dogs. Keep only pet-safe plants, or keep them out of reach.",
		choices: []happeningChoice{
			{id: "move", label: "Move the plant out of reach", right: true, text: "The plant goes up high, and {pet} loses interest.", effect: nil},
			{id: "leave", label: "Let it nibble", text: "The plant was toxic, and {pet} becomes ill.", effect: poisoned},
		}},
	{id: "bee", icon: "🐝", title: "A bee in the house",
		text:   "A bee is buzzing at the window, and {pet} is trying to catch it.",
		lesson: "Pets that catch bees and wasps get stung, often on the mouth or paws. Swelling of the face or trouble breathing needs a vet at once.",
		choices: []happeningChoice{
			{id: "window", label: "Keep {pet} back and open the window", right: true, text: "The bee flies out, and {pet} is safe.", effect: nil},
			{id: "watch", label: "Let {pet} chase it", text: "{pet} is stung on the nose. Ouch!", effect: hurt},
		}},
	{id: "wont_eat", icon: "🍽️", title: "Not eating",
		text:   "{pet} sniffs its food and walks away.",
		lesson: "A pet that stops eating may have sore teeth or gums. Check its mouth, brush its teeth, and see a vet if it doesn't eat for a day.",
		when:   soreTeeth,
		choices: []happeningChoice{
			{id: "teeth", label: "Check its teeth and gums", right: true, text: "Its gums are red and sore: time for teeth care and perhaps a vet.", effect: nil},
			{id: "treats", label: "Offer treats instead", text: "Treats hide the problem, and sore teeth get worse.", effect: upset},
		}},
	{id: "tangle", icon: "🪮", title: "A tangle in the coat",
		text:   "{pet}'s long coat has a tangled mat behind one ear.",
		lesson: "Mats pull painfully on the skin. Brush them out gently, a little at a time; never cut them with scissors, because skin is easily cut.",
		when:   longCoat,
		choices: []happeningChoice{
			{id: "brush", label: "Brush it out gently", right: true, text: "Little by little, the tangle comes free.", effect: groomed},
			{id: "scissors", label: "Cut it out with scissors", text: "The scissors nick {pet}'s skin.", effect: hurt},
		}},
	{id: "doorbell", icon: "🔔", title: "The doorbell",
		text:   "The doorbell rings and {pet} rushes to the door.",
		lesson: "Shouting sounds like joining in. Reward a pet for staying calm instead.",
		choices: []happeningChoice{
			{id: "reward", label: "Calmly reward it for being quiet", right: true, text: "{pet} settles and gets a treat for it.", effect: calmed},
			{id: "shout", label: "Shout at it", text: "{pet} gets more worked up.", effect: frustrated},
		}},
	{id: "hiding", icon: "🫣", title: "Hiding more than usual",
		text:   "{pet} has been hiding under the bed a lot today.",
		lesson: "Hiding is often the first sign that a pet feels unwell. Check it over and watch for other signs.",
		when:   showsSigns,
		choices: []happeningChoice{
			{id: "check", label: "Check it for warning signs", right: true, text: "You find signs to look into; the Body panel lists them.", effect: nil},
			{id: "pull", label: "Pull it out to play", text: "{pet} is frightened and hides again.", effect: scared},
		}},
	{id: "new_toy", icon: "🎁", title: "A new toy",
		text:   "A new toy has arrived for {pet}.",
		lesson: "New things keep a pet's mind busy. Rotating toys keeps old ones interesting too.",
		choices: []happeningChoice{
			{id: "introduce", label: "Introduce it with a game", right: true, text: "{pet} loves its new toy.", effect: stimulated},
			{id: "box", label: "Leave it in the box", text: "{pet} sniffs the box and wanders off.", effect: nil},
		}},
	{id: "nails", icon: "💅", title: "Nails need a trim",
		text:   "{pet}'s nails are clicking on the floor.",
		lesson: "Long nails are uncomfortable and can grow into the paw. Trim only the tips, avoiding the pink quick, which bleeds and hurts.",
		choices: []happeningChoice{
			{id: "trim", label: "Trim just the tips", right: true, text: "A careful trim, and {pet} walks comfortably.", effect: nil},
			{id: "short", label: "Cut them nice and short", text: "You cut into the quick, and it bleeds.", effect: hurt},
		}},
	{id: "visitor_pet", icon: "🐾", title: "A visitor's pet",
		text:   "A friend has brought their pet to visit.",
		lesson: "Introduce pets slowly, on neutral ground, with time apart if either is nervous.",
		choices: []happeningChoice{
			{id: "slow", label: "Introduce them slowly", right: true, text: "The two sniff each other politely.", effect: calmed},
			{id: "fast", label: "Put them together straight away", text: "{pet} is startled and hisses or growls.", effect: scared},
		}},
	{id: "gift", icon: "🍂", title: "A gift for you",
		text:   "{pet} proudly brings you a leaf and drops it at your feet.",
		lesson: "Bringing things to people is natural behavior. Praise it, and swap anything unsafe for a treat.",
		choices: []happeningChoice{
			{id: "praise", label: "Praise it", right: true, text: "{pet} is delighted.", effect: calmed},
			{id: "scold", label: "Tell it off for the mess", text: "{pet} doesn't understand why.", effect: frustrated},
		}},

	// Cats
	{id: "glass", icon: "🥛", title: "Knocking things off", species: SpeciesCat,
		text:   "{pet} looks at you, then pushes a glass toward the edge of the table.",
		lesson: "Cats knock things over to explore and for attention. Put breakables away and give it attention through play.",
		choices: []happeningChoice{
			{id: "play", label: "Move the glass and play with it", right: true, text: "{pet} pounces on a toy instead.", effect: played},
			{id: "shout", label: "Shout", text: "{pet} knocks the glass off anyway and runs.", effect: frustrated},
		}},
	{id: "curtains", icon: "🪟", title: "Climbing the curtains", species: SpeciesCat,
		text:   "{pet} is halfway up the curtains.",
		lesson: "Cats need to climb. A cat tree or shelves give it a safe place to do it.",
		choices: []happeningChoice{
			{id: "tree", label: "Lift it down onto a cat tree", right: true, text: "{pet} climbs the cat tree happily.", effect: played},
			{id: "spray", label: "Spray it with water", text: "{pet} is frightened and wary of you.", effect: scared},
		}},
	{id: "sofa", icon: "🛋️", title: "Scratching the sofa", species: SpeciesCat,
		text:   "{pet} is scratching the side of the sofa.",
		lesson: "Scratching keeps claws healthy and marks territory. Redirect it to a sturdy post; never declaw.",
		choices: []happeningChoice{
			{id: "post", label: "Redirect it to a scratching post", right: true, text: "{pet} has a good scratch on the post.", effect: calmed},
			{id: "shout", label: "Shout at it", text: "{pet} runs off and will scratch the sofa later.", effect: frustrated},
		}},
	{id: "hairball", icon: "🧶", title: "A hairball", species: SpeciesCat,
		text:   "{pet} coughs up a hairball.",
		lesson: "Brushing removes loose fur before it is swallowed. Frequent hairballs, or retching with nothing coming up, need a vet.",
		choices: []happeningChoice{
			{id: "brush", label: "Clean up and brush it more often", right: true, text: "Regular brushing means fewer hairballs.", effect: groomed},
			{id: "ignore", label: "Ignore it", text: "More fur gets swallowed.", effect: upset},
		}},
	{id: "birds", icon: "🐦", title: "Chattering at birds", species: SpeciesCat,
		text:   "{pet} chatters at the birds outside the window.",
		lesson: "Watching birds is great entertainment. A window perch gives a cat a safe view.",
		choices: []happeningChoice{
			{id: "perch", label: "Set up a window perch", right: true, text: "{pet} settles in to watch.", effect: stimulated},
			{id: "curtain", label: "Close the curtains", text: "{pet} is bored.", effect: frustrated},
		}},
	{id: "night_sprint", icon: "🌙", title: "A 3 a.m. sprint", species: SpeciesCat,
		text:   "{pet} thunders through the house in the middle of the night.",
		lesson: "Cats are most active at dawn and dusk. Play before bedtime, then a meal, helps them sleep through.",
		choices: []happeningChoice{
			{id: "play", label: "Play with it before bed from now on", right: true, text: "{pet} sleeps better after an evening game.", effect: played},
			{id: "shut", label: "Shut it out of the bedroom and shout", text: "{pet} cries at the door.", effect: frustrated},
		}},
	{id: "box", icon: "📦", title: "A box arrives", species: SpeciesCat,
		text:   "A delivery box sits empty on the floor, and {pet} is staring at it.",
		lesson: "Boxes feel safe to cats: enclosed spaces reduce stress.",
		choices: []happeningChoice{
			{id: "explore", label: "Let it explore the box", right: true, text: "{pet} climbs in and looks very pleased.", effect: calmed},
			{id: "bin", label: "Throw the box away", text: "{pet} looks disappointed.", effect: nil},
		}},
	{id: "string", icon: "🧵", title: "A piece of thread", species: SpeciesCat,
		text:   "{pet} is batting at a long piece of thread on the floor.",
		lesson: "Swallowed string, thread or tinsel can tangle in the gut and needs surgery. Put it away, and never leave a cat playing with string alone.",
		choices: []happeningChoice{
			{id: "away", label: "Take it away and offer a toy", right: true, text: "{pet} chases a toy instead.", effect: played},
			{id: "leave", label: "Let it play", text: "{pet} swallows some thread and feels unwell.", effect: upset},
		}},
	{id: "litter", icon: "🚽", title: "Avoiding the litter box", species: SpeciesCat,
		text:   "{pet} is going beside the litter box instead of in it.",
		lesson: "Cats avoid dirty boxes, which can lead to urinary problems. Keep the box clean, and see a vet for straining or blood in the urine.",
		when:   homeDirty,
		choices: []happeningChoice{
			{id: "clean", label: "Clean the box and watch for urinary signs", right: true, text: "{pet} uses the clean box.", effect: cleaned},
			{id: "scold", label: "Rub its nose in it", text: "Punishment teaches {pet} to hide, not to use the box.", effect: scared},
		}},
	{id: "kneading", icon: "🐾", title: "Kneading", species: SpeciesCat,
		text:   "{pet} is purring and kneading your lap with its paws.",
		lesson: "Kneading is a sign of contentment, left over from kittenhood.",
		choices: []happeningChoice{
			{id: "enjoy", label: "Enjoy it", right: true, text: "{pet} purrs louder.", effect: calmed},
			{id: "push", label: "Push it off", text: "{pet} walks away.", effect: frustrated},
		}},

	// Dogs
	{id: "shoe", icon: "👟", title: "Chewing a shoe", species: SpeciesDog,
		text:   "{pet} is chewing one of your shoes.",
		lesson: "Dogs need to chew. Swap the shoe for a chew toy and praise it for chewing that.",
		choices: []happeningChoice{
			{id: "swap", label: "Swap it for a chew toy", right: true, text: "{pet} happily chews the toy.", effect: played},
			{id: "chase", label: "Chase it to get the shoe back", text: "{pet} thinks it's a game and runs off with it.", effect: frustrated},
		}},
	{id: "digging", icon: "🕳️", title: "Digging", species: SpeciesDog,
		text:   "{pet} is digging holes in the garden.",
		lesson: "Digging is natural, often from boredom. A digging spot, more exercise and play help.",
		choices: []happeningChoice{
			{id: "spot", label: "Give it its own digging spot", right: true, text: "{pet} digs happily in its spot.", effect: played},
			{id: "scold", label: "Scold it", text: "{pet} digs when you're not looking.", effect: frustrated},
		}},
	{id: "barking", icon: "🗣️", title: "Barking at the window", species: SpeciesDog,
		text:   "{pet} barks at everyone passing the window.",
		lesson: "Block the view and reward quiet. Yelling sounds like barking along.",
		choices: []happeningChoice{
			{id: "reward", label: "Call it away and reward quiet", right: true, text: "{pet} trots over for its treat.", effect: calmed},
			{id: "yell", label: "Yell at it to stop", text: "{pet} barks louder.", effect: frustrated},
		}},
	{id: "jumping", icon: "🦘", title: "Jumping up at visitors", species: SpeciesDog,
		text:   "{pet} jumps up at a visitor.",
		lesson: "Turn away from jumping and reward four paws on the floor. Pushing a dog away can feel like play.",
		choices: []happeningChoice{
			{id: "paws", label: "Reward four paws on the floor", right: true, text: "{pet} learns that sitting gets attention.", effect: calmed},
			{id: "push", label: "Push it down", text: "{pet} thinks it's a game and jumps more.", effect: frustrated},
		}},
	{id: "counter", icon: "🍗", title: "Counter-surfing", species: SpeciesDog,
		text:   "{pet} has its paws on the kitchen counter, sniffing at food.",
		lesson: "Keep counters clear: cooked bones splinter, and onions, garlic, grapes and xylitol are toxic.",
		choices: []happeningChoice{
			{id: "clear", label: "Clear the counter and teach 'off'", right: true, text: "Nothing to find, and {pet} soon stops looking.", effect: nil},
			{id: "leave", label: "Leave the leftovers there", text: "{pet} steals greasy leftovers and gets an upset stomach.", effect: upset},
		}},
	{id: "grass", icon: "🌱", title: "Eating grass", species: SpeciesDog,
		text:   "{pet} is nibbling grass in the garden.",
		lesson: "Many dogs eat grass and it's usually harmless, as long as the grass hasn't been sprayed with chemicals.",
		choices: []happeningChoice{
			{id: "let", label: "Let it, since the lawn isn't sprayed", right: true, text: "{pet} has a nibble and moves on.", effect: calmed},
			{id: "panic", label: "Rush it to the vet", text: "The trip stresses {pet}, and the vet says it was fine.", effect: frustrated},
		}},
	{id: "mud", icon: "🐾", title: "Muddy paw prints", species: SpeciesDog,
		text:   "{pet} has left a trail of muddy paw prints across the floor.",
		lesson: "Wipe paws after walks; it also removes grit, salt and anything harmful.",
		choices: []happeningChoice{
			{id: "wipe", label: "Wipe its paws and the floor", right: true, text: "Clean paws and a clean floor.", effect: groomed},
			{id: "shout", label: "Shout at it", text: "{pet} doesn't understand what it did wrong.", effect: frustrated},
		}},
	{id: "accident", icon: "💧", title: "An accident inside", species: SpeciesDog,
		text:   "{pet} has had an accident on the floor.",
		lesson: "Never scold accidents; it teaches a dog to hide. Clean with an enzyme cleaner and take it out more often.",
		choices: []happeningChoice{
			{id: "clean", label: "Clean up and take it out more often", right: true, text: "{pet} goes outside next time.", effect: cleaned},
			{id: "scold", label: "Scold it", text: "{pet} learns to hide when it needs to go.", effect: scared},
		}},
	{id: "socks", icon: "🧦", title: "Stolen socks", species: SpeciesDog,
		text:   "{pet} has run off with a sock.",
		lesson: "Socks can block the gut if swallowed. Trade for a treat instead of chasing, which turns it into a game.",
		choices: []happeningChoice{
			{id: "trade", label: "Trade it for a treat", right: true, text: "{pet} drops the sock for the treat.", effect: calmed},
			{id: "chase", label: "Chase it", text: "{pet} gulps the sock down to keep it, and feels sick.", effect: upset},
		}},
	{id: "whining", icon: "🥺", title: "Whining when you leave", species: SpeciesDog,
		text:   "{pet} whines and scratches at the door when you go out.",
		lesson: "Practice short absences, leave calmly, and give a long-lasting chew. Making a fuss when you leave makes it worse.",
		choices: []happeningChoice{
			{id: "practice", label: "Practice short, calm absences", right: true, text: "{pet} learns you always come back.", effect: calmed},
			{id: "fuss", label: "Make a big fuss every time you leave", text: "{pet} becomes more anxious.", effect: scared},
		}},

	// Seasons and holidays
	{id: "fireworks", icon: "🎆", title: "Fireworks",
		holidays: []string{"new_year", "lunar_new_year", "diwali"},
		text:     "Fireworks are going off outside, and {pet} is shaking.",
		lesson:   "Keep pets inside during fireworks, with curtains closed, some background noise and a den to hide in.",
		choices: []happeningChoice{
			{id: "den", label: "Close the curtains and make a den", right: true, text: "{pet} hides in its den and calms down.", effect: calmed},
			{id: "outside", label: "Take it outside to watch", text: "{pet} is terrified.", effect: scared},
		}},
	{id: "festive_tree", icon: "🎄", title: "The festive tree",
		holidays: []string{"christmas"},
		text:     "{pet} is very interested in the shiny decorations on the tree.",
		lesson:   "Tinsel can block a pet's gut, glass ornaments break into sharp pieces, and tree water may hold chemicals.",
		choices: []happeningChoice{
			{id: "high", label: "Hang decorations high and skip the tinsel", right: true, text: "{pet} admires the tree from below.", effect: nil},
			{id: "low", label: "Leave tinsel on the low branches", text: "{pet} swallows some tinsel and feels unwell.", effect: upset},
		}},
	{id: "holiday_sweets", icon: "🍬", title: "Holiday sweets",
		holidays: []string{"easter", "halloween", "valentines"},
		text:     "A bowl of chocolate and sweets sits on the coffee table.",
		lesson:   "Chocolate poisons pets, and sweets with xylitol can be deadly to dogs. Keep them out of reach.",
		choices: []happeningChoice{
			{id: "away", label: "Put the bowl away", right: true, text: "No sweets for {pet}, safely.", effect: nil},
			{id: "leave", label: "Leave the bowl out", text: "{pet} eats some chocolate and becomes ill.", effect: poisoned},
		}},
	{id: "candles", icon: "🕯️", title: "Candles and lanterns",
		holidays: []string{"hanukkah", "diwali", "kwanzaa", "vesak"},
		text:     "Candles are lit, and {pet}'s tail swishes close to a flame.",
		lesson:   "Keep candles and lamps out of reach of paws and tails, or use electric lights.",
		choices: []happeningChoice{
			{id: "move", label: "Move the candles out of reach", right: true, text: "The flames are safely out of the way.", effect: nil},
			{id: "leave", label: "Leave them", text: "{pet}'s fur is singed.", effect: hurt},
		}},
	{id: "hot_pavement", icon: "🔥", title: "Hot pavement", species: SpeciesDog,
		seasons: []environment.Season{environment.SeasonSummer},
		text:    "It's a hot afternoon, and {pet} wants a walk.",
		lesson:  "If the pavement is too hot for the back of your hand for seven seconds, it's too hot for paws. Walk early or late, on grass.",
		choices: []happeningChoice{
			{id: "test", label: "Test the pavement and walk on grass later", right: true, text: "{pet}'s paws stay safe.", effect: nil},
			{id: "go", label: "Go now on the pavement", text: "{pet}'s paw pads are sore.", effect: hurt},
		}},
	{id: "salt", icon: "🧂", title: "Salt on the paws", species: SpeciesDog,
		seasons: []environment.Season{environment.SeasonWinter},
		text:    "{pet} comes in from an icy walk and starts licking its paws.",
		lesson:  "De-icing salt stings paws and upsets the stomach when licked off. Wipe paws after winter walks.",
		choices: []happeningChoice{
			{id: "wipe", label: "Wipe its paws", right: true, text: "Clean, comfortable paws.", effect: groomed},
			{id: "leave", label: "Let it lick them clean", text: "The salt upsets {pet}'s stomach.", effect: upset},
		}},
	{id: "pollen", icon: "🌼", title: "Pollen season",
		seasons: []environment.Season{environment.SeasonSpring},
		text:    "{pet} keeps scratching and licking its paws after time outside.",
		lesson:  "Pets get pollen allergies too, often as itchy skin and paws. Wipe pollen off, and see a vet for treatment.",
		choices: []happeningChoice{
			{id: "wipe", label: "Wipe the pollen off its coat and paws", right: true, text: "{pet} is less itchy.", effect: groomed},
			{id: "ignore", label: "Ignore the scratching", text: "The itching gets worse.", effect: frustrated},
		}},
	{id: "leaf_mold", icon: "🍂", title: "Moldy leaves",
		seasons: []environment.Season{environment.SeasonFall},
		text:    "{pet} is nosing around in a damp, moldy pile of fallen leaves.",
		lesson:  "Mold on leaves and compost can make toxins that cause tremors. Keep pets away from moldy piles.",
		choices: []happeningChoice{
			{id: "away", label: "Call it away from the pile", right: true, text: "{pet} comes away and plays elsewhere.", effect: nil},
			{id: "let", label: "Let it rummage", text: "{pet} eats something moldy and becomes ill.", effect: poisoned},
		}},
}

// findHappening returns a happening or trip event
func findHappening(id string) (happeningDef, bool) {
	for _, list := range [][]happeningDef{happenings, tripEvents} {
		if i := slices.IndexFunc(list, func(h happeningDef) bool { return h.id == id }); i >= 0 {
			return list[i], true
		}
	}
	return happeningDef{}, false
}

// markHappeningLocked records that a happening came today. Caller must
// hold p.mu.
func (p *DigitalPet) markHappeningLocked(id string) {
	if p.Marks == nil {
		p.Marks = make(map[string]string)
	}
	p.Marks["happening:"+id] = today()
}

// view returns the happening as the player sees it
func (h happeningDef) view(name string) *Happening {
	v := &Happening{ID: h.id, Icon: h.icon, Title: h.title, Text: petText(h.text, name)}
	// The choices are listed starting from one set by the happening's ID,
	// so the right one isn't always first
	f := fnv.New32a()
	f.Write([]byte(h.id))
	start := int(f.Sum32() % uint32(len(h.choices)))
	for i := range h.choices {
		c := h.choices[(start+i)%len(h.choices)]
		v.Choices = append(v.Choices, HappeningChoice{ID: c.id, Label: petText(c.label, name)})
	}
	return v
}

// petText puts the pet's name into a happening's text
func petText(text, name string) string { return strings.ReplaceAll(text, "{pet}", name) }

// happeningFitsLocked reports whether a happening can come now. Caller
// must hold p.mu.
func (p *DigitalPet) happeningFitsLocked(h happeningDef) bool {
	if h.species != "" && h.species != p.speciesLocked() {
		return false
	}
	if seen := p.Marks["happening:"+h.id]; seen != "" {
		if day, err := time.Parse("2006-01-02", seen); err == nil && now().Sub(day) < happeningRepeat*24*time.Hour {
			return false
		}
	}
	if len(h.seasons) > 0 {
		p.ensureEnvironmentLocked()
		if !slices.Contains(h.seasons, p.Environment.GetSeason()) {
			return false
		}
	}
	if len(h.holidays) > 0 && !slices.ContainsFunc(h.holidays, HolidayOpen) {
		return false
	}
	return h.when == nil || h.when(p)
}

// happeningReadyLocked reports whether the pet can have a happening now:
// home, awake and well. Caller must hold p.mu.
func (p *DigitalPet) happeningReadyLocked() bool {
	return p.Biology.IsAlive && !p.Hibernating && !p.isExploringLocked() && p.Illness == nil && !p.Injured &&
		p.CurrentBehavior != types.BehaviorSleeping
}

// CheckHappening may start a happening, and returns it if it did. The
// game server calls it while the player is present.
func (p *DigitalPet) CheckHappening() *Happening {
	p.mu.Lock()
	defer p.mu.Unlock()
	t := p.playNowLocked()
	if p.Happening != nil {
		if t.Sub(p.Happening.Since) > happeningExpiry {
			p.Happening = nil // It passed without a choice
		}
		return nil
	}
	if !p.happeningReadyLocked() || t.Sub(p.LastHappening) < happeningGap || t.Sub(p.happeningChecked) < time.Minute {
		return nil
	}
	p.happeningChecked = t
	if !chance(happeningChance) {
		return nil
	}
	var fits []happeningDef
	for _, h := range happenings {
		if p.happeningFitsLocked(h) {
			fits = append(fits, h)
		}
	}
	if len(fits) == 0 {
		return nil
	}
	h := fits[randIntn(len(fits))]
	p.Happening = &PendingHappening{ID: h.id, Since: t}
	p.LastHappening = t
	p.markHappeningLocked(h.id)
	return h.view(p.Name)
}

// CurrentHappening returns the happening waiting for a choice, or nil
func (p *DigitalPet) CurrentHappening() *Happening {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Happening == nil {
		return nil
	}
	h, ok := findHappening(p.Happening.ID)
	if !ok {
		return nil
	}
	return h.view(p.Name)
}

// DismissHappening closes the happening without a choice; nothing happens
func (p *DigitalPet) DismissHappening() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Happening = nil
}

// ChooseHappening makes a choice in the happening waiting for one
func (p *DigitalPet) ChooseHappening(choiceID string) (HappeningResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Happening == nil {
		return HappeningResult{}, ErrNoHappening
	}
	h, ok := findHappening(p.Happening.ID)
	if !ok {
		p.Happening = nil
		return HappeningResult{}, ErrNoHappening
	}
	i := slices.IndexFunc(h.choices, func(c happeningChoice) bool { return c.id == choiceID })
	if i < 0 {
		return HappeningResult{}, ErrBadChoice
	}
	c := h.choices[i]
	p.Happening = nil
	if c.effect != nil {
		c.effect(p)
	}
	result := HappeningResult{Right: c.right, Text: petText(c.text, p.Name), Lesson: petText(h.lesson, p.Name)}
	if c.right {
		v := p.Biology.Vitals
		v.Happiness += 0.05
		v.Stress -= 0.03
		result.Points = p.happeningPointsLocked()
		p.FriendshipPoints += result.Points
		p.PointsEarned += result.Points
		p.countLocked("good_calls", 1)
		p.addHistoryLocked("Happening", h.title+": "+c.label, "good call")
	} else {
		p.addHistoryLocked("Happening", h.title+": "+c.label, "learned a lesson")
	}
	p.Biology.Vitals.Clamp()
	return result, nil
}

// happeningPointsLocked limits points from happenings to
// happeningPointsPerDay a calendar day. Caller must hold p.mu.
func (p *DigitalPet) happeningPointsLocked() int {
	if p.Marks == nil {
		p.Marks = make(map[string]string)
	}
	if p.Marks["happening_points_day"] != today() {
		p.Marks["happening_points_day"] = today()
		delete(p.Counters, "happening_points")
	}
	points := min(happeningPoints, happeningPointsPerDay-int(p.Counters["happening_points"]))
	p.countLocked("happening_points", float64(points))
	return points
}

func init() {
	achievements = append(achievements, struct {
		Achievement
		earned func(p *DigitalPet) bool
	}{Achievement{ID: "happy_helper", Name: "Happy Helper", Icon: "🏠", Rarity: "rare", Points: 25,
		Description: "Handled 25 home happenings well"},
		func(p *DigitalPet) bool { return p.Counters["good_calls"] >= 25 }})
}
