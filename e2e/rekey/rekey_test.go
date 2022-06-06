///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rekey

// todo: this test is broken in release, needs to be fixed for restructure too
//  may need a full rewrite
/*
func TestRekey(t *testing.T) {

	grp := getGroup()
	mci := mockCommsInstance{
		ds.NewRoundEvents(),
	}
	rng := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)
	stream := rng.GetStream()
	defer stream.Close()
	// Generate alice and bob's session
	//aliceSession, networkManager := InitTestingContextGeneric(t)
	//bobSession, _ := InitTestingContextGeneric(t)

	// Generate a ratchet data here so that private data within
	// ratchet.Ratchet is accessible for setup
	bobPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng.GetStream())
	bobPubKey := dh.GeneratePublicKey(bobPrivKey, grp)
	alicePrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng.GetStream())
	alicePubKey := dh.GeneratePublicKey(alicePrivKey, grp)

	cyHanlder := mockCyHandler{}
	service := mockServiceHandler{}
	aliceID = id.NewIdFromString("Alice", id.User, t)

	aliceSIDHPrivKey, bobSIDHPubKey, bobSIDHPrivKey, aliceSIDHPubKey := genSidhKeys()

	bobID = id.NewIdFromUInt(rand.Uint64(), id.User, t)

	kv := versioned.NewKV(ekv.MakeMemstore())

	err := ratchet.New(kv, aliceID, alicePrivKey, grp)
	if err != nil {
		t.Errorf("Failed to create ratchet: %+v", err)
	}
	r, err = ratchet.Load(kv, aliceID, grp, cyHanlder, service, rng)
	if err != nil {
		t.Fatalf("ratchet.Load() produced an error: %v", err)
	}

	// Add bob as a partner
	sendParams := session.GetDefaultParams()
	receiveParams := session.GetDefaultParams()
	aliceManager, err := r.AddPartner(aliceID, bobID, bobPubKey,
		alicePrivKey, bobSIDHPubKey, aliceSIDHPrivKey,
		sendParams, receiveParams)
	if err != nil {
		t.Errorf("Failed to add partner to ratchet: %+v", err)
	}
	bobManager, err := r.AddPartner(bobID, aliceID, alicePubKey, bobPrivKey,
		aliceSIDHPubKey, bobSIDHPrivKey,
		sendParams, receiveParams)
	if err != nil {
		t.Errorf("Failed to add partner to ratchet: %+v", err)
	}

	for i := 0; i < 3; i++ {
		ri := &pb.RoundInfo{
			ID:    uint64(i),
			State: uint32(states.COMPLETED),
		}

		mci.TriggerRoundEvent(ds.NewVerifiedRound(ri, nil))

	}
	//
	//partnerPubKey := diffieHellman.GeneratePublicKey(
	//	testRatchet.GetDHPrivateKey(), grp)
	//p := session.GetDefaultParams()
	//_, partnerPubSIDHKey := genSidhKeys(stream, sidh.KeyVariantSidhA)
	//myPrivSIDHKey, _ := genSidhKeys(stream, sidh.KeyVariantSidhB)
	//partnerManager, err := testRatchet.AddPartner(myId, partnerID,
	//	partnerPubKey, myPrivKey,
	//	partnerPubSIDHKey, myPrivSIDHKey, p, p)
	//if err != nil {
	//	t.Fatalf("AddPartner returned an error: %v", err)
	//}

	baseKey := session.GenerateE2ESessionBaseKey(alicePrivKey, bobPubKey,
		grp, aliceSIDHPrivKey, bobSIDHPubKey)

	// Generate a session ID, bypassing some business logic here
	sessionID := session.GetSessionIDFromBaseKey(baseKey)

	// Trigger negotiations, so that negotiation statuses
	// can be transitioned
	bobManager.TriggerNegotiations()
	aliceManager.TriggerNegotiations()
	aliceSess := aliceManager.GetSendSession(sessionID)
	t.Logf("aliceSession: %v", aliceSess)

	bobE2ESession := bobManager.GetSendSession(sessionID)
	t.Logf("bobE2eSession: %v", bobE2ESession)
	err = negotiate(&mci, grp, testSendE2E, GetDefaultParams(),
		bobE2ESession, 1*time.Second)
	if err != nil {
		t.Errorf("Negotiate resulted in error: %v", err)
	}

	t.Logf("Alice status: %s", aliceSess.NegotiationStatus())

	if bobE2ESession.NegotiationStatus() != session.Sent {
		t.Errorf("Session not in expected state after negotiation."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", session.Sent, bobE2ESession.NegotiationStatus())
	}
}
*/
