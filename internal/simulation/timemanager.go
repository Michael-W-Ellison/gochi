package simulation

import (
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// TimeManager manages simulation time with configurable time scales
type TimeManager struct {
	mu sync.RWMutex

	// Time tracking
	RealStartTime       time.Time     // When the simulation started in real time
	SimulationStartTime time.Time     // Simulation epoch (game time zero)
	CurrentSimTime      time.Time     // Current simulation time
	LastUpdateRealTime  time.Time     // Last real-world update time
	TotalSimulatedDays  float64       // Total game days elapsed

	// Time scale settings
	CurrentTimeScale    types.TimeScale // Current time scaling mode
	TimeScaleMultiplier float64         // Current multiplier (derived from TimeScale)
	IsPaused            bool            // Whether time is paused

	// Delta time tracking
	LastDeltaTime       float64 // Last frame's delta time in game seconds
	AccumulatedTime     float64 // Accumulated time for fixed timesteps

	// Statistics
	TotalRealTimeElapsed time.Duration // Total real time since start
	TickCount            uint64        // Number of update ticks
}

// NewTimeManager creates a new time manager with specified time scale
func NewTimeManager(timeScale types.TimeScale) *TimeManager {
	now := time.Now()

	tm := &TimeManager{
		RealStartTime:       now,
		SimulationStartTime: now,
		CurrentSimTime:      now,
		LastUpdateRealTime:  now,
		TotalSimulatedDays:  0.0,
		CurrentTimeScale:    timeScale,
		IsPaused:            false,
		LastDeltaTime:       0.0,
		AccumulatedTime:     0.0,
		TotalRealTimeElapsed: 0,
		TickCount:           0,
	}

	tm.TimeScaleMultiplier = tm.getMultiplierForScale(timeScale)
	return tm
}

// Update processes time advancement and returns delta time in game seconds
func (tm *TimeManager) Update() float64 {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.IsPaused {
		tm.LastDeltaTime = 0.0
		return 0.0
	}

	// Calculate real time elapsed since last update
	now := time.Now()
	realDelta := now.Sub(tm.LastUpdateRealTime).Seconds()
	tm.LastUpdateRealTime = now

	// Calculate total real time elapsed
	tm.TotalRealTimeElapsed = now.Sub(tm.RealStartTime)

	// Calculate simulation delta time (scaled by time scale)
	simDelta := realDelta * tm.TimeScaleMultiplier
	tm.LastDeltaTime = simDelta

	// Update simulation time
	tm.CurrentSimTime = tm.CurrentSimTime.Add(time.Duration(simDelta * float64(time.Second)))

	// Track total simulated days
	tm.TotalSimulatedDays = tm.CurrentSimTime.Sub(tm.SimulationStartTime).Hours() / 24.0

	// Increment tick counter
	tm.TickCount++

	return simDelta
}

// GetDeltaTime returns the last calculated delta time in game seconds
func (tm *TimeManager) GetDeltaTime() float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.LastDeltaTime
}

// GetSimulationTime returns the current simulation time
func (tm *TimeManager) GetSimulationTime() time.Time {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.CurrentSimTime
}

// GetAgeInDays returns the age in game days since simulation start
func (tm *TimeManager) GetAgeInDays() float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.TotalSimulatedDays
}

// GetAgeInHours returns the age in game hours since simulation start
func (tm *TimeManager) GetAgeInHours() float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.TotalSimulatedDays * 24.0
}

// SetTimeScale changes the current time scale
func (tm *TimeManager) SetTimeScale(scale types.TimeScale) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.CurrentTimeScale = scale
	tm.TimeScaleMultiplier = tm.getMultiplierForScale(scale)
}

// GetTimeScale returns the current time scale
func (tm *TimeManager) GetTimeScale() types.TimeScale {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.CurrentTimeScale
}

// Pause pauses the simulation time
func (tm *TimeManager) Pause() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.IsPaused = true
}

// Resume resumes the simulation time
func (tm *TimeManager) Resume() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.IsPaused {
		tm.IsPaused = false
		// Reset last update time to prevent large delta jump
		tm.LastUpdateRealTime = time.Now()
	}
}

// TogglePause toggles pause state
func (tm *TimeManager) TogglePause() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.IsPaused = !tm.IsPaused

	if !tm.IsPaused {
		// Reset last update time to prevent large delta jump
		tm.LastUpdateRealTime = time.Now()
	}

	return tm.IsPaused
}

// IsPausedState returns whether time is currently paused
func (tm *TimeManager) IsPausedState() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.IsPaused
}

// GetTimeScaleMultiplier returns the current time scale multiplier
func (tm *TimeManager) GetTimeScaleMultiplier() float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.TimeScaleMultiplier
}

// GetTimeOfDay returns the current time of day in hours (0-24)
func (tm *TimeManager) GetTimeOfDay() float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	hour := float64(tm.CurrentSimTime.Hour())
	minute := float64(tm.CurrentSimTime.Minute()) / 60.0
	second := float64(tm.CurrentSimTime.Second()) / 3600.0

	return hour + minute + second
}

// IsDaytime returns true if current time is during day (6am - 8pm)
func (tm *TimeManager) IsDaytime() bool {
	hour := tm.GetTimeOfDay()
	return hour >= 6.0 && hour < 20.0
}

// IsNighttime returns true if current time is during night (8pm - 6am)
func (tm *TimeManager) IsNighttime() bool {
	return !tm.IsDaytime()
}

// GetDayPhase returns a description of the current day phase
func (tm *TimeManager) GetDayPhase() string {
	hour := tm.GetTimeOfDay()

	switch {
	case hour >= 5.0 && hour < 8.0:
		return "dawn"
	case hour >= 8.0 && hour < 12.0:
		return "morning"
	case hour >= 12.0 && hour < 17.0:
		return "afternoon"
	case hour >= 17.0 && hour < 20.0:
		return "evening"
	case hour >= 20.0 && hour < 22.0:
		return "dusk"
	default:
		return "night"
	}
}

// GetStats returns time manager statistics
func (tm *TimeManager) GetStats() TimeStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return TimeStats{
		RealTimeElapsed:     tm.TotalRealTimeElapsed,
		SimulatedDays:       tm.TotalSimulatedDays,
		SimulatedHours:      tm.TotalSimulatedDays * 24.0,
		CurrentTimeScale:    tm.CurrentTimeScale,
		TimeScaleMultiplier: tm.TimeScaleMultiplier,
		IsPaused:            tm.IsPaused,
		CurrentDayPhase:     tm.GetDayPhase(),
		TickCount:           tm.TickCount,
		AverageTickRate:     float64(tm.TickCount) / tm.TotalRealTimeElapsed.Seconds(),
	}
}

// TimeStats represents statistics about time management
type TimeStats struct {
	RealTimeElapsed     time.Duration
	SimulatedDays       float64
	SimulatedHours      float64
	CurrentTimeScale    types.TimeScale
	TimeScaleMultiplier float64
	IsPaused            bool
	CurrentDayPhase     string
	TickCount           uint64
	AverageTickRate     float64 // Ticks per second
}

// Reset resets the time manager to initial state
func (tm *TimeManager) Reset() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	tm.RealStartTime = now
	tm.SimulationStartTime = now
	tm.CurrentSimTime = now
	tm.LastUpdateRealTime = now
	tm.TotalSimulatedDays = 0.0
	tm.LastDeltaTime = 0.0
	tm.AccumulatedTime = 0.0
	tm.TotalRealTimeElapsed = 0
	tm.TickCount = 0
	tm.IsPaused = false
}

// AdvanceTime manually advances simulation time by a specified duration (for testing/debugging)
func (tm *TimeManager) AdvanceTime(duration time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.CurrentSimTime = tm.CurrentSimTime.Add(duration)
	tm.TotalSimulatedDays = tm.CurrentSimTime.Sub(tm.SimulationStartTime).Hours() / 24.0
}

// getMultiplierForScale returns the time scale multiplier for a given time scale
func (tm *TimeManager) getMultiplierForScale(scale types.TimeScale) float64 {
	switch scale {
	case types.TimeScaleRealTime:
		return 1.0
	case types.TimeScaleAccelerated4X:
		return 4.0
	case types.TimeScaleAccelerated24X:
		return 24.0
	case types.TimeScalePaused:
		return 0.0
	default:
		return 1.0
	}
}

// ConvertRealToSimTime converts a real-world duration to simulation duration
func (tm *TimeManager) ConvertRealToSimTime(realDuration time.Duration) time.Duration {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	simSeconds := realDuration.Seconds() * tm.TimeScaleMultiplier
	return time.Duration(simSeconds * float64(time.Second))
}

// ConvertSimToRealTime converts a simulation duration to real-world duration
func (tm *TimeManager) ConvertSimToRealTime(simDuration time.Duration) time.Duration {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.TimeScaleMultiplier == 0 {
		return time.Duration(0)
	}

	realSeconds := simDuration.Seconds() / tm.TimeScaleMultiplier
	return time.Duration(realSeconds * float64(time.Second))
}

// GetElapsedSince returns the simulation time elapsed since a given time
func (tm *TimeManager) GetElapsedSince(since time.Time) time.Duration {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.CurrentSimTime.Sub(since)
}
