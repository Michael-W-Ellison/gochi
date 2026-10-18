package ai

import (
	"errors"
	"sort"
	"strconv"
	"strings"
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
	MemoryEmotional
	MemoryTrauma
	MemoryPositive
)

// String returns the name of the memory type
func (mt MemoryType) String() string {
	switch mt {
	case MemoryInteraction:
		return "Interaction"
	case MemoryEvent:
		return "Event"
	case MemoryLocation:
		return "Location"
	case MemorySocial:
		return "Social"
	case MemoryTraining:
		return "Training"
	case MemoryEmotional:
		return "Emotional"
	case MemoryTrauma:
		return "Trauma"
	case MemoryPositive:
		return "Positive"
	default:
		return "Unknown"
	}
}

// MemoryImportance ranks how significant a memory is
type MemoryImportance int

const (
	ImportanceTrivial MemoryImportance = iota
	ImportanceMinor
	ImportanceModerate
	ImportanceSignificant
	ImportanceMajor
	ImportanceFormative
)

// String returns the name of the importance level
func (mi MemoryImportance) String() string {
	switch mi {
	case ImportanceTrivial:
		return "Trivial"
	case ImportanceMinor:
		return "Minor"
	case ImportanceModerate:
		return "Moderate"
	case ImportanceSignificant:
		return "Significant"
	case ImportanceMajor:
		return "Major"
	case ImportanceFormative:
		return "Formative"
	default:
		return "Unknown"
	}
}

// Memory represents a single memory stored by the pet
type Memory struct {
	ID           string
	Type         MemoryType
	Description  string
	Timestamp    time.Time
	GameTime     float64 // Game time when memory was formed
	Strength     float64 // How strong/important the memory is (0.0 to 1.0)
	Emotion      string  // Associated emotional state
	Details      map[string]interface{}
	IsLongTerm   bool             // Whether this has been consolidated to long-term memory
	Valence      float64          // Emotional valence (-1.0 negative to 1.0 positive)
	Arousal      float64          // Emotional intensity (0.0 to 1.0)
	Importance   MemoryImportance // How significant the memory is
	Tags         []string         // Searchable tags
	LinkedIDs    []string         // IDs of associated memories
	RecallCount  int              // Times the memory has been recalled
	LastRecalled time.Time        // When the memory was last recalled
}

// MemoryAssociation links two memories together
type MemoryAssociation struct {
	FromID   string
	ToID     string
	Type     string  // Kind of association (e.g. "related", "cause")
	Strength float64 // Association strength (0.0 to 1.0)
}

// MemorySummary provides an overview of the memory system
type MemorySummary struct {
	TotalMemories      int
	ShortTermCount     int
	LongTermCount      int
	WorkingMemoryCount int
	TraumaticCount     int
	PositiveCount      int
	AverageValence     float64
	DominantEmotion    string
	EmotionalTendency  string
}

// MemorySystem manages both short-term and long-term memories
type MemorySystem struct {
	ShortTermMemories []*Memory           // Recent memories (last ~100)
	LongTermMemories  []*Memory           // Important/consolidated memories
	MemoryCapacity    int                 // Max short-term memories
	LongTermCapacity  int                 // Max long-term memories
	ConsolidationRate float64             // How often memories are consolidated
	TotalMemories     int                 // Total memories ever formed
	MemoryIndex       map[string]*Memory  // Quick lookup by ID
	TagIndex          map[string][]string // Memory IDs by tag
	Associations      []MemoryAssociation // Links between memories
	WorkingMemory     []*Memory           `json:"-"` // Currently active memories
	WorkingMemoryCap  int                 // Max working memory items
	TraumaThreshold   float64             // Valence at or below which a memory is traumatic
	PositiveThreshold float64             // Valence at or above which a memory is positive
	RetentionBonus    float64             // Strength bonus applied to new memories
	EmotionalBaseline float64             // Strength-weighted average valence of memories
}

// NewMemorySystem creates a new memory system
func NewMemorySystem(capacity int) *MemorySystem {
	return &MemorySystem{
		ShortTermMemories: make([]*Memory, 0, capacity),
		LongTermMemories:  make([]*Memory, 0),
		MemoryCapacity:    capacity,
		LongTermCapacity:  500,
		ConsolidationRate: 0.1,
		MemoryIndex:       make(map[string]*Memory),
		TagIndex:          make(map[string][]string),
		Associations:      make([]MemoryAssociation, 0),
		WorkingMemory:     make([]*Memory, 0),
		WorkingMemoryCap:  7,
		TraumaThreshold:   -0.7,
		PositiveThreshold: 0.7,
	}
}

// NewMemorySystemWithPersonality creates a memory system tuned by personality:
// openness widens working memory (5-9 items), conscientiousness adds retention (0.0-0.2)
func NewMemorySystemWithPersonality(capacity int, openness, conscientiousness float64) *MemorySystem {
	m := NewMemorySystem(capacity)
	m.WorkingMemoryCap = 5 + int(clamp(openness, 0.0, 1.0)*4.0+0.5)
	m.RetentionBonus = clamp(conscientiousness, 0.0, 1.0) * 0.2
	return m
}

// RecordMemory adds a new memory to short-term storage
func (m *MemorySystem) RecordMemory(memType MemoryType, description string, gameTime float64, strength float64, emotion string, details map[string]interface{}) *Memory {
	return m.RecordMemoryFull(memType, description, gameTime, strength, emotion, details, 0.0, 0.5, ImportanceMinor, nil)
}

// RecordMemoryFull adds a new memory with emotional and indexing metadata.
// Memories whose valence crosses the positive or trauma threshold are
// recorded as MemoryPositive or MemoryTrauma respectively.
func (m *MemorySystem) RecordMemoryFull(memType MemoryType, description string, gameTime float64, strength float64, emotion string, details map[string]interface{}, valence, arousal float64, importance MemoryImportance, tags []string) *Memory {
	valence = clamp(valence, -1.0, 1.0)

	switch {
	case valence <= m.TraumaThreshold:
		memType = MemoryTrauma
	case valence >= m.PositiveThreshold:
		memType = MemoryPositive
	}

	memory := &Memory{
		ID:          generateMemoryID(m.TotalMemories),
		Type:        memType,
		Description: description,
		Timestamp:   time.Now(),
		GameTime:    gameTime,
		Strength:    clamp(strength+m.RetentionBonus, 0.0, 1.0),
		Emotion:     emotion,
		Details:     details,
		IsLongTerm:  false,
		Valence:     valence,
		Arousal:     clamp(arousal, 0.0, 1.0),
		Importance:  importance,
		Tags:        append([]string(nil), tags...),
	}

	m.ShortTermMemories = append(m.ShortTermMemories, memory)
	m.MemoryIndex[memory.ID] = memory
	m.indexTags(memory)
	m.TotalMemories++

	// If over capacity, remove weakest memories
	if len(m.ShortTermMemories) > m.MemoryCapacity {
		m.pruneWeakMemories()
	}

	return memory
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

// RecordInteractionFull records a user interaction with emotional metadata
func (m *MemorySystem) RecordInteractionFull(interactionType types.InteractionType, gameTime float64, quality float64, emotion string, valence, arousal float64) *Memory {
	details := map[string]interface{}{
		"interaction_type": interactionType.String(),
		"quality":          quality,
	}
	strength := 0.3 + (quality * 0.7)
	tags := []string{"interaction", interactionType.String()}

	return m.RecordMemoryFull(MemoryInteraction, "User interaction: "+interactionType.String(), gameTime, strength, emotion, details, valence, arousal, importanceFromIntensity(arousal), tags)
}

// RecordSocialMemory records an encounter with another pet.
// Quality (0.0 to 1.0) maps to valence (-1.0 to 1.0).
func (m *MemorySystem) RecordSocialMemory(otherPetID types.PetID, interactionType string, gameTime float64, quality float64, emotion string) *Memory {
	details := map[string]interface{}{
		"pet_id":           string(otherPetID),
		"interaction_type": interactionType,
		"quality":          quality,
	}
	valence := (quality - 0.5) * 2.0
	tags := []string{"social", interactionType, string(otherPetID)}

	return m.RecordMemoryFull(MemorySocial, "Social "+interactionType+" with "+string(otherPetID), gameTime, 0.3+quality*0.5, emotion, details, valence, 0.5, ImportanceModerate, tags)
}

// RecordTrainingMemory records the outcome of a training session
func (m *MemorySystem) RecordTrainingMemory(skill string, success bool, gameTime float64, emotion string) *Memory {
	details := map[string]interface{}{
		"skill":   skill,
		"success": success,
	}

	strength, valence, outcome := 0.6, 0.6, "succeeded"
	if !success {
		strength, valence, outcome = 0.4, -0.2, "failed"
	}

	return m.RecordMemoryFull(MemoryTraining, "Training "+skill+" "+outcome, gameTime, strength, emotion, details, valence, 0.5, ImportanceModerate, []string{"training", skill})
}

// RecordLocationMemory records an experience at a location
func (m *MemorySystem) RecordLocationMemory(location string, gameTime float64, emotion string, positive bool) *Memory {
	details := map[string]interface{}{
		"location": location,
	}

	valence := 0.5
	if !positive {
		valence = -0.3
	}

	return m.RecordMemoryFull(MemoryLocation, "Visited "+location, gameTime, 0.5, emotion, details, valence, 0.4, ImportanceMinor, []string{"location", location})
}

// RecordEmotionalMemory records a notable emotional experience
func (m *MemorySystem) RecordEmotionalMemory(description string, gameTime float64, emotion string, intensity, valence float64) *Memory {
	return m.RecordMemoryFull(MemoryEmotional, description, gameTime, intensity, emotion, nil, valence, intensity, importanceFromIntensity(intensity), []string{"emotional", emotion})
}

// RecordTraumaticMemory records a traumatic experience at full strength
func (m *MemorySystem) RecordTraumaticMemory(description string, gameTime float64, details map[string]interface{}) *Memory {
	memory := m.RecordMemoryFull(MemoryTrauma, description, gameTime, 1.0, "fear", details, -1.0, 1.0, ImportanceMajor, []string{"trauma"})
	memory.Type = MemoryTrauma
	return memory
}

// RecordPositiveMemory records a strongly positive experience
func (m *MemorySystem) RecordPositiveMemory(description string, gameTime float64, emotion string, details map[string]interface{}) *Memory {
	memory := m.RecordMemoryFull(MemoryPositive, description, gameTime, 0.8, emotion, details, 0.9, 0.7, ImportanceModerate, []string{"positive"})
	memory.Type = MemoryPositive
	return memory
}

// importanceFromIntensity maps an emotional intensity to an importance level
func importanceFromIntensity(intensity float64) MemoryImportance {
	switch {
	case intensity >= 0.8:
		return ImportanceSignificant
	case intensity >= 0.5:
		return ImportanceModerate
	default:
		return ImportanceMinor
	}
}

// ConsolidateMemories moves important short-term memories to long-term storage
func (m *MemorySystem) ConsolidateMemories() {
	m.ConsolidateMemoriesWithThreshold(0.6) // Memories must be this strong to consolidate
}

// ConsolidateMemoriesWithThreshold moves short-term memories that are at least
// threshold strong, traumatic, or of significant importance to long-term storage.
// Returns the number of memories consolidated.
func (m *MemorySystem) ConsolidateMemoriesWithThreshold(threshold float64) int {
	consolidated := 0

	for i := len(m.ShortTermMemories) - 1; i >= 0; i-- {
		memory := m.ShortTermMemories[i]

		qualifies := memory.Strength >= threshold ||
			memory.Type == MemoryTrauma ||
			memory.Importance >= ImportanceSignificant

		if qualifies && !memory.IsLongTerm {
			// Make room in long-term memory if needed
			if m.LongTermCapacity > 0 && len(m.LongTermMemories) >= m.LongTermCapacity {
				m.pruneWeakestLongTermMemory()
			}

			// Move to long-term memory
			memory.IsLongTerm = true
			m.LongTermMemories = append(m.LongTermMemories, memory)

			// Remove from short-term
			m.ShortTermMemories = append(m.ShortTermMemories[:i], m.ShortTermMemories[i+1:]...)
			consolidated++
		}
	}

	return consolidated
}

// pruneWeakestLongTermMemory removes the weakest long-term memory
func (m *MemorySystem) pruneWeakestLongTermMemory() {
	if len(m.LongTermMemories) == 0 {
		return
	}

	weakestIdx := 0
	for i, memory := range m.LongTermMemories {
		if memory.Strength < m.LongTermMemories[weakestIdx].Strength {
			weakestIdx = i
		}
	}

	m.ForgetMemory(m.LongTermMemories[weakestIdx].ID)
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
	m.ForgetMemory(m.ShortTermMemories[weakestIdx].ID)
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

// LinkMemories creates a bidirectional association between two memories
func (m *MemorySystem) LinkMemories(fromID, toID, associationType string, strength float64) error {
	from, ok := m.MemoryIndex[fromID]
	if !ok {
		return errors.New("memory not found: " + fromID)
	}
	to, ok := m.MemoryIndex[toID]
	if !ok {
		return errors.New("memory not found: " + toID)
	}

	m.Associations = append(m.Associations, MemoryAssociation{
		FromID:   fromID,
		ToID:     toID,
		Type:     associationType,
		Strength: clamp(strength, 0.0, 1.0),
	})
	from.LinkedIDs = append(from.LinkedIDs, toID)
	to.LinkedIDs = append(to.LinkedIDs, fromID)

	return nil
}

// GetLinkedMemories returns memories associated with the given memory
func (m *MemorySystem) GetLinkedMemories(memoryID string) []*Memory {
	memory, ok := m.MemoryIndex[memoryID]
	if !ok {
		return nil
	}

	var linked []*Memory
	for _, id := range memory.LinkedIDs {
		if other, ok := m.MemoryIndex[id]; ok {
			linked = append(linked, other)
		}
	}
	return linked
}

// AddToWorkingMemory brings a memory into working memory, counting it as a
// recall that slightly strengthens it. The oldest item is dropped when full.
func (m *MemorySystem) AddToWorkingMemory(memory *Memory) {
	if memory == nil {
		return
	}

	memory.RecallCount++
	memory.LastRecalled = time.Now()
	memory.Strength = clamp(memory.Strength+0.05, 0.0, 1.0)

	m.removeFromWorkingMemory(memory.ID)
	m.WorkingMemory = append(m.WorkingMemory, memory)

	if m.WorkingMemoryCap > 0 && len(m.WorkingMemory) > m.WorkingMemoryCap {
		m.WorkingMemory = m.WorkingMemory[len(m.WorkingMemory)-m.WorkingMemoryCap:]
	}
}

// GetWorkingMemory returns the memories currently in working memory
func (m *MemorySystem) GetWorkingMemory() []*Memory {
	return append([]*Memory(nil), m.WorkingMemory...)
}

// ClearWorkingMemory empties working memory
func (m *MemorySystem) ClearWorkingMemory() {
	m.WorkingMemory = make([]*Memory, 0)
}

// removeFromWorkingMemory removes a memory from working memory by ID
func (m *MemorySystem) removeFromWorkingMemory(memoryID string) {
	for i, memory := range m.WorkingMemory {
		if memory.ID == memoryID {
			m.WorkingMemory = append(m.WorkingMemory[:i], m.WorkingMemory[i+1:]...)
			return
		}
	}
}

// allMemories returns long-term followed by short-term memories
func (m *MemorySystem) allMemories() []*Memory {
	all := make([]*Memory, 0, len(m.LongTermMemories)+len(m.ShortTermMemories))
	all = append(all, m.LongTermMemories...)
	return append(all, m.ShortTermMemories...)
}

// SearchByTag returns memories with the given tag, or nil if none
func (m *MemorySystem) SearchByTag(tag string) []*Memory {
	var results []*Memory
	for _, id := range m.TagIndex[tag] {
		if memory, ok := m.MemoryIndex[id]; ok {
			results = append(results, memory)
		}
	}
	return results
}

// SearchByDescription returns memories whose description contains query (case-insensitive)
func (m *MemorySystem) SearchByDescription(query string) []*Memory {
	var results []*Memory
	for _, memory := range m.allMemories() {
		if containsIgnoreCase(memory.Description, query) {
			results = append(results, memory)
		}
	}
	return results
}

// GetMemoryByID returns a memory by its ID
func (m *MemorySystem) GetMemoryByID(memoryID string) (*Memory, bool) {
	memory, ok := m.MemoryIndex[memoryID]
	return memory, ok
}

// RecallWithContext returns up to limit memories ranked by strength plus
// bonuses for matching the given emotion and tags
func (m *MemorySystem) RecallWithContext(emotion string, tags []string, limit int) []*Memory {
	type scored struct {
		memory *Memory
		score  float64
	}

	var candidates []scored
	for _, memory := range m.allMemories() {
		score := memory.Strength
		if emotion != "" && memory.Emotion == emotion {
			score += 0.3
		}
		for _, tag := range tags {
			for _, memTag := range memory.Tags {
				if memTag == tag {
					score += 0.2
					break
				}
			}
		}
		candidates = append(candidates, scored{memory, score})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if limit >= 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	results := make([]*Memory, len(candidates))
	for i, c := range candidates {
		results[i] = c.memory
	}
	return results
}

// GetMemorySummary returns an overview of the memory system
func (m *MemorySystem) GetMemorySummary() MemorySummary {
	all := m.allMemories()
	summary := MemorySummary{
		TotalMemories:      m.TotalMemories,
		ShortTermCount:     len(m.ShortTermMemories),
		LongTermCount:      len(m.LongTermMemories),
		WorkingMemoryCount: len(m.WorkingMemory),
		TraumaticCount:     len(m.GetTraumaticMemories()),
		PositiveCount:      len(m.GetPositiveMemories()),
		EmotionalTendency:  m.GetEmotionalTendency(),
	}

	emotionCounts := make(map[string]int)
	totalValence := 0.0
	for _, memory := range all {
		totalValence += memory.Valence
		if memory.Emotion != "" {
			emotionCounts[memory.Emotion]++
		}
	}
	if len(all) > 0 {
		summary.AverageValence = totalValence / float64(len(all))
	}

	// Most frequent emotion; ties broken alphabetically for determinism
	for emotion, count := range emotionCounts {
		best := emotionCounts[summary.DominantEmotion]
		if count > best || (count == best && emotion < summary.DominantEmotion) {
			summary.DominantEmotion = emotion
		}
	}

	return summary
}

// GetMemoriesByValence returns memories with valence in [minValence, maxValence]
func (m *MemorySystem) GetMemoriesByValence(minValence, maxValence float64) []*Memory {
	var results []*Memory
	for _, memory := range m.allMemories() {
		if memory.Valence >= minValence && memory.Valence <= maxValence {
			results = append(results, memory)
		}
	}
	return results
}

// GetPositiveMemories returns memories at or above the positive threshold
func (m *MemorySystem) GetPositiveMemories() []*Memory {
	return m.GetMemoriesByValence(m.PositiveThreshold, 1.0)
}

// GetNegativeMemories returns memories with valence below -0.3
func (m *MemorySystem) GetNegativeMemories() []*Memory {
	var results []*Memory
	for _, memory := range m.allMemories() {
		if memory.Valence < -0.3 {
			results = append(results, memory)
		}
	}
	return results
}

// GetTraumaticMemories returns trauma memories and those at or below the trauma threshold
func (m *MemorySystem) GetTraumaticMemories() []*Memory {
	var results []*Memory
	for _, memory := range m.allMemories() {
		if memory.Type == MemoryTrauma || memory.Valence <= m.TraumaThreshold {
			results = append(results, memory)
		}
	}
	return results
}

// ForgetMemory removes a memory from all storage, indexes and associations
func (m *MemorySystem) ForgetMemory(memoryID string) bool {
	memory, ok := m.MemoryIndex[memoryID]
	if !ok {
		return false
	}

	m.ShortTermMemories = removeMemoryByID(m.ShortTermMemories, memoryID)
	m.LongTermMemories = removeMemoryByID(m.LongTermMemories, memoryID)
	m.removeFromWorkingMemory(memoryID)
	delete(m.MemoryIndex, memoryID)

	for _, tag := range memory.Tags {
		ids := removeString(m.TagIndex[tag], memoryID)
		if len(ids) == 0 {
			delete(m.TagIndex, tag)
		} else {
			m.TagIndex[tag] = ids
		}
	}

	for _, linkedID := range memory.LinkedIDs {
		if linked, ok := m.MemoryIndex[linkedID]; ok {
			linked.LinkedIDs = removeString(linked.LinkedIDs, memoryID)
		}
	}

	associations := m.Associations[:0]
	for _, a := range m.Associations {
		if a.FromID != memoryID && a.ToID != memoryID {
			associations = append(associations, a)
		}
	}
	m.Associations = associations

	return true
}

// StrengthenMemory increases a memory's strength by amount
func (m *MemorySystem) StrengthenMemory(memoryID string, amount float64) bool {
	memory, ok := m.MemoryIndex[memoryID]
	if !ok {
		return false
	}
	memory.Strength = clamp(memory.Strength+amount, 0.0, 1.0)
	return true
}

// WeakenMemory decreases a memory's strength by amount
func (m *MemorySystem) WeakenMemory(memoryID string, amount float64) bool {
	memory, ok := m.MemoryIndex[memoryID]
	if !ok {
		return false
	}
	memory.Strength = clamp(memory.Strength-amount, 0.0, 1.0)
	return true
}

// UpdateEmotionalBaseline recalculates the strength-weighted average valence
func (m *MemorySystem) UpdateEmotionalBaseline() {
	m.EmotionalBaseline = m.calculateEmotionalBaseline()
}

// calculateEmotionalBaseline returns the strength-weighted average valence
func (m *MemorySystem) calculateEmotionalBaseline() float64 {
	totalWeight, weightedValence := 0.0, 0.0
	for _, memory := range m.allMemories() {
		totalWeight += memory.Strength
		weightedValence += memory.Valence * memory.Strength
	}
	if totalWeight == 0 {
		return 0
	}
	return weightedValence / totalWeight
}

// GetEmotionalTendency describes the overall emotional tone of memories
func (m *MemorySystem) GetEmotionalTendency() string {
	baseline := m.calculateEmotionalBaseline()
	switch {
	case baseline > 0.3:
		return "optimistic"
	case baseline < -0.3:
		return "pessimistic"
	default:
		return "balanced"
	}
}

// RebuildIndexes reconstructs the ID and tag indexes from stored memories
// (for example after deserialization)
func (m *MemorySystem) RebuildIndexes() {
	m.MemoryIndex = make(map[string]*Memory)
	m.TagIndex = make(map[string][]string)
	for _, memory := range m.allMemories() {
		m.MemoryIndex[memory.ID] = memory
		m.indexTags(memory)
	}
}

// indexTags adds a memory's tags to the tag index
func (m *MemorySystem) indexTags(memory *Memory) {
	if m.TagIndex == nil {
		m.TagIndex = make(map[string][]string)
	}
	for _, tag := range memory.Tags {
		m.TagIndex[tag] = append(m.TagIndex[tag], memory.ID)
	}
}

// Copy returns a deep copy of the memory system
func (m *MemorySystem) Copy() *MemorySystem {
	c := *m

	copyMemory := func(memory *Memory) *Memory {
		mc := *memory
		mc.Tags = append([]string(nil), memory.Tags...)
		mc.LinkedIDs = append([]string(nil), memory.LinkedIDs...)
		if memory.Details != nil {
			mc.Details = make(map[string]interface{}, len(memory.Details))
			for k, v := range memory.Details {
				mc.Details[k] = v
			}
		}
		return &mc
	}

	c.ShortTermMemories = make([]*Memory, 0, cap(m.ShortTermMemories))
	for _, memory := range m.ShortTermMemories {
		c.ShortTermMemories = append(c.ShortTermMemories, copyMemory(memory))
	}
	c.LongTermMemories = make([]*Memory, 0, len(m.LongTermMemories))
	for _, memory := range m.LongTermMemories {
		c.LongTermMemories = append(c.LongTermMemories, copyMemory(memory))
	}
	c.Associations = append([]MemoryAssociation(nil), m.Associations...)

	c.RebuildIndexes()

	c.WorkingMemory = make([]*Memory, 0, len(m.WorkingMemory))
	for _, memory := range m.WorkingMemory {
		if copied, ok := c.MemoryIndex[memory.ID]; ok {
			c.WorkingMemory = append(c.WorkingMemory, copied)
		}
	}

	return &c
}

// removeMemoryByID returns memories without the memory with the given ID
func removeMemoryByID(memories []*Memory, memoryID string) []*Memory {
	for i, memory := range memories {
		if memory.ID == memoryID {
			return append(memories[:i], memories[i+1:]...)
		}
	}
	return memories
}

// removeString returns values without the first occurrence of target
func removeString(values []string, target string) []string {
	for i, v := range values {
		if v == target {
			return append(values[:i], values[i+1:]...)
		}
	}
	return values
}

// containsIgnoreCase reports whether substr is within s, ignoring case
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Helper function to generate unique memory IDs.
// The count is the system's TotalMemories, which only increases, so IDs
// within one memory system never repeat.
func generateMemoryID(count int) string {
	return "mem_" + time.Now().Format("20060102150405") + "_" + strconv.Itoa(count)
}
