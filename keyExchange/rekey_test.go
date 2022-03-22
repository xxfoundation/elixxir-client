///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package keyExchange

/*
func TestRekey(t *testing.T) {
	// Generate alice and bob's session
	aliceSession, networkManager := InitTestingContextGeneric(t)
	bobSession, _ := InitTestingContextGeneric(t)

	// Pull the keys for Alice and Bob
	alicePrivKey := aliceSession.E2e().GetDHPrivateKey()
	bobPubKey := bobSession.E2e().GetDHPublicKey()

	// Maintain an ID for bob
	bobID := id.NewIdFromBytes([]byte("test"), t)

	// Generate a session ID, bypassing some business logic here
	sessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup)

	// Add bob as a partner
	aliceSession.E2e().AddPartner(bobID, bobPubKey, alicePrivKey,
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())

	// get Alice's manager for Bob
	bobManager, err := aliceSession.E2e().GetPartner(bobID)
	if err != nil {
		t.Errorf("Bob is not recognized as Alice's partner: %v", err)
	}
	//// Trigger negotiations, so that negotiation statuses
	//// can be transitioned
	bobManager.TriggerNegotiations()

	bobE2ESession := bobManager.GetSendSession(sessionID)

	err = negotiate(networkManager.GetInstance(), networkManager.SendE2E, aliceSession, bobE2ESession, 1*time.Second)
	if err != nil {
		t.Errorf("Negotiate resulted in error: %v", err)
	}

	if bobE2ESession.NegotiationStatus() != e2e.Sent {
		t.Errorf("Session not in expected state after negotiation."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", e2e.Sent, bobE2ESession.NegotiationStatus())
	}
}*/
