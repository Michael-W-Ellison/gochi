package core

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/interaction"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Mini-games: short games played on the page. Finishing one counts as the
// matching care (playing, enrichment or grooming), with its points under
// the usual limit. There are no scores: the page reports how well the
// game went (0 to 1), which sets how much the pet got out of it, and the
// pet's enjoyment is described in words. A game opens once the pet has
// had its kind of care.

// Mini-game IDs
const (
	GameFetch     = "fetch"
	GameWand      = "wand"
	GamePuzzle    = "puzzle"
	GameScent     = "scent"
	GameGrooming  = "grooming"
	minGameEffect = 0.4 // A game that went badly still counts for this much
)

// ErrUnknownGame is returned for a mini-game that doesn't exist or isn't
// for the pet's species
var ErrUnknownGame = errors.New("unknown game")

// miniGameDef is a mini-game
type miniGameDef struct {
	id, name, icon, description, lesson string
	species                             Species // "" for both
	care                                types.InteractionType
}

// miniGames are every mini-game
var miniGames = []miniGameDef{
	{id: GameFetch, name: "Fetch", icon: "🎾", species: SpeciesDog, care: types.InteractionPlaying,
		description: "Hold to wind up the throw, and let go when the marker is in the green. Five throws.",
		lesson:      "Stop a game of fetch before your dog is worn out, and throw a ball or toy, never a stick: sticks splinter and injure mouths."},
	{id: GameWand, name: "Feather wand", icon: "🪶", species: SpeciesCat, care: types.InteractionPlaying,
		description: "Move the wand like prey: mix quick dashes with slow, creeping moves, then let your cat catch it.",
		lesson:      "Always let a cat catch the toy at the end, or it gets frustrated; never use your hands as toys, or it learns to pounce on them."},
	{id: GamePuzzle, name: "Puzzle feeder", icon: "🧩", care: types.InteractionEnvironmentalEnrichment,
		description: "Find the hidden kibble. The feeder changes from game to game and gets harder as your pet gets better at it.",
		lesson:      "Food puzzles slow down fast eaters and keep a pet's mind busy; start easy and make them harder over time."},
	{id: GameScent, name: "Scent trail", icon: "👃", care: types.InteractionEnvironmentalEnrichment,
		description: "Treats are hidden around the house. Follow your pet's nose: its tail and ears say warmer or colder.",
		lesson:      "Sniffing is tiring in a good way: ten minutes of scent work can tire a dog as much as a walk, and it calms anxious pets."},
	{id: GameGrooming, name: "Gentle grooming", icon: "🪮", care: types.InteractionGrooming,
		description: "Brush each part of the coat the way the fur lies, and go gently on the spots your pet is sensitive about.",
		lesson:      "Brush in the direction the fur lies, and slowly at sensitive spots; a pet that flinches needs a break, not more brushing."},
}

// GroomRegion is a part of the coat in the grooming game
type GroomRegion struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Sensitive bool   `json:"sensitive"` // The pet doesn't like it touched: go gently
}

// MiniGame is a mini-game as the player sees it
type MiniGame struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Icon        string        `json:"icon"`
	Description string        `json:"description"`
	Lesson      string        `json:"lesson"`
	Tiles       int           `json:"tiles,omitempty"`   // Puzzle feeder: how many tiles (as in Puzzle)
	Puzzle      *PuzzleSetup  `json:"puzzle,omitempty"`  // Puzzle feeder: how it is set up for the pet now
	Regions     []GroomRegion `json:"regions,omitempty"` // Gentle grooming: the parts of the coat
	Unavailable string        `json:"unavailable,omitempty"`
}

// MiniGameResult is how a finished game went for the pet
type MiniGameResult struct {
	InteractionResult
	Text      string `json:"text"`
	Enjoyment string `json:"enjoyment"` // loved, enjoyed, some_fun or bored
	Lesson    string `json:"lesson"`
}

// findMiniGameLocked returns a game for the pet's species. Caller must
// hold p.mu.
func (p *DigitalPet) findMiniGameLocked(id string) (miniGameDef, bool) {
	for _, g := range miniGames {
		if g.id == id && (g.species == "" || g.species == p.speciesLocked()) {
			return g, true
		}
	}
	return miniGameDef{}, false
}

// The puzzle feeder gets harder as the pet gets better at it, as real
// food puzzles should: each pet has a level, from 1 to maxPuzzleLevel,
// raised by a puzzle solved well and lowered by one it gave up on. The
// level sets how many tiles there are (more for clever pets), how much
// kibble is hidden, how many empty picks the pet puts up with, how many
// times the player can watch its nose for a hint and how much a hint
// says, and which feeders it can be: lifting tiles, a memory board (the
// kibble is shown, then covered), a sniff mat (an empty spot says how
// much kibble is next to it) and a cup shuffle (the kibble is shown under
// a cup, the cups are swapped, then the player picks one). The page picks
// one of the feeders each game.

// Puzzle feeders, from level 1, 2, 3 and 4 on
const (
	PuzzleTiles  = "tiles"
	PuzzleMemory = "memory"
	PuzzleSniff  = "sniff"
	PuzzleCups   = "cups"
)

// puzzleKinds are the feeders, in the order levels open them
var puzzleKinds = []string{PuzzleTiles, PuzzleMemory, PuzzleSniff, PuzzleCups}

const (
	// maxPuzzleLevel is the hardest puzzle level
	maxPuzzleLevel = 10

	// Quality at or above puzzleLevelUp raises the level; below
	// puzzleLevelDown lowers it
	puzzleLevelUp   = 0.8
	puzzleLevelDown = 0.34
)

// PuzzleSetup is how the puzzle feeder is set up for the pet now
type PuzzleSetup struct {
	Level       int      `json:"level"`        // 1 to maxPuzzleLevel
	Kinds       []string `json:"kinds"`        // The feeders it can be
	Tiles       int      `json:"tiles"`        // Spots on the board (tiles, memory and sniff mat)
	Kibble      int      `json:"kibble"`       // Kibble hidden on the board
	Misses      int      `json:"misses"`       // Empty picks the pet puts up with before giving up
	Sniffs      int      `json:"sniffs"`       // Hints from the pet's nose
	Exact       bool     `json:"exact"`        // A hint names the spot (or cup), not only the row (or side)
	ShowSeconds float64  `json:"show_seconds"` // Memory board: how long the kibble shows
	Cups        int      `json:"cups"`         // Cup shuffle: how many cups
	Swaps       int      `json:"swaps"`        // Cup shuffle: swaps each round
	SwapSeconds float64  `json:"swap_seconds"` // Cup shuffle: how long each swap takes
	Rounds      int      `json:"rounds"`       // Cup shuffle: rounds in a game
}

// puzzleLevelLocked returns the pet's puzzle level. Caller must hold p.mu.
func (p *DigitalPet) puzzleLevelLocked() int {
	return min(max(p.PuzzleLevel, 1), maxPuzzleLevel)
}

// puzzleSetupLocked sets up the puzzle feeder for the pet's level and
// intelligence. Caller must hold p.mu.
func (p *DigitalPet) puzzleSetupLocked() PuzzleSetup {
	level := p.puzzleLevelLocked()
	step := level - 1
	tiles := min(3+int(math.Round(3*p.Personality.Traits.Intelligence))+step, 12)
	return PuzzleSetup{
		Level:       level,
		Kinds:       slices.Clone(puzzleKinds[:min(level, len(puzzleKinds))]),
		Tiles:       tiles,
		Kibble:      max(2, (tiles+1)/2-step/3),
		Misses:      max(1, 3-step/3),
		Sniffs:      max(1, 3-step/2),
		Exact:       level <= 3,
		ShowSeconds: max(1, 3-0.25*float64(step)),
		Cups:        min(3+step/3, 5),
		Swaps:       3 + level,
		SwapSeconds: max(0.3, 0.8-0.05*float64(step)),
		Rounds:      3,
	}
}

// groomRegionsLocked returns the parts of the pet's coat, and which it is
// sensitive about. Caller must hold p.mu.
func (p *DigitalPet) groomRegionsLocked() []GroomRegion {
	var regions []GroomRegion
	for _, r := range p.defLocked().PetRegions {
		regions = append(regions, GroomRegion{ID: r.ID, Name: r.Name, Sensitive: p.regionLikingLocked(r) < 0})
	}
	return regions
}

// hadCareLocked reports whether the pet has had a game's kind of care: for
// enrichment, a trip or a change to its surroundings. Caller must hold
// p.mu.
func (p *DigitalPet) hadCareLocked(g miniGameDef) bool {
	n := p.CareCounts[g.care.String()]
	if g.care == types.InteractionEnvironmentalEnrichment {
		n += p.CareCounts[types.InteractionEnvironmentChange.String()]
	}
	return n > 0
}

// gameRefusalLocked says why the pet won't play a game now (asleep, too
// tired, away), or returns "", so the player knows before playing.
// Caller must hold p.mu.
func (p *DigitalPet) gameRefusalLocked(g miniGameDef) string {
	if !p.Biology.IsAlive || p.Hibernating || p.isExploringLocked() {
		return p.Name + " isn't here to play."
	}
	p.Biology.EnsurePhysiology()
	if refusal := p.careRefusalLocked(g.care); refusal != "" {
		return refusal
	}
	if ok, reason := interaction.ValidateInteraction(g.care, p.interactionContextLocked()); !ok {
		return strings.Replace(reason, "Pet", p.Name, 1) + "."
	}
	return ""
}

// MiniGames returns the games for the pet's species
func (p *DigitalPet) MiniGames() []MiniGame {
	p.mu.Lock()
	defer p.mu.Unlock()
	var list []MiniGame
	for _, g := range miniGames {
		if g.species != "" && g.species != p.speciesLocked() {
			continue
		}
		v := MiniGame{ID: g.id, Name: g.name, Icon: g.icon, Description: g.description, Lesson: g.lesson}
		switch g.id {
		case GamePuzzle:
			setup := p.puzzleSetupLocked()
			v.Puzzle, v.Tiles = &setup, setup.Tiles
		case GameGrooming:
			v.Regions = p.groomRegionsLocked()
		}
		if !p.hadCareLocked(g) {
			v.Unavailable = fmt.Sprintf("Opens once %s has had some %s.", p.Name, careNoun(g.care))
		} else {
			v.Unavailable = p.gameRefusalLocked(g)
		}
		list = append(list, v)
	}
	return list
}

// careNoun names a kind of care for the game list
func careNoun(care types.InteractionType) string {
	switch care {
	case types.InteractionPlaying:
		return "play"
	case types.InteractionGrooming:
		return "grooming"
	}
	return "enrichment"
}

// enjoymentWords describes how much the pet enjoyed a game
func enjoymentWords(quality float64) (string, string) {
	switch {
	case quality >= 0.8:
		return "loved", "loved"
	case quality >= 0.5:
		return "enjoyed", "enjoyed"
	case quality >= 0.2:
		return "some_fun", "had some fun"
	}
	return "bored", "lost interest in"
}

// adjustPuzzleLevelLocked raises the puzzle level after a puzzle solved
// well and lowers it after one the pet gave up on, and says so (or
// returns ""). Caller must hold p.mu.
func (p *DigitalPet) adjustPuzzleLevelLocked(quality float64) string {
	level := p.puzzleLevelLocked()
	switch {
	case quality >= puzzleLevelUp && level < maxPuzzleLevel:
		p.PuzzleLevel = level + 1
		text := fmt.Sprintf("The puzzle feeder gets harder for %s: level %d.", p.Name, level+1)
		if level < len(puzzleKinds) {
			text += " A new kind of feeder can turn up."
		}
		return text
	case quality < puzzleLevelDown && level > 1:
		p.PuzzleLevel = level - 1
		return fmt.Sprintf("The puzzle feeder gets a little easier for %s: level %d.", p.Name, level-1)
	}
	p.PuzzleLevel = level
	return ""
}

// FinishMiniGame records a finished game. quality is how well it went,
// from 0 to 1, as the page judged it.
func (p *DigitalPet) FinishMiniGame(id string, quality float64) (MiniGameResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	g, ok := p.findMiniGameLocked(id)
	if !ok {
		return MiniGameResult{}, ErrUnknownGame
	}
	if !p.hadCareLocked(g) {
		return MiniGameResult{}, RefusedError{fmt.Sprintf("This game opens once %s has had some %s.", p.Name, careNoun(g.care))}
	}
	if math.IsNaN(quality) {
		quality = 0
	}
	quality = min(max(quality, 0), 1)

	care := p.interactLocked(g.care, minGameEffect+(1-minGameEffect)*quality)
	result := MiniGameResult{InteractionResult: care, Lesson: g.lesson}
	if !care.Applied {
		result.Text = care.Refusal
		return result, nil
	}
	enjoyment, words := enjoymentWords(quality)
	result.Enjoyment = enjoyment
	result.Text = fmt.Sprintf("%s %s %s.", p.Name, words, map[string]string{
		GameFetch: "chasing the ball", GameWand: "hunting the feather", GamePuzzle: "working out the puzzle",
		GameScent: "following the trail", GameGrooming: "being brushed",
	}[g.id])
	if g.id == GamePuzzle {
		result.Note = p.adjustPuzzleLevelLocked(quality)
	}
	p.addHistoryLocked("Game", g.name, words)
	return result, nil
}
