package core

import "strings"

// Species is the kind of animal a pet is. Care differs between species,
// and the game teaches each one's real-world needs.
type Species string

const (
	SpeciesCat Species = "cat"
	SpeciesDog Species = "dog"
)

// ParseSpecies returns the species with the given name ("" means a cat,
// the original pet), or false if there is no such species
func ParseSpecies(name string) (Species, bool) {
	switch Species(strings.ToLower(strings.TrimSpace(name))) {
	case "", SpeciesCat:
		return SpeciesCat, true
	case SpeciesDog:
		return SpeciesDog, true
	}
	return "", false
}

// speciesLocked returns the pet's species; pets saved before species
// existed are cats. Caller must hold p.mu.
func (p *DigitalPet) speciesLocked() Species {
	if p.Species == SpeciesDog {
		return SpeciesDog
	}
	return SpeciesCat
}

// GetSpecies returns the pet's species
func (p *DigitalPet) GetSpecies() Species {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.speciesLocked()
}

// Plural returns the species' plural name, such as "cats"
func (s Species) Plural() string {
	return string(s) + "s"
}

// livingAreaNames name what gets dirty: a cat's litter box, a dog's home
var livingAreaNames = map[Species]string{
	SpeciesCat: "Litter box",
	SpeciesDog: "Home",
}
