////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"encoding/binary"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
	"time"
)

// Test User GetRegistrationValidationSignature function
func TestUser_GetRegistrationValidationSignature(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey)
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
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	sig := []byte("testtransmissionsignature")
	u.SetTransmissionRegistrationValidationSignature(sig)
	if bytes.Compare(sig, u.transmissionRegValidationSig) != 0 {
		t.Errorf("Failed to set user object signature field.  Expected: %+v, Received: %+v",
			sig, u.transmissionRegValidationSig)
	}

	obj, err := u.kv.Get(transmissionRegValidationSigKey, 0)
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

	obj, err = u.kv.Get(receptionRegValidationSigKey, 0)
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
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	sig := []byte("transmissionsignature")
	err = kv.Set(transmissionRegValidationSigKey,
		&versioned.Object{
			Version:   currentRegValidationSigVersion,
			Timestamp: netTime.Now(),
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
	err = kv.Set(receptionRegValidationSigKey,
		&versioned.Object{
			Version:   currentRegValidationSigVersion,
			Timestamp: netTime.Now(),
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

// Test User's getter/setter functions for TimeStamp
func TestUser_GetRegistrationTimestamp(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("Could not parse precanned time: %v", err.Error())
	}

	// Test that User has been modified for timestamp
	u.SetRegistrationTimestamp(testTime.UnixNano())
	if !testTime.Equal(u.registrationTimestamp) {
		t.Errorf("SetRegistrationTimestamp did not set user's timestamp value."+
			"\n\tExpected: %s\n\tReceieved: %s", testTime.String(), u.registrationTimestamp)
	}

	// Pull timestamp from kv
	obj, err := u.kv.Get(registrationTimestampKey, registrationTimestampVersion)
	if err != nil {
		t.Errorf("Failed to get reg vaildation signature key: %+v", err)
	}

	// Check if kv data is expected
	unixNano := binary.BigEndian.Uint64(obj.Data)
	if testTime.UnixNano() != int64(unixNano) {
		t.Errorf("Timestamp pulled from kv was not expected."+
			"\n\tExpected: %d\n\tReceieved: %d", testTime.UnixNano(), unixNano)
	}

	if testTime.UnixNano() != u.GetRegistrationTimestamp().UnixNano() {
		t.Errorf("Timestamp from GetRegistrationTimestampNano was not expected."+
			"\n\tExpected: %d\n\tReceieved: %d", testTime.UnixNano(), u.GetRegistrationTimestamp().UnixNano())
	}

	if !testTime.Equal(u.GetRegistrationTimestamp()) {
		t.Errorf("Timestamp from GetRegistrationTimestamp was not expected."+
			"\n\tExpected: %s\n\tReceieved: %s", testTime, u.GetRegistrationTimestamp())

	}

}

// Test loading registrationTimestamp from the KV store
func TestUser_loadRegistrationTimestamp(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("Could not parse precanned time: %v", err.Error())
	}

	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(testTime.UnixNano()))
	vo := &versioned.Object{
		Version:   registrationTimestampVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = kv.Set(registrationTimestampKey, vo)
	if err != nil {
		t.Errorf("Failed to set reg validation sig key in kv store: %+v", err)
	}

	u.loadRegistrationTimestamp()
	if !testTime.Equal(u.registrationTimestamp) {
		t.Errorf("SetRegistrationTimestamp did not set user's timestamp value."+
			"\n\tExpected: %s\n\tReceieved: %s", testTime.String(), u.registrationTimestamp)
	}
}
