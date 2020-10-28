package keyExchange

import (
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/crypto/csprng"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

// Smoke test for handleTrigger
func TestHandleTrigger(t *testing.T) {
	// Generate alice and bob's session
	aliceSession, aliceManager := InitTestingContextGeneric(t)
	bobSession, _ := InitTestingContextGeneric(t)

	// Pull the keys for Alice and Bob
	alicePrivKey := aliceSession.E2e().GetDHPrivateKey()
	bobPubKey := bobSession.E2e().GetDHPublicKey()

	// Generate bob's new keypair
	newBobPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, genericGroup, csprng.NewSystemRNG())
	newBobPubKey := dh.GeneratePublicKey(newBobPrivKey, genericGroup)

	// Maintain an ID for bob
	bobID := id.NewIdFromBytes([]byte("test"), t)

	// Add bob as a partner
	aliceSession.E2e().AddPartner(bobID, bobSession.E2e().GetDHPublicKey(),
		alicePrivKey, e2e.GetDefaultSessionParams(),
		e2e.GetDefaultSessionParams())

	// Generate a session ID, bypassing some business logic here
	oldSessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup)

	// Generate the message
	rekey, _ := proto.Marshal(&RekeyTrigger{
		SessionID: oldSessionID.Marshal(),
		PublicKey: newBobPubKey.Bytes(),
	})

	receiveMsg := message.Receive{
		Payload:     rekey,
		MessageType: message.NoType,
		Sender:      bobID,
		Timestamp:   time.Now(),
		Encryption:  message.E2E,
	}

	// Handle the trigger and check for an error
	rekeyParams := params.GetDefaultRekey()
	rekeyParams.RoundTimeout = 0 * time.Second
	err := handleTrigger(aliceSession, aliceManager, receiveMsg, rekeyParams)
	if err != nil {
		t.Errorf("Handle trigger error: %v", err)
	}

	// Get Alice's manager for reception from Bob
	receivedManager, err := aliceSession.E2e().GetPartner(bobID)
	if err != nil {
		t.Errorf("Failed to get bob's manager: %v", err)
	}

	// Generate the new session ID based off of Bob's new keys
	baseKey := dh.GenerateSessionKey(alicePrivKey, newBobPubKey, genericGroup)
	newSessionID := e2e.GetSessionIDFromBaseKeyForTesting(baseKey, t)

	// Check that this new session ID is now in the manager
	newSession := receivedManager.GetReceiveSession(newSessionID)
	if newSession == nil {
		t.Errorf("Did not get expected session")
	}

	// Generate a keypair alice will not recognize
	unknownPrivateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, genericGroup, csprng.NewSystemRNG())
	unknownPubliceKey := dh.GeneratePublicKey(unknownPrivateKey, genericGroup)

	// Generate a new session ID based off of these unrecognized keys
	badSessionID := e2e.GetSessionIDFromBaseKeyForTesting(unknownPubliceKey, t)

	// Check that this session with unrecognized keys is not valid
	badSession := receivedManager.GetReceiveSession(badSessionID)
	if badSession != nil {
		t.Errorf("Alice found a session from an unknown keypair. "+
			"\nSession: %v", badSession)
	}

}
