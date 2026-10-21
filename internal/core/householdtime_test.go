package core

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/biology"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestLivingInTheHousehold(t *testing.T) {
	pet := NewDigitalPet("Pip", "owner")
	pet.Biology.Processes.Age = biology.AdultAge - 5
	v := pet.Biology.Vitals
	v.Nutrition, v.Hydration, v.Stress = 0.2, 0.3, 0.9
	pet.LivingAreaWaste = 0.7
	news := pet.LiveInHousehold(10)
	if !news.GrewUp || news.Died || pet.Biology.GetAgeInDays() != biology.AdultAge+5 {
		t.Errorf("Ten days in the household: %+v, age %.1f", news, pet.Biology.GetAgeInDays())
	}
	if v.Nutrition < householdLevel || v.Hydration < householdLevel || v.Stress > 0.2 || pet.LivingAreaWaste != 0 {
		t.Errorf("The household looks after it: %+v waste %.2f", *v, pet.LivingAreaWaste)
	}
	if pet.FriendshipPoints != 0 {
		t.Error("No points for the household's care")
	}

	// A paused pet stays as it is
	paused := NewDigitalPet("Still", "owner")
	paused.SetHouseholdPaused(true)
	before := paused.Biology.GetAgeInDays()
	paused.LiveInHousehold(10)
	if paused.Biology.GetAgeInDays() != before || !paused.IsHouseholdPaused() {
		t.Error("A paused pet doesn't age")
	}

	// Old age comes on schedule
	old := NewDigitalPet("Old", "owner")
	old.Biology.Processes.Age = 1e6
	if news := old.LiveInHousehold(1); !news.Died || old.Biology.IsAlive {
		t.Errorf("A very old pet: %+v", news)
	}
}

func TestPetsLivingTogetherGrowCloser(t *testing.T) {
	a, b, c := NewDigitalPet("A", "o"), NewDigitalPet("B", "o"), NewDigitalPet("C", "o")
	a.Relationships.AddRelationship(b.ID, types.RelationshipFriend).BondStrength = 0.3
	b.Relationships.AddRelationship(a.ID, types.RelationshipFriend).BondStrength = 0.3
	a.Relationships.AddRelationship(c.ID, types.RelationshipFriend).BondStrength = 0.3
	c.HouseholdPaused = true
	LiveTogether(10, a, b, c)
	ab, _ := a.Relationships.GetRelationship(b.ID)
	ba, _ := b.Relationships.GetRelationship(a.ID)
	ac, _ := a.Relationships.GetRelationship(c.ID)
	if ab.BondStrength <= 0.3 || ba.BondStrength <= 0.3 || ac.BondStrength != 0.3 {
		t.Errorf("Bonds: a-b %.3f, b-a %.3f, a-c (paused) %.3f", ab.BondStrength, ba.BondStrength, ac.BondStrength)
	}
	if _, ok := b.Relationships.GetRelationship(c.ID); ok {
		t.Error("Pets that haven't met don't become friends by living together")
	}
}
