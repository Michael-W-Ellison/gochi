package core

import (
	"math"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestTally(t *testing.T) {
	pet := NewDigitalPet("Mochi", "owner")
	applied := 0
	for i := 0; i < 3; i++ {
		if pet.Interact(types.InteractionPetting, 1).Applied {
			applied++
		}
	}
	pet.HibernationEnabled = true
	pet.Update(10000) // Hibernates
	tally := pet.Tally()
	if applied == 0 || tally.Care["Petting"] != applied || tally.Interactions != applied || !tally.Alive || tally.CauseOfDeath != "" {
		t.Errorf("Tally = %+v, want %d petting", tally, applied)
	}
	if tally.Hibernations != 1 || tally.Generation != 1 || tally.Species != SpeciesCat {
		t.Errorf("Tally = %+v", tally)
	}

	// The tally is a copy
	tally.Care["Petting"] = 100
	if pet.CareCounts["Petting"] != applied {
		t.Error("Changing a tally should not change the pet")
	}

	// Counts are kept in the save
	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Tally(); got.Care["Petting"] != applied || got.Hibernations != 1 {
		t.Errorf("Loaded tally = %+v", got)
	}

	pet.HibernationEnabled, pet.Hibernating = false, false
	pet.Update(10000) // Dies
	if tally := pet.Tally(); tally.Alive || tally.CauseOfDeath == "" {
		t.Errorf("A dead pet's tally should say how it died: %+v", tally)
	}
}

func TestInteractionIntensityIsLimited(t *testing.T) {
	for _, intensity := range []float64{1.7e308, -1e6, math.NaN(), math.Inf(1)} {
		pet := NewDigitalPet("Mochi", "owner")
		before := pet.Biology.Vitals.Happiness
		pet.Interact(types.InteractionPetting, intensity)
		if _, err := pet.Save(); err != nil {
			t.Errorf("Intensity %v made the pet unsaveable: %v", intensity, err)
		}
		if intensity < 0 && pet.Biology.Vitals.Happiness < before-0.01 {
			t.Errorf("Intensity %v should count as 0, happiness %v -> %v", intensity, before, pet.Biology.Vitals.Happiness)
		}
	}
}

func TestSaveReplacesValuesThatArentNumbers(t *testing.T) {
	pet := NewDigitalPet("Mochi", "owner")
	pet.Biology.Vitals.Happiness = math.NaN()
	pet.TotalPlayTime = math.Inf(1)
	pet.TrickSkills = map[string]float64{"sit": math.NaN()}
	data, err := pet.Save()
	if err != nil {
		t.Fatalf("The pet should still save: %v", err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Biology.Vitals.Happiness != 0 || loaded.TotalPlayTime != 0 || loaded.TrickSkills["sit"] != 0 {
		t.Errorf("Values that aren't numbers should be saved as 0: %v %v %v", loaded.Biology.Vitals.Happiness,
			loaded.TotalPlayTime, loaded.TrickSkills)
	}
}
