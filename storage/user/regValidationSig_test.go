///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Test User GetRegistrationValidationSignature function
func TestUser_GetRegistrationValidationSignature(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")
	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{}, &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	sig := []byte("testreceptionsignature")
	u.SetReceptionRegistrationValidationSignature(sig)
	if bytes.Compare(sig, u.receptionRegValidationSig) != 0 {
		t.Errorf("Failed to set user object signature field.  Expected: %+v, Received: %+v",
			sig, u.receptionRegValidationSig)
	}

	if bytes.Compare(u.GetReceptionRegistrationValidationSignature(), sig) != 0 {
		t.Errorf("Did not receive expected result from GetRegistrationValidationSignature.  "+
			"Expected: %+v, Received: %+v", sig, u.GetReceptionRegistrationValidationSignature())
	}

	sig = []byte("testtransmissionsignature")
	u.SetTransmissionRegistrationValidationSignature(sig)
	if bytes.Compare(sig, u.transmissionRegValidationSig) != 0 {
		t.Errorf("Failed to set user object signature field.  Expected: %+v, Received: %+v",
			sig, u.transmissionRegValidationSig)
	}

	if bytes.Compare(u.GetTransmissionRegistrationValidationSignature(), sig) != 0 {
		t.Errorf("Did not receive expected result from GetRegistrationValidationSignature.  "+
			"Expected: %+v, Received: %+v", sig, u.GetTransmissionRegistrationValidationSignature())
	}
}

// Test SetRegistrationValidationSignature setter
func TestUser_SetRegistrationValidationSignature(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")
	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{}, &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	sig := []byte("testtransmissionsignature")
	u.SetTransmissionRegistrationValidationSignature(sig)
	if bytes.Compare(sig, u.transmissionRegValidationSig) != 0 {
		t.Errorf("Failed to set user object signature field.  Expected: %+v, Received: %+v",
			sig, u.transmissionRegValidationSig)
	}

	obj, err := u.kv.Get(transmissionRegValidationSigKey)
	if err != nil {
		t.Errorf("Failed to get reg vaildation signature key: %+v", err)
	}
	if bytes.Compare(obj.Data, sig) != 0 {
		t.Errorf("Did not properly set reg validation signature key in kv store.\nExpected: %+v, Received: %+v",
			sig, obj.Data)
	}

	sig = []byte("testreceptionsignature")
	u.SetReceptionRegistrationValidationSignature(sig)
	if bytes.Compare(sig, u.receptionRegValidationSig) != 0 {
		t.Errorf("Failed to set user object signature field.  Expected: %+v, Received: %+v",
			sig, u.receptionRegValidationSig)
	}

	obj, err = u.kv.Get(receptionRegValidationSigKey)
	if err != nil {
		t.Errorf("Failed to get reg vaildation signature key: %+v", err)
	}
	if bytes.Compare(obj.Data, sig) != 0 {
		t.Errorf("Did not properly set reg validation signature key in kv store.\nExpected: %+v, Received: %+v",
			sig, obj.Data)
	}
}

// Test loading registrationValidationSignature from the KV store
func TestUser_loadRegistrationValidationSignature(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")
	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{}, &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	sig := []byte("transmissionsignature")
	err = kv.Set(transmissionRegValidationSigKey, &versioned.Object{
		Version:   currentRegValidationSigVersion,
		Timestamp: time.Now(),
		Data:      sig,
	})
	if err != nil {
		t.Errorf("Failed to set reg validation sig key in kv store: %+v", err)
	}

	u.loadTransmissionRegistrationValidationSignature()
	if bytes.Compare(u.transmissionRegValidationSig, sig) != 0 {
		t.Errorf("Expected sig did not match loaded.  Expected: %+v, Received: %+v", sig, u.transmissionRegValidationSig)
	}

	sig = []byte("receptionsignature")
	err = kv.Set(receptionRegValidationSigKey, &versioned.Object{
		Version:   currentRegValidationSigVersion,
		Timestamp: time.Now(),
		Data:      sig,
	})
	if err != nil {
		t.Errorf("Failed to set reg validation sig key in kv store: %+v", err)
	}

	u.loadReceptionRegistrationValidationSignature()
	if bytes.Compare(u.receptionRegValidationSig, sig) != 0 {
		t.Errorf("Expected sig did not match loaded.  Expected: %+v, Received: %+v", sig, u.receptionRegValidationSig)
	}
}
