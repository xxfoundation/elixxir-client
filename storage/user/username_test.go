////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
)

// Test normal function and errors for User's SetUsername function
func TestUser_SetUsername(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	tid := id.NewIdFromString("trans", id.User, t)
	rid := id.NewIdFromString("recv", id.User, t)
	tsalt := []byte("tsalt")
	rsalt := []byte("rsalt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := NewUser(kv, tid, rid, tsalt, rsalt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	u1 := "zezima"
	u2 := "dunkey"
	err = u.SetUsername(u1)
	if err != nil {
		t.Errorf("Failed to set username: %+v", err)
	}

	err = u.SetUsername(u2)
	if err == nil {
		t.Error("Did not error when attempting to set a new username")
	}

	o, err := u.kv.Get(usernameKey, 0)
	if err != nil {
		t.Errorf("Didn't get username from user kv store: %+v", err)
	}

	if string(o.Data) != u1 {
		t.Errorf("Expected username was not stored.\nExpected: %s\tReceived: %s", u1, string(o.Data))
	}
}

// Test functionality of User's GetUsername function
func TestUser_GetUsername(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	tid := id.NewIdFromString("trans", id.User, t)
	rid := id.NewIdFromString("recv", id.User, t)
	tsalt := []byte("tsalt")
	rsalt := []byte("rsalt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := NewUser(kv, tid, rid, tsalt, rsalt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	_, err = u.GetUsername()
	if err == nil {
		t.Error("GetUsername should return an error if username is not set")
	}

	u1 := "zezima"
	u.username = u1
	username, err := u.GetUsername()
	if err != nil {
		t.Errorf("Failed to get username when set: %+v", err)
	}
	if username != u1 {
		t.Errorf("Somehow got the wrong username")
	}
}

// Test the loadUsername helper function
func TestUser_loadUsername(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	tid := id.NewIdFromString("trans", id.User, t)
	rid := id.NewIdFromString("recv", id.User, t)
	tsalt := []byte("tsalt")
	rsalt := []byte("rsalt")

	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		diffieHellman.DefaultPrivateKeyLength, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)

	u, err := NewUser(kv, tid, rid, tsalt, rsalt, &rsa.PrivateKey{},
		&rsa.PrivateKey{}, false, dhPrivKey, dhPubKey)
	if err != nil || u == nil {
		t.Errorf("Failed to create new user: %+v", err)
	}

	u1 := "zezima"

	err = u.kv.Set(usernameKey, &versioned.Object{
		Version:   currentUsernameVersion,
		Timestamp: netTime.Now(),
		Data:      []byte(u1),
	})
	u.loadUsername()
	if u.username != u1 {
		t.Errorf("Username was not properly loaded from kv.\nExpected: %s, Received: %s", u1, u.username)
	}
}
