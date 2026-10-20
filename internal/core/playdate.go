package core

import (
	"strings"

	"github.com/Michael-W-Ellison/gochi/internal/ai"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Playdates: players meet in one place, each with their active pet from
// their own game, and their pets play together. Each game sends the others
// a PlaymateCard describing its pet. When one player has their pet play
// with another's, their game works out how it goes (as for an
// introduction) and the other game records the same meeting from its
// pet's side with PlayedWithPlaymate.

// PlaymateCard describes a pet to the other players at a playdate. It
// holds only what a meeting needs.
type PlaymateCard struct {
	ID          types.PetID `json:"id"`
	Name        string      `json:"name"`
	Species     Species     `json:"species"`
	Young       bool        `json:"young"`
	Sociability float64     `json:"sociability"`           // How readily it takes to others, 0.0 to 1.0
	Unavailable string      `json:"unavailable,omitempty"` // Why it can't play now (an Unavailable code), or ""
}

// Card limits
const (
	maxCardIDLength   = 100
	maxCardNameLength = 40
)

// playdateMeeting is what a playdate meeting is recorded as
const playdateMeeting = "played at a playdate"

// PlaymateCard returns the card describing this pet
func (p *DigitalPet) PlaymateCard() PlaymateCard {
	info := p.socialInfo()
	return PlaymateCard{ID: info.id, Name: info.name, Species: info.species, Young: info.young,
		Sociability: sociability(info.traits), Unavailable: info.unavailableReason()}
}

// Valid reports whether a card from another game can be used
func (c PlaymateCard) Valid() bool {
	_, reasonKnown := unavailablePhrases[c.Unavailable]
	return c.ID != "" && len(c.ID) <= maxCardIDLength && strings.TrimSpace(c.Name) != "" &&
		len([]rune(c.Name)) <= maxCardNameLength && c.Species.Known() && (c.Unavailable == "" || reasonKnown)
}

// info returns the social information a card gives: traits whose
// sociability is the card's
func (c PlaymateCard) info() socialInfo {
	s := min(max(c.Sociability, 0), 1)
	return socialInfo{id: c.ID, name: c.Name, species: c.Species, young: c.Young, alive: true,
		traits: ai.Traits{Extraversion: s, Agreeableness: s, Territoriality: 1 - s, Neuroticism: 1 - s}}
}

// PlayWithPlaymate has this pet play with another player's pet at a
// playdate. It returns the meeting and the bond the other game records
// with PlayedWithPlaymate.
func (p *DigitalPet) PlayWithPlaymate(other PlaymateCard) (Meeting, float64, error) {
	if !other.Valid() {
		return Meeting{}, 0, RefusedError{"That pet can't be found."}
	}
	if other.ID == p.ID {
		return Meeting{}, 0, ErrSamePet
	}
	if other.Unavailable != "" {
		return Meeting{}, 0, RefusedError{other.Name + " " + unavailablePhrases[other.Unavailable] + "."}
	}
	me := p.socialInfo()
	if reason := me.unavailable(); reason != "" {
		return Meeting{}, 0, RefusedError{reason + "."}
	}
	return p.meet(me, other.info(), playdateMeeting)
}

// PlayedWithPlaymate records, from this pet's side, that another player's
// pet played with it: how the meeting went (quality, 0.0 to 1.0) and the
// bond it made (-1.0 to 1.0). It returns how they get along now, or "" if
// this pet can't play now.
func (p *DigitalPet) PlayedWithPlaymate(other PlaymateCard, quality, bond float64) string {
	if !other.Valid() || other.ID == p.ID {
		return ""
	}
	if p.socialInfo().unavailable() != "" {
		return ""
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	description := p.recordMeetingLocked(other.info(), min(max(bond, -1), 1), playdateMeeting)
	p.meetLocked(min(max(quality, 0), 1))
	return description
}
