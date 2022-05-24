///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/contact"
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

func (fl *FactList) Get(i int) *Fact {
	return &Fact{f: &(fl.c.Facts)[i]}
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

/* ID list */
// IdList contains a list of IDs.
type IdList struct {
	list []*id.ID
}

// MakeIdList creates a new empty IdList.
func MakeIdList() *IdList {
	return &IdList{[]*id.ID{}}
}

// Len returns the number of IDs in the list.
func (idl *IdList) Len() int {
	return len(idl.list)
}

// Add appends the ID bytes to the end of the list.
func (idl *IdList) Add(idBytes []byte) error {
	newID, err := id.Unmarshal(idBytes)
	if err != nil {
		return err
	}

	idl.list = append(idl.list, newID)
	return nil
}

// Get returns the ID at the index. An error is returned if the index is out of
// range.
func (idl *IdList) Get(i int) ([]byte, error) {
	if i < 0 || i > len(idl.list) {
		return nil, errors.Errorf("ID list index must be between %d and the "+
			"last element %d.", 0, len(idl.list))
	}

	return idl.list[i].Bytes(), nil
}
