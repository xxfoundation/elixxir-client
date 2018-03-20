////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"testing"
	"bytes"
	"encoding/gob"
	"strconv"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
)

func TestRegister(t *testing.T) {
	globals.InitStorage(&globals.RamStorage{}, "")

	huid, _ := strconv.ParseUint("be50nhqpqjtjj", 32, 64)

	nick := "Alduin"
	// populate a gob in the store
	Register(huid, nick, SERVER_ADDRESS, 1)

	// todo: maybe some of this stuff should get broken out into a user
	// package or something. at the very least, the register code doesn't
	// seem to have a single responsibility right now
	if globals.LocalStorage == nil {
		t.Errorf("Local storage was nil")
	}

	// get the gob out of there
	sessionGob := globals.LocalStorage.Load()

	var sessionBytes bytes.Buffer

	sessionBytes.Write(sessionGob)

	dec := gob.NewDecoder(&sessionBytes)

	session := globals.SessionObj{}

	dec.Decode(&session)

	if session.GetNodeAddress() != SERVER_ADDRESS {
		t.Errorf("GetNodeAddress() returned %v, expected %v",
			session.GetNodeAddress(), SERVER_ADDRESS)
	}
	if session.GetCurrentUser().Nick != nick {
		t.Errorf("User's nick was %v, expected %v",
			session.GetCurrentUser().Nick, nick)
	}
	if session.GetCurrentUser().UserID != 5 {
		t.Errorf("User's ID was %v, expected %v",
			session.GetCurrentUser().UserID, 5)
	}
	if session.GetKeys()[0].PublicKey.Cmp(cyclic.NewInt(0)) != 0 {
		t.Errorf("Public key was %v, expected %v",
			session.GetKeys()[0].PublicKey.Text(16), "0")
	}

	expectedTransmissionRecursiveKey := cyclic.NewIntFromString(
		"ad333f4ccea0ccf2afcab6c1b9aa2384e561aee970046e39b7f2a78c3942a251",
		16)
	if session.GetKeys()[0].TransmissionKeys.Recursive.Cmp(
		expectedTransmissionRecursiveKey) != 0 {
		t.Errorf("Transmission recursive key was %v, expected %v",
			session.GetKeys()[0].TransmissionKeys.Recursive.Text(16),
			expectedTransmissionRecursiveKey.Text(16))
	}

	expectedTransmissionBaseKey := cyclic.NewIntFromString(
		"c1248f42f8127999e07c657896a26b56fd9a499c6199e1265053132451128f52",
		16)
	if session.GetKeys()[0].TransmissionKeys.Base.Cmp(
		expectedTransmissionBaseKey) != 0 {
		t.Errorf("Transmission base key was %v, expected %v",
			session.GetKeys()[0].TransmissionKeys.Base.Text(16),
			expectedTransmissionBaseKey.Text(16))
	}

	expectedReceptionRecursiveKey := cyclic.NewIntFromString(
		"979e574166ef0cd06d34e3260fe09512b69af6a414cf481770600d9c7447837b",
		16)
	if session.GetKeys()[0].ReceptionKeys.Recursive.Cmp(
		expectedReceptionRecursiveKey) != 0 {
		t.Errorf("Reception recursive key was %v, expected %v",
			session.GetKeys()[0].ReceptionKeys.Recursive.Text(16),
			expectedReceptionRecursiveKey.Text(16))
	}

	expectedReceptionBaseKey := cyclic.NewIntFromString(
		"83120e7bfaba497f8e2c95457a28006f73ff4ec75d3ad91d27bf7ce8f04e772c",
		16)
	if session.GetKeys()[0].ReceptionKeys.Base.Cmp(
		expectedReceptionBaseKey) != 0 {
		t.Errorf("Reception base key was %v, expected %v",
			session.GetKeys()[0].ReceptionKeys.Base.Text(16),
			expectedReceptionBaseKey.Text(16))
	}

	if session.GetKeys()[0].ReturnKeys.Recursive == nil {
		t.Logf("warning: return recursive key is nil")
	} else {
		t.Logf("return recursive key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if session.GetKeys()[0].ReturnKeys.Base == nil {
		t.Logf("warning: return base key is nil")
	} else {
		t.Logf("return base key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if session.GetKeys()[0].ReceiptKeys.Recursive == nil {
		t.Logf("warning: receipt recursive key is nil")
	} else {
		t.Logf("receipt recursive key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if session.GetKeys()[0].ReceiptKeys.Base == nil {
		t.Logf("warning: receipt recursive key is nil")
	} else {
		t.Logf("receipt base key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
}
