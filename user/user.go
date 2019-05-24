////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
)

const NUM_DEMO_USERS = 40

var DemoUserNicks = []string{"David", "Payments", "UDB", "Jim", "Ben", "Steph",
	"Rick", "Jake", "Spencer", "Stephanie", "Mario", "Jono", "Amanda",
	"Margaux", "Kevin", "Bruno", "Konstantino", "Bernardo", "Tigran",
	"Kate", "Will", "Katie", "Bryan"}
var DemoChannelNames = []string{"#General", "#Engineering", "#Lunch",
	"#Random"}

// Struct representing a User in the system
type User struct {
	User *id.User
	Nick string
}

// DeepCopy performs a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	nu := new(User)
	nu.User = u.User
	nu.Nick = u.Nick
	return nu
}

// NewUser creates a new User object with default fields and given address.
func NewUser(id *id.User, nickname string) *User {
	return &User{User: id, Nick: nickname}
}

// Interface for User Registry operations
type Registry interface {
	UpsertUser(user *User)
	DeleteUser(id *id.User)
	GetUser(id *id.User) (user *User, ok bool)
	GetUserList() []*User
	LookupUser(hid string) (uid *id.User, ok bool)
	LookupKeys(uid *id.User) (*NodeKeys, bool)
	GobEncode() ([]byte, error)
	GobDecode(in []byte) error
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[id.User]*User
	// Temporary map acting as a lookup table for demo user registration codes
	// Key type is string because keys must implement == and []byte doesn't
	userLookup map[string]*id.User
	//Temporary placed to store the keys for each user
	keysLookup map[id.User]*NodeKeys
}

// NewRegistry creates a new Registry interface
func NewRegistry(grp *cyclic.Group) Registry {
	if len(DemoChannelNames) > 10 || len(DemoUserNicks) > 30 {
		globals.Log.ERROR.Print("Not enough demo users have been hardcoded.")
	}
	uc := make(map[id.User]*User)
	ul := make(map[string]*id.User)
	nk := make(map[id.User]*NodeKeys)

	// Deterministically create NUM_DEMO_USERS users
	// TODO Replace this with real user registration/discovery
	for i := uint64(1); i <= NUM_DEMO_USERS; i++ {
		currentID := id.NewUserFromUints(&[4]uint64{0, 0, 0, i})
		t := new(User)
		k := new(NodeKeys)

		// Generate user parameters
		t.User = currentID
		currentID.RegistrationCode()
		// TODO We need a better way to generate base/recursive keys
		h := sha256.New()
		h.Write([]byte(string(20000 + i)))
		k.TransmissionKey = grp.NewIntFromBytes(h.Sum(nil))
		h = sha256.New()
		h.Write([]byte(string(40000 + i)))
		k.ReceptionKey = grp.NewIntFromBytes(h.Sum(nil))

		// Add user to collection and lookup table
		uc[*t.User] = t
		// Detect collisions in the registration code
		if _, ok := ul[t.User.RegistrationCode()]; ok {
			globals.Log.ERROR.Printf(
				"Collision in demo user list creation at %v. "+
					"Please fix ASAP (include more bits to the reg code.", i)
		}
		ul[t.User.RegistrationCode()] = t.User
		nk[*t.User] = k
	}

	// Channels have been hardcoded to users starting with 31
	for i := 0; i < len(DemoUserNicks); i++ {
		currentID := id.NewUserFromUints(&[4]uint64{0, 0, 0, uint64(i) + 1})
		uc[*currentID].Nick = DemoUserNicks[i]
	}

	for i := 0; i < len(DemoChannelNames); i++ {
		currentID := id.NewUserFromUints(&[4]uint64{0, 0, 0, uint64(i) + 31})
		uc[*currentID].Nick = DemoChannelNames[i]
	}

	// With an underlying UserMap data structure
	return Registry(&UserMap{userCollection: uc,
		userLookup: ul,
		keysLookup: nk})
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id *id.User) (user *User, ok bool) {
	user, ok = m.userCollection[*id]
	user = user.DeepCopy()
	return
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id *id.User) {
	// If key does not exist, do nothing
	delete(m.userCollection, *id)
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	m.userCollection[*user.User] = user
}

// LookupUser returns the user id corresponding to the demo registration code
func (m *UserMap) LookupUser(hid string) (*id.User, bool) {
	uid, ok := m.userLookup[hid]
	return uid, ok
}

// LookupKeys returns the keys for the given user from the temporary key map
func (m *UserMap) LookupKeys(uid *id.User) (*NodeKeys, bool) {
	nk, t := m.keysLookup[*uid]
	return nk, t
}

// Get a slice of all Users in UserMap
func (m *UserMap) GetUserList() []*User {
	list := make([]*User, 0, len(m.userCollection))
	for _, u := range m.userCollection {
		list = append(list, u)
	}
	return list
}

// GobEncode the UserMap to bytes
func (m *UserMap) GobEncode() ([]byte, error) {
	var buf bytes.Buffer

	// Create new encoder that will transmit the buffer
	enc := gob.NewEncoder(&buf)

	// Transmit the user collection map
	err := enc.Encode(m.userCollection)

	if err != nil {
		return nil, err
	}

	// Transmit the user lookup map
	err = enc.Encode(m.userLookup)

	if err != nil {
		return nil, err
	}

	// Transmit the keys map
	err = enc.Encode(m.keysLookup)
	return buf.Bytes(), err
}

// GobDecode the UserMap from bytes
func (m *UserMap) GobDecode(in []byte) error {
	var buf bytes.Buffer

	// Write bytes to the buffer
	buf.Write(in)

	// Create new decoder that reads from the buffer
	dec := gob.NewDecoder(&buf)

	// Decode user collection map
	err := dec.Decode(&m.userCollection)

	if err != nil {
		return err
	}

	// Decode user lookup map
	err = dec.Decode(&m.userLookup)

	if err != nil {
		return err
	}

	// Decode keys map
	return dec.Decode(&m.keysLookup)
}
