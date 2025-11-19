package core

import (
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// EventType represents different types of game events
type EventType int

const (
	EventPetCreated EventType = iota
	EventPetDied
	EventPetHungry
	EventPetSick
	EventPetHappy
	EventPetSad
	EventPetLevelUp
	EventInteraction
	EventBehaviorChanged
	EventMemoryConsolidated
	EventRelationshipFormed
	EventSkillLearned
	EventMilestoneReached
	EventNeedsCritical
	EventAutoSave
	EventAutoBackup
)

// String returns the string representation of EventType
func (et EventType) String() string {
	return [...]string{
		"PetCreated", "PetDied", "PetHungry", "PetSick", "PetHappy", "PetSad",
		"PetLevelUp", "Interaction", "BehaviorChanged", "MemoryConsolidated",
		"RelationshipFormed", "SkillLearned", "MilestoneReached", "NeedsCritical",
		"AutoSave", "AutoBackup",
	}[et]
}

// GameEvent represents an event that occurred in the game
type GameEvent struct {
	Type      EventType
	Timestamp time.Time
	PetID     types.PetID
	Data      map[string]interface{}
	Message   string
}

// EventHandler is a function that handles game events
type EventHandler func(event *GameEvent)

// EventSystem manages game events and notifications
type EventSystem struct {
	mu sync.RWMutex

	handlers      map[EventType][]EventHandler
	eventHistory  []*GameEvent
	maxHistory    int
	totalEvents   int64
}

// NewEventSystem creates a new event system
func NewEventSystem() *EventSystem {
	return &EventSystem{
		handlers:     make(map[EventType][]EventHandler),
		eventHistory: make([]*GameEvent, 0),
		maxHistory:   100, // Keep last 100 events
	}
}

// RegisterHandler registers a handler for a specific event type
func (es *EventSystem) RegisterHandler(eventType EventType, handler EventHandler) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.handlers[eventType] == nil {
		es.handlers[eventType] = make([]EventHandler, 0)
	}

	es.handlers[eventType] = append(es.handlers[eventType], handler)
}

// Emit emits an event to all registered handlers
func (es *EventSystem) Emit(event *GameEvent) {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store in history
	es.eventHistory = append(es.eventHistory, event)
	if len(es.eventHistory) > es.maxHistory {
		es.eventHistory = es.eventHistory[1:]
	}

	es.totalEvents++

	// Get handlers for this event type
	handlers := es.handlers[event.Type]

	// Release lock before calling handlers to avoid deadlock
	es.mu.Unlock()

	// Call all handlers
	for _, handler := range handlers {
		// Call in goroutine to avoid blocking
		go func(h EventHandler, evt *GameEvent) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash
					// In production, this would use proper logging
				}
			}()
			h(evt)
		}(handler, event)
	}

	// Re-acquire lock (deferred unlock will release it)
	es.mu.Lock()
}

// GetRecentEvents returns recent events, optionally filtered by type
func (es *EventSystem) GetRecentEvents(limit int, eventType *EventType) []*GameEvent {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events := make([]*GameEvent, 0)

	// Iterate backwards through history
	for i := len(es.eventHistory) - 1; i >= 0 && len(events) < limit; i-- {
		event := es.eventHistory[i]

		// Filter by type if specified
		if eventType != nil && event.Type != *eventType {
			continue
		}

		events = append(events, event)
	}

	return events
}

// GetEventCount returns the total number of events emitted
func (es *EventSystem) GetEventCount() int64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return es.totalEvents
}

// ClearHandlers removes all handlers for a specific event type
func (es *EventSystem) ClearHandlers(eventType EventType) {
	es.mu.Lock()
	defer es.mu.Unlock()

	delete(es.handlers, eventType)
}

// ClearAllHandlers removes all event handlers
func (es *EventSystem) ClearAllHandlers() {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.handlers = make(map[EventType][]EventHandler)
}

// CreateEvent is a helper to create a new game event
func CreateEvent(eventType EventType, petID types.PetID, message string) *GameEvent {
	return &GameEvent{
		Type:      eventType,
		PetID:     petID,
		Message:   message,
		Data:      make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// CreateEventWithData creates an event with custom data
func CreateEventWithData(eventType EventType, petID types.PetID, message string, data map[string]interface{}) *GameEvent {
	return &GameEvent{
		Type:      eventType,
		PetID:     petID,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}
