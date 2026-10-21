package core

import (
	"fmt"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Adaptive goals: each calendar day the pet gets three goals sized to how
// its owner has been doing. One asks for the part of the daily routine the
// owner has given least over the last week, one for an activity, and one
// for the pet's happiness. Completing two or three of a day's goals makes
// the next day's harder; completing none (after trying) makes them easier.
// Each completed goal earns bonus Friendship Points.

// Goal kinds (more in goalkinds.go)
const (
	GoalCare      = "care"      // Give a kind of care a number of times
	GoalLearn     = "learn"     // Learn a new trick
	GoalHappiness = "happiness" // Get the pet's happiness to a level
)

// Goal is one of the day's goals
type Goal struct {
	Kind     string                `json:"kind"`
	Care     types.InteractionType `json:"care"` // For care goals
	Text     string                `json:"text"`
	Icon     string                `json:"icon"`
	Detail   string                `json:"detail,omitempty"` // What it is about: a place, trick, upkeep or change
	Target   int                   `json:"target"`           // Times, happiness in percent, or seconds of play
	Progress int                   `json:"progress"`
	Unit     string                `json:"unit,omitempty"`  // "seconds" for goals kept up over time
	Reward   int                   `json:"reward"`          // Bonus Friendship Points
	Stars    int                   `json:"stars,omitempty"` // Home Stars too (the weekly goal)
	Done     bool                  `json:"done"`
}

// GoalBoard is the pet's goals for a day
type GoalBoard struct {
	Day            string    `json:"day"`
	Started        time.Time `json:"started"` // On the play clock
	Level          float64   `json:"level"`   // Difficulty, 1.0 to 5.0
	Goals          []Goal    `json:"goals"`
	LearnedAtStart int       `json:"learned_at_start"` // Tricks known when the day began
	TimedAt        time.Time `json:"timed_at"`         // Play clock when goals kept up over time last counted
}

// Goal difficulty
const (
	minGoalLevel   = 1.0
	maxGoalLevel   = 5.0
	goalLevelStep  = 0.5
	pointsPerLevel = 5 // Reward per level of difficulty
	goalHistory    = 7 * 24 * time.Hour

	// minGoalDay is how long the game must have run since a day's goals
	// started before the next day's can, so moving the computer's clock
	// forward can't bring new goals straight away
	minGoalDay = 15 * time.Minute
)

// careGoalTexts describe care goals; %s is the pet's name, %d the times
var careGoalTexts = map[types.InteractionType]struct{ icon, text string }{
	types.InteractionFeeding:                 {"🍽️", "Feed %s %d measured meals"},
	types.InteractionDrinking:                {"💧", "Give %s fresh water %d times"},
	types.InteractionCleaning:                {"🧹", "Clean up after %s %d times"},
	types.InteractionWalking:                 {"🦮", "Take %s for %d walks"},
	types.InteractionPlaying:                 {"🎾", "Play with %s %d times"},
	types.InteractionGrooming:                {"🪮", "Brush %s %d times"},
	types.InteractionTraining:                {"🎓", "Practice tricks with %s %d times"},
	types.InteractionEnvironmentalEnrichment: {"🧭", "Take %s exploring %d times"},
}

// careGoal makes a goal to give care a number of times
func careGoal(name string, care types.InteractionType, times int, reward int) Goal {
	t := careGoalTexts[care]
	return Goal{Kind: GoalCare, Care: care, Icon: t.icon, Text: fmt.Sprintf(t.text, name, times), Target: times, Reward: reward}
}

// recentCareLocked counts each kind of care in the pet's history over the
// last week. Caller must hold p.mu.
func (p *DigitalPet) recentCareCountsLocked() map[string]int {
	counts := make(map[string]int)
	since := now().Add(-goalHistory)
	for _, h := range p.History {
		if h.Time.After(since) && h.Action != "event" && h.Action != "achievement" {
			counts[h.Action]++
		}
	}
	return counts
}

// planGoalsLocked makes a day's goals at the board's level. Caller must
// hold p.mu.
func (p *DigitalPet) planGoalsLocked(level float64) []Goal {
	reward := int(level+0.5) * pointsPerLevel
	steps := int(level - minGoalLevel) // 0 to 4

	// The routine care given least, relative to how often it's due
	counts := p.recentCareCountsLocked()
	routine := p.defLocked().DailyRoutine
	neglected := routine[0]
	worst := 2.0
	for _, item := range routine {
		ratio := float64(counts[item.Care.String()]) / float64(item.Target*7)
		if ratio < worst {
			neglected, worst = item, ratio
		}
	}
	goals := []Goal{careGoal(p.Name, neglected.Care, neglected.Target+steps/2, reward)}

	// An activity: on the first day the classic ones below; after that
	// whichever suits the pet best that wasn't asked for lately
	if g, ok := p.chooseActivityGoalLocked(level, reward); ok {
		goals = append(goals, g)
		goals = append(goals, p.wellbeingGoalLocked(level, reward))
		return goals
	}
	switch {
	case level >= 4 && len(p.TricksLearned) < len(tricks):
		goals = append(goals, Goal{Kind: GoalLearn, Icon: "🌟", Text: fmt.Sprintf("Teach %s a new trick", p.Name),
			Target: 1, Reward: reward})
	case level >= 3:
		goals = append(goals, careGoal(p.Name, types.InteractionEnvironmentalEnrichment, steps-1, reward))
	case level >= 2:
		goals = append(goals, careGoal(p.Name, types.InteractionTraining, 2+steps, reward))
	default:
		activity := types.InteractionPlaying
		if neglected.Care == types.InteractionPlaying {
			activity = types.InteractionTraining
		}
		goals = append(goals, careGoal(p.Name, activity, 2, reward))
	}

	// Happiness: higher than it is now, so there is something to do
	target := min(max(60+5*int(level+0.5), int(p.Biology.Vitals.Happiness*100)+10), 95)
	goals = append(goals, Goal{Kind: GoalHappiness, Icon: "😊", Target: target, Reward: reward,
		Text: fmt.Sprintf("Get %s's happiness to %d%%", p.Name, target)})
	return goals
}

// ensureGoalsLocked starts a new day's goals when the day changes,
// adapting the level to how the last day went. Caller must hold p.mu.
func (p *DigitalPet) ensureGoalsLocked() {
	day, played := today(), p.playNowLocked()
	if p.Goals != nil && (day <= p.Goals.Day || played.Sub(p.Goals.Started) < minGoalDay) {
		return // Still the same day, the clock went back, or too soon
	}
	level := minGoalLevel
	if p.Goals != nil {
		level = p.Goals.Level
		done, tried := 0, false
		for _, g := range p.Goals.Goals {
			if g.Done {
				done++
			}
			tried = tried || g.Progress > 0
		}
		switch {
		case done*3 >= len(p.Goals.Goals)*2:
			level += goalLevelStep
		case done == 0 && tried:
			level -= goalLevelStep
		}
		level = min(max(level, minGoalLevel), maxGoalLevel)
	}
	p.Goals = &GoalBoard{Day: day, Started: played, Level: level, LearnedAtStart: len(p.TricksLearned)}
	p.Goals.Goals = p.planGoalsLocked(level)
	p.rememberGoalKindsLocked()
	p.checkGoalsLocked()
}

// progressGoalsLocked counts care towards the day's goals. Caller must
// hold p.mu.
func (p *DigitalPet) progressGoalsLocked(interactionType types.InteractionType) {
	p.ensureGoalsLocked()
	for i := range p.Goals.Goals {
		g := &p.Goals.Goals[i]
		if g.Kind == GoalCare && g.Care == interactionType && !g.Done {
			g.Progress++
		}
	}
	p.weeklyEventLocked(interactionType.String())
	p.checkGoalsLocked()
}

// checkGoalsLocked marks goals done and pays their rewards. Caller must
// hold p.mu.
func (p *DigitalPet) checkGoalsLocked() {
	if p.Goals == nil || !p.Biology.IsAlive {
		return
	}
	for i := range p.Goals.Goals {
		g := &p.Goals.Goals[i]
		if g.Done {
			continue
		}
		switch g.Kind {
		case GoalHappiness:
			g.Progress = max(g.Progress, int(p.Biology.Vitals.Happiness*100))
		case GoalLearn:
			if len(p.TricksLearned) > p.Goals.LearnedAtStart {
				g.Progress = 1
			}
		}
		if g.Progress >= g.Target && g.Target > 0 {
			g.Done = true
			p.FriendshipPoints += g.Reward
			p.PointsEarned += g.Reward
			p.addHistoryLocked("goal", fmt.Sprintf("%s %s", g.Icon, g.Text), fmt.Sprintf("+%d points", g.Reward))
			p.completedGoals = append(p.completedGoals, *g)
		}
	}
	// A day all of the goals were finished counts towards Daily Carer
	all := len(p.Goals.Goals) > 0
	for _, g := range p.Goals.Goals {
		all = all && g.Done
	}
	if all && p.LastGoalDay != p.Goals.Day {
		p.LastGoalDay = p.Goals.Day
		p.GoalDays++
		p.weeklyEventLocked(WeeklyGoalDays)
	}
	p.checkWeeklyLocked()
}

// DailyGoals returns today's goals and their difficulty level
func (p *DigitalPet) DailyGoals() ([]Goal, float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ensureGoalsLocked()
	p.checkGoalsLocked()
	return append([]Goal(nil), p.Goals.Goals...), p.Goals.Level
}

// TakeCompletedGoals returns the goals completed since it was last called
func (p *DigitalPet) TakeCompletedGoals() []Goal {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Goals != nil {
		p.checkGoalsLocked()
	}
	completed := p.completedGoals
	p.completedGoals = nil
	return completed
}
