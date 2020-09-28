package keyExchange

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/crypto/csprng"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/primitives/id"
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
	bobPubKey := bobSession.E2e().GetDHPublicKey()
	// Generate bob's new keypair
	newBobPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, genericGroup, csprng.NewSystemRNG())
	newBobPubKey := dh.GeneratePublicKey(newBobPrivKey, genericGroup)

	// Add Alice and Bob as partners
	aliceSession.E2e().AddPartner(exchangeBobId, bobPubKey,
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())
	bobSession.E2e().AddPartner(exchangeAliceId, alicePubKey,
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())

	// Start the listeners for alice and bob
	rekeyParams := params.GetDefaultRekey()
	rekeyParams.RoundTimeout = 1 * time.Second
	Start(aliceSwitchboard, aliceSession, aliceManager, rekeyParams)
	Start(bobSwitchboard, bobSession, bobManager, rekeyParams)

	// Generate a session ID, bypassing some business logic here
	oldSessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup)

	// Generate the message
	rekeyTrigger, _ := proto.Marshal(&RekeyTrigger{
		SessionID: oldSessionID.Marshal(),
		PublicKey: newBobPubKey.Bytes(),
	})

	triggerMsg := message.Receive{
		Payload:     rekeyTrigger,
		MessageType: message.KeyExchangeTrigger,
		Sender:      exchangeBobId,
		Timestamp:   time.Now(),
		Encryption:  message.E2E,
	}

	// Get Alice's manager for reception from Bob
	receivedManager, err := aliceSession.E2e().GetPartner(exchangeBobId)
	if err != nil {
		t.Errorf("Failed to get bob's manager: %v", err)
	}

	// Speak the message to Bob, triggers the SendE2E in utils_test
	aliceSwitchboard.Speak(triggerMsg)

	// Allow the test time to work it's goroutines
	time.Sleep(1 * time.Second)

	// Get Alice's session for Bob
	confirmedSession := receivedManager.GetSendSession(oldSessionID)

	// Generate the new session ID based off of Bob's new keys
	baseKey := dh.GenerateSessionKey(alicePrivKey, newBobPubKey, genericGroup)
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
