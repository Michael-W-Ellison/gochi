package hub

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Players don't choose names: the hub gives each account a handle made of
// a color, an animal and a number, such as "Blue Otter 42". Handles are the
// only way other players see each other, so nobody's real name, or
// anything else a player typed, is ever shown to anyone.

var handleColors = []string{
	"Amber", "Aqua", "Azure", "Berry", "Blue", "Bronze", "Coral", "Cream", "Crimson", "Gold", "Green", "Hazel",
	"Indigo", "Ivory", "Jade", "Lemon", "Lilac", "Lime", "Maple", "Mint", "Navy", "Olive", "Orange", "Peach",
	"Pearl", "Pink", "Plum", "Purple", "Ruby", "Sage", "Silver", "Teal",
}

var handleAnimals = []string{
	"Badger", "Beaver", "Bunny", "Cub", "Deer", "Dolphin", "Duck", "Finch", "Fox", "Frog", "Gecko", "Goose",
	"Hedgehog", "Heron", "Koala", "Lamb", "Lark", "Lynx", "Mole", "Moose", "Newt", "Otter", "Owl", "Panda",
	"Puffin", "Robin", "Seal", "Sparrow", "Squirrel", "Swan", "Turtle", "Wren",
}

// randomIndex returns a random number from 0 to n-1
func randomIndex(n int) (int, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}

// handleCount is how many handles there are: every color, animal and
// number from 10 to 99
var handleCount = len(handleColors) * len(handleAnimals) * 90

// handleAt returns the handle numbered i, from 0 to handleCount-1
func handleAt(i int) string {
	n, i := i%90, i/90
	a, c := i%len(handleAnimals), i/len(handleAnimals)
	return fmt.Sprintf("%s %s %d", handleColors[c], handleAnimals[a], n+10)
}

// seededHandle returns the handle a player's number (seed) gives, or, if
// that one is taken, the next free one after it. The same number gives
// the same handle on every hub where it is free.
func seededHandle(seed uint32, taken func(string) bool) (string, error) {
	start := int(uint64(seed) * 2654435761 % uint64(handleCount)) // Spreads close numbers apart
	for step := 0; step < handleCount; step++ {
		if handle := handleAt((start + step) % handleCount); !taken(handle) {
			return handle, nil
		}
	}
	return "", fmt.Errorf("no free handle found")
}

// newHandle returns a random handle that taken doesn't report as taken
func newHandle(taken func(string) bool) (string, error) {
	for tries := 0; tries < 1000; tries++ {
		c, err := randomIndex(len(handleColors))
		if err != nil {
			return "", err
		}
		a, err := randomIndex(len(handleAnimals))
		if err != nil {
			return "", err
		}
		n, err := randomIndex(90)
		if err != nil {
			return "", err
		}
		handle := fmt.Sprintf("%s %s %d", handleColors[c], handleAnimals[a], n+10)
		if !taken(handle) {
			return handle, nil
		}
	}
	return "", fmt.Errorf("no free handle found")
}

// ValidHandle reports whether text is a handle the hub could have given
func ValidHandle(text string) bool {
	var color, animal string
	var n int
	if c, err := fmt.Sscanf(text, "%s %s %d", &color, &animal, &n); err != nil || c != 3 || n < 10 || n > 99 {
		return false
	}
	if fmt.Sprintf("%s %s %d", color, animal, n) != text {
		return false
	}
	return contains(handleColors, color) && contains(handleAnimals, animal)
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
