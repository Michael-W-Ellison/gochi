package core

import (
	"sync"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewEventSystem(t *testing.T) {
	es := NewEventSystem()

	if es == nil {
		t.Fatal("NewEventSystem returned nil")
	}

	if es.handlers == nil {
		t.Error("handlers map not initialized")
	}

	if es.maxHistory != 100 {
		t.Errorf("maxHistory = %d, want 100", es.maxHistory)
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      string
	}{
		{EventPetCreated, "PetCreated"},
		{EventPetDied, "PetDied"},
		{EventInteraction, "Interaction"},
		{EventAutoSave, "AutoSave"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.eventType.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRegisterHandler(t *testing.T) {
	es := NewEventSystem()

	handlerCalled := false
	handler := func(event *GameEvent) {
		handlerCalled = true
	}

	es.RegisterHandler(EventPetCreated, handler)

	// Emit event
	event := CreateEvent(EventPetCreated, "pet_123", "Test pet created")
	es.Emit(event)

	// Give goroutine time to execute
	time.Sleep(10 * time.Millisecond)

	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

func TestMultipleHandlers(t *testing.T) {
	es := NewEventSystem()

	handler1Called := false
	handler2Called := false

	handler1 := func(event *GameEvent) {
		handler1Called = true
	}

	handler2 := func(event *GameEvent) {
		handler2Called = true
	}

	es.RegisterHandler(EventPetCreated, handler1)
	es.RegisterHandler(EventPetCreated, handler2)

	event := CreateEvent(EventPetCreated, "pet_123", "Test")
	es.Emit(event)

	time.Sleep(10 * time.Millisecond)

	if !handler1Called {
		t.Error("Handler 1 was not called")
	}

	if !handler2Called {
		t.Error("Handler 2 was not called")
	}
}

func TestEmitEvent(t *testing.T) {
	es := NewEventSystem()

	var receivedEvent *GameEvent
	handler := func(event *GameEvent) {
		receivedEvent = event
	}

	es.RegisterHandler(EventInteraction, handler)

	petID := types.PetID("test_pet")
	message := "Test interaction"
	event := CreateEvent(EventInteraction, petID, message)
	es.Emit(event)

	time.Sleep(10 * time.Millisecond)

	if receivedEvent == nil {
		t.Fatal("Handler did not receive event")
	}

	if receivedEvent.Type != EventInteraction {
		t.Errorf("Event type = %v, want %v", receivedEvent.Type, EventInteraction)
	}

	if receivedEvent.PetID != petID {
		t.Errorf("PetID = %v, want %v", receivedEvent.PetID, petID)
	}

	if receivedEvent.Message != message {
		t.Errorf("Message = %s, want %s", receivedEvent.Message, message)
	}
}

func TestEventHistory(t *testing.T) {
	es := NewEventSystem()

	// Emit multiple events
	for i := 0; i < 5; i++ {
		event := CreateEvent(EventInteraction, types.PetID("pet_123"), "Test")
		es.Emit(event)
	}

	// Get recent events
	events := es.GetRecentEvents(10, nil)

	if len(events) != 5 {
		t.Errorf("GetRecentEvents returned %d events, want 5", len(events))
	}
}

func TestEventHistoryLimit(t *testing.T) {
	es := NewEventSystem()
	es.maxHistory = 10

	// Emit more events than max history
	for i := 0; i < 20; i++ {
		event := CreateEvent(EventInteraction, types.PetID("pet_123"), "Test")
		es.Emit(event)
	}

	// Should only keep last 10
	if len(es.eventHistory) != 10 {
		t.Errorf("Event history length = %d, want 10", len(es.eventHistory))
	}
}

func TestGetRecentEventsWithFilter(t *testing.T) {
	es := NewEventSystem()

	// Emit different event types
	es.Emit(CreateEvent(EventPetCreated, "pet_1", "Created"))
	es.Emit(CreateEvent(EventInteraction, "pet_1", "Interaction 1"))
	es.Emit(CreateEvent(EventInteraction, "pet_1", "Interaction 2"))
	es.Emit(CreateEvent(EventPetHappy, "pet_1", "Happy"))

	// Filter for interactions only
	interactionType := EventInteraction
	events := es.GetRecentEvents(10, &interactionType)

	if len(events) != 2 {
		t.Errorf("Filtered events = %d, want 2", len(events))
	}

	for _, event := range events {
		if event.Type != EventInteraction {
			t.Errorf("Event type = %v, want %v", event.Type, EventInteraction)
		}
	}
}

func TestGetEventCount(t *testing.T) {
	es := NewEventSystem()

	initialCount := es.GetEventCount()
	if initialCount != 0 {
		t.Errorf("Initial count = %d, want 0", initialCount)
	}

	// Emit events
	for i := 0; i < 5; i++ {
		es.Emit(CreateEvent(EventInteraction, "pet_123", "Test"))
	}

	count := es.GetEventCount()
	if count != 5 {
		t.Errorf("Event count = %d, want 5", count)
	}
}

func TestClearHandlers(t *testing.T) {
	es := NewEventSystem()

	handlerCalled := false
	handler := func(event *GameEvent) {
		handlerCalled = true
	}

	es.RegisterHandler(EventPetCreated, handler)
	es.ClearHandlers(EventPetCreated)

	// Emit event - handler should not be called
	es.Emit(CreateEvent(EventPetCreated, "pet_123", "Test"))
	time.Sleep(10 * time.Millisecond)

	if handlerCalled {
		t.Error("Handler should not be called after clearing")
	}
}

func TestClearAllHandlers(t *testing.T) {
	es := NewEventSystem()

	handler1Called := false
	handler2Called := false

	es.RegisterHandler(EventPetCreated, func(e *GameEvent) { handler1Called = true })
	es.RegisterHandler(EventInteraction, func(e *GameEvent) { handler2Called = true })

	es.ClearAllHandlers()

	// Emit events - no handlers should be called
	es.Emit(CreateEvent(EventPetCreated, "pet_123", "Test"))
	es.Emit(CreateEvent(EventInteraction, "pet_123", "Test"))
	time.Sleep(10 * time.Millisecond)

	if handler1Called || handler2Called {
		t.Error("No handlers should be called after clearing all")
	}
}

func TestCreateEvent(t *testing.T) {
	eventType := EventPetCreated
	petID := types.PetID("pet_123")
	message := "Test message"

	event := CreateEvent(eventType, petID, message)

	if event.Type != eventType {
		t.Errorf("Type = %v, want %v", event.Type, eventType)
	}

	if event.PetID != petID {
		t.Errorf("PetID = %v, want %v", event.PetID, petID)
	}

	if event.Message != message {
		t.Errorf("Message = %s, want %s", event.Message, message)
	}

	if event.Data == nil {
		t.Error("Data map should be initialized")
	}

	if event.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestCreateEventWithData(t *testing.T) {
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	event := CreateEventWithData(EventInteraction, "pet_123", "Test", data)

	if event.Data == nil {
		t.Fatal("Data should not be nil")
	}

	if event.Data["key1"] != "value1" {
		t.Error("Data not set correctly")
	}

	if event.Data["key2"] != 123 {
		t.Error("Data not set correctly")
	}
}

func TestConcurrentEventEmission(t *testing.T) {
	es := NewEventSystem()

	var wg sync.WaitGroup
	eventCount := 100

	// Emit events concurrently
	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.Emit(CreateEvent(EventInteraction, "pet_123", "Test"))
		}()
	}

	wg.Wait()

	count := es.GetEventCount()
	if count != int64(eventCount) {
		t.Errorf("Event count = %d, want %d", count, eventCount)
	}
}

func TestConcurrentHandlerRegistration(t *testing.T) {
	es := NewEventSystem()

	var wg sync.WaitGroup
	handlerCount := 50

	// Register handlers concurrently
	for i := 0; i < handlerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.RegisterHandler(EventInteraction, func(e *GameEvent) {})
		}()
	}

	wg.Wait()

	// Should not panic and all handlers should be registered
	if len(es.handlers[EventInteraction]) != handlerCount {
		t.Errorf("Handler count = %d, want %d",
			len(es.handlers[EventInteraction]), handlerCount)
	}
}

func TestEventTimestamp(t *testing.T) {
	es := NewEventSystem()

	before := time.Now()
	event := CreateEvent(EventPetCreated, "pet_123", "Test")
	es.Emit(event)
	after := time.Now()

	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Error("Event timestamp not within expected range")
	}
}

func TestHandlerErrorDoesNotBlockOthers(t *testing.T) {
	es := NewEventSystem()

	handler1Called := false
	handler2Called := false

	// Handler 1 panics
	es.RegisterHandler(EventInteraction, func(e *GameEvent) {
		panic("test panic")
	})

	// Handler 2 should still be called
	es.RegisterHandler(EventInteraction, func(e *GameEvent) {
		handler2Called = true
	})

	// Should not panic the test
	es.Emit(CreateEvent(EventInteraction, "pet_123", "Test"))
	time.Sleep(10 * time.Millisecond)

	if !handler2Called {
		t.Error("Handler 2 should be called even if Handler 1 panics")
	}

	// handler1Called is not set because it panics, which is expected
	_ = handler1Called
}
