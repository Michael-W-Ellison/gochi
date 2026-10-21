package core

import (
	"fmt"
	"slices"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// More kinds of goal. After a pet's first day, the day's activity goal is
// whichever suits the pet best and wasn't asked for in the last three
// days: brushing teeth for a breed prone to dental disease, upkeep that is
// due, a rusty trick, a place to explore, something new at home. The
// wellbeing goal is happiness, or for a stressed pet a calm spell, for a
// pet that feels the heat or cold staying comfortable, and at higher
// levels steady care. Goals that are kept up over time count minutes of
// play. Each week also has one larger goal, worth Home Stars too.

// More goal kinds
const (
	GoalTeeth    = "teeth"    // Brush the teeth
	GoalEyes     = "eyes"     // Wipe the eyes
	GoalUpkeep   = "upkeep"   // Renew something that is due (Detail: the upkeep)
	GoalEnrich   = "enrich"   // Make a change at home not made lately (Detail: the change)
	GoalExplore  = "explore"  // Take a trip (Detail: the place)
	GoalPractice = "practice" // Practice a rusty trick (Detail: the trick)
	GoalMeet     = "meet"     // Meet another pet
	GoalGroup    = "group"    // A group activity
	GoalShow     = "show"     // Enter a show (Detail: the show)
	GoalCalm     = "calm"     // Keep stress low for a while
	GoalComfort  = "comfort"  // Stay comfortable in hot or cold weather for a while
	GoalSteady   = "steady"   // Keep food and water above half for a while
	goalClassic  = "classic"  // The first day's activity goals (care, learning)
)

// Weekly goal kinds
const (
	WeeklyTeethDays = "teeth_days" // Brush the teeth on a number of days
	WeeklyTrips     = "trips"      // Take trips
	WeeklyTricks    = "tricks"     // Learn tricks
	WeeklyGoalDays  = "goal_days"  // Finish all of a day's goals on a number of days
	WeeklyPlays     = "plays"      // Play a number of times
	WeeklyRibbon    = "ribbon"     // Win a ribbon at a show
)

// Weekly goal rewards
const (
	weeklyPoints = 100
	weeklyStars  = 15
)

// How long goals kept up over time last, in seconds of play
const (
	calmGoalSeconds   = 20 * 60
	steadyGoalSeconds = 30 * 60
	timedGoalMaxStep  = 10 * time.Second // Longer gaps between updates don't count
)

// recentGoalDays is how many days' activity goals aren't repeated
const recentGoalDays = 3

// WeeklyGoal is the week's larger goal
type WeeklyGoal struct {
	Week    string `json:"week"` // ISO year and week, e.g. 2026-W43
	Goal    Goal   `json:"goal"`
	LastDay string `json:"last_day,omitempty"` // The last day counted for goals counted in days
}

// goalCandidate is a goal that could be the day's activity goal, and how
// well it suits the pet now
type goalCandidate struct {
	goal  Goal
	score float64
}

// chooseActivityGoalLocked picks the day's activity goal, or reports
// false on the pet's first day (when the classic goals are used). Caller
// must hold p.mu.
func (p *DigitalPet) chooseActivityGoalLocked(level float64, reward int) (Goal, bool) {
	if len(p.RecentGoalKinds) == 0 {
		return Goal{}, false
	}
	var candidates []goalCandidate
	add := func(score float64, g Goal) {
		g.Reward = reward
		if g.Target == 0 {
			g.Target = 1
		}
		if slices.Contains(p.RecentGoalKinds, g.Kind) {
			score -= 1
		}
		candidates = append(candidates, goalCandidate{g, score})
	}
	ph := p.Biology.Physiology
	p.Biology.EnsurePhysiology()

	// The classic activity
	classic := Goal{Kind: goalClassic}
	add(1.0, classic)

	teeth := 0.6 + 0.6*p.Tendencies[TendTeeth]
	if ph.DentalHealth < 0.6 {
		teeth += 0.6
	}
	add(teeth, Goal{Kind: GoalTeeth, Icon: "🪥", Text: fmt.Sprintf("Brush %s's teeth (pet toothpaste, in the pantry)", p.Name)})
	if p.Tendencies[TendEyes] > 0 || ph.EyeHealth < 0.8 {
		eyes := 0.8
		if ph.EyeHealth < 0.6 {
			eyes += 0.5
		}
		add(eyes, Goal{Kind: GoalEyes, Icon: "👁️", Text: fmt.Sprintf("Wipe %s's eyes (eye wipes, in the pantry)", p.Name)})
	}
	if id, name := p.upkeepDueLocked(); id != "" {
		add(1.3, Goal{Kind: GoalUpkeep, Detail: id, Icon: "🧺", Text: "Renew the " + name})
	}
	if c, ok := p.staleChangeLocked(); ok {
		add(0.8, Goal{Kind: GoalEnrich, Detail: c.ID, Icon: c.Icon, Text: c.Name + " for " + p.Name})
	}
	if place, name := p.placeToExploreLocked(); place != "" {
		add(0.8, Goal{Kind: GoalExplore, Detail: place, Icon: "🧭", Text: fmt.Sprintf("Take %s to %s", p.Name, name)})
	}
	if t, ok := p.rustyTrickLocked(); ok {
		add(1.5, Goal{Kind: GoalPractice, Detail: t.ID, Icon: "🎓", Text: fmt.Sprintf("Practice '%s' with %s", t.Name, p.Name)})
	}
	if len(p.KnownPets) > 0 {
		add(0.6, Goal{Kind: GoalMeet, Icon: "🤝", Text: fmt.Sprintf("Let %s meet another of your pets", p.Name)})
		add(0.5, Goal{Kind: GoalGroup, Icon: "🐾", Text: fmt.Sprintf("Do a group activity with %s", p.Name)})
	}

	for _, s := range p.showsForLocked() {
		if s.Fee > 0 && p.showProblemLocked(s) == "" {
			add(0.7, Goal{Kind: GoalShow, Detail: s.ID, Icon: s.Icon, Text: fmt.Sprintf("Enter %s in the %s", p.Name, s.Name)})
			break
		}
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	if best.goal.Kind == goalClassic {
		return Goal{}, false
	}
	return best.goal, true
}

// wellbeingGoalLocked is the day's goal for how the pet feels. Caller must
// hold p.mu.
func (p *DigitalPet) wellbeingGoalLocked(level float64, reward int) Goal {
	v := p.Biology.Vitals
	p.ensureEnvironmentLocked()
	temp := p.Environment.Weather.Current.Temperature
	minutes := func(s int) int { return s / 60 }
	switch {
	case v.Stress > 0.4:
		return Goal{Kind: GoalCalm, Icon: "🧘", Unit: "seconds", Target: calmGoalSeconds, Reward: reward,
			Text: fmt.Sprintf("Keep %s calm (stress under 30%%) for %d minutes", p.Name, minutes(calmGoalSeconds))}
	case (temp > 27 && p.Tendencies[TendShortAirways]+p.Tendencies[TendFeelsHeat] > 0) ||
		(temp < 10 && p.Tendencies[TendFeelsCold] > 0):
		word := "cool"
		if temp < 10 {
			word = "warm"
		}
		return Goal{Kind: GoalComfort, Icon: "🌡️", Unit: "seconds", Target: calmGoalSeconds, Reward: reward,
			Text: fmt.Sprintf("Keep %s %s and relaxed (stress under 40%%) for %d minutes", p.Name, word, minutes(calmGoalSeconds))}
	case level >= 3 && p.LastWellbeingGoal == GoalHappiness:
		return Goal{Kind: GoalSteady, Icon: "📈", Unit: "seconds", Target: steadyGoalSeconds, Reward: reward,
			Text: fmt.Sprintf("Keep %s's food and water above half for %d minutes", p.Name, minutes(steadyGoalSeconds))}
	}
	target := min(max(60+5*int(level+0.5), int(v.Happiness*100)+10), 95)
	return Goal{Kind: GoalHappiness, Icon: "😊", Target: target, Reward: reward,
		Text: fmt.Sprintf("Get %s's happiness to %d%%", p.Name, target)}
}

// rememberGoalKindsLocked records the day's activity and wellbeing goal
// kinds, so they aren't asked for again straight away. Caller must hold
// p.mu.
func (p *DigitalPet) rememberGoalKindsLocked() {
	goals := p.Goals.Goals
	if len(goals) < 3 {
		return
	}
	kind := goals[1].Kind
	if kind == GoalCare || kind == GoalLearn {
		kind = goalClassic
	}
	p.RecentGoalKinds = append(p.RecentGoalKinds, kind)
	if len(p.RecentGoalKinds) > recentGoalDays {
		p.RecentGoalKinds = p.RecentGoalKinds[len(p.RecentGoalKinds)-recentGoalDays:]
	}
	p.LastWellbeingGoal = goals[2].Kind
}

// upkeepDueLocked returns upkeep that has run out or will within 50 game
// days. Caller must hold p.mu.
func (p *DigitalPet) upkeepDueLocked() (string, string) {
	age := p.Biology.GetAgeInDays()
	_, hasBed := p.bestInGroupLocked("bed")
	for _, u := range []struct {
		id, name string
		owned    bool
	}{
		{fountainFilterUpkeep, "fountain filter", p.ownsLocked("water_fountain")},
		{beddingUpkeep, "bedding", hasBed},
		{fleaCollarUpkeep, "flea and tick collar", p.ownsLocked("flea_collar")},
		{vaccinationUpkeep, "vaccination", p.Upkeep[vaccinationUpkeep] > 0},
	} {
		if u.owned && p.Upkeep[u.id]-age < 50 {
			return u.id, u.name
		}
	}
	return "", ""
}

// staleChangeLocked returns a change at home not made for 5 game days.
// Caller must hold p.mu.
func (p *DigitalPet) staleChangeLocked() (environmentChange, bool) {
	age := p.Biology.GetAgeInDays()
	for _, c := range p.defLocked().EnvironmentChanges {
		if last, ok := p.EnrichmentUsed[c.ID]; !ok || age-last >= 5 {
			return c, true
		}
	}
	return environmentChange{}, false
}

// placeToExploreLocked returns the discovered place visited least. Caller
// must hold p.mu.
func (p *DigitalPet) placeToExploreLocked() (string, string) {
	best, name, fewest := "", "", -1
	for _, id := range environment.TripDestinations() {
		loc, ok := p.placeLocked(id)
		if !ok || !p.discoveredLocked(id, loc) {
			continue
		}
		if visits := p.PlaceVisits[id]; fewest < 0 || visits < fewest {
			best, name, fewest = id, placeDetails[id].phrase, visits
		}
	}
	return best, name
}

// rustyTrickLocked returns a trick that has got rusty. Caller must hold
// p.mu.
func (p *DigitalPet) rustyTrickLocked() (Trick, bool) {
	for _, t := range p.tricksLocked() {
		if p.TricksLearned[t.ID] && p.TrickSkills[t.ID] < 1.0 {
			return t, true
		}
	}
	return Trick{}, false
}

// goalEventLocked counts something the player did towards the day's and
// the week's goals: kind is a goal kind, detail what it was about. Caller
// must hold p.mu.
func (p *DigitalPet) goalEventLocked(kind, detail string) {
	p.ensureGoalsLocked()
	for i := range p.Goals.Goals {
		g := &p.Goals.Goals[i]
		if !g.Done && g.Kind == kind && (g.Detail == "" || g.Detail == detail) {
			g.Progress++
		}
	}
	p.weeklyEventLocked(kind)
	p.checkGoalsLocked()
}

// GoalEvent counts something the player did towards the pet's goals
func (p *DigitalPet) GoalEvent(kind, detail string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.goalEventLocked(kind, detail)
}

// updateTimedGoalsLocked counts play time towards goals kept up over
// time. Caller must hold p.mu.
func (p *DigitalPet) updateTimedGoalsLocked() {
	if p.Goals == nil {
		return
	}
	played := p.playNowLocked()
	elapsed := played.Sub(p.Goals.TimedAt)
	p.Goals.TimedAt = played
	if elapsed <= 0 || elapsed > timedGoalMaxStep {
		return
	}
	v := p.Biology.Vitals
	for i := range p.Goals.Goals {
		g := &p.Goals.Goals[i]
		if g.Done || g.Unit != "seconds" {
			continue
		}
		ok := false
		switch g.Kind {
		case GoalCalm:
			ok = v.Stress < 0.3
		case GoalComfort:
			ok = v.Stress < 0.4
		case GoalSteady:
			ok = v.Nutrition > 0.5 && v.Hydration > 0.5
		}
		if ok {
			g.Progress = min(g.Progress+int(elapsed.Seconds()+0.5), g.Target)
		}
	}
	p.checkGoalsLocked()
}

// weekID returns the ISO week of a day, such as 2026-W43
func weekID(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// ensureWeeklyLocked starts the week's goal when the week changes. Caller
// must hold p.mu.
func (p *DigitalPet) ensureWeeklyLocked() {
	t := now()
	week := weekID(t)
	if p.Weekly != nil && p.Weekly.Week >= week {
		return
	}
	var options []Goal
	options = append(options,
		Goal{Kind: WeeklyTeethDays, Icon: "🪥", Target: 5, Text: fmt.Sprintf("Brush %s's teeth on 5 days this week", p.Name)},
		Goal{Kind: WeeklyTrips, Icon: "🧭", Target: 7, Text: fmt.Sprintf("Take %s on 7 trips this week", p.Name)},
		Goal{Kind: WeeklyGoalDays, Icon: "📅", Target: 3, Text: "Finish all of a day's goals on 3 days this week"},
		Goal{Kind: WeeklyPlays, Icon: "🎾", Target: 20, Text: fmt.Sprintf("Play with %s 20 times this week", p.Name)},
	)
	if p.Biology.LifeStage() != biology.StageYoung {
		options = append(options, Goal{Kind: WeeklyRibbon, Icon: "🎀", Target: 1,
			Text: fmt.Sprintf("Win a ribbon with %s at a show this week", p.Name)})
	}
	if len(p.TricksLearned)+2 <= len(p.tricksLocked()) {
		options = append(options, Goal{Kind: WeeklyTricks, Icon: "🌟", Target: 2,
			Text: fmt.Sprintf("Teach %s 2 new tricks this week", p.Name)})
	}
	_, n := t.ISOWeek()
	g := options[n%len(options)]
	g.Reward, g.Stars = weeklyPoints, weeklyStars
	p.Weekly = &WeeklyGoal{Week: week, Goal: g}
}

// weeklyEventLocked counts a goal event towards the week's goal. Caller
// must hold p.mu.
func (p *DigitalPet) weeklyEventLocked(kind string) {
	p.ensureWeeklyLocked()
	w := p.Weekly
	if w.Goal.Done {
		return
	}
	switch {
	case kind == GoalTeeth && w.Goal.Kind == WeeklyTeethDays:
		if day := today(); w.LastDay != day {
			w.LastDay = day
			w.Goal.Progress++
		}
	case kind == GoalExplore && w.Goal.Kind == WeeklyTrips,
		kind == GoalLearn && w.Goal.Kind == WeeklyTricks,
		kind == WeeklyGoalDays && w.Goal.Kind == WeeklyGoalDays,
		kind == WeeklyRibbon && w.Goal.Kind == WeeklyRibbon,
		kind == types.InteractionPlaying.String() && w.Goal.Kind == WeeklyPlays:
		w.Goal.Progress++
	}
}

// checkWeeklyLocked pays the week's goal when it is done. Caller must hold
// p.mu.
func (p *DigitalPet) checkWeeklyLocked() {
	if p.Weekly == nil || p.Weekly.Goal.Done || p.Weekly.Goal.Progress < p.Weekly.Goal.Target {
		return
	}
	g := &p.Weekly.Goal
	g.Done = true
	p.FriendshipPoints += g.Reward
	p.PointsEarned += g.Reward
	p.addHistoryLocked("goal", g.Icon+" "+g.Text, fmt.Sprintf("+%d points", g.Reward))
	p.completedGoals = append(p.completedGoals, *g)
}

// WeekGoal returns the week's goal
func (p *DigitalPet) WeekGoal() WeeklyGoal {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ensureWeeklyLocked()
	return *p.Weekly
}
