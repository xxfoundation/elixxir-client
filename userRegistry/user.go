////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package userRegistry

import (
	"crypto/sha256"
	"encoding/binary"
	"gitlab.com/elixxir/client/globals"
	user2 "gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// Globally instantiated Registry
var Users Registry

const NumDemoUsers = 40

var DemoUserNicks = []string{"David", "Payments", "UDB", "Jim", "Ben", "Steph",
	"Rick", "Jake", "Niamh", "Stephanie", "Mario", "Jono", "Amanda",
	"Margaux", "Kevin", "Bruno", "Konstantino", "Bernardo", "Tigran",
	"Kate", "Will", "Katie", "Bryan"}
var DemoChannelNames = []string{"#General", "#Engineering", "#Lunch",
	"#Random"}

func InitUserRegistry(grp *cyclic.Group) {
	Users = newRegistry(grp)
}

// Interface for User Registry operations
type Registry interface {
	NewUser(id *id.ID, nickname string) *user2.User
	DeleteUser(id *id.ID)
	GetUser(id *id.ID) (user *user2.User, ok bool)
	UpsertUser(user *user2.User)
	CountUsers() int
	LookupUser(hid string) (uid *id.ID, ok bool)
	LookupKeys(uid *id.ID) (*user.NodeKeys, bool)
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[id.ID]*user2.User
	// Increments sequentially for User.ID values
	idCounter uint64
	// Temporary map acting as a lookup table for demo user registration codes
	// Key type is string because keys must implement == and []byte doesn't
	userLookup map[string]*id.ID
	//Temporary placed to store the keys for each user
	keysLookup map[id.ID]*user.NodeKeys
}

// newRegistry creates a new Registry interface
func newRegistry(grp *cyclic.Group) Registry {
	if len(DemoChannelNames) > 10 || len(DemoUserNicks) > 30 {
		globals.Log.ERROR.Print("Not enough demo users have been hardcoded.")
	}
	userUserIdMap := make(map[id.ID]*user2.User)
	userRegCodeMap := make(map[string]*id.ID)
	nk := make(map[id.ID]*user.NodeKeys)

	// Deterministically create NumDemoUsers users
	// TODO Replace this with real user registration/discovery
	for i := uint64(1); i <= NumDemoUsers; i++ {
		currentID := new(id.ID)
		binary.BigEndian.PutUint64(currentID[:], i)
		currentID.SetType(id.User)
		newUsr := new(user2.User)
		nodeKey := new(user.NodeKeys)

		// Generate user parameters
		newUsr.User = currentID
		newUsr.Precan = true
		// TODO We need a better way to generate base/recursive keys
		h := sha256.New()
		h.Write([]byte(string(40000 + i)))
		nodeKey.TransmissionKey = grp.NewIntFromBytes(h.Sum(nil))
		h = sha256.New()
		h.Write([]byte(string(60000 + i)))
		nodeKey.ReceptionKey = grp.NewIntFromBytes(h.Sum(nil))

		// Add user to collection and lookup table
		userUserIdMap[*newUsr.User] = newUsr
		// Detect collisions in the registration code
		if _, ok := userRegCodeMap[RegistrationCode(newUsr.User)]; ok {
			globals.Log.ERROR.Printf(
				"Collision in demo user list creation at %v. "+
					"Please fix ASAP (include more bits to the reg code.", i)
		}
		userRegCodeMap[RegistrationCode(newUsr.User)] = newUsr.User
		nk[*newUsr.User] = nodeKey
	}

	// Channels have been hardcoded to users starting with 31
	for i := 0; i < len(DemoUserNicks); i++ {
		currentID := new(id.ID)
		binary.BigEndian.PutUint64(currentID[:], uint64(i)+1)
		currentID.SetType(id.User)
		userUserIdMap[*currentID].Username = DemoUserNicks[i]
	}

	for i := 0; i < len(DemoChannelNames); i++ {
		currentID := new(id.ID)
		binary.BigEndian.PutUint64(currentID[:], uint64(i)+31)
		currentID.SetType(id.User)
		userUserIdMap[*currentID].Username = DemoChannelNames[i]
	}

	// With an underlying UserMap data structure
	return Registry(&UserMap{userCollection: userUserIdMap,
		idCounter:  uint64(NumDemoUsers),
		userLookup: userRegCodeMap,
		keysLookup: nk})
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(id *id.ID, username string) *user2.User {
	return &user2.User{User: id, Username: username}
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id *id.ID) (user *user2.User, ok bool) {
	user, ok = m.userCollection[*id]
	user = user.DeepCopy()
	return
}

// DeleteContactKeys deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id *id.ID) {
	// If key does not exist, do nothing
	delete(m.userCollection, *id)
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *user2.User) {
	m.userCollection[*user.User] = user
}

// CountUsers returns a count of the users in userCollection
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
}

// LookupUser returns the user id corresponding to the demo registration code
func (m *UserMap) LookupUser(hid string) (*id.ID, bool) {
	uid, ok := m.userLookup[hid]
	return uid, ok
}

// LookupKeys returns the keys for the given user from the temporary key map
func (m *UserMap) LookupKeys(uid *id.ID) (*user.NodeKeys, bool) {
	nk, t := m.keysLookup[*uid]
	return nk, t
}
