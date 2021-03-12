///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"errors"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

/*IntList*/

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

/*RoundList*/

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

/*ContactList*/

type ContactList struct {
	list []contact.Contact
}

// Gets the number of round IDs stored
func (cl *ContactList) Len() int {
	return len(cl.list)
}

// Gets a stored round ID at the given index
func (cl *ContactList) Get(i int) (*Contact, error) {
	if i < 0 || i > len(cl.list) {
		return nil, errors.New("contact cannot be under 0 or over" +
			" list len")
	}

	return &Contact{c: &cl.list[i]}, nil
}

/*FactList*/
func NewFactList() *FactList {
	return &FactList{c: &contact.Contact{
		ID:             nil,
		DhPubKey:       nil,
		OwnershipProof: nil,
		Facts:          make([]fact.Fact, 0),
	}}
}

type FactList struct {
	c *contact.Contact
}

func (fl *FactList) Num() int {
	return len(fl.c.Facts)
}

func (fl *FactList) Get(i int) Fact {
	return Fact{f: &(fl.c.Facts)[i]}
}

func (fl *FactList) Add(factData string, factType int) error {
	f, err := fact.NewFact(fact.FactType(factType), factData)
	if err != nil {
		return err
	}
	fl.c.Facts = append(fl.c.Facts, f)
	return nil
}

func (fl *FactList) Stringify() (string, error) {
	return fl.c.Facts.Stringify(), nil
}
