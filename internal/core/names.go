package core

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Pet names stay on the player's computer (they are never sent to a hub)
// but are still kept short and printable

// MaxPetNameLength is the longest a pet's name may be, in characters
const MaxPetNameLength = 24

// PetNameRule describes a valid pet name
var PetNameRule = fmt.Sprintf("A pet's name is 1 to %d letters, digits, spaces or punctuation", MaxPetNameLength)

// ValidPetName reports whether a pet name follows PetNameRule (spaces at
// either end don't count)
func ValidPetName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxPetNameLength {
		return false
	}
	for _, r := range name {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
