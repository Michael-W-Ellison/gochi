package biology

// Hormones are the pet's hormone levels, 0.0 to 1.0. Each moves towards
// a level set by the pet's state (its target) at its own speed, and some
// also rise sharply with events (Pulse): a fright releases cortisol,
// affection oxytocin.
type Hormones struct {
	Cortisol  float64 `json:"cortisol"`  // Stress hormone: follows stress and frights
	Growth    float64 `json:"growth"`    // Growth hormone: high in the young and in deep sleep; repairs the body
	Serotonin float64 `json:"serotonin"` // Mood regulator: raised by exercise, company, daylight and food
	Oxytocin  float64 `json:"oxytocin"`  // Bonding hormone: released by affection; calms
	Melatonin float64 `json:"melatonin"` // Sleep hormone: rises in the dark
}

// Hormone names, for Pulse
const (
	HormoneCortisol  = "cortisol"
	HormoneGrowth    = "growth"
	HormoneSerotonin = "serotonin"
	HormoneOxytocin  = "oxytocin"
	HormoneMelatonin = "melatonin"
)

// How fast each hormone follows its target: the share of the gap closed
// per game day
const (
	cortisolSpeed  = 4.0
	growthSpeed    = 6.0
	serotoninSpeed = 2.0
	oxytocinSpeed  = 3.0
	melatoninSpeed = 8.0
)

// NewHormones returns a calm adult's resting levels
func NewHormones() *Hormones {
	return &Hormones{Cortisol: 0.2, Growth: 0.3, Serotonin: 0.5, Oxytocin: 0.2, Melatonin: 0.1}
}

// Clamp keeps every level in range
func (h *Hormones) Clamp() {
	h.Cortisol = clamp(h.Cortisol, 0, 1)
	h.Growth = clamp(h.Growth, 0, 1)
	h.Serotonin = clamp(h.Serotonin, 0, 1)
	h.Oxytocin = clamp(h.Oxytocin, 0, 1)
	h.Melatonin = clamp(h.Melatonin, 0, 1)
}

// Pulse raises (or, with a negative amount, lowers) a hormone at once
func (h *Hormones) Pulse(hormone string, amount float64) {
	switch hormone {
	case HormoneCortisol:
		h.Cortisol += amount
	case HormoneGrowth:
		h.Growth += amount
	case HormoneSerotonin:
		h.Serotonin += amount
	case HormoneOxytocin:
		h.Oxytocin += amount
	case HormoneMelatonin:
		h.Melatonin += amount
	}
	h.Clamp()
}

// approach moves level towards target, closing speed x deltaTime of the gap
// (all of it if that is 1 or more)
func approach(level, target, speed, deltaTime float64) float64 {
	return level + (target-level)*min(speed*deltaTime, 1)
}

// BodyInputs are what the body responds to from outside it, set by the
// pet before each update (not saved)
type BodyInputs struct {
	TimeOfDay float64 // Hour, 0-24
	Daylight  float64 // 0.0 dark to 1.0 full daylight
	Asleep    bool
	Exercise  float64 // How well the pet's need for exercise is met, 0.0 to 1.0
	Company   float64 // How well its needs for company and affection are met, 0.0 to 1.0
}

// updateHormones moves each hormone towards its target over deltaTime game
// days
func (b *BiologicalSystems) updateHormones(deltaTime float64) {
	h, v, in := b.Hormones, b.Vitals, b.Inputs

	// Oxytocin settles back to a low resting level after affection
	h.Oxytocin = approach(h.Oxytocin, 0.1+0.2*in.Company, oxytocinSpeed, deltaTime)

	// Cortisol follows stress; affection (oxytocin) damps it
	h.Cortisol = approach(h.Cortisol, clamp(0.1+0.8*v.Stress-0.3*h.Oxytocin, 0, 1), cortisolSpeed, deltaTime)

	// Growth hormone is high in the young and pulses in deep sleep
	growth := b.growthTarget()
	if b.Circadian.GetCurrentPhase() == PhaseDeepSleep {
		growth = min(growth*2+0.2, 1)
	}
	h.Growth = approach(h.Growth, growth, growthSpeed, deltaTime)

	// Serotonin rises with exercise, company, daylight and a full stomach,
	// and falls with lasting stress
	serotonin := 0.15 + 0.25*in.Exercise + 0.25*in.Company + 0.2*in.Daylight + 0.15*v.Nutrition - 0.2*h.Cortisol
	h.Serotonin = approach(h.Serotonin, clamp(serotonin, 0, 1), serotoninSpeed, deltaTime)

	// Melatonin rises in the dark and stays up while the pet sleeps
	melatonin := 0.9 * (1 - in.Daylight)
	if in.Asleep {
		melatonin = max(melatonin, 0.5)
	}
	h.Melatonin = approach(h.Melatonin, melatonin, melatoninSpeed, deltaTime)

	h.Clamp()
}

// EnsureHormones gives a pet saved before hormones existed resting levels
func (b *BiologicalSystems) EnsureHormones() {
	if b.Hormones == nil {
		b.Hormones = NewHormones()
	}
}

// ImmuneResponse is how well the pet fights off illness now, 0.0 to 1.0:
// its inherited immune strength, weakened by lasting stress (cortisol) and
// old age
func (b *BiologicalSystems) ImmuneResponse() float64 {
	b.EnsureHormones()
	suppression := 0.4 * max(b.Hormones.Cortisol-0.3, 0) / 0.7
	return clamp(b.Processes.ImmuneStrength*(1-suppression)*b.agingFactor(0.3), 0, 1)
}

// SleepQualityFactor is how hormones change how restful sleep is: darkness
// (melatonin) helps, stress (cortisol) hurts
func (b *BiologicalSystems) SleepQualityFactor() float64 {
	b.EnsureHormones()
	return (1 + 0.2*b.Hormones.Melatonin) * (1 - 0.3*b.Hormones.Cortisol)
}

// moodSetPoint is the happiness the pet drifts towards when nothing
// happens: higher with more serotonin
func (b *BiologicalSystems) moodSetPoint() float64 {
	return 0.3 + 0.4*b.Hormones.Serotonin
}
