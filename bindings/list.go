package bindings

import (
	"errors"
	"gitlab.com/xx_network/primitives/id"
)

type IntList struct {
	lst []int
}

func MakeIntList() *IntList {
	return &IntList{lst: make([]int, 0)}
}

func (il *IntList) Add(i int) {
	il.lst = append(il.lst, i)
}

func (il *IntList) Len() int {
	return len(il.lst)
}

func (il *IntList) Get(i int) (int, error) {
	if i < 0 || i >= len(il.lst) {
		return 0, errors.New("invalid index")
	}
	return il.lst[i], nil
}

type roundList struct {
	list []id.Round
}

// RoundList contains a list of contacts
type RoundList interface {
	// Len returns the number of contacts in the list
	Len() int
	// Get returns the round ID at index i
	Get(i int) (int, error)
}

// Gets the number of round IDs stored
func (rl *roundList) Len() int {
	return len(rl.list)
}

// Gets a stored round ID at the given index
func (rl *roundList) Get(i int) (int, error) {
	if i < 0 || i > len(rl.list) {
		return -1, errors.New("round ID cannot be under 0 or over" +
			" list len")
	}

	return int(rl.list[i]), nil
}
