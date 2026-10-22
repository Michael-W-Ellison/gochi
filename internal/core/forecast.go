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

// Forecast returns what the pet will ask for within horizon game days if
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
		if len(forecasts) >= maxForecasts {
			break
		}
	}
	sort.SliceStable(forecasts, func(i, j int) bool { return forecasts[i].InDays < forecasts[j].InDays })
	if len(forecasts) > maxForecasts { // Needs starting at the same step, the most urgent first
		forecasts = forecasts[:maxForecasts]
	}
	return forecasts
}
