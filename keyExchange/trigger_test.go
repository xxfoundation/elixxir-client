package keyExchange

import (
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage/e2e"
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

	// Maintain an ID for bob
	bobID := id.NewIdFromBytes([]byte("test"), t)

	// Add bob as a partner
	aliceSession.E2e().AddPartner(bobID, bobSession.E2e().GetDHPublicKey(),
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())

	// Generate a session ID, bypassing some business logic here
	sessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup)

	// Generate the message
	rekey, _ := proto.Marshal(&RekeyTrigger{
		SessionID: sessionID.Marshal(),
		PublicKey: bobPubKey.Bytes(),
	})
	payload := make([]byte, 0)

	payload = append(payload, rekey...)

	receiveMsg := message.Receive{
		Payload:     payload,
		MessageType: message.NoType,
		Sender:      bobID,
		Timestamp:   time.Now(),
		Encryption:  message.E2E,
	}

	// Handle the trigger and check for an error
	err := handleTrigger(aliceSession, aliceManager, receiveMsg)
	if err != nil {
		t.Errorf("Handle trigger error: %v", err)
	}
}
