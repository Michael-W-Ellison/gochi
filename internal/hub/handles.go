package hub

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"os"
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

// seededHandle returns the handle a player's number (seed) gives on this
// hub, or, if that one is taken, the next free one after it. The number
// is mixed with the hub's secret key, so the same number gives a different
// handle on each hub (players can't be followed from hub to hub) and
// nobody can work out the number that gives someone else's handle.
func seededHandle(key []byte, seed uint32, taken func(string) bool) (string, error) {
	mac := hmac.New(sha256.New, key)
	_ = binary.Write(mac, binary.BigEndian, seed) // Writing to a hash never fails
	start := int(binary.BigEndian.Uint64(mac.Sum(nil)) % uint64(handleCount))
	for step := 0; step < handleCount; step++ {
		if handle := handleAt((start + step) % handleCount); !taken(handle) {
			return handle, nil
		}
	}
	return "", fmt.Errorf("no free handle found")
}

// loadOrMakeKey reads the hub's secret key from path, making a random one
// the first time
func loadOrMakeKey(path string) ([]byte, error) {
	key, err := os.ReadFile(path)
	if err == nil && len(key) == 32 {
		return key, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, os.WriteFile(path, key, 0o600)
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
