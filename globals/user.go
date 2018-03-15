////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/sha256"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/hash"
)

// Globally instantiated UserRegistry
var Users = newUserRegistry()
var NUM_DEMO_USERS = int(10)
var DEMO_NICKS = []string {"David", "Jim", "Ben", "Rick", "Spencer", "Jake",
"Mario", "Will", "Allan", "Jono"}

// Interface for User Registry operations
type UserRegistry interface {
	NewUser(id uint64, nickname string) *User
	DeleteUser(id uint64)
	GetUser(id uint64) (user *User, ok bool)
	UpsertUser(user *User)
	CountUsers() int
	LookupUser(hid uint64) (uid uint64, ok bool)
	LookupKeys(uid uint64) (*NodeKeys, bool)
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[uint64]*User
	// Increments sequentially for User.UID values
	idCounter uint64
	// Temporary map acting as a lookup table for demo user registration codes
	userLookup map[uint64]uint64
	//Temporary placed to store the keys for each user
	keysLookup map[uint64]*NodeKeys
}

// newUserRegistry creates a new UserRegistry interface
func newUserRegistry() UserRegistry {

	uc := make(map[uint64]*User)
	ul := make(map[uint64]uint64)
	nk := make(map[uint64]*NodeKeys)

	// Deterministically create 1000 users
	for i := 1; i <= NUM_DEMO_USERS; i++ {
		t := new(User)
		k := new(NodeKeys)
		h := sha256.New()

		// Generate user parameters
		t.UID = uint64(i)
		h.Write([]byte(string(20000 + i)))
		k.TransmissionKeys.Base = cyclic.NewIntFromBytes(h.Sum(nil))
		h.Write([]byte(string(30000 + i)))
		k.TransmissionKeys.Recursive = cyclic.NewIntFromBytes(h.Sum(nil))
		h.Write([]byte(string(40000 + i)))
		k.ReceptionKeys.Base = cyclic.NewIntFromBytes(h.Sum(nil))
		h.Write([]byte(string(50000 + i)))
		k.ReceptionKeys.Recursive = cyclic.NewIntFromBytes(h.Sum(nil))

		// Add user to collection and lookup table
		uc[t.UID] = t
		ul[UserHash(t.UID)] = t.UID
		nk[t.UID] = k
	}

	for i := 0; i < len(DEMO_NICKS); i++ {
		uc[uint64(i+1)].Nick = DEMO_NICKS[i]
	}

	// With an underlying UserMap data structure
	return UserRegistry(&UserMap{userCollection: uc,
		idCounter:  uint64(NUM_DEMO_USERS),
		userLookup: ul,
		keysLookup: nk})
}

// Struct representing a User in the system
type User struct {
	UID  uint64
	Nick string
}

// DeepCopy performs a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	nu := new(User)
	nu.UID = u.UID
	nu.Nick = u.Nick
	return nu
}

// UserHash generates a hash of the UID to be used as a registration code for
// demos
func UserHash(uid uint64) uint64 {
	var huid []byte
	h, _ := hash.NewCMixHash()
	h.Write(cyclic.NewIntFromUInt(uid).LeftpadBytes(8))
	huid = h.Sum(huid)
	return cyclic.NewIntFromBytes(huid).Uint64()
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(id uint64, nickname string) *User {

	if id < uint64(NUM_DEMO_USERS) {
		jww.FATAL.Panicf("Invalid User ID!")
	}
	return &User{UID: id, Nick: nickname}
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id uint64) (user *User, ok bool) {
	user, ok = m.userCollection[id]
	user = user.DeepCopy()
	return
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id uint64) {
	// If key does not exist, do nothing
	delete(m.userCollection, id)
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	m.userCollection[user.UID] = user
}

// CountUsers returns a count of the users in userCollection
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}

// LookupUser returns the user id corresponding to the demo registration code
func (m *UserMap) LookupUser(hid uint64) (uid uint64, ok bool) {
	uid, ok = m.userLookup[hid]
	return
}

// LookupKeys returns the keys for the given user from the temporary key map
func (m *UserMap) LookupKeys(uid uint64) (*NodeKeys, bool) {
	nk, t := m.keysLookup[uid]
	return nk, t
}
