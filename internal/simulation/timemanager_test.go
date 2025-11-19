package simulation

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewTimeManager(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	if tm == nil {
		t.Fatal("TimeManager should not be nil")
	}

	if tm.CurrentTimeScale != types.TimeScaleRealTime {
		t.Errorf("Expected TimeScaleRealTime, got %v", tm.CurrentTimeScale)
	}

	if tm.TimeScaleMultiplier != 1.0 {
		t.Errorf("Expected multiplier 1.0, got %.2f", tm.TimeScaleMultiplier)
	}

	if tm.IsPaused {
		t.Error("New TimeManager should not be paused")
	}
}

func TestTimeManagerUpdate(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	// Wait a bit and update
	time.Sleep(50 * time.Millisecond)
	delta := tm.Update()

	if delta <= 0 {
		t.Errorf("Delta time should be positive, got %.6f", delta)
	}

	if delta < 0.04 || delta > 0.06 {
		t.Errorf("Delta time should be around 0.05s, got %.6f", delta)
	}

	if tm.TickCount != 1 {
		t.Errorf("Tick count should be 1, got %d", tm.TickCount)
	}
}

func TestTimeScaleMultipliers(t *testing.T) {
	tests := []struct {
		scale      types.TimeScale
		multiplier float64
	}{
		{types.TimeScaleRealTime, 1.0},
		{types.TimeScaleAccelerated4X, 4.0},
		{types.TimeScaleAccelerated24X, 24.0},
		{types.TimeScalePaused, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.scale.String(), func(t *testing.T) {
			tm := NewTimeManager(tt.scale)
			if tm.TimeScaleMultiplier != tt.multiplier {
				t.Errorf("Expected multiplier %.1f, got %.1f", tt.multiplier, tm.TimeScaleMultiplier)
			}
		})
	}
}

func TestSetTimeScale(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	tm.SetTimeScale(types.TimeScaleAccelerated4X)

	if tm.GetTimeScale() != types.TimeScaleAccelerated4X {
		t.Errorf("Expected TimeScaleAccelerated4X, got %v", tm.GetTimeScale())
	}

	if tm.TimeScaleMultiplier != 4.0 {
		t.Errorf("Expected multiplier 4.0, got %.2f", tm.TimeScaleMultiplier)
	}
}

func TestPauseAndResume(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	// Pause
	tm.Pause()
	if !tm.IsPausedState() {
		t.Error("TimeManager should be paused")
	}

	// Update while paused should return 0
	delta := tm.Update()
	if delta != 0.0 {
		t.Errorf("Delta should be 0 when paused, got %.6f", delta)
	}

	// Resume
	tm.Resume()
	if tm.IsPausedState() {
		t.Error("TimeManager should not be paused after resume")
	}

	// Update after resume should work
	time.Sleep(10 * time.Millisecond)
	delta = tm.Update()
	if delta <= 0 {
		t.Error("Delta should be positive after resume")
	}
}

func TestTogglePause(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	initialState := tm.IsPausedState()
	paused := tm.TogglePause()

	if paused == initialState {
		t.Error("Toggle should change pause state")
	}

	if tm.IsPausedState() != paused {
		t.Error("Pause state mismatch")
	}
}

func TestAcceleratedTime(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleAccelerated4X)

	time.Sleep(50 * time.Millisecond)
	delta := tm.Update()

	// With 4x acceleration, 50ms real time should be ~200ms sim time
	expectedDelta := 0.05 * 4.0 // seconds
	if delta < expectedDelta*0.8 || delta > expectedDelta*1.2 {
		t.Errorf("Expected delta around %.3f, got %.3f", expectedDelta, delta)
	}
}

func TestGetAgeInDays(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	// Advance time by 24 hours
	tm.AdvanceTime(24 * time.Hour)

	age := tm.GetAgeInDays()
	if age < 0.99 || age > 1.01 {
		t.Errorf("Expected age around 1.0 days, got %.6f", age)
	}
}

func TestGetAgeInHours(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	// Advance time by 12 hours
	tm.AdvanceTime(12 * time.Hour)

	hours := tm.GetAgeInHours()
	if hours < 11.99 || hours > 12.01 {
		t.Errorf("Expected age around 12.0 hours, got %.6f", hours)
	}
}

func TestGetTimeOfDay(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	// The time will be based on actual system time, so just check it's valid
	tod := tm.GetTimeOfDay()

	if tod < 0 || tod >= 24 {
		t.Errorf("Time of day should be between 0 and 24, got %.2f", tod)
	}
}

func TestIsDaytimeNighttime(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	// Set to noon
	now := time.Now()
	noon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	tm.CurrentSimTime = noon

	if !tm.IsDaytime() {
		t.Error("Noon should be daytime")
	}

	if tm.IsNighttime() {
		t.Error("Noon should not be nighttime")
	}

	// Set to midnight
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tm.CurrentSimTime = midnight

	if tm.IsDaytime() {
		t.Error("Midnight should not be daytime")
	}

	if !tm.IsNighttime() {
		t.Error("Midnight should be nighttime")
	}
}

func TestGetDayPhase(t *testing.T) {
	tests := []struct {
		hour  int
		phase string
	}{
		{6, "dawn"},
		{10, "morning"},
		{14, "afternoon"},
		{18, "evening"},
		{21, "dusk"},
		{23, "night"},
	}

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			tm := NewTimeManager(types.TimeScaleRealTime)
			now := time.Now()
			testTime := time.Date(now.Year(), now.Month(), now.Day(), tt.hour, 0, 0, 0, now.Location())
			tm.CurrentSimTime = testTime

			phase := tm.GetDayPhase()
			if phase != tt.phase {
				t.Errorf("Expected phase '%s' at hour %d, got '%s'", tt.phase, tt.hour, phase)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleAccelerated4X)

	time.Sleep(20 * time.Millisecond)
	tm.Update()
	tm.Update()

	stats := tm.GetStats()

	if stats.CurrentTimeScale != types.TimeScaleAccelerated4X {
		t.Error("Stats should show correct time scale")
	}

	if stats.TimeScaleMultiplier != 4.0 {
		t.Error("Stats should show correct multiplier")
	}

	if stats.TickCount != 2 {
		t.Errorf("Expected 2 ticks, got %d", stats.TickCount)
	}

	if stats.IsPaused {
		t.Error("Stats should show not paused")
	}
}

func TestReset(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	// Advance time and ticks
	time.Sleep(50 * time.Millisecond)
	tm.Update()
	tm.Update()
	tm.Update()

	if tm.TickCount == 0 {
		t.Error("Should have ticks before reset")
	}

	// Reset
	tm.Reset()

	if tm.TickCount != 0 {
		t.Error("Tick count should be 0 after reset")
	}

	if tm.TotalSimulatedDays != 0 {
		t.Error("Simulated days should be 0 after reset")
	}

	if tm.IsPaused {
		t.Error("Should not be paused after reset")
	}
}

func TestConvertRealToSimTime(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleAccelerated4X)

	realDuration := 1 * time.Hour
	simDuration := tm.ConvertRealToSimTime(realDuration)

	expected := 4 * time.Hour
	if simDuration != expected {
		t.Errorf("Expected %v sim time, got %v", expected, simDuration)
	}
}

func TestConvertSimToRealTime(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleAccelerated4X)

	simDuration := 4 * time.Hour
	realDuration := tm.ConvertSimToRealTime(simDuration)

	expected := 1 * time.Hour
	if realDuration != expected {
		t.Errorf("Expected %v real time, got %v", expected, realDuration)
	}
}

func TestGetElapsedSince(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	startTime := tm.GetSimulationTime()
	tm.AdvanceTime(5 * time.Hour)

	elapsed := tm.GetElapsedSince(startTime)

	if elapsed < 4*time.Hour || elapsed > 6*time.Hour {
		t.Errorf("Expected around 5 hours elapsed, got %v", elapsed)
	}
}

func TestConcurrentAccess(t *testing.T) {
	tm := NewTimeManager(types.TimeScaleRealTime)

	// Test concurrent reads and writes
	done := make(chan bool, 3)

	// Goroutine 1: Update loop
	go func() {
		for i := 0; i < 10; i++ {
			tm.Update()
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Read stats
	go func() {
		for i := 0; i < 10; i++ {
			_ = tm.GetStats()
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 3: Change time scale
	go func() {
		for i := 0; i < 5; i++ {
			tm.SetTimeScale(types.TimeScaleAccelerated4X)
			time.Sleep(10 * time.Millisecond)
			tm.SetTimeScale(types.TimeScaleRealTime)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should complete without panicking
}
