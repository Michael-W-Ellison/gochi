// Package interaction provides the rules and items behind caring for a
// pet in Gochi:
//   - Which interactions a pet's state allows (ValidateInteraction), and
//     a description of each
//   - Items with types and rarities, an inventory (a pet's pantry), item
//     catalogs, and a shop that applies discounts and takes payment
//
// The core package applies interactions to the pet and builds its
// catalogs and shop from these.
package interaction
