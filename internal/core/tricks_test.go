package core

import (
	"errors"
	"math"
	"testing"
)

// passTrickTime ages the pet by days and lets its tricks fade over them
func passTrickTime(pet *DigitalPet, days float64) {
	pet.Biology.Processes.Age += days
	pet.decayTricksLocked(days)
}

// learnTrick trains a trick until the pet knows it
func learnTrick(t *testing.T, pet *DigitalPet, trick string) {
	t.Helper()
	for i := 0; pet.TrickSkills[trick] < 1.0; i++ {
		if i > 50 {
			t.Fatalf("%s was not learned within 50 sessions", trick)
		}
		setCalmHealthyState(pet)
		if _, err := pet.TrainTrick(trick); err != nil {
			t.Fatalf("TrainTrick: %v", err)
		}
	}
}

func TestPracticedTricksStaySharp(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	learnTrick(t, pet, "sit")

	passTrickTime(pet, trickRustGrace)
	if pet.TrickSkills["sit"] != 1.0 {
		t.Errorf("A trick practiced within the grace period should not fade, skill %.3f", pet.TrickSkills["sit"])
	}

	// Performing counts as practice
	setCalmHealthyState(pet)
	if _, err := pet.PerformTrick("sit"); err != nil {
		t.Fatal(err)
	}
	passTrickTime(pet, trickRustGrace)
	if pet.TrickSkills["sit"] != 1.0 {
		t.Errorf("Performing should restart the grace period, skill %.3f", pet.TrickSkills["sit"])
	}
}

func TestUnpracticedTricksGetRusty(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	learnTrick(t, pet, "sit")

	// Halfway from sharp to the floor after the grace period
	half := (1.0 - rustyTrickFloor) / 2 / trickRustRate
	passTrickTime(pet, trickRustGrace+half)
	if got := pet.TrickSkills["sit"]; math.Abs(got-(1.0+rustyTrickFloor)/2) > 1e-9 {
		t.Errorf("Skill = %.3f, want %.3f", got, (1.0+rustyTrickFloor)/2)
	}
	tricks := pet.Tricks()
	if tricks[0].Learned || !tricks[0].Rusty {
		t.Errorf("Sit should be rusty: %+v", tricks[0])
	}
	setCalmHealthyState(pet)
	if _, err := pet.PerformTrick("sit"); !errors.Is(err, ErrTrickRusty) {
		t.Errorf("Performing a rusty trick: %v", err)
	}

	// A learned trick never fades below the floor
	passTrickTime(pet, 10*trickRustGrace)
	if pet.TrickSkills["sit"] != rustyTrickFloor {
		t.Errorf("Skill = %.3f, want the floor %.2f", pet.TrickSkills["sit"], rustyTrickFloor)
	}

	// Practice brings it back without counting as learning it anew
	var result TrickResult
	for i := 0; pet.TrickSkills["sit"] < 1.0; i++ {
		if i > 3 {
			t.Fatal("A rusty trick should come back within a few sessions")
		}
		setCalmHealthyState(pet)
		var err error
		if result, err = pet.TrainTrick("sit"); err != nil {
			t.Fatal(err)
		}
	}
	if result.JustLearned || result.Text != "Biscuit remembers Sit again! 🐾" {
		t.Errorf("Relearning: JustLearned=%v, text %q", result.JustLearned, result.Text)
	}
	if tricks := pet.Tricks(); !tricks[0].Learned || tricks[0].Rusty {
		t.Errorf("Sit should be sharp again: %+v", tricks[0])
	}
}

func TestPartialTrainingFadesCompletely(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	setCalmHealthyState(pet)
	if _, err := pet.TrainTrick("play_dead"); err != nil {
		t.Fatal(err)
	}
	if pet.TrickSkills["play_dead"] >= 1.0 {
		t.Fatal("One session should not teach the hardest trick")
	}
	passTrickTime(pet, trickRustGrace+1/trickRustRate)
	if pet.TrickSkills["play_dead"] != 0 {
		t.Errorf("Unlearned progress should fade to nothing, skill %.3f", pet.TrickSkills["play_dead"])
	}
	if pet.Tricks()[4].Rusty {
		t.Error("A trick never learned is not rusty")
	}
}

func TestTricksFromOlderSavesStartTheirGracePeriod(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.TrickSkills = map[string]float64{"sit": 1.0} // No practice or learning recorded
	passTrickTime(pet, 1)
	if !pet.TricksLearned["sit"] || pet.TrickSkills["sit"] != 1.0 {
		t.Errorf("A known trick should be recorded as learned and kept: %+v", pet.TrickSkills)
	}
	passTrickTime(pet, trickRustGrace-1)
	if pet.TrickSkills["sit"] != 1.0 {
		t.Errorf("The grace period should start when the save is first updated, skill %.3f", pet.TrickSkills["sit"])
	}
}

func TestRustyTrickEvent(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	learnTrick(t, pet, "sit")
	before := pet.Conditions()
	passTrickTime(pet, trickRustGrace+1)

	events := ConditionEvents(before, pet.Conditions())
	if len(events) != 1 || events[0].Type != EventTrickRusty ||
		events[0].Message != "Biscuit is getting rusty at Sit. Practice it to keep it sharp." {
		t.Errorf("Expected a rusty trick event, got %+v", events)
	}
}
