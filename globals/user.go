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

// TODO use this type for User IDs consistently throughout
type UserID string

// Globally instantiated UserRegistry
var Users = newUserRegistry()
var NUM_DEMO_USERS = int(40)
var DEMO_USER_NICKS = []string{"David", "Jim", "Ben", "Rick", "Spencer", "Jake",
	"Mario", "Will", "Allan", "Jono", "", "", "UDB"}
var DEMO_CHANNEL_NAMES = []string{"#General", "#Engineering", "#Lunch",
	"#Random"}

// Interface for User Registry operations
type UserRegistry interface {
	NewUser(id uint64, nickname string) *User
	DeleteUser(id uint64)
	GetUser(id uint64) (user *User, ok bool)
	UpsertUser(user *User)
	CountUsers() int
	LookupUser(hid uint64) (uid uint64, ok bool)
	LookupKeys(uid uint64) (*NodeKeys, bool)
	GetContactList() ([]uint64, []string)
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[uint64]*User
	// Increments sequentially for User.UserID values
	idCounter uint64
	// Temporary map acting as a lookup table for demo user registration codes
	userLookup map[uint64]uint64
	//Temporary placed to store the keys for each user
	keysLookup map[uint64]*NodeKeys
}

// newUserRegistry creates a new UserRegistry interface
func newUserRegistry() UserRegistry {
	if len(DEMO_CHANNEL_NAMES) > 10 || len(DEMO_USER_NICKS) > 30 {
		jww.ERROR.Print("Not enough demo users have been hardcoded.")
	}
	uc := make(map[uint64]*User)
	ul := make(map[uint64]uint64)
	nk := make(map[uint64]*NodeKeys)

	// Deterministically create NUM_DEMO_USERS users
	for i := 1; i <= NUM_DEMO_USERS; i++ {
		t := new(User)
		k := new(NodeKeys)

		// Generate user parameters
		t.UserID = uint64(i)
		h := sha256.New()
		h.Write([]byte(string(20000 + i)))
		k.TransmissionKeys.Base = cyclic.NewIntFromBytes(h.Sum(nil))
		h = sha256.New()
		h.Write([]byte(string(30000 + i)))
		k.TransmissionKeys.Recursive = cyclic.NewIntFromBytes(h.Sum(nil))
		h = sha256.New()
		h.Write([]byte(string(40000 + i)))
		k.ReceptionKeys.Base = cyclic.NewIntFromBytes(h.Sum(nil))
		h = sha256.New()
		h.Write([]byte(string(50000 + i)))
		k.ReceptionKeys.Recursive = cyclic.NewIntFromBytes(h.Sum(nil))

		// Add user to collection and lookup table
		uc[t.UserID] = t
		ul[UserHash(t.UserID)] = t.UserID
		nk[t.UserID] = k
	}

	// Channels have been hardcoded to users 101-200
	for i := 0; i < len(DEMO_USER_NICKS); i++ {
		uc[uint64(i+1)].Nick = DEMO_USER_NICKS[i]
	}
	for i := 0; i < len(DEMO_CHANNEL_NAMES); i++ {
		uc[uint64(i+31)].Nick = DEMO_CHANNEL_NAMES[i]
	}

	// With an underlying UserMap data structure
	return UserRegistry(&UserMap{userCollection: uc,
		idCounter:  uint64(NUM_DEMO_USERS),
		userLookup: ul,
		keysLookup: nk})
}

// Struct representing a User in the system
type User struct {
	UserID uint64
	Nick   string
}

// DeepCopy performs a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	nu := new(User)
	nu.UserID = u.UserID
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
	return &User{UserID: id, Nick: nickname}
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
	/*delete(m.keysLookup, id)
	delete(m.userLookup, id)*/
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	m.userCollection[user.UserID] = user
	/*m.userLookup[huid] = user.UserID
	m.keysLookup[user.UserID] = keys*/
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

func (m *UserMap) GetContactList() (ids []uint64, nicks []string) {
	ids = make([]uint64, len(m.userCollection))
	nicks = make([]string, len(m.userCollection))

	index := uint64(0)
	for _, user := range m.userCollection {
		ids[index] = user.UserID
		nicks[index] = user.Nick
		index++
	}
	return ids, nicks
}
