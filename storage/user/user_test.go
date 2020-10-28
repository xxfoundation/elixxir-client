package user

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Test loading user from a KV store
func TestLoadUser(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	_, err := LoadUser(kv)

	if err == nil {
		t.Errorf("Should have failed to load identity from empty kv")
	}

	uid := id.NewIdFromString("test", id.User, t)
	ci := newCryptographicIdentity(uid, []byte("salt"), &rsa.PrivateKey{}, false, kv)
	err = ci.save(kv)
	if err != nil {
		t.Errorf("Failed to save ci to kv: %+v", err)
	}

	_, err = LoadUser(kv)
	if err != nil {
		t.Errorf("Failed to load user: %+v", err)
	}
}

// Test NewUser function
func TestNewUser(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("test", id.User, t)
	u, err := NewUser(kv, uid, []byte("salt"), &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}
}

// Test GetCryptographicIdentity function from user
func TestUser_GetCryptographicIdentity(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	uid := id.NewIdFromString("test", id.User, t)
	u, err := NewUser(kv, uid, []byte("salt"), &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	ci := u.GetCryptographicIdentity()
	if bytes.Compare(ci.salt, []byte("salt")) != 0 {
		t.Errorf("Cryptographic Identity not retrieved properly")
	}
}
