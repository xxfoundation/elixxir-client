////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"crypto/sha256"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the Registry interface
func TestUserSession(t *testing.T) {

	test := 11
	pass := 0

	u := new(User)
	// This is 65 so you can see the letter A in the gob if you need to make
	// sure that the gob contains the user ID
	UID := uint64(65)

	u.User = id.NewUserFromUint(UID, t)
	u.Nick = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2), large.NewInt(5))

	keys := make([]NodeKeys, 1)
	keys[0] = NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	}

	err := globals.InitStorage(&globals.RamStorage{}, "")

	if err != nil {
		t.Errorf("User Session: Local storage could not be created: %s", err.Error())
	}

	rng := rand.New(rand.NewSource(42))
	params := signature.NewDSAParams(rng, signature.L3072N256)
	privateKey := params.PrivateKeyGen(rng)
	publicKey := privateKey.PublicKeyGen()
	ses := NewSession(u, "abc", keys, publicKey, privateKey, grp)

	ses.SetLastMessageID("totally unique ID")

	err = ses.StoreSession()

	if err != nil {
		t.Errorf("Error: Session not stored correctly: %s", err.Error())
	}

	ses.Immolate()

	//TODO: write test which validates the immolation

	if TheSession != nil {
		t.Errorf("Error: The session wasn't nil after immolation!")
	} else {
		pass++
	}

	_, err = LoadSession(id.NewUserFromUint(UID, t))

	if err != nil {
		t.Errorf("Error: Unable to login with valid user: %v", err.Error())
	} else {
		pass++
	}

	_, err = LoadSession(id.NewUserFromUint(10002, t))

	if err == nil {
		t.Errorf("Error: Able to login with invalid user!")
	} else {
		pass++
	}

	if TheSession == nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	} else {
		pass++
	}

	if TheSession.GetGWAddress() == "" {
		t.Errorf("Error: Node Address not set correctly with Regestration!")
	} else {
		pass++
	}

	TheSession.SetGWAddress("test")

	if TheSession.GetGWAddress() != "test" {
		t.Errorf("Error: Node Address not set correctly with SetNodeAddress!")
	} else {
		pass++
	}

	if TheSession.GetLastMessageID() != "totally unique ID" {
		t.Errorf("Error: Last message ID should have been stored and loaded")
	} else {
		pass++
	}

	TheSession.SetLastMessageID("test")

	if TheSession.GetLastMessageID() != "test" {
		t.Errorf("Error: Last message ID not set correctly with" +
			" SetLastMessageID!")
	} else {
		pass++
	}

	if TheSession.GetKeys() == nil {
		t.Errorf("Error: Keys not set correctly!")
	} else {

		test += len(TheSession.GetKeys())

		for i := 0; i < len(TheSession.GetKeys()); i++ {

			if !reflect.DeepEqual(*TheSession.GetPublicKey(), *publicKey) {
				t.Errorf("Error: Public key not set correctly!")
			} else if !reflect.DeepEqual(*TheSession.GetPrivateKey(), *privateKey) {
				t.Errorf("Error: Private key not set correctly!")
			} else if TheSession.GetKeys()[i].ReceptionKey.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Error: Reception key not set correctly!")
			} else if TheSession.GetKeys()[i].TransmissionKey.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Error: Transmission key not set correctly!")
			}

			pass++
		}
	}

	//TODO: FIX THIS?
	if TheSession.GetPrivateKey() == nil {
		t.Errorf("Error: Private Keys not set correctly!")
	} else {
		pass++
	}

	err = TheSession.UpsertMap("test", 5)

	if err != nil {
		t.Errorf("Error: Could not store in session map interface: %s",
			err.Error())
	}

	element, err := TheSession.QueryMap("test")

	if err != nil {
		t.Errorf("Error: Could not read element in session map "+
			"interface: %s", err.Error())
	}

	if element.(int) != 5 {
		t.Errorf("Error: Could not read element in session map "+
			"interface: Expected: 5, Recieved: %v", element)
	}

	TheSession.DeleteMap("test")

	_, err = TheSession.QueryMap("test")

	if err == nil {
		t.Errorf("Error: Could not delete element in session map " +
			"interface")
	}

	//Logout
	TheSession.Immolate()

	if TheSession != nil {
		t.Errorf("Error: Logout / Immolate did not work!")
	} else {
		pass++
	}

	// Error tests

	// Test nil LocalStorage
	temp := globals.LocalStorage
	globals.LocalStorage = nil

	_, err = LoadSession(id.NewUserFromUint(6, t))

	if err == nil {
		t.Errorf("Error did not catch a nil LocalStorage")
	}
	globals.LocalStorage = temp

	// Test invalid / corrupted LocalStorage
	h := sha256.New()
	h.Write([]byte(string(20000)))
	randBytes := h.Sum(nil)
	globals.LocalStorage.Save(randBytes)

	_, err = LoadSession(id.NewUserFromUint(6, t))

	if err == nil {
		t.Errorf("Error did not catch a corrupt LocalStorage")
	}
}

func TestGetPubKey(t *testing.T) {
	u := new(User)
	UID := id.NewUserFromUint(1, t)

	u.User = UID
	u.Nick = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2), large.NewInt(5))

	keys := make([]NodeKeys, 1)
	keys[0] = NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	}

	rng := rand.New(rand.NewSource(42))
	params := signature.NewDSAParams(rng, signature.L3072N256)
	privateKey := params.PrivateKeyGen(rng)
	publicKey := privateKey.PublicKeyGen()
	ses := NewSession(u, "abc", keys, publicKey, privateKey, grp)

	pubKey := ses.GetPublicKey()
	if !reflect.DeepEqual(pubKey, publicKey) {
		t.Errorf("Public key not returned correctly!")
	}
}

func TestGetPrivKey(t *testing.T) {
	u := new(User)
	UID := id.NewUserFromUint(1, t)

	u.User = UID
	u.Nick = "Mario"

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2), large.NewInt(5))

	keys := make([]NodeKeys, 1)
	keys[0] = NodeKeys{
		TransmissionKey: grp.NewInt(2),
		ReceptionKey:    grp.NewInt(2),
	}

	rng := rand.New(rand.NewSource(42))
	params := signature.NewDSAParams(rng, signature.L3072N256)
	privateKey := params.PrivateKeyGen(rng)
	publicKey := privateKey.PublicKeyGen()
	ses := NewSession(u, "abc", keys, publicKey, privateKey, grp)

	privKey := ses.GetPrivateKey()
	if !reflect.DeepEqual(*privKey, *privateKey) {
		t.Errorf("Private key is not returned correctly!")
	}
}
