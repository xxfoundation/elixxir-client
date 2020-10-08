package bindings

import "gitlab.com/xx_network/primitives/id"

type roundList struct {
	list []id.Round
}

// RoundList contains a list of contacts
type RoundList interface {
	// Len returns the number of contacts in the list
	Len() int
	// Get returns the round ID at index i
	Get(i int) int
}
