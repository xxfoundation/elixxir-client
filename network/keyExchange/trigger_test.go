package keyExchange

import (
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

// Smoke test for handleTrigger
func TestHandleTrigger(t *testing.T) {

	aliceContext := InitTestingContextGeneric(t)
	bobContext := InitTestingContextGeneric(t)

	bobID := id.NewIdFromBytes([]byte("test"), t)
	aliceContext.Session.E2e().AddPartner(bobID, bobContext.Session.E2e().GetDHPublicKey(),
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())

	sessionID := GeneratePartnerID(aliceContext, bobContext, genericGroup)

	pubKey := bobContext.Session.E2e().GetDHPrivateKey().Bytes()
	rekey, _ := proto.Marshal(&RekeyTrigger{
		SessionID: sessionID.Marshal(),
		PublicKey: pubKey,
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
	err := handleTrigger(aliceContext.Session, aliceContext.Manager, receiveMsg, nil)
	if err != nil {
		t.Errorf("Handle trigger error: %v", err)
	}
}
