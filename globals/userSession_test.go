////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"testing"
)

// TestUserRegistry tests the constructors/getters/setters
// surrounding the User struct and the UserRegistry interface
func TestUserSession(t *testing.T) {

	Session = newUserSession(1)

	if Session.GetCurrentUser() != nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	}

	if !Session.Login(1, "") {
		t.Errorf("Error: Unable to login with valid user!")
	}

	if Session.Login(1, "") {
		t.Errorf("Error: Able to login with invalid user!")
	}

	if Session.GetCurrentUser() == nil {
		t.Errorf("Error: CurrentUser not set correctly!")
	}

	if Session.GetKeys() == nil {
		t.Errorf("Error: Keys not set correctly!")
	} else {
		for i := 0; i < len(Session.GetKeys()); i++ {
			if Session.GetKeys()[i].PublicKey == nil {
				t.Errorf("Error: Public key not set correctly!")
			} else if Session.GetKeys()[i].ReceiptKeys.Base == nil {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReceiptKeys.Recursive == nil {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReceptionKeys.Base == nil {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReceptionKeys.Recursive == nil {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReturnKeys.Base == nil {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].ReturnKeys.Recursive == nil {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].TransmissionKeys.Base == nil {
				t.Errorf("Error: Receipt base key not set correctly!")
			} else if Session.GetKeys()[i].TransmissionKeys.Recursive == nil {
				t.Errorf("Error: Receipt base key not set correctly!")
			}
		}
	}

	inmsg := NewMessage(42, 69, "test")[0]

	Session.PushFifo(inmsg)

	outmsg := Session.PopFifo()

	if inmsg != outmsg {
		t.Errorf("Error: Incorrect Return Message from Fifo")
	}

}
