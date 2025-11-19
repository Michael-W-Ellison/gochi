package simulation

import (
	"sort"
	"sync"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Priority represents the importance level of needs
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// String returns the string representation of Priority
func (p Priority) String() string {
	return [...]string{"Low", "Medium", "High", "Critical"}[p]
}

// Need represents a single need with its current state and parameters
type Need struct {
	Type              types.NeedType
	CurrentLevel      float64  // 0.0 = critical (need is high), 1.0 = satisfied (need is low)
	DecayRate         float64  // How quickly need level decreases per game day
	CriticalThreshold float64  // When need becomes urgent
	WarningThreshold  float64  // When need starts to be concerning
	SatisfactionRate  float64  // How quickly need is fulfilled when addressed
	Priority          Priority // Impact on overall well-being
	LastSatisfied     float64  // Game time when last satisfied
}

// NewNeed creates a new need with default parameters
func NewNeed(needType types.NeedType) *Need {
	need := &Need{
		Type:              needType,
		CurrentLevel:      1.0, // Start satisfied
		CriticalThreshold: 0.2,
		WarningThreshold:  0.4,
		LastSatisfied:     0.0,
	}

	// Set specific parameters based on need type
	switch needType {
	case types.NeedHunger:
		need.DecayRate = 0.3 // Decays faster
		need.SatisfactionRate = 0.5
		need.Priority = PriorityHigh

	case types.NeedThirst:
		need.DecayRate = 0.5 // Decays fastest
		need.SatisfactionRate = 0.6
		need.Priority = PriorityCritical

	case types.NeedSleep:
		need.DecayRate = 0.25
		need.SatisfactionRate = 0.4
		need.Priority = PriorityHigh

	case types.NeedExercise:
		need.DecayRate = 0.15
		need.SatisfactionRate = 0.3
		need.Priority = PriorityMedium

	case types.NeedSocial:
		need.DecayRate = 0.1
		need.SatisfactionRate = 0.25
		need.Priority = PriorityMedium

	case types.NeedMentalStimulation:
		need.DecayRate = 0.12
		need.SatisfactionRate = 0.3
		need.Priority = PriorityMedium

	case types.NeedAffection:
		need.DecayRate = 0.08
		need.SatisfactionRate = 0.35
		need.Priority = PriorityMedium

	case types.NeedCleanliness:
		need.DecayRate = 0.05
		need.SatisfactionRate = 0.5
		need.Priority = PriorityLow

	case types.NeedMedicalCare:
		need.DecayRate = 0.02
		need.SatisfactionRate = 0.4
		need.Priority = PriorityHigh
		need.CurrentLevel = 1.0 // Start healthy

	case types.NeedExploration:
		need.DecayRate = 0.1
		need.SatisfactionRate = 0.25
		need.Priority = PriorityLow

	default:
		need.DecayRate = 0.1
		need.SatisfactionRate = 0.3
		need.Priority = PriorityMedium
	}

	return need
}

// IsCritical returns true if the need is at a critical level
func (n *Need) IsCritical() bool {
	return n.CurrentLevel < n.CriticalThreshold
}

// IsWarning returns true if the need is at a warning level
func (n *Need) IsWarning() bool {
	return n.CurrentLevel < n.WarningThreshold
}

// IsSatisfied returns true if the need is well satisfied
func (n *Need) IsSatisfied() bool {
	return n.CurrentLevel > 0.7
}

// GetUrgency returns a normalized urgency value (0.0 = satisfied, 1.0 = critical)
func (n *Need) GetUrgency() float64 {
	return 1.0 - n.CurrentLevel
}

// Decay decreases the need level over time (increases the need)
func (n *Need) Decay(deltaTime float64) {
	n.CurrentLevel -= n.DecayRate * deltaTime
	n.Clamp()
}

// Satisfy increases the need level (decreases the need)
func (n *Need) Satisfy(amount float64, gameTime float64) {
	n.CurrentLevel += amount * n.SatisfactionRate
	n.LastSatisfied = gameTime
	n.Clamp()
}

// Clamp ensures the need level stays within valid range
func (n *Need) Clamp() {
	if n.CurrentLevel < 0.0 {
		n.CurrentLevel = 0.0
	}
	if n.CurrentLevel > 1.0 {
		n.CurrentLevel = 1.0
	}
}

// NeedsManager manages all needs for a digital pet
type NeedsManager struct {
	mu sync.RWMutex

	Needs             map[types.NeedType]*Need
	OverallWellbeing  float64
	CriticalNeedCount int
	WarningNeedCount  int
}

// NewNeedsManager creates a new needs manager with all 10 need types
func NewNeedsManager() *NeedsManager {
	nm := &NeedsManager{
		Needs: make(map[types.NeedType]*Need),
	}

	// Initialize all 10 need types
	needTypes := []types.NeedType{
		types.NeedHunger,
		types.NeedThirst,
		types.NeedSleep,
		types.NeedExercise,
		types.NeedSocial,
		types.NeedMentalStimulation,
		types.NeedAffection,
		types.NeedCleanliness,
		types.NeedMedicalCare,
		types.NeedExploration,
	}

	for _, needType := range needTypes {
		nm.Needs[needType] = NewNeed(needType)
	}

	nm.UpdateWellbeing()
	return nm
}

// Update processes need decay over time
func (nm *NeedsManager) Update(deltaTime float64) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for _, need := range nm.Needs {
		need.Decay(deltaTime)
	}

	nm.UpdateWellbeing()
}

// SatisfyNeed fulfills a specific need
func (nm *NeedsManager) SatisfyNeed(needType types.NeedType, amount float64, gameTime float64) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if need, exists := nm.Needs[needType]; exists {
		need.Satisfy(amount, gameTime)
		nm.UpdateWellbeing()
	}
}

// GetNeed returns a specific need (thread-safe copy)
func (nm *NeedsManager) GetNeed(needType types.NeedType) *Need {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if need, exists := nm.Needs[needType]; exists {
		// Return a copy to prevent external modification
		needCopy := *need
		return &needCopy
	}
	return nil
}

// GetNeedLevel returns the current level of a specific need
func (nm *NeedsManager) GetNeedLevel(needType types.NeedType) float64 {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if need, exists := nm.Needs[needType]; exists {
		return need.CurrentLevel
	}
	return 1.0 // Default to satisfied if need doesn't exist
}

// UpdateWellbeing recalculates overall wellbeing based on all needs
// Must be called with lock held
func (nm *NeedsManager) UpdateWellbeing() {
	totalWeight := 0.0
	weightedSum := 0.0
	criticalCount := 0
	warningCount := 0

	for _, need := range nm.Needs {
		// Weight by priority
		weight := float64(need.Priority + 1)
		totalWeight += weight
		weightedSum += need.CurrentLevel * weight

		if need.IsCritical() {
			criticalCount++
		} else if need.IsWarning() {
			warningCount++
		}
	}

	if totalWeight > 0 {
		nm.OverallWellbeing = weightedSum / totalWeight
	} else {
		nm.OverallWellbeing = 1.0
	}

	nm.CriticalNeedCount = criticalCount
	nm.WarningNeedCount = warningCount
}

// GetOverallWellbeing returns the overall wellbeing score
func (nm *NeedsManager) GetOverallWellbeing() float64 {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.OverallWellbeing
}

// GetCriticalNeeds returns a list of needs at critical levels
func (nm *NeedsManager) GetCriticalNeeds() []types.NeedType {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	var critical []types.NeedType
	for needType, need := range nm.Needs {
		if need.IsCritical() {
			critical = append(critical, needType)
		}
	}

	// Sort by urgency (most urgent first)
	sort.Slice(critical, func(i, j int) bool {
		return nm.Needs[critical[i]].GetUrgency() > nm.Needs[critical[j]].GetUrgency()
	})

	return critical
}

// GetWarningNeeds returns a list of needs at warning levels
func (nm *NeedsManager) GetWarningNeeds() []types.NeedType {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	var warning []types.NeedType
	for needType, need := range nm.Needs {
		if need.IsWarning() && !need.IsCritical() {
			warning = append(warning, needType)
		}
	}

	// Sort by urgency
	sort.Slice(warning, func(i, j int) bool {
		return nm.Needs[warning[i]].GetUrgency() > nm.Needs[warning[j]].GetUrgency()
	})

	return warning
}

// GetMostUrgentNeed returns the need with highest urgency
func (nm *NeedsManager) GetMostUrgentNeed() types.NeedType {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	var mostUrgent types.NeedType
	maxUrgency := 0.0

	for needType, need := range nm.Needs {
		urgency := need.GetUrgency()
		if urgency > maxUrgency {
			maxUrgency = urgency
			mostUrgent = needType
		}
	}

	return mostUrgent
}

// GetNeedsByPriority returns needs sorted by priority (highest first)
func (nm *NeedsManager) GetNeedsByPriority() []types.NeedType {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	needList := make([]types.NeedType, 0, len(nm.Needs))
	for needType := range nm.Needs {
		needList = append(needList, needType)
	}

	sort.Slice(needList, func(i, j int) bool {
		needI := nm.Needs[needList[i]]
		needJ := nm.Needs[needList[j]]

		// Sort by priority first
		if needI.Priority != needJ.Priority {
			return needI.Priority > needJ.Priority
		}

		// Then by urgency
		return needI.GetUrgency() > needJ.GetUrgency()
	})

	return needList
}

// GetAllNeedsStatus returns a snapshot of all needs
func (nm *NeedsManager) GetAllNeedsStatus() NeedsStatus {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	status := NeedsStatus{
		OverallWellbeing:  nm.OverallWellbeing,
		CriticalNeedCount: nm.CriticalNeedCount,
		WarningNeedCount:  nm.WarningNeedCount,
		NeedLevels:        make(map[types.NeedType]float64),
		CriticalNeeds:     []types.NeedType{},
		WarningNeeds:      []types.NeedType{},
	}

	for needType, need := range nm.Needs {
		status.NeedLevels[needType] = need.CurrentLevel

		if need.IsCritical() {
			status.CriticalNeeds = append(status.CriticalNeeds, needType)
		} else if need.IsWarning() {
			status.WarningNeeds = append(status.WarningNeeds, needType)
		}
	}

	return status
}

// NeedsStatus represents a snapshot of the current needs state
type NeedsStatus struct {
	OverallWellbeing  float64
	CriticalNeedCount int
	WarningNeedCount  int
	NeedLevels        map[types.NeedType]float64
	CriticalNeeds     []types.NeedType
	WarningNeeds      []types.NeedType
}

// HasCriticalNeeds returns true if any needs are critical
func (nm *NeedsManager) HasCriticalNeeds() bool {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.CriticalNeedCount > 0
}

// HasWarningNeeds returns true if any needs are at warning level
func (nm *NeedsManager) HasWarningNeeds() bool {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.WarningNeedCount > 0
}

// SetNeedLevel manually sets a need level (for testing/debugging)
func (nm *NeedsManager) SetNeedLevel(needType types.NeedType, level float64) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if need, exists := nm.Needs[needType]; exists {
		need.CurrentLevel = level
		need.Clamp()
		nm.UpdateWellbeing()
	}
}

// Reset resets all needs to satisfied state
func (nm *NeedsManager) Reset() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for _, need := range nm.Needs {
		need.CurrentLevel = 1.0
		need.LastSatisfied = 0.0
	}

	nm.UpdateWellbeing()
}
