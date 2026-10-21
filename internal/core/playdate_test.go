package core

import (
	"errors"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
	"testing"
)

func TestPlaymateCard(t *testing.T) {
	pet := sociablePet("Biscuit", SpeciesDog)
	card := pet.PlaymateCard()
	if card.ID != pet.ID || card.Name != "Biscuit" || card.Species != SpeciesDog || !card.Valid() {
		t.Errorf("Card = %+v", card)
	}
	if want := sociability(*pet.Personality.Traits); card.Sociability != want {
		t.Errorf("Card sociability = %v, want %v", card.Sociability, want)
	}
	if got := sociability(card.info().traits); got != card.Sociability {
		t.Errorf("A card's traits should have its sociability: %v, want %v", got, card.Sociability)
	}

	pet.Biology.IsAlive = false
	if card := pet.PlaymateCard(); card.Unavailable == "" {
		t.Error("A dead pet's card should say it can't play")
	}

	for _, bad := range []PlaymateCard{
		{Name: "No ID", Species: SpeciesCat},
		{ID: "x", Species: SpeciesCat},
		{ID: "x", Name: "Fish", Species: "fish"},
		{ID: "x", Name: "This name is much too long to be a pet's name at all", Species: SpeciesCat},
	} {
		if bad.Valid() {
			t.Errorf("Card %+v should be invalid", bad)
		}
	}
}

func TestPlaydateMeetingOnBothSides(t *testing.T) {
	// Two pets in different games
	mine := sociablePet("Biscuit", SpeciesDog)
	theirs := sociablePet("Rex", SpeciesDog)

	meeting, bond, err := mine.PlayWithPlaymate(theirs.PlaymateCard())
	if err != nil {
		t.Fatal(err)
	}
	if meeting.Quality < goodMeeting || bond <= 0 {
		t.Fatalf("Two sociable dogs should play well, got quality %v bond %v", meeting.Quality, bond)
	}
	if mine.RelationTo(theirs.ID) == "" {
		t.Error("My pet should now know theirs")
	}

	// The other game records it from its pet's side
	if got := theirs.PlayedWithPlaymate(mine.PlaymateCard(), meeting.Quality, bond); got == "" {
		t.Error("Their pet should record the playdate")
	}
	rels := theirs.Relations()
	if len(rels) != 1 || rels[0].Name != "Biscuit" || rels[0].LastShared != playdateMeeting {
		t.Errorf("Their pet's relations = %+v", rels)
	}
}

func TestPlaydateRefusals(t *testing.T) {
	mine := sociablePet("Biscuit", SpeciesDog)
	if _, _, err := mine.PlayWithPlaymate(mine.PlaymateCard()); !errors.Is(err, ErrSamePet) {
		t.Errorf("A pet can't play with itself, got %v", err)
	}
	away := sociablePet("Rex", SpeciesDog).PlaymateCard()
	away.Unavailable = UnavailableAsleep
	var refused RefusedError
	if _, _, err := mine.PlayWithPlaymate(away); !errors.As(err, &refused) || refused.Error() != "Rex is asleep." {
		t.Errorf("An unavailable playmate should be refused, got %v", err)
	}
	if _, _, err := mine.PlayWithPlaymate(PlaymateCard{}); err == nil {
		t.Error("An invalid card should be refused")
	}

	// A pet that can't play now doesn't record a meeting
	other := sociablePet("Rex", SpeciesDog)
	mine.Biology.IsAlive = false
	if got := mine.PlayedWithPlaymate(other.PlaymateCard(), 0.8, 0.5); got != "" || len(mine.Relations()) != 0 {
		t.Errorf("A dead pet shouldn't record a playdate, got %q", got)
	}
}

func TestPlaymateCardCarriesNoText(t *testing.T) {
	pet := sociablePet("Biscuit", SpeciesDog)
	pet.CurrentBehavior = types.BehaviorSleeping
	if card := pet.PlaymateCard(); card.Unavailable != UnavailableAsleep {
		t.Errorf("Why it can't play should be a code, got %q", card.Unavailable)
	}
	bad := sociablePet("Rex", SpeciesDog).PlaymateCard()
	bad.Unavailable = "meet me at the park"
	if bad.Valid() {
		t.Error("A card with a reason that isn't a code should be invalid")
	}
}
