////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"crypto/sha256"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/hash"
	"encoding/base32"
)

// Most string types in most languages (with C excepted) support 0 as a
// character in a string, for Unicode support. So it's possible to use normal
// strings as an immutable container for bytes in all the languages we care
// about supporting.
type ID string

// Length of IDs in bytes
// 128 bits
const IDLen = 16
// This can't be a const because the golang compiler doesn't support putting
// constant expressions in a constant
var ZeroID = ID(make([]byte, IDLen))
// So smart

// Length of registration code in raw bytes
// Must be a multiple of 5 bytes to work with base 32
// 8 character long reg codes when base-32 encoded currently with length of 5
const RegCodeLen = 5

func (u ID) RegistrationCode() string {
	return base32.StdEncoding.EncodeToString(UserHash(u))
}

// Globally instantiated Registry
var Users = newRegistry()
var NUM_DEMO_USERS = int(40)
var DEMO_USER_NICKS = []string{"David", "Jim", "Ben", "Rick", "Spencer", "Jake",
	"Mario", "Will", "Allan", "Jono", "", "", "UDB", "", "", "", "Payments"}
var DEMO_CHANNEL_NAMES = []string{"#General", "#Engineering", "#Lunch",
	"#Random"}

// Interface for User Registry operations
type Registry interface {
	NewUser(id ID, nickname string) *User
	DeleteUser(id ID)
	GetUser(id ID) (user *User, ok bool)
	UpsertUser(user *User)
	CountUsers() int
	LookupUser(hid string) (uid ID, ok bool)
	LookupKeys(uid ID) (*NodeKeys, bool)
	GetContactList() ([]ID, []string)
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	// NOTA BENE MOTHERFUCKERS When you index into this map, make sure to use
	// a proper ID that has IDLen bytes in it
	userCollection map[ID]*User
	// Increments sequentially for User.ID values
	idCounter uint64
	// Temporary map acting as a lookup table for demo user registration codes
	// Key type is string because keys must implement == and []byte doesn't
	userLookup map[string]ID
	//Temporary placed to store the keys for each user
	keysLookup map[ID]*NodeKeys
}

// newRegistry creates a new Registry interface
func newRegistry() Registry {
	if len(DEMO_CHANNEL_NAMES) > 10 || len(DEMO_USER_NICKS) > 30 {
		globals.Log.ERROR.Print("Not enough demo users have been hardcoded.")
	}
	uc := make(map[ID]*User)
	ul := make(map[string]ID)
	nk := make(map[ID]*NodeKeys)

	// Deterministically create NUM_DEMO_USERS users
	// Start at ID 1
	firstID := []byte{0x01}
	firstID = append(make([]byte, IDLen-len(firstID)), firstID...)
	currentID := ID(firstID)
	for i := 1; i <= NUM_DEMO_USERS; i++ {
		t := new(User)
		k := new(NodeKeys)

		// Generate user parameters
		t.UserID = currentID
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
		// Detect collisions in the registration code
		if _, ok := ul[t.UserID.RegistrationCode()]; ok {
			globals.Log.ERROR.Printf(
				"Collision in demo user list creation at %v. "+
					"Please fix ASAP (include more bits to the reg code.", i)
		}
		ul[t.UserID.RegistrationCode()] = t.UserID
		nk[t.UserID] = k
		currentID = currentID.nextID()
	}

	// Channels have been hardcoded to users starting with 31
	firstID = []byte{1}
	firstID = append(make([]byte, IDLen-len(firstID)), firstID...)
	currentID = ID(firstID)
	for i := 0; i < len(DEMO_USER_NICKS); i++ {
		uc[currentID].Nick = DEMO_USER_NICKS[i]
		currentID = currentID.nextID()
	}

	firstID = []byte{31}
	firstID = append(make([]byte, IDLen-len(firstID)), firstID...)
	currentID = ID(firstID)
	for i := 0; i < len(DEMO_CHANNEL_NAMES); i++ {
		uc[currentID].Nick = DEMO_CHANNEL_NAMES[i]
		currentID = currentID.nextID()
	}

	// With an underlying UserMap data structure
	return Registry(&UserMap{userCollection: uc,
		idCounter: uint64(NUM_DEMO_USERS),
		userLookup: ul,
		keysLookup: nk})
}

// In most situations we only need to compare IDs for equality, so this func
// isn't exported.
// Adding a number to an ID, or incrementing an ID,
// will normally have no meaning.
func (u ID) nextID() ID {
	// IDs are fixed length byte strings so it's actually straightforward to
	// increment them without going out to a big.Int
	if len(u) != IDLen {
		panic("nextID(): length of ID was incorrect")
	}
	result := make([]byte, IDLen)
	copy(result, u)
	// increment byte by byte starting from the end of the array
	for i := IDLen - 1; i >= 0; i-- {
		result[i]++
		if result[i] != 0 {
			break
		}
	}
	return ID(result)
}

// Struct representing a User in the system
type User struct {
	UserID ID
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
// TODO Should we use the full-length hash? Should we even be doing registration
// like this?
func UserHash(uid ID) []byte {
	h, _ := hash.NewCMixHash()
	h.Write([]byte(uid))
	huid := h.Sum(nil)
	huid = huid[len(huid)-RegCodeLen:]
	return huid
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(id ID, nickname string) *User {
	return &User{UserID: id, Nick: nickname}
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id ID) (user *User, ok bool) {
	user, ok = m.userCollection[id]
	user = user.DeepCopy()
	return
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id ID) {
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
func (m *UserMap) LookupUser(hid string) (uid ID, ok bool) {
	uid, ok = m.userLookup[hid]
	return
}

// LookupKeys returns the keys for the given user from the temporary key map
func (m *UserMap) LookupKeys(uid ID) (*NodeKeys, bool) {
	nk, t := m.keysLookup[uid]
	return nk, t
}

func (m *UserMap) GetContactList() (ids []ID, nicks []string) {
	ids = make([]ID, len(m.userCollection))
	nicks = make([]string, len(m.userCollection))

	index := uint64(0)
	for _, user := range m.userCollection {
		ids[index] = user.UserID
		nicks[index] = user.Nick
		index++
	}
	return ids, nicks
}
