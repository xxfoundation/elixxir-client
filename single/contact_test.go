///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// Happy path.
func TestNewContact(t *testing.T) {
	grp := getGroup()
	used := int32(0)
	expected := Contact{
		partner:       id.NewIdFromString("sender ID", id.User, t),
		partnerPubKey: grp.NewInt(99),
		dhKey:         grp.NewInt(42),
		tagFP:         singleUse.UnmarshalTagFP([]byte("test tagFP")),
		maxParts:      uint8(rand.Uint64()),
		used:          &used,
	}

	testC := NewContact(expected.partner, expected.partnerPubKey,
		expected.dhKey, expected.tagFP, expected.maxParts)

	if !expected.Equal(testC) {
		t.Errorf("NewContact() did not return the expected Contact."+
			"\nexpected: %s\nrecieved: %s", expected, testC)
	}
}

// Happy path.
func TestContact_GetMaxParts(t *testing.T) {
	grp := getGroup()
	maxParts := uint8(rand.Uint64())
	c := NewContact(id.NewIdFromString("sender ID", id.User, t), grp.NewInt(99),
		grp.NewInt(42), singleUse.TagFP{}, maxParts)

	if maxParts != c.GetMaxParts() {
		t.Errorf("GetMaxParts() failed to return the expected maxParts."+
			"\nexpected %d\nreceived: %d", maxParts, c.GetMaxParts())
	}
}

// Happy path.
func TestContact_GetPartner(t *testing.T) {
	grp := getGroup()
	senderID := id.NewIdFromString("sender ID", id.User, t)
	c := NewContact(senderID, grp.NewInt(99), grp.NewInt(42), singleUse.TagFP{},
		uint8(rand.Uint64()))

	if !senderID.Cmp(c.GetPartner()) {
		t.Errorf("GetPartner() failed to return the expected sender ID."+
			"\nexpected %s\nreceived: %s", senderID, c.GetPartner())
	}

}

// Happy path.
func TestContact_String(t *testing.T) {
	grp := getGroup()
	a := NewContact(id.NewIdFromString("sender ID 1", id.User, t), grp.NewInt(99),
		grp.NewInt(42), singleUse.UnmarshalTagFP([]byte("test tagFP 1")), uint8(rand.Uint64()))
	b := NewContact(id.NewIdFromString("sender ID 2", id.User, t),
		grp.NewInt(98), grp.NewInt(43), singleUse.UnmarshalTagFP([]byte("test tagFP 2")), uint8(rand.Uint64()))
	c := NewContact(a.GetPartner(), a.partnerPubKey.DeepCopy(), a.dhKey.DeepCopy(), a.tagFP, a.maxParts)

	if a.String() == b.String() {
		t.Errorf("String() did not return the expected string."+
			"\na: %s\nb: %s", a, b)
	}
	if a.String() != c.String() {
		t.Errorf("String() did not return the expected string."+
			"\na: %s\nc: %s", a, c)
	}
}

// Happy path.
func TestContact_Equal(t *testing.T) {
	grp := getGroup()
	a := NewContact(id.NewIdFromString("sender ID 1", id.User, t),
		grp.NewInt(99), grp.NewInt(42), singleUse.UnmarshalTagFP([]byte("test tagFP 1")), uint8(rand.Uint64()))
	b := NewContact(id.NewIdFromString("sender ID 2", id.User, t),
		grp.NewInt(98), grp.NewInt(43), singleUse.UnmarshalTagFP([]byte("test tagFP 2")), uint8(rand.Uint64()))
	c := NewContact(a.GetPartner(), a.partnerPubKey.DeepCopy(), a.dhKey.DeepCopy(), a.tagFP, a.maxParts)

	if a.Equal(b) {
		t.Errorf("Equal() found two different Contacts as equal."+
			"\na: %s\nb: %s", a, b)
	}
	if !a.Equal(c) {
		t.Errorf("Equal() found two equal Contacts as not equal."+
			"\na: %s\nc: %s", a, c)
	}
}
