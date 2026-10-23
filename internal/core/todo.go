package core

import (
	"errors"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Things to do while the pet doesn't need care: a pet that isn't thirsty
// only takes a drink reluctantly, and care earns points only now and then,
// but there is usually something else to do. ThingsToDo lists the
// activities, each ready now or with why not, and care the pet turns down
// or takes reluctantly suggests the ones that are ready.

// ThingToDo is an activity the player can do with the pet
type ThingToDo struct {
	ID    string `json:"id"` // train, hide_seek, games, explore, shows or enrich
	Icon  string `json:"icon"`
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
	Why   string `json:"why"` // What it offers when ready, or why it isn't
}

// ThingsToDo returns the activities, those ready now first
func (p *DigitalPet) ThingsToDo() []ThingToDo {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.thingsToDoLocked()
}

// thingsToDoLocked returns the activities, those ready now first. Caller
// must hold p.mu.
func (p *DigitalPet) thingsToDoLocked() []ThingToDo {
	things := []ThingToDo{
		p.thingLocked("train", "🎓", "Train", "Practice a trick: each session builds its skill.",
			p.careProblemLocked(types.InteractionTraining)),
		p.thingLocked("hide_seek", "🙈", "Hide & Seek", "Find where "+p.Name+" is hiding.",
			p.careProblemLocked(types.InteractionPlaying)),
		p.gamesThingLocked(),
		p.thingLocked("explore", "🧭", "Explore", p.Name+" goes on a trip, and may bring back a souvenir.",
			p.activityProblemLocked(p.tripReadyLocked())),
		p.showsThingLocked(),
		p.thingLocked("enrich", "🏠", "Enrich", "Change something in "+p.Name+"'s home.", p.enrichProblemLocked()),
	}
	var ready, waiting []ThingToDo
	for _, t := range things {
		if t.Ready {
			ready = append(ready, t)
		} else {
			waiting = append(waiting, t)
		}
	}
	return append(ready, waiting...)
}

// thingLocked returns an activity, ready if problem is "". Caller must
// hold p.mu.
func (p *DigitalPet) thingLocked(id, icon, name, offer, problem string) ThingToDo {
	if problem != "" {
		return ThingToDo{ID: id, Icon: icon, Name: name, Why: problem}
	}
	return ThingToDo{ID: id, Icon: icon, Name: name, Ready: true, Why: offer}
}

// careProblemLocked says why the pet can't do an activity that gives this
// care now, or returns "". Caller must hold p.mu.
func (p *DigitalPet) careProblemLocked(care types.InteractionType) string {
	if problem := p.activityProblemLocked(p.checkActiveLocked()); problem != "" {
		return problem
	}
	if ok, reason := interaction.ValidateInteraction(care, p.interactionContextLocked()); !ok {
		return strings.Replace(reason, "Pet", p.Name, 1) + "."
	}
	return ""
}

// activityProblemLocked says, to the player, why an activity check failed,
// or returns "" for nil. Caller must hold p.mu.
func (p *DigitalPet) activityProblemLocked(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrSleeping):
		return p.Name + " is asleep."
	case errors.Is(err, ErrHibernating):
		return p.Name + " is hibernating."
	case errors.Is(err, ErrPetNotAvailable):
		return p.Name + " has passed away."
	case errors.Is(err, ErrExploring):
		return p.Name + " is out exploring."
	case errors.Is(err, ErrTooTired):
		return p.Name + " is too tired. Let it rest, or give it some food."
	case errors.Is(err, ErrNeedsCare):
		return p.Name + " needs food, water and energy first."
	case errors.Is(err, ErrInjured):
		return p.Name + " is hurt. Heal it first."
	}
	return err.Error()
}

// gamesThingLocked returns the mini-games as one activity: ready if any
// can be played now. Caller must hold p.mu.
func (p *DigitalPet) gamesThingLocked() ThingToDo {
	var names []string
	refusal := ""
	for _, g := range miniGames {
		if g.species != "" && g.species != p.speciesLocked() || !p.hadCareLocked(g) {
			continue
		}
		if r := p.gameRefusalLocked(g); r != "" {
			refusal = r
			continue
		}
		names = append(names, g.icon+" "+g.name)
	}
	switch {
	case len(names) > 0:
		return p.thingLocked("games", "🎮", "Games", "Play "+strings.Join(names, ", ")+".", "")
	case refusal != "":
		return p.thingLocked("games", "🎮", "Games", "", refusal)
	}
	return p.thingLocked("games", "🎮", "Games", "", "Games open after play, enrichment or grooming.")
}

// showsThingLocked returns the shows as one activity: ready if the pet can
// enter one now. Caller must hold p.mu.
func (p *DigitalPet) showsThingLocked() ThingToDo {
	problem := ""
	for _, s := range p.showsForLocked() {
		if problem = p.showProblemLocked(s); problem == "" {
			return p.thingLocked("shows", "🏅", "Shows", "Enter the "+s.Name+" show.", "")
		}
	}
	if problem == "" {
		problem = "There are no shows for " + p.Name + "."
	}
	return p.thingLocked("shows", "🏅", "Shows", "", problem)
}

// enrichProblemLocked says why the pet's surroundings can't be changed now,
// or returns "". Caller must hold p.mu.
func (p *DigitalPet) enrichProblemLocked() string {
	switch {
	case !p.Biology.IsAlive:
		return p.Name + " has passed away."
	case p.Hibernating:
		return p.Name + " is hibernating."
	case p.isExploringLocked():
		return p.Name + " is out exploring."
	}
	return ""
}

// suggestionLocked suggests up to two activities ready now, or returns ""
// if none is. Caller must hold p.mu.
func (p *DigitalPet) suggestionLocked() string {
	var names []string
	for _, t := range p.thingsToDoLocked() {
		if t.Ready && len(names) < 2 {
			names = append(names, t.Icon+" "+t.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "Meanwhile, try " + strings.Join(names, " or ") + "."
}
