////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/sha256"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
func TestUserSession(t *testing.T) {

	test := 9
	pass := 0

	u := new(User)
	UID := uint64(1)

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

	err := InitStorage(&RamStorage{}, "")

	if err != nil {
		t.Errorf("User Session: Local storage could not be created: %s", err.Error())
	}

	//Ask Ben if there should be a Node Address here!
	ses := NewUserSession(u, "abc", "", keys)

	ses.(*SessionObj).PrivateKey.SetInt64(2)

	ses.StoreSession()

	ses.Immolate()

	//TODO: write test which validates the immolation

	if Session != nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	} else {
		pass++
	}

	if LoadSession(UID, nil) != nil {
		t.Errorf("Error: Unable to login with valid user!")
	} else {
		pass++
	}

	if LoadSession(100002, nil) == nil {
		t.Errorf("Error: Able to login with invalid user!")
	} else {
		pass++
	}

	if Session == nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	} else {
		pass++
	}

	if Session.GetNodeAddress() == "" {
		t.Errorf("Error: Node Address not set correctly with Regestration!")
	} else {
		pass++
	}

	Session.SetNodeAddress("test")

	if Session.GetNodeAddress() != "test" {
		t.Errorf("Error: Node Address not set correctly with SetNodeAddress!")
	} else {
		pass++
	}

	if Session.GetKeys() == nil {
		t.Errorf("Error: Keys not set correctly!")
	} else {

		test += len(Session.GetKeys())

		for i := 0; i < len(Session.GetKeys()); i++ {

			if Session.GetKeys()[i].PublicKey.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Public key not set correctly!")
			} else if Session.GetKeys()[i].ReceiptKeys.Base.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReceiptKeys.Recursive.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReceptionKeys.Base.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReceptionKeys.Recursive.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReturnKeys.Base.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReturnKeys.Recursive.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].TransmissionKeys.Base.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].TransmissionKeys.Recursive.Cmp(cyclic.NewInt(2)) != 0 {
				t.Errorf("Error: Receipt base key not set correctly!")
			}

			pass++
		}
	}

	//TODO: FIX THIS?
	if Session.GetPrivateKey() == nil {
		t.Errorf("Error: Private Keys not set correctly!")
	} else {
		pass++
	}

	inmsg, err := format.NewMessage(42, 69, "test")
	if err != nil {
		t.Errorf("Error: Couldn't create new message%v", err.Error())
	}

	Session.PushFifo(&inmsg[0])

	outmsg, _ := Session.PopFifo()

	if &inmsg[0] != outmsg {
		t.Errorf("Error: Incorrect Return Message from fifo")
	} else {
		pass++
	}

	// Test immolate on nonempty FIFO
	inmsg, err = format.NewMessage(42, 69, "test")
	if err != nil {
		t.Errorf("Error: Couldn't create new message%v", err.Error())
	}

	Session.PushFifo(&inmsg[0])

	//Logout
	Session.Immolate()

	if Session != nil {
		t.Errorf("Error: Logout / Immolate did not work!")
	} else {
		pass++
	}

	// Error tests

	// Test nil LocalStorage
	temp := LocalStorage
	LocalStorage = nil
	if LoadSession(6, nil) == nil {
		t.Errorf("Error did not catch a nil LocalStorage")
	}
	LocalStorage = temp

	// Test invalid / corrupted LocalStorage
	h := sha256.New()
	h.Write([]byte(string(20000)))
	randBytes := h.Sum(nil)
	LocalStorage.Save(randBytes)
	if LoadSession(6, nil) == nil {
		t.Errorf("Error did not catch a corrupt LocalStorage")
	}
}
