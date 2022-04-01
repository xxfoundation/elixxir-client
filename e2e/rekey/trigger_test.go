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
	session2 "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/stoppable"
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
	"time"
)

// Smoke test for handleTrigger
func TestHandleTrigger(t *testing.T) {
	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	// Generate alice and bob's session
	// Pull the keys for Alice and Bob
	bobPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng.GetStream())
	bobPubKey := dh.GeneratePublicKey(bobPrivKey, grp)
	alicePrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng.GetStream())
	alicePubKey := dh.GeneratePublicKey(alicePrivKey, grp)

	// Generate bob's new keypair
	newBobPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, csprng.NewSystemRNG())
	newBobPubKey := dh.GeneratePublicKey(newBobPrivKey, grp)

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

	newBobSIDHPrivKey := util.NewSIDHPrivateKey(bobVariant)
	newBobSIDHPubKey := util.NewSIDHPublicKey(bobVariant)
	newBobSIDHPrivKey.Generate(prng2)
	newBobSIDHPrivKey.GeneratePublicKey(newBobSIDHPubKey)
	newBobSIDHPubKeyBytes := make([]byte, newBobSIDHPubKey.Size()+1)
	newBobSIDHPubKeyBytes[0] = byte(bobVariant)
	newBobSIDHPubKey.Export(newBobSIDHPubKeyBytes[1:])

	// Maintain an ID for bob
	bobID = id.NewIdFromBytes([]byte("test"), t)
	myID = id.NewIdFromString("zezima", id.User, t)
	kv := versioned.NewKV(ekv.Memstore{})

	err := ratchet.New(kv, myID, alicePrivKey, grp)
	if err != nil {
		t.Errorf("Failed to create ratchet: %+v", err)
	}
	r, err = ratchet.Load(kv, myID, grp, mockCyHandler{}, mockServiceHandler{}, rng)
	if err != nil {
		t.Errorf("Failed to load ratchet: %+v", err)
	}

	// Add bob as a partner
	sendParams := session2.GetDefaultParams()
	receiveParams := session2.GetDefaultParams()
	_, err = r.AddPartner(myID, bobID, bobPubKey, alicePrivKey, bobSIDHPubKey, aliceSIDHPrivKey, sendParams, receiveParams, false)
	if err != nil {
		t.Errorf("Failed to add partner to ratchet: %+v", err)
	}
	_, err = r.AddPartner(bobID, myID, alicePubKey, bobPrivKey, aliceSIDHPubKey, bobSIDHPrivKey, sendParams, receiveParams, false)
	if err != nil {
		t.Errorf("Failed to add partner to ratchet: %+v", err)
	}
	// Generate a session ID, bypassing some business logic here
	oldSessionID := GeneratePartnerID(alicePrivKey, bobPubKey, grp,
		aliceSIDHPrivKey, bobSIDHPubKey)

	// Generate the message
	rekey, _ := proto.Marshal(&RekeyTrigger{
		SessionID:     oldSessionID.Marshal(),
		PublicKey:     newBobPubKey.Bytes(),
		SidhPublicKey: newBobSIDHPubKeyBytes,
	})

	receiveMsg := receive.Message{
		Payload:     rekey,
		MessageType: catalog.NoType,
		Sender:      bobID,
		Timestamp:   netTime.Now(),
		Encrypted:   true,
	}

	// Handle the trigger and check for an error
	rekeyParams := GetDefaultParams()
	stop := stoppable.NewSingle("stoppable")
	rekeyParams.RoundTimeout = 0 * time.Second
	err = handleTrigger(r, testSendE2E, &mockNetManager{}, grp, receiveMsg, rekeyParams, stop)
	if err != nil {
		t.Errorf("Handle trigger error: %v", err)
	}

	// get Alice's manager for reception from Bob

	receivedManager, err := r.GetPartner(bobID, myID)
	if err != nil {
		t.Errorf("Failed to get bob's manager: %v", err)
	}

	// Generate the new session ID based off of Bob's new keys
	baseKey := session2.GenerateE2ESessionBaseKey(alicePrivKey, newBobPubKey,
		grp, aliceSIDHPrivKey, newBobSIDHPubKey)
	newSessionID := session2.GetSessionIDFromBaseKeyForTesting(baseKey, t)

	// Check that this new session ID is now in the manager
	newSession := receivedManager.GetReceiveSession(newSessionID)
	if newSession == nil {
		t.Errorf("Did not get expected session")
	}

	// Generate a keypair alice will not recognize
	unknownPrivateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, csprng.NewSystemRNG())
	unknownPubliceKey := dh.GeneratePublicKey(unknownPrivateKey, grp)

	// Generate a new session ID based off of these unrecognized keys
	badSessionID := session2.GetSessionIDFromBaseKeyForTesting(unknownPubliceKey, t)

	// Check that this session with unrecognized keys is not valid
	badSession := receivedManager.GetReceiveSession(badSessionID)
	if badSession != nil {
		t.Errorf("Alice found a session from an unknown keypair. "+
			"\nSession: %v", badSession)
	}

}
