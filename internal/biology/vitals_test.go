package biology

import (
	"testing"
)

func TestNewVitalStats(t *testing.T) {
	vitals := NewVitalStats()

	if vitals.Health != 1.0 {
		t.Errorf("Expected Health to be 1.0, got %.2f", vitals.Health)
	}

	if vitals.Energy != 1.0 {
		t.Errorf("Expected Energy to be 1.0, got %.2f", vitals.Energy)
	}
}

func TestVitalStatsClamp(t *testing.T) {
	vitals := &VitalStats{
		Health:    1.5,  // Over max
		Energy:    -0.5, // Under min
		Happiness: 0.5,  // Normal
	}

	vitals.Clamp()

	if vitals.Health != 1.0 {
		t.Errorf("Expected Health to be clamped to 1.0, got %.2f", vitals.Health)
	}

	if vitals.Energy != 0.0 {
		t.Errorf("Expected Energy to be clamped to 0.0, got %.2f", vitals.Energy)
	}

	if vitals.Happiness != 0.5 {
		t.Errorf("Expected Happiness to remain 0.5, got %.2f", vitals.Happiness)
	}
}

func TestGetOverallWellbeing(t *testing.T) {
	vitals := NewVitalStats()
	wellbeing := vitals.GetOverallWellbeing()

	if wellbeing < 0.0 || wellbeing > 1.0 {
		t.Errorf("Wellbeing should be between 0 and 1, got %.2f", wellbeing)
	}

	if wellbeing < 0.7 {
		t.Errorf("New pet should have high wellbeing, got %.2f", wellbeing)
	}
}

func TestIsCritical(t *testing.T) {
	vitals := NewVitalStats()

	if vitals.IsCritical(0.2) {
		t.Error("New pet should not be in critical state")
	}

	vitals.Health = 0.1
	if !vitals.IsCritical(0.2) {
		t.Error("Pet with low health should be critical")
	}
}

func TestGetCriticalStats(t *testing.T) {
	vitals := &VitalStats{
		Health:      0.1,
		Energy:      0.9,
		Hydration:   0.15,
		Nutrition:   0.5,
		Happiness:   0.5,
		Stress:      0.1,
		Fatigue:     0.1,
		Cleanliness: 0.5,
	}

	critical := vitals.GetCriticalStats(0.2)

	// Health and Hydration should be critical
	if len(critical) < 2 {
		t.Errorf("Expected at least 2 critical stats, got %d", len(critical))
	}

	// Verify Health is in the list
	found := false
	for _, stat := range critical {
		if stat == "Health" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Health should be in critical stats")
	}
}

func TestNewBiologicalSystems(t *testing.T) {
	bio := NewBiologicalSystems()

	if !bio.IsAlive {
		t.Error("New pet should be alive")
	}

	if bio.Vitals == nil {
		t.Error("Vitals should be initialized")
	}

	if bio.Processes == nil {
		t.Error("Processes should be initialized")
	}
}

func TestBiologicalSystemsUpdate(t *testing.T) {
	bio := NewBiologicalSystems()
	initialEnergy := bio.Vitals.Energy
	initialAge := bio.Processes.Age

	bio.Update(1.0)

	if bio.Processes.Age <= initialAge {
		t.Error("Age should increase after update")
	}

	if bio.Vitals.Energy >= initialEnergy {
		t.Error("Energy should decrease after update")
	}
}

func TestDeathConditions(t *testing.T) {
	bio := NewBiologicalSystems()
	bio.Vitals.Health = 0.0
	bio.CheckDeathConditions() // Manually trigger death check

	if bio.IsAlive {
		t.Error("Pet should die when health reaches 0")
	}

	if bio.CauseOfDeath == "" {
		t.Error("Cause of death should be recorded")
	}
}

func TestGetAgeInDays(t *testing.T) {
	bio := NewBiologicalSystems()
	bio.Processes.Age = 5.5

	age := bio.GetAgeInDays()
	if age != 5.5 {
		t.Errorf("Expected age 5.5, got %.2f", age)
	}
}
