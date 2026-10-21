package core

import "slices"

// Care records: counts and streaks that achievements are earned by, such
// as days the teeth were brushed, game days of calm, and illnesses caught
// early. Streaks are in game days and start over when broken. Some
// achievements are about the whole household (several pets, the house);
// the game server awards those to the pet that earned them.

// countLocked adds to a care record. Caller must hold p.mu.
func (p *DigitalPet) countLocked(key string, amount float64) {
	if p.Counters == nil {
		p.Counters = make(map[string]float64)
	}
	p.Counters[key] += amount
}

// countDayLocked counts a care record at most once a calendar day. Caller
// must hold p.mu.
func (p *DigitalPet) countDayLocked(key string) {
	if p.Marks == nil {
		p.Marks = make(map[string]string)
	}
	if day := today(); p.Marks[key] != day {
		p.Marks[key] = day
		p.countLocked(key, 1)
	}
}

// streakLocked extends a streak while ok holds and ends it otherwise.
// Caller must hold p.mu.
func (p *DigitalPet) streakLocked(key string, ok bool, deltaTime float64) {
	if ok {
		p.countLocked(key, deltaTime)
	} else if p.Counters[key] != 0 {
		p.Counters[key] = 0
	}
}

// updateCareRecordsLocked advances the care records by deltaTime game
// days. Caller must hold p.mu.
func (p *DigitalPet) updateCareRecordsLocked(deltaTime float64) {
	ph, v := p.Biology.Physiology, p.Biology.Vitals
	if p.Tendencies[TendEyes] > 0 {
		p.streakLocked("clear_eyes", ph.EyeHealth > 0.8, deltaTime)
	}
	p.streakLocked("calm", v.Stress < 0.2, deltaTime)

	age := p.Biology.GetAgeInDays()
	kept, any := true, false
	for _, id := range []string{fountainFilterUpkeep, beddingUpkeep, fleaCollarUpkeep, vaccinationUpkeep} {
		if until, ok := p.Upkeep[id]; ok {
			any = true
			kept = kept && until > age
		}
	}
	if any {
		p.streakLocked("well_kept", kept, deltaTime)
	}

	learned, rusty := 0, false
	for id := range p.TricksLearned {
		learned++
		rusty = rusty || p.TrickSkills[id] < 1.0
	}
	if learned > 0 {
		p.streakLocked("no_rust", !rusty, deltaTime)
	}
	if p.Counters["season_"+RealSeason().String()] == 0 {
		p.countLocked("season_"+RealSeason().String(), 1)
	}
}

// Award records achievements the game server decided the pet earned (ones
// about the whole household); they are announced like any other
func (p *DigitalPet) Award(ids ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, id := range ids {
		if p.Marks == nil {
			p.Marks = make(map[string]string)
		}
		p.Marks["award:"+id] = "1"
	}
}

// count adds to a care record
func (p *DigitalPet) count(key string, amount float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.countLocked(key, amount)
}

// awarded reports whether the game server awarded an achievement. Caller
// must hold p.mu.
func (p *DigitalPet) awarded(id string) bool {
	return p.Marks["award:"+id] != ""
}

// holidaysCelebratedLocked counts the holidays the pet has items or
// cosmetics from. Caller must hold p.mu.
func (p *DigitalPet) holidaysCelebratedLocked() int {
	var seen []string
	for _, id := range append(slices.Clone(p.Inventory), p.Cosmetics...) {
		if h := HolidayOf(id); h != "" && !slices.Contains(seen, h) {
			seen = append(seen, h)
		}
	}
	return len(seen)
}

// mixedFromPurebredsLocked reports whether the pet is a mixed breed whose
// parents were two different purebreds. Caller must hold p.mu.
func (p *DigitalPet) mixedFromPurebredsLocked() bool {
	if p.Lineage == nil || len(p.Lineage.Parents) != 2 || !p.breedDefLocked().Mixed {
		return false
	}
	a, b := p.Lineage.Parents[0], p.Lineage.Parents[1]
	ba, okA := FindBreed(a.Breed)
	bb, okB := FindBreed(b.Breed)
	return okA && okB && !ba.Mixed && !bb.Mixed && a.Breed != b.Breed
}

// careAchievements are achievements for care, training, family and
// seasons (added to the list in records.go). Conditions run with p.mu
// held.
var careAchievements = []struct {
	Achievement
	earned func(p *DigitalPet) bool
}{
	{Achievement{ID: "clean_teeth", Name: "Clean Teeth", Icon: "🪥", Rarity: "common", Points: 10,
		Description: "Had its teeth brushed on 7 days"},
		func(p *DigitalPet) bool { return p.Counters["teeth_days"] >= 7 }},
	{Achievement{ID: "clear_eyes", Name: "Clear Eyes", Icon: "👁️", Rarity: "rare", Points: 25,
		Description: "A breed prone to sore eyes kept them healthy for 1,000 game days"},
		func(p *DigitalPet) bool { return p.Counters["clear_eyes"] >= 1000 }},
	{Achievement{ID: "well_kept", Name: "Well Kept", Icon: "🧺", Rarity: "rare", Points: 25,
		Description: "No upkeep ran out for 500 game days"},
		func(p *DigitalPet) bool { return p.Counters["well_kept"] >= 500 }},
	{Achievement{ID: "calm_companion", Name: "Calm Companion", Icon: "🧘", Rarity: "rare", Points: 25,
		Description: "Stress stayed under 20% for 300 game days in a row"},
		func(p *DigitalPet) bool { return p.Counters["calm"] >= 300 }},
	{Achievement{ID: "kidney_care", Name: "Kidney Care", Icon: "🩺", Rarity: "epic", Points: 50,
		Description: "Lived to 90% of its lifespan with inherited kidney disease"},
		func(p *DigitalPet) bool {
			return p.Tendencies[TendKidneys] > 0 && p.Biology.GetAgeInDays() >= 0.9*p.Biology.ExpectedLifespan()
		}},
	{Achievement{ID: "vets_friend", Name: "Vet's Friend", Icon: "🏥", Rarity: "rare", Points: 25,
		Description: "Had 3 illnesses treated early"},
		func(p *DigitalPet) bool { return p.Counters["early_treated"] >= 3 }},
	{Achievement{ID: "six_tricks", Name: "Six Tricks", Icon: "🎓", Rarity: "common", Points: 10,
		Description: "Learned six tricks"},
		func(p *DigitalPet) bool { return len(p.TricksLearned) >= 6 }},
	{Achievement{ID: "signal_master", Name: "Signal Master", Icon: "👋", Rarity: "rare", Points: 25,
		Description: "Learned 3 tricks while hard of hearing"},
		func(p *DigitalPet) bool {
			return p.Biology.Physiology != nil && p.Biology.Physiology.Hearing <= 0.55 && len(p.TricksLearned) >= 3
		}},
	{Achievement{ID: "no_rust", Name: "No Rust", Icon: "✨", Rarity: "rare", Points: 25,
		Description: "Kept every trick sharp for 500 game days"},
		func(p *DigitalPet) bool { return p.Counters["no_rust"] >= 500 }},
	{Achievement{ID: "patient_trainer", Name: "Patient Trainer", Icon: "🍬", Rarity: "rare", Points: 25,
		Description: "50 training sessions with rewards and never a scolding"},
		func(p *DigitalPet) bool { return p.Counters["reward_training"] >= 50 && p.Counters["scolding"] == 0 }},
	{Achievement{ID: "three_generations", Name: "Three Generations", Icon: "🌳", Rarity: "rare", Points: 25,
		Description: "Is the grandchild of pets you raised"},
		func(p *DigitalPet) bool { p.ensureGenomeLocked(); return p.Genome.Generation >= 3 }},
	{Achievement{ID: "five_generations", Name: "Five Generations", Icon: "🏡", Rarity: "legendary", Points: 100,
		Description: "Is the fifth generation of a family you raised"},
		func(p *DigitalPet) bool { p.ensureGenomeLocked(); return p.Genome.Generation >= 5 }},
	{Achievement{ID: "family_reunion", Name: "Family Reunion", Icon: "👪", Rarity: "common", Points: 10,
		Description: "Did a group activity with its parent or child"},
		func(p *DigitalPet) bool { return p.Counters["family_reunion"] > 0 }},
	{Achievement{ID: "mixed_blessings", Name: "Mixed Blessings", Icon: "🧬", Rarity: "common", Points: 10,
		Description: "A mixed breed whose parents were two different breeds"},
		func(p *DigitalPet) bool { return p.mixedFromPurebredsLocked() }},
	{Achievement{ID: "four_seasons", Name: "Four Seasons", Icon: "🍂", Rarity: "epic", Points: 50,
		Description: "Lived through all four real seasons"},
		func(p *DigitalPet) bool {
			for _, s := range []string{"Spring", "Summer", "Fall", "Winter"} {
				if p.Counters["season_"+s] == 0 {
					return false
				}
			}
			return true
		}},
	{Achievement{ID: "holiday_spirit", Name: "Holiday Spirit", Icon: "🎉", Rarity: "rare", Points: 25,
		Description: "Has things from 5 different holidays"},
		func(p *DigitalPet) bool { return p.holidaysCelebratedLocked() >= 5 }},
	// Awarded by the game server
	{Achievement{ID: "full_house", Name: "Full House", Icon: "🏠", Rarity: "rare", Points: 25,
		Description: "Lives with 4 or more other pets"},
		func(p *DigitalPet) bool { return p.awarded("full_house") }},
	{Achievement{ID: "dream_home", Name: "Dream Home", Icon: "🏰", Rarity: "epic", Points: 50,
		Description: "Its home has every house upgrade"},
		func(p *DigitalPet) bool { return p.awarded("dream_home") }},
	{Achievement{ID: "cat_fancier", Name: "Cat Fancier", Icon: "🐈", Rarity: "rare", Points: 25,
		Description: "You have raised 5 breeds of cat"},
		func(p *DigitalPet) bool { return p.awarded("cat_fancier") }},
	{Achievement{ID: "dog_lover", Name: "Dog Lover", Icon: "🐕", Rarity: "rare", Points: 25,
		Description: "You have raised 5 breeds of dog"},
		func(p *DigitalPet) bool { return p.awarded("dog_lover") }},
}

func init() {
	achievements = append(achievements, careAchievements...)
}
