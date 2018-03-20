////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"testing"
	"gitlab.com/privategrity/crypto/cyclic"
)

func TestVerifyRegisterGobAddress(t *testing.T) {

	if Session.GetNodeAddress() != SERVER_ADDRESS {
		t.Errorf("GetNodeAddress() returned %v, expected %v",
			Session.GetNodeAddress(), SERVER_ADDRESS)
	}
}

func TestVerifyRegisterGobNick(t *testing.T) {
	if Session.GetCurrentUser().Nick != NICK {
		t.Errorf("User's nick was %v, expected %v",
			Session.GetCurrentUser().Nick, NICK)
	}
}

func TestVerifyRegisterGobUserID(t *testing.T) {
	if Session.GetCurrentUser().UserID != 5 {
		t.Errorf("User's ID was %v, expected %v",
			Session.GetCurrentUser().UserID, 5)
	}
}

func TestVerifyRegisterGobKeys(t *testing.T){
	if Session.GetKeys()[0].PublicKey.Cmp(cyclic.NewInt(0)) != 0 {
		t.Errorf("Public key was %v, expected %v",
			Session.GetKeys()[0].PublicKey.Text(16), "0")
	}

	expectedTransmissionRecursiveKey := cyclic.NewIntFromString(
		"ad333f4ccea0ccf2afcab6c1b9aa2384e561aee970046e39b7f2a78c3942a251",
		16)
	if Session.GetKeys()[0].TransmissionKeys.Recursive.Cmp(
		expectedTransmissionRecursiveKey) != 0 {
		t.Errorf("Transmission recursive key was %v, expected %v",
			Session.GetKeys()[0].TransmissionKeys.Recursive.Text(16),
			expectedTransmissionRecursiveKey.Text(16))
	}

	expectedTransmissionBaseKey := cyclic.NewIntFromString(
		"c1248f42f8127999e07c657896a26b56fd9a499c6199e1265053132451128f52",
		16)
	if Session.GetKeys()[0].TransmissionKeys.Base.Cmp(
		expectedTransmissionBaseKey) != 0 {
		t.Errorf("Transmission base key was %v, expected %v",
			Session.GetKeys()[0].TransmissionKeys.Base.Text(16),
			expectedTransmissionBaseKey.Text(16))
	}

	expectedReceptionRecursiveKey := cyclic.NewIntFromString(
		"979e574166ef0cd06d34e3260fe09512b69af6a414cf481770600d9c7447837b",
		16)
	if Session.GetKeys()[0].ReceptionKeys.Recursive.Cmp(
		expectedReceptionRecursiveKey) != 0 {
		t.Errorf("Reception recursive key was %v, expected %v",
			Session.GetKeys()[0].ReceptionKeys.Recursive.Text(16),
			expectedReceptionRecursiveKey.Text(16))
	}

	expectedReceptionBaseKey := cyclic.NewIntFromString(
		"83120e7bfaba497f8e2c95457a28006f73ff4ec75d3ad91d27bf7ce8f04e772c",
		16)
	if Session.GetKeys()[0].ReceptionKeys.Base.Cmp(
		expectedReceptionBaseKey) != 0 {
		t.Errorf("Reception base key was %v, expected %v",
			Session.GetKeys()[0].ReceptionKeys.Base.Text(16),
			expectedReceptionBaseKey.Text(16))
	}

	if Session.GetKeys()[0].ReturnKeys.Recursive == nil {
		t.Logf("warning: return recursive key is nil")
	} else {
		t.Logf("return recursive key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if Session.GetKeys()[0].ReturnKeys.Base == nil {
		t.Logf("warning: return base key is nil")
	} else {
		t.Logf("return base key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if Session.GetKeys()[0].ReceiptKeys.Recursive == nil {
		t.Logf("warning: receipt recursive key is nil")
	} else {
		t.Logf("receipt recursive key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if Session.GetKeys()[0].ReceiptKeys.Base == nil {
		t.Logf("warning: receipt recursive key is nil")
	} else {
		t.Logf("receipt base key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
}
