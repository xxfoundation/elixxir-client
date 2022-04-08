///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"github.com/cloudflare/circl/dh/sidh"
	auth2 "gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"math/rand"
	"testing"
	"time"
)

func TestManager_ReplayRequests(t *testing.T) {
	s := storage.InitTestingSession(t)
	numReceived := 10

	// Construct barebones manager
	m := state{
		requestCallbacks: newCallbackMap(),
		storage:          s,
		replayRequests:   true,
	}

	ch := make(chan struct{}, numReceived)

	// Add multiple received contact requests
	for i := 0; i < numReceived; i++ {
		c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
		rng := csprng.NewSystemRNG()
		_, sidhPubKey := genSidhAKeys(rng)

		if err := m.storage.Auth().AddReceived(c, sidhPubKey); err != nil {
			t.Fatalf("AddReceived() returned an error: %+v", err)
		}

		m.requestCallbacks.AddSpecific(c.ID, interfaces.RequestCallback(func(c contact.Contact) {
			ch <- struct{}{}
		}))

	}

	m.ReplayRequests()

	timeout := time.NewTimer(1 * time.Second)
	numChannelReceived := 0
loop:
	for {
		select {
		case <-ch:
			numChannelReceived++
		case <-timeout.C:
			break loop
		}
	}

	if numReceived != numChannelReceived {
		t.Errorf("Unexpected number of callbacks called"+
			"\nExpected: %d"+
			"\nReceived: %d", numChannelReceived, numReceived)
	}
}

func makeTestStore(t *testing.T) (*auth2.Store, *versioned.KV, []*cyclic.Int) {
	kv := versioned.NewKV(make(ekv.Memstore))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(0))
	privKeys := make([]*cyclic.Int, 10)
	for i := range privKeys {
		privKeys[i] = grp.NewInt(rand.Int63n(170) + 1)
	}

	store, err := auth2.NewStore(kv, grp, privKeys)
	if err != nil {
		t.Fatalf("Failed to create new Store: %+v", err)
	}

	return store, kv, privKeys
}

func genSidhAKeys(rng io.Reader) (*sidh.PrivateKey, *sidh.PublicKey) {
	sidHPrivKeyA := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	sidHPubKeyA := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)

	if err := sidHPrivKeyA.Generate(rng); err != nil {
		panic("failure to generate SidH A private key")
	}
	sidHPrivKeyA.GeneratePublicKey(sidHPubKeyA)

	return sidHPrivKeyA, sidHPubKeyA
}
