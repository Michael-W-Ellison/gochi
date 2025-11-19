package simulation

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewNeed(t *testing.T) {
	need := NewNeed(types.NeedHunger)

	if need.Type != types.NeedHunger {
		t.Errorf("Expected NeedHunger, got %v", need.Type)
	}

	if need.CurrentLevel != 1.0 {
		t.Errorf("New need should start satisfied (1.0), got %.2f", need.CurrentLevel)
	}

	if need.DecayRate <= 0 {
		t.Error("Decay rate should be positive")
	}

	if need.SatisfactionRate <= 0 {
		t.Error("Satisfaction rate should be positive")
	}
}

func TestNeedDecay(t *testing.T) {
	need := NewNeed(types.NeedHunger)
	initialLevel := need.CurrentLevel

	need.Decay(1.0) // 1 game day

	if need.CurrentLevel >= initialLevel {
		t.Error("Need level should decrease after decay")
	}

	expectedLevel := initialLevel - need.DecayRate
	if need.CurrentLevel != expectedLevel {
		t.Errorf("Expected level %.2f, got %.2f", expectedLevel, need.CurrentLevel)
	}
}

func TestNeedSatisfy(t *testing.T) {
	need := NewNeed(types.NeedHunger)
	need.CurrentLevel = 0.5

	need.Satisfy(1.0, 10.0)

	if need.CurrentLevel <= 0.5 {
		t.Error("Need level should increase after satisfying")
	}

	if need.LastSatisfied != 10.0 {
		t.Errorf("LastSatisfied should be 10.0, got %.2f", need.LastSatisfied)
	}
}

func TestNeedClamp(t *testing.T) {
	need := NewNeed(types.NeedHunger)

	// Test upper bound
	need.CurrentLevel = 1.5
	need.Clamp()
	if need.CurrentLevel != 1.0 {
		t.Errorf("Need level should be clamped to 1.0, got %.2f", need.CurrentLevel)
	}

	// Test lower bound
	need.CurrentLevel = -0.5
	need.Clamp()
	if need.CurrentLevel != 0.0 {
		t.Errorf("Need level should be clamped to 0.0, got %.2f", need.CurrentLevel)
	}
}

func TestNeedIsCritical(t *testing.T) {
	need := NewNeed(types.NeedHunger)

	need.CurrentLevel = 0.1
	if !need.IsCritical() {
		t.Error("Need should be critical at level 0.1")
	}

	need.CurrentLevel = 0.5
	if need.IsCritical() {
		t.Error("Need should not be critical at level 0.5")
	}
}

func TestNeedIsWarning(t *testing.T) {
	need := NewNeed(types.NeedHunger)

	need.CurrentLevel = 0.3
	if !need.IsWarning() {
		t.Error("Need should be at warning level at 0.3")
	}

	need.CurrentLevel = 0.8
	if need.IsWarning() {
		t.Error("Need should not be at warning level at 0.8")
	}
}

func TestNeedIsSatisfied(t *testing.T) {
	need := NewNeed(types.NeedHunger)

	need.CurrentLevel = 0.8
	if !need.IsSatisfied() {
		t.Error("Need should be satisfied at level 0.8")
	}

	need.CurrentLevel = 0.5
	if need.IsSatisfied() {
		t.Error("Need should not be satisfied at level 0.5")
	}
}

func TestNeedGetUrgency(t *testing.T) {
	need := NewNeed(types.NeedHunger)

	need.CurrentLevel = 1.0
	urgency := need.GetUrgency()
	if urgency != 0.0 {
		t.Errorf("Urgency should be 0.0 when fully satisfied, got %.2f", urgency)
	}

	need.CurrentLevel = 0.0
	urgency = need.GetUrgency()
	if urgency != 1.0 {
		t.Errorf("Urgency should be 1.0 when critical, got %.2f", urgency)
	}

	need.CurrentLevel = 0.5
	urgency = need.GetUrgency()
	if urgency != 0.5 {
		t.Errorf("Urgency should be 0.5 at mid-level, got %.2f", urgency)
	}
}

func TestNewNeedsManager(t *testing.T) {
	nm := NewNeedsManager()

	if nm == nil {
		t.Fatal("NeedsManager should not be nil")
	}

	if len(nm.Needs) != 10 {
		t.Errorf("Expected 10 needs, got %d", len(nm.Needs))
	}

	// Verify all need types are present
	expectedNeeds := []types.NeedType{
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

	for _, needType := range expectedNeeds {
		if _, exists := nm.Needs[needType]; !exists {
			t.Errorf("Missing need type: %v", needType)
		}
	}
}

func TestNeedsManagerUpdate(t *testing.T) {
	nm := NewNeedsManager()

	initialHunger := nm.GetNeedLevel(types.NeedHunger)

	nm.Update(1.0) // 1 game day

	newHunger := nm.GetNeedLevel(types.NeedHunger)

	if newHunger >= initialHunger {
		t.Error("Hunger need should decrease after update")
	}
}

func TestNeedsManagerSatisfyNeed(t *testing.T) {
	nm := NewNeedsManager()

	nm.SetNeedLevel(types.NeedHunger, 0.3)
	initialLevel := nm.GetNeedLevel(types.NeedHunger)

	nm.SatisfyNeed(types.NeedHunger, 0.5, 10.0)

	newLevel := nm.GetNeedLevel(types.NeedHunger)

	if newLevel <= initialLevel {
		t.Error("Need level should increase after satisfaction")
	}
}

func TestNeedsManagerGetOverallWellbeing(t *testing.T) {
	nm := NewNeedsManager()

	wellbeing := nm.GetOverallWellbeing()

	if wellbeing < 0.0 || wellbeing > 1.0 {
		t.Errorf("Wellbeing should be between 0 and 1, got %.2f", wellbeing)
	}

	// New manager should have high wellbeing
	if wellbeing < 0.9 {
		t.Errorf("New manager should have high wellbeing, got %.2f", wellbeing)
	}

	// Set all needs to critical
	for needType := range nm.Needs {
		nm.SetNeedLevel(needType, 0.1)
	}

	wellbeing = nm.GetOverallWellbeing()
	if wellbeing > 0.3 {
		t.Errorf("Wellbeing should be low when all needs critical, got %.2f", wellbeing)
	}
}

func TestNeedsManagerGetCriticalNeeds(t *testing.T) {
	nm := NewNeedsManager()

	// Initially no critical needs
	critical := nm.GetCriticalNeeds()
	if len(critical) != 0 {
		t.Errorf("Expected 0 critical needs initially, got %d", len(critical))
	}

	// Make hunger critical
	nm.SetNeedLevel(types.NeedHunger, 0.1)
	critical = nm.GetCriticalNeeds()

	if len(critical) != 1 {
		t.Errorf("Expected 1 critical need, got %d", len(critical))
	}

	if critical[0] != types.NeedHunger {
		t.Errorf("Expected NeedHunger to be critical, got %v", critical[0])
	}
}

func TestNeedsManagerGetWarningNeeds(t *testing.T) {
	nm := NewNeedsManager()

	// Make thirst at warning level
	nm.SetNeedLevel(types.NeedThirst, 0.35)
	warning := nm.GetWarningNeeds()

	if len(warning) != 1 {
		t.Errorf("Expected 1 warning need, got %d", len(warning))
	}

	if warning[0] != types.NeedThirst {
		t.Errorf("Expected NeedThirst to be at warning, got %v", warning[0])
	}
}

func TestNeedsManagerGetMostUrgentNeed(t *testing.T) {
	nm := NewNeedsManager()

	// Set different urgency levels
	nm.SetNeedLevel(types.NeedHunger, 0.1)  // Very urgent
	nm.SetNeedLevel(types.NeedThirst, 0.5)  // Less urgent
	nm.SetNeedLevel(types.NeedSleep, 0.9)   // Not urgent

	mostUrgent := nm.GetMostUrgentNeed()

	if mostUrgent != types.NeedHunger {
		t.Errorf("Expected NeedHunger to be most urgent, got %v", mostUrgent)
	}
}

func TestNeedsManagerGetNeedsByPriority(t *testing.T) {
	nm := NewNeedsManager()

	needsByPriority := nm.GetNeedsByPriority()

	if len(needsByPriority) != 10 {
		t.Errorf("Expected 10 needs, got %d", len(needsByPriority))
	}

	// Verify sorting - critical priority needs should come first
	firstNeed := nm.GetNeed(needsByPriority[0])
	lastNeed := nm.GetNeed(needsByPriority[len(needsByPriority)-1])

	if firstNeed.Priority < lastNeed.Priority {
		t.Error("Needs should be sorted by priority (highest first)")
	}
}

func TestNeedsManagerGetAllNeedsStatus(t *testing.T) {
	nm := NewNeedsManager()

	nm.SetNeedLevel(types.NeedHunger, 0.1)  // Critical
	nm.SetNeedLevel(types.NeedThirst, 0.35) // Warning

	status := nm.GetAllNeedsStatus()

	if len(status.NeedLevels) != 10 {
		t.Errorf("Expected 10 need levels, got %d", len(status.NeedLevels))
	}

	if status.CriticalNeedCount != 1 {
		t.Errorf("Expected 1 critical need, got %d", status.CriticalNeedCount)
	}

	if status.WarningNeedCount != 1 {
		t.Errorf("Expected 1 warning need, got %d", status.WarningNeedCount)
	}

	if status.OverallWellbeing < 0.0 || status.OverallWellbeing > 1.0 {
		t.Errorf("Wellbeing should be 0-1, got %.2f", status.OverallWellbeing)
	}
}

func TestNeedsManagerHasCriticalNeeds(t *testing.T) {
	nm := NewNeedsManager()

	if nm.HasCriticalNeeds() {
		t.Error("New manager should not have critical needs")
	}

	nm.SetNeedLevel(types.NeedHunger, 0.1)

	if !nm.HasCriticalNeeds() {
		t.Error("Should have critical needs after setting one")
	}
}

func TestNeedsManagerHasWarningNeeds(t *testing.T) {
	nm := NewNeedsManager()

	if nm.HasWarningNeeds() {
		t.Error("New manager should not have warning needs")
	}

	nm.SetNeedLevel(types.NeedThirst, 0.35)

	if !nm.HasWarningNeeds() {
		t.Error("Should have warning needs after setting one")
	}
}

func TestNeedsManagerReset(t *testing.T) {
	nm := NewNeedsManager()

	// Decay all needs
	for i := 0; i < 10; i++ {
		nm.Update(1.0)
	}

	// Verify some needs have decayed
	if nm.GetNeedLevel(types.NeedThirst) > 0.9 {
		t.Error("Needs should have decayed")
	}

	// Reset
	nm.Reset()

	// All needs should be satisfied
	for needType := range nm.Needs {
		level := nm.GetNeedLevel(needType)
		if level != 1.0 {
			t.Errorf("Need %v should be at 1.0 after reset, got %.2f", needType, level)
		}
	}
}

func TestDifferentNeedDecayRates(t *testing.T) {
	nm := NewNeedsManager()

	thirstDecay := nm.GetNeed(types.NeedThirst).DecayRate
	cleanlinessDecay := nm.GetNeed(types.NeedCleanliness).DecayRate

	// Thirst should decay faster than cleanliness
	if thirstDecay <= cleanlinessDecay {
		t.Errorf("Thirst (%.2f) should decay faster than cleanliness (%.2f)",
			thirstDecay, cleanlinessDecay)
	}
}

func TestPriorityWeighting(t *testing.T) {
	nm := NewNeedsManager()

	// Set a high priority need to critical
	nm.SetNeedLevel(types.NeedThirst, 0.1) // Critical priority
	wellbeingWithCritical := nm.GetOverallWellbeing()

	// Reset and set a low priority need to critical
	nm.Reset()
	nm.SetNeedLevel(types.NeedCleanliness, 0.1) // Low priority
	wellbeingWithLow := nm.GetOverallWellbeing()

	// Critical priority need should have more impact
	if wellbeingWithCritical >= wellbeingWithLow {
		t.Errorf("Critical priority need should impact wellbeing more than low priority")
	}
}

func TestNeedsManagerConcurrentAccess(t *testing.T) {
	nm := NewNeedsManager()
	done := make(chan bool, 3)

	// Goroutine 1: Update loop
	go func() {
		for i := 0; i < 50; i++ {
			nm.Update(0.1)
		}
		done <- true
	}()

	// Goroutine 2: Satisfy needs
	go func() {
		for i := 0; i < 50; i++ {
			nm.SatisfyNeed(types.NeedHunger, 0.1, float64(i))
		}
		done <- true
	}()

	// Goroutine 3: Read stats
	go func() {
		for i := 0; i < 50; i++ {
			_ = nm.GetOverallWellbeing()
			_ = nm.GetCriticalNeeds()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should complete without panicking
}

func TestPriorityString(t *testing.T) {
	priorities := []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical}
	expected := []string{"Low", "Medium", "High", "Critical"}

	for i, priority := range priorities {
		str := priority.String()
		if str != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], str)
		}
	}
}
