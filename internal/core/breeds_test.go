package core

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestBreedCatalog(t *testing.T) {
	cats, dogs := Breeds(SpeciesCat), Breeds(SpeciesDog)
	if len(cats) != 21 || len(dogs) != 22 {
		t.Errorf("%d cat and %d dog breeds; want 21 and 22", len(cats), len(dogs))
	}
	patterns := []string{PatternSolid, PatternBicolor, PatternTuxedo, PatternTabby, PatternTicked, PatternSpotted,
		PatternPoint, PatternMitted, PatternTortie, PatternCalico, PatternVan, PatternWhite, PatternMerle, PatternBrindle,
		PatternSaddle, PatternTricolor, PatternMask, PatternTan, PatternPatches}
	seen := map[string]bool{}
	for _, b := range Breeds("") {
		if seen[b.ID] {
			t.Errorf("%s is listed twice", b.ID)
		}
		seen[b.ID] = true
		if b.Tier < TierEasy || b.Tier > TierDemanding || b.Lifespan < 0.6 || b.Lifespan > 1.2 || b.About == "" ||
			b.Temperament == "" || b.Body.Size[0] > b.Body.Size[1] {
			t.Errorf("%s has a bad tier, lifespan, size or description", b.ID)
		}
		for _, c := range append(slices.Clone(b.Colors), b.Eyes...) {
			if _, ok := FindCoatColor(c); !ok {
				t.Errorf("%s has an unknown color %s", b.ID, c)
			}
		}
		for _, p := range b.Patterns {
			if !slices.Contains(patterns, p) {
				t.Errorf("%s has an unknown pattern %s", b.ID, p)
			}
		}
		for _, id := range b.Tendencies {
			if _, ok := FindTendency(id); !ok {
				t.Errorf("%s has an unknown tendency %s", b.ID, id)
			}
		}
		for name := range b.traits {
			if traitSetters[name] == nil {
				t.Errorf("%s has an unknown trait %s", b.ID, name)
			}
		}
	}
	// Breeds whose looks come from a harmful condition are left out
	for _, id := range []string{"scottish_fold", "munchkin", "dachshund", "corgi", "pembroke_welsh_corgi"} {
		if _, ok := FindBreed(id); ok {
			t.Errorf("%s should not be a breed", id)
		}
	}
	// Each species has easy breeds to start with
	for _, list := range [][]BreedDef{cats, dogs} {
		easy := 0
		for _, b := range list {
			if b.Tier == TierEasy {
				easy++
			}
		}
		if easy < 4 {
			t.Errorf("Only %d easy %s breeds", easy, list[0].Species)
		}
	}
}

func TestDrawingPaletteMatches(t *testing.T) {
	art, err := os.ReadFile("../../web/static/js/pet-art.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range CoatColors() {
		if !strings.Contains(string(art), c.ID+": '"+c.Hex+"'") {
			t.Errorf("pet-art.js doesn't draw %s as %s", c.ID, c.Hex)
		}
	}
}

func TestPetOfBreed(t *testing.T) {
	b, _ := FindBreed("siamese")
	for i := 0; i < 20; i++ {
		pet := NewPetOfBreed("Mochi", "owner", SpeciesCat, "siamese")
		_, look := pet.GetLook()
		if pet.Breed != "siamese" || look.Pattern != PatternPoint || look.Eye != "eye_blue" ||
			look.Size < b.Body.Size[0] || look.Size > b.Body.Size[1] || pet.Tendencies[TendSeparation] != 1 {
			t.Fatalf("A Siamese: %s %+v %v", pet.Breed, look, pet.Tendencies)
		}
		if e := pet.Personality.Traits.Extraversion; e < 0.8-breedSpread-0.01 || e > 0.8+breedSpread+0.01 {
			t.Errorf("A Siamese's sociability %.2f should be near the breed's 0.8", e)
		}
	}
	// A breed of another species, or an unknown one, is the default
	if pet := NewPetOfBreed("Rex", "owner", SpeciesDog, "siamese"); pet.Breed != BreedMixed {
		t.Errorf("A dog asked to be a Siamese is a %s", pet.Breed)
	}

	// Giant breeds live shorter lives
	dane, lab := NewPetOfBreed("Dane", "o", SpeciesDog, "great_dane"), NewPetOfBreed("Lab", "o", SpeciesDog, "labrador")
	if ratio := dane.lifespanFactorLocked() / lab.lifespanFactorLocked(); ratio < 0.64 || ratio > 0.66 {
		t.Errorf("A Great Dane should live about 65%% as long as a Labrador, got %.2f", ratio)
	}
	dane.Update(0.01)
	if dane.Biology.SpeciesLifespan != dane.lifespanFactorLocked() {
		t.Error("The body uses the breed's lifespan")
	}
}

func TestTendenciesChangeTheSimulation(t *testing.T) {
	pug := NewPetOfBreed("Pug", "o", SpeciesDog, "pug")
	lab := NewPetOfBreed("Lab", "o", SpeciesDog, "labrador")
	for _, p := range []*DigitalPet{pug, lab} {
		p.ensureEnvironmentLocked()
		p.Environment.Weather.Current.Temperature = 33
	}
	if pug.effectLocked(effWeatherStress) <= lab.effectLocked(effWeatherStress) {
		t.Error("Short airways make hot weather more stressful")
	}
	if pug.effectLocked(effAirDamage) <= 1 || lab.effectLocked(effAirDamage) != 1 {
		t.Error("Foul air harms short airways faster")
	}
	if lab.effectLocked(effTripInjury) != 2 || lab.effectLocked(effHunger) <= 1 || lab.effectLocked(effTreatTraining) <= 1 {
		t.Error("A Labrador's joints and appetite")
	}

	persian := NewPetOfBreed("Persian", "o", SpeciesCat, "persian")
	if persian.effectLocked(effCleanliness) <= 1 {
		t.Error("A Persian's coat gets dirty faster")
	}

	// An energetic breed needs exercise sooner
	collie := NewPetOfBreed("Collie", "o", SpeciesDog, "border_collie")
	collie.Update(0.01)
	lab.Update(0.01)
	if collie.Needs.GetNeed(types.NeedExercise).DecayRate <= lab.Needs.GetNeed(types.NeedExercise).DecayRate {
		t.Error("A Border Collie's need for exercise should build faster")
	}
	pre := persian.predispositionsLocked()
	if pre.KidneyDisease != 1 || pre.Eyes != 1 || pre.KidneyAging != 1 {
		t.Errorf("A Persian's predispositions: %+v", pre)
	}
}

// adultOfBreed returns a healthy adult of a breed, ready to breed
func adultOfBreed(name, breed string, species Species) *DigitalPet {
	pet := NewPetOfBreed(name, "user123", species, breed)
	pet.Biology.Processes.Age = 100
	pet.Biology.Vitals.Health, pet.Biology.Vitals.Happiness = 1, 0.9
	return pet
}

func TestSameBreedHasThatBreed(t *testing.T) {
	a, b := adultOfBreed("A", "beagle", SpeciesDog), adultOfBreed("B", "beagle", SpeciesDog)
	child := breedUntilSuccess(t, a, b, "Pup")
	if child.Breed != "beagle" || child.Tendencies[TendFoodMotivated] != 1 || child.BreedLifespan != 1.05 {
		t.Errorf("A Beagle puppy: %s %v %.2f", child.Breed, child.Tendencies, child.BreedLifespan)
	}
	info := child.BreedInfo()
	if info.Lineage == nil || len(info.Lineage.Parents) != 2 || info.Lineage.Parents[0].BreedName != "Beagle" {
		t.Errorf("Lineage: %+v", info.Lineage)
	}
}

func TestDifferentBreedsHaveAMixedBreed(t *testing.T) {
	lab, golden := adultOfBreed("Lab", "labrador", SpeciesDog), adultOfBreed("Goldie", "golden_retriever", SpeciesDog)
	child := breedUntilSuccess(t, lab, golden, "Pup")
	if child.Breed != BreedMixed {
		t.Fatalf("A Labrador and Golden Retriever's puppy is a %s", child.Breed)
	}
	// Both have joints; only the Labrador is food-motivated and only the
	// Golden needs grooming
	if child.Tendencies[TendJoints] != 0.5 || child.Tendencies[TendFoodMotivated] != 0 || child.Tendencies[TendGrooming] != 0 {
		t.Errorf("A mixed breed inherits shared tendencies at half strength: %v", child.Tendencies)
	}
	if want := (1.0 + 0.95) / 2; child.BreedLifespan != want {
		t.Errorf("Lifespan %.3f, want %.3f", child.BreedLifespan, want)
	}
	info := child.BreedInfo()
	if len(info.Lineage.From) != 10 || info.Lineage.Parents[1].BreedName != "Golden Retriever" {
		t.Errorf("A mixed breed records where each feature came from: %+v", info.Lineage)
	}
	for feature, parent := range info.Lineage.From {
		if parent != "Lab" && parent != "Goldie" {
			t.Errorf("%s came from %s", feature, parent)
		}
	}

	// Its own puppies record its parents' breeds
	other := adultOfBreed("Rex", "beagle", SpeciesDog)
	child.Biology.Processes.Age, child.Biology.Vitals.Health, child.Biology.Vitals.Happiness = 100, 1, 0.9
	grandchild := breedUntilSuccess(t, child, other, "Pip")
	if gp := grandchild.Lineage.Parents[0].ParentBreeds; len(gp) != 2 || gp[0] != "Labrador Retriever" {
		t.Errorf("A grandparent's breeds: %v", gp)
	}

	// Two cats of different breeds have a domestic cat
	siamese, persian := adultOfBreed("Sia", "siamese", SpeciesCat), adultOfBreed("Per", "persian", SpeciesCat)
	kitten := breedUntilSuccess(t, siamese, persian, "Kit")
	if kitten.Breed != BreedDomesticShorthair && kitten.Breed != BreedDomesticLonghair {
		t.Errorf("A mixed kitten is a %s", kitten.Breed)
	}
}

func TestMerleToMerle(t *testing.T) {
	useRandom(t, 0)
	if got := inheritPattern(PatternMerle, PatternMerle); got != PatternDoubleMerl {
		t.Errorf("Two merles can have a double merle, got %q", got)
	}
	useRandom(t, 3)
	if got := inheritPattern(PatternMerle, PatternMerle); got != PatternMerle {
		t.Errorf("Or a merle, got %q", got)
	}
	useRandom(t, 0)
	if got := inheritPattern(PatternMerle, PatternTricolor); got != PatternMerle {
		t.Errorf("A merle and a tricolor have one of their patterns, got %q", got)
	}
}

func TestCongenitalDeafness(t *testing.T) {
	white := PetLook{Pattern: PatternWhite, Eye: "eye_blue", Eye2: "eye_blue"}
	useRandom(t, 0) // Every chance happens
	if got := congenitalHearing(white); got != 0.5 {
		t.Errorf("A deaf white cat with blue eyes, deaf in one ear: %.1f", got)
	}
	useRandom(t, 0, 999_999)
	if got := congenitalHearing(white); got != 0 {
		t.Errorf("Deaf in both ears: %.1f", got)
	}
	useRandom(t, 999_999) // No chance happens
	if got := congenitalHearing(white); got != 1 {
		t.Errorf("A hearing white cat: %.1f", got)
	}
	useRandom(t, 0)
	if got := congenitalHearing(PetLook{Pattern: PatternTabby, Eye: "eye_blue"}); got != 1 {
		t.Errorf("A tabby isn't at risk: %.1f", got)
	}
}

func TestPetsFromBeforeBreeds(t *testing.T) {
	pet := NewDigitalPet("Old", "owner")
	pet.Species = SpeciesDog
	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	// What an old save had: no breed, look or senses
	text := strings.Replace(string(data), `"senses_set":true`, `"senses_set":false`, 1)
	text = strings.Replace(text, `"hearing":1`, `"hearing":0`, 1)
	loaded, err := Load([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	breed, look := loaded.GetLook()
	if breed != BreedMixed || look.Color == "" {
		t.Errorf("An old dog becomes a mixed breed with its genes' colors: %s %+v", breed, look)
	}
	if ph := loaded.Biology.Physiology; ph.Hearing != 1 || ph.KidneyReserve != 1 || ph.DentalHealth != 1 {
		t.Errorf("An old pet's new body systems start healthy: %+v", ph)
	}

	// A save naming a breed of another species, or unknown tendencies
	pet = NewPetOfBreed("Mix", "owner", SpeciesCat, "persian")
	pet.Tendencies["made_up"] = 1
	pet.Tendencies[TendEyes] = 5
	pet.Breed = "labrador"
	data, _ = pet.Save()
	loaded, _ = Load(data)
	if breed, _ := loaded.GetLook(); breed != BreedDomesticShorthair || len(loaded.Tendencies) != 0 {
		t.Errorf("A cat saved as a Labrador becomes a domestic shorthair, with its tendencies: %s %v", breed, loaded.Tendencies)
	}
	pet = NewPetOfBreed("Per", "owner", SpeciesCat, "persian")
	pet.Tendencies["made_up"] = 1
	pet.Tendencies[TendEyes] = 5
	data, _ = pet.Save()
	loaded, _ = Load(data)
	if loaded.Tendencies["made_up"] != 0 || loaded.Tendencies[TendEyes] != 1 || loaded.Breed != "persian" {
		t.Errorf("Unknown tendencies are dropped and strengths kept to 1: %v", loaded.Tendencies)
	}
}

func TestKidneysEyesTeethAndHearing(t *testing.T) {
	persian := NewPetOfBreed("Persian", "o", SpeciesCat, "persian")
	persian.Biology.Processes.Age = 1000
	ph := persian.Biology.Physiology
	ph.KidneyReserve, ph.KidneyHealth = 0.8, 0.8
	persian.HibernationEnabled = true
	for i := 0; i < 200; i++ {
		persian.Update(1)
		setCalmHealthyState(persian)
	}
	if ph.KidneyReserve >= 0.8 {
		t.Error("Inherited kidney disease lowers the kidneys' reserve")
	}

	// A kidney-support diet is only for pets that need it
	healthy := wealthyPet(t, SpeciesCat)
	healthy.mu.Lock()
	if _, err := healthy.applyCareSupplyLocked("kidney_diet"); err == nil {
		t.Error("A healthy cat doesn't need a kidney diet")
	}
	healthy.mu.Unlock()
	persian.mu.Lock()
	if _, err := persian.applyCareSupplyLocked("kidney_diet"); err != nil {
		t.Errorf("A Persian with kidney disease: %v", err)
	}
	if persian.effectLocked(effKidneyDecline) != 0.5 {
		t.Error("The diet halves kidney decline for the day")
	}
	persian.mu.Unlock()

	// Dehydration harms the kidneys; fluids (Heal) restore what they can do
	ph.KidneyHealth = 0.4
	persian.mu.Lock()
	if !persian.giveFluidsLocked() || ph.KidneyHealth != ph.KidneyReserve {
		t.Error("Fluids bring the kidneys back to their reserve")
	}
	persian.mu.Unlock()

	// Plaque, brushing and dental disease
	dog := NewPetOfBreed("Chi", "o", SpeciesDog, "chihuahua")
	dog.mu.Lock()
	dog.Biology.Physiology.DentalHealth = 0.5
	if _, err := dog.applyCareSupplyLocked("pet_toothpaste"); err != nil || dog.Biology.Physiology.DentalHealth != 0.6 {
		t.Errorf("Brushing: %v %.2f", err, dog.Biology.Physiology.DentalHealth)
	}
	if _, err := dog.applyCareSupplyLocked("pet_toothpaste"); err == nil {
		t.Error("One brushing a day counts")
	}
	if dog.predispositionsLocked().Dental != 1.5 {
		t.Error("A Chihuahua's crowded teeth build plaque faster")
	}
	dog.Biology.Physiology.DentalHealth = 0.1
	if kind, rate := dog.senseIllnessLocked(1); kind != IllnessDental || rate <= 0 {
		t.Errorf("Neglected teeth lead to dental disease: %s %.2f", kind, rate)
	}
	dog.Illness = &Illness{Kind: IllnessDental, Severity: 0.3}
	dog.treatIllnessLocked()
	if dog.Biology.Physiology.DentalHealth < 0.9 || dog.IllnessesTreated != 1 {
		t.Error("The vet cleans the teeth")
	}
	signs := dog.senseSymptomsLocked(0)
	dog.Biology.Physiology.Hearing = 0
	deaf := dog.senseSymptomsLocked(0)
	dog.mu.Unlock()
	if len(deaf) != len(signs)+1 || deaf[len(deaf)-1].Topic != "hearing" {
		t.Errorf("A deaf pet doesn't come when called: %+v", deaf)
	}

	// Words and clicks don't reach a deaf pet; hand signals do
	dog.mu.Lock()
	clicker, _ := findMethod("clicker")
	signals, _ := findMethod("signals")
	deafClicker, deafSignals := dog.methodProgressLocked(clicker), dog.methodProgressLocked(signals)
	dog.Biology.Physiology.Hearing = 1
	hearingClicker := dog.methodProgressLocked(clicker)
	dog.mu.Unlock()
	if deafClicker >= hearingClicker || deafSignals <= deafClicker {
		t.Errorf("Clicker for a deaf pet %.2f (hearing %.2f), hand signals %.2f", deafClicker, hearingClicker, deafSignals)
	}

	// Eyes
	pug := NewPetOfBreed("Pug", "o", SpeciesDog, "pug")
	for i := 0; i < 50; i++ {
		pug.Update(1)
		setCalmHealthyState(pug)
	}
	if pug.Biology.Physiology.EyeHealth >= 1 {
		t.Error("A Pug's prominent eyes get sore")
	}
}

func TestCareAchievementsUnlockBreeds(t *testing.T) {
	if UnlockedTier(nil) != TierEasy || UnlockedTier([]string{"grown_up", "daily_carer"}) != TierModerate ||
		UnlockedTier([]string{"grown_up", "daily_carer", "nurse"}) != TierModerate ||
		UnlockedTier([]string{"grown_up", "daily_carer", "nurse", "attentive"}) != TierDemanding ||
		UnlockedTier([]string{"nurse", "attentive"}) != TierEasy {
		t.Error("Breed tiers unlock in order")
	}
	pet := NewDigitalPet("Mochi", "owner")
	pet.GoalDays, pet.IllnessesTreated = 3, 1
	pet.Biology.Processes.Age, pet.Biology.CareQuality = attentiveAge+1, 0.85
	earned := map[string]bool{}
	for _, a := range pet.CheckAchievements() {
		earned[a.ID] = true
	}
	for _, id := range []string{"daily_carer", "nurse", "attentive", "grown_up"} {
		if !earned[id] {
			t.Errorf("%s should be earned", id)
		}
	}
}
