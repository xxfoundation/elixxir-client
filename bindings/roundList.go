package bindings

import (
	"errors"
	"gitlab.com/xx_network/primitives/id"
)

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

func (rl roundList) Len() int {
	return len(rl.list)
}

func (rl roundList) Get(i int) (int, error) {
	if i < 0 || i > len(rl.list) {
		return -1, errors.New("round ID cannot be under 0 or over list len")
	}

	return int(rl.list[i]), nil
}
