///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
)

// Smoke test for handleConfirm
func TestHandleConfirm(t *testing.T) {
	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	myID := id.NewIdFromString("zezima", id.User, t)

	kv := versioned.NewKV(ekv.Memstore{})

	// Maintain an ID for bob
	bobID := id.NewIdFromBytes([]byte("test"), t)

	// Pull the keys for Alice and Bob
	bobPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng.GetStream())
	bobPubKey := dh.GeneratePublicKey(bobPrivKey, grp)
	alicePrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng.GetStream())

	aliceVariant := sidh.KeyVariantSidhA
	prng1 := rand.New(rand.NewSource(int64(1)))
	aliceSIDHPrivKey := util.NewSIDHPrivateKey(aliceVariant)
	aliceSIDHPubKey := util.NewSIDHPublicKey(aliceVariant)
	aliceSIDHPrivKey.Generate(prng1)
	aliceSIDHPrivKey.GeneratePublicKey(aliceSIDHPubKey)

	bobVariant := sidh.KeyVariant(sidh.KeyVariantSidhB)
	prng2 := rand.New(rand.NewSource(int64(2)))
	bobSIDHPrivKey := util.NewSIDHPrivateKey(bobVariant)
	bobSIDHPubKey := util.NewSIDHPublicKey(bobVariant)
	bobSIDHPrivKey.Generate(prng2)
	bobSIDHPrivKey.GeneratePublicKey(bobSIDHPubKey)

	err := ratchet.New(kv, myID, alicePrivKey, grp)
	if err != nil {
		t.Errorf("Failed to create ratchet: %+v", err)
	}
	r, err := ratchet.Load(kv, myID, grp, mockCyHandler{}, mockServiceHandler{}, rng)
	if err != nil {
		t.Errorf("Failed to load ratchet: %+v", err)
	}

	// Add bob as a partner
	sendParams := session.GetDefaultParams()
	receiveParams := session.GetDefaultParams()
	_, err = r.AddPartner(myID, bobID, bobPubKey, alicePrivKey, bobSIDHPubKey, aliceSIDHPrivKey, sendParams, receiveParams)
	if err != nil {
		t.Errorf("Failed to add partner to ratchet: %+v", err)
	}
	// Generate a session ID, bypassing some business logic here
	sessionID := GeneratePartnerID(alicePrivKey, bobPubKey, grp,
		aliceSIDHPrivKey, bobSIDHPubKey)

	// get Alice's manager for Bob
	receivedManager, err := r.GetPartner(bobID, myID)
	if err != nil {
		t.Errorf("Bob is not recognized as Alice's partner: %v", err)
	}

	// Trigger negotiations, so that negotiation statuses
	// can be transitioned
	receivedManager.TriggerNegotiations()

	// Generate the message
	rekey, _ := proto.Marshal(&RekeyConfirm{
		SessionID: sessionID.Marshal(),
	})

	receiveMsg := receive.Message{
		MessageType: catalog.KeyExchangeConfirm,
		Payload:     rekey,
		Sender:      bobID,
		RecipientID: myID,
		Encrypted:   true,
		Timestamp:   netTime.Now(),
	}

	// Handle the confirmation
	handleConfirm(r, receiveMsg)

	// get Alice's session for Bob
	confirmedSession := receivedManager.GetSendSession(sessionID)

	// Check that the session is in the proper status
	newSession := receivedManager.GetSendSession(sessionID)
	if newSession.NegotiationStatus() != session.Confirmed {
		t.Errorf("Session not in confirmed status!"+
			"\n\tExpected: Confirmed"+
			"\n\tReceived: %s", confirmedSession.NegotiationStatus())
	}

}
