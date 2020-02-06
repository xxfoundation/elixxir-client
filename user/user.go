////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"crypto/sha256"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
)

// Globally instantiated Registry
var Users Registry

const NumDemoUsers = 40

var DemoUserNicks = []string{"David", "Payments", "UDB", "Jim", "Ben", "Steph",
	"Rick", "Jake", "Spencer", "Stephanie", "Mario", "Jono", "Amanda",
	"Margaux", "Kevin", "Bruno", "Konstantino", "Bernardo", "Tigran",
	"Kate", "Will", "Katie", "Bryan"}
var DemoChannelNames = []string{"#General", "#Engineering", "#Lunch",
	"#Random"}

func InitUserRegistry(grp *cyclic.Group) {
	Users = newRegistry(grp)
}

// Interface for User Registry operations
type Registry interface {
	NewUser(id *id.User, nickname string) *User
	DeleteUser(id *id.User)
	GetUser(id *id.User) (user *User, ok bool)
	UpsertUser(user *User)
	CountUsers() int
	LookupUser(hid string) (uid *id.User, ok bool)
	LookupKeys(uid *id.User) (*NodeKeys, bool)
}

type UserMap struct {
	// Map acting as the User Registry containing User -> ID mapping
	userCollection map[id.User]*User
	// Increments sequentially for User.ID values
	idCounter uint64
	// Temporary map acting as a lookup table for demo user registration codes
	// Key type is string because keys must implement == and []byte doesn't
	userLookup map[string]*id.User
	//Temporary placed to store the keys for each user
	keysLookup map[id.User]*NodeKeys
}

// newRegistry creates a new Registry interface
func newRegistry(grp *cyclic.Group) Registry {
	if len(DemoChannelNames) > 10 || len(DemoUserNicks) > 30 {
		globals.Log.ERROR.Print("Not enough demo users have been hardcoded.")
	}
	userUserIdMap := make(map[id.User]*User)
	userRegCodeMap := make(map[string]*id.User)
	nk := make(map[id.User]*NodeKeys)

	// Deterministically create NumDemoUsers users
	// TODO Replace this with real user registration/discovery
	for i := uint64(1); i <= NumDemoUsers; i++ {
		currentID := id.NewUserFromUints(&[4]uint64{0, 0, 0, i})
		newUsr := new(User)
		nodeKey := new(NodeKeys)

		// Generate user parameters
		newUsr.User = currentID
		newUsr.Precan = true
		currentID.RegistrationCode()
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
		if _, ok := userRegCodeMap[newUsr.User.RegistrationCode()]; ok {
			globals.Log.ERROR.Printf(
				"Collision in demo user list creation at %v. "+
					"Please fix ASAP (include more bits to the reg code.", i)
		}
		userRegCodeMap[newUsr.User.RegistrationCode()] = newUsr.User
		nk[*newUsr.User] = nodeKey
	}

	// Channels have been hardcoded to users starting with 31
	for i := 0; i < len(DemoUserNicks); i++ {
		currentID := id.NewUserFromUints(&[4]uint64{0, 0, 0, uint64(i) + 1})
		userUserIdMap[*currentID].Username = DemoUserNicks[i]
	}

	for i := 0; i < len(DemoChannelNames); i++ {
		currentID := id.NewUserFromUints(&[4]uint64{0, 0, 0, uint64(i) + 31})
		userUserIdMap[*currentID].Username = DemoChannelNames[i]
	}

	// With an underlying UserMap data structure
	return Registry(&UserMap{userCollection: userUserIdMap,
		idCounter:  uint64(NumDemoUsers),
		userLookup: userRegCodeMap,
		keysLookup: nk})
}

// Struct representing a User in the system
type User struct {
	User     *id.User
	Username string
	Precan   bool
}

// DeepCopy performs a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	nu := new(User)
	nu.User = u.User
	nu.Username = u.Username
	nu.Precan = u.Precan
	return nu
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(id *id.User, username string) *User {
	return &User{User: id, Username: username}
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserMap) GetUser(id *id.User) (user *User, ok bool) {
	user, ok = m.userCollection[*id]
	user = user.DeepCopy()
	return
}

// DeleteContactKeys deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id *id.User) {
	// If key does not exist, do nothing
	delete(m.userCollection, *id)
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	m.userCollection[*user.User] = user
}

// CountUsers returns a count of the users in userCollection
func (m *UserMap) CountUsers() int {
	return len(m.userCollection)
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
