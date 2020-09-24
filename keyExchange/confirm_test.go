package keyExchange

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces/message"
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
	aliceSession.E2e().AddPartner(bobID, bobPubKey,
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())

	// Generate a session ID, bypassing some business logic here
	sessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup)

	// Generate the message
	rekey, _ := proto.Marshal(&RekeyConfirm{
		SessionID: sessionID.Marshal(),
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

	// Handle the confirmation
	handleConfirm(aliceSession, receiveMsg)
}
