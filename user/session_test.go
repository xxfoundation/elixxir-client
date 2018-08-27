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
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the Registry interface
func TestUserSession(t *testing.T) {

	test := 11
	pass := 0

	u := new(User)
	UID := ID(1)

	u.UserID = UID
	u.Nick = "Mario"

	keys := make([]NodeKeys, 1)
	keys[0] = NodeKeys{
		PublicKey:        cyclic.NewInt(2),
		TransmissionKeys: RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReceptionKeys:    RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReceiptKeys:      RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReturnKeys:       RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
	}

	err := globals.InitStorage(&globals.RamStorage{}, "")

	if err != nil {
		t.Errorf("User Session: Local storage could not be created: %s", err.Error())
	}

	//Ask Ben if there should be a Node Address here!
	ses := NewSession(u, "abc", keys)

	ses.(*SessionObj).PrivateKey.SetInt64(2)
	ses.SetLastMessageID("totally unique ID")

	err = ses.StoreSession()

	if err != nil {
		t.Errorf("Error: Session not stored correctly: %s", err.Error())
	}

	ses.Immolate()

	//TODO: write test which validates the immolation

	if TheSession != nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	} else {
		pass++
	}

	_, err = LoadSession(UID)

	if err != nil {
		t.Errorf("Error: Unable to login with valid user!")
	} else {
		pass++
	}

	_, err = LoadSession(ID(10002))

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

			if TheSession.GetKeys()[i].PublicKey.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Public key not set correctly!")
			} else if TheSession.GetKeys()[i].ReceiptKeys.Base.Cmp(cyclic.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if TheSession.GetKeys()[i].ReceiptKeys.Recursive.Cmp(cyclic.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if TheSession.GetKeys()[i].ReceptionKeys.Base.Cmp(cyclic.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if TheSession.GetKeys()[i].ReceptionKeys.Recursive.Cmp(
				cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if TheSession.GetKeys()[i].ReturnKeys.Base.Cmp(cyclic.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if TheSession.GetKeys()[i].ReturnKeys.Recursive.Cmp(cyclic.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if TheSession.GetKeys()[i].TransmissionKeys.Base.Cmp(cyclic.
				NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if TheSession.GetKeys()[i].TransmissionKeys.Recursive.Cmp(
				cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
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

	_, err = LoadSession(ID(6))

	if err == nil {
		t.Errorf("Error did not catch a nil LocalStorage")
	}
	globals.LocalStorage = temp

	// Test invalid / corrupted LocalStorage
	h := sha256.New()
	h.Write([]byte(string(20000)))
	randBytes := h.Sum(nil)
	globals.LocalStorage.Save(randBytes)

	_, err = LoadSession(ID(6))

	if err == nil {
		t.Errorf("Error did not catch a corrupt LocalStorage")
	}
}

func TestGetPubKey(t *testing.T) {
	u := new(User)
	UID := ID(1)

	u.UserID = UID
	u.Nick = "Mario"

	keys := make([]NodeKeys, 1)
	keys[0] = NodeKeys{
		PublicKey:        cyclic.NewInt(2),
		TransmissionKeys: RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReceptionKeys:    RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReceiptKeys:      RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReturnKeys:       RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
	}

	ses := NewSession(u, "abc", keys)
	pubKey := ses.GetPublicKey()
	if pubKey.Cmp(cyclic.NewMaxInt()) != 0 {
		t.Errorf("Public key is not set to max int!")
	}
}
