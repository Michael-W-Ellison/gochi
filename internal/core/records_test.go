package core

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestTally(t *testing.T) {
	pet := NewDigitalPet("Mochi", "owner")
	applied := 0
	for i := 0; i < 3; i++ {
		if pet.Interact(types.InteractionPetting, 1).Applied {
			applied++
		}
	}
	pet.HibernationEnabled = true
	pet.Update(10000) // Hibernates
	tally := pet.Tally()
	if applied == 0 || tally.Care["Petting"] != applied || tally.Interactions != applied || !tally.Alive || tally.CauseOfDeath != "" {
		t.Errorf("Tally = %+v, want %d petting", tally, applied)
	}
	if tally.Hibernations != 1 || tally.Generation != 1 || tally.Species != SpeciesCat {
		t.Errorf("Tally = %+v", tally)
	}

	// The tally is a copy
	tally.Care["Petting"] = 100
	if pet.CareCounts["Petting"] != applied {
		t.Error("Changing a tally should not change the pet")
	}

	// Counts are kept in the save
	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Tally(); got.Care["Petting"] != applied || got.Hibernations != 1 {
		t.Errorf("Loaded tally = %+v", got)
	}

	pet.HibernationEnabled, pet.Hibernating = false, false
	pet.Update(10000) // Dies
	if tally := pet.Tally(); tally.Alive || tally.CauseOfDeath == "" {
		t.Errorf("A dead pet's tally should say how it died: %+v", tally)
	}
}
