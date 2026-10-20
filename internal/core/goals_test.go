package core

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestGoalsAskForNeglectedCare(t *testing.T) {
	useClock(t, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "user123")
	// A week of everything but brushing
	for _, care := range []types.InteractionType{types.InteractionFeeding, types.InteractionDrinking,
		types.InteractionCleaning, types.InteractionPlaying} {
		for i := 0; i < 14; i++ {
			pet.History = append(pet.History, HistoryEntry{Time: now().Add(-time.Hour), Action: care.String()})
		}
	}
	pet.Biology.Vitals.Happiness = 0.5

	goals, level := pet.DailyGoals()
	if level != 1 || len(goals) != 3 {
		t.Fatalf("Got %d goals at level %v, want 3 at level 1", len(goals), level)
	}
	if g := goals[0]; g.Kind != GoalCare || g.Care != types.InteractionGrooming || g.Target != 1 {
		t.Errorf("First goal should be one brushing, got %+v", g)
	}
	if g := goals[1]; g.Kind != GoalCare || g.Care != types.InteractionPlaying || g.Target != 2 {
		t.Errorf("Second goal should be playing twice, got %+v", g)
	}
	if g := goals[2]; g.Kind != GoalHappiness || g.Target != 65 {
		t.Errorf("Third goal should be 65%% happiness, got %+v", g)
	}
	for _, g := range goals {
		if g.Reward != pointsPerLevel {
			t.Errorf("Level 1 goals should earn %d points, got %+v", pointsPerLevel, g)
		}
	}
}

func TestGoalProgressPaysOnce(t *testing.T) {
	useClock(t, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "user123")
	pet.Biology.Vitals.Happiness = 0.5
	pet.Goals = &GoalBoard{Day: today(), Level: 1, Goals: []Goal{
		careGoal(pet.Name, types.InteractionDrinking, 2, 5),
		{Kind: GoalHappiness, Target: 95, Reward: 5},
	}}

	pet.Interact(types.InteractionDrinking, 1.0)
	if goals, _ := pet.DailyGoals(); goals[0].Progress != 1 || goals[0].Done {
		t.Fatalf("One drink should count 1 of 2, got %+v", goals[0])
	}
	if done := pet.TakeCompletedGoals(); len(done) != 0 {
		t.Errorf("Nothing should be completed yet, got %+v", done)
	}

	before := pet.FriendshipPoints
	pet.Interact(types.InteractionDrinking, 1.0)
	gained := pet.FriendshipPoints - before
	if gained < 5 {
		t.Errorf("Completing the goal should pay 5 bonus points, gained %d", gained)
	}
	done := pet.TakeCompletedGoals()
	if len(done) != 1 || done[0].Care != types.InteractionDrinking {
		t.Fatalf("The water goal should be completed, got %+v", done)
	}
	if again := pet.TakeCompletedGoals(); len(again) != 0 {
		t.Errorf("A completed goal should be reported once, got %+v", again)
	}

	// More drinks don't pay again
	before = pet.FriendshipPoints
	pet.Interact(types.InteractionDrinking, 1.0)
	if goals, _ := pet.DailyGoals(); goals[0].Progress != 2 {
		t.Errorf("A done goal should stop counting, got %+v", goals[0])
	}
	if pet.FriendshipPoints-before >= 5 {
		t.Errorf("A done goal should not pay again")
	}
}

func TestHappinessAndLearnGoals(t *testing.T) {
	useClock(t, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "user123")
	pet.Biology.Vitals.Happiness = 0.5
	pet.Goals = &GoalBoard{Day: today(), Level: 4, LearnedAtStart: len(pet.TricksLearned)}
	pet.Goals.Goals = pet.planGoalsLocked(4)
	if g := pet.Goals.Goals[1]; g.Kind != GoalLearn || g.Reward != 4*pointsPerLevel {
		t.Fatalf("A level 4 activity goal should be learning a trick, got %+v", g)
	}

	pet.Biology.Vitals.Happiness = float64(pet.Goals.Goals[2].Target) / 100
	if pet.TricksLearned == nil {
		pet.TricksLearned = make(map[string]bool)
	}
	pet.TricksLearned[tricks[0].Name] = true
	done := pet.TakeCompletedGoals()
	if len(done) != 2 || done[0].Kind != GoalLearn || done[1].Kind != GoalHappiness {
		t.Errorf("Learning a trick and reaching the happiness target should complete both goals, got %+v", done)
	}
}

func TestGoalLevelAdapts(t *testing.T) {
	clock := useClock(t, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	tests := []struct {
		name  string
		done  int
		tried bool
		start float64
		want  float64
	}{
		{"two of three done", 2, true, 2, 2.5},
		{"all done at the top", 3, true, 5, 5},
		{"one done", 1, true, 2, 2},
		{"tried and missed", 0, true, 2, 1.5},
		{"tried and missed at the bottom", 0, true, 1, 1},
		{"not tried", 0, false, 2, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pet := NewDigitalPet("Mochi", "user123")
			board := &GoalBoard{Day: clock.AddDate(0, 0, -1).Format("2006-01-02"), Level: tt.start}
			for i := 0; i < 3; i++ {
				g := Goal{Kind: GoalCare, Target: 2}
				if i < tt.done {
					g.Done, g.Progress = true, 2
				} else if tt.tried {
					g.Progress = 1
				}
				board.Goals = append(board.Goals, g)
			}
			pet.Goals = board
			if _, level := pet.DailyGoals(); level != tt.want {
				t.Errorf("Level = %v, want %v", level, tt.want)
			}
		})
	}
}

func TestGoalsSurviveSaving(t *testing.T) {
	useClock(t, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	pet := NewDigitalPet("Mochi", "user123")
	goals, _ := pet.DailyGoals()
	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := loaded.DailyGoals(); len(got) != len(goals) || got[0].Text != goals[0].Text {
		t.Errorf("Goals after loading = %+v, want %+v", got, goals)
	}
}
