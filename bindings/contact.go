///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
)

/* fact object*/
//creates a new fact. The factType must be either:
//  0 - Username
//  1 - Email
//  2 - Phone Number
// The fact must be well formed for the type and must not include commas or
// semicolons. If it is not well formed, it will be rejected.  Phone numbers
// must have the two letter country codes appended.  For the complete set of
// validation, see /elixxir/primitives/fact/fact.go
func NewFact(factType int, factStr string) (*Fact, error) {
	f, err := fact.NewFact(fact.FactType(factType), factStr)
	if err != nil {
		return nil, err
	}
	return &Fact{f: &f}, nil
}

type Fact struct {
	f *fact.Fact
}

func (f *Fact) Get() string {
	return f.f.Fact
}

func (f *Fact) Type() int {
	return int(f.f.T)
}

func (f *Fact) Stringify() string {
	return f.f.Stringify()
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

// GetAPIContact returns the api contact object. Not exported to bindings.
func (c *Contact) GetAPIContact() *contact.Contact {
	return c.c
}
