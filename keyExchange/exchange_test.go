///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package keyExchange

import (
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/e2e"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/switchboard"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
	"time"
)

var exchangeAliceId, exchangeBobId *id.ID
var aliceSession, bobSession *storage.Session
var aliceSwitchboard, bobSwitchboard *switchboard.Switchboard
var aliceManager, bobManager interfaces.NetworkManager

func TestFullExchange(t *testing.T) {
	// Initialzie alice's and bob's session, switchboard and network managers
	aliceSession, aliceSwitchboard, aliceManager = InitTestingContextFullExchange(t)
	bobSession, bobSwitchboard, bobManager = InitTestingContextFullExchange(t)

	// Assign ID's to alice and bob
	exchangeAliceId = id.NewIdFromBytes([]byte("1234"), t)
	exchangeBobId = id.NewIdFromBytes([]byte("test"), t)

	// Pull alice's and bob's keys for later use
	alicePrivKey := aliceSession.E2e().GetDHPrivateKey()
	alicePubKey := aliceSession.E2e().GetDHPublicKey()
	bobPrivKey := bobSession.E2e().GetDHPrivateKey()
	bobPubKey := bobSession.E2e().GetDHPublicKey()

	// Generate bob's new keypair
	newBobPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, genericGroup, csprng.NewSystemRNG())
	newBobPubKey := dh.GeneratePublicKey(newBobPrivKey, genericGroup)

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

	// Add Alice and Bob as partners
	aliceSession.E2e().AddPartner(exchangeBobId, bobPubKey, alicePrivKey,
		bobSIDHPubKey, aliceSIDHPrivKey,
		params.GetDefaultE2ESessionParams(),
		params.GetDefaultE2ESessionParams())
	bobSession.E2e().AddPartner(exchangeAliceId, alicePubKey, bobPrivKey,
		aliceSIDHPubKey, bobSIDHPrivKey,
		params.GetDefaultE2ESessionParams(),
		params.GetDefaultE2ESessionParams())

	// Start the listeners for alice and bob
	rekeyParams := params.GetDefaultRekey()
	rekeyParams.RoundTimeout = 1 * time.Second
	Start(aliceSwitchboard, aliceSession, aliceManager, rekeyParams)
	Start(bobSwitchboard, bobSession, bobManager, rekeyParams)

	// Generate a session ID, bypassing some business logic here
	oldSessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup,
		aliceSIDHPrivKey, bobSIDHPubKey)

	// Generate the message
	rekeyTrigger, _ := proto.Marshal(&RekeyTrigger{
		SessionID:     oldSessionID.Marshal(),
		PublicKey:     newBobPubKey.Bytes(),
		SidhPublicKey: newBobSIDHPubKeyBytes,
	})

	triggerMsg := message.Receive{
		Payload:     rekeyTrigger,
		MessageType: message.KeyExchangeTrigger,
		Sender:      exchangeBobId,
		Timestamp:   netTime.Now(),
		Encryption:  message.E2E,
	}

	// get Alice's manager for reception from Bob
	receivedManager, err := aliceSession.E2e().GetPartner(exchangeBobId)
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
	baseKey := e2e.GenerateE2ESessionBaseKey(alicePrivKey, newBobPubKey,
		genericGroup, aliceSIDHPrivKey, newBobSIDHPubKey)
	newSessionID := e2e.GetSessionIDFromBaseKeyForTesting(baseKey, t)

	// Check that the Alice's session for Bob is in the proper status
	newSession := receivedManager.GetReceiveSession(newSessionID)
	fmt.Printf("newSession: %v\n", newSession)
	if newSession == nil || newSession.NegotiationStatus() != e2e.Confirmed {
		t.Errorf("Session not in confirmed status!"+
			"\n\tExpected: Confirmed"+
			"\n\tReceived: %s", confirmedSession.NegotiationStatus())
	}

	fmt.Printf("after status: %v\n", confirmedSession.NegotiationStatus())

}
