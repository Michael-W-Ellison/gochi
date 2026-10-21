package core

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Trips go to a place from the environment package's locations. Each
// place has its own odds - wilder places hold more to find and more
// danger - and its own resources: what the pet finds is taken from the
// place, which grows back over time, so returning to the same place again
// and again finds less. New places open up as the pet takes more trips.

// Place errors
var (
	ErrUnknownPlace = errors.New("unknown place")
	ErrPlaceLocked  = errors.New("pet hasn't discovered this place yet")
)

// defaultPlace is where a trip goes when no place is chosen, and where
// trips from saves made before places existed went
const defaultPlace = "park"

// placeDetails are how each place is shown, and how many trips it takes
// to discover it
var placeDetails = map[string]struct {
	icon, phrase string
	tripsNeeded  int
}{
	"park":     {"🌳", "the park", 0},
	"beach":    {"🏖️", "the beach", 2},
	"farm":     {"🚜", "the farm", 4},
	"forest":   {"🌲", "the forest", 6},
	"city":     {"🏙️", "the city", 9},
	"mountain": {"⛰️", "the mountain", 12},
}

// How a place changes a trip. Danger adds dangerInjury percentage points
// of injury chance per unit above the park's; exploration value scales
// the chance of a find relative to the park's. A place with no resources
// left still gives depletedFinds of its usual chance of a find.
const (
	parkDanger      = 0.1
	parkExploration = 0.7
	dangerInjury    = 30.0
	depletedFinds   = 0.3
	maxFindChance   = 90
)

// How much of a place's resources a trip takes
const (
	treasureDepletion = 0.3 // Found a souvenir
	nothingDepletion  = 0.1 // Looked around and found nothing of value
)

// Place is somewhere the pet can explore
type Place struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Icon        string  `json:"icon"`
	Description string  `json:"description"`
	Discovered  bool    `json:"discovered"`
	TripsNeeded int     `json:"trips_needed"` // Total trips that discover it
	Resources   float64 `json:"resources"`    // How much is left to find (0.0 to 1.0)
	Danger      float64 `json:"danger"`       // 0.0 to 1.0
}

// ensureLocationsLocked creates the locations for environments saved
// without them. Caller must hold p.mu.
func (p *DigitalPet) ensureLocationsLocked() *environment.LocationManager {
	p.ensureEnvironmentLocked()
	if p.Environment.Locations == nil {
		p.Environment.Locations = environment.NewLocationManager()
	}
	return p.Environment.Locations
}

// placeLocked returns the location for a place. Caller must hold p.mu.
func (p *DigitalPet) placeLocked(id string) (*environment.Location, bool) {
	if _, ok := placeDetails[id]; !ok {
		return nil, false
	}
	loc, ok := p.ensureLocationsLocked().Locations[id]
	return loc, ok
}

// discoveredLocked reports whether the pet has discovered a place: it has
// taken enough trips, or the place was unlocked another way. Caller must
// hold p.mu.
func (p *DigitalPet) discoveredLocked(id string, loc *environment.Location) bool {
	return loc.Unlocked || p.TripsTaken >= placeDetails[id].tripsNeeded
}

// Places returns the places the pet can explore, nearest first
func (p *DigitalPet) Places() []Place {
	p.mu.Lock()
	defer p.mu.Unlock()

	places := []Place{}
	for _, id := range environment.TripDestinations() {
		loc, ok := p.placeLocked(id)
		if !ok {
			continue
		}
		places = append(places, Place{
			ID: id, Name: loc.Name, Icon: placeDetails[id].icon, Description: loc.Description,
			Discovered: p.discoveredLocked(id, loc), TripsNeeded: placeDetails[id].tripsNeeded,
			Resources: loc.ResourceLevel(), Danger: loc.Features.DangerLevel,
		})
	}
	return places
}

// tripPlaceLocked returns where the current or last trip went. Caller
// must hold p.mu.
func (p *DigitalPet) tripPlaceLocked() (string, *environment.Location) {
	id := p.ExploringTo
	loc, ok := p.placeLocked(id)
	if !ok {
		id = defaultPlace
		loc, _ = p.placeLocked(id)
	}
	return id, loc
}

// placeOddsLocked returns how the trip's place changes its odds: extra
// injury chance (percentage points) and a multiplier for the chance of a
// find. Caller must hold p.mu.
func (p *DigitalPet) placeOddsLocked() (extraInjury int, findFactor float64) {
	_, loc := p.tripPlaceLocked()
	if loc == nil {
		return 0, 1
	}
	extraInjury = int((loc.Features.DangerLevel - parkDanger) * dangerInjury)
	findFactor = loc.Features.ExplorationValue / parkExploration * (depletedFinds + (1-depletedFinds)*loc.ResourceLevel())
	return extraInjury, findFactor
}

// takeFromPlaceLocked records what a trip took from its place, counts the
// trip, and returns a note naming any place the pet discovered by it.
// Caller must hold p.mu.
func (p *DigitalPet) takeFromPlaceLocked(outcome string) string {
	if _, loc := p.tripPlaceLocked(); loc != nil {
		switch outcome {
		case OutcomeTreasure:
			loc.Deplete(treasureDepletion)
		case OutcomeNothing:
			loc.Deplete(nothingDepletion)
		}
	}

	p.TripsTaken++
	for _, id := range environment.TripDestinations() {
		if needed := placeDetails[id].tripsNeeded; needed > 0 && needed == p.TripsTaken {
			if loc, ok := p.placeLocked(id); ok {
				loc.Unlocked = true
				return fmt.Sprintf(" 🗺️ %s can now explore %s!", p.Name, placeDetails[id].phrase)
			}
		}
	}
	return ""
}

// lastDiscoveredLocked returns the place the pet's trips opened up most
// recently. Caller must hold p.mu.
func (p *DigitalPet) lastDiscoveredLocked() string {
	for _, id := range environment.TripDestinations() {
		if placeDetails[id].tripsNeeded == p.TripsTaken {
			return id
		}
	}
	return ""
}

// placePhrase names where the current or last trip went, e.g. "the park".
// Caller must hold p.mu.
func (p *DigitalPet) placePhraseLocked() string {
	id, _ := p.tripPlaceLocked()
	return placeDetails[id].phrase
}

// visitPlaceLocked applies what a trip's place offers besides finds: the
// company of other animals (its social bonus) and new things to take in
// (its exploration bonus) meet the pet's needs, and places with a health
// risk expose the pet to parasites. Exposure builds up with each visit
// (by the place's risk, more for a weak immune system) and the pet picks
// up parasites when it reaches 1. It returns a note for the trip's report.
// Caller must hold p.mu.
func (p *DigitalPet) visitPlaceLocked() string {
	_, loc := p.tripPlaceLocked()
	if loc == nil {
		return ""
	}
	effects := loc.Effects
	age := p.Biology.GetAgeInDays()
	p.ensureNeedsLocked()
	p.Needs.SatisfyNeed(types.NeedSocial, effects.SocialBonus, age)
	p.Needs.SatisfyNeed(types.NeedExploration, effects.ExplorationBonus, age)

	note := ""
	if effects.SocialBonus >= 0.5 && sociability(*p.Personality.Traits) >= 0.5 {
		v := p.Biology.Vitals
		v.Happiness += 0.05
		v.Clamp()
		note = fmt.Sprintf(" %s made some friends along the way.", p.Name)
	}
	resistance := max(1.2-p.Biology.ImmuneResponse(), 0.1)
	p.ParasiteExposure += effects.HealthRisk * resistance * p.effectLocked(effParasites)
	if p.Illness == nil && p.ParasiteExposure >= 1 {
		p.ParasiteExposure = 0
		p.Illness = &Illness{Kind: IllnessParasites, Severity: 0.1, SinceAge: age}
		p.recordHealthEventLocked("vet", "%s may have picked up parasites while exploring. Watch for signs of illness.", p.Name)
		note += fmt.Sprintf(" %s may have picked something up there.", p.Name)
	}
	return note
}

// KnownPlace reports whether a place is one the game knows: home or a
// place to explore
func KnownPlace(id string) bool {
	return id == "home" || slices.Contains(environment.TripDestinations(), id)
}
