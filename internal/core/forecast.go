package core

import (
	"math"
	"sort"
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

// Forecast returns what the pet will ask for within horizon game days if
// nothing changes, soonest first: the needs a copy of the pet comes to
// have as time passes, other than any it has now. Every need counts, not
// only the most urgent, so one starting while a more urgent one waits is
// still seen.
//
// The copy is looked at on a fixed grid of the pet's age (see
// forecastStep), so forecasts made a moment apart look at the same moments
// of the pet's future and agree; a forecast is accurate to one step.
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

	seen := make(map[string]bool)
	for _, need := range p.currentNeeds() {
		seen[need.Action] = true
	}
	step := forecastStep(horizon)
	now := sim.Biology.GetAgeInDays()
	simAge := now
	for n := math.Floor(now/step) + 1; ; n++ { // The grid points after now
		at := n * step
		if at-now > horizon {
			break
		}
		sim.Update(at - simAge)
		simAge = at
		if !sim.IsAlive() || sim.IsHibernating() {
			break
		}
		for _, r := range sim.currentNeeds() {
			if !seen[r.Action] && r.Action != "revive" {
				seen[r.Action] = true
				forecasts = append(forecasts, NeedForecast{Action: r.Action, Icon: r.Icon, Message: r.Message, InDays: at - now})
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
