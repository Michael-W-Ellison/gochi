package main

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// ticksPerPet is how many one-second ticks a benchmark runs on one pet
// (ten game days at the default pace) before starting a new one, so the
// pet never neglects its way into hibernation
const ticksPerPet = 600

func BenchmarkTick(b *testing.B) {
	s := newTestServer(b)
	s.awayTimeout = 0
	now := time.Now()
	for i := 0; i < b.N; i++ {
		if i%ticksPerPet == 0 {
			b.StopTimer()
			s.pet = core.NewDigitalPet("BenchPet", "bench")
			s.tick(now) // Start the clock without advancing the new pet
			b.StartTimer()
		}
		now = now.Add(time.Second)
		s.tick(now)
	}
}

func BenchmarkSaveActive(b *testing.B) {
	s := newTestServer(b)
	s.pet = core.NewDigitalPet("BenchPet", "bench")
	s.slot = "bench"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.saveNow(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadSlot(b *testing.B) {
	s := newTestServer(b)
	s.pet = core.NewDigitalPet("BenchPet", "bench")
	s.slot = "bench"
	if _, err := s.saveNow(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, _, err := s.store.Load("bench"); err != nil {
			b.Fatal(err)
		}
	}
}
