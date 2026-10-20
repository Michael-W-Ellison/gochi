package core

import "sort"

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

// forecastSteps is how many steps a forecast simulates
const forecastSteps = 60

// maxForecasts is how many forecasts are returned
const maxForecasts = 3

// Forecast returns what the pet will ask for within horizon game days if
// nothing changes, soonest first: the requests a copy of the pet makes as
// time passes, other than any it is making now. The copy is simulated in
// up to 60 steps, so a forecast is accurate to about 1/60 of the horizon.
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
	if now := p.CurrentRequest(); now != nil {
		seen[now.Action] = true
	}
	step := horizon / forecastSteps
	for elapsed := step; elapsed <= horizon+step/2; elapsed += step {
		sim.Update(step)
		if !sim.IsAlive() || sim.IsHibernating() {
			break
		}
		if r := sim.CurrentRequest(); r != nil && !seen[r.Action] && r.Action != "revive" {
			seen[r.Action] = true
			forecasts = append(forecasts, NeedForecast{Action: r.Action, Icon: r.Icon, Message: r.Message, InDays: elapsed})
			if len(forecasts) >= maxForecasts {
				break
			}
		}
	}
	sort.SliceStable(forecasts, func(i, j int) bool { return forecasts[i].InDays < forecasts[j].InDays })
	return forecasts
}
