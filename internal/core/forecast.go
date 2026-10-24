package core

import (
	"hash/fnv"
	"math"
	"math/rand/v2"
	"sort"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Predictive care: the game runs a copy of the pet forward in time to see
// what it will need, and when, so the owner can see it coming instead of
// waiting for the pet to ask.

// NeedForecast is something the pet will soon ask for
type NeedForecast struct {
	Action  string  `json:"action"` // The care that meets it, as in PetRequest
	Icon    string  `json:"icon"`
	Message string  `json:"message"` // What the pet will ask
	InDays  float64 `json:"in_days"` // Game days from now
}

// minForecastSteps is the fewest steps a forecast simulates
const minForecastSteps = 60

// maxForecastStep is the longest step a forecast simulates, in game days:
// a quarter of a day, so every day has a step in daylight (8 hours at
// least) and needs that wait for daylight, such as exploring, are seen
const maxForecastStep = 0.25

// maxForecasts is how many forecasts are returned
const maxForecasts = 3

// forecastStep returns the step a forecast over horizon game days takes:
// a quarter of a day, halved until the horizon has at least
// minForecastSteps steps
func forecastStep(horizon float64) float64 {
	step := maxForecastStep
	for horizon/step < minForecastSteps {
		step /= 2
	}
	return step
}

// forecastWeather returns the source of the forecast copy's weather
// chances for the pet with id: one generator for each weather check,
// seeded from the pet and the game hour of the check. Every forecast of
// the pet then sees the same weather at a given hour (the pet's own
// weather is left to chance), so needs that wait for good weather, such as
// exploring, keep their place from one forecast to the next.
func forecastWeather(id types.PetID) func(at float64) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(id)) // A hash's Write never fails
	seed := h.Sum64()
	return func(at float64) *rand.Rand {
		return rand.New(rand.NewPCG(seed, uint64(int64(math.Round(at)))))
	}
}

// Forecast returns up to 3 of what the pet will ask for within horizon game days if
// nothing changes, soonest first: the needs a copy of the pet comes to
// have as time passes, other than any it has now. Every need counts, not
// only the most urgent, so one starting while a more urgent one waits is
// still seen.
//
// The copy is looked at on a fixed grid of the pet's age (see
// forecastStep), and its weather follows forecastWeather, so forecasts
// made a moment apart look at the same moments of the same future and
// agree; a forecast is accurate to one step.
func (p *DigitalPet) Forecast(horizon float64) []NeedForecast {
	return p.forecast(horizon, maxForecasts)
}

// forecast returns up to limit of the needs Forecast would find (all of
// them within horizon, with no limit)
func (p *DigitalPet) forecast(horizon float64, limit int) []NeedForecast {
	forecasts := []NeedForecast{}
	if horizon <= 0 || !p.IsAlive() || p.IsHibernating() || p.IsExploring() {
		return forecasts
	}
	data, err := p.Save()
	if err != nil {
		return forecasts
	}
	sim, err := Load(data)
	if err != nil {
		return forecasts
	}

	sim.ensureEnvironmentLocked() // Only this goroutine has the copy
	sim.Environment.Weather.UseChances(forecastWeather(p.ID))

	seen := make(map[string]bool)
	for _, need := range p.currentNeeds() {
		seen[need.Action] = true
	}
	// The first grid point after now, then a step at a time (counted from
	// now, so a damaged age can't stop the loop ending)
	step := forecastStep(horizon)
	age := sim.Biology.GetAgeInDays()
	first := (math.Floor(age/step)+1)*step - age
	if !(first > 0 && first <= step) { // An age too large, or not a number
		first = step
	}
	for in, by := first, first; in <= horizon; in, by = in+step, step {
		sim.Update(by) // Whole steps after the first, so the copy's own clocks (such as the weather's checks) count them exactly
		if !sim.IsAlive() || sim.IsHibernating() {
			break
		}
		for _, r := range sim.currentNeeds() {
			if !seen[r.Action] && r.Action != "revive" {
				seen[r.Action] = true
				forecasts = append(forecasts, NeedForecast{Action: r.Action, Icon: r.Icon, Message: r.Message, InDays: in})
			}
		}
		if limit > 0 && len(forecasts) >= limit {
			break
		}
	}
	sort.SliceStable(forecasts, func(i, j int) bool { return forecasts[i].InDays < forecasts[j].InDays })
	if limit > 0 && len(forecasts) > limit { // Needs starting at the same step, the most urgent first
		forecasts = forecasts[:limit]
	}
	return forecasts
}

// shownNeed is a forecast UpcomingNeeds has shown, with the pet's age it is
// due at
type shownNeed struct {
	NeedForecast
	due float64
}

// careGiven is the care a pet has had: its care counts by kind and its
// trips
type careGiven struct {
	counts map[string]int
	trips  int
}

// meeting is the care that meets each need: the care the page's button for
// it gives (treats are among the foods)
var meeting = map[string][]types.InteractionType{
	"feed":  {types.InteractionFeeding, types.InteractionRewards},
	"drink": {types.InteractionDrinking},
	"pet":   {types.InteractionPetting},
	"play":  {types.InteractionPlaying},
	"groom": {types.InteractionGrooming},
	"medic": {types.InteractionMedicalCare},
	"clean": {types.InteractionCleaning},
	"walk":  {types.InteractionWalking},
	"train": {types.InteractionTraining},
}

// careGivenLocked returns the care the pet has had so far. Caller must hold
// p.mu.
func (p *DigitalPet) careGivenLocked() careGiven {
	counts := make(map[string]int, len(p.CareCounts))
	for kind, n := range p.CareCounts {
		counts[kind] = n
	}
	return careGiven{counts, p.TripsTaken}
}

// meets reports whether care given between before and c meets the need
// asked for with action (exploring is met by a trip)
func (c careGiven) meets(before careGiven, action string) bool {
	if action == "explore" {
		return c.trips > before.trips
	}
	for _, kind := range meeting[action] {
		if c.counts[kind.String()] > before.counts[kind.String()] {
			return true
		}
	}
	return false
}

// UpcomingNeeds returns what the pet will ask for within horizon game days,
// soonest first, for the owner to see coming: Forecast's needs, except that
// a need once shown stays until it is due, the pet has it, or the care that
// meets it has been given and it is no longer forecast. A need that is
// forecast again takes its new time; new needs fill the places left, up to
// 3, so one can't push a need already shown off the list. While the pet is
// away exploring nothing is shown, and what was shown is kept for its
// return.
func (p *DigitalPet) UpcomingNeeds(horizon float64) []NeedForecast {
	fresh := p.forecast(horizon, 0) // Every need, so one shown before is found however many come sooner

	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.Biology.IsAlive || p.Hibernating || horizon <= 0 {
		p.shownNeeds = nil
		return []NeedForecast{}
	}
	if p.isExploringLocked() {
		return []NeedForecast{}
	}
	age := p.Biology.GetAgeInDays()
	care := p.careGivenLocked()
	before := p.shownAfter
	p.shownAfter = care
	now := make(map[string]bool)
	for _, need := range p.requestsLocked(false) {
		now[need.Action] = true
	}
	forecast := make(map[string]NeedForecast, len(fresh))
	for _, f := range fresh {
		forecast[f.Action] = f
	}

	var shown []shownNeed
	listed := make(map[string]bool)
	for _, s := range p.shownNeeds {
		f, still := forecast[s.Action]
		switch {
		case s.due <= age, now[s.Action]: // Due, or the pet has it
		case still:
			shown = append(shown, shownNeed{f, age + f.InDays})
			listed[s.Action] = true
		case !care.meets(before, s.Action): // Not met: only its care meets a need
			shown = append(shown, s)
			listed[s.Action] = true
		}
	}
	for _, f := range fresh { // Soonest first
		if len(shown) >= maxForecasts {
			break
		}
		if !listed[f.Action] {
			shown = append(shown, shownNeed{f, age + f.InDays})
			listed[f.Action] = true
		}
	}
	sort.SliceStable(shown, func(i, j int) bool { return shown[i].due < shown[j].due })
	p.shownNeeds = shown

	needs := make([]NeedForecast, len(shown))
	for i, s := range shown {
		needs[i] = s.NeedForecast
		needs[i].InDays = s.due - age
	}
	return needs
}
