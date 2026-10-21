package core

// Trip events: one trip in three ends with a choice about something that
// happened there, shown when the pet comes home. They use the happenings'
// choices and rewards (happenings.go). A good choice brings a small bonus,
// such as a calmer pet or an extra find; a poor one a mild cost. Each
// teaches something true about keeping a pet safe outdoors.

// tripEventChance is the chance a finished trip brings a choice
const tripEventChance = 1.0 / 3

// muddy dirties the pet's coat
func muddy(p *DigitalPet) {
	p.Biology.Vitals.Cleanliness -= 0.2
}

// tired wears the pet out
func tired(p *DigitalPet) {
	p.Biology.Vitals.Energy -= 0.1
	p.Biology.Vitals.Stress += 0.05
}

// extraFind lets the pet find another souvenir at the trip's place
func extraFind(p *DigitalPet) {
	place, _ := p.tripPlaceLocked()
	s := p.pickSouvenirLocked(place)
	p.Souvenirs = append(p.Souvenirs, s.ID)
	if a := p.LastAdventure; a != nil {
		a.Extra = append(a.Extra, s.ID)
	}
}

// tripEvents are the events for each place
var tripEvents = []happeningDef{
	{id: "trip_park_dog", icon: "🐕", title: "An off-lead dog", places: []string{"park"},
		text:   "While at the park, a big dog off its lead bounds over to {pet}.",
		lesson: "Not every dog is friendly. Stay calm, step between your pet and the dog, and move away; never let an unknown dog rush up to a cat.",
		choices: []happeningChoice{
			{id: "step", label: "Calmly step between them and walk away", right: true, text: "The dog loses interest, and {pet} relaxes.", effect: calmed},
			{id: "wait", label: "Let them sort it out", text: "The dog is too rough, and {pet} is frightened.", effect: scared},
		}},
	{id: "trip_park_food", icon: "🥪", title: "A dropped sandwich", places: []string{"park"},
		text:   "While at the park, {pet} finds a half-eaten sandwich under a bench.",
		lesson: "Food left in parks can hold onions, cooked bones or mold. Teach 'leave it' and reward it.",
		choices: []happeningChoice{
			{id: "leave", label: "Say 'leave it' and reward it", right: true, text: "{pet} leaves it and gets a treat, then sniffs out something better.", effect: extraFind},
			{id: "eat", label: "Let it have it", text: "The sandwich had onion in it, and {pet}'s stomach is upset.", effect: upset},
		}},

	{id: "trip_beach_seawater", icon: "🌊", title: "Drinking seawater", places: []string{"beach"},
		text:   "While at the beach, {pet} is lapping at the seawater.",
		lesson: "Seawater makes pets sick and too much is dangerous. Bring fresh water and a bowl to the beach.",
		choices: []happeningChoice{
			{id: "fresh", label: "Give it fresh water from your bottle", right: true, text: "{pet} drinks the fresh water gratefully.", effect: calmed},
			{id: "let", label: "Let it drink", text: "The salty water upsets {pet}'s stomach.", effect: upset},
		}},
	{id: "trip_beach_sun", icon: "☀️", title: "Hot sand", places: []string{"beach"},
		text:   "While at the beach, the sand gets very hot and {pet} is panting.",
		lesson: "Hot sand burns paws, and pets overheat quickly. Find shade, offer water and go in the cool of the day.",
		choices: []happeningChoice{
			{id: "shade", label: "Rest in the shade with water", right: true, text: "{pet} cools down in the shade.", effect: calmed},
			{id: "keep", label: "Keep playing in the sun", text: "{pet} gets too hot and tired.", effect: tired},
		}},
	{id: "trip_beach_fish", icon: "🐟", title: "A smelly fish", places: []string{"beach"},
		text:   "While at the beach, {pet} has found a dead fish and wants to roll in it.",
		lesson: "Dead animals carry germs and parasites, and fish bones choke. Lead a pet away from them.",
		choices: []happeningChoice{
			{id: "away", label: "Lead it away down the beach", right: true, text: "Further along, {pet} finds something nicer.", effect: extraFind},
			{id: "let", label: "Let it", text: "{pet} rolls in the fish and needs a bath.", effect: muddy},
		}},

	{id: "trip_meadow_bee", icon: "🐝", title: "Chasing bees", places: []string{"meadow"},
		text:   "While in the meadow, {pet} snaps at a bee and yelps.",
		lesson: "A sting usually swells a little and settles. Swelling of the face or throat, or trouble breathing, needs a vet at once.",
		choices: []happeningChoice{
			{id: "check", label: "Check it and watch for swelling", right: true, text: "Just a little swelling, and {pet} is soon fine.", effect: calmed},
			{id: "keep", label: "Keep playing", text: "{pet} gets stung again.", effect: hurt},
		}},
	{id: "trip_meadow_ticks", icon: "🕷️", title: "Long grass", places: []string{"meadow"},
		text:   "While in the meadow, {pet} has been running through the long grass.",
		lesson: "Ticks wait in long grass and can spread disease. Check your pet's coat after walks, and use tick prevention from your vet.",
		choices: []happeningChoice{
			{id: "check", label: "Check its coat for ticks", right: true, text: "You find a tick and remove it safely.", effect: groomed},
			{id: "skip", label: "Skip the check", text: "A tick stays on {pet} and makes it itch.", effect: frustrated},
		}},

	{id: "trip_farm_sheep", icon: "🐑", title: "Sheep in the field", places: []string{"farm"},
		text:   "While at the farm, {pet} spots a field of sheep.",
		lesson: "Chasing livestock frightens and harms them, and a kick can hurt a pet badly. Keep pets close around farm animals.",
		choices: []happeningChoice{
			{id: "close", label: "Keep it close and walk calmly past", right: true, text: "{pet} watches the sheep with interest.", effect: stimulated},
			{id: "chase", label: "Let it run up to them", text: "A ewe stamps at {pet}, who runs back frightened.", effect: scared},
		}},
	{id: "trip_farm_puddle", icon: "💧", title: "A farmyard puddle", places: []string{"farm"},
		text:   "While at the farm, {pet} wants to drink from a muddy puddle.",
		lesson: "Puddles on farms can hold animal waste, fertilizer and bacteria such as leptospirosis. Bring clean water.",
		choices: []happeningChoice{
			{id: "fresh", label: "Give it clean water instead", right: true, text: "{pet} drinks the clean water.", effect: calmed},
			{id: "let", label: "Let it drink", text: "The dirty water upsets {pet}'s stomach.", effect: upset},
		}},
	{id: "trip_farm_manure", icon: "💩", title: "Something smelly", places: []string{"farm"},
		text:   "While at the farm, {pet} wants to roll in a manure heap.",
		lesson: "Rolling in smells is natural, but manure can carry parasites. Lead a pet away, and wash it if it rolls.",
		choices: []happeningChoice{
			{id: "away", label: "Lead it away", right: true, text: "{pet} trots on and finds something interesting.", effect: extraFind},
			{id: "let", label: "Let it roll", text: "{pet} is filthy and smells awful.", effect: muddy},
		}},

	{id: "trip_garden_harness", icon: "🦺", title: "A loose harness", places: []string{"garden"},
		text:   "While in the garden, you notice {pet}'s harness has slipped loose.",
		lesson: "A harness should fit snugly, with room for two fingers underneath. A frightened cat can back out of a loose one and bolt.",
		choices: []happeningChoice{
			{id: "fit", label: "Stop and adjust the fit", right: true, text: "Snug and safe, {pet} explores happily.", effect: calmed},
			{id: "keep", label: "Carry on", text: "{pet} wriggles half out, panics and has to be scooped up.", effect: scared},
		}},
	{id: "trip_garden_lily", icon: "🌺", title: "Lilies in the flowerbed", places: []string{"garden"},
		text:   "While in the garden, {pet} heads for a bed of lilies.",
		lesson: "Every part of a lily, even the pollen, can cause fatal kidney failure in cats. If a cat touches one, wash it off and call a vet at once.",
		choices: []happeningChoice{
			{id: "steer", label: "Steer it away from the lilies", right: true, text: "{pet} explores the herbs instead, and finds something.", effect: extraFind},
			{id: "let", label: "Let it sniff them", text: "Pollen gets on {pet}'s fur; you wash it off at once, and it feels unwell.", effect: upset},
		}},

	{id: "trip_dogpark_growl", icon: "😠", title: "A stiff, growling dog", places: []string{"dog_park"},
		text:   "While at the dog park, another dog stiffens and growls at {pet}.",
		lesson: "A stiff body, hard stare or growl means 'back off'. Call your dog away calmly before it becomes a fight.",
		choices: []happeningChoice{
			{id: "call", label: "Call {pet} away calmly", right: true, text: "{pet} comes away, and both dogs relax.", effect: calmed},
			{id: "wait", label: "Let them work it out", text: "There's a scuffle, and {pet} is frightened.", effect: scared},
		}},
	{id: "trip_dogpark_bowl", icon: "🥣", title: "The shared water bowl", places: []string{"dog_park"},
		text:   "While at the dog park, {pet} heads for the shared water bowl.",
		lesson: "Shared bowls can spread kennel cough and stomach bugs. Bring your dog's own water.",
		choices: []happeningChoice{
			{id: "own", label: "Give it water from its own bottle", right: true, text: "{pet} drinks its own fresh water.", effect: calmed},
			{id: "shared", label: "Let it drink from the shared bowl", text: "{pet} picks up a stomach bug.", effect: upset},
		}},
	{id: "trip_dogpark_crowd", icon: "🐕‍🦺", title: "Too many dogs", places: []string{"dog_park"},
		text:   "While at the dog park, a crowd of dogs mobs {pet}, who tucks its tail.",
		lesson: "A tucked tail, lip licking and trying to hide mean a dog is overwhelmed. Give it a break somewhere quieter.",
		choices: []happeningChoice{
			{id: "break", label: "Take a break outside the gate", right: true, text: "{pet} calms down and plays again later.", effect: calmed},
			{id: "stay", label: "Make it stay and get used to it", text: "{pet} becomes more frightened of other dogs.", effect: scared},
		}},

	{id: "trip_forest_mushroom", icon: "🍄", title: "A mushroom", places: []string{"forest"},
		text:   "While in the forest, {pet} sniffs a mushroom growing by a log.",
		lesson: "Some wild mushrooms are deadly to pets, and they are hard to tell apart. Lead pets away from all of them.",
		choices: []happeningChoice{
			{id: "away", label: "Lead it away", right: true, text: "{pet} follows, and finds something along the path.", effect: extraFind},
			{id: "let", label: "Let it sniff around", text: "{pet} takes a bite, and its stomach is upset; call a vet if a pet eats a wild mushroom.", effect: upset},
		}},
	{id: "trip_forest_deer", icon: "🦌", title: "A deer", places: []string{"forest"},
		text:   "While in the forest, a deer leaps across the path and {pet} wants to chase it.",
		lesson: "Chasing wildlife can get a pet lost or hurt. Keep it close where wild animals live.",
		choices: []happeningChoice{
			{id: "hold", label: "Hold it close and let the deer go", right: true, text: "{pet} watches the deer bound away.", effect: stimulated},
			{id: "chase", label: "Let it chase", text: "{pet} runs into brambles and is scratched.", effect: hurt},
		}},
	{id: "trip_forest_berries", icon: "🫐", title: "Wild berries", places: []string{"forest"},
		text:   "While in the forest, {pet} noses at a bush of wild berries.",
		lesson: "Many wild berries, such as yew, holly and nightshade, are poisonous to pets. Don't let pets eat them.",
		choices: []happeningChoice{
			{id: "away", label: "Lead it on", right: true, text: "{pet} moves on happily.", effect: calmed},
			{id: "let", label: "Let it try one", text: "The berries upset {pet}'s stomach.", effect: upset},
		}},

	{id: "trip_river_pond", icon: "🫗", title: "A still pond", places: []string{"river"},
		text:   "While at the river, {pet} wants to drink from a still, green pond.",
		lesson: "Still water can carry parasites, bacteria and toxic algae. Give your pet clean water from home.",
		choices: []happeningChoice{
			{id: "fresh", label: "Give it water from your bottle", right: true, text: "{pet} drinks the clean water.", effect: calmed},
			{id: "let", label: "Let it drink", text: "The pond water upsets {pet}'s stomach.", effect: upset},
		}},
	{id: "trip_river_current", icon: "🌊", title: "A strong current", places: []string{"river"},
		text:   "While at the river, {pet} wants to wade out into the fast water.",
		lesson: "Even strong swimmers get swept away. Keep pets in calm shallows; a dog life jacket helps.",
		choices: []happeningChoice{
			{id: "shallows", label: "Keep it in the calm shallows", right: true, text: "{pet} splashes happily in the shallows.", effect: played},
			{id: "deep", label: "Let it swim out", text: "The current tugs {pet} and gives it a fright.", effect: scared},
		}},
	{id: "trip_river_rocks", icon: "🪨", title: "Slippery rocks", places: []string{"river"},
		text:   "While at the river, {pet} wants to leap across the wet rocks.",
		lesson: "Wet rocks are slippery, and falls cause sprains and cuts. Take the easy path.",
		choices: []happeningChoice{
			{id: "path", label: "Take the path round", right: true, text: "{pet} finds something by the path.", effect: extraFind},
			{id: "leap", label: "Let it leap", text: "{pet} slips and bangs a leg.", effect: hurt},
		}},

	{id: "trip_snow_paws", icon: "🐾", title: "Snowballs in the paws", places: []string{"snowy_hill"},
		text:   "While on the snowy hill, {pet} is limping: ice has balled up between its toes.",
		lesson: "Ice balls between the pads hurt. Clear them out, keep the fur between the toes trimmed, and wipe paws after walks.",
		choices: []happeningChoice{
			{id: "clear", label: "Clear out the ice and check its paws", right: true, text: "{pet} trots on comfortably.", effect: calmed},
			{id: "keep", label: "Keep going", text: "{pet}'s paws are sore and cracked.", effect: hurt},
		}},
	{id: "trip_snow_cold", icon: "🥶", title: "Shivering", places: []string{"snowy_hill"},
		text:   "While on the snowy hill, {pet} is shivering.",
		lesson: "Small, short-coated, young and old pets feel the cold. Shivering means it's time to warm up; a coat helps.",
		choices: []happeningChoice{
			{id: "warm", label: "Head somewhere warm", right: true, text: "{pet} warms up and cheers up.", effect: calmed},
			{id: "stay", label: "Stay out a bit longer", text: "{pet} gets chilled and miserable.", effect: tired},
		}},

	{id: "trip_city_truck", icon: "🚚", title: "A loud truck", places: []string{"city"},
		text:   "While in the city, a loud truck rumbles past and {pet} startles.",
		lesson: "Calmly comforting a frightened pet helps it recover. Rushing about tells it there really is danger.",
		choices: []happeningChoice{
			{id: "comfort", label: "Comfort it calmly", right: true, text: "{pet} settles and carries on.", effect: calmed},
			{id: "rush", label: "Hurry it home", text: "{pet} stays jumpy all the way home.", effect: scared},
		}},
	{id: "trip_city_chips", icon: "🍟", title: "Dropped chips", places: []string{"city"},
		text:   "While in the city, {pet} finds a pile of dropped, salty chips.",
		lesson: "Salty, greasy food upsets pets' stomachs, and street food can hide toxic ingredients. Teach 'leave it'.",
		choices: []happeningChoice{
			{id: "leave", label: "Say 'leave it' and reward it", right: true, text: "{pet} leaves them and gets a treat.", effect: calmed},
			{id: "eat", label: "Let it eat them", text: "The greasy chips upset {pet}'s stomach.", effect: upset},
		}},
	{id: "trip_city_crossing", icon: "🚦", title: "A busy road", places: []string{"city"},
		text:   "While in the city, you reach a busy crossing.",
		lesson: "Keep a pet on a short lead at roads; a sudden noise can make it bolt into traffic.",
		choices: []happeningChoice{
			{id: "short", label: "Keep it close on a short lead", right: true, text: "You cross safely together.", effect: calmed},
			{id: "long", label: "Let it walk ahead on a long lead", text: "A horn blares, and {pet} jumps in fright.", effect: scared},
		}},

	{id: "trip_lake_algae", icon: "🟢", title: "Green scum", places: []string{"lakeside"},
		text:   "While by the lake, {pet} wants to wade into water with a green scum on top.",
		lesson: "Blue-green algae blooms can kill pets within hours. Keep them out of scummy water, and call a vet if they go in.",
		choices: []happeningChoice{
			{id: "away", label: "Keep it out of the water", right: true, text: "{pet} explores the shore instead, and finds something.", effect: extraFind},
			{id: "let", label: "Let it paddle", text: "{pet} swallows some water and is sick; you rinse it and call the vet.", effect: upset},
		}},
	{id: "trip_lake_heat", icon: "🌡️", title: "Midday heat", places: []string{"lakeside"},
		text:   "While by the lake, the midday sun is fierce and {pet} is panting hard.",
		lesson: "Heavy panting, drooling and weakness are signs of heatstroke. Rest in the shade with water, and walk early or late on hot days.",
		choices: []happeningChoice{
			{id: "shade", label: "Rest in the shade with water", right: true, text: "{pet} cools off in the shade.", effect: calmed},
			{id: "keep", label: "Keep walking", text: "{pet} overheats and is exhausted.", effect: tired},
		}},

	{id: "trip_mountain_scree", icon: "⛰️", title: "Loose rocks", places: []string{"mountain"},
		text:   "While on the mountain, the path crosses a slope of loose rocks.",
		lesson: "Loose rock cuts paws and causes falls. Choose safe paths, and check paws after rough ground.",
		choices: []happeningChoice{
			{id: "safe", label: "Take the safer path", right: true, text: "{pet} finds something on the safer path.", effect: extraFind},
			{id: "scramble", label: "Scramble straight up", text: "{pet} cuts a paw pad on the rocks.", effect: hurt},
		}},
	{id: "trip_mountain_tired", icon: "😮‍💨", title: "Worn out", places: []string{"mountain"},
		text:   "While on the mountain, {pet} lags behind and lies down.",
		lesson: "A pet that lies down on a walk has had enough. Rest, give water, and turn back.",
		choices: []happeningChoice{
			{id: "rest", label: "Rest, give water and turn back", right: true, text: "{pet} recovers and trots home happily.", effect: calmed},
			{id: "push", label: "Push on to the top", text: "{pet} is exhausted.", effect: tired},
		}},
}

// startTripEventLocked may start an event about the trip just finished.
// Caller must hold p.mu.
func (p *DigitalPet) startTripEventLocked(place string) {
	if p.Happening != nil || !chance(tripEventChance) {
		return
	}
	var fits []happeningDef
	for _, h := range tripEvents {
		if len(h.places) > 0 && h.places[0] == place && p.happeningFitsLocked(h) {
			fits = append(fits, h)
		}
	}
	if len(fits) == 0 {
		return
	}
	h := fits[randIntn(len(fits))]
	p.Happening = &PendingHappening{ID: h.id, Since: p.playNowLocked()}
	p.markHappeningLocked(h.id)
}
