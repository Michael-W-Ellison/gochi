package ai

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewMemorySystem(t *testing.T) {
	capacity := 50
	ms := NewMemorySystem(capacity)

	if ms == nil {
		t.Fatal("NewMemorySystem returned nil")
	}

	if ms.MemoryCapacity != capacity {
		t.Errorf("Expected capacity %d, got %d", capacity, ms.MemoryCapacity)
	}

	if ms.ShortTermMemories == nil {
		t.Error("ShortTermMemories should not be nil")
	}

	if ms.LongTermMemories == nil {
		t.Error("LongTermMemories should not be nil")
	}

	if ms.MemoryIndex == nil {
		t.Error("MemoryIndex should not be nil")
	}

	if ms.TotalMemories != 0 {
		t.Errorf("Expected TotalMemories 0, got %d", ms.TotalMemories)
	}
}

func TestRecordMemory(t *testing.T) {
	ms := NewMemorySystem(50)

	details := map[string]interface{}{
		"location": "forest",
		"value":    100,
	}

	ms.RecordMemory(MemoryEvent, "Test event", 100.0, 0.8, "happy", details)

	if len(ms.ShortTermMemories) != 1 {
		t.Errorf("Expected 1 memory, got %d", len(ms.ShortTermMemories))
	}

	mem := ms.ShortTermMemories[0]

	if mem.Type != MemoryEvent {
		t.Errorf("Expected MemoryEvent, got %v", mem.Type)
	}

	if mem.Description != "Test event" {
		t.Errorf("Expected 'Test event', got %s", mem.Description)
	}

	if mem.Strength != 0.8 {
		t.Errorf("Expected strength 0.8, got %f", mem.Strength)
	}

	if mem.Emotion != "happy" {
		t.Errorf("Expected emotion 'happy', got %s", mem.Emotion)
	}

	if mem.IsLongTerm {
		t.Error("New memories should not be long-term")
	}

	if ms.TotalMemories != 1 {
		t.Errorf("Expected TotalMemories 1, got %d", ms.TotalMemories)
	}
}

func TestRecordInteraction(t *testing.T) {
	ms := NewMemorySystem(50)

	ms.RecordInteraction(types.InteractionFeeding, 100.0, 0.9, "content")

	if len(ms.ShortTermMemories) != 1 {
		t.Errorf("Expected 1 memory, got %d", len(ms.ShortTermMemories))
	}

	mem := ms.ShortTermMemories[0]

	if mem.Type != MemoryInteraction {
		t.Errorf("Expected MemoryInteraction, got %v", mem.Type)
	}

	// Strength should be 0.3 + (0.9 * 0.7) = 0.93
	expectedStrength := 0.3 + (0.9 * 0.7)
	tolerance := 0.0001
	if mem.Strength < expectedStrength-tolerance || mem.Strength > expectedStrength+tolerance {
		t.Errorf("Expected strength %f, got %f", expectedStrength, mem.Strength)
	}

	interactionType, ok := mem.Details["interaction_type"]
	if !ok {
		t.Error("Details should contain interaction_type")
	}

	if interactionType != types.InteractionFeeding.String() {
		t.Errorf("Expected %s, got %v", types.InteractionFeeding.String(), interactionType)
	}
}

func TestConsolidateMemories(t *testing.T) {
	ms := NewMemorySystem(50)

	// Add weak memory (should not consolidate)
	ms.RecordMemory(MemoryEvent, "Weak memory", 100.0, 0.4, "neutral", nil)

	// Add strong memory (should consolidate)
	ms.RecordMemory(MemoryEvent, "Strong memory", 100.0, 0.8, "happy", nil)

	ms.ConsolidateMemories()

	if len(ms.LongTermMemories) != 1 {
		t.Errorf("Expected 1 long-term memory, got %d", len(ms.LongTermMemories))
	}

	if !ms.LongTermMemories[0].IsLongTerm {
		t.Error("Consolidated memory should be marked as long-term")
	}

	if ms.LongTermMemories[0].Description != "Strong memory" {
		t.Errorf("Expected 'Strong memory', got %s", ms.LongTermMemories[0].Description)
	}
}

func TestDecayMemories(t *testing.T) {
	ms := NewMemorySystem(50)

	// Add memories
	ms.RecordMemory(MemoryEvent, "Memory 1", 100.0, 0.8, "happy", nil)
	ms.RecordMemory(MemoryEvent, "Memory 2", 100.0, 0.6, "neutral", nil)

	initialStrength1 := ms.ShortTermMemories[0].Strength
	initialStrength2 := ms.ShortTermMemories[1].Strength

	// Decay memories
	ms.DecayMemories(1.0)

	// Strength should have decreased
	if ms.ShortTermMemories[0].Strength >= initialStrength1 {
		t.Error("Memory strength should have decayed")
	}

	if ms.ShortTermMemories[1].Strength >= initialStrength2 {
		t.Error("Memory strength should have decayed")
	}
}

func TestPruneWeakMemories(t *testing.T) {
	ms := NewMemorySystem(5)

	// Add more memories than capacity
	for i := 0; i < 10; i++ {
		strength := float64(i) / 10.0
		ms.RecordMemory(MemoryEvent, "Memory", 100.0, strength, "neutral", nil)
	}

	// Should have pruned to capacity
	if len(ms.ShortTermMemories) > ms.MemoryCapacity {
		t.Errorf("Expected at most %d memories, got %d", ms.MemoryCapacity, len(ms.ShortTermMemories))
	}

	// Remaining memories should be stronger ones
	for _, mem := range ms.ShortTermMemories {
		if mem.Strength < 0.5 {
			t.Errorf("Weak memory should have been pruned: strength %f", mem.Strength)
		}
	}
}

func TestRecallMemories(t *testing.T) {
	ms := NewMemorySystem(50)

	// Add memories with different types
	ms.RecordMemory(MemoryInteraction, "Interaction 1", 100.0, 0.8, "happy", nil)
	ms.RecordMemory(MemoryEvent, "Event 1", 100.0, 0.7, "neutral", nil)
	ms.RecordMemory(MemoryInteraction, "Interaction 2", 100.0, 0.9, "excited", nil)
	ms.RecordMemory(MemoryLocation, "Location 1", 100.0, 0.6, "curious", nil)

	// Recall interaction memories
	recalled := ms.RecallMemories(MemoryInteraction, 3)

	if len(recalled) != 2 {
		t.Errorf("Expected 2 interaction memories, got %d", len(recalled))
	}

	// Verify they are interaction type
	for _, mem := range recalled {
		if mem.Type != MemoryInteraction {
			t.Errorf("Expected MemoryInteraction, got %v", mem.Type)
		}
	}
}

func TestGetRecentMemories(t *testing.T) {
	ms := NewMemorySystem(50)

	// Add memories
	for i := 0; i < 10; i++ {
		ms.RecordMemory(MemoryEvent, "Memory", 100.0+float64(i), 0.5, "neutral", nil)
	}

	// Get recent 5
	recent := ms.GetRecentMemories(5)

	if len(recent) != 5 {
		t.Errorf("Expected 5 memories, got %d", len(recent))
	}

	// Recent memories are the last N in the array (in chronological order)
	// The last one should have the highest GameTime
	if recent[len(recent)-1].GameTime != 109.0 {
		t.Errorf("Last recent memory should have GameTime 109.0, got %f", recent[len(recent)-1].GameTime)
	}

	// Should be in chronological order (oldest to newest)
	for i := 0; i < len(recent)-1; i++ {
		if recent[i].GameTime > recent[i+1].GameTime {
			t.Error("Recent memories should be in chronological order")
		}
	}
}

func TestGetStrongestMemories(t *testing.T) {
	ms := NewMemorySystem(50)

	// Add memories with different strengths
	strengths := []float64{0.3, 0.8, 0.5, 0.9, 0.4}
	for i, strength := range strengths {
		ms.RecordMemory(MemoryEvent, "Memory", 100.0+float64(i), strength, "neutral", nil)
	}

	// Get strongest 3
	strongest := ms.GetStrongestMemories(3)

	if len(strongest) != 3 {
		t.Errorf("Expected 3 memories, got %d", len(strongest))
	}

	// Should be sorted by strength (strongest first)
	if strongest[0].Strength < strongest[1].Strength {
		t.Error("Strongest memories should be sorted by strength (descending)")
	}

	if strongest[1].Strength < strongest[2].Strength {
		t.Error("Strongest memories should be sorted by strength (descending)")
	}
}

func TestGetMemoryCount(t *testing.T) {
	ms := NewMemorySystem(50)

	shortTerm, longTerm := ms.GetMemoryCount()
	if shortTerm != 0 || longTerm != 0 {
		t.Errorf("Expected 0 memories, got %d short-term, %d long-term", shortTerm, longTerm)
	}

	ms.RecordMemory(MemoryEvent, "Memory 1", 100.0, 0.5, "neutral", nil)
	ms.RecordMemory(MemoryEvent, "Memory 2", 100.0, 0.5, "neutral", nil)

	shortTerm, longTerm = ms.GetMemoryCount()
	if shortTerm != 2 || longTerm != 0 {
		t.Errorf("Expected 2 short-term, 0 long-term, got %d short-term, %d long-term", shortTerm, longTerm)
	}

	// Consolidate one memory
	ms.ShortTermMemories[0].Strength = 0.8
	ms.ConsolidateMemories()

	shortTerm, longTerm = ms.GetMemoryCount()
	expectedShortTerm := len(ms.ShortTermMemories)
	expectedLongTerm := len(ms.LongTermMemories)
	if shortTerm != expectedShortTerm || longTerm != expectedLongTerm {
		t.Errorf("Expected %d short-term, %d long-term, got %d short-term, %d long-term",
			expectedShortTerm, expectedLongTerm, shortTerm, longTerm)
	}
}

func TestHasMemoryOf(t *testing.T) {
	ms := NewMemorySystem(50)

	ms.RecordInteraction(types.InteractionFeeding, 100.0, 0.8, "happy")
	ms.RecordMemory(MemoryEvent, "Event memory", 100.0, 0.7, "excited", nil)

	if !ms.HasMemoryOf(MemoryInteraction) {
		t.Error("Should have memory of interaction type")
	}

	if !ms.HasMemoryOf(MemoryEvent) {
		t.Error("Should have memory of event type")
	}

	if ms.HasMemoryOf(MemoryLocation) {
		t.Error("Should not have memory of location type")
	}
}

func TestMemoryStrengthClamping(t *testing.T) {
	ms := NewMemorySystem(50)

	// Try to record memory with out-of-bounds strength
	ms.RecordMemory(MemoryEvent, "Too strong", 100.0, 1.5, "happy", nil)
	ms.RecordMemory(MemoryEvent, "Too weak", 100.0, -0.5, "sad", nil)

	if ms.ShortTermMemories[0].Strength > 1.0 {
		t.Errorf("Strength should be clamped to 1.0, got %f", ms.ShortTermMemories[0].Strength)
	}

	if ms.ShortTermMemories[1].Strength < 0.0 {
		t.Errorf("Strength should be clamped to 0.0, got %f", ms.ShortTermMemories[1].Strength)
	}
}

func TestMemoryIndex(t *testing.T) {
	ms := NewMemorySystem(50)

	ms.RecordMemory(MemoryEvent, "Indexed memory", 100.0, 0.5, "neutral", nil)

	mem := ms.ShortTermMemories[0]

	// Check that memory is in index
	indexed, ok := ms.MemoryIndex[mem.ID]
	if !ok {
		t.Error("Memory should be in index")
	}

	if indexed.Description != "Indexed memory" {
		t.Errorf("Expected 'Indexed memory', got %s", indexed.Description)
	}
}

func TestMultipleConsolidations(t *testing.T) {
	ms := NewMemorySystem(50)

	// Add strong memories
	for i := 0; i < 5; i++ {
		ms.RecordMemory(MemoryEvent, "Strong memory", 100.0+float64(i), 0.9, "happy", nil)
	}

	ms.ConsolidateMemories()

	if len(ms.LongTermMemories) != 5 {
		t.Errorf("Expected 5 long-term memories, got %d", len(ms.LongTermMemories))
	}

	// Try to consolidate again - should not duplicate
	initialLongTermCount := len(ms.LongTermMemories)
	ms.ConsolidateMemories()

	if len(ms.LongTermMemories) != initialLongTermCount {
		t.Error("Should not duplicate memories in long-term storage")
	}
}
