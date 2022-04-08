///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// Unit test of getCryptKey.
func Test_getCryptKey(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	g := newTestGroup(getGroup(), getGroup().NewInt(42), prng, t)
	salt, err := newSalt(prng)
	if err != nil {
		t.Errorf("failed to create new salt: %+v", err)
	}
	payload := []byte("payload")
	ts := netTime.Now()

	expectedKey, err := group.NewKdfKey(
		g.Key, group.ComputeEpoch(ts.Add(5*time.Minute)), salt)
	if err != nil {
		t.Errorf("failed to create new key: %+v", err)
	}
	mac := group.NewMAC(expectedKey, payload, g.DhKeys[*g.Members[4].ID])

	key, err := getCryptKey(g.Key, salt, mac, payload, g.DhKeys, ts)
	if err != nil {
		t.Errorf("getCryptKey() returned an error: %+v", err)
	}

	if expectedKey != key {
		t.Errorf("getCryptKey() did not return the expected key."+
			"\nexpected: %v\nreceived: %v", expectedKey, key)
	}
}

// Error path: return an error when the MAC cannot be verified because the
// timestamp is incorrect and generates the wrong epoch.
func Test_getCryptKey_EpochError(t *testing.T) {
	expectedErr := strings.SplitN(genCryptKeyMacErr, "%", 2)[0]

	prng := rand.New(rand.NewSource(42))
	g := newTestGroup(getGroup(), getGroup().NewInt(42), prng, t)
	salt, err := newSalt(prng)
	if err != nil {
		t.Errorf("failed to create new salt: %+v", err)
	}
	payload := []byte("payload")
	ts := netTime.Now()

	key, err := group.NewKdfKey(g.Key, group.ComputeEpoch(ts), salt)
	if err != nil {
		t.Errorf("getCryptKey() returned an error: %+v", err)
	}
	mac := group.NewMAC(key, payload, g.Members[4].DhKey)

	_, err = getCryptKey(g.Key, salt, mac, payload, g.DhKeys, ts.Add(time.Hour))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("getCryptKey() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}
