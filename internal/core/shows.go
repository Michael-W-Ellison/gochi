package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Pet shows. A pet can enter local shows against five other entrants made
// up by the game. Shows judge care, training and wellbeing, never looks
// against a breed standard, so a mixed breed can win anything. Each show
// has three levels; winning a level opens the next. Prizes are rosettes,
// Friendship Points (no more than showPointsPerDay a calendar day) and
// Home Stars (paid by the game server). A show tires and stresses a pet a
// little, like a trip, and it must rest a game day before the next.

// Show IDs
const (
	ShowTricks   = "tricks"
	ShowGrooming = "grooming"
	ShowAgility  = "agility"  // Dogs
	ShowPlaytime = "playtime" // Cats
	ShowHappiest = "happiest"
)

// Show levels
const (
	LevelNovice   = 1
	LevelOpen     = 2
	LevelChampion = 3
)

// Show limits
const (
	showPointsPerDay = 300 // Friendship Points from shows a calendar day
	showRest         = 1.0 // Game days between shows
	showEnergyCost   = 0.15
	showStress       = 0.05
	routineLength    = 3 // Tricks in a trick show routine
	showEntrants     = 5
)

// ShowDef is a kind of show
type ShowDef struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Judged      string    `json:"judged"` // What it is judged on
	Fee         int       `json:"fee"`    // Friendship Points to enter (0: free once a day)
	Lesson      string    `json:"lesson"`
	species     []Species // Who can enter; nil for both
	needsTricks []string  // Tricks that count towards the score
}

// showDefs are the shows
var showDefs = []ShowDef{
	{ID: ShowTricks, Name: "Trick Show", Icon: "🎪", Fee: 20,
		Judged: "A routine of three tricks you choose; ending on the hardest earns a bonus",
		Lesson: "Shows should be fun for your pet. Keep routines short, reward every trick, and stop if it seems stressed (ears back, tail tucked)."},
	{ID: ShowGrooming, Name: "Grooming Show", Icon: "🪮", Fee: 20,
		Judged: "A clean, recently brushed coat, clear eyes and clean teeth",
		Lesson: "Good grooming is health care: brushing spreads skin oils and finds lumps, ticks and mats early."},
	{ID: ShowAgility, Name: "Agility Course", Icon: "🏁", Fee: 30, species: dogs, needsTricks: []string{"weave", "hoop", "come"},
		Judged: "Fitness, energy, and the tricks weave, jump through a hoop and come",
		Lesson: "Agility is great exercise and teamwork. Warm up first, and keep jumps low for young dogs and dogs with joint problems."},
	{ID: ShowPlaytime, Name: "Playtime Challenge", Icon: "🧶", Fee: 30, species: cats, needsTricks: []string{"fetch", "touch"},
		Judged: "A busy, curious mind, playfulness, and the tricks fetch and touch",
		Lesson: "Cats need play every day: short bursts of hunting games, ending with a 'catch', keep them fit and content."},
	{ID: ShowHappiest, Name: "Happiest Pet", Icon: "😊", Fee: 0,
		Judged: "Happiness, calm and how well it has been cared for",
		Lesson: "A happy pet is the best prize: routine, play, company and good care add up."},
}

// levelNames name the show levels
var levelNames = map[int]string{LevelNovice: "Novice", LevelOpen: "Open", LevelChampion: "Champion"}

// entrantRange is the range of the made-up entrants' scores at each level
var entrantRange = map[int][2]int{LevelNovice: {40, 85}, LevelOpen: {50, 90}, LevelChampion: {65, 95}}

// showPrizes are the prizes for 1st, 2nd and 3rd place at each level:
// Friendship Points and Home Stars
var showPrizes = map[int][3][2]int{
	LevelNovice:   {{60, 5}, {30, 2}, {15, 0}},
	LevelOpen:     {{120, 10}, {60, 5}, {30, 2}},
	LevelChampion: {{200, 20}, {100, 10}, {50, 5}},
}

// rosettes name the ribbons for 1st, 2nd and 3rd place
var rosettes = [3]string{"Blue rosette", "Red rosette", "Yellow rosette"}

// entrantNames are the made-up entrants' names
var entrantNames = []string{"Pepper", "Biscuit", "Luna", "Milo", "Daisy", "Oliver", "Bella", "Charlie", "Coco",
	"Max", "Willow", "Teddy", "Poppy", "Ziggy", "Hazel", "Leo", "Nala", "Ollie", "Rosie", "Simba", "Maple", "Juniper",
	"Pickles", "Waffles", "Noodle", "Clover", "Mochi", "Kiwi", "Pumpkin", "Sunny", "Bean", "Ginger", "Smokey",
	"Shadow", "Peanut", "Bramble", "Toffee", "Olive", "Pip", "Socks"}

// Show errors
var (
	ErrUnknownShow  = errors.New("unknown show")
	ErrShowLevel    = errors.New("that level isn't open yet")
	ErrShowRoutine  = errors.New("a trick routine needs three different tricks the pet knows well")
	ErrShowRest     = errors.New("needs to rest after its last show")
	ErrShowToday    = errors.New("already entered today")
	ErrShowSpecies  = errors.New("not for this kind of pet")
	ErrShowUnsuited = errors.New("can't enter shows now")
)

// Ribbon is a prize a pet won at a show
type Ribbon struct {
	Show  string `json:"show"`
	Level int    `json:"level"`
	Place int    `json:"place"` // 1 to 3
	Day   string `json:"day"`
}

// Entrant is one of a show's entrants
type Entrant struct {
	Name  string `json:"name"`
	Breed string `json:"breed"` // Breed name
	Score int    `json:"score"`
	You   bool   `json:"you,omitempty"`
}

// ShowResult is how a show went
type ShowResult struct {
	Show     ShowDef   `json:"show"`
	Level    int       `json:"level"`
	Score    int       `json:"score"`
	Place    int       `json:"place"` // 1 is first
	Entrants []Entrant `json:"entrants"`
	Points   int       `json:"points"`
	Stars    int       `json:"stars"` // Home Stars, paid by the game server
	Ribbon   *Ribbon   `json:"ribbon,omitempty"`
	Title    string    `json:"title,omitempty"` // A title won now, e.g. "Grand Champion"
	Text     string    `json:"text"`
	Lesson   string    `json:"lesson"`
}

// ShowOption is a show as the page offers it
type ShowOption struct {
	ShowDef
	Levels      []int  `json:"levels"`                // Levels open
	Unavailable string `json:"unavailable,omitempty"` // Why it can't be entered now
}

// FindShow returns a show
func FindShow(id string) (ShowDef, bool) {
	for _, s := range showDefs {
		if s.ID == id {
			return s, true
		}
	}
	return ShowDef{}, false
}

// showsForLocked returns the shows the pet's species can enter. Caller
// must hold p.mu.
func (p *DigitalPet) showsForLocked() []ShowDef {
	var list []ShowDef
	for _, s := range showDefs {
		if s.species == nil || slices.Contains(s.species, p.speciesLocked()) {
			list = append(list, s)
		}
	}
	return list
}

// openLevelsLocked returns the levels of a show the pet may enter: Novice,
// and each level above one it has won. Caller must hold p.mu.
func (p *DigitalPet) openLevelsLocked(show string) []int {
	levels := []int{LevelNovice}
	for level := LevelNovice; level < LevelChampion; level++ {
		if !slices.ContainsFunc(p.Ribbons, func(r Ribbon) bool { return r.Show == show && r.Level == level && r.Place == 1 }) {
			break
		}
		levels = append(levels, level+1)
	}
	return levels
}

// showProblemLocked says why the pet can't enter a show now, or returns
// "". Caller must hold p.mu.
func (p *DigitalPet) showProblemLocked(s ShowDef) string {
	b := p.Biology
	switch {
	case !b.IsAlive || p.Hibernating || p.isExploringLocked():
		return p.Name + " isn't here to enter."
	case b.LifeStage() == biology.StageYoung:
		return p.Name + " is too young for shows; let it grow up first."
	case b.LifeStage() == biology.StageSenior && b.OldAgeDeath && b.CalculateLifespan() < frailDays:
		return p.Name + " is too frail for shows now; let it take life easy."
	case p.Illness != nil || p.Injured:
		return p.Name + " isn't well enough; get it treated first."
	case p.CurrentBehavior == types.BehaviorSleeping:
		return p.Name + " is asleep."
	case b.Vitals.Energy < 0.4:
		return p.Name + " is too tired for a show."
	case p.LastShowAge > 0 && b.GetAgeInDays()-p.LastShowAge < showRest:
		return p.Name + " needs to rest after its last show."
	case s.Fee == 0 && p.Marks["show_free_day"] == today():
		return p.Name + " already entered this today."
	case s.Fee > p.FriendshipPoints:
		return fmt.Sprintf("Entry costs %d Friendship Points.", s.Fee)
	}
	return ""
}

// Shows returns the shows the pet can enter, with the levels open
func (p *DigitalPet) Shows() []ShowOption {
	p.mu.Lock()
	defer p.mu.Unlock()
	var list []ShowOption
	for _, s := range p.showsForLocked() {
		list = append(list, ShowOption{ShowDef: s, Levels: p.openLevelsLocked(s.ID), Unavailable: p.showProblemLocked(s)})
	}
	return list
}

// showFocusLocked is how well the pet can concentrate: energy, calm and
// enough sleep. Caller must hold p.mu.
func (p *DigitalPet) showFocusLocked() float64 {
	v := p.Biology.Vitals
	debt := 0.0
	if p.Biology.Circadian != nil {
		debt = p.Biology.SleepDebtLevel()
	}
	return min(max(0.5*v.Energy+0.4*(1-v.Stress)+0.1*(1-debt), 0.3), 1)
}

// variation returns a random factor from 0.9 to 1.1
func variation() float64 {
	return 0.9 + float64(randIntn(201))/1000
}

// trickSkillLocked is how well the pet knows a trick, 0.0 to 1.0. Caller
// must hold p.mu.
func (p *DigitalPet) trickSkillLocked(id string) float64 {
	return min(p.TrickSkills[id], 1)
}

// showScoreLocked works out the pet's score, 0 to 100. Caller must hold
// p.mu.
func (p *DigitalPet) showScoreLocked(s ShowDef, routine []string) (int, error) {
	v, ph := p.Biology.Vitals, p.Biology.Physiology
	p.Biology.EnsurePhysiology()
	focus := p.showFocusLocked()
	tricksScore := func(ids []string) float64 {
		total := 0.0
		for _, id := range ids {
			total += p.trickSkillLocked(id)
		}
		return total / float64(len(ids))
	}
	score := 0.0
	switch s.ID {
	case ShowTricks:
		if len(routine) != routineLength {
			return 0, ErrShowRoutine
		}
		hardest, seen := 0, map[string]bool{}
		for i, id := range routine {
			t, ok := findTrick(id)
			t, forPet := forSpeciesTrick(t, p.speciesLocked())
			if !ok || !forPet || t.Special || p.trickSkillLocked(id) < 1 || seen[id] {
				return 0, ErrShowRoutine
			}
			seen[id] = true
			score += float64(20+4*t.Difficulty) * focus * variation()
			if i < routineLength-1 {
				hardest = max(hardest, t.Difficulty)
			} else if t.Difficulty > hardest {
				score += 5 * 120.0 / 100
			}
		}
		score = score * 100 / 120
	case ShowGrooming:
		brushed := min(p.recentCareLocked(types.InteractionGrooming), 1)
		score = (40*v.Cleanliness + 20*ph.EyeHealth + 20*ph.DentalHealth + 20*brushed) * (0.85 + 0.15*focus) * variation()
	case ShowAgility:
		score = (40*ph.CardiovascularHealth + 20*v.Energy + 40*tricksScore(s.needsTricks)) * focus * variation()
	case ShowPlaytime:
		p.ensureNeedsLocked()
		stim := p.Needs.GetNeedLevel(types.NeedMentalStimulation)
		score = (40*stim + 20*p.Personality.Traits.Playfulness + 40*tricksScore(s.needsTricks)) * focus * variation()
	case ShowHappiest:
		care := p.Biology.CareQuality
		if care <= 0 {
			care = 0.5
		}
		score = (50*v.Happiness + 30*(1-v.Stress) + 20*care) * variation()
	}
	return int(min(max(score, 0), 100) + 0.5), nil
}

// makeEntrants makes up a show's other entrants of a species
func makeEntrants(species Species, level int) []Entrant {
	r := entrantRange[level]
	breeds := Breeds(species)
	names := slices.Clone(entrantNames)
	entrants := make([]Entrant, showEntrants)
	for i := range entrants {
		n := randIntn(len(names))
		entrants[i] = Entrant{Name: names[n], Breed: breeds[randIntn(len(breeds))].Name, Score: r[0] + randIntn(r[1]-r[0]+1)}
		names = slices.Delete(names, n, n+1)
	}
	return entrants
}

// EnterShow enters the pet in a show at a level; routine is the tricks for
// a trick show, in order
func (p *DigitalPet) EnterShow(showID string, level int, routine []string) (ShowResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	s, ok := FindShow(showID)
	if !ok {
		return ShowResult{}, ErrUnknownShow
	}
	if s.species != nil && !slices.Contains(s.species, p.speciesLocked()) {
		return ShowResult{}, ErrShowSpecies
	}
	if !slices.Contains(p.openLevelsLocked(s.ID), level) {
		return ShowResult{}, ErrShowLevel
	}
	if problem := p.showProblemLocked(s); problem != "" {
		return ShowResult{}, RefusedError{problem}
	}
	score, err := p.showScoreLocked(s, routine)
	if err != nil {
		return ShowResult{}, err
	}

	// Entry, and the show itself
	p.FriendshipPoints -= s.Fee
	if s.Fee == 0 {
		if p.Marks == nil {
			p.Marks = make(map[string]string)
		}
		p.Marks["show_free_day"] = today()
	}
	v := p.Biology.Vitals
	v.Energy -= showEnergyCost
	v.Stress += showStress * (1 + p.Personality.Traits.Neuroticism)
	p.LastShowAge = p.Biology.GetAgeInDays()

	result := ShowResult{Show: s, Level: level, Score: score, Lesson: s.Lesson}
	result.Entrants = makeEntrants(p.speciesLocked(), level)
	result.Place = 1
	for _, e := range result.Entrants {
		if e.Score > score {
			result.Place++
		}
	}
	result.Entrants = append(result.Entrants, Entrant{Name: p.Name, Breed: p.breedDefLocked().Name, Score: score, You: true})
	slices.SortStableFunc(result.Entrants, func(a, b Entrant) int { return b.Score - a.Score })

	where := fmt.Sprintf("the %s %s", levelNames[level], s.Name)
	if result.Place <= 3 {
		prize := showPrizes[level][result.Place-1]
		result.Points = p.showPointsLocked(prize[0])
		result.Stars = prize[1]
		p.FriendshipPoints += result.Points
		p.PointsEarned += result.Points
		ribbon := Ribbon{Show: s.ID, Level: level, Place: result.Place, Day: today()}
		p.Ribbons = append(p.Ribbons, ribbon)
		p.keepLocked(KeepsakeRibbon, fmt.Sprintf("%s/%d/%d", s.ID, level, result.Place))
		result.Ribbon = &ribbon
		v.Happiness += 0.05
		result.Text = fmt.Sprintf("%s placed %s at %s with %d points and won a %s! 🏅", p.Name,
			[]string{"1st", "2nd", "3rd"}[result.Place-1], where, score, strings.ToLower(rosettes[result.Place-1]))
		if result.Place == 1 && level == LevelChampion && !p.hasTitleLocked(TitleGrandChampion) && p.championWinsLocked() >= 3 {
			p.Titles = append(p.Titles, TitleGrandChampion)
			result.Title = "Grand Champion"
			result.Text += fmt.Sprintf(" %s is now a Grand Champion! 🏆", p.Name)
		}
	} else {
		result.Text = fmt.Sprintf("%s came %d of %d at %s with %d points. Practice and good care will help next time.",
			p.Name, result.Place, showEntrants+1, where, score)
	}
	v.Clamp()
	p.addHistoryLocked("show", s.Icon+" "+where, fmt.Sprintf("place %d", result.Place))
	p.goalEventLocked(GoalShow, s.ID)
	if result.Ribbon != nil {
		p.weeklyEventLocked(WeeklyRibbon)
	}
	return result, nil
}

// showPointsLocked limits Friendship Points from shows to
// showPointsPerDay a calendar day. Caller must hold p.mu.
func (p *DigitalPet) showPointsLocked(points int) int {
	if p.Marks == nil {
		p.Marks = make(map[string]string)
	}
	if p.Marks["show_points_day"] != today() {
		p.Marks["show_points_day"] = today()
		p.countLocked("show_points_today", -p.Counters["show_points_today"])
	}
	points = min(points, showPointsPerDay-int(p.Counters["show_points_today"]))
	points = max(points, 0)
	p.countLocked("show_points_today", float64(points))
	return points
}

// ShowRecord returns the ribbons and titles the pet has won
func (p *DigitalPet) ShowRecord() ([]Ribbon, []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Ribbon{}, p.Ribbons...), append([]string{}, p.Titles...)
}

// Titles a pet can earn
const TitleGrandChampion = "grand_champion"

// hasTitleLocked reports whether the pet has a title. Caller must hold
// p.mu.
func (p *DigitalPet) hasTitleLocked(title string) bool {
	return slices.Contains(p.Titles, title)
}

// championWinsLocked counts the different shows the pet has won at
// Champion level. Caller must hold p.mu.
// ChampionWins returns how many different shows the pet has won at
// Champion level
func (p *DigitalPet) ChampionWins() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.championWinsLocked()
}

func (p *DigitalPet) championWinsLocked() int {
	shows := map[string]bool{}
	for _, r := range p.Ribbons {
		if r.Level == LevelChampion && r.Place == 1 {
			shows[r.Show] = true
		}
	}
	return len(shows)
}

// showAchievements are achievements for shows. Conditions run with p.mu
// held.
var showAchievements = []struct {
	Achievement
	earned func(p *DigitalPet) bool
}{
	{Achievement{ID: "first_ribbon", Name: "First Ribbon", Icon: "🎀", Rarity: "common", Points: 10,
		Description: "Won a ribbon at a show"},
		func(p *DigitalPet) bool { return len(p.Ribbons) > 0 }},
	{Achievement{ID: "blue_rosette", Name: "Blue Rosette", Icon: "🥇", Rarity: "common", Points: 10,
		Description: "Won first place at a show"},
		func(p *DigitalPet) bool {
			return slices.ContainsFunc(p.Ribbons, func(r Ribbon) bool { return r.Place == 1 })
		}},
	{Achievement{ID: "champion", Name: "Champion", Icon: "🏆", Rarity: "rare", Points: 25,
		Description: "Won first place at Champion level"},
		func(p *DigitalPet) bool { return p.championWinsLocked() > 0 }},
	{Achievement{ID: "grand_champion", Name: "Grand Champion", Icon: "👑", Rarity: "epic", Points: 50,
		Description: "Won three different shows at Champion level"},
		func(p *DigitalPet) bool { return p.hasTitleLocked(TitleGrandChampion) }},
	{Achievement{ID: "best_in_grooming", Name: "Best in Grooming", Icon: "✨", Rarity: "rare", Points: 25,
		Description: "Won the Champion Grooming Show with a long coat"},
		func(p *DigitalPet) bool {
			long := p.Look != nil && slices.Contains([]string{CoatLong, CoatShaggy, CoatSemi}, p.Look.Coat)
			return long && slices.ContainsFunc(p.Ribbons, func(r Ribbon) bool {
				return r.Show == ShowGrooming && r.Level == LevelChampion && r.Place == 1
			})
		}},
	{Achievement{ID: "underdog", Name: "Underdog", Icon: "🐾", Rarity: "rare", Points: 25,
		Description: "A mixed breed won first place at Champion level"},
		func(p *DigitalPet) bool { return p.breedDefLocked().Mixed && p.championWinsLocked() > 0 }},
}

func init() {
	achievements = append(achievements, showAchievements...)
}
