///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"fmt"
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
	"time"
)

var r *ratchet.Ratchet
var aliceID, bobID *id.ID
var aliceSwitchboard = receive.New()
var bobSwitchboard = receive.New()

func TestFullExchange(t *testing.T) {
	// Initialzie alice's and bob's session, switchboard and network managers
	// Assign ID's to alice and bob
	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	aliceID = id.NewIdFromString("zezima", id.User, t)

	kv := versioned.NewKV(ekv.Memstore{})

	// Maintain an ID for bob
	bobID = id.NewIdFromBytes([]byte("test"), t)

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

	err := ratchet.New(kv, aliceID, alicePrivKey, grp)
	if err != nil {
		t.Errorf("Failed to create ratchet: %+v", err)
	}
	r, err = ratchet.Load(kv, aliceID, grp, mockCyHandler{}, mockServiceHandler{}, rng)
	if err != nil {
		t.Errorf("Failed to load ratchet: %+v", err)
	}

	// Add Alice and Bob as partners
	sendParams := session.GetDefaultParams()
	receiveParams := session.GetDefaultParams()
	_, err = r.AddPartner(aliceID, bobID, bobPubKey,
		alicePrivKey, bobSIDHPubKey, aliceSIDHPrivKey,
		sendParams, receiveParams)
	if err != nil {
		t.Errorf("Failed to add partner to ratchet: %+v", err)
	}
	_, err = r.AddPartner(bobID, aliceID, alicePubKey,
		bobPrivKey, aliceSIDHPubKey, bobSIDHPrivKey,
		sendParams, receiveParams)
	if err != nil {
		t.Errorf("Failed to add partner to ratchet: %+v", err)
	}

	// Start the listeners for alice and bob
	rekeyParams := GetDefaultParams()
	rekeyParams.RoundTimeout = 1 * time.Second
	_, err = Start(aliceSwitchboard, r, testSendE2E, &mockNetManager{}, grp, rekeyParams)
	if err != nil {
		t.Errorf("Failed to Start alice: %+v", err)
	}
	_, err = Start(bobSwitchboard, r, testSendE2E, &mockNetManager{}, grp, rekeyParams)
	if err != nil {
		t.Errorf("Failed to Start bob: %+v", err)
	}
	fmt.Println("1")
	// Generate a session ID, bypassing some business logic here
	oldSessionID := GeneratePartnerID(alicePrivKey, bobPubKey, grp,
		aliceSIDHPrivKey, bobSIDHPubKey)

	// Generate the message
	rekeyTrigger, _ := proto.Marshal(&RekeyTrigger{
		SessionID:     oldSessionID.Marshal(),
		PublicKey:     newBobPubKey.Bytes(),
		SidhPublicKey: newBobSIDHPubKeyBytes,
	})

	triggerMsg := receive.Message{
		Payload:     rekeyTrigger,
		MessageType: catalog.KeyExchangeTrigger,
		Sender:      bobID,
		Timestamp:   netTime.Now(),
		Encrypted:   true,
	}

	// get Alice's manager for reception from Bob
	receivedManager, err := r.GetPartner(bobID, aliceID)
	if err != nil {
		t.Errorf("Failed to get bob's manager: %v", err)
	}

	// Speak the message to Bob, triggers the SendE2E in utils_test
	aliceSwitchboard.Speak(triggerMsg)

	// Allow the test time to work it's goroutines
	time.Sleep(1 * time.Second)

	// get Alice's session for Bob
	confirmedSession := receivedManager.GetSendSession(oldSessionID)

	// Generate the new session ID based off of Bob's new keys
	baseKey := session.GenerateE2ESessionBaseKey(alicePrivKey, newBobPubKey,
		grp, aliceSIDHPrivKey, newBobSIDHPubKey)
	newSessionID := session.GetSessionIDFromBaseKeyForTesting(baseKey, t)

	// Check that the Alice's session for Bob is in the proper status
	newSession := receivedManager.GetReceiveSession(newSessionID)
	if newSession == nil || newSession.NegotiationStatus() != session.Confirmed {
		t.Errorf("Session not in confirmed status!"+
			"\n\tExpected: Confirmed"+
			"\n\tReceived: %s", confirmedSession.NegotiationStatus())
	}
}
