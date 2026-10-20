package hub

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// BenchmarkNearby finds nearby pets on a hub with 300 players, each with
// two pets with 64 KB saves, spread over 30 areas
func BenchmarkNearby(b *testing.B) {
	h, err := New(b.TempDir(), "secret")
	if err != nil {
		b.Fatal(err)
	}
	save := json.RawMessage(`"` + strings.Repeat("x", 64<<10) + `"`)
	for i := 0; i < 300; i++ {
		name := fmt.Sprintf("player%d", i)
		if _, err := h.Register(name, "secret"); err != nil {
			b.Fatal(err)
		}
		if _, err := h.SetProfile(name, fmt.Sprintf("Area %d", i%30), true, true); err != nil {
			b.Fatal(err)
		}
		for j := 0; j < 2; j++ {
			info := PetInfo{Key: PetKey(fmt.Sprintf("%s_%d", name, j)), Name: "Pet", Species: "cat", Alive: true}
			if _, err := h.PutPet(name, info, save, 0); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if pets, err := h.Nearby("player0"); err != nil || len(pets) != 18 {
			b.Fatalf("Nearby: %d %v", len(pets), err)
		}
	}
}
