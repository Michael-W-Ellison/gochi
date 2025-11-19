package biology

import (
	"math"
	"sync"
)

// CircadianPhase represents the current phase of the circadian cycle
type CircadianPhase int

const (
	PhaseAwake CircadianPhase = iota
	PhaseDrowsy
	PhaseAsleep
	PhaseDeepSleep
	PhaseREMSleep
	PhaseWaking
)

// String returns the string representation of CircadianPhase
func (cp CircadianPhase) String() string {
	return [...]string{
		"Awake", "Drowsy", "Asleep", "Deep Sleep", "REM Sleep", "Waking",
	}[cp]
}

// CircadianRhythm manages sleep/wake cycles and related biological processes
type CircadianRhythm struct {
	mu sync.RWMutex

	// Current state
	CurrentPhase      CircadianPhase
	TimeInPhase       float64 // Game hours in current phase
	TotalSleepTime    float64 // Total sleep accumulated today (hours)
	LastSleepTime     float64 // Game time when last went to sleep
	LastWakeTime      float64 // Game time when last woke up

	// Sleep cycle tracking
	SleepCycleCount   int     // Number of complete sleep cycles
	CurrentCycleTime  float64 // Time in current sleep cycle (minutes)
	SleepQuality      float64 // Quality of current/last sleep (0.0-1.0)

	// Sleep debt and pressure
	SleepDebt         float64 // Accumulated sleep debt (hours)
	SleepPressure     float64 // Drive to sleep (0.0-1.0)
	HomeostaticDrive  float64 // Biological sleep drive (0.0-1.0)

	// Circadian settings
	OptimalSleepStart float64 // Optimal time to sleep (hour, 0-24)
	OptimalSleepEnd   float64 // Optimal time to wake (hour, 0-24)
	SleepDuration     float64 // Target sleep duration (hours)
	SleepCycleLength  float64 // Length of one sleep cycle (minutes)

	// Statistics
	TotalLifetimeSleep float64 // Total hours slept over lifetime
	SleepCyclesCompleted int   // Total sleep cycles completed
}

// NewCircadianRhythm creates a new circadian rhythm system
func NewCircadianRhythm() *CircadianRhythm {
	return &CircadianRhythm{
		CurrentPhase:       PhaseAwake,
		TimeInPhase:        0,
		TotalSleepTime:     0,
		LastSleepTime:      0,
		LastWakeTime:       0,
		SleepCycleCount:    0,
		CurrentCycleTime:   0,
		SleepQuality:       1.0,
		SleepDebt:          0,
		SleepPressure:      0,
		HomeostaticDrive:   0,
		OptimalSleepStart:  22.0, // 10 PM
		OptimalSleepEnd:    6.0,  // 6 AM
		SleepDuration:      8.0,  // 8 hours
		SleepCycleLength:   90.0, // 90 minutes per cycle
		TotalLifetimeSleep: 0,
		SleepCyclesCompleted: 0,
	}
}

// Update processes circadian rhythm changes over time
func (cr *CircadianRhythm) Update(deltaTime float64, currentTimeOfDay float64, isNighttime bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.TimeInPhase += deltaTime

	// Update homeostatic drive (increases with time awake, decreases with sleep)
	// Check phase directly since we already hold the lock
	if cr.CurrentPhase == PhaseAwake || cr.CurrentPhase == PhaseWaking || cr.CurrentPhase == PhaseDrowsy {
		cr.HomeostaticDrive += deltaTime * 0.05
		if cr.HomeostaticDrive > 1.0 {
			cr.HomeostaticDrive = 1.0
		}
	} else {
		cr.HomeostaticDrive -= deltaTime * 0.1
		if cr.HomeostaticDrive < 0 {
			cr.HomeostaticDrive = 0
		}
	}

	// Calculate circadian component (time of day influence)
	circadianComponent := cr.calculateCircadianComponent(currentTimeOfDay)

	// Sleep pressure is combination of homeostatic and circadian drives
	cr.SleepPressure = (cr.HomeostaticDrive * 0.7) + (circadianComponent * 0.3)
	if cr.SleepPressure > 1.0 {
		cr.SleepPressure = 1.0
	}

	// Process current phase
	switch cr.CurrentPhase {
	case PhaseAwake:
		cr.updateAwakePhase(deltaTime, isNighttime)

	case PhaseDrowsy:
		cr.updateDrowsyPhase(deltaTime)

	case PhaseAsleep:
		cr.updateAsleepPhase(deltaTime)

	case PhaseDeepSleep:
		cr.updateDeepSleepPhase(deltaTime)

	case PhaseREMSleep:
		cr.updateREMSleepPhase(deltaTime)

	case PhaseWaking:
		cr.updateWakingPhase(deltaTime)
	}

	// Update sleep debt
	// Check phase directly since we already hold the lock
	if cr.CurrentPhase == PhaseAwake || cr.CurrentPhase == PhaseWaking || cr.CurrentPhase == PhaseDrowsy {
		// Accumulate sleep debt while awake
		targetSleep := cr.SleepDuration / 24.0 // Per hour
		cr.SleepDebt += (targetSleep - (cr.TotalSleepTime / 24.0)) * deltaTime
		if cr.SleepDebt < 0 {
			cr.SleepDebt = 0
		}
	}
}

// updateAwakePhase handles the awake phase
func (cr *CircadianRhythm) updateAwakePhase(deltaTime float64, isNighttime bool) {
	// Transition to drowsy if sleep pressure is high
	if cr.SleepPressure > 0.7 || (isNighttime && cr.HomeostaticDrive > 0.5) {
		cr.transitionToPhase(PhaseDrowsy)
	}
}

// updateDrowsyPhase handles the drowsy phase
func (cr *CircadianRhythm) updateDrowsyPhase(deltaTime float64) {
	// Automatically fall asleep after being drowsy for a while
	if cr.TimeInPhase > 0.5 { // 30 minutes
		cr.transitionToPhase(PhaseAsleep)
		cr.LastSleepTime = 0 // Would be set by external system
	}
}

// updateAsleepPhase handles light sleep phase
func (cr *CircadianRhythm) updateAsleepPhase(deltaTime float64) {
	cr.TotalSleepTime += deltaTime
	cr.TotalLifetimeSleep += deltaTime
	cr.CurrentCycleTime += deltaTime * 60 // Convert to minutes

	// Transition to deep sleep after ~20 minutes
	if cr.CurrentCycleTime > 20 {
		cr.transitionToPhase(PhaseDeepSleep)
	}
}

// updateDeepSleepPhase handles deep sleep phase
func (cr *CircadianRhythm) updateDeepSleepPhase(deltaTime float64) {
	cr.TotalSleepTime += deltaTime
	cr.TotalLifetimeSleep += deltaTime
	cr.CurrentCycleTime += deltaTime * 60

	// Deep sleep lasts ~40 minutes, then move to REM
	if cr.CurrentCycleTime > 60 {
		cr.transitionToPhase(PhaseREMSleep)
	}

	// Reduce sleep debt during deep sleep
	cr.SleepDebt -= deltaTime * 0.2
	if cr.SleepDebt < 0 {
		cr.SleepDebt = 0
	}
}

// updateREMSleepPhase handles REM sleep phase
func (cr *CircadianRhythm) updateREMSleepPhase(deltaTime float64) {
	cr.TotalSleepTime += deltaTime
	cr.TotalLifetimeSleep += deltaTime
	cr.CurrentCycleTime += deltaTime * 60

	// REM sleep completes the ~90 minute cycle
	if cr.CurrentCycleTime >= cr.SleepCycleLength {
		cr.SleepCycleCount++
		cr.SleepCyclesCompleted++
		cr.CurrentCycleTime = 0

		// Check if should wake up or start new cycle
		if cr.TotalSleepTime >= cr.SleepDuration || cr.SleepDebt <= 0 {
			cr.transitionToPhase(PhaseWaking)
		} else {
			// Start new sleep cycle
			cr.transitionToPhase(PhaseAsleep)
		}
	}

	// Reduce sleep debt during REM
	cr.SleepDebt -= deltaTime * 0.15
	if cr.SleepDebt < 0 {
		cr.SleepDebt = 0
	}
}

// updateWakingPhase handles the waking phase
func (cr *CircadianRhythm) updateWakingPhase(deltaTime float64) {
	// Brief transition phase, then fully awake
	if cr.TimeInPhase > 0.1 { // ~6 minutes
		cr.calculateSleepQuality()
		cr.transitionToPhase(PhaseAwake)
		cr.LastWakeTime = 0 // Would be set by external system
		cr.TotalSleepTime = 0 // Reset daily sleep counter
		cr.SleepCycleCount = 0
	}
}

// transitionToPhase changes the current phase
func (cr *CircadianRhythm) transitionToPhase(newPhase CircadianPhase) {
	cr.CurrentPhase = newPhase
	cr.TimeInPhase = 0
}

// calculateCircadianComponent returns circadian influence on sleep drive (0.0-1.0)
func (cr *CircadianRhythm) calculateCircadianComponent(timeOfDay float64) float64 {
	// Peak sleep drive in the middle of optimal sleep period
	optimalMidpoint := (cr.OptimalSleepStart + cr.OptimalSleepEnd) / 2.0
	if cr.OptimalSleepEnd < cr.OptimalSleepStart {
		optimalMidpoint += 12.0 // Handle overnight period
		if optimalMidpoint >= 24 {
			optimalMidpoint -= 24
		}
	}

	// Use cosine wave for smooth circadian rhythm
	// Peak at optimal sleep time, trough at optimal wake time
	hoursDiff := timeOfDay - optimalMidpoint
	if hoursDiff > 12 {
		hoursDiff -= 24
	} else if hoursDiff < -12 {
		hoursDiff += 24
	}

	// Convert to 0-1 range with peak at optimal sleep time
	component := (math.Cos((hoursDiff/12.0)*math.Pi) + 1) / 2.0
	return component
}

// calculateSleepQuality determines quality of the completed sleep session
func (cr *CircadianRhythm) calculateSleepQuality() {
	quality := 1.0

	// Penalize if sleep duration too short or too long
	if cr.TotalSleepTime < cr.SleepDuration*0.7 {
		quality -= 0.3
	} else if cr.TotalSleepTime > cr.SleepDuration*1.3 {
		quality -= 0.2
	}

	// Reward complete sleep cycles
	expectedCycles := cr.SleepDuration / (cr.SleepCycleLength / 60.0)
	cycleRatio := float64(cr.SleepCycleCount) / expectedCycles
	if cycleRatio < 0.8 {
		quality -= 0.2
	}

	// Bonus for sleeping at optimal time
	// This would need time of day info, simplified for now

	if quality < 0.1 {
		quality = 0.1
	}
	if quality > 1.0 {
		quality = 1.0
	}

	cr.SleepQuality = quality
}

// InitiateSleep forces the pet to start sleeping
func (cr *CircadianRhythm) InitiateSleep() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Check phase directly since we already hold the lock
	if cr.CurrentPhase == PhaseAwake || cr.CurrentPhase == PhaseWaking || cr.CurrentPhase == PhaseDrowsy {
		cr.transitionToPhase(PhaseAsleep)
		cr.CurrentCycleTime = 0
	}
}

// ForceWake forces the pet to wake up
func (cr *CircadianRhythm) ForceWake() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Check phase directly since we already hold the lock
	if cr.CurrentPhase == PhaseAsleep || cr.CurrentPhase == PhaseDeepSleep || cr.CurrentPhase == PhaseREMSleep {
		cr.calculateSleepQuality()
		cr.transitionToPhase(PhaseAwake)
		cr.TotalSleepTime = 0
		cr.SleepCycleCount = 0
		cr.CurrentCycleTime = 0
	}
}

// IsAwakePhase returns true if in an awake phase
func (cr *CircadianRhythm) IsAwakePhase() bool {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.CurrentPhase == PhaseAwake || cr.CurrentPhase == PhaseWaking || cr.CurrentPhase == PhaseDrowsy
}

// IsSleepingPhase returns true if in a sleeping phase
func (cr *CircadianRhythm) IsSleepingPhase() bool {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.CurrentPhase == PhaseAsleep || cr.CurrentPhase == PhaseDeepSleep || cr.CurrentPhase == PhaseREMSleep
}

// GetCurrentPhase returns the current circadian phase
func (cr *CircadianRhythm) GetCurrentPhase() CircadianPhase {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.CurrentPhase
}

// GetSleepPressure returns the current sleep pressure (0.0-1.0)
func (cr *CircadianRhythm) GetSleepPressure() float64 {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.SleepPressure
}

// GetSleepDebt returns accumulated sleep debt in hours
func (cr *CircadianRhythm) GetSleepDebt() float64 {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.SleepDebt
}

// GetSleepQuality returns the quality of the last sleep (0.0-1.0)
func (cr *CircadianRhythm) GetSleepQuality() float64 {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.SleepQuality
}

// GetStats returns circadian rhythm statistics
func (cr *CircadianRhythm) GetStats() CircadianStats {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	// Check phase directly since we already hold the lock
	isAsleep := cr.CurrentPhase == PhaseAsleep || cr.CurrentPhase == PhaseDeepSleep || cr.CurrentPhase == PhaseREMSleep

	return CircadianStats{
		CurrentPhase:        cr.CurrentPhase,
		TimeInPhase:         cr.TimeInPhase,
		SleepPressure:       cr.SleepPressure,
		SleepDebt:           cr.SleepDebt,
		SleepQuality:        cr.SleepQuality,
		TotalSleepToday:     cr.TotalSleepTime,
		SleepCyclesCompleted: cr.SleepCyclesCompleted,
		IsAsleep:            isAsleep,
	}
}

// CircadianStats contains statistics about circadian rhythm
type CircadianStats struct {
	CurrentPhase        CircadianPhase
	TimeInPhase         float64
	SleepPressure       float64
	SleepDebt           float64
	SleepQuality        float64
	TotalSleepToday     float64
	SleepCyclesCompleted int
	IsAsleep            bool
}

// SetOptimalSleepWindow sets the preferred sleep time
func (cr *CircadianRhythm) SetOptimalSleepWindow(startHour, endHour float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.OptimalSleepStart = startHour
	cr.OptimalSleepEnd = endHour
}

// Reset resets the circadian rhythm to initial state
func (cr *CircadianRhythm) Reset() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.CurrentPhase = PhaseAwake
	cr.TimeInPhase = 0
	cr.TotalSleepTime = 0
	cr.SleepCycleCount = 0
	cr.CurrentCycleTime = 0
	cr.SleepQuality = 1.0
	cr.SleepDebt = 0
	cr.SleepPressure = 0
	cr.HomeostaticDrive = 0
}
