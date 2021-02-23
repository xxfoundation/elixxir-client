///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package keyExchange

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Smoke test for handleTrigger
func TestHandleConfirm(t *testing.T) {
	// Generate alice and bob's session
	aliceSession, _ := InitTestingContextGeneric(t)
	bobSession, _ := InitTestingContextGeneric(t)

	// Maintain an ID for bob
	bobID := id.NewIdFromBytes([]byte("test"), t)

	// Pull the keys for Alice and Bob
	alicePrivKey := aliceSession.E2e().GetDHPrivateKey()
	bobPubKey := bobSession.E2e().GetDHPublicKey()

	// Add bob as a partner
	aliceSession.E2e().AddPartner(bobID, bobPubKey, alicePrivKey,
		params.GetDefaultE2ESessionParams(),
		params.GetDefaultE2ESessionParams())

	// Generate a session ID, bypassing some business logic here
	sessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup)

	// Get Alice's manager for Bob
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
		Timestamp:   time.Now(),
		Encryption:  message.E2E,
	}

	// Handle the confirmation
	handleConfirm(aliceSession, receiveMsg)

	// Get Alice's session for Bob
	confirmedSession := receivedManager.GetSendSession(sessionID)

	// Check that the session is in the proper status
	newSession := receivedManager.GetSendSession(sessionID)
	if newSession.NegotiationStatus() != e2e.Confirmed {
		t.Errorf("Session not in confirmed status!"+
			"\n\tExpected: Confirmed"+
			"\n\tReceived: %s", confirmedSession.NegotiationStatus())
	}

}
