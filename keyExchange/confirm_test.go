package keyExchange

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Smoke test for handleTrigger
func TestHandleConfirm(t *testing.T) {

	aliceContext := InitTestingContextGeneric(t)
	bobContext := InitTestingContextGeneric(t)

	bobID := id.NewIdFromBytes([]byte("test"), t)

	aliceContext.Session.E2e().AddPartner(bobID, bobContext.Session.E2e().GetDHPublicKey(),
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())
	sessionID := GeneratePartnerID(aliceContext, bobContext, genericGroup)

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
	handleConfirm(aliceContext.Session, receiveMsg)
}
