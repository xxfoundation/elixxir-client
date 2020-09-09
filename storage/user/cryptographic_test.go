package user

import (
	"bytes"
	"crypto/rand"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Test for NewCryptographicIdentity function
func TestNewCryptographicIdentity(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("zezima", id.User, t)
	_ = newCryptographicIdentity(uid, []byte("salt"), &rsa.PrivateKey{}, false, kv)

	_, err := kv.Get(cryptographicIdentityKey)
	if err != nil {
		t.Errorf("Did not store cryptographic identity")
	}
}

// Test loading cryptographic identity from KV store
func TestLoadCryptographicIdentity(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("zezima", id.User, t)
	ci := newCryptographicIdentity(uid, []byte("salt"), &rsa.PrivateKey{}, false, kv)

	err := ci.save(kv)
	if err != nil {
		t.Errorf("Did not store cryptographic identity: %+v", err)
	}

	newCi, err := loadCryptographicIdentity(kv)
	if err != nil {
		t.Errorf("Failed to load cryptographic identity: %+v", err)
	}
	if !ci.userID.Cmp(newCi.userID) {
		t.Errorf("Did not load expected ci.  Expected: %+v, Received: %+v", ci.userID, newCi.userID)
	}
}

// Happy path for GetRSA function
func TestCryptographicIdentity_GetRSA(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("zezima", id.User, t)
	pk, err := rsa.GenerateKey(rand.Reader, 64)
	if err != nil {
		t.Errorf("Failed to generate pk")
	}
	ci := newCryptographicIdentity(uid, []byte("salt"), pk, false, kv)
	if ci.GetRSA().D != pk.D {
		t.Errorf("Did not receive expected RSA key.  Expected: %+v, Received: %+v", pk, ci.GetRSA())
	}
}

// Happy path for GetSalt function
func TestCryptographicIdentity_GetSalt(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("zezima", id.User, t)
	salt := []byte("NaCl")
	ci := newCryptographicIdentity(uid, salt, &rsa.PrivateKey{}, false, kv)
	if bytes.Compare(ci.GetSalt(), salt) != 0 {
		t.Errorf("Did not get expected salt.  Expected: %+v, Received: %+v", salt, ci.GetSalt())
	}
}

// Happy path for GetUserID function
func TestCryptographicIdentity_GetUserID(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("zezima", id.User, t)
	ci := newCryptographicIdentity(uid, []byte("salt"), &rsa.PrivateKey{}, false, kv)
	if !ci.GetUserID().Cmp(uid) {
		t.Errorf("Did not receive expected user ID.  Expected: %+v, Received: %+v", uid, ci.GetUserID())
	}
}

// Happy path for IsPrecanned functions
func TestCryptographicIdentity_IsPrecanned(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("zezima", id.User, t)
	ci := newCryptographicIdentity(uid, []byte("salt"), &rsa.PrivateKey{}, true, kv)
	if !ci.IsPrecanned() {
		t.Error("I really don't know how this could happen")
	}
}
