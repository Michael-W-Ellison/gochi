package core

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// exercise runs what the game does with a pet and returns any panic
func exercise(pet *DigitalPet) (failure string) {
	done := make(chan string, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Sprint(r)
			}
		}()
		pet.Tally()
		pet.Conditions()
		pet.SaveRecord()
		pet.BodySystems()
		pet.Symptoms()
		pet.DailyChecklist()
		pet.DailyGoals()
		pet.PsychologicalNeeds()
		pet.EnvironmentStatus()
		pet.Tricks()
		pet.GetLivingArea()
		pet.IllnessName()
		pet.Relations()
		pet.LifeStatus()
		pet.Forecast(0.5)
		pet.CheckAchievements()
		pet.Generation()
		for it := types.InteractionFeeding; it <= types.InteractionEnvironmentChange; it++ {
			pet.Interact(it, 1)
		}
		pet.Update(0.5)
		if _, err := pet.Save(); err != nil {
			done <- "save: " + err.Error()
			return
		}
		done <- ""
	}()
	select {
	case failure = <-done:
		return failure
	case <-time.After(30 * time.Second):
		return "did not finish"
	}
}

// paths returns the paths of every object and array (in objects) and of
// every number (in numbers) in a decoded JSON value
func paths(v interface{}, prefix []string, objects, numbers *[][]string) {
	visit := func(key string, child interface{}) {
		p := append(append([]string{}, prefix...), key)
		switch child.(type) {
		case map[string]interface{}, []interface{}:
			*objects = append(*objects, p)
		case float64:
			*numbers = append(*numbers, p)
		}
		paths(child, p, objects, numbers)
	}
	switch t := v.(type) {
	case map[string]interface{}:
		for k, child := range t {
			visit(k, child)
		}
	case []interface{}:
		for i, child := range t {
			visit(strconv.Itoa(i), child)
		}
	}
}

// setPath returns a copy of the JSON with the value at path replaced
func setPath(v interface{}, path []string, value interface{}) interface{} {
	switch t := v.(type) {
	case []interface{}:
		c := append([]interface{}{}, t...)
		i, _ := strconv.Atoi(path[0])
		if len(path) == 1 {
			c[i] = value
		} else {
			c[i] = setPath(t[i], path[1:], value)
		}
		return c
	default:
		m := v.(map[string]interface{})
		c := make(map[string]interface{}, len(m))
		for k, x := range m {
			c[k] = x
		}
		if len(path) == 1 {
			c[path[0]] = value
		} else {
			c[path[0]] = setPath(m[path[0]], path[1:], value)
		}
		return c
	}
}

// Damaged saves (a part missing, for example after a hand edit or disk
// corruption) are refused or repaired when loaded, never crash the game
func TestDamagedSavesDontCrash(t *testing.T) {
	pet := NewDigitalPet("Mochi", "owner")
	pet.Interact(types.InteractionPetting, 1)
	pet.Update(0.2)
	data, err := pet.Save()
	if err != nil {
		t.Fatal(err)
	}
	var tree interface{}
	if err := json.Unmarshal(data, &tree); err != nil {
		t.Fatal(err)
	}
	var objects, numbers [][]string
	paths(tree, nil, &objects, &numbers)
	type change struct {
		path  []string
		value interface{}
	}
	var changes []change
	for _, path := range objects {
		changes = append(changes, change{path, nil}, change{path, map[string]interface{}{}})
	}
	if !testing.Short() {
		for _, path := range numbers {
			for _, value := range []float64{1e308, -1e308, -1, 0, 1e15} {
				changes = append(changes, change{path, value})
			}
		}
	}
	sort.SliceStable(changes, func(i, j int) bool {
		return strings.Join(changes[i].path, ".") < strings.Join(changes[j].path, ".")
	})
	for _, c := range changes {
		path, value := c.path, c.value
		{
			damaged, _ := json.Marshal(setPath(tree, path, value))
			loaded, err := Load(damaged)
			if err != nil {
				continue // Refused: the game falls back to a backup
			}
			if failure := exercise(loaded); failure != "" {
				t.Errorf("%s = %v: %s", strings.Join(path, "."), value, failure)
			}
		}
	}
}
