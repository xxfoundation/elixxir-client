///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

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

type RoundList struct {
	list []id.Round
}

// Gets the number of round IDs stored
func (rl *RoundList) Len() int {
	return len(rl.list)
}

// Gets a stored round ID at the given index
func (rl *RoundList) Get(i int) (int, error) {
	if i < 0 || i > len(rl.list) {
		return -1, errors.New("round ID cannot be under 0 or over" +
			" list len")
	}

	return int(rl.list[i]), nil
}
