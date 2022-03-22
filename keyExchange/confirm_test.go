///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package keyExchange

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/e2e"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
)

// Smoke test for handleTrigger
func TestHandleConfirm(t *testing.T) {
	// Generate alice and bob's session
	aliceSession, _, err := InitTestingContextGeneric(t)
	if err != nil {
		t.Fatalf("Failed to create alice session: %v", err)
	}
	bobSession, _, err := InitTestingContextGeneric(t)
	if err != nil {
		t.Fatalf("Failed to create bob session: %v", err)
	}

	// Maintain an ID for bob
	bobID := id.NewIdFromBytes([]byte("test"), t)

	// Pull the keys for Alice and Bob
	alicePrivKey := aliceSession.E2e().GetDHPrivateKey()
	bobPubKey := bobSession.E2e().GetDHPublicKey()

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

	// Add bob as a partner
	aliceSession.E2e().AddPartner(bobID, bobPubKey, alicePrivKey,
		bobSIDHPubKey, aliceSIDHPrivKey,
		params.GetDefaultE2ESessionParams(),
		params.GetDefaultE2ESessionParams())

	// Generate a session ID, bypassing some business logic here
	sessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup,
		aliceSIDHPrivKey, bobSIDHPubKey)

	// get Alice's manager for Bob
	receivedManager, err := aliceSession.E2e().GetPartner(bobID)
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

	receiveMsg := message.Receive{
		Payload:     rekey,
		MessageType: message.KeyExchangeConfirm,
		Sender:      bobID,
		Timestamp:   netTime.Now(),
		Encryption:  message.E2E,
	}

	// Handle the confirmation
	handleConfirm(aliceSession, receiveMsg)

	// get Alice's session for Bob
	confirmedSession := receivedManager.GetSendSession(sessionID)

	// Check that the session is in the proper status
	newSession := receivedManager.GetSendSession(sessionID)
	if newSession.NegotiationStatus() != e2e.Confirmed {
		t.Errorf("Session not in confirmed status!"+
			"\n\tExpected: Confirmed"+
			"\n\tReceived: %s", confirmedSession.NegotiationStatus())
	}

}
