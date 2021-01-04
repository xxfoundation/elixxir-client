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
)

/* fact object*/
type Fact struct {
	f *fact.Fact
}

func (f *Fact) Get() string {
	return f.f.Fact
}

func (f *Fact) Type() int {
	return int(f.f.T)
}

/* contact object*/
type Contact struct {
	c *contact.Contact
}

// GetID returns the user ID for this user.
func (c *Contact) GetID() []byte {
	return c.c.ID.Bytes()
}

// GetDHPublicKey returns the public key associated with the Contact.
func (c *Contact) GetDHPublicKey() []byte {
	return c.c.DhPubKey.Bytes()
}

// GetDHPublicKey returns hash of a DH proof of key ownership.
func (c *Contact) GetOwnershipProof() []byte {
	return c.c.OwnershipProof
}

// Returns a fact list for adding and getting facts to and from the contact
func (c *Contact) GetFactList() *FactList {
	return &FactList{c: c.c}
}

func (c *Contact) Marshal() ([]byte, error) {
	return c.c.Marshal(), nil
}

/* FactList object*/
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
	ft := fact.FactType(factType)
	if !ft.IsValid() {
		return errors.New("Invalid fact type")
	}
	fl.c.Facts = append(fl.c.Facts, fact.Fact{
		Fact: factData,
		T:    ft,
	})
	return nil
}

func (fl *FactList) Marshal() ([]byte, error) {
	return []byte(fl.c.Facts.Stringify()), nil
}
