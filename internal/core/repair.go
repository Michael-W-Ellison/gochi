package core

import (
	"errors"
	"math"
	"reflect"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/environment"
	"github.com/Michael-W-Ellison/gochi/internal/simulation"
	"github.com/Michael-W-Ellison/gochi/internal/social"
)

// A save can be damaged: edited by hand, cut short or corrupted on disk, or
// made by a modified game. Load refuses a save without the parts that make
// the pet what it is (its body and personality), so the game uses a backup
// instead, and rebuilds the other parts, which older saves may not have,
// with their defaults, so a damaged save never crashes the game.

// ErrDamagedSave is returned by Load for a save missing an essential part
var ErrDamagedSave = errors.New("the save is damaged")

// repair checks a loaded pet and rebuilds any part that is missing
func (p *DigitalPet) repair() error {
	if p.Biology == nil || p.Biology.Vitals == nil || p.Biology.Processes == nil ||
		p.Personality == nil || p.Personality.Traits == nil {
		return ErrDamagedSave
	}
	if c := p.Biology.Circadian; c != nil && (c.CurrentPhase < biology.PhaseAwake || c.CurrentPhase > biology.PhaseWaking) {
		c.CurrentPhase = biology.PhaseAwake
	}
	if !(p.Biology.LifeExtension >= 0) { // Also NaN
		p.Biology.LifeExtension = 0
	}
	// Saves from before potions were limited could have more
	p.Biology.LifeExtension = min(p.Biology.LifeExtension, MaxLifePotions*lifePotionDays)
	if p.Emotions == nil {
		p.Emotions = ai.NewEmotionStateWithPersonality(p.Personality.Traits)
	}
	if p.Memory == nil {
		p.Memory = ai.NewMemorySystem(100)
	}
	if p.Relationships == nil {
		p.Relationships = social.NewSocialRelationships(20)
	}
	if p.Biology.Physiology != nil {
		p.Biology.Physiology.EnsureSenses()
		p.Biology.Physiology.Clamp()
	}
	p.repairBreed()
	p.repairNeeds()
	p.repairEnvironment()
	if p.Checklist != nil && p.Checklist.Done == nil {
		p.Checklist.Done = make(map[string]int)
	}
	return nil
}

// repairNeeds rebuilds the psychological needs that are missing
func (p *DigitalPet) repairNeeds() {
	defaults := simulation.NewNeedsManager()
	if p.Needs == nil {
		p.Needs = defaults
		return
	}
	if p.Needs.Needs == nil {
		p.Needs.Needs = defaults.Needs
		return
	}
	for kind, need := range defaults.Needs {
		if p.Needs.Needs[kind] == nil {
			p.Needs.Needs[kind] = need
		}
	}
}

// repairEnvironment rebuilds the parts of the environment that are missing
// and brings its clock and calendar back into range
func (p *DigitalPet) repairEnvironment() {
	env := p.Environment
	if env == nil {
		return // Created when first needed (ensureEnvironmentLocked)
	}
	defaults := environment.NewEnvironment()
	if env.Weather == nil {
		env.Weather = defaults.Weather
	}
	w, dw := env.Weather, defaults.Weather
	w.TransitionRate = inRange(w.TransitionRate, 0, 1, dw.TransitionRate)
	w.BaseTemperature = inRange(w.BaseTemperature, -40, 50, dw.BaseTemperature)
	w.TemperatureRange = inRange(w.TemperatureRange, 0, 60, dw.TemperatureRange)
	w.ChangeFrequency = inRange(w.ChangeFrequency, 0.5, 240, dw.ChangeFrequency)
	if env.Seasons == nil || env.Seasons.DaysPerSeason < 1 {
		env.Seasons = defaults.Seasons
	}
	if env.Seasons.CurrentSeason < environment.SeasonSpring || env.Seasons.CurrentSeason > environment.SeasonWinter {
		env.Seasons.CurrentSeason = environment.SeasonSpring
	}
	if !validWeather(env.Weather.Current.Type) {
		env.Weather.Current.Type = environment.WeatherClear
	}
	if t := env.Weather.TransitionTarget; t != nil && !validWeather(t.Type) {
		env.Weather.TransitionTarget = nil
	}
	if env.Seasons.DayOfSeason < 1 || env.Seasons.DayOfSeason > env.Seasons.DaysPerSeason {
		env.Seasons.DayOfSeason = 1
	}
	if env.Locations == nil {
		env.Locations = defaults.Locations
	}
	if env.Locations.Locations == nil {
		env.Locations.Locations = defaults.Locations.Locations
	}
	for name, loc := range env.Locations.Locations {
		if loc == nil {
			if fresh := defaults.Locations.Locations[name]; fresh != nil {
				env.Locations.Locations[name] = fresh
			} else {
				delete(env.Locations.Locations, name)
			}
		}
	}
	if !finite(env.TimeOfDay) {
		env.TimeOfDay = defaults.TimeOfDay
	}
	env.TimeOfDay = math.Mod(env.TimeOfDay, 24)
	if env.TimeOfDay < 0 {
		env.TimeOfDay += 24
	}
}

// validWeather reports whether a weather type is one the game knows
func validWeather(w environment.WeatherType) bool {
	return w >= environment.WeatherClear && w <= environment.WeatherCold
}

// finite reports whether x is a number other than an infinity
func finite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

// inRange returns x if it is from low to high, otherwise fallback
func inRange(x, low, high, fallback float64) float64 {
	if x >= low && x <= high { // False for NaN
		return x
	}
	return fallback
}

// replaceNonFinite sets every float in v that is saved (exported fields
// not marked json:"-", followed through pointers, slices and maps) and
// isn't a finite number to 0. What is saved has no cycles, or saving would
// fail anyway.
func replaceNonFinite(v reflect.Value) {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		if f := v.Float(); (math.IsNaN(f) || math.IsInf(f, 0)) && v.CanSet() {
			v.SetFloat(0)
		}
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			replaceNonFinite(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if field := v.Type().Field(i); field.IsExported() && field.Tag.Get("json") != "-" {
				replaceNonFinite(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			replaceNonFinite(v.Index(i))
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			// Map values can't be changed in place: change a copy
			value := reflect.New(v.Type().Elem()).Elem()
			value.Set(v.MapIndex(key))
			replaceNonFinite(value)
			v.SetMapIndex(key, value)
		}
	}
}
