////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"math/rand"
	"testing"

	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
)

// Test loading user from a KV store
func TestLoadUser(t *testing.T) {
	sch := rsa.GetScheme()

	kv := versioned.NewKV(ekv.MakeMemstore())
	_, err := LoadUser(kv)

	if err == nil {
		t.Errorf("Should have failed to load identity from empty kv")
	}

	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	transmission, _ := sch.Generate(prng, 256)
	reception, _ := sch.Generate(prng, 256)

	ci := newCryptographicIdentity(uid, uid, salt, salt, transmission,
		reception, false, dhPrivKey, dhPubKey, kv)
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
	sch := rsa.GetScheme()

	kv := versioned.NewKV(ekv.MakeMemstore())
	uid := id.NewIdFromString("test", id.User, t)
	salt := []byte("salt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	transmission, _ := sch.Generate(prng, 256)
	reception, _ := sch.Generate(prng, 256)

	u, err := NewUser(kv, uid, uid, salt, salt, transmission,
		reception, false, dhPrivKey, dhPubKey)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}
}
