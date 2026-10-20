package core

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEventTypeString(t *testing.T) {
	tests := map[EventType]string{
		EventPetDied:       "PetDied",
		EventPetHibernated: "PetHibernated",
		EventPetSick:       "PetSick",
		EventPetRecovered:  "PetRecovered",
		EventPetHungry:     "PetHungry",
		EventPetThirsty:    "PetThirsty",
		EventWarningSign:   "WarningSign",
		EventTripReturned:  "TripReturned",
		EventTrickRusty:    "TrickRusty",
	}
	for eventType, want := range tests {
		if got := eventType.String(); got != want {
			t.Errorf("String() = %s, want %s", got, want)
		}
	}

	data, err := json.Marshal(GameEvent{Type: EventPetSick})
	if err != nil || !strings.Contains(string(data), `"type":"PetSick"`) {
		t.Errorf("Event types should be encoded by name, got %s (%v)", data, err)
	}
}

func TestEmitNumbersAndRecordsEvents(t *testing.T) {
	es := NewEventSystem()
	before := time.Now()
	first := es.Emit(GameEvent{Type: EventPetHungry, PetID: "pet_1", Message: "hungry"})
	second := es.Emit(GameEvent{Type: EventPetThirsty, PetID: "pet_1", Message: "thirsty"})

	if first.Seq != 1 || second.Seq != 2 || es.Count() != 2 {
		t.Errorf("Events should be numbered in order: got %d, %d (count %d)", first.Seq, second.Seq, es.Count())
	}
	if first.Timestamp.Before(before) {
		t.Error("Emit should set the time of an event that has none")
	}
	recent := es.Recent(10, nil)
	if len(recent) != 2 || recent[0].Message != "thirsty" || recent[1].Message != "hungry" {
		t.Errorf("Recent should list newest first, got %+v", recent)
	}
}

func TestHandlersRunInOrder(t *testing.T) {
	es := NewEventSystem()
	var calls []string
	es.RegisterHandler(EventPetSick, func(e GameEvent) { calls = append(calls, "first:"+e.Message) })
	es.RegisterHandler(EventPetSick, func(e GameEvent) { calls = append(calls, "second:"+e.Message) })
	es.RegisterHandler(EventPetHungry, func(e GameEvent) { calls = append(calls, "hungry") })

	es.Emit(GameEvent{Type: EventPetSick, Message: "ill"})
	if strings.Join(calls, ",") != "first:ill,second:ill" {
		t.Errorf("Handlers for the event's type should run in order, got %v", calls)
	}
}

func TestHandlerMayEmit(t *testing.T) {
	es := NewEventSystem()
	es.RegisterHandler(EventPetHibernated, func(e GameEvent) {
		es.Emit(GameEvent{Type: EventWarningSign, PetID: e.PetID, Message: "follow-up"})
	})
	es.Emit(GameEvent{Type: EventPetHibernated, PetID: "pet_1"})
	if es.Count() != 2 {
		t.Errorf("A handler should be able to emit an event, count = %d", es.Count())
	}
}

func TestHandlerPanicDoesNotStopOthers(t *testing.T) {
	es := NewEventSystem()
	called := false
	es.RegisterHandler(EventPetDied, func(GameEvent) { panic("test panic") })
	es.RegisterHandler(EventPetDied, func(GameEvent) { called = true })

	es.Emit(GameEvent{Type: EventPetDied})
	if !called {
		t.Error("A handler should run even if an earlier one panics")
	}
}

func TestClearHandlers(t *testing.T) {
	es := NewEventSystem()
	calls := 0
	es.RegisterHandler(EventPetSick, func(GameEvent) { calls++ })
	es.RegisterHandler(EventPetHungry, func(GameEvent) { calls++ })

	es.ClearHandlers(EventPetSick)
	es.Emit(GameEvent{Type: EventPetSick})
	es.Emit(GameEvent{Type: EventPetHungry})
	if calls != 1 {
		t.Errorf("Only the hunger handler should remain, calls = %d", calls)
	}

	es.ClearAllHandlers()
	es.Emit(GameEvent{Type: EventPetHungry})
	if calls != 1 {
		t.Errorf("No handlers should remain, calls = %d", calls)
	}
}

func TestEventHistoryLimit(t *testing.T) {
	es := NewEventSystem()
	for i := 0; i < defaultEventHistory+20; i++ {
		es.Emit(GameEvent{Type: EventPetHungry, PetID: "pet_1"})
	}
	all := es.Recent(1000, nil)
	if len(all) != defaultEventHistory {
		t.Fatalf("History should keep %d events, kept %d", defaultEventHistory, len(all))
	}
	if all[0].Seq != defaultEventHistory+20 || all[len(all)-1].Seq != 21 {
		t.Errorf("History should keep the newest events, kept %d to %d", all[len(all)-1].Seq, all[0].Seq)
	}
}

func TestRecentFiltersByType(t *testing.T) {
	es := NewEventSystem()
	es.Emit(GameEvent{Type: EventPetHungry})
	es.Emit(GameEvent{Type: EventPetSick})
	es.Emit(GameEvent{Type: EventPetHungry})

	hungry := EventPetHungry
	if got := es.Recent(10, &hungry); len(got) != 2 {
		t.Errorf("Recent(hungry) = %d events, want 2", len(got))
	}
	if got := es.Recent(1, nil); len(got) != 1 || got[0].Type != EventPetHungry {
		t.Errorf("Recent(1) should return the newest event, got %+v", got)
	}
}

func TestSinceReturnsOnePetsNewEvents(t *testing.T) {
	es := NewEventSystem()
	es.Emit(GameEvent{Type: EventPetHungry, PetID: "pet_1", Message: "old"})
	es.Emit(GameEvent{Type: EventPetHungry, PetID: "pet_2", Message: "other pet"})
	es.Emit(GameEvent{Type: EventPetSick, PetID: "pet_1", Message: "new"})

	events, last := es.Since(1, "pet_1")
	if last != 3 || len(events) != 1 || events[0].Message != "new" {
		t.Errorf("Since(1, pet_1) = %+v, last %d", events, last)
	}
	if events, _ := es.Since(last, "pet_1"); len(events) != 0 {
		t.Errorf("Nothing should be new after the last event, got %+v", events)
	}
}

func TestConcurrentEmitAndRegister(t *testing.T) {
	es := NewEventSystem()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			es.Emit(GameEvent{Type: EventPetHungry, PetID: "pet_1"})
		}()
		go func() {
			defer wg.Done()
			es.RegisterHandler(EventPetHungry, func(GameEvent) {})
		}()
	}
	wg.Wait()
	if es.Count() != 50 {
		t.Errorf("Count = %d, want 50", es.Count())
	}
	if events, _ := es.Since(0, "pet_1"); len(events) != 50 {
		t.Errorf("All 50 events should be recorded, got %d", len(events))
	}
}

// eventTypes lists the types of events
func eventTypes(events []GameEvent) []EventType {
	kinds := make([]EventType, len(events))
	for i, e := range events {
		kinds[i] = e.Type
	}
	return kinds
}

func TestConditionEventsWhenHungry(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	before := pet.Conditions()
	if events := ConditionEvents(before, pet.Conditions()); len(events) != 0 {
		t.Fatalf("Nothing changed, but got %v", eventTypes(events))
	}

	pet.Biology.Vitals.Nutrition = 0.1

	events := ConditionEvents(before, pet.Conditions())
	if len(events) != 1 || events[0].Type != EventPetHungry || events[0].Message != "Mochi is hungry." ||
		events[0].PetID != pet.ID {
		t.Errorf("Expected a hunger event, got %+v", events)
	}

	// A need that stays low is reported once
	if events := ConditionEvents(pet.Conditions(), pet.Conditions()); len(events) != 0 {
		t.Errorf("Hunger should be reported only when it starts, got %v", eventTypes(events))
	}
}

func TestConditionEventsForIllness(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	well := pet.Conditions()
	pet.Illness = &Illness{Kind: IllnessRespiratory, Severity: 0.2}
	sick := pet.Conditions()

	events := ConditionEvents(well, sick)
	if len(events) != 1 || events[0].Type != EventPetSick || events[0].Good {
		t.Fatalf("The illness, and not its warning sign, should be reported: %+v", events)
	}

	pet.Illness = nil
	events = ConditionEvents(sick, pet.Conditions())
	if len(events) != 1 || events[0].Type != EventPetRecovered || !events[0].Good {
		t.Errorf("Expected a recovery event, got %+v", events)
	}
}

func TestConditionEventsForWarningSign(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	before := pet.Conditions()
	pet.LivingAreaWaste = 0.8

	events := ConditionEvents(before, pet.Conditions())
	if len(events) != 1 || events[0].Type != EventWarningSign || !strings.Contains(events[0].Message, "litter box") {
		t.Errorf("Expected a warning sign about the litter box, got %+v", events)
	}
}

func TestConditionEventsForTrip(t *testing.T) {
	useClock(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	useRandom(t, 50, 0) // A treasure
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	if err := pet.SendExploring(); err != nil {
		t.Fatal(err)
	}
	before := pet.Conditions()
	pet.Update(exploreDuration + 0.1)

	var trip *GameEvent
	for _, e := range ConditionEvents(before, pet.Conditions()) {
		if e.Type == EventTripReturned {
			trip = &e
		}
	}
	if trip == nil || trip.Message != pet.LastAdventure.Text || !trip.Good {
		t.Errorf("Expected the trip's report as an event, got %+v", trip)
	}
}

func TestConditionEventsForDeath(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	before := pet.Conditions()
	pet.LivingAreaWaste = 0.8 // A warning sign that a death makes moot
	pet.Biology.IsAlive = false

	events := ConditionEvents(before, pet.Conditions())
	if len(events) != 1 || events[0].Type != EventPetDied || events[0].Message != "Mochi died." {
		t.Errorf("A death should be the only event, got %+v", events)
	}
}

func TestConditionEventsForHibernation(t *testing.T) {
	pet := NewDigitalPet("Mochi", "user123")
	before := pet.Conditions()
	pet.Hibernating = true

	events := ConditionEvents(before, pet.Conditions())
	if len(events) == 0 || events[0].Type != EventPetHibernated {
		t.Errorf("Expected a hibernation event first, got %v", eventTypes(events))
	}
}
