////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"crypto/rand"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Test for NewCryptographicIdentity function
func TestNewCryptographicIdentity(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("zezima", id.User, t)
	salt := []byte("salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	_ = newCryptographicIdentity(uid, uid, salt, salt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey, kv)

	_, err := kv.Get(cryptographicIdentityKey, currentCryptographicIdentityVersion)
	if err != nil {
		t.Errorf("Did not store cryptographic identity: %+v", err)
	}
}

// Test loading cryptographic identity from KV store
func TestLoadCryptographicIdentity(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("zezima", id.User, t)
	salt := []byte("salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	ci := newCryptographicIdentity(uid, uid, salt, salt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey, kv)

	err := ci.save(kv)
	if err != nil {
		t.Errorf("Did not store cryptographic identity: %+v", err)
	}

	newCi, err := loadCryptographicIdentity(kv)
	if err != nil {
		t.Errorf("Failed to load cryptographic identity: %+v", err)
	}
	if !ci.transmissionID.Cmp(newCi.transmissionID) {
		t.Errorf("Did not load expected ci.  Expected: %+v, Received: %+v", ci.transmissionID, newCi.transmissionID)
	}
}

// Happy path for GetReceptionRSA function
func TestCryptographicIdentity_GetReceptionRSA(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("zezima", id.User, t)
	pk1, err := rsa.GenerateKey(rand.Reader, 64)
	if err != nil {
		t.Errorf("Failed to generate pk1")
	}
	pk2, err := rsa.GenerateKey(rand.Reader, 64)
	if err != nil {
		t.Errorf("Failed to generate pk2")
	}
	salt := []byte("salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	ci := newCryptographicIdentity(
		uid, uid, salt, salt, pk1, pk2, false, dhPrivKey, dhPubKey, kv)
	if ci.GetReceptionRSA().D != pk2.D {
		t.Errorf("Did not receive expected RSA key.  Expected: %+v, Received: %+v", pk2, ci.GetReceptionRSA())
	}
}

// Happy path for GetTransmissionRSA function
func TestCryptographicIdentity_GetTransmissionRSA(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("zezima", id.User, t)
	pk1, err := rsa.GenerateKey(rand.Reader, 64)
	if err != nil {
		t.Errorf("Failed to generate pk1")
	}
	pk2, err := rsa.GenerateKey(rand.Reader, 64)
	if err != nil {
		t.Errorf("Failed to generate pk2")
	}
	salt := []byte("salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	ci := newCryptographicIdentity(
		uid, uid, salt, salt, pk1, pk2, false, dhPrivKey, dhPubKey, kv)
	if ci.GetTransmissionRSA().D != pk1.D {
		t.Errorf("Did not receive expected RSA key.  Expected: %+v, Received: %+v", pk1, ci.GetTransmissionRSA())
	}
}

// Happy path for GetSalt function
func TestCryptographicIdentity_GetTransmissionSalt(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("zezima", id.User, t)
	ts := []byte("transmission salt")
	rs := []byte("reception salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	ci := newCryptographicIdentity(uid, uid, ts, rs, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey, kv)
	if bytes.Compare(ci.GetTransmissionSalt(), ts) != 0 {
		t.Errorf("Did not get expected salt.  Expected: %+v, Received: %+v", ts, ci.GetTransmissionSalt())
	}
}

// Happy path for GetSalt function
func TestCryptographicIdentity_GetReceptionSalt(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("zezima", id.User, t)
	ts := []byte("transmission salt")
	rs := []byte("reception salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	ci := newCryptographicIdentity(uid, uid, ts, rs, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey, kv)
	if bytes.Compare(ci.GetReceptionSalt(), rs) != 0 {
		t.Errorf("Did not get expected salt.  Expected: %+v, Received: %+v", rs, ci.GetReceptionSalt())
	}
}

// Happy path for GetUserID function
func TestCryptographicIdentity_GetTransmissionID(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rid := id.NewIdFromString("zezima", id.User, t)
	tid := id.NewIdFromString("jakexx360", id.User, t)
	salt := []byte("salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	ci := newCryptographicIdentity(tid, rid, salt, salt, &rsa.PrivateKey{}, &rsa.PrivateKey{}, false, dhPrivKey, dhPubKey, kv)
	if !ci.GetTransmissionID().Cmp(tid) {
		t.Errorf("Did not receive expected user ID.  Expected: %+v, Received: %+v", tid, ci.GetTransmissionID())
	}
}

// Happy path for GetUserID function
func TestCryptographicIdentity_GetReceptionID(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rid := id.NewIdFromString("zezima", id.User, t)
	tid := id.NewIdFromString("jakexx360", id.User, t)
	salt := []byte("salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	ci := newCryptographicIdentity(tid, rid, salt, salt, &rsa.PrivateKey{}, &rsa.PrivateKey{}, false, dhPrivKey, dhPubKey, kv)
	if !ci.GetReceptionID().Cmp(rid) {
		t.Errorf("Did not receive expected user ID.  Expected: %+v, Received: %+v", rid, ci.GetReceptionID())
	}
}

// Happy path for IsPrecanned functions
func TestCryptographicIdentity_IsPrecanned(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("zezima", id.User, t)
	salt := []byte("salt")

	prng := rand.Reader
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	ci := newCryptographicIdentity(uid, uid, salt, salt, &rsa.PrivateKey{}, &rsa.PrivateKey{}, true, dhPrivKey, dhPubKey, kv)
	if !ci.IsPrecanned() {
		t.Error("I really don't know how this could happen")
	}
}
