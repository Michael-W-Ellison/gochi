package core

import (
	"errors"
	"slices"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Families: a parent can give Friendship Points to its child while the
// child is young; once the child is grown up, only when the parent dies.
// When a pet dies, its belongings (equipment, decorations, cosmetics and
// unopened supplies) can be passed on to one of its living children, once.
// A baby starts with nothing.

// Family errors
var (
	ErrNotChild        = errors.New("not this pet's child")
	ErrChildGrownUp    = errors.New("the child has grown up")
	ErrNothingToGive   = errors.New("nothing to give")
	ErrStillAlive      = errors.New("the pet is still alive")
	ErrAlreadyPassedOn = errors.New("already passed on")
)

// IsChildOf reports whether the pet is a child of the pet with parentID
func (p *DigitalPet) IsChildOf(parentID types.PetID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if slices.Contains(p.Parents, parentID) {
		return true
	}
	// Born before parents were recorded: the relationship says so
	rel, ok := p.Relationships.GetRelationship(parentID)
	return ok && rel.Type == types.RelationshipParent
}

// IsYoung reports whether the pet hasn't grown up yet
func (p *DigitalPet) IsYoung() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Biology.LifeStage() == biology.StageYoung
}

// YoungDaysLeft returns the game days until the pet grows up (0 if it has)
func (p *DigitalPet) YoungDaysLeft() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return max(biology.AdultAge-p.Biology.GetAgeInDays(), 0)
}

// lockPair locks two pets in a fixed order, so two transfers can't
// deadlock, and returns the unlock
func lockPair(a, b *DigitalPet) func() {
	first, second := a, b
	if b.ID < a.ID {
		first, second = b, a
	}
	first.mu.Lock()
	second.mu.Lock()
	return func() {
		second.mu.Unlock()
		first.mu.Unlock()
	}
}

// TransferPoints gives amount of a parent's Friendship Points to its
// child: to a young child while the parent lives, to any living child once
// it has died
func TransferPoints(parent, child *DigitalPet, amount int) error {
	if !child.IsChildOf(parent.ID) {
		return ErrNotChild
	}
	unlock := lockPair(parent, child)
	defer unlock()
	switch {
	case !child.Biology.IsAlive:
		return ErrPetNotAvailable
	case parent.Biology.IsAlive && child.Biology.LifeStage() != biology.StageYoung:
		return ErrChildGrownUp
	case amount <= 0 || amount > parent.FriendshipPoints:
		return ErrNothingToGive
	}
	parent.FriendshipPoints -= amount
	child.FriendshipPoints += amount
	child.GiftedPoints += amount
	return nil
}

// Inheritance is what a child was given when its parent died
type Inheritance struct {
	Items     []string `json:"items"`     // Equipment and decorations
	Cosmetics []string `json:"cosmetics"` // Cosmetics
	Supplies  int      `json:"supplies"`  // Units of supplies
}

// PassOn gives a dead pet's belongings to one of its living children:
// equipment, decorations and cosmetics the child doesn't have, and the
// unopened supplies in its pantry (as far as the child's pantry has room).
// A pet's belongings can be passed on once.
func PassOn(parent, child *DigitalPet) (Inheritance, error) {
	if !child.IsChildOf(parent.ID) {
		return Inheritance{}, ErrNotChild
	}
	unlock := lockPair(parent, child)
	defer unlock()
	switch {
	case parent.Biology.IsAlive:
		return Inheritance{}, ErrStillAlive
	case parent.PassedOnTo != "":
		return Inheritance{}, ErrAlreadyPassedOn
	case !child.Biology.IsAlive:
		return Inheritance{}, ErrPetNotAvailable
	}

	got := Inheritance{Items: []string{}, Cosmetics: []string{}}
	for _, id := range parent.Inventory {
		if !child.ownsLocked(id) {
			child.Inventory = append(child.Inventory, id)
			if item, ok := findItem(id); ok {
				child.startUpkeepLocked(item)
			}
			got.Items = append(got.Items, id)
		}
	}
	for _, id := range parent.Cosmetics {
		if !slices.Contains(child.Cosmetics, id) {
			child.Cosmetics = append(child.Cosmetics, id)
			got.Cosmetics = append(got.Cosmetics, id)
		}
	}
	child.FirstAidUses += parent.FirstAidUses
	parent.ensureSuppliesLocked()
	child.ensureSuppliesLocked()
	for _, owned := range parent.Supplies.Items {
		if owned.Quantity <= 0 || !child.Supplies.AddItem(owned.Item, owned.Quantity) {
			continue
		}
		got.Supplies += owned.Quantity
	}

	parent.Inventory, parent.Cosmetics, parent.Outfit, parent.FirstAidUses = nil, nil, nil, 0
	parent.Supplies.Items = nil
	parent.Supplies = nil
	parent.PassedOnTo = child.ID
	return got, nil
}

// MarkAdultWarned records that the player was warned the pet is about to
// grow up, and reports whether it already had been
func (p *DigitalPet) MarkAdultWarned() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	warned := p.AdultWarned
	p.AdultWarned = true
	return warned
}
