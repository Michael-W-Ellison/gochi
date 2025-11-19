package biology

import (
	"math"
	"testing"
)

func TestNewCircadianRhythm(t *testing.T) {
	cr := NewCircadianRhythm()

	if cr == nil {
		t.Fatal("CircadianRhythm should not be nil")
	}

	if cr.CurrentPhase != PhaseAwake {
		t.Errorf("Expected initial phase Awake, got %v", cr.CurrentPhase)
	}

	if cr.SleepDebt != 0 {
		t.Errorf("Expected 0 sleep debt initially, got %.2f", cr.SleepDebt)
	}

	if cr.SleepDuration != 8.0 {
		t.Errorf("Expected 8 hours sleep duration, got %.2f", cr.SleepDuration)
	}
}

func TestCircadianPhaseString(t *testing.T) {
	phases := []CircadianPhase{
		PhaseAwake, PhaseDrowsy, PhaseAsleep,
		PhaseDeepSleep, PhaseREMSleep, PhaseWaking,
	}
	expected := []string{
		"Awake", "Drowsy", "Asleep",
		"Deep Sleep", "REM Sleep", "Waking",
	}

	for i, phase := range phases {
		str := phase.String()
		if str != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], str)
		}
	}
}

func TestInitiateSleep(t *testing.T) {
	cr := NewCircadianRhythm()

	cr.InitiateSleep()

	if cr.GetCurrentPhase() != PhaseAsleep {
		t.Errorf("Expected Asleep phase, got %v", cr.GetCurrentPhase())
	}

	if !cr.IsSleepingPhase() {
		t.Error("Should be in sleeping phase")
	}
}

func TestForceWake(t *testing.T) {
	cr := NewCircadianRhythm()

	cr.InitiateSleep()
	cr.ForceWake()

	if cr.GetCurrentPhase() != PhaseAwake {
		t.Errorf("Expected Awake phase, got %v", cr.GetCurrentPhase())
	}

	if cr.IsSleepingPhase() {
		t.Error("Should not be in sleeping phase")
	}
}

func TestIsAwakePhase(t *testing.T) {
	cr := NewCircadianRhythm()

	if !cr.IsAwakePhase() {
		t.Error("Should be in awake phase initially")
	}

	cr.InitiateSleep()

	if cr.IsAwakePhase() {
		t.Error("Should not be in awake phase after initiating sleep")
	}
}

func TestIsSleepingPhase(t *testing.T) {
	cr := NewCircadianRhythm()

	if cr.IsSleepingPhase() {
		t.Error("Should not be in sleeping phase initially")
	}

	cr.InitiateSleep()

	if !cr.IsSleepingPhase() {
		t.Error("Should be in sleeping phase after initiating sleep")
	}
}

func TestHomeostaticDriveIncreases(t *testing.T) {
	cr := NewCircadianRhythm()

	initialDrive := cr.HomeostaticDrive

	// Simulate being awake for several hours
	for i := 0; i < 10; i++ {
		cr.Update(1.0, 12.0, false) // 1 hour updates, noon, daytime
	}

	if cr.HomeostaticDrive <= initialDrive {
		t.Error("Homeostatic drive should increase while awake")
	}
}

func TestSleepPressureCalculation(t *testing.T) {
	cr := NewCircadianRhythm()

	// Stay awake for many hours
	for i := 0; i < 16; i++ {
		cr.Update(1.0, float64(6+i), i >= 14) // Start at 6 AM
	}

	pressure := cr.GetSleepPressure()

	if pressure < 0.5 {
		t.Errorf("Sleep pressure should be high after 16 hours awake, got %.2f", pressure)
	}
}

func TestSleepCycleProgression(t *testing.T) {
	cr := NewCircadianRhythm()

	cr.InitiateSleep()

	// Update for ~20 minutes (should transition to deep sleep)
	cr.Update(0.35, 22.0, true) // ~21 minutes

	if cr.GetCurrentPhase() != PhaseDeepSleep {
		t.Errorf("Expected Deep Sleep after 20 min, got %v", cr.GetCurrentPhase())
	}
}

func TestSleepDebtAccumulation(t *testing.T) {
	cr := NewCircadianRhythm()

	// Stay awake without sleeping
	for i := 0; i < 24; i++ {
		cr.Update(1.0, float64(i), i >= 20 || i < 6)
	}

	debt := cr.GetSleepDebt()

	if debt <= 0 {
		t.Error("Sleep debt should accumulate while awake")
	}
}

func TestSleepDebtReduction(t *testing.T) {
	cr := NewCircadianRhythm()

	// Accumulate significant sleep debt (48 hours awake to prevent early waking)
	for i := 0; i < 48; i++ {
		cr.Update(1.0, math.Mod(float64(i), 24), false)
	}

	initialDebt := cr.GetSleepDebt()

	// Sleep to reduce debt (force sleep for 4 hours to stay asleep)
	cr.InitiateSleep()
	for i := 0; i < 4; i++ {
		cr.Update(1.0, math.Mod(float64(22+i), 24), true)
	}

	finalDebt := cr.GetSleepDebt()

	if finalDebt >= initialDebt {
		t.Errorf("Sleep debt should decrease during sleep (initial: %.2f, final: %.2f)", initialDebt, finalDebt)
	}
}

func TestGetStats(t *testing.T) {
	cr := NewCircadianRhythm()

	stats := cr.GetStats()

	if stats.CurrentPhase != PhaseAwake {
		t.Errorf("Expected Awake in stats, got %v", stats.CurrentPhase)
	}

	if stats.IsAsleep {
		t.Error("Stats should show not asleep")
	}

	cr.InitiateSleep()
	stats = cr.GetStats()

	if !stats.IsAsleep {
		t.Error("Stats should show asleep")
	}
}

func TestSetOptimalSleepWindow(t *testing.T) {
	cr := NewCircadianRhythm()

	cr.SetOptimalSleepWindow(23.0, 7.0)

	if cr.OptimalSleepStart != 23.0 {
		t.Errorf("Expected sleep start 23.0, got %.2f", cr.OptimalSleepStart)
	}

	if cr.OptimalSleepEnd != 7.0 {
		t.Errorf("Expected sleep end 7.0, got %.2f", cr.OptimalSleepEnd)
	}
}

func TestReset(t *testing.T) {
	cr := NewCircadianRhythm()

	// Make changes
	cr.InitiateSleep()
	cr.Update(2.0, 22.0, true)

	// Reset
	cr.Reset()

	if cr.GetCurrentPhase() != PhaseAwake {
		t.Error("Phase should be reset to Awake")
	}

	if cr.GetSleepDebt() != 0 {
		t.Error("Sleep debt should be reset to 0")
	}

	if cr.GetSleepPressure() != 0 {
		t.Error("Sleep pressure should be reset to 0")
	}
}

func TestCircadianComponent(t *testing.T) {
	cr := NewCircadianRhythm()

	// Test at optimal sleep time (22:00)
	componentSleep := cr.calculateCircadianComponent(22.0)

	// Test at optimal wake time (6:00)
	componentWake := cr.calculateCircadianComponent(6.0)

	// Sleep time should have higher circadian drive
	if componentSleep < componentWake {
		t.Errorf("Circadian component at sleep time (%.2f) should be > wake time (%.2f)",
			componentSleep, componentWake)
	}
}

func TestTransitionToDrowsy(t *testing.T) {
	cr := NewCircadianRhythm()

	// Build up high homeostatic drive (sleep pressure is calculated from this)
	cr.HomeostaticDrive = 0.9

	cr.Update(0.1, 22.0, true)

	if cr.GetCurrentPhase() != PhaseDrowsy {
		t.Errorf("Should transition to Drowsy with high sleep pressure, got %v", cr.GetCurrentPhase())
	}
}

func TestAutoSleepFromDrowsy(t *testing.T) {
	cr := NewCircadianRhythm()

	cr.transitionToPhase(PhaseDrowsy)

	// Stay drowsy for over 30 minutes
	cr.Update(0.6, 22.0, true)

	if cr.GetCurrentPhase() != PhaseAsleep {
		t.Errorf("Should auto-sleep after being drowsy, got %v", cr.GetCurrentPhase())
	}
}

func TestCompleteSleepCycle(t *testing.T) {
	cr := NewCircadianRhythm()

	cr.InitiateSleep()

	// Simulate full sleep cycle (~90 minutes)
	// Asleep (20 min) -> Deep (40 min) -> REM (30 min)
	cr.Update(0.35, 22.0, true) // 21 min -> Deep Sleep
	if cr.GetCurrentPhase() != PhaseDeepSleep {
		t.Errorf("Should be in Deep Sleep, got %v", cr.GetCurrentPhase())
	}

	cr.Update(0.70, 22.5, true) // 42 min more -> REM Sleep
	if cr.GetCurrentPhase() != PhaseREMSleep {
		t.Errorf("Should be in REM Sleep, got %v", cr.GetCurrentPhase())
	}
}

func TestSleepQualityCalculation(t *testing.T) {
	cr := NewCircadianRhythm()

	cr.InitiateSleep()

	// Sleep for optimal duration
	for i := 0; i < 8; i++ {
		cr.Update(1.0, math.Mod(float64(22+i), 24), true)
	}

	// Wake up
	cr.ForceWake()

	quality := cr.GetSleepQuality()

	if quality <= 0.4 {
		t.Errorf("Sleep quality should be decent after full sleep, got %.2f", quality)
	}
}

func TestConcurrentAccess(t *testing.T) {
	cr := NewCircadianRhythm()
	done := make(chan bool, 3)

	// Goroutine 1: Update loop
	go func() {
		for i := 0; i < 50; i++ {
			cr.Update(0.1, 12.0, false)
		}
		done <- true
	}()

	// Goroutine 2: Initiate sleep/wake
	go func() {
		for i := 0; i < 50; i++ {
			if i%2 == 0 {
				cr.InitiateSleep()
			} else {
				cr.ForceWake()
			}
		}
		done <- true
	}()

	// Goroutine 3: Read stats
	go func() {
		for i := 0; i < 50; i++ {
			_ = cr.GetStats()
			_ = cr.GetSleepPressure()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should complete without panicking
}

func TestWakingPhase(t *testing.T) {
	cr := NewCircadianRhythm()

	cr.transitionToPhase(PhaseWaking)

	// Waking phase should be brief
	cr.Update(0.15, 6.0, false) // ~9 minutes

	if cr.GetCurrentPhase() != PhaseAwake {
		t.Errorf("Should transition to Awake after waking phase, got %v", cr.GetCurrentPhase())
	}
}

func TestSleepCycleCountTracking(t *testing.T) {
	cr := NewCircadianRhythm()

	initialCycles := cr.SleepCyclesCompleted

	cr.InitiateSleep()

	// Complete multiple sleep cycles (90 min each)
	for i := 0; i < 3; i++ {
		cr.Update(1.5, float64(22+i), true) // 90 minutes each
	}

	if cr.SleepCyclesCompleted <= initialCycles {
		t.Error("Sleep cycles should be tracked")
	}
}

func TestLifetimeSleepTracking(t *testing.T) {
	cr := NewCircadianRhythm()

	initialLifetime := cr.TotalLifetimeSleep

	cr.InitiateSleep()
	cr.Update(2.0, 22.0, true) // 2 hours of sleep

	if cr.TotalLifetimeSleep <= initialLifetime {
		t.Error("Lifetime sleep should accumulate")
	}
}
