package main

import (
	"runtime"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Performance targets (SDD section 6.1): the server stays well under
// 500 MB, its memory and a pet's save stop growing, and a tick is quick
const (
	memoryTarget    = 500 << 20  // Bytes
	maxMemoryGrowth = 20 << 20   // Between one and twelve hours of play
	maxSaveSize     = 1536 << 10 // A pet's save, however long it is played
	maxTickTime     = 20 * time.Millisecond

	// core's limit on a pet's long-term memories
	maxLongTermMemoriesForTest = 1000
)

// heapInUse returns the live heap after a garbage collection
func heapInUse() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// TestLongPlayStaysWithinTargets plays twelve hours at the default pace
// (one tick a second, 720 game days), with care every half minute, and
// checks memory, save size and tick time. Twelve hours is long enough for
// everything a pet keeps to reach its limits.
func TestLongPlayStaysWithinTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("long run")
	}
	s := newTestServer(t)
	s.awayTimeout = 0
	if err := s.createPet("Biscuit", core.SpeciesDog, false, true); err != nil {
		t.Fatal(err)
	}
	cares := []types.InteractionType{types.InteractionFeeding, types.InteractionDrinking, types.InteractionWalking,
		types.InteractionPlaying, types.InteractionCleaning, types.InteractionGrooming, types.InteractionPetting}
	clock := time.Now()
	var tickTime time.Duration
	play := func(seconds int) {
		for i := 0; i < seconds; i++ {
			clock = clock.Add(time.Second)
			start := time.Now()
			s.tick(clock)
			tickTime += time.Since(start)
			if i%30 == 0 {
				s.mu.Lock()
				s.markActiveLocked(clock)
				if s.pet.IsHibernating() {
					s.pet.Revive()
				}
				v := s.pet.Biology.Vitals
				v.Nutrition, v.Hydration = max(v.Nutrition, 0.5), max(v.Hydration, 0.5)
				s.pet.Interact(cares[(i/30)%len(cares)], 1.0)
				s.mu.Unlock()
			}
		}
	}

	play(3600)
	afterHour := heapInUse()
	play(11 * 3600)
	afterTwelve := heapInUse()
	ticks := 12 * 3600

	if afterTwelve > memoryTarget {
		t.Errorf("Heap after twelve hours: %d MB, target under %d MB", afterTwelve>>20, memoryTarget>>20)
	}
	if afterTwelve > afterHour && afterTwelve-afterHour > maxMemoryGrowth {
		t.Errorf("Heap grew by %d MB from one to twelve hours of play", (afterTwelve-afterHour)>>20)
	}
	data, err := s.pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > maxSaveSize {
		t.Errorf("Save after twelve hours: %d KB", len(data)>>10)
	}
	_, long := s.pet.Memory.GetMemoryCount()
	if long > maxLongTermMemoriesForTest {
		t.Errorf("%d long-term memories, limit %d", long, maxLongTermMemoriesForTest)
	}
	if mean := tickTime / time.Duration(ticks); mean > maxTickTime {
		t.Errorf("Mean tick %v, target under %v", mean, maxTickTime)
	}
	if !s.pet.IsAlive() {
		t.Error("The pet should have lived through the run")
	}
	t.Logf("Heap %d KB after one hour, %d KB after twelve; save %d KB with %d long-term memories; mean tick %v",
		afterHour>>10, afterTwelve>>10, len(data)>>10, long, tickTime/time.Duration(ticks))
}
