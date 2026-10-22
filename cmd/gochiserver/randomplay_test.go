package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/core"
	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// Random play: a player who does anything the page offers, in any order,
// with time passing in between. Most requests use the choices the game
// offers at that moment (its places, tricks, shows, items and saved
// pets); some use choices that don't exist. After every step the game
// must hold together: no server errors, the active pet's numbers finite
// and in range, its save loading back the same, the household's records
// matching the pets, and a book for every saved pet. A failure names the
// seed and the steps that led to it, so it can be replayed.

// options are the choices the game offers now, read from the routes the
// page reads them from
type options struct {
	foods, toys, regions, methods, changes []string
	tricks, learned, shows, places, games  []string
	items, cosmetics, supplies, looks      []string
	upgrades, holidays, slots, souvenirs   []string
	cosmeticSlot, lookSlot                 map[string]string
	choices                                []string // Of the happening waiting, if any
	activeName, activeSlot                 string
	need                                   string // The care the active pet needs most

	// What the player has: so most requests are ones the game can carry out
	ownedCosmetics, ownedLooks, ownedSupplies []string
	openPlaces, carried, collected, books     []string
}

// owned collects the "id" of each object in a list whose "owned" is set
func owned(list interface{}) []string {
	var out []string
	items, _ := list.([]interface{})
	for _, it := range items {
		if m, ok := it.(map[string]interface{}); ok && m["owned"] == true {
			if id, ok := m["id"].(string); ok {
				out = append(out, id)
			}
		}
	}
	return out
}

// ids collects the "id" of each object in a list
func ids(list interface{}) []string {
	var out []string
	items, _ := list.([]interface{})
	for _, it := range items {
		if m, ok := it.(map[string]interface{}); ok {
			if id, ok := m["id"].(string); ok {
				out = append(out, id)
			}
		}
	}
	return out
}

// decode reads a route's JSON reply into v
func (h *harness) decode(path string, v interface{}) {
	h.t.Helper()
	body := h.do(http.MethodGet, path, "").Body.Bytes()
	if err := json.Unmarshal(body, v); err != nil {
		h.t.Fatalf("GET %s: %v: %s", path, err, body)
	}
}

// get returns a route's data as a map ({} if it isn't one)
func (h *harness) get(path string) map[string]interface{} {
	h.t.Helper()
	var resp struct {
		Data interface{} `json:"data"`
	}
	h.decode(path, &resp)
	m, _ := resp.Data.(map[string]interface{})
	if m == nil {
		m = map[string]interface{}{}
	}
	return m
}

// readOptions reads what the game offers now
func (h *harness) readOptions() options {
	var o options
	care := h.get("/api/care/options")
	o.foods, o.toys, o.regions = ids(care["foods"]), ids(care["toys"]), ids(care["regions"])
	o.methods, o.changes = ids(care["methods"]), ids(care["changes"])
	shows := h.get("/api/shows")
	o.tricks, o.shows = ids(shows["tricks"]), ids(shows["shows"])
	if list, ok := shows["tricks"].([]interface{}); ok {
		for _, t := range list {
			if m, _ := t.(map[string]interface{}); m != nil && m["learned"] == true {
				o.learned = append(o.learned, m["id"].(string))
			}
		}
	}
	shop := h.get("/api/shop")
	o.items, o.cosmetics = ids(shop["items"]), ids(shop["cosmetics"])
	o.ownedCosmetics = owned(shop["cosmetics"])
	o.cosmeticSlot = map[string]string{}
	if list, ok := shop["cosmetics"].([]interface{}); ok {
		for _, c := range list {
			if m, _ := c.(map[string]interface{}); m != nil {
				o.cosmeticSlot[m["id"].(string)], _ = m["slot"].(string)
			}
		}
	}
	supplies := h.get("/api/supplies")
	o.supplies, o.ownedSupplies = ids(supplies["supplies"]), owned(supplies["supplies"])
	home := h.get("/api/home")
	o.looks, o.upgrades, o.ownedLooks = ids(home["looks"]), ids(home["upgrades"]), owned(home["looks"])
	collection := h.get("/api/collection")
	if list, ok := collection["places"].([]interface{}); ok {
		for _, set := range list {
			if m, _ := set.(map[string]interface{}); m != nil {
				for _, id := range m["found"].([]interface{}) {
					o.collected = append(o.collected, id.(string))
				}
			}
		}
	}
	var books struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	h.decode("/api/books", &books)
	for _, b := range books.Data {
		o.books = append(o.books, b.ID)
	}
	o.lookSlot = map[string]string{}
	if list, ok := home["looks"].([]interface{}); ok {
		for _, l := range list {
			if m, _ := l.(map[string]interface{}); m != nil {
				o.lookSlot[m["id"].(string)], _ = m["slot"].(string)
			}
		}
	}
	o.holidays = ids(h.get("/api/holidays")["holidays"])
	var places struct {
		Data struct {
			Places []core.Place `json:"places"`
		} `json:"data"`
	}
	h.decode("/api/explore/places", &places)
	for _, p := range places.Data.Places {
		o.places = append(o.places, p.ID)
		if p.Discovered && p.Unavailable == "" {
			o.openPlaces = append(o.openPlaces, p.ID)
		}
	}
	var games struct {
		Data []core.MiniGame `json:"data"`
	}
	h.decode("/api/games", &games)
	for _, g := range games.Data {
		o.games = append(o.games, g.ID)
	}
	for _, sets := range core.SouvenirSets() {
		for _, s := range sets {
			o.souvenirs = append(o.souvenirs, s.ID)
		}
	}
	list, _ := h.server.store.List()
	// Slot names come from pets' random IDs: order them by the pets instead
	sort.Slice(list, func(i, j int) bool {
		if list[i].Name != list[j].Name {
			return list[i].Name < list[j].Name
		}
		return list[i].Age < list[j].Age
	})
	for _, p := range list {
		o.slots = append(o.slots, p.Slot)
	}
	// In a fixed order, so the same seed picks the same
	for _, list := range []*[]string{&o.foods, &o.toys, &o.regions, &o.methods, &o.changes, &o.tricks, &o.learned,
		&o.shows, &o.places, &o.games, &o.items, &o.cosmetics, &o.supplies, &o.looks, &o.upgrades, &o.holidays,
		&o.souvenirs, &o.ownedCosmetics, &o.ownedLooks, &o.ownedSupplies, &o.openPlaces, &o.collected} {
		sort.Strings(*list)
	}
	h.server.mu.RLock()
	if pet := h.server.pet; pet != nil {
		o.activeName, o.activeSlot = pet.Name, h.server.slot
		if hp := pet.CurrentHappening(); hp != nil {
			for _, c := range hp.Choices {
				o.choices = append(o.choices, c.ID)
			}
		}
		o.need = careNeeded(pet)
		o.carried = append([]string(nil), pet.Souvenirs...)
	}
	h.server.mu.RUnlock()
	return o
}

// player makes random choices
type player struct {
	r *rand.Rand
}

// pick returns one of list, or now and then something that isn't on it
func (p player) pick(list []string) string {
	if len(list) == 0 || p.r.Intn(12) == 0 {
		return []string{"", "nothing", "../x"}[p.r.Intn(3)]
	}
	return list[p.r.Intn(len(list))]
}

// some returns up to n of list
func (p player) some(list []string, n int) []string {
	var out []string
	for i := p.r.Intn(n + 1); i > 0; i-- {
		out = append(out, p.pick(list))
	}
	return out
}

func (p player) b() bool { return p.r.Intn(2) == 0 }

func js(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// action is something the player can do, with how often they do it
type action struct {
	weight int
	do     func(p player, o options) (string, string) // The route and its body
}

// careNeeded is the care a pet needs most: what a caring player does
func careNeeded(pet *core.DigitalPet) string {
	v := pet.Biology.Vitals
	switch {
	case pet.Injured || pet.Illness != nil || v.Health < 0.5:
		return "medic"
	case v.Hydration < 0.6:
		return "drink"
	case v.Nutrition < 0.6:
		return "feed"
	case pet.LivingAreaWaste > 0.4:
		return "clean"
	case v.Cleanliness < 0.5:
		return "groom"
	case v.Happiness < 0.6:
		return "play"
	}
	return "pet"
}

var playerActions = []action{
	{40, func(p player, o options) (string, string) { // A caring player
		return "/api/interact", js(map[string]string{"action": o.need})
	}},
	{30, func(p player, o options) (string, string) {
		act := p.pick([]string{"feed", "drink", "pet", "play", "groom", "treat", "medic", "clean", "walk"})
		return "/api/interact", js(map[string]string{"action": act, "food": p.pick(o.foods), "toy": p.pick(o.toys),
			"region": p.pick(o.regions)})
	}},
	{6, func(p player, o options) (string, string) {
		return "/api/train", js(map[string]string{"trick": p.pick(o.tricks), "method": p.pick(o.methods)})
	}},
	{3, func(p player, o options) (string, string) {
		return "/api/perform", js(map[string]string{"trick": p.pick(o.learned)})
	}},
	{6, func(p player, o options) (string, string) {
		place := p.pick(o.openPlaces)
		if p.r.Intn(5) == 0 {
			place = p.pick(o.places) // Somewhere it may not be able to go
		}
		body := map[string]interface{}{"place": place, "long": p.r.Intn(4) == 0}
		if p.r.Intn(3) == 0 {
			body["friend_slot"] = p.pick(o.slots)
		}
		return "/api/explore", js(body)
	}},
	{1, func(p player, o options) (string, string) { return "/api/explore/recall", `{}` }},
	{2, func(p player, o options) (string, string) { return "/api/hideseek/start", `{}` }},
	{2, func(p player, o options) (string, string) {
		return "/api/hideseek/guess", js(map[string]int{"spot": p.r.Intn(8) - 1})
	}},
	{1, func(p player, o options) (string, string) { return "/api/hideseek/quit", `{}` }},
	{4, func(p player, o options) (string, string) {
		return "/api/games/finish", js(map[string]interface{}{"game": p.pick(o.games), "quality": p.r.Float64()*1.4 - 0.2})
	}},
	{5, func(p player, o options) (string, string) {
		return "/api/happening", js(map[string]string{"choice": p.pick(append(o.choices, ""))})
	}},
	{3, func(p player, o options) (string, string) {
		return "/api/shows/enter", js(map[string]interface{}{"show": p.pick(o.shows), "level": 1 + p.r.Intn(3),
			"routine": p.some(o.learned, 3)})
	}},
	{3, func(p player, o options) (string, string) {
		return "/api/shop/buy", js(map[string]string{"item": p.pick(o.items)})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/cosmetics/buy", js(map[string]string{"id": p.pick(o.cosmetics)})
	}},
	{2, func(p player, o options) (string, string) {
		id := p.pick(o.ownedCosmetics)
		return "/api/wear", js(map[string]string{"id": id, "slot": o.cosmeticSlot[id]})
	}},
	{3, func(p player, o options) (string, string) {
		return "/api/supplies/buy", js(map[string]string{"id": p.pick(o.supplies)})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/care/give", js(map[string]string{"id": p.pick(append(o.ownedSupplies, "first_aid"))})
	}},
	{3, func(p player, o options) (string, string) {
		return "/api/enrich", js(map[string]string{"change": p.pick(o.changes)})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/home/buy", js(map[string]string{"id": p.pick(o.upgrades)})
	}},
	{2, func(p player, o options) (string, string) {
		id := p.pick(o.ownedLooks)
		return "/api/home/look", js(map[string]string{"id": id, "slot": o.lookSlot[id]})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/home/shelf", js(map[string][]string{"items": p.some(o.collected, 5)})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/souvenirs/gift", js(map[string]string{"souvenir": p.pick(o.carried), "to_slot": p.pick(o.slots)})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/introduce", js(map[string]string{"partner_slot": p.pick(o.slots)})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/group", js(map[string]interface{}{"activity": p.pick([]string{"play", "walk", "nap"}),
			"partner_slots": p.some(o.slots, 3)})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/breed", js(map[string]string{"partner_slot": p.pick(o.slots), "name": "Kit"})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/family/transfer", js(map[string]interface{}{"child_slot": p.pick(o.slots), "amount": p.r.Intn(200) - 20})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/family/pass_on", js(map[string]interface{}{"child_slot": p.pick(o.slots), "points": p.r.Intn(200)})
	}},
	{2, func(p player, o options) (string, string) {
		return "/api/load", js(map[string]string{"slot": p.pick(o.slots)})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/create", js(map[string]interface{}{"name": p.pick([]string{"Pip", "Rex", "Luna", "Kit"}),
			"species": p.pick([]string{"cat", "dog"}), "breed": p.pick([]string{"", "random"}), "hibernation": p.b()})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/saves/pause", js(map[string]interface{}{"slot": p.pick(o.slots), "paused": p.b()})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/pause", js(map[string]bool{"paused": p.r.Intn(4) == 0})
	}},
	{1, func(p player, o options) (string, string) { return "/api/save", `{}` }},
	{1, func(p player, o options) (string, string) { return "/api/revive", `{}` }},
	{1, func(p player, o options) (string, string) { return "/api/potion", `{}` }},
	{1, func(p player, o options) (string, string) {
		return "/api/hibernation", js(map[string]bool{"enabled": p.b()})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/tips/seen", js(map[string]string{"id": p.pick(tipIDs)})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/settings", js(map[string]string{"hemisphere": p.pick([]string{"northern", "southern"})})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/privacy", js(map[string]interface{}{"sound_off": p.b(), "holidays_off": p.some(o.holidays, 3),
			"all_holidays_off": p.r.Intn(5) == 0})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/books/candle", js(map[string]interface{}{"id": p.pick(o.books), "lit": p.b()})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/popup/dismiss", js(map[string]string{"id": p.pick([]string{"breeds_2", "died_x"})})
	}},
	{1, func(p player, o options) (string, string) {
		return "/api/rename", js(map[string]string{"name": p.pick([]string{"Mochi", "Biscuit", "Pepper"})})
	}},
	{1, func(p player, o options) (string, string) {
		// Deleting a pet now and then, confirming with its name
		if p.r.Intn(4) != 0 || len(o.slots) < 3 {
			return "/api/status", ``
		}
		slot := p.pick(o.slots)
		name := o.activeName
		if slot != o.activeSlot {
			name = slot
		}
		return "/api/delete", js(map[string]interface{}{"slot": slot, "confirm_name": name, "delete_book": p.b()})
	}},
}

// randomPlay plays steps random steps with seed, checking the game after
// each
func randomPlay(t *testing.T, seed int64, steps int) []string {
	r := rand.New(rand.NewSource(seed))
	p := player{r}
	// The game's own chances and clock follow the seed and the play, so a
	// seed plays the same way each time
	//lint:ignore SA1019 The game uses the shared generator, so a seed replays only if it is seeded
	rand.Seed(seed) //nolint:staticcheck // The game uses the shared generator
	h := newHarness(t)
	h.now = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC).AddDate(0, 0, int(seed)*41) // Seeds start in different seasons
	core.UseClock(func() time.Time { return h.now })
	t.Cleanup(func() { core.UseClock(nil) })
	var replies []string
	h.createPet("Biscuit", core.SpeciesDog)
	h.createPet("Mochi", core.SpeciesCat)
	h.createPet("Pip", core.SpeciesCat)
	if seed%2 == 0 { // An experienced household: grown pets with points and tricks
		var pets []*core.DigitalPet
		var petSlots []string
		list, _ := h.server.store.List()
		for _, e := range list {
			pet, _, _, err := h.server.store.Load(e.Slot)
			if err != nil {
				t.Fatal(err)
			}
			if e.Slot == h.server.slot {
				pet = h.activePet()
			}
			pet.Biology.Processes.Age = 150 + float64(r.Intn(400))
			pet.FriendshipPoints = 4000
			pet.TricksLearned = map[string]bool{"sit": true, "come": true, "touch": true}
			pet.TrickSkills = map[string]float64{"sit": 1, "come": 1, "touch": 1}
			pet.TripsTaken = r.Intn(13)
			pet.Souvenirs = []string{"pebble", "shell", "acorn"}
			pets = append(pets, pet)
			petSlots = append(petSlots, e.Slot)
		}
		// Everyone knows everyone, closely
		for _, a := range pets {
			for _, b := range pets {
				if a != b {
					a.Relationships.AddRelationship(b.ID, types.RelationshipFriend).BondStrength = 0.7
					if a.KnownPets == nil {
						a.KnownPets = map[types.PetID]string{}
					}
					a.KnownPets[b.ID] = b.Name
				}
			}
		}
		for i, pet := range pets {
			if petSlots[i] != h.server.slot {
				if err := h.server.store.SavePet(petSlots[i], pet); err != nil {
					t.Fatal(err)
				}
			}
		}
		// A young child of two of them, and Home Stars to spend
		h.server.mu.Lock()
		h.server.collectLocked(nil, []string{"pebble", "shell", "acorn"})
		h.server.house.earn(2000, "test", false)
		h.server.mu.Unlock()
		for i, a := range pets {
			for j, b := range pets {
				if i < j && a.GetSpecies() == b.GetSpecies() {
					for try := 0; try < 50; try++ {
						if child, err := core.Breed(a, b, "Kit"); err == nil {
							if err := h.server.store.SavePet("kit", child); err != nil {
								t.Fatal(err)
							}
							break
						}
					}
				}
			}
		}
		for i, pet := range pets {
			if petSlots[i] != h.server.slot {
				if err := h.server.store.SavePet(petSlots[i], pet); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	total := 0
	for _, a := range playerActions {
		total += a.weight
	}
	var log []string
	fail := func(format string, args ...interface{}) {
		t.Helper()
		t.Fatalf("Seed %d, step %d: %s\nThe last steps:\n%s", seed, len(log), fmt.Sprintf(format, args...),
			strings.Join(log[max(len(log)-15, 0):], "\n"))
	}
	var o options
	codes := map[string]int{}
	defer func() {
		if os.Getenv("GOCHI_PLAY_REPORT") != "" {
			pet := h.activePet()
			list, _ := h.server.store.List()
			t.Logf("Seed %d: replies %v; %d saved pets; active %v", seed, codes, len(list), petSummary(pet))
		}
	}()
	for step := 0; step < steps; step++ {
		if step%10 == 0 {
			o = h.readOptions()
		} else {
			o.choices = h.readOptions().choices
		}
		n := r.Intn(total)
		var a action
		for _, a = range playerActions {
			if n < a.weight {
				break
			}
			n -= a.weight
		}
		path, body := a.do(p, o)
		method := http.MethodPost
		if body == "" {
			method = http.MethodGet
		}
		rec := h.do(method, path, body)
		log = append(log, fmt.Sprintf("  %s %s -> %d %.120s", path, body, rec.Code, rec.Body.String()))
		codes[fmt.Sprintf("%s %d", path, rec.Code)]++
		replies = append(replies, fmt.Sprintf("%s %d", path, rec.Code))
		if rec.Code >= 500 || !json.Valid(rec.Body.Bytes()) {
			fail("%s %s: %d %s", path, body, rec.Code, rec.Body.String())
		}
		wait := time.Duration(1+r.Intn(20)) * time.Second
		if r.Intn(20) == 0 {
			wait = time.Duration(1+r.Intn(10)) * time.Minute // Now and then a long while, to see trips end and pets age
		}
		h.tick(wait)
		log = append(log, fmt.Sprintf("  (%s of play)", wait))
		context := fmt.Sprintf("seed %d step %d", seed, step)
		h.checkPet(context)
		h.checkHousehold(context)
		if problem := collectionProblem(h); problem != "" {
			fail("%s", problem)
		}
		if step%25 == 24 {
			checkBooks(t, h, context)
		}
	}
	return replies
}

// collectionProblem returns a souvenir a pet holds that isn't in the
// household's collection, described, or ""
func collectionProblem(h *harness) string {
	d := h.server.house.snapshot()
	h.server.mu.RLock()
	active := h.server.pet
	h.server.mu.RUnlock()
	pets := []*core.DigitalPet{active}
	list, _ := h.server.store.List()
	for _, e := range list {
		if pet, _, _, err := h.server.store.Load(e.Slot); err == nil {
			pets = append(pets, pet)
		}
	}
	for _, pet := range pets {
		if pet == nil {
			continue
		}
		for _, id := range pet.Souvenirs {
			if !contains(d.Collected, id) {
				return fmt.Sprintf("%s holds %s, which isn't in the household's collection", pet.Name, id)
			}
		}
	}
	return ""
}

// checkBooks saves, then checks that every saved pet has a book with all
// its pages
func checkBooks(t *testing.T, h *harness, context string) {
	t.Helper()
	if _, err := h.server.saveNow(); err != nil {
		t.Fatalf("%s: saving: %v", context, err)
	}
	list, _ := h.server.store.List()
	for _, e := range list {
		pet, _, _, err := h.server.store.Load(e.Slot)
		if err != nil {
			t.Fatalf("%s: loading %s: %v", context, e.Slot, err)
		}
		data, err := os.ReadFile(filepath.Join(h.server.store.books.dir, string(pet.ID)+".json"))
		if err != nil {
			t.Fatalf("%s: %s has no book: %v", context, pet.Name, err)
		}
		var book petBook
		if err := json.Unmarshal(data, &book); err != nil || len(book.Pages) != len(pet.BookPages()) {
			t.Fatalf("%s: %s's book has %d pages, the pet %d", context, pet.Name, len(book.Pages), len(pet.BookPages()))
		}
	}
}

// TestRandomPlay plays several seeds. GOCHI_PLAY_SEEDS and GOCHI_PLAY_STEPS
// play more (or one seed: GOCHI_PLAY_SEED replays it)
func TestRandomPlay(t *testing.T) {
	first, seeds, steps := 1, 8, 300
	if testing.Short() {
		seeds, steps = 2, 80
	}
	if n, err := strconv.Atoi(os.Getenv("GOCHI_PLAY_SEEDS")); err == nil {
		seeds = n
	}
	if n, err := strconv.Atoi(os.Getenv("GOCHI_PLAY_STEPS")); err == nil {
		steps = n
	}
	if n, err := strconv.Atoi(os.Getenv("GOCHI_PLAY_SEED")); err == nil {
		first, seeds = n, 1
	}
	for seed := int64(first); seed < int64(first+seeds); seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			randomPlay(t, seed, steps)
		})
	}
}

// TestRandomPlayIsRepeatable plays a seed twice: the same seed must play
// the same way, so a failure can be replayed to find its cause
func TestRandomPlayIsRepeatable(t *testing.T) {
	var first, second []string
	t.Run("first", func(t *testing.T) { first = randomPlay(t, 5, 60) })
	t.Run("second", func(t *testing.T) { second = randomPlay(t, 5, 60) })
	for i := range first {
		if i >= len(second) || first[i] != second[i] {
			t.Fatalf("The plays differ at step %d: %q then %q", i, first[i], second[min(i, len(second)-1)])
		}
	}
}

// TestConcurrentPlay sends random requests from several players at once
// while time passes, as several browser tabs and the update loop do. Run
// with -race, it finds unsafe sharing; a deadlock fails it by timing out.
func TestConcurrentPlay(t *testing.T) {
	h := newHarness(t)
	h.createPet("Biscuit", core.SpeciesDog)
	h.createPet("Mochi", core.SpeciesCat)
	o := h.readOptions()
	total := 0
	for _, a := range playerActions {
		total += a.weight
	}
	requests := 150
	if testing.Short() {
		requests = 40
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failures []string
	done := make(chan struct{})
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			p := player{r}
			for n := 0; n < requests; n++ {
				k := r.Intn(total)
				var a action
				for _, a = range playerActions {
					if k < a.weight {
						break
					}
					k -= a.weight
				}
				path, body := a.do(p, o)
				method := http.MethodPost
				if body == "" {
					method = http.MethodGet
				}
				req := newRequest(method, path, body)
				rec := httptest.NewRecorder()
				func() {
					defer func() {
						if r := recover(); r != nil {
							mu.Lock()
							failures = append(failures, fmt.Sprintf("%s %s panicked: %v", path, body, r))
							mu.Unlock()
						}
					}()
					h.handler.ServeHTTP(rec, req)
				}()
				if rec.Code >= 500 {
					mu.Lock()
					failures = append(failures, fmt.Sprintf("%s %s: %d %s", path, body, rec.Code, rec.Body.String()))
					mu.Unlock()
				}
			}
		}(int64(i + 1))
	}
	ticking := make(chan struct{})
	go func() { // The update loop and autosave
		defer close(ticking)
		now := time.Now()
		for {
			select {
			case <-done:
				return
			default:
			}
			now = now.Add(time.Second)
			func() {
				defer func() {
					if r := recover(); r != nil {
						mu.Lock()
						failures = append(failures, fmt.Sprintf("a tick panicked: %v", r))
						mu.Unlock()
					}
				}()
				h.server.tick(now)
				h.server.shareTripWithFriend()
				h.server.householdTime(now)
				if now.Second()%20 == 0 {
					if _, err := h.server.saveNow(); err != nil {
						mu.Lock()
						failures = append(failures, fmt.Sprintf("a save failed: %v", err))
						mu.Unlock()
					}
				}
			}()
		}
	}()
	finished := make(chan struct{})
	go func() { wg.Wait(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(2 * time.Minute):
		t.Fatal("The players didn't finish in 2 minutes: a deadlock?")
	}
	close(done)
	<-ticking
	for _, f := range failures {
		t.Error(f)
	}
}

// petSummary describes a pet for the play report
func petSummary(p *core.DigitalPet) string {
	if p == nil {
		return "none"
	}
	return fmt.Sprintf("%s, %.0f days, alive %v, %d pages, %d souvenirs, %d trips, %d ribbons, %d points",
		p.Name, p.Biology.GetAgeInDays(), p.IsAlive(), len(p.BookPages()), len(p.Souvenirs), p.TripsTaken,
		len(p.Ribbons), p.FriendshipPoints)
}

// TestLongTripFindsAreCollected: random play found that a long trip's
// second find wasn't added to the household's collection
func TestLongTripFindsAreCollected(t *testing.T) {
	h := newHarness(t)
	h.createPet("Mochi", core.SpeciesCat)
	pet := h.activePet()
	h.server.mu.Lock()
	pet.LastAdventure = &core.Adventure{Place: "forest", Souvenir: &core.Souvenir{ID: "leaf"}, Extra: []string{"mushroom"}}
	h.server.starsForTripLocked()
	h.server.mu.Unlock()
	if d := h.server.house.snapshot(); !contains(d.Collected, "leaf") || !contains(d.Collected, "mushroom") {
		t.Errorf("Both finds should be collected: %v", d.Collected)
	}
}
