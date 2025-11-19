package ai

import (
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// MemoryType categorizes different types of memories
type MemoryType int

const (
	MemoryInteraction MemoryType = iota
	MemoryEvent
	MemoryLocation
	MemorySocial
	MemoryTraining
)

// Memory represents a single memory stored by the pet
type Memory struct {
	ID          string
	Type        MemoryType
	Description string
	Timestamp   time.Time
	GameTime    float64 // Game time when memory was formed
	Strength    float64 // How strong/important the memory is (0.0 to 1.0)
	Emotion     string  // Associated emotional state
	Details     map[string]interface{}
	IsLongTerm  bool // Whether this has been consolidated to long-term memory
}

// MemorySystem manages both short-term and long-term memories
type MemorySystem struct {
	ShortTermMemories []*Memory          // Recent memories (last ~100)
	LongTermMemories  []*Memory          // Important/consolidated memories
	MemoryCapacity    int                // Max short-term memories
	ConsolidationRate float64            // How often memories are consolidated
	TotalMemories     int                // Total memories ever formed
	MemoryIndex       map[string]*Memory // Quick lookup by ID
}

// NewMemorySystem creates a new memory system
func NewMemorySystem(capacity int) *MemorySystem {
	return &MemorySystem{
		ShortTermMemories: make([]*Memory, 0, capacity),
		LongTermMemories:  make([]*Memory, 0),
		MemoryCapacity:    capacity,
		ConsolidationRate: 0.1,
		MemoryIndex:       make(map[string]*Memory),
	}
}

// RecordMemory adds a new memory to short-term storage
func (m *MemorySystem) RecordMemory(memType MemoryType, description string, gameTime float64, strength float64, emotion string, details map[string]interface{}) {
	memory := &Memory{
		ID:          generateMemoryID(m.TotalMemories),
		Type:        memType,
		Description: description,
		Timestamp:   time.Now(),
		GameTime:    gameTime,
		Strength:    clamp(strength, 0.0, 1.0),
		Emotion:     emotion,
		Details:     details,
		IsLongTerm:  false,
	}

	m.ShortTermMemories = append(m.ShortTermMemories, memory)
	m.MemoryIndex[memory.ID] = memory
	m.TotalMemories++

	// If over capacity, remove weakest memories
	if len(m.ShortTermMemories) > m.MemoryCapacity {
		m.pruneWeakMemories()
	}
}

// RecordInteraction records a user interaction as a memory
func (m *MemorySystem) RecordInteraction(interactionType types.InteractionType, gameTime float64, quality float64, emotion string) {
	details := map[string]interface{}{
		"interaction_type": interactionType.String(),
		"quality":          quality,
	}

	// Interactions are more memorable if they're very positive or very negative
	strength := 0.3 + (quality * 0.7)

	m.RecordMemory(MemoryInteraction, "User interaction: "+interactionType.String(), gameTime, strength, emotion, details)
}

// ConsolidateMemories moves important short-term memories to long-term storage
func (m *MemorySystem) ConsolidateMemories() {
	consolidationThreshold := 0.6 // Memories must be this strong to consolidate

	for i := len(m.ShortTermMemories) - 1; i >= 0; i-- {
		memory := m.ShortTermMemories[i]

		if memory.Strength >= consolidationThreshold && !memory.IsLongTerm {
			// Move to long-term memory
			memory.IsLongTerm = true
			m.LongTermMemories = append(m.LongTermMemories, memory)

			// Remove from short-term
			m.ShortTermMemories = append(m.ShortTermMemories[:i], m.ShortTermMemories[i+1:]...)
		}
	}
}

// DecayMemories weakens memories over time
func (m *MemorySystem) DecayMemories(deltaTime float64) {
	decayRate := 0.01 * deltaTime

	// Decay short-term memories
	for _, memory := range m.ShortTermMemories {
		memory.Strength -= decayRate
		if memory.Strength < 0 {
			memory.Strength = 0
		}
	}

	// Long-term memories decay much slower
	longTermDecayRate := decayRate * 0.1
	for _, memory := range m.LongTermMemories {
		memory.Strength -= longTermDecayRate
		if memory.Strength < 0 {
			memory.Strength = 0
		}
	}
}

// pruneWeakMemories removes the weakest memories when over capacity
func (m *MemorySystem) pruneWeakMemories() {
	// Find weakest memory
	weakestIdx := 0
	weakestStrength := m.ShortTermMemories[0].Strength

	for i, memory := range m.ShortTermMemories {
		if memory.Strength < weakestStrength {
			weakestIdx = i
			weakestStrength = memory.Strength
		}
	}

	// Remove weakest memory
	removed := m.ShortTermMemories[weakestIdx]
	delete(m.MemoryIndex, removed.ID)
	m.ShortTermMemories = append(m.ShortTermMemories[:weakestIdx], m.ShortTermMemories[weakestIdx+1:]...)
}

// RecallMemories retrieves memories matching criteria
func (m *MemorySystem) RecallMemories(memType MemoryType, limit int) []*Memory {
	var recalled []*Memory

	// Search long-term memories first (stronger/more important)
	for _, memory := range m.LongTermMemories {
		if memory.Type == memType {
			recalled = append(recalled, memory)
			if len(recalled) >= limit {
				return recalled
			}
		}
	}

	// Then search short-term memories
	for _, memory := range m.ShortTermMemories {
		if memory.Type == memType {
			recalled = append(recalled, memory)
			if len(recalled) >= limit {
				return recalled
			}
		}
	}

	return recalled
}

// GetRecentMemories returns the N most recent memories
func (m *MemorySystem) GetRecentMemories(count int) []*Memory {
	if count > len(m.ShortTermMemories) {
		count = len(m.ShortTermMemories)
	}

	// Return last N memories (most recent)
	startIdx := len(m.ShortTermMemories) - count
	if startIdx < 0 {
		startIdx = 0
	}

	return m.ShortTermMemories[startIdx:]
}

// GetStrongestMemories returns the N strongest memories
func (m *MemorySystem) GetStrongestMemories(count int) []*Memory {
	allMemories := append(m.ShortTermMemories, m.LongTermMemories...)

	// Simple selection of strongest memories
	var strongest []*Memory
	for i := 0; i < count && i < len(allMemories); i++ {
		maxIdx := -1
		maxStrength := 0.0

		for j, memory := range allMemories {
			// Skip already selected
			alreadySelected := false
			for _, s := range strongest {
				if s.ID == memory.ID {
					alreadySelected = true
					break
				}
			}

			if !alreadySelected && memory.Strength > maxStrength {
				maxIdx = j
				maxStrength = memory.Strength
			}
		}

		if maxIdx >= 0 {
			strongest = append(strongest, allMemories[maxIdx])
		}
	}

	return strongest
}

// GetMemoryCount returns the count of memories by type
func (m *MemorySystem) GetMemoryCount() (shortTerm, longTerm int) {
	return len(m.ShortTermMemories), len(m.LongTermMemories)
}

// HasMemoryOf checks if the pet has any memory of a specific type
func (m *MemorySystem) HasMemoryOf(memType MemoryType) bool {
	for _, memory := range m.ShortTermMemories {
		if memory.Type == memType {
			return true
		}
	}
	for _, memory := range m.LongTermMemories {
		if memory.Type == memType {
			return true
		}
	}
	return false
}

// Helper function to generate unique memory IDs
func generateMemoryID(count int) string {
	return time.Now().Format("20060102150405") + string(rune(count))
}
