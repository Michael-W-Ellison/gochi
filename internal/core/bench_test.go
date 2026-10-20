package core

import (
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// gameDayPerTick is the time one server tick advances the pet at the
// game's default pace (one game day per real minute, one tick per second)
const gameDayPerTick = 1.0 / 60

// ticksPerPet is how many ticks a benchmark runs on one pet before
// starting a new one, so the pet never neglects its way into hibernation
const ticksPerPet = 600

func BenchmarkNewDigitalPet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewDigitalPet("BenchPet", "bench")
	}
}

func BenchmarkUpdate(b *testing.B) {
	pet := NewDigitalPet("BenchPet", "bench")
	for i := 0; i < b.N; i++ {
		if i%ticksPerPet == 0 {
			b.StopTimer()
			pet = NewDigitalPet("BenchPet", "bench")
			b.StartTimer()
		}
		pet.Update(gameDayPerTick)
	}
}

// benchCare is the care an aged pet is given
var benchCare = []types.InteractionType{
	types.InteractionFeeding, types.InteractionDrinking, types.InteractionPetting,
	types.InteractionPlaying, types.InteractionGrooming, types.InteractionCleaning,
}

// agedPet is a pet that has lived and been cared for a while, so its
// memories and learning have content to save
func agedPet(b *testing.B) *DigitalPet {
	b.Helper()
	pet := NewDigitalPet("BenchPet", "bench")
	for i := 0; i < 200; i++ {
		pet.Update(gameDayPerTick)
		if i%20 == 0 {
			pet.Interact(benchCare[(i/20)%len(benchCare)], 1.0)
		}
	}
	if !pet.IsAlive() {
		b.Fatal("the benchmark pet died while ageing")
	}
	return pet
}

func BenchmarkSave(b *testing.B) {
	pet := agedPet(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := pet.Save(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoad(b *testing.B) {
	data, err := agedPet(b).Save()
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Load(data); err != nil {
			b.Fatal(err)
		}
	}
}
