///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package user

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Test loading user from a KV store
func TestLoadUser(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	_, err := LoadUser(kv)

	if err == nil {
		t.Errorf("Should have failed to load identity from empty kv")
	}

	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")
	ci := newCryptographicIdentity(uid, uid, salt, salt, &rsa.PrivateKey{}, &rsa.PrivateKey{}, false, kv)
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
	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")
	u, err := NewUser(kv, uid, uid, salt, salt, &rsa.PrivateKey{}, &rsa.PrivateKey{}, false)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}
}
