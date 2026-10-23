package main

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// TestBrowserSmoke plays the page in a headless browser (test/e2e/smoke.js)
// against a game with a fresh data folder holding one saved pet. It needs node and Playwright, so
// it runs only with GOCHI_E2E=1 (make e2e); NODE_PATH tells node where
// Playwright is.
func TestBrowserSmoke(t *testing.T) {
	if os.Getenv("GOCHI_E2E") == "" {
		t.Skip("set GOCHI_E2E=1 to play the page in a browser")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("node isn't installed")
	}
	h := newHarness(t)

	// Pepper, saved groomed before and in need of it again (the grooming
	// game opens once a pet has been groomed, and a pet that turns grooming
	// down, as a new one is clean and one grooming makes it clean, won't
	// play it), and owning every cosmetic
	pepper := core.NewPetOfBreed("Pepper", localOwner, core.SpeciesCat, "")
	pepper.CareCounts = map[string]int{types.InteractionGrooming.String(): 1}
	pepper.Biology.Vitals.Cleanliness = 0.5
	for _, c := range core.CosmeticCatalog() { // Every cosmetic, to dress her and draw them all
		pepper.Cosmetics = append(pepper.Cosmetics, c.ID)
	}
	if err := h.server.store.SavePet(h.server.store.NewSlot(pepper), pepper); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(h.handler)
	defer server.Close()
	go h.server.updateLoop() // Time passes as in play
	script, _ := filepath.Abs(filepath.Join("..", "..", "test", "e2e", "smoke.js"))
	out, err := exec.Command(node, script, server.URL).CombinedOutput()
	t.Logf("%s", out)
	if err != nil {
		t.Fatalf("The page had problems: %v", err)
	}
}
