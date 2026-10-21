package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// friendlyAdult returns a sociable, playful adult pet (see sociablePet)
func friendlyAdult(name string, species Species) *DigitalPet {
	pet := sociablePet(name, species)
	pet.Biology.Processes.Age = 100
	pet.Personality.Traits.Playfulness = 0.8
	return pet
}

// grumpyAdult returns a territorial, irritable adult pet (see grumpyPet)
func grumpyAdult(name string, species Species) *DigitalPet {
	pet := grumpyPet(name, species)
	pet.Biology.Processes.Age = 100
	pet.Personality.Traits.Agreeableness = 0.1
	return pet
}

func TestGroupPlayBetweenFriendlyDogs(t *testing.T) {
	rex, max, bella := friendlyAdult("Rex", SpeciesDog), friendlyAdult("Max", SpeciesDog), friendlyAdult("Bella", SpeciesDog)
	rex.Personality.Traits.Territoriality = 0.6 // Rex ranks highest

	result, err := ProcessGroupActivity(GroupPlay, rex, max, bella)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || len(result.Conflicts) != 0 || result.Harmony < goodMeeting {
		t.Fatalf("Friendly dogs should play happily: %+v", result)
	}
	if result.Leader != "Rex" || result.Order[0] != "Rex" || len(result.Order) != 3 {
		t.Errorf("Pecking order: %v", result.Order)
	}
	if !strings.HasPrefix(result.Text, "🎾 Rex, Max and Bella played together happily!") || !strings.Contains(result.Text, "Rex leads the pack.") {
		t.Errorf("Text = %q", result.Text)
	}
	if len(result.Signals) != 3 || !strings.Contains(strings.Join(result.Signals, " "), "play-bowed to") {
		t.Errorf("Dogs should signal friendliness: %v", result.Signals)
	}

	// Every pair now knows each other as friends, with the time recorded
	for _, pet := range []*DigitalPet{rex, max, bella} {
		relations := pet.Relations()
		if len(relations) != 2 {
			t.Fatalf("%s should know 2 pets: %+v", pet.Name, relations)
		}
		for _, r := range relations {
			if r.Kind != "friend" || r.LastShared != "played together" {
				t.Errorf("%s's relation: %+v", pet.Name, r)
			}
		}
	}
	if result.Happiness <= 0 || result.Happiness > 1 {
		t.Errorf("Group happiness = %.2f", result.Happiness)
	}
}

func TestGroupConflicts(t *testing.T) {
	// Two cats of similar rank that don't get along scuffle
	tom, felix := grumpyAdult("Tom", SpeciesCat), grumpyAdult("Felix", SpeciesCat)
	result, err := ProcessGroupActivity(GroupPlay, tom, felix)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].Outcome != "scuffle" ||
		!strings.Contains(result.Conflicts[0].Lesson, "Cats that clash need space") {
		t.Fatalf("Grumpy cats should scuffle: %+v", result.Conflicts)
	}
	if tom.Biology.Vitals.Stress < 0.15 || felix.Biology.Vitals.Stress < 0.15 {
		t.Error("A scuffle should stress both cats")
	}
	if tom.RelationTo(felix.ID) == "" || tom.Relations()[0].Kind != "rival" {
		t.Errorf("A scuffle should make rivals: %+v", tom.Relations())
	}

	// A clearly lower-ranked pet backs down instead
	boss, timid := grumpyAdult("Boss", SpeciesCat), grumpyAdult("Timid", SpeciesCat)
	boss.Personality.Traits.Extraversion, boss.Personality.Traits.Neuroticism = 0.9, 0.3
	timid.Personality.Traits.Territoriality = 0.3
	timid.Biology.Processes.Age = 10 // Young, so lower in rank
	result, err = ProcessGroupActivity(GroupPlay, boss, timid)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].Outcome != "backed down" ||
		result.Conflicts[0].Text != "Boss hissed at Timid, and Timid backed down." {
		t.Errorf("The lower-ranked cat should back down: %+v", result.Conflicts)
	}
	if boss.Biology.Vitals.Stress >= timid.Biology.Vitals.Stress {
		t.Error("Backing down stresses the one that backed down")
	}
}

func TestConflict(t *testing.T) {
	a, b := grumpyAdult("Tom", SpeciesCat), grumpyAdult("Felix", SpeciesCat)
	result, err := handleConflict(a, b)
	if err != nil || result.Outcome != "scuffle" {
		t.Fatalf("HandleConflict = %+v, %v", result, err)
	}
	if relation := a.Relations(); len(relation) != 1 || relation[0].LastShared != "clashed" {
		t.Errorf("The clash should be recorded: %+v", relation)
	}
	if _, err := handleConflict(a, a); !errors.Is(err, ErrSamePet) {
		t.Errorf("A pet can't clash with itself: %v", err)
	}
}

func TestTerritorialCatsDislikeCrowds(t *testing.T) {
	a, b := socialInfo{name: "A", species: SpeciesCat}, socialInfo{name: "B", species: SpeciesCat}
	a.traits.Territoriality, b.traits.Territoriality = 0.9, 0.9
	for _, info := range []*socialInfo{&a, &b} {
		info.traits.Extraversion, info.traits.Agreeableness = 0.6, 0.6
	}
	play := groupActivities[GroupPlay]
	if pairHarmony(a, b, play, 4) >= pairHarmony(a, b, play, 2) {
		t.Error("Territorial cats should get on worse in a bigger group")
	}
}

func TestNappingTogetherNeedsCloseCompanions(t *testing.T) {
	a, b := friendlyAdult("A", SpeciesCat).socialInfo(), friendlyAdult("B", SpeciesCat).socialInfo()
	nap := groupActivities[GroupNap]
	strangers := pairHarmony(a, b, nap, 2)
	a.relations[b.id], b.relations[a.id] = 0.8, 0.8
	if close := pairHarmony(a, b, nap, 2); close <= strangers {
		t.Errorf("Close companions should nap together better: %.2f vs %.2f", close, strangers)
	}
}

func TestGroupActivityRules(t *testing.T) {
	cat, dog := friendlyAdult("Biscuit", SpeciesCat), friendlyAdult("Rex", SpeciesDog)
	if _, err := ProcessGroupActivity(GroupWalk, dog, cat); !errors.Is(err, ErrDogsOnlyWalk) {
		t.Errorf("Cats don't go for walks: %v", err)
	}
	if _, err := ProcessGroupActivity("dance", dog, cat); !errors.Is(err, ErrUnknownGroup) {
		t.Errorf("Unknown activity: %v", err)
	}
	if _, err := ProcessGroupActivity(GroupPlay, dog); !errors.Is(err, ErrGroupSize) {
		t.Errorf("One pet is not a group: %v", err)
	}
	if _, err := ProcessGroupActivity(GroupPlay, dog, dog); !errors.Is(err, ErrGroupSize) {
		t.Errorf("A pet can't join twice: %v", err)
	}
	tired := friendlyAdult("Sleepy", SpeciesDog)
	tired.Biology.Vitals.Energy = 0.1
	var refused RefusedError
	if _, err := ProcessGroupActivity(GroupPlay, dog, tired); !errors.As(err, &refused) || !strings.Contains(err.Error(), "Sleepy is too tired") {
		t.Errorf("A tired pet can't join: %v", err)
	}
}

func TestGroupWalkExercisesDogs(t *testing.T) {
	rex, max := friendlyAdult("Rex", SpeciesDog), friendlyAdult("Max", SpeciesDog)
	for _, pet := range []*DigitalPet{rex, max} {
		pet.Biology.Physiology.CardiovascularHealth = 0.5
	}
	result, err := ProcessGroupActivity(GroupWalk, rex, max)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.Text, "🦮 Rex and Max went for a walk together") {
		t.Errorf("Text = %q", result.Text)
	}
	if max.Biology.Physiology.CardiovascularHealth <= 0.5 {
		t.Error("A walk should exercise every dog")
	}
	if rex.Memory.GetLearnedAffinity(types.InteractionSocialIntroduction) == 0 {
		t.Error("The active pet should take the activity as care")
	}
}

func TestCalculateGroupHappiness(t *testing.T) {
	a, b := friendlyAdult("A", SpeciesDog), friendlyAdult("B", SpeciesDog)
	a.Biology.Vitals.Happiness, b.Biology.Vitals.Happiness = 0.8, 0.6
	if got := CalculateGroupHappiness(a, b); got < 0.69 || got > 0.71 {
		t.Errorf("Strangers: the average happiness, got %.2f", got)
	}
	a.mu.Lock()
	rel := a.Relationships.AddRelationship(b.ID, types.RelationshipRival)
	rel.BondStrength, rel.Rivalry = 0.1, 0.9
	a.mu.Unlock()
	if got := CalculateGroupHappiness(a, b); got >= 0.7 {
		t.Errorf("Rivals should make a less happy group, got %.2f", got)
	}
}

func TestDominance(t *testing.T) {
	bold, shy := socialInfo{}, socialInfo{}
	bold.traits.Extraversion, bold.traits.Territoriality = 0.9, 0.9
	shy.traits.Extraversion, shy.traits.Neuroticism, shy.traits.Agreeableness = 0.1, 0.9, 0.9
	if dominance(bold) <= dominance(shy) {
		t.Error("A bold pet should outrank a shy one")
	}
	young := bold
	young.young = true
	if dominance(young) >= dominance(bold) {
		t.Error("The young rank lower")
	}
}

// handleConflict settles a clash between two pets: the lower-ranked one
// backs down if their ranks are clearly apart, otherwise they scuffle and
// both are stressed. Both relationships record it.
func handleConflict(a, b *DigitalPet) (ConflictResult, error) {
	if a == b {
		return ConflictResult{}, ErrSamePet
	}
	infoA, infoB := a.socialInfo(), b.socialInfo()
	result, bond, loser := conflict(infoA, infoB, 0)
	for i, side := range []struct {
		pet   *DigitalPet
		other socialInfo
	}{{a, infoB}, {b, infoA}} {
		side.pet.mu.Lock()
		side.pet.conflictLocked(loser == neitherBackedDown, loser == i)
		side.pet.recordMeetingLocked(side.other, bond, "clashed")
		side.pet.mu.Unlock()
	}
	return result, nil
}
