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
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
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

	grp := cyclic.NewGroup(large.NewInt(1000), large.NewInt(2), large.NewInt(5))

	keys := make([]NodeKeys, 1)
	keys[0] = NodeKeys{
		TransmissionKeys: RatchetKey{grp.NewInt(2), grp.NewInt(2)},
		ReceptionKeys:    RatchetKey{grp.NewInt(2), grp.NewInt(2)},
		ReceiptKeys:      RatchetKey{grp.NewInt(2), grp.NewInt(2)},
		ReturnKeys:       RatchetKey{grp.NewInt(2), grp.NewInt(2)},
	}

	// Storage
	storage := &globals.RamStorage{}

	//Ask Ben if there should be a Node Address here!
	ses := NewSession(storage,
	u, "abc", keys, grp.NewInt(2), grp)

	ses.(*SessionObj).PrivateKey = grp.NewInt(2)
	ses.SetLastMessageID("totally unique ID")

	err := ses.StoreSession()

	if err != nil {
		t.Errorf("Error: Session not stored correctly: %s", err.Error())
	}

	ses.Immolate()

	//TODO: write test which validates the immolation

	ses, err = LoadSession(storage, id.NewUserFromUint(UID, t))

	if err != nil {
		t.Errorf("Error: Unable to login with valid user: %v", err.Error())
	} else {
		pass++
	}
	
	_, err = LoadSession(storage,
		id.NewUserFromUint(10002, t))

	if err == nil {
		t.Errorf("Error: Able to login with invalid user!")
	} else {
		pass++
	}

	if ses == nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	} else {
		pass++
	}

	if ses.GetGWAddress() == "" {
		t.Errorf("Error: Node Address not set correctly with Regestration!")
	} else {
		pass++
	}

	ses.SetGWAddress("test")

	if ses.GetGWAddress() != "test" {
		t.Errorf("Error: Node Address not set correctly with SetNodeAddress!")
	} else {
		pass++
	}

	if ses.GetLastMessageID() != "totally unique ID" {
		t.Errorf("Error: Last message ID should have been stored and loaded")
	} else {
		pass++
	}

	ses.SetLastMessageID("test")

	if ses.GetLastMessageID() != "test" {
		t.Errorf("Error: Last message ID not set correctly with" +
			" SetLastMessageID!")
	} else {
		pass++
	}

	if ses.GetKeys() == nil {
		t.Errorf("Error: Keys not set correctly!")
	} else {

		test += len(ses.GetKeys())

		for i := 0; i < len(ses.GetKeys()); i++ {

			if ses.GetPublicKey().Cmp(grp.NewInt(2)) != 0 {
				t.Errorf("Error: Public key not set correctly!")
			} else if ses.GetKeys()[i].ReceiptKeys.Base.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if ses.GetKeys()[i].ReceiptKeys.Recursive.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if ses.GetKeys()[i].ReceptionKeys.Base.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if ses.GetKeys()[i].ReceptionKeys.Recursive.Cmp(
				grp.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if ses.GetKeys()[i].ReturnKeys.Base.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if ses.GetKeys()[i].ReturnKeys.Recursive.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if ses.GetKeys()[i].TransmissionKeys.Base.Cmp(grp.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if ses.GetKeys()[i].TransmissionKeys.Recursive.Cmp(
				grp.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			}

			pass++
		}
	}

	//TODO: FIX THIS?
	if ses.GetPrivateKey() == nil {
		t.Errorf("Error: Private Keys not set correctly!")
	} else {
		pass++
	}

	err = ses.UpsertMap("test", 5)

	if err != nil {
		t.Errorf("Error: Could not store in session map interface: %s",
			err.Error())
	}

	element, err := ses.QueryMap("test")

	if err != nil {
		t.Errorf("Error: Could not read element in session map "+
			"interface: %s", err.Error())
	}

	if element.(int) != 5 {
		t.Errorf("Error: Could not read element in session map "+
			"interface: Expected: 5, Recieved: %v", element)
	}

	ses.DeleteMap("test")

	_, err = ses.QueryMap("test")

	if err == nil {
		t.Errorf("Error: Could not delete element in session map " +
			"interface")
	}

	//Logout
	ses.Immolate()

	// Error tests

	// Test nil LocalStorage

	_, err = LoadSession(nil, id.NewUserFromUint(6, t))

	if err == nil {
		t.Errorf("Error did not catch a nil LocalStorage")
	}

	// Test invalid / corrupted LocalStorage
	h := sha256.New()
	h.Write([]byte(string(20000)))
	randBytes := h.Sum(nil)
	storage.Save(randBytes)

	_, err = LoadSession(storage, id.NewUserFromUint(6, t))

	if err == nil {
		t.Errorf("Error did not catch a corrupt LocalStorage")
	}
}

func TestGetPubKey(t *testing.T) {
	u := new(User)
	UID := id.NewUserFromUint(1, t)

	u.User = UID
	u.Nick = "Mario"

	grp := cyclic.NewGroup(large.NewInt(1000), large.NewInt(0), large.NewInt(0))

	keys := make([]NodeKeys, 1)
	keys[0] = NodeKeys{
		TransmissionKeys: RatchetKey{grp.NewInt(2), grp.NewInt(2)},
		ReceptionKeys:    RatchetKey{grp.NewInt(2), grp.NewInt(2)},
		ReceiptKeys:      RatchetKey{grp.NewInt(2), grp.NewInt(2)},
		ReturnKeys:       RatchetKey{grp.NewInt(2), grp.NewInt(2)},
	}

	ses := NewSession(nil, u, "abc", keys, grp.NewInt(2), grp)
	pubKey := ses.GetPublicKey()
	if pubKey.Cmp(grp.NewInt(2)) != 0 {
		t.Errorf("Public key is not set correctly!")
	}
}
