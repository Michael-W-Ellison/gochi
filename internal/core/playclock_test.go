package core

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Time while the game is closed doesn't count towards reward waits
func TestPlayClockStopsWhileClosed(t *testing.T) {
	clock := useClock(t, time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "owner")
	if pet.Interact(types.InteractionPetting, 1).PointsEarned == 0 {
		t.Fatal("The first petting should earn points")
	}
	*clock = clock.Add(time.Minute)
	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}

	// Closed for a day (or the computer's clock moved a day ahead)
	*clock = clock.Add(24 * time.Hour)
	pet, err = Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if pet.Interact(types.InteractionPetting, 1).PointsEarned != 0 {
		t.Error("Time while closed shouldn't end the wait")
	}
	*clock = clock.Add(CareRewardCooldown)
	if pet.Interact(types.InteractionPetting, 1).PointsEarned == 0 {
		t.Error("Time playing should end the wait")
	}
}

// A new day's goals need some play since the last day's started, and the
// clock going back keeps the goals there are
func TestGoalDaysNeedPlay(t *testing.T) {
	clock := useClock(t, time.Date(2026, 5, 1, 23, 55, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "owner")
	first, _ := pet.DailyGoals()
	day := pet.Goals.Day

	*clock = clock.Add(10 * time.Minute) // Past midnight, but little play
	pet.DailyGoals()
	if pet.Goals.Day != day {
		t.Error("New goals shouldn't start so soon")
	}
	*clock = clock.Add(minGoalDay)
	pet.DailyGoals()
	if pet.Goals.Day == day {
		t.Error("New goals should start after enough play on a new day")
	}
	day = pet.Goals.Day

	*clock = clock.Add(-72 * time.Hour) // The computer's clock goes back
	pet.DailyGoals()
	if pet.Goals.Day != day {
		t.Errorf("Going back in time shouldn't change the goals: %s", pet.Goals.Day)
	}
	if len(first) == 0 {
		t.Error("There should be goals")
	}
}
