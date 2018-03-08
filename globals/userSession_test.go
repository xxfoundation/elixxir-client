////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"testing"
	"gitlab.com/privategrity/crypto/cyclic"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
func TestUserSession(t *testing.T) {

	test := 8
	pass := 0

	u := new(User)
	UID := uint64(1)

	u.UID = UID
	u.Nick = "Mario"

	keys := make([]NodeKeys, 1)
	keys[0] = NodeKeys{
		PublicKey: cyclic.NewInt(2),
		TransmissionKeys: 	RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReceptionKeys: 		RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReceiptKeys: 		RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
		ReturnKeys: 		RatchetKey{cyclic.NewInt(2), cyclic.NewInt(2)},
	}

	success := InitStorage(RamStorage{},"")

	if !success{
		t.Errorf("User Session: Local storage could not be created")
	}

	//Ask Ben if there should be a Node Address here!
	ses := NewUserSession(u, "abc", keys)

	ses.(*sessionObj).PrivateKey.SetInt64(2)

	ses.StoreSession()

	ses.Immolate()

	//TODO: write test which validates the immolation

	if Session!= nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	} else{
		pass++
	}

	if !LoadSession(UID, nil) {
		t.Errorf("Error: Unable to login with valid user!")
	} else{
		pass++
	}

	if LoadSession(100002, nil) {
		t.Errorf("Error: Able to login with invalid user!")
	} else{
		pass++
	}

	if Session == nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	} else{
		pass++
	}

	if Session.GetNodeAddress() == "" {
		t.Errorf("Error: Node Address not set correctly!")
	} else{
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
			} else if Session.GetKeys()[i].ReceptionKeys.Base.Cmp(cyclic.NewInt(2)) != 0{
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


	inmsg := NewMessage(42, 69, "test")[0]

	Session.PushFifo(inmsg)

	outmsg := Session.PopFifo()

	if inmsg != outmsg {
		t.Errorf("Error: Incorrect Return Message from fifo")
	} else{
		pass++
	}

	//Logout
	Session.Immolate()



	if Session != nil {
		t.Errorf("Error: Logout / Immolate did not work!")
	} else{
		pass++
	}
}
