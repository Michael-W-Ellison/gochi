package ai

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestMemoryTypeString(t *testing.T) {
	tests := []struct {
		memType  MemoryType
		expected string
	}{
		{MemoryInteraction, "Interaction"},
		{MemoryEvent, "Event"},
		{MemoryLocation, "Location"},
		{MemorySocial, "Social"},
		{MemoryTraining, "Training"},
		{MemoryEmotional, "Emotional"},
		{MemoryTrauma, "Trauma"},
		{MemoryPositive, "Positive"},
	}

	for _, tt := range tests {
		if got := tt.memType.String(); got != tt.expected {
			t.Errorf("MemoryType.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestMemoryImportanceString(t *testing.T) {
	tests := []struct {
		importance MemoryImportance
		expected   string
	}{
		{ImportanceTrivial, "Trivial"},
		{ImportanceMinor, "Minor"},
		{ImportanceModerate, "Moderate"},
		{ImportanceSignificant, "Significant"},
		{ImportanceMajor, "Major"},
		{ImportanceFormative, "Formative"},
	}

	for _, tt := range tests {
		if got := tt.importance.String(); got != tt.expected {
			t.Errorf("MemoryImportance.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestNewMemorySystem(t *testing.T) {
	ms := NewMemorySystem(100)

	if ms.MemoryCapacity != 100 {
		t.Errorf("Expected capacity 100, got %d", ms.MemoryCapacity)
	}

	if ms.LongTermCapacity != 500 {
		t.Errorf("Expected long-term capacity 500, got %d", ms.LongTermCapacity)
	}

	if ms.WorkingMemoryCap != 7 {
		t.Errorf("Expected working memory capacity 7, got %d", ms.WorkingMemoryCap)
	}

	if ms.TraumaThreshold != -0.7 {
		t.Errorf("Expected trauma threshold -0.7, got %f", ms.TraumaThreshold)
	}

	if ms.PositiveThreshold != 0.7 {
		t.Errorf("Expected positive threshold 0.7, got %f", ms.PositiveThreshold)
	}
}

func TestNewMemorySystemWithPersonality(t *testing.T) {
	// High openness = more working memory
	ms := NewMemorySystemWithPersonality(100, 1.0, 1.0)

	if ms.WorkingMemoryCap != 9 {
		t.Errorf("Expected working memory cap 9 with high openness, got %d", ms.WorkingMemoryCap)
	}

	if ms.RetentionBonus != 0.2 {
		t.Errorf("Expected retention bonus 0.2 with high conscientiousness, got %f", ms.RetentionBonus)
	}

	// Low personality
	ms2 := NewMemorySystemWithPersonality(100, 0.0, 0.0)

	if ms2.WorkingMemoryCap != 5 {
		t.Errorf("Expected working memory cap 5 with low openness, got %d", ms2.WorkingMemoryCap)
	}

	if ms2.RetentionBonus != 0.0 {
		t.Errorf("Expected retention bonus 0.0 with low conscientiousness, got %f", ms2.RetentionBonus)
	}
}

func TestRecordMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	details := map[string]interface{}{
		"test": "value",
	}

	memory := ms.RecordMemory(MemoryEvent, "Test event", 100.0, 0.5, "happy", details)

	if memory == nil {
		t.Fatal("Expected memory to be created")
	}

	if memory.Type != MemoryEvent {
		t.Errorf("Expected type MemoryEvent, got %v", memory.Type)
	}

	if memory.Description != "Test event" {
		t.Errorf("Expected description 'Test event', got '%s'", memory.Description)
	}

	if memory.GameTime != 100.0 {
		t.Errorf("Expected game time 100.0, got %f", memory.GameTime)
	}

	if memory.Strength != 0.5 {
		t.Errorf("Expected strength 0.5, got %f", memory.Strength)
	}

	if memory.Emotion != "happy" {
		t.Errorf("Expected emotion 'happy', got '%s'", memory.Emotion)
	}

	if ms.TotalMemories != 1 {
		t.Errorf("Expected total memories 1, got %d", ms.TotalMemories)
	}

	// Check it's in short-term memory
	if len(ms.ShortTermMemories) != 1 {
		t.Errorf("Expected 1 short-term memory, got %d", len(ms.ShortTermMemories))
	}

	// Check it's in the index
	if _, exists := ms.MemoryIndex[memory.ID]; !exists {
		t.Error("Memory not found in index")
	}
}

func TestRecordMemoryFull(t *testing.T) {
	ms := NewMemorySystem(100)

	tags := []string{"test", "event"}
	memory := ms.RecordMemoryFull(MemoryEvent, "Full test", 200.0, 0.7, "joy", nil, 0.8, 0.9, ImportanceSignificant, tags)

	if memory.Valence != 0.8 {
		t.Errorf("Expected valence 0.8, got %f", memory.Valence)
	}

	if memory.Arousal != 0.9 {
		t.Errorf("Expected arousal 0.9, got %f", memory.Arousal)
	}

	if memory.Importance != ImportanceSignificant {
		t.Errorf("Expected ImportanceSignificant, got %v", memory.Importance)
	}

	if len(memory.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(memory.Tags))
	}

	// Check tags are indexed
	tagMemories := ms.SearchByTag("test")
	if len(tagMemories) != 1 {
		t.Errorf("Expected 1 memory with 'test' tag, got %d", len(tagMemories))
	}
}

func TestRecordMemoryFullPositiveType(t *testing.T) {
	ms := NewMemorySystem(100)

	// High positive valence should mark as MemoryPositive
	memory := ms.RecordMemoryFull(MemoryEvent, "Positive event", 100.0, 0.5, "joy", nil, 0.9, 0.5, ImportanceMinor, nil)

	if memory.Type != MemoryPositive {
		t.Errorf("Expected MemoryPositive type for high valence, got %v", memory.Type)
	}
}

func TestRecordMemoryFullTraumaType(t *testing.T) {
	ms := NewMemorySystem(100)

	// Low negative valence should mark as MemoryTrauma
	memory := ms.RecordMemoryFull(MemoryEvent, "Traumatic event", 100.0, 0.5, "fear", nil, -0.9, 0.9, ImportanceMinor, nil)

	if memory.Type != MemoryTrauma {
		t.Errorf("Expected MemoryTrauma type for low valence, got %v", memory.Type)
	}
}

func TestRecordMemoryWithRetentionBonus(t *testing.T) {
	ms := NewMemorySystem(100)
	ms.RetentionBonus = 0.2

	memory := ms.RecordMemory(MemoryEvent, "Test", 100.0, 0.5, "happy", nil)

	// Strength should be 0.5 + 0.2 = 0.7
	if memory.Strength != 0.7 {
		t.Errorf("Expected strength 0.7 with retention bonus, got %f", memory.Strength)
	}
}

func TestRecordInteraction(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordInteraction(types.InteractionFeeding, 100.0, 0.8, "happy")

	if len(ms.ShortTermMemories) != 1 {
		t.Errorf("Expected 1 memory, got %d", len(ms.ShortTermMemories))
	}

	memory := ms.ShortTermMemories[0]
	if memory.Type != MemoryInteraction {
		t.Errorf("Expected MemoryInteraction type, got %v", memory.Type)
	}
}

func TestRecordInteractionFull(t *testing.T) {
	ms := NewMemorySystem(100)

	memory := ms.RecordInteractionFull(types.InteractionPlaying, 100.0, 0.9, "excitement", 0.8, 0.9)

	if memory.Importance != ImportanceSignificant {
		t.Errorf("Expected ImportanceSignificant for high arousal, got %v", memory.Importance)
	}

	// Check tags
	found := false
	for _, tag := range memory.Tags {
		if tag == "interaction" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'interaction' tag")
	}
}

func TestRecordSocialMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	memory := ms.RecordSocialMemory(types.PetID("pet_123"), "meeting", 100.0, 0.8, "happy")

	if memory.Type != MemoryPositive {
		// High quality (0.8) should result in high valence (0.6) which is below positive threshold
		// Actually valence = (0.8 - 0.5) * 2 = 0.6, which is below 0.7
		if memory.Type != MemorySocial {
			t.Errorf("Expected MemorySocial type, got %v", memory.Type)
		}
	}

	// Check social tag
	tagMemories := ms.SearchByTag("social")
	if len(tagMemories) != 1 {
		t.Errorf("Expected 1 social memory, got %d", len(tagMemories))
	}
}

func TestRecordTrainingMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	// Successful training
	memory := ms.RecordTrainingMemory("sit", true, 100.0, "happy")

	if memory.Valence != 0.6 {
		t.Errorf("Expected valence 0.6 for successful training, got %f", memory.Valence)
	}

	// Failed training
	memory2 := ms.RecordTrainingMemory("roll", false, 100.0, "frustrated")

	if memory2.Valence != -0.2 {
		t.Errorf("Expected valence -0.2 for failed training, got %f", memory2.Valence)
	}
}

func TestRecordLocationMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	memory := ms.RecordLocationMemory("park", 100.0, "happy", true)

	if memory.Valence != 0.5 {
		t.Errorf("Expected valence 0.5 for positive location, got %f", memory.Valence)
	}

	// Check location tag
	tagMemories := ms.SearchByTag("location")
	if len(tagMemories) != 1 {
		t.Errorf("Expected 1 location memory, got %d", len(tagMemories))
	}
}

func TestRecordEmotionalMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	memory := ms.RecordEmotionalMemory("Feeling great", 100.0, "joy", 0.9, 0.9)

	if memory.Importance != ImportanceSignificant {
		t.Errorf("Expected ImportanceSignificant for high intensity, got %v", memory.Importance)
	}

	if memory.Type != MemoryPositive {
		t.Errorf("Expected MemoryPositive type, got %v", memory.Type)
	}
}

func TestRecordTraumaticMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	memory := ms.RecordTraumaticMemory("Bad experience", 100.0, nil)

	if memory.Type != MemoryTrauma {
		t.Errorf("Expected MemoryTrauma type, got %v", memory.Type)
	}

	if memory.Strength != 1.0 {
		t.Errorf("Expected strength 1.0 for trauma, got %f", memory.Strength)
	}

	if memory.Importance != ImportanceMajor {
		t.Errorf("Expected ImportanceMajor for trauma, got %v", memory.Importance)
	}
}

func TestRecordPositiveMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	memory := ms.RecordPositiveMemory("Great day", 100.0, "joy", nil)

	if memory.Type != MemoryPositive {
		t.Errorf("Expected MemoryPositive type, got %v", memory.Type)
	}

	if memory.Valence != 0.9 {
		t.Errorf("Expected valence 0.9, got %f", memory.Valence)
	}
}

func TestConsolidateMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	// Create a weak memory (won't consolidate)
	ms.RecordMemory(MemoryEvent, "Weak memory", 100.0, 0.3, "neutral", nil)

	// Create a strong memory (will consolidate)
	ms.RecordMemory(MemoryEvent, "Strong memory", 100.0, 0.8, "happy", nil)

	// Consolidate
	ms.ConsolidateMemories()

	if len(ms.LongTermMemories) != 1 {
		t.Errorf("Expected 1 long-term memory, got %d", len(ms.LongTermMemories))
	}

	if len(ms.ShortTermMemories) != 1 {
		t.Errorf("Expected 1 short-term memory remaining, got %d", len(ms.ShortTermMemories))
	}

	if ms.LongTermMemories[0].Description != "Strong memory" {
		t.Errorf("Expected 'Strong memory' to be consolidated")
	}
}

func TestConsolidateMemoriesWithThreshold(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemory(MemoryEvent, "Memory 1", 100.0, 0.4, "neutral", nil)
	ms.RecordMemory(MemoryEvent, "Memory 2", 100.0, 0.5, "neutral", nil)
	ms.RecordMemory(MemoryEvent, "Memory 3", 100.0, 0.6, "neutral", nil)

	// Lower threshold
	consolidated := ms.ConsolidateMemoriesWithThreshold(0.5)

	if consolidated != 2 {
		t.Errorf("Expected 2 memories consolidated, got %d", consolidated)
	}
}

func TestDecayMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	memory := ms.RecordMemory(MemoryEvent, "Test", 100.0, 0.5, "happy", nil)

	// Decay
	ms.DecayMemories(10.0) // 10 time units

	// Decay rate is 0.01 * deltaTime = 0.1
	expectedStrength := 0.4
	tolerance := 0.001
	if memory.Strength < expectedStrength-tolerance || memory.Strength > expectedStrength+tolerance {
		t.Errorf("Expected strength ~%f after decay, got %f", expectedStrength, memory.Strength)
	}
}

func TestDecayLongTermMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	memory := ms.RecordMemory(MemoryEvent, "Test", 100.0, 0.8, "happy", nil)
	ms.ConsolidateMemories() // Move to long-term

	initialStrength := memory.Strength

	// Decay
	ms.DecayMemories(10.0)

	// Long-term decay rate is 0.01 * deltaTime * 0.1 = 0.01
	expectedDecay := 0.01
	actualDecay := initialStrength - memory.Strength
	tolerance := 0.001

	if actualDecay < expectedDecay-tolerance || actualDecay > expectedDecay+tolerance {
		t.Errorf("Expected decay ~%f for long-term memory, got %f", expectedDecay, actualDecay)
	}
}

func TestPruneWeakMemories(t *testing.T) {
	ms := NewMemorySystem(3) // Small capacity

	ms.RecordMemory(MemoryEvent, "Memory 1", 100.0, 0.5, "neutral", nil)
	ms.RecordMemory(MemoryEvent, "Memory 2", 100.0, 0.3, "neutral", nil) // Weakest
	ms.RecordMemory(MemoryEvent, "Memory 3", 100.0, 0.8, "neutral", nil)
	ms.RecordMemory(MemoryEvent, "Memory 4", 100.0, 0.6, "neutral", nil) // Should trigger prune

	// Should still only have 3 memories
	if len(ms.ShortTermMemories) != 3 {
		t.Errorf("Expected 3 memories after pruning, got %d", len(ms.ShortTermMemories))
	}

	// Memory 2 (weakest) should be removed
	for _, m := range ms.ShortTermMemories {
		if m.Description == "Memory 2" {
			t.Error("Weakest memory should have been pruned")
		}
	}
}

func TestRecallMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemory(MemoryEvent, "Event 1", 100.0, 0.5, "neutral", nil)
	ms.RecordMemory(MemoryInteraction, "Interaction 1", 100.0, 0.5, "happy", nil)
	ms.RecordMemory(MemoryEvent, "Event 2", 100.0, 0.8, "excited", nil)
	ms.ConsolidateMemories() // Event 2 should consolidate

	// Recall events
	events := ms.RecallMemories(MemoryEvent, 10)

	if len(events) != 2 {
		t.Errorf("Expected 2 event memories, got %d", len(events))
	}
}

func TestGetRecentMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemory(MemoryEvent, "Memory 1", 100.0, 0.5, "neutral", nil)
	time.Sleep(10 * time.Millisecond)
	ms.RecordMemory(MemoryEvent, "Memory 2", 100.0, 0.5, "neutral", nil)
	time.Sleep(10 * time.Millisecond)
	ms.RecordMemory(MemoryEvent, "Memory 3", 100.0, 0.5, "neutral", nil)

	recent := ms.GetRecentMemories(2)

	if len(recent) != 2 {
		t.Errorf("Expected 2 recent memories, got %d", len(recent))
	}

	if recent[1].Description != "Memory 3" {
		t.Errorf("Expected most recent to be 'Memory 3', got '%s'", recent[1].Description)
	}
}

func TestGetStrongestMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemory(MemoryEvent, "Weak", 100.0, 0.3, "neutral", nil)
	ms.RecordMemory(MemoryEvent, "Strong", 100.0, 0.9, "happy", nil)
	ms.RecordMemory(MemoryEvent, "Medium", 100.0, 0.6, "neutral", nil)

	strongest := ms.GetStrongestMemories(2)

	if len(strongest) != 2 {
		t.Errorf("Expected 2 strongest memories, got %d", len(strongest))
	}

	if strongest[0].Description != "Strong" {
		t.Errorf("Expected strongest to be 'Strong', got '%s'", strongest[0].Description)
	}
}

func TestGetMemoryCount(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemory(MemoryEvent, "Memory 1", 100.0, 0.5, "neutral", nil)
	ms.RecordMemory(MemoryEvent, "Memory 2", 100.0, 0.8, "happy", nil)
	ms.ConsolidateMemories()

	shortTerm, longTerm := ms.GetMemoryCount()

	if shortTerm != 1 {
		t.Errorf("Expected 1 short-term memory, got %d", shortTerm)
	}

	if longTerm != 1 {
		t.Errorf("Expected 1 long-term memory, got %d", longTerm)
	}
}

func TestHasMemoryOf(t *testing.T) {
	ms := NewMemorySystem(100)

	if ms.HasMemoryOf(MemoryEvent) {
		t.Error("Should not have event memory initially")
	}

	ms.RecordMemory(MemoryEvent, "Event", 100.0, 0.5, "neutral", nil)

	if !ms.HasMemoryOf(MemoryEvent) {
		t.Error("Should have event memory after recording")
	}

	if ms.HasMemoryOf(MemorySocial) {
		t.Error("Should not have social memory")
	}
}

func TestLinkMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	mem1 := ms.RecordMemory(MemoryEvent, "Event 1", 100.0, 0.5, "neutral", nil)
	mem2 := ms.RecordMemory(MemoryEvent, "Event 2", 100.0, 0.5, "neutral", nil)

	err := ms.LinkMemories(mem1.ID, mem2.ID, "related", 0.8)
	if err != nil {
		t.Errorf("Failed to link memories: %v", err)
	}

	// Check associations
	if len(ms.Associations) != 1 {
		t.Errorf("Expected 1 association, got %d", len(ms.Associations))
	}

	// Check linked IDs
	if len(mem1.LinkedIDs) != 1 || mem1.LinkedIDs[0] != mem2.ID {
		t.Error("Memory 1 should link to Memory 2")
	}

	if len(mem2.LinkedIDs) != 1 || mem2.LinkedIDs[0] != mem1.ID {
		t.Error("Memory 2 should link to Memory 1")
	}
}

func TestLinkMemoriesNotFound(t *testing.T) {
	ms := NewMemorySystem(100)

	mem1 := ms.RecordMemory(MemoryEvent, "Event 1", 100.0, 0.5, "neutral", nil)

	err := ms.LinkMemories(mem1.ID, "nonexistent", "related", 0.8)
	if err == nil {
		t.Error("Should fail when memory not found")
	}
}

func TestGetLinkedMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	mem1 := ms.RecordMemory(MemoryEvent, "Event 1", 100.0, 0.5, "neutral", nil)
	mem2 := ms.RecordMemory(MemoryEvent, "Event 2", 100.0, 0.5, "neutral", nil)
	mem3 := ms.RecordMemory(MemoryEvent, "Event 3", 100.0, 0.5, "neutral", nil)

	ms.LinkMemories(mem1.ID, mem2.ID, "related", 0.8)
	ms.LinkMemories(mem1.ID, mem3.ID, "related", 0.5)

	linked := ms.GetLinkedMemories(mem1.ID)

	if len(linked) != 2 {
		t.Errorf("Expected 2 linked memories, got %d", len(linked))
	}
}

func TestWorkingMemory(t *testing.T) {
	ms := NewMemorySystem(100)
	ms.WorkingMemoryCap = 3

	mem1 := ms.RecordMemory(MemoryEvent, "Event 1", 100.0, 0.5, "neutral", nil)
	mem2 := ms.RecordMemory(MemoryEvent, "Event 2", 100.0, 0.5, "neutral", nil)
	mem3 := ms.RecordMemory(MemoryEvent, "Event 3", 100.0, 0.5, "neutral", nil)
	mem4 := ms.RecordMemory(MemoryEvent, "Event 4", 100.0, 0.5, "neutral", nil)

	ms.AddToWorkingMemory(mem1)
	ms.AddToWorkingMemory(mem2)
	ms.AddToWorkingMemory(mem3)

	wm := ms.GetWorkingMemory()
	if len(wm) != 3 {
		t.Errorf("Expected 3 in working memory, got %d", len(wm))
	}

	// Add fourth - should prune first
	ms.AddToWorkingMemory(mem4)

	wm = ms.GetWorkingMemory()
	if len(wm) != 3 {
		t.Errorf("Expected 3 in working memory after overflow, got %d", len(wm))
	}

	// First memory should be gone
	for _, m := range wm {
		if m.ID == mem1.ID {
			t.Error("First memory should have been pruned from working memory")
		}
	}
}

func TestWorkingMemoryRecallStrengthens(t *testing.T) {
	ms := NewMemorySystem(100)

	mem := ms.RecordMemory(MemoryEvent, "Test", 100.0, 0.5, "neutral", nil)
	initialStrength := mem.Strength

	ms.AddToWorkingMemory(mem)

	if mem.RecallCount != 1 {
		t.Errorf("Expected recall count 1, got %d", mem.RecallCount)
	}

	if mem.Strength <= initialStrength {
		t.Error("Memory should be strengthened after recall")
	}
}

func TestClearWorkingMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	mem := ms.RecordMemory(MemoryEvent, "Test", 100.0, 0.5, "neutral", nil)
	ms.AddToWorkingMemory(mem)

	ms.ClearWorkingMemory()

	if len(ms.GetWorkingMemory()) != 0 {
		t.Error("Working memory should be empty after clear")
	}
}

func TestSearchByTag(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemoryFull(MemoryEvent, "Tagged 1", 100.0, 0.5, "neutral", nil, 0.0, 0.5, ImportanceMinor, []string{"test", "alpha"})
	ms.RecordMemoryFull(MemoryEvent, "Tagged 2", 100.0, 0.5, "neutral", nil, 0.0, 0.5, ImportanceMinor, []string{"test", "beta"})
	ms.RecordMemoryFull(MemoryEvent, "Tagged 3", 100.0, 0.5, "neutral", nil, 0.0, 0.5, ImportanceMinor, []string{"other"})

	results := ms.SearchByTag("test")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'test' tag, got %d", len(results))
	}

	results = ms.SearchByTag("alpha")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'alpha' tag, got %d", len(results))
	}

	results = ms.SearchByTag("nonexistent")
	if results != nil {
		t.Error("Expected nil for nonexistent tag")
	}
}

func TestSearchByDescription(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemory(MemoryEvent, "Happy event occurred", 100.0, 0.5, "happy", nil)
	ms.RecordMemory(MemoryEvent, "Sad event happened", 100.0, 0.5, "sad", nil)
	ms.RecordMemory(MemoryEvent, "Another happy moment", 100.0, 0.5, "happy", nil)

	results := ms.SearchByDescription("happy")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'happy', got %d", len(results))
	}

	// Case insensitive
	results = ms.SearchByDescription("HAPPY")
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'HAPPY' (case insensitive), got %d", len(results))
	}
}

func TestGetMemoryByID(t *testing.T) {
	ms := NewMemorySystem(100)

	mem := ms.RecordMemory(MemoryEvent, "Test", 100.0, 0.5, "neutral", nil)

	found, exists := ms.GetMemoryByID(mem.ID)
	if !exists {
		t.Error("Memory should exist")
	}
	if found.Description != "Test" {
		t.Errorf("Expected description 'Test', got '%s'", found.Description)
	}

	_, exists = ms.GetMemoryByID("nonexistent")
	if exists {
		t.Error("Should not find nonexistent memory")
	}
}

func TestRecallWithContext(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemoryFull(MemoryEvent, "Happy event", 100.0, 0.8, "happy", nil, 0.5, 0.5, ImportanceModerate, []string{"fun"})
	ms.RecordMemoryFull(MemoryEvent, "Sad event", 100.0, 0.5, "sad", nil, -0.5, 0.5, ImportanceModerate, []string{"bad"})
	ms.RecordMemoryFull(MemoryEvent, "Fun time", 100.0, 0.6, "happy", nil, 0.3, 0.5, ImportanceModerate, []string{"fun"})

	// Recall with happy emotion and fun tag
	results := ms.RecallWithContext("happy", []string{"fun"}, 2)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Happy event should score highest (strength 0.8, emotion match, tag match)
	if results[0].Description != "Happy event" {
		t.Errorf("Expected 'Happy event' to be first, got '%s'", results[0].Description)
	}
}

func TestGetMemorySummary(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemory(MemoryEvent, "Event 1", 100.0, 0.5, "happy", nil)
	ms.RecordMemory(MemoryInteraction, "Interaction 1", 100.0, 0.8, "happy", nil)
	ms.ConsolidateMemories()

	summary := ms.GetMemorySummary()

	if summary.TotalMemories != 2 {
		t.Errorf("Expected total 2, got %d", summary.TotalMemories)
	}

	if summary.ShortTermCount != 1 {
		t.Errorf("Expected 1 short-term, got %d", summary.ShortTermCount)
	}

	if summary.LongTermCount != 1 {
		t.Errorf("Expected 1 long-term, got %d", summary.LongTermCount)
	}

	if summary.DominantEmotion != "happy" {
		t.Errorf("Expected dominant emotion 'happy', got '%s'", summary.DominantEmotion)
	}
}

func TestGetMemoriesByValence(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemoryFull(MemoryEvent, "Positive", 100.0, 0.5, "happy", nil, 0.8, 0.5, ImportanceMinor, nil)
	ms.RecordMemoryFull(MemoryEvent, "Neutral", 100.0, 0.5, "neutral", nil, 0.0, 0.5, ImportanceMinor, nil)
	ms.RecordMemoryFull(MemoryEvent, "Negative", 100.0, 0.5, "sad", nil, -0.8, 0.5, ImportanceMinor, nil)

	positive := ms.GetMemoriesByValence(0.5, 1.0)
	if len(positive) != 1 {
		t.Errorf("Expected 1 positive memory, got %d", len(positive))
	}

	negative := ms.GetMemoriesByValence(-1.0, -0.5)
	if len(negative) != 1 {
		t.Errorf("Expected 1 negative memory, got %d", len(negative))
	}
}

func TestGetPositiveMemories(t *testing.T) {
	ms := NewMemorySystem(100)
	ms.PositiveThreshold = 0.7

	ms.RecordMemoryFull(MemoryEvent, "Very positive", 100.0, 0.5, "happy", nil, 0.9, 0.5, ImportanceMinor, nil)
	ms.RecordMemoryFull(MemoryEvent, "Slightly positive", 100.0, 0.5, "happy", nil, 0.5, 0.5, ImportanceMinor, nil)

	positive := ms.GetPositiveMemories()
	if len(positive) != 1 {
		t.Errorf("Expected 1 positive memory above threshold, got %d", len(positive))
	}
}

func TestGetNegativeMemories(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemoryFull(MemoryEvent, "Very negative", 100.0, 0.5, "sad", nil, -0.8, 0.5, ImportanceMinor, nil)
	ms.RecordMemoryFull(MemoryEvent, "Slightly negative", 100.0, 0.5, "sad", nil, -0.2, 0.5, ImportanceMinor, nil)
	ms.RecordMemoryFull(MemoryEvent, "Positive", 100.0, 0.5, "happy", nil, 0.5, 0.5, ImportanceMinor, nil)

	negative := ms.GetNegativeMemories()
	if len(negative) != 1 {
		t.Errorf("Expected 1 negative memory below -0.3, got %d", len(negative))
	}
}

func TestGetTraumaticMemories(t *testing.T) {
	ms := NewMemorySystem(100)
	ms.TraumaThreshold = -0.7

	ms.RecordMemoryFull(MemoryEvent, "Traumatic", 100.0, 0.5, "fear", nil, -0.9, 0.5, ImportanceMinor, nil)
	ms.RecordMemoryFull(MemoryEvent, "Bad", 100.0, 0.5, "sad", nil, -0.5, 0.5, ImportanceMinor, nil)

	traumatic := ms.GetTraumaticMemories()
	if len(traumatic) != 1 {
		t.Errorf("Expected 1 traumatic memory, got %d", len(traumatic))
	}
}

func TestForgetMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	mem := ms.RecordMemoryFull(MemoryEvent, "To forget", 100.0, 0.5, "neutral", nil, 0.0, 0.5, ImportanceMinor, []string{"test"})

	// Also add to working memory
	ms.AddToWorkingMemory(mem)

	success := ms.ForgetMemory(mem.ID)
	if !success {
		t.Error("ForgetMemory should return true")
	}

	// Check removed from all places
	if _, exists := ms.MemoryIndex[mem.ID]; exists {
		t.Error("Memory should be removed from index")
	}

	if len(ms.ShortTermMemories) != 0 {
		t.Error("Memory should be removed from short-term")
	}

	if len(ms.GetWorkingMemory()) != 0 {
		t.Error("Memory should be removed from working memory")
	}

	// Tag index should be empty
	if len(ms.TagIndex["test"]) != 0 {
		t.Error("Memory should be removed from tag index")
	}
}

func TestForgetMemoryLongTerm(t *testing.T) {
	ms := NewMemorySystem(100)

	mem := ms.RecordMemory(MemoryEvent, "To consolidate and forget", 100.0, 0.8, "happy", nil)
	ms.ConsolidateMemories()

	success := ms.ForgetMemory(mem.ID)
	if !success {
		t.Error("ForgetMemory should return true for long-term memory")
	}

	if len(ms.LongTermMemories) != 0 {
		t.Error("Memory should be removed from long-term")
	}
}

func TestForgetMemoryNotFound(t *testing.T) {
	ms := NewMemorySystem(100)

	success := ms.ForgetMemory("nonexistent")
	if success {
		t.Error("ForgetMemory should return false for nonexistent memory")
	}
}

func TestStrengthenMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	mem := ms.RecordMemory(MemoryEvent, "Test", 100.0, 0.5, "neutral", nil)

	success := ms.StrengthenMemory(mem.ID, 0.2)
	if !success {
		t.Error("StrengthenMemory should return true")
	}

	if mem.Strength != 0.7 {
		t.Errorf("Expected strength 0.7, got %f", mem.Strength)
	}

	// Test clamping
	ms.StrengthenMemory(mem.ID, 0.5)
	if mem.Strength != 1.0 {
		t.Errorf("Expected strength clamped to 1.0, got %f", mem.Strength)
	}
}

func TestWeakenMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	mem := ms.RecordMemory(MemoryEvent, "Test", 100.0, 0.5, "neutral", nil)

	success := ms.WeakenMemory(mem.ID, 0.2)
	if !success {
		t.Error("WeakenMemory should return true")
	}

	if mem.Strength != 0.3 {
		t.Errorf("Expected strength 0.3, got %f", mem.Strength)
	}

	// Test clamping
	ms.WeakenMemory(mem.ID, 0.5)
	if mem.Strength != 0.0 {
		t.Errorf("Expected strength clamped to 0.0, got %f", mem.Strength)
	}
}

func TestUpdateEmotionalBaseline(t *testing.T) {
	ms := NewMemorySystem(100)

	ms.RecordMemoryFull(MemoryEvent, "Positive", 100.0, 0.5, "happy", nil, 0.8, 0.5, ImportanceMinor, nil)
	ms.RecordMemoryFull(MemoryEvent, "Neutral", 100.0, 0.5, "neutral", nil, 0.0, 0.5, ImportanceMinor, nil)

	ms.UpdateEmotionalBaseline()

	// Baseline should be weighted average of valences
	if ms.EmotionalBaseline <= 0 || ms.EmotionalBaseline >= 0.8 {
		t.Errorf("Expected positive baseline between 0 and 0.8, got %f", ms.EmotionalBaseline)
	}
}

func TestGetEmotionalTendency(t *testing.T) {
	ms := NewMemorySystem(100)

	// Very positive memories
	ms.RecordMemoryFull(MemoryEvent, "Great", 100.0, 0.5, "happy", nil, 0.9, 0.5, ImportanceMinor, nil)
	ms.RecordMemoryFull(MemoryEvent, "Wonderful", 100.0, 0.5, "happy", nil, 0.8, 0.5, ImportanceMinor, nil)

	tendency := ms.GetEmotionalTendency()
	if tendency != "optimistic" {
		t.Errorf("Expected 'optimistic' for high valence, got '%s'", tendency)
	}

	// Reset and add negative memories
	ms2 := NewMemorySystem(100)
	ms2.RecordMemoryFull(MemoryEvent, "Bad", 100.0, 0.5, "sad", nil, -0.8, 0.5, ImportanceMinor, nil)
	ms2.RecordMemoryFull(MemoryEvent, "Terrible", 100.0, 0.5, "sad", nil, -0.7, 0.5, ImportanceMinor, nil)

	tendency = ms2.GetEmotionalTendency()
	if tendency != "pessimistic" {
		t.Errorf("Expected 'pessimistic' for low valence, got '%s'", tendency)
	}
}

func TestRebuildIndexes(t *testing.T) {
	ms := NewMemorySystem(100)

	mem := ms.RecordMemoryFull(MemoryEvent, "Test", 100.0, 0.5, "neutral", nil, 0.0, 0.5, ImportanceMinor, []string{"test"})

	// Clear indexes manually
	ms.MemoryIndex = make(map[string]*Memory)
	ms.TagIndex = make(map[string][]string)

	// Rebuild
	ms.RebuildIndexes()

	// Check memory index restored
	if _, exists := ms.MemoryIndex[mem.ID]; !exists {
		t.Error("Memory should be in index after rebuild")
	}

	// Check tag index restored
	if len(ms.TagIndex["test"]) != 1 {
		t.Error("Tag index should be restored")
	}
}

func TestMemorySystemCopy(t *testing.T) {
	ms := NewMemorySystem(100)

	mem1 := ms.RecordMemoryFull(MemoryEvent, "Event 1", 100.0, 0.8, "happy", nil, 0.5, 0.5, ImportanceModerate, []string{"test"})
	mem2 := ms.RecordMemory(MemoryEvent, "Event 2", 100.0, 0.5, "neutral", nil)
	ms.ConsolidateMemories()
	ms.LinkMemories(mem1.ID, mem2.ID, "related", 0.7)

	// Copy
	copied := ms.Copy()

	// Verify copy is independent
	if copied.MemoryCapacity != ms.MemoryCapacity {
		t.Error("Copy should have same capacity")
	}

	if len(copied.ShortTermMemories) != len(ms.ShortTermMemories) {
		t.Error("Copy should have same short-term memory count")
	}

	if len(copied.LongTermMemories) != len(ms.LongTermMemories) {
		t.Error("Copy should have same long-term memory count")
	}

	if len(copied.Associations) != len(ms.Associations) {
		t.Error("Copy should have same association count")
	}

	// Verify modifications don't affect original
	copied.ShortTermMemories[0].Strength = 0.1
	if ms.ShortTermMemories[0].Strength == 0.1 {
		t.Error("Copy should be independent of original")
	}
}

func TestGenerateMemoryID(t *testing.T) {
	id1 := generateMemoryID(0)
	id2 := generateMemoryID(1)

	if id1 == id2 {
		t.Error("Memory IDs should be unique")
	}

	if len(id1) < 10 {
		t.Error("Memory ID should have reasonable length")
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "foo", false},
		{"", "", true},
		{"abc", "", true},
		{"", "abc", false},
	}

	for _, tt := range tests {
		result := containsIgnoreCase(tt.s, tt.substr)
		if result != tt.expected {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
		}
	}
}

func TestPruneWeakestLongTermMemory(t *testing.T) {
	ms := NewMemorySystem(100)
	ms.LongTermCapacity = 2

	// Create memories with specific strengths
	// Order matters: consolidation processes from end to start
	// mem1 (index 0), mem2 (index 1), mem3 (index 2)
	// Consolidation order: mem3 first, then mem2, then mem1
	mem1 := ms.RecordMemory(MemoryEvent, "First", 100.0, 0.65, "neutral", nil)
	mem2 := ms.RecordMemory(MemoryEvent, "Second", 100.0, 0.75, "happy", nil)
	mem3 := ms.RecordMemory(MemoryEvent, "Third", 100.0, 0.95, "neutral", nil)

	// Manually set strengths to ensure correct ordering
	mem1.Strength = 0.65
	mem2.Strength = 0.75
	mem3.Strength = 0.95

	// Consolidate with a low threshold so all three consolidate
	// Processing order: Third (0.95) -> Second (0.75) -> First (0.65)
	// After Third: LT = [Third]
	// After Second: LT = [Third, Second] (capacity reached)
	// After First: prune weakest (Second at 0.75), add First -> LT = [Third, First]
	ms.ConsolidateMemoriesWithThreshold(0.6)

	// Should have only 2 long-term memories (capacity limit)
	if len(ms.LongTermMemories) != 2 {
		t.Errorf("Expected 2 long-term memories, got %d", len(ms.LongTermMemories))
	}

	// Second (0.75) should have been pruned (it was the weakest when First tried to consolidate)
	for _, m := range ms.LongTermMemories {
		if m.Description == "Second" {
			t.Errorf("Second memory (strength %f) should have been pruned", m.Strength)
		}
	}

	// Third and First should remain
	foundThird := false
	foundFirst := false
	for _, m := range ms.LongTermMemories {
		if m.Description == "Third" {
			foundThird = true
		}
		if m.Description == "First" {
			foundFirst = true
		}
	}

	if !foundThird {
		t.Error("Third memory should be present")
	}
	if !foundFirst {
		t.Error("First memory should be present")
	}
}

func TestConsolidateTraumaticMemory(t *testing.T) {
	ms := NewMemorySystem(100)

	// Even with low strength, trauma should consolidate
	mem := ms.RecordMemoryFull(MemoryTrauma, "Traumatic event", 100.0, 0.3, "fear", nil, -0.9, 0.9, ImportanceMinor, nil)
	mem.Type = MemoryTrauma // Force type

	consolidated := ms.ConsolidateMemoriesWithThreshold(0.6)

	if consolidated != 1 {
		t.Errorf("Expected 1 memory consolidated (trauma), got %d", consolidated)
	}
}

func TestConsolidateSignificantImportance(t *testing.T) {
	ms := NewMemorySystem(100)

	// Low strength but high importance should consolidate
	ms.RecordMemoryFull(MemoryEvent, "Important event", 100.0, 0.3, "neutral", nil, 0.0, 0.5, ImportanceSignificant, nil)

	consolidated := ms.ConsolidateMemoriesWithThreshold(0.6)

	if consolidated != 1 {
		t.Errorf("Expected 1 memory consolidated (significant importance), got %d", consolidated)
	}
}
