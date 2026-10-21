package core

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// secondDayGoals returns a pet's goals on a day after its first
func secondDayGoals(t *testing.T, pet *DigitalPet, level float64) []Goal {
	t.Helper()
	pet.mu.Lock()
	defer pet.mu.Unlock()
	if len(pet.RecentGoalKinds) == 0 {
		pet.RecentGoalKinds = []string{goalClassic}
	}
	return pet.planGoalsLocked(level)
}

func TestActivityGoalsFollowThePet(t *testing.T) {
	// A rusty trick comes first
	pet := NewDigitalPet("Mochi", "owner")
	pet.TricksLearned = map[string]bool{"sit": true}
	pet.TrickSkills = map[string]float64{"sit": 0.8}
	if g := secondDayGoals(t, pet, 2)[1]; g.Kind != GoalPractice || g.Detail != "sit" {
		t.Errorf("A rusty trick: %+v", g)
	}

	// Then upkeep that is due
	pet = NewDigitalPet("Mochi", "owner")
	pet.Inventory = []string{"water_fountain"}
	if g := secondDayGoals(t, pet, 2)[1]; g.Kind != GoalUpkeep || g.Detail != fountainFilterUpkeep {
		t.Errorf("A fountain filter due: %+v", g)
	}

	// A breed prone to dental disease is asked to brush teeth
	chi := NewPetOfBreed("Chi", "owner", SpeciesDog, "chihuahua")
	if g := secondDayGoals(t, chi, 2)[1]; g.Kind != GoalTeeth {
		t.Errorf("A Chihuahua's goal: %+v", g)
	}
	// ... but not three days running
	chi.RecentGoalKinds = []string{GoalTeeth}
	if g := secondDayGoals(t, chi, 2)[1]; g.Kind == GoalTeeth {
		t.Error("The same activity goal shouldn't come straight back")
	}
}

func TestGoalEventsAndDetails(t *testing.T) {
	pet := NewDigitalPet("Mochi", "owner")
	pet.mu.Lock()
	pet.ensureGoalsLocked()
	pet.Goals.Goals[1] = Goal{Kind: GoalExplore, Detail: "beach", Target: 1, Reward: 5, Text: "Take Mochi to the beach"}
	before := pet.FriendshipPoints
	pet.goalEventLocked(GoalExplore, "park")
	if pet.Goals.Goals[1].Done {
		t.Error("A trip to the park doesn't count for the beach")
	}
	pet.goalEventLocked(GoalExplore, "beach")
	if !pet.Goals.Goals[1].Done || pet.FriendshipPoints != before+5 {
		t.Errorf("A trip to the beach: %+v", pet.Goals.Goals[1])
	}
	pet.mu.Unlock()
}

func TestTimedGoals(t *testing.T) {
	clock := useClock(t, time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "owner")
	pet.mu.Lock()
	defer pet.mu.Unlock()
	pet.ensureGoalsLocked()
	pet.Goals.Goals[2] = Goal{Kind: GoalCalm, Unit: "seconds", Target: 60, Reward: 5}
	pet.Biology.Vitals.Stress = 0.1
	pet.updateTimedGoalsLocked()
	for i := 0; i < 15; i++ {
		*clock = clock.Add(5 * time.Second)
		pet.updateTimedGoalsLocked()
	}
	if g := pet.Goals.Goals[2]; !g.Done {
		t.Errorf("A minute of calm: %+v", g)
	}

	// Time while stressed, or a long gap, doesn't count
	pet.Goals.Goals[2] = Goal{Kind: GoalCalm, Unit: "seconds", Target: 60, Reward: 5}
	pet.Biology.Vitals.Stress = 0.5
	*clock = clock.Add(5 * time.Second)
	pet.updateTimedGoalsLocked()
	pet.Biology.Vitals.Stress = 0.1
	*clock = clock.Add(time.Hour)
	pet.updateTimedGoalsLocked()
	if pet.Goals.Goals[2].Progress != 0 {
		t.Errorf("Stressed time and a long gap counted: %d", pet.Goals.Goals[2].Progress)
	}
}

func TestWeeklyGoal(t *testing.T) {
	clock := useClock(t, time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC)) // Week 11
	pet := NewDigitalPet("Mochi", "owner")
	week := pet.WeekGoal()
	if week.Week != "2026-W11" || week.Goal.Reward != weeklyPoints || week.Goal.Stars != weeklyStars {
		t.Fatalf("The week's goal: %+v", week)
	}
	pet.mu.Lock()
	pet.Weekly.Goal = Goal{Kind: WeeklyPlays, Target: 2, Reward: weeklyPoints, Stars: weeklyStars}
	pet.progressGoalsLocked(types.InteractionPlaying)
	pet.progressGoalsLocked(types.InteractionPlaying)
	done := pet.Weekly.Goal.Done
	completed := pet.completedGoals
	pet.mu.Unlock()
	if !done || len(completed) == 0 || completed[len(completed)-1].Stars != weeklyStars {
		t.Errorf("Two plays finish the goal: %v %+v", done, completed)
	}

	// Brushing counts once a day
	*clock = clock.AddDate(0, 0, 7)
	pet.mu.Lock()
	pet.ensureWeeklyLocked()
	pet.Weekly.Goal = Goal{Kind: WeeklyTeethDays, Target: 5}
	pet.goalEventLocked(GoalTeeth, "")
	pet.goalEventLocked(GoalTeeth, "")
	if pet.Weekly.Goal.Progress != 1 {
		t.Errorf("Teeth brushed twice in a day count once: %d", pet.Weekly.Goal.Progress)
	}
	pet.mu.Unlock()
}

func TestCareRecordAchievements(t *testing.T) {
	pet := NewDigitalPet("Mochi", "owner")
	pet.mu.Lock()
	pet.Biology.Vitals.Stress = 0.1
	pet.updateCareRecordsLocked(150)
	pet.updateCareRecordsLocked(150)
	pet.countLocked("reward_training", 50)
	pet.countLocked("early_treated", 3)
	pet.mu.Unlock()
	pet.Award("full_house")
	earned := map[string]bool{}
	for _, a := range pet.CheckAchievements() {
		earned[a.ID] = true
	}
	for _, id := range []string{"calm_companion", "patient_trainer", "vets_friend", "full_house"} {
		if !earned[id] {
			t.Errorf("%s should be earned", id)
		}
	}

	// A stressful moment ends the calm streak; scolding spoils patient training
	pet = NewDigitalPet("Rex", "owner")
	pet.mu.Lock()
	pet.updateCareRecordsLocked(200)
	pet.Biology.Vitals.Stress = 0.5
	pet.updateCareRecordsLocked(1)
	pet.countLocked("reward_training", 60)
	pet.countLocked("scolding", 1)
	pet.mu.Unlock()
	for _, a := range pet.CheckAchievements() {
		if a.ID == "calm_companion" || a.ID == "patient_trainer" {
			t.Errorf("%s shouldn't be earned", a.ID)
		}
	}
	if len(achievements) < 35 {
		t.Errorf("%d achievements", len(achievements))
	}
}
