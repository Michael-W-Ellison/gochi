package core

import (
	"fmt"
	"log"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Events record what happens to a pet on its own while time passes - it
// gets hungry, falls ill, comes home from a trip - so the player can see
// what happened while they weren't watching. Care the player gives is
// reported directly and is not an event.

// EventType is the kind of thing that happened
type EventType int

const (
	EventPetDied EventType = iota
	EventPetHibernated
	EventPetSick
	EventPetRecovered
	EventPetHungry
	EventPetThirsty
	EventWarningSign
	EventTripReturned
	EventTrickRusty
	EventLifeStage
	EventAchievement
	EventGoal // A daily goal was completed
)

// String returns the event type's name
func (et EventType) String() string {
	return [...]string{
		"PetDied", "PetHibernated", "PetSick", "PetRecovered", "PetHungry",
		"PetThirsty", "WarningSign", "TripReturned", "TrickRusty", "LifeStage", "Achievement", "Goal",
	}[et]
}

// MarshalText encodes the event type by name
func (et EventType) MarshalText() ([]byte, error) {
	return []byte(et.String()), nil
}

// GameEvent is something that happened to a pet
type GameEvent struct {
	Seq       int64       `json:"seq"` // Order of the event; increases by one per event
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	PetID     types.PetID `json:"pet_id"`
	Message   string      `json:"message"`
	Good      bool        `json:"good"` // Good news, such as a recovery
}

// EventHandler handles events of one type
type EventHandler func(event GameEvent)

// defaultEventHistory is how many events an EventSystem keeps
const defaultEventHistory = 100

// EventSystem records events and passes them to registered handlers
type EventSystem struct {
	mu         sync.Mutex
	handlers   map[EventType][]EventHandler
	history    []GameEvent // Oldest first
	maxHistory int
	lastSeq    int64
}

// NewEventSystem creates an event system that keeps the last 100 events
func NewEventSystem() *EventSystem {
	return &EventSystem{
		handlers:   make(map[EventType][]EventHandler),
		maxHistory: defaultEventHistory,
	}
}

// RegisterHandler registers a handler for one type of event
func (es *EventSystem) RegisterHandler(eventType EventType, handler EventHandler) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.handlers[eventType] = append(es.handlers[eventType], handler)
}

// Emit records an event and calls its handlers, in the order they were
// registered, and returns the event as recorded. Handlers run after the
// event is recorded, without the event system's lock held, so they may
// emit events themselves. A handler that panics is logged and does not
// stop the others.
func (es *EventSystem) Emit(event GameEvent) GameEvent {
	es.mu.Lock()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	es.lastSeq++
	event.Seq = es.lastSeq
	es.history = append(es.history, event)
	if over := len(es.history) - es.maxHistory; over > 0 {
		es.history = append(es.history[:0:0], es.history[over:]...)
	}
	handlers := append([]EventHandler(nil), es.handlers[event.Type]...)
	es.mu.Unlock()

	for _, handler := range handlers {
		callHandler(handler, event)
	}
	return event
}

// callHandler calls a handler, recovering from a panic
func callHandler(handler EventHandler, event GameEvent) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Event handler for %s panicked: %v\n%s", event.Type, r, debug.Stack())
		}
	}()
	handler(event)
}

// Recent returns up to limit of the most recent events, newest first. If
// eventType is not nil, only events of that type are returned.
func (es *EventSystem) Recent(limit int, eventType *EventType) []GameEvent {
	es.mu.Lock()
	defer es.mu.Unlock()

	events := []GameEvent{}
	for i := len(es.history) - 1; i >= 0 && len(events) < limit; i-- {
		if eventType == nil || es.history[i].Type == *eventType {
			events = append(events, es.history[i])
		}
	}
	return events
}

// Since returns the events for a pet recorded after seq, oldest first,
// and the sequence number of the last event recorded
func (es *EventSystem) Since(seq int64, petID types.PetID) ([]GameEvent, int64) {
	es.mu.Lock()
	defer es.mu.Unlock()

	events := []GameEvent{}
	for _, event := range es.history {
		if event.Seq > seq && event.PetID == petID {
			events = append(events, event)
		}
	}
	return events, es.lastSeq
}

// Count returns how many events have been emitted
func (es *EventSystem) Count() int64 {
	es.mu.Lock()
	defer es.mu.Unlock()
	return es.lastSeq
}

// ClearHandlers removes the handlers for one type of event
func (es *EventSystem) ClearHandlers(eventType EventType) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.handlers, eventType)
}

// ClearAllHandlers removes all handlers
func (es *EventSystem) ClearAllHandlers() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.handlers = make(map[EventType][]EventHandler)
}

// Conditions is a snapshot of the parts of a pet's state that events
// report changes in
type Conditions struct {
	id          types.PetID
	name        string
	alive       bool
	hibernating bool
	illness     string
	hungry      bool
	thirsty     bool
	adventure   *Adventure // The last trip's report
	signs       map[string]bool
	rusty       map[string]bool // Names of tricks the pet is rusty at
	stage       biology.LifeStage
	cause       string // Cause of death
}

// Conditions snapshots the pet's state, to compare with a later snapshot
// using ConditionEvents
func (p *DigitalPet) Conditions() Conditions {
	p.mu.Lock()
	defer p.mu.Unlock()

	c := Conditions{
		id: p.ID, name: p.Name, alive: p.Biology.IsAlive, hibernating: p.Hibernating,
		illness: p.illnessNameLocked(), adventure: p.LastAdventure, signs: make(map[string]bool),
		rusty: make(map[string]bool), stage: p.Biology.LifeStage(), cause: p.Biology.CauseOfDeath,
	}
	for _, t := range tricks {
		if p.TricksLearned[t.ID] && p.TrickSkills[t.ID] < 1.0 {
			c.rusty[t.Name] = true
		}
	}
	if c.alive {
		p.ensureNeedsLocked()
		p.syncPhysicalNeedsLocked()
		c.hungry = p.needWarningLocked(types.NeedHunger)
		c.thirsty = p.needWarningLocked(types.NeedThirst)
	}
	for _, symptom := range p.symptomsLocked() {
		if !symptom.Good {
			c.signs[symptom.Sign] = true
		}
	}
	return c
}

// ConditionEvents returns the events between two snapshots of the same
// pet, most serious first
func ConditionEvents(before, after Conditions) []GameEvent {
	var events []GameEvent
	add := func(eventType EventType, good bool, format string, args ...interface{}) {
		events = append(events, GameEvent{Type: eventType, PetID: after.id, Good: good,
			Message: fmt.Sprintf(format, args...)})
	}

	if before.alive && !after.alive {
		if after.cause == causeOldAge {
			add(EventPetDied, false, "%s died of old age, after a long life.", after.name)
		} else {
			add(EventPetDied, false, "%s died.", after.name)
		}
		return events
	}
	if after.stage != before.stage {
		switch after.stage {
		case biology.StageAdult:
			add(EventLifeStage, true, "%s is all grown up!", after.name)
		case biology.StageSenior:
			add(EventLifeStage, false, "%s is a senior now. Gentle exercise, easy-to-digest food and regular checkups help older pets.", after.name)
		}
	}
	if !before.hibernating && after.hibernating {
		add(EventPetHibernated, false, "%s went into hibernation and needs to be revived.", after.name)
	}
	switch {
	case after.illness != "" && after.illness != before.illness:
		add(EventPetSick, false, "%s fell ill: %s.", after.name, after.illness)
	case before.illness != "" && after.illness == "":
		add(EventPetRecovered, true, "%s recovered from %s.", after.name, before.illness)
	}
	if after.adventure != nil && after.adventure != before.adventure {
		add(EventTripReturned, after.adventure.Outcome != OutcomeInjured, "%s", after.adventure.Text)
	}
	if !before.hungry && after.hungry {
		add(EventPetHungry, false, "%s is hungry.", after.name)
	}
	if !before.thirsty && after.thirsty {
		add(EventPetThirsty, false, "%s is thirsty.", after.name)
	}
	for _, sign := range sortedKeys(after.signs) {
		if !before.signs[sign] && !illnessSign(sign) {
			add(EventWarningSign, false, "%s: %s.", after.name, sign)
		}
	}
	for _, trick := range sortedKeys(after.rusty) {
		if !before.rusty[trick] {
			add(EventTrickRusty, false, "%s is getting rusty at %s. Practice it to keep it sharp.", after.name, trick)
		}
	}
	return events
}

// sortedKeys returns a set's members in order, so events come out in the
// same order every time
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// illnessSign reports whether a warning sign is an illness's, which
// EventPetSick already reports
func illnessSign(sign string) bool {
	for _, bySpecies := range illnessSigns {
		for _, symptom := range bySpecies {
			if symptom.Sign == sign {
				return true
			}
		}
	}
	return false
}
