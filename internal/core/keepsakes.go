package core

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/internal/calendar"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Keepsakes are the pages of a pet's book: the moments of its life, each
// with a snapshot of how it looked (its look and outfit, as drawing
// settings, not an image) and a line of text from a fixed list. The game
// server keeps each pet's book in a file on this computer, which stays
// after the pet is deleted until the player deletes it too.

// Kinds of keepsake
const (
	KeepsakeHome       = "home"        // It came home: created, or born ("born")
	KeepsakeGrewUp     = "grew_up"     // It grew up
	KeepsakeSenior     = "senior"      // It became a senior
	KeepsakeTrick      = "first_trick" // Its first trick (the trick's ID)
	KeepsakeFirstTrip  = "first_trip"  // Its first trip (the place)
	KeepsakePlace      = "new_place"   // It discovered a place (the place)
	KeepsakeRibbon     = "ribbon"      // A show ribbon ("show/level/place")
	KeepsakeSet        = "set"         // It completed a place's set of souvenirs (the place)
	KeepsakeBaby       = "first_baby"  // Its first baby (the baby's name)
	KeepsakeBestFriend = "best_friend" // Its first best friend (the friend's name)
	KeepsakeHoliday    = "holiday"     // A holiday played together (the holiday's ID)
)

// maxKeepsakes is how many pages a pet's book holds
const maxKeepsakes = 120

// bestFriendBond is the bond that makes a best friend
const bestFriendBond = 0.8

// Keepsake is a page of a pet's book
type Keepsake struct {
	Kind   string            `json:"kind"`
	Detail string            `json:"detail,omitempty"`
	Age    float64           `json:"age"`    // The pet's age, in game days
	Day    string            `json:"day"`    // The real date, YYYY-MM-DD
	Look   *PetLook          `json:"look"`   // How it looked
	Outfit map[string]string `json:"outfit"` // What it wore
}

// Memorial is the closing page of the book of a pet that has died
type Memorial struct {
	Age        float64 `json:"age"`
	Cause      string  `json:"cause"`
	Tricks     int     `json:"tricks"`
	Places     int     `json:"places"`
	Ribbons    int     `json:"ribbons"`
	Favorite   string  `json:"favorite,omitempty"`    // What it liked best
	BestFriend string  `json:"best_friend,omitempty"` // The pet it was closest to
}

// onceKeepsakes are the kinds a book has at most once
var onceKeepsakes = []string{KeepsakeHome, KeepsakeGrewUp, KeepsakeSenior, KeepsakeTrick, KeepsakeFirstTrip,
	KeepsakeBaby, KeepsakeBestFriend}

// keepLocked adds a page to the pet's book. Caller must hold p.mu.
func (p *DigitalPet) keepLocked(kind, detail string) {
	if slices.Contains(onceKeepsakes, kind) && p.hasKeepsakeLocked(kind, "") {
		return
	}
	if len(p.Keepsakes) >= maxKeepsakes {
		return
	}
	k := Keepsake{Kind: kind, Detail: detail, Age: p.Biology.GetAgeInDays(), Day: today(), Outfit: maps.Clone(p.Outfit)}
	if p.Look != nil {
		look := *p.Look
		k.Look = &look
	}
	p.Keepsakes = append(p.Keepsakes, k)
}

// hasKeepsakeLocked reports whether the book has a page of a kind (with
// the detail, unless detail is ""). Caller must hold p.mu.
func (p *DigitalPet) hasKeepsakeLocked(kind, detail string) bool {
	return slices.ContainsFunc(p.Keepsakes, func(k Keepsake) bool {
		return k.Kind == kind && (detail == "" || k.Detail == detail)
	})
}

// KeepSetCompleted records that the pet completed a place's set of
// souvenirs for the household
func (p *DigitalPet) KeepSetCompleted(place string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.hasKeepsakeLocked(KeepsakeSet, place) {
		p.keepLocked(KeepsakeSet, place)
	}
}

// BookInfo is what a pet's book shows about it besides its pages
type BookInfo struct {
	Name    string
	Species Species
	Breed   string
	Look    *PetLook
	Outfit  map[string]string
	Alive   bool
	Parents []string // Pet IDs
}

// BookInfo returns what the pet's book shows about it
func (p *DigitalPet) BookInfo() BookInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	info := BookInfo{Name: p.Name, Species: p.speciesLocked(), Breed: p.Breed, Outfit: maps.Clone(p.Outfit),
		Alive: p.Biology.IsAlive}
	if p.Look != nil {
		look := *p.Look
		info.Look = &look
	}
	for _, id := range p.Parents {
		info.Parents = append(info.Parents, string(id))
	}
	return info
}

// BookPages returns the pages of the pet's book
func (p *DigitalPet) BookPages() []Keepsake {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.Keepsakes)
}

// keepLifeStageLocked records growing up and becoming a senior, given the
// stage the pet was at. Caller must hold p.mu.
func (p *DigitalPet) keepLifeStageLocked(before biology.LifeStage) {
	switch now := p.Biology.LifeStage(); {
	case before == biology.StageYoung && now != biology.StageYoung:
		p.keepLocked(KeepsakeGrewUp, "")
	case before == biology.StageAdult && now == biology.StageSenior:
		p.keepLocked(KeepsakeSenior, "")
	}
}

// keepMomentsLocked records the moments noticed as time passes: a best
// friend, and a holiday played together. Caller must hold p.mu.
func (p *DigitalPet) keepMomentsLocked() {
	if !p.hasKeepsakeLocked(KeepsakeBestFriend, "") {
		for _, id := range p.relationIDsLocked() {
			if rel := p.Relationships.Relationships[id]; rel.BondStrength >= bestFriendBond {
				if name := p.KnownPets[id]; name != "" {
					p.keepLocked(KeepsakeBestFriend, name)
					break
				}
			}
		}
	}
	// Holidays, checked once a day
	if p.Marks["keepsake_day"] == today() {
		return
	}
	if p.Marks == nil {
		p.Marks = make(map[string]string)
	}
	p.Marks["keepsake_day"] = today()
	year := today()[:4]
	for _, h := range holidaysToday() {
		if !slices.ContainsFunc(p.Keepsakes, func(k Keepsake) bool {
			return k.Kind == KeepsakeHoliday && k.Detail == h.ID && strings.HasPrefix(k.Day, year)
		}) {
			p.keepLocked(KeepsakeHoliday, h.ID)
		}
	}
}

// relationIDsLocked returns the IDs of the pets the pet has a relationship
// with, in order. Caller must hold p.mu.
func (p *DigitalPet) relationIDsLocked() []types.PetID {
	ids := make([]types.PetID, 0, len(p.Relationships.Relationships))
	for id := range p.Relationships.Relationships {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// MemorialOf returns the closing page of the book of a pet that has died,
// or nil if it is alive
func (p *DigitalPet) MemorialOf() *Memorial {
	p.mu.Lock()
	if p.Biology.IsAlive {
		p.mu.Unlock()
		return nil
	}
	m := &Memorial{Age: p.Biology.GetAgeInDays(), Cause: p.Biology.CauseOfDeath, Tricks: len(p.TricksLearned),
		Places: len(p.PlaceVisits), Ribbons: len(p.Ribbons)}
	best := 0.0
	for _, id := range p.relationIDsLocked() {
		if rel := p.Relationships.Relationships[id]; rel.BondStrength > best && p.KnownPets[id] != "" {
			best, m.BestFriend = rel.BondStrength, p.KnownPets[id]
		}
	}
	p.mu.Unlock()
	if likes, _ := p.LikesAndDislikes(); len(likes) > 0 {
		m.Favorite = likes[0]
	}
	return m
}

// KeepsakeText is a page's line, from a fixed list
func KeepsakeText(k Keepsake, name string) string {
	switch k.Kind {
	case KeepsakeHome:
		if k.Detail == "born" {
			return name + " was born."
		}
		return name + " came home."
	case KeepsakeGrewUp:
		return name + " grew up."
	case KeepsakeSenior:
		return name + " became a senior: time for gentle days."
	case KeepsakeTrick:
		if t, ok := findTrick(k.Detail); ok {
			return fmt.Sprintf("%s learned a first trick: %s.", name, strings.ToLower(t.Name))
		}
		return name + " learned a first trick."
	case KeepsakeFirstTrip:
		return fmt.Sprintf("%s's first trip, to %s.", name, placePhrase(k.Detail))
	case KeepsakePlace:
		return fmt.Sprintf("%s discovered %s.", name, placePhrase(k.Detail))
	case KeepsakeRibbon:
		parts := strings.Split(k.Detail, "/")
		if len(parts) == 3 {
			show, _ := FindShow(parts[0])
			level := map[string]string{"1": "Novice", "2": "Open", "3": "Champion"}[parts[1]]
			place := map[string]string{"1": "1st", "2": "2nd", "3": "3rd"}[parts[2]]
			return fmt.Sprintf("%s placed %s at the %s %s.", name, place, level, show.Name)
		}
		return name + " won a ribbon."
	case KeepsakeSet:
		return fmt.Sprintf("%s completed the household's set of souvenirs from %s.", name, placePhrase(k.Detail))
	case KeepsakeBaby:
		return fmt.Sprintf("%s's first baby, %s, was born.", name, k.Detail)
	case KeepsakeBestFriend:
		return fmt.Sprintf("%s and %s became best friends.", name, k.Detail)
	case KeepsakeHoliday:
		if h, ok := FindHoliday(k.Detail); ok {
			return fmt.Sprintf("%s %s together.", h.Icon, h.Name)
		}
	}
	return ""
}

// placePhrase names a place, e.g. "the park"
func placePhrase(id string) string {
	if d, ok := placeDetails[id]; ok {
		return d.phrase
	}
	return "somewhere new"
}

// holidaysToday returns the holidays (turned on) that are today
func holidaysToday() []Holiday {
	var list []Holiday
	day := calendar.DateOf(now())
	for _, h := range holidays {
		if !HolidayOn(h.ID) {
			continue
		}
		if start, end := h.Next(day); start != (calendar.Date{}) && !day.Before(start) && !end.Before(day) {
			list = append(list, h)
		}
	}
	return list
}

// fillHomeLookLocked gives the first page the pet's look once it has one
// (a new pet's look is set after it is made). Caller must hold p.mu.
func (p *DigitalPet) fillHomeLookLocked() {
	for i := range p.Keepsakes {
		if p.Keepsakes[i].Kind == KeepsakeHome && p.Keepsakes[i].Look == nil && p.Look != nil {
			look := *p.Look
			p.Keepsakes[i].Look = &look
		}
	}
}

// keepBornLocked marks the book of a pet that was born, not created.
// Caller must hold p.mu.
func (p *DigitalPet) keepBornLocked() {
	for i := range p.Keepsakes {
		if p.Keepsakes[i].Kind == KeepsakeHome {
			p.Keepsakes[i].Detail = "born"
		}
	}
}
