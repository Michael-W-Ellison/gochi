package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/internal/core"
)

// The wardrobe through the page's routes: a player buys an item for each
// slot, dresses the pet, changes and takes off items, and the outfit shows
// in the pet's status and the pet list, and is still worn when the pet is
// loaded again
func TestDressingAPetThroughThePage(t *testing.T) {
	outfit := map[string]string{core.SlotHead: "flower_crown", core.SlotEyes: "round_glasses",
		core.SlotNeck: "bow_tie", core.SlotCollar: "collar_bell", core.SlotTag: "tag_fish", core.SlotBody: "hoodie_blue"}
	for _, species := range []core.Species{core.SpeciesCat, core.SpeciesDog} {
		t.Run(string(species), func(t *testing.T) {
			h := newHarness(t)
			h.createPet("Mochi", species)
			h.createPet("Biscuit", species) // Mochi is saved; Biscuit is dressed
			h.server.mu.Lock()
			h.server.pet.FriendshipPoints = 2000
			h.server.mu.Unlock()

			for slot, id := range outfit {
				if rec := h.post("/api/wear", `{"slot":"`+slot+`","id":"`+id+`"}`); rec.Code != http.StatusConflict {
					t.Errorf("Wearing %s before buying it: %d %s", id, rec.Code, rec.Body.String())
				}
				if rec := h.post("/api/cosmetics/buy", `{"id":"`+id+`"}`); rec.Code != http.StatusOK {
					t.Fatalf("Buying %s: %d %s", id, rec.Code, rec.Body.String())
				}
				if rec := h.post("/api/wear", `{"slot":"`+slot+`","id":"`+id+`"}`); rec.Code != http.StatusOK {
					t.Fatalf("Wearing %s: %d %s", id, rec.Code, rec.Body.String())
				}
			}
			if rec := h.post("/api/wear", `{"slot":"`+core.SlotBody+`","id":"tag_fish"}`); rec.Code != http.StatusBadRequest {
				t.Errorf("A tag worn as a coat: %d %s", rec.Code, rec.Body.String())
			}

			worn := func() map[string]core.Cosmetic {
				var status struct {
					Data struct {
						Outfit map[string]core.Cosmetic `json:"outfit"`
					} `json:"data"`
				}
				if err := json.Unmarshal(h.do(http.MethodGet, "/api/status", "").Body.Bytes(), &status); err != nil {
					t.Fatal(err)
				}
				return status.Data.Outfit
			}
			got := worn()
			for slot, id := range outfit {
				if got[slot].ID != id {
					t.Errorf("The status should show %s in the %s slot: %v", id, slot, got)
				}
			}

			var shop struct {
				Data struct {
					Points    int                  `json:"friendship_points"`
					Cosmetics []core.CosmeticOffer `json:"cosmetics"`
				} `json:"data"`
			}
			if err := json.Unmarshal(h.do(http.MethodGet, "/api/shop", "").Body.Bytes(), &shop); err != nil {
				t.Fatal(err)
			}
			wornInShop := 0
			for _, c := range shop.Data.Cosmetics {
				if c.Worn {
					wornInShop++
				}
			}
			if wornInShop != 6 || shop.Data.Points != 2000-80-55-50-45-20-70 {
				t.Errorf("The wardrobe should show six worn, paid for: %d worn, %d points left", wornInShop, shop.Data.Points)
			}

			// Change one, take one off
			if rec := h.post("/api/cosmetics/buy", `{"id":"sun_hat"}`); rec.Code != http.StatusOK {
				t.Fatal(rec.Body.String())
			}
			h.post("/api/wear", `{"slot":"head","id":"sun_hat"}`)
			h.post("/api/wear", `{"slot":"eyes","id":""}`)
			got = worn()
			if got[core.SlotHead].ID != "sun_hat" || len(got) != 5 {
				t.Errorf("The sun hat should replace the crown, and the glasses come off: %v", got)
			}

			// The pet list shows it; it is still worn when Biscuit is loaded again
			var saves struct {
				Data struct {
					Pets []savedPet `json:"pets"`
				} `json:"data"`
			}
			if err := json.Unmarshal(h.do(http.MethodGet, "/api/saves", "").Body.Bytes(), &saves); err != nil {
				t.Fatal(err)
			}
			slot, mochi := "", ""
			for _, p := range saves.Data.Pets {
				switch p.Name {
				case "Biscuit":
					slot = p.Slot
					if !strings.Contains(p.Outfit, "👒") || !strings.Contains(p.Outfit, "🎀") {
						t.Errorf("The pet list should show Biscuit's outfit: %q", p.Outfit)
					}
				case "Mochi":
					mochi = p.Slot
				}
			}
			for _, load := range []string{mochi, slot} {
				if rec := h.post("/api/load", `{"slot":"`+load+`"}`); rec.Code != http.StatusOK {
					t.Fatalf("Loading %s: %s", load, rec.Body.String())
				}
			}
			if got := worn(); got[core.SlotHead].ID != "sun_hat" || len(got) != 5 {
				t.Errorf("Biscuit's outfit should be kept when loaded again: %v", got)
			}
		})
	}
}
