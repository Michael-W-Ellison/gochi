package core

import (
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func newDog(name string) *DigitalPet {
	pet := NewDigitalPet(name, "user123")
	pet.Species = SpeciesDog
	return pet
}

func hasSymptom(pet *DigitalPet, text string) bool {
	for _, s := range pet.Symptoms() {
		if strings.Contains(s.Sign, text) {
			return true
		}
	}
	return false
}

func TestOldPetsAreCats(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Species = ""
	if pet.GetSpecies() != SpeciesCat {
		t.Errorf("A pet saved before species existed should be a cat, got %s", pet.GetSpecies())
	}
	if s, ok := ParseSpecies("Dog"); !ok || s != SpeciesDog {
		t.Error("ParseSpecies should accept dog")
	}
	if _, ok := ParseSpecies("hamster"); ok {
		t.Error("ParseSpecies should reject unknown species")
	}
}

func TestCatUsesLitterBox(t *testing.T) {
	useRandom(t, 999_999) // Never fall ill
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Biology.Physiology.Bowel = catUseLevel
	pet.Update(0.05)
	if pet.LivingAreaWaste < stoolWaste || pet.Biology.Physiology.Bowel != 0 {
		t.Fatalf("The cat should use its litter box: waste %f, bowel %f", pet.LivingAreaWaste, pet.Biology.Physiology.Bowel)
	}

	// A cat avoids a filthy box
	pet.LivingAreaWaste = catAvoidLevel
	stress := pet.Biology.Vitals.Stress
	pet.Biology.Physiology.Bladder = catUseLevel
	pet.Update(0.05)
	if pet.Biology.Vitals.Stress <= stress || !hasSymptom(pet, "outside the litter box") {
		t.Errorf("A cat with a filthy box should go elsewhere and be stressed: %+v", pet.Symptoms())
	}

	// Cleaning empties the box
	r := pet.Interact(types.InteractionCleaning, 1.0)
	if !r.Applied || pet.LivingAreaWaste != 0 || r.Lesson == "" {
		t.Errorf("Cleaning should empty the box and teach why: %+v", r)
	}
	if r := pet.Interact(types.InteractionCleaning, 1.0); r.Refusal != "The litter box is already clean." {
		t.Errorf("Cleaning a clean box: %+v", r)
	}
}

func TestDogNeedsWalks(t *testing.T) {
	useRandom(t, 999_999)
	dog := newDog("Rex")
	dog.Biology.Physiology.Bladder = 0.6
	if r := dog.CurrentRequest(); r == nil || r.Action != "walk" {
		t.Errorf("A dog that needs to go out should ask for a walk, got %+v", r)
	}
	if !hasSymptom(dog, "Waiting by the door") {
		t.Error("A dog that needs to go out should wait by the door")
	}

	r := dog.Interact(types.InteractionWalking, 1.0)
	if !r.Applied || dog.Biology.Physiology.Bladder != 0 || dog.LivingAreaWaste != 0 {
		t.Fatalf("A walk should let the dog relieve itself outside: %+v", r)
	}

	// Left too long, a dog has an accident
	dog.Biology.Physiology.Bowel = dogAccidentLevel
	dog.Update(0.05)
	if dog.LivingAreaWaste == 0 || !hasSymptom(dog, "accident in the house") {
		t.Errorf("A dog left too long should have an accident: waste %f", dog.LivingAreaWaste)
	}

	cat := NewDigitalPet("Biscuit", "user123")
	if r := cat.Interact(types.InteractionWalking, 1.0); r.Applied || !strings.Contains(r.Refusal, "Cats don't go for walks") {
		t.Errorf("Cats shouldn't go for walks: %+v", r)
	}
}

func TestDirtyLitterBoxCausesIllness(t *testing.T) {
	useRandom(t, 0) // Fall ill at the first chance
	pet := NewDigitalPet("Biscuit", "user123")
	pet.LivingAreaWaste = 0.5
	pet.Update(0.05)
	if pet.Illness != nil {
		t.Fatal("A box that is only a little dirty should not cause illness")
	}

	pet.LivingAreaWaste = 1.0
	pet.Update(0.05)
	if pet.IllnessName() != "Urinary tract infection" {
		t.Fatalf("A filthy litter box should make the cat ill, got %q", pet.IllnessName())
	}
	if r := pet.CurrentRequest(); r == nil || r.Action != "medic" {
		t.Errorf("An ill pet should ask for medical care, got %+v", r)
	}
	if !hasSymptom(pet, "Straining") {
		t.Errorf("An ill cat should show signs of its illness: %+v", pet.Symptoms())
	}

	dog := newDog("Rex")
	dog.LivingAreaWaste = 1.0
	dog.Update(0.05)
	if dog.IllnessName() != "Intestinal parasites" {
		t.Errorf("A filthy home should give a dog parasites, got %q", dog.IllnessName())
	}
}

func TestIllnessWorsensUntilTreated(t *testing.T) {
	useRandom(t, 999_999)
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Illness = &Illness{Kind: IllnessUrinary, Severity: 0.1}
	for i := 0; i < 10; i++ {
		setCalmHealthyState(pet)
		pet.Biology.Vitals.Health = 0.9
		pet.Update(1)
	}
	if pet.Illness.Severity <= 0.4 {
		t.Errorf("An untreated illness should get worse, severity %f", pet.Illness.Severity)
	}
	if pet.Activity() != "is feeling sick" {
		t.Errorf("A seriously ill pet should behave sick, got %q", pet.Activity())
	}

	r := pet.Interact(types.InteractionMedicalCare, 1.0)
	if !r.Applied || pet.Illness != nil || !strings.Contains(r.Lesson, "litter box") {
		t.Errorf("Medical care should cure the illness and explain how to prevent it: %+v", r)
	}
	if symptoms := pet.Symptoms(); len(symptoms) == 0 || !symptoms[0].Good || !strings.Contains(symptoms[0].Sign, "recovering") {
		t.Errorf("Recovery should be shown as good news: %+v", symptoms)
	}
}

func TestUntreatedIllnessCanKill(t *testing.T) {
	useRandom(t, 999_999)
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Illness = &Illness{Kind: IllnessRespiratory, Severity: 1}
	pet.Biology.Vitals.Health = 0.02
	for i := 0; i < 20 && pet.IsAlive(); i++ {
		v := pet.Biology.Vitals
		v.Nutrition, v.Hydration, v.Energy = 1, 1, 1
		pet.Update(1)
	}
	if pet.IsAlive() {
		t.Fatal("An untreated illness should eventually be fatal")
	}
	if pet.Biology.CauseOfDeath != "Untreated upper respiratory infection" {
		t.Errorf("Cause of death = %q", pet.Biology.CauseOfDeath)
	}

	// In gentle mode the pet hibernates instead
	gentle := NewDigitalPet("Mochi", "user123")
	gentle.HibernationEnabled = true
	gentle.Illness = &Illness{Kind: IllnessRespiratory, Severity: 1}
	gentle.Biology.Vitals.Health = 0.12
	for i := 0; i < 20 && !gentle.IsHibernating(); i++ {
		v := gentle.Biology.Vitals
		v.Nutrition, v.Hydration, v.Energy = 1, 1, 1
		gentle.Update(1)
	}
	if !gentle.IsHibernating() || !gentle.IsAlive() {
		t.Error("In gentle mode an untreated illness should send the pet into hibernation")
	}
}

func TestTooManyTreatsUpsetStomach(t *testing.T) {
	pet := NewDigitalPet("Biscuit", "user123")
	for i := 0; i < 15; i++ {
		pet.Interact(types.InteractionRewards, 1.0)
	}
	if !hasSymptom(pet, "threw up") {
		t.Errorf("Many treats at once should make the pet throw up: %+v", pet.Symptoms())
	}
}

func TestLessonsFollowSpecies(t *testing.T) {
	cat := NewDigitalPet("Biscuit", "user123")
	cat.Biology.Vitals.Hydration = 0.3
	if r := cat.Interact(types.InteractionDrinking, 1.0); !strings.Contains(r.Lesson, "cats") {
		t.Errorf("A cat's lesson should be about cats: %q", r.Lesson)
	}
	dog := newDog("Rex")
	dog.Biology.Vitals.Hydration = 0.3
	if r := dog.Interact(types.InteractionDrinking, 1.0); !strings.Contains(r.Lesson, "Dogs") {
		t.Errorf("A dog's lesson should be about dogs: %q", r.Lesson)
	}
	if len(CareGuide(SpeciesCat)) == 0 || len(CareGuide(SpeciesDog)) == 0 {
		t.Error("Each species should have a care guide")
	}
}

func TestEverySymptomTopicIsInTheGuide(t *testing.T) {
	topics := map[Species]map[string]bool{}
	for species, guide := range careGuide {
		topics[species] = map[string]bool{}
		for _, topic := range guide {
			topics[species][topic.ID] = true
		}
	}
	for kind, bySpecies := range illnessSigns {
		for species, sign := range bySpecies {
			if !topics[species][sign.Topic] {
				t.Errorf("%s sign for %s refers to missing topic %q", kind, species, sign.Topic)
			}
		}
	}
	for kind, topic := range illnessTopics {
		for species := range careGuide {
			if !topics[species][topic] && !(kind == IllnessParasites && species == SpeciesCat) &&
				!(kind == IllnessUrinary && species == SpeciesDog) {
				t.Errorf("Illness %s refers to topic %q, missing for %s", kind, topic, species)
			}
		}
	}
}

func TestDailyChecklist(t *testing.T) {
	clock := useClock(t, time.Date(2026, 10, 19, 9, 0, 0, 0, time.Local))
	pet := NewDigitalPet("Biscuit", "user123")
	pet.Biology.Vitals.Nutrition = 0.2
	pet.Interact(types.InteractionFeeding, 1.0)

	items := pet.DailyChecklist()
	if items[0].Label != "Feed two meals" || items[0].Done != 1 || items[0].Target != 2 {
		t.Errorf("One meal should be checked off: %+v", items[0])
	}

	*clock = clock.Add(24 * time.Hour)
	if items := pet.DailyChecklist(); items[0].Done != 0 {
		t.Errorf("The checklist should start over each day: %+v", items[0])
	}

	if items := newDog("Rex").DailyChecklist(); items[2].Label != "Take three walks" {
		t.Errorf("A dog's checklist should include walks: %+v", items)
	}
}

func TestHealthSurvivesSaveLoad(t *testing.T) {
	dog := newDog("Rex")
	dog.LivingAreaWaste = 0.4
	dog.Illness = &Illness{Kind: IllnessParasites, Severity: 0.3}
	dog.Biology.Physiology.CardiovascularHealth = 0.6
	data, err := dog.Save()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GetSpecies() != SpeciesDog || loaded.LivingAreaWaste != 0.4 || loaded.IllnessName() != "Intestinal parasites" ||
		loaded.Biology.Physiology.CardiovascularHealth != 0.6 {
		t.Errorf("Species, living area, illness and body should be saved: %+v", loaded)
	}
}

func TestCatsAndDogsCantBreed(t *testing.T) {
	cat, dog := adultPet("Biscuit"), adultPet("Rex")
	dog.Species = SpeciesDog
	if _, reason := BreedingCompatibility(cat, dog); reason != errDifferentSpecies {
		t.Errorf("Cats and dogs shouldn't breed: %q", reason)
	}
	if _, err := Breed(cat, dog, "Pip"); err == nil {
		t.Error("Breeding a cat with a dog should fail")
	}
}
