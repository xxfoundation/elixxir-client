////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/sha256"
	"gitlab.com/privategrity/crypto/cyclic"
)

// Globally instantiated UserRegistry
var Users = newUserRegistry()

// Interface for User Registry operations
type UserRegistry interface {
	GetUser(id uint64) (user *User, ok bool)
	CountUsers() int
	LookupUser(hid uint64) (uid uint64, ok bool)
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[uint64]*User
	// Increments sequentially for User.UID values
	idCounter uint64
	// Temporary map acting as a lookup table for demo user registration codes
	userLookup map[uint64]uint64
}

// Creates a new UserRegistry interface
func newUserRegistry() UserRegistry {

	uc := make(map[uint64]*User)
	ul := make(map[uint64]uint64)
	// Deterministically create 1000 users
	for i := 1; i<=1000; i++ {
		t := new(User)
		h := sha256.New()
		// Generate user parameters
		t.UID = uint64(i)
		h.Write([]byte(string(20000+i)))
		t.Transmission.BaseKey = cyclic.NewIntFromBytes(h.Sum(nil))
		h.Write([]byte(string(30000+i)))
		t.Transmission.RecursiveKey = cyclic.NewIntFromBytes(h.Sum(nil))
		h.Write([]byte(string(40000+i)))
		t.Reception.BaseKey = cyclic.NewIntFromBytes(h.Sum(nil))
		h.Write([]byte(string(50000+i)))
		t.Reception.RecursiveKey = cyclic.NewIntFromBytes(h.Sum(nil))
		// Add user to collection and lookup table
		uc[t.UID] = t
		ul[t.UID+10000] = t.UID
	}
	uc[1].Nick = "David"
	uc[2].Nick = "Jim"
	uc[3].Nick = "Ben"
	uc[4].Nick = "Rick"
	uc[5].Nick = "Spencer"
	uc[6].Nick = "Jake"
	uc[7].Nick = "Mario"
	uc[8].Nick = "Will"

	// With an underlying UserMap data structure
	return UserRegistry(&UserMap{userCollection: uc, idCounter: 1000,
	userLookup: ul})
}

// Struct representing a User in the system
type User struct {
	UID uint64
	Transmission ForwardKey
	Reception    ForwardKey
	Nick string
}

type ForwardKey struct {
	BaseKey        *cyclic.Int
	RecursiveKey   *cyclic.Int
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id uint64) (user *User, ok bool) {
	user, ok = m.userCollection[id]
	return
}

// CountUsers returns a count of the users in userCollection
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}

// Looks up the user id corresponding to the demo registration code
func (m *UserMap) LookupUser(hid uint64) (uid uint64, ok bool) {
	uid, ok = m.userLookup[hid]
	return
}

