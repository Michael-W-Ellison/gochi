package core

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// A pet's record: a history of the care it has had and what happened to
// it, the achievements it has earned, what it has unlocked, and its
// lifetime statistics. They are kept with the pet and copied into its
// save file (see the data package's SaveData).

// HistoryEntry is one thing in the pet's history
type HistoryEntry struct {
	Time        time.Time `json:"time"`
	Age         float64   `json:"age"`         // The pet's age (game days)
	Action      string    `json:"action"`      // The kind of care, or "event"
	Description string    `json:"description"` // What happened
	Outcome     string    `json:"outcome"`     // How the pet took it, or what kind of event
}

// maxHistory is how many history entries a pet keeps
const maxHistory = 500

// addHistoryLocked adds an entry to the pet's history. Caller must hold
// p.mu.
func (p *DigitalPet) addHistoryLocked(action, description, outcome string) {
	p.History = append(p.History, HistoryEntry{Time: now(), Age: p.Biology.GetAgeInDays(), Action: action,
		Description: description, Outcome: outcome})
	if over := len(p.History) - maxHistory; over > 0 {
		p.History = append(p.History[:0:0], p.History[over:]...)
	}
}

// RecordEvent adds something that happened to the pet on its own (see
// ConditionEvents) to its history
func (p *DigitalPet) RecordEvent(event GameEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.addHistoryLocked("event", event.Message, event.Type.String())
}

// RecentHistory returns up to limit of the pet's most recent history
// entries, newest first
func (p *DigitalPet) RecentHistory(limit int) []HistoryEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	entries := []HistoryEntry{}
	for i := len(p.History) - 1; i >= 0 && len(entries) < limit; i-- {
		entries = append(entries, p.History[i])
	}
	return entries
}

// careHistoryText describes care for the history
func careHistoryText(interactionType types.InteractionType, result InteractionResult) (string, string) {
	description := interactionType.String()
	if result.ReactionText != "" {
		description += ": " + result.ReactionText
	}
	return description, string(result.Reaction)
}

// Achievement is a milestone a pet can reach
type Achievement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Description string    `json:"description"`
	Rarity      string    `json:"rarity"` // common, rare, epic or legendary
	Points      int       `json:"points"` // How much it counts towards an achievement score
	EarnedAt    time.Time `json:"earned_at,omitempty"`
}

// achievements are the milestones and how to tell a pet has reached them.
// Conditions run with p.mu held.
var achievements = []struct {
	Achievement
	earned func(p *DigitalPet) bool
}{
	{Achievement{ID: "first_trick", Name: "Good Pet", Icon: "🐾", Rarity: "common", Points: 10,
		Description: "Learned a first trick"},
		func(p *DigitalPet) bool { return len(p.TricksLearned) > 0 }},
	{Achievement{ID: "all_tricks", Name: "Trick Master", Icon: "🎓", Rarity: "epic", Points: 50,
		Description: "Learned every trick"},
		func(p *DigitalPet) bool { return len(p.TricksLearned) >= len(tricks) }},
	{Achievement{ID: "first_treasure", Name: "Treasure Hunter", Icon: "🪨", Rarity: "common", Points: 10,
		Description: "Brought home a first souvenir"},
		func(p *DigitalPet) bool { return len(p.Souvenirs) > 0 }},
	{Achievement{ID: "crystal", Name: "Shiny!", Icon: "💎", Rarity: "rare", Points: 25,
		Description: "Found a sparkly crystal"},
		func(p *DigitalPet) bool { return slices.Contains(p.Souvenirs, "crystal") }},
	{Achievement{ID: "explorer", Name: "Explorer", Icon: "🗺️", Rarity: "epic", Points: 50,
		Description: "Discovered every place to explore"},
		func(p *DigitalPet) bool { return p.TripsTaken >= placeDetails["mountain"].tripsNeeded }},
	{Achievement{ID: "grown_up", Name: "All Grown Up", Icon: "🎂", Rarity: "common", Points: 10,
		Description: "Grew up into an adult"},
		func(p *DigitalPet) bool { return p.Biology.LifeStage() != biology.StageYoung }},
	{Achievement{ID: "senior", Name: "Golden Years", Icon: "👑", Rarity: "rare", Points: 25,
		Description: "Became a senior"},
		func(p *DigitalPet) bool { return p.Biology.LifeStage() == biology.StageSenior }},
	{Achievement{ID: "long_life", Name: "Long Life", Icon: "🧪", Rarity: "legendary", Points: 100,
		Description: "Lived past a typical lifespan"},
		func(p *DigitalPet) bool { return p.Biology.GetAgeInDays() >= biology.BaseLifespan }},
	{Achievement{ID: "best_friends", Name: "Best Friends", Icon: "🤝", Rarity: "rare", Points: 25,
		Description: "Became best friends with another pet"},
		func(p *DigitalPet) bool {
			for _, rel := range p.Relationships.Relationships {
				if rel.Type == types.RelationshipFriend && relationScore(rel) >= 0.7 {
					return true
				}
			}
			return false
		}},
	{Achievement{ID: "family", Name: "Family", Icon: "🍼", Rarity: "rare", Points: 25,
		Description: "Had a baby"},
		func(p *DigitalPet) bool {
			for _, rel := range p.Relationships.Relationships {
				if rel.Type == types.RelationshipOffspring {
					return true
				}
			}
			return false
		}},
	{Achievement{ID: "devoted", Name: "Devoted", Icon: "💖", Rarity: "epic", Points: 50,
		Description: "Earned 500 Friendship Points"},
		func(p *DigitalPet) bool { return p.PointsEarned >= 500 }},
	{Achievement{ID: "well_equipped", Name: "Well Equipped", Icon: "🛍️", Rarity: "rare", Points: 25,
		Description: "Owns five items from the shop"},
		func(p *DigitalPet) bool { return len(p.Inventory) >= 5 }},
}

// CheckAchievements records the achievements the pet has newly reached and
// returns them
func (p *DigitalPet) CheckAchievements() []Achievement {
	p.mu.Lock()
	defer p.mu.Unlock()

	var earned []Achievement
	for _, a := range achievements {
		if _, ok := p.Achievements[a.ID]; ok || !a.earned(p) {
			continue
		}
		if p.Achievements == nil {
			p.Achievements = make(map[string]time.Time)
		}
		p.Achievements[a.ID] = now()
		achievement := a.Achievement
		achievement.EarnedAt = p.Achievements[a.ID]
		earned = append(earned, achievement)
		p.addHistoryLocked("achievement", fmt.Sprintf("%s %s: %s", achievement.Icon, achievement.Name,
			achievement.Description), achievement.Rarity)
	}
	return earned
}

// EarnedAchievements returns the achievements the pet has earned, in the
// order they were earned
func (p *DigitalPet) EarnedAchievements() []Achievement {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.earnedAchievementsLocked()
}

// earnedAchievementsLocked lists earned achievements. Caller must hold
// p.mu.
func (p *DigitalPet) earnedAchievementsLocked() []Achievement {
	earned := []Achievement{}
	for _, a := range achievements {
		if at, ok := p.Achievements[a.ID]; ok {
			achievement := a.Achievement
			achievement.EarnedAt = at
			earned = append(earned, achievement)
		}
	}
	sort.SliceStable(earned, func(i, j int) bool { return earned[i].EarnedAt.Before(earned[j].EarnedAt) })
	return earned
}

// AchievementCount returns how many achievements there are
func AchievementCount() int {
	return len(achievements)
}

// Unlocks are what the pet has unlocked
type Unlocks struct {
	Places   []string // Places it has discovered
	Items    []string // Shop items it owns
	Foods    []string // Foods and treats in its pantry
	Toys     []string // Toys in its pantry
	Tricks   []string // Tricks it has learned
	Variants []string // Its species
}

// Stats are the pet's lifetime statistics
type Stats struct {
	PlayTime     float64 // Hours
	Interactions int
	Age          float64 // Game days
	Happiness    float64 // Now
	Achievements int
	PointsEarned int
	Alive        bool
}

// SaveRecord is what a save file records besides the pet itself
type SaveRecord struct {
	History      []HistoryEntry
	Achievements []Achievement
	Unlocks      Unlocks
	Stats        Stats
}

// SaveRecord returns the pet's record for its save file
func (p *DigitalPet) SaveRecord() SaveRecord {
	p.mu.Lock()
	defer p.mu.Unlock()

	record := SaveRecord{
		History:      append([]HistoryEntry(nil), p.History...),
		Achievements: p.earnedAchievementsLocked(),
		Unlocks:      Unlocks{Places: []string{}, Items: []string{}, Foods: []string{}, Toys: []string{}, Tricks: []string{}},
		Stats: Stats{PlayTime: p.TotalPlayTime, Interactions: p.TotalInteractions, Age: p.Biology.GetAgeInDays(),
			Happiness: p.Biology.Vitals.Happiness, Achievements: len(p.Achievements), PointsEarned: p.PointsEarned,
			Alive: p.Biology.IsAlive},
	}
	for _, id := range environment.TripDestinations() {
		if loc, ok := p.placeLocked(id); ok && p.discoveredLocked(id, loc) {
			record.Unlocks.Places = append(record.Unlocks.Places, id)
		}
	}
	record.Unlocks.Items = append(record.Unlocks.Items, p.Inventory...)
	p.ensureSuppliesLocked()
	for id := range p.Supplies.Items {
		if s, ok := findSupply(id); ok {
			switch s.kind {
			case SupplyFood, SupplyTreat:
				record.Unlocks.Foods = append(record.Unlocks.Foods, id)
			case SupplyToy:
				record.Unlocks.Toys = append(record.Unlocks.Toys, id)
			}
		}
	}
	sort.Strings(record.Unlocks.Foods)
	sort.Strings(record.Unlocks.Toys)
	for _, t := range tricks {
		if p.TricksLearned[t.ID] {
			record.Unlocks.Tricks = append(record.Unlocks.Tricks, t.ID)
		}
	}
	record.Unlocks.Variants = []string{string(p.speciesLocked())}
	return record
}
