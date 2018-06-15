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
	"encoding/binary"
)

// TODO use this type for User IDs consistently throughout
// FIXME use string or []byte for this - string works as a key for hash maps
// and []byte is compatible with more other languages.
// string probably makes more sense
type UserID uint64

var UserIDLen = 8

// TODO remove this when UserID becomes a string
func (u UserID) Bytes() []byte {
	result := make([]byte, UserIDLen)
	binary.BigEndian.PutUint64(result, uint64(u))
	return result
}

// TODO clean this up
func (u UserID) RegistrationCode() string {
	return cyclic.NewIntFromUInt(uint64(NewUserIDFromBytes(UserHash(u)))).TextVerbose(32, 0)
}

func NewUserIDFromBytes(id []byte) UserID {
	return UserID(binary.BigEndian.Uint64(id))
}

// Globally instantiated UserRegistry
var Users = newUserRegistry()
var NUM_DEMO_USERS = int(40)
var DEMO_USER_NICKS = []string{"David", "Jim", "Ben", "Rick", "Spencer", "Jake",
	"Mario", "Will", "Allan", "Jono", "", "", "UDB"}
var DEMO_CHANNEL_NAMES = []string{"#General", "#Engineering", "#Lunch",
	"#Random"}

// Interface for User Registry operations
type UserRegistry interface {
	NewUser(id UserID, nickname string) *User
	DeleteUser(id UserID)
	GetUser(id UserID) (user *User, ok bool)
	UpsertUser(user *User)
	CountUsers() int
	LookupUser(hid string) (uid UserID, ok bool)
	LookupKeys(uid UserID) (*NodeKeys, bool)
	GetContactList() ([]UserID, []string)
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[UserID]*User
	// Increments sequentially for User.UserID values
	idCounter uint64
	// Temporary map acting as a lookup table for demo user registration codes
	// Key type is string because keys must implement == and []byte doesn't
	userLookup map[string]UserID
	//Temporary placed to store the keys for each user
	keysLookup map[UserID]*NodeKeys
}

// newUserRegistry creates a new UserRegistry interface
func newUserRegistry() UserRegistry {
	if len(DEMO_CHANNEL_NAMES) > 10 || len(DEMO_USER_NICKS) > 30 {
		jww.ERROR.Print("Not enough demo users have been hardcoded.")
	}
	uc := make(map[UserID]*User)
	ul := make(map[string]UserID)
	nk := make(map[UserID]*NodeKeys)

	// Deterministically create NUM_DEMO_USERS users
	for i := 1; i <= NUM_DEMO_USERS; i++ {
		t := new(User)
		k := new(NodeKeys)

		// Generate user parameters
		t.UserID = UserID(i)
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
		ul[string(UserHash(t.UserID))] = t.UserID
		nk[t.UserID] = k
	}

	// Channels have been hardcoded to users 101-200
	for i := 0; i < len(DEMO_USER_NICKS); i++ {
		uc[UserID(i+1)].Nick = DEMO_USER_NICKS[i]
	}
	for i := 0; i < len(DEMO_CHANNEL_NAMES); i++ {
		uc[UserID(i+31)].Nick = DEMO_CHANNEL_NAMES[i]
	}

	// With an underlying UserMap data structure
	return UserRegistry(&UserMap{userCollection: uc,
		idCounter:  uint64(NUM_DEMO_USERS),
		userLookup: ul,
		keysLookup: nk})
}

// Struct representing a User in the system
type User struct {
	UserID UserID
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
func UserHash(uid UserID) []byte {
	var huid []byte
	h, _ := hash.NewCMixHash()
	uidBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(uidBytes, uint64(uid))
	h.Write(uidBytes)
	huid = h.Sum(huid)
	return huid
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(id UserID, nickname string) *User {
	return &User{UserID: id, Nick: nickname}
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id UserID) (user *User, ok bool) {
	user, ok = m.userCollection[id]
	user = user.DeepCopy()
	return
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id UserID) {
	// If key does not exist, do nothing
	delete(m.userCollection, id)
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	m.userCollection[user.UserID] = user
}

// CountUsers returns a count of the users in userCollection
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}

// LookupUser returns the user id corresponding to the demo registration code
func (m *UserMap) LookupUser(hid string) (uid UserID, ok bool) {
	uid, ok = m.userLookup[hid]
	return
}

// LookupKeys returns the keys for the given user from the temporary key map
func (m *UserMap) LookupKeys(uid UserID) (*NodeKeys, bool) {
	nk, t := m.keysLookup[uid]
	return nk, t
}

func (m *UserMap) GetContactList() (ids []UserID, nicks []string) {
	ids = make([]UserID, len(m.userCollection))
	nicks = make([]string, len(m.userCollection))

	index := uint64(0)
	for _, user := range m.userCollection {
		ids[index] = user.UserID
		nicks[index] = user.Nick
		index++
	}
	return ids, nicks
}
