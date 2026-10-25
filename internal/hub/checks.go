package hub

import (
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// What the game shows from a hub, it rebuilds from its own lists: a hub
// is run by whoever set it up, so it must not be able to put words of its
// own in front of a child. Handles must be handles the hub could give;
// achievements, pet cards and playdate events are written again from the
// game's own text; anything else is dropped.

// cardIDPattern is a playdate card's ID (see makeCard)
var cardIDPattern = regexp.MustCompile(`^playmate_[a-f0-9]{24}$`)

// cardName is a playdate pet's name, as the hub gives it
func cardName(handle string, species core.Species) string {
	return handle + "'s " + strings.ToLower(species.Def().Name)
}

// checkFeed keeps the feed entries with a handle and a known species and
// achievement, with the game's own name, icon and description
func checkFeed(feed []SharedAchievement) []SharedAchievement {
	kept := []SharedAchievement{}
	for _, a := range feed {
		known, ok := core.FindAchievement(a.ID)
		if !ok || !ValidHandle(a.Handle) || !core.Species(a.Species).Known() {
			continue
		}
		kept = append(kept, SharedAchievement{Handle: a.Handle, Species: a.Species, ID: known.ID, Name: known.Name,
			Icon: known.Icon, Description: known.Description, SharedAt: a.SharedAt})
	}
	return kept
}

// checkCard makes a card of a player's pet from what the game knows: its
// name from the handle and species, and only values from the game's lists
// and numbers in range. It reports false for a card to drop.
func checkCard(c *Card, handle string) bool {
	species := core.Species(c.Species)
	if !ValidHandle(handle) || !species.Known() || !cardIDPattern.MatchString(c.ID) {
		return false
	}
	c.Name = cardName(handle, species)
	if !slices.Contains(core.SpriteStates, c.Look) {
		c.Look = core.SpriteStates[0]
	}
	if !slices.Contains(unavailableCodes, c.Unavailable) {
		c.Unavailable = ""
	}
	if len(c.Outfit) > len(core.CosmeticSlots) || !core.ValidOutfit(c.Outfit) {
		c.Outfit = nil
	}
	if b, ok := core.FindBreed(c.Breed); c.Breed != "" && (!ok || b.Species != species) {
		c.Breed = ""
	}
	c.Sociability = clampLevel(c.Sociability, 0)
	return true
}

// clampLevel brings a level into lowest to 1, rounded as the hub rounds it
func clampLevel(v, lowest float64) float64 {
	if math.IsNaN(v) {
		return lowest
	}
	return roundLevel(min(max(v, lowest), 1), lowest)
}

// checkState keeps a playdate's members and events that check out, with
// cards and event text rebuilt (see checkCard)
func checkState(s *PlaydateState) {
	if !ValidHandle(s.Host) {
		s.Host = ""
	}
	if !ValidHandle(s.You) {
		s.You = ""
	}
	members := []PlaydateMember{}
	for _, m := range s.Members {
		if checkCard(&m.Card, m.Handle) {
			m.Position = m.Position.clamp(m.Position)
			members = append(members, m)
		}
	}
	s.Members = members
	events := []PlaydateEvent{}
	for _, e := range s.Events {
		if checkEvent(&e, members) {
			events = append(events, e)
		}
	}
	s.Events = events
}

// checkEvent rebuilds a play event's card and text from the game's own
// words. It reports false for an event to drop.
func checkEvent(e *PlaydateEvent, members []PlaydateMember) bool {
	if e.Kind != "play" || !ValidHandle(e.To) || !checkCard(&e.FromCard, e.From) {
		return false
	}
	e.Quality, e.Bond = clampLevel(e.Quality, 0), clampLevel(e.Bond, -1)
	from := core.Species(e.FromCard.Species)
	if i := slices.IndexFunc(members, func(m PlaydateMember) bool { return m.Handle == e.To }); i >= 0 {
		e.ToPet = members[i].Card.ID
		e.Text = core.MeetingText(e.FromCard.Name, from, members[i].Card.Name, e.Quality)
		return true
	}
	if !cardIDPattern.MatchString(e.ToPet) {
		return false
	}
	// The other pet isn't in the list: the hub's text is kept if it is the
	// game's own for that pet, whatever its species
	hubText := e.Text
	e.Text = core.MeetingText(e.FromCard.Name, from, e.To+"'s pet", e.Quality)
	for _, def := range core.SpeciesList() {
		if text := core.MeetingText(e.FromCard.Name, from, cardName(e.To, def.ID), e.Quality); text == hubText {
			e.Text = text
		}
	}
	return true
}
