////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/xeipuuv/gojsonschema"
	"gitlab.com/privategrity/crypto/cyclic"
	"github.com/spf13/jwalterweatherman"
)

var ContactListJsonSchema = `{
	"type": "array",
	"items": {
		"type": "object",
		"properties": {
			"UserID": { "type": "number" },
			"Nick": { "type": "string" }
		}
	}
}`

// Globally instantiated UserRegistry
var Users = newUserRegistry()
var NUM_DEMO_USERS = int(10)

// Interface for User Registry operations
type UserRegistry interface {
	GetUser(id uint64) (user *User, ok bool)
	CountUsers() int
	LookupUser(hid uint64) (uid uint64, ok bool)
	LookupKeys(uid uint64) (*NodeKeys, bool)
	UpsertUser(user *User)
	GetContactListJSON() ([]byte, error)
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

// Creates a new UserRegistry interface
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
		t.UserID = uint64(i)
		h.Write([]byte(string(20000 + i)))
		k.TransmissionKeys.Base = cyclic.NewIntFromString(
			"c1248f42f8127999e07c657896a26b56fd9a499c6199e1265053132451128f52", 16)
		h.Write([]byte(string(30000 + i)))
		k.TransmissionKeys.Recursive = cyclic.NewIntFromString(
			"ad333f4ccea0ccf2afcab6c1b9aa2384e561aee970046e39b7f2a78c3942a251", 16)
		h.Write([]byte(string(40000 + i)))
		k.ReceptionKeys.Base = cyclic.NewIntFromString(
			"83120e7bfaba497f8e2c95457a28006f73ff4ec75d3ad91d27bf7ce8f04e772c", 16)
		h.Write([]byte(string(50000 + i)))
		k.ReceptionKeys.Recursive = cyclic.NewIntFromString(
			"979e574166ef0cd06d34e3260fe09512b69af6a414cf481770600d9c7447837b", 16)
		// Add user to collection and lookup table
		uc[t.UserID] = t
		ul[UserHash(t.UserID)] = t.UserID
		nk[t.UserID] = k
	}

	uc[1].Nick = "David"
	uc[2].Nick = "Jim"
	uc[3].Nick = "Ben"
	uc[4].Nick = "Rick"
	uc[5].Nick = "Spencer"
	uc[6].Nick = "Jake"
	uc[7].Nick = "Mario"
	uc[8].Nick = "Will"
	uc[9].Nick = "Sydney"
	uc[10].Nick = "Jono"

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

func UserHash(uid uint64) uint64 {
	return uid + 10000
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

func (m *UserMap) UpsertUser(user *User) {
	m.userCollection[user.UserID] = user
}

// Looks up the user id corresponding to the demo registration code
func (m *UserMap) LookupUser(hid uint64) (uid uint64, ok bool) {
	uid, ok = m.userLookup[hid]
	return
}

func (m *UserMap) LookupKeys(uid uint64) (*NodeKeys, bool) {
	nk, t := m.keysLookup[uid]
	return nk, t
}

func (m *UserMap) buildContactListJSON() ([]byte, error) {
	var result []byte
	result = append(result, '[')
	for _, user := range m.userCollection {
		nextChunk, err := json.Marshal(user)

		if err != nil {
			jwalterweatherman.ERROR.Println(err.Error())
			return nil, err
		}

		result = append(result, nextChunk...)
		result = append(result, ',')
	}
	// replace the last byte with a bracket, ending the list
	result[len(result)-1] = ']'

	return result, nil
}

func (m *UserMap) GetContactListJSON() ([]byte, error) {
	contactListSchema, err := gojsonschema.NewSchema(gojsonschema.
		NewStringLoader(ContactListJsonSchema))
	if err != nil {
		jwalterweatherman.ERROR.Printf(
			"Couldn't instantiate JSON schema: %v", err.Error())
		return nil, err
	}

	result, err := m.buildContactListJSON()
	if err != nil {
		return nil, err
	}

	jsonLoader := gojsonschema.NewStringLoader(string(result))
	valid, err := contactListSchema.Validate(jsonLoader)

	if err != nil {
		jwalterweatherman.ERROR.Printf(
			"Failed to validate JSON: %v", err.Error())
		return nil, err
	}
	if !valid.Valid() {
		jwalterweatherman.ERROR.Println("The produced JSON wasn't valid")
		for _, validationError := range (valid.Errors()) {
			jwalterweatherman.ERROR.Println(validationError.String())
		}
	}

	return result, nil
}
