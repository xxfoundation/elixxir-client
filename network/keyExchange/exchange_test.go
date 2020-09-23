package keyExchange

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

var exchangeAliceContext, exchangeBobContext *context.Context
var exchangeAliceId, exchangeBobId *id.ID


func TestFullExchange(t *testing.T) {

	exchangeAliceContext = InitTestingContextFullExchange(t)
	exchangeBobContext = InitTestingContextFullExchange(t)
	exchangeAliceId = id.NewIdFromBytes([]byte("1234"), t)
	exchangeBobId   = id.NewIdFromBytes([]byte("test"), t)
	exchangeAliceContext.Session.E2e().AddPartner(exchangeBobId, exchangeBobContext.Session.E2e().GetDHPublicKey(),
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())
	exchangeBobContext.Session.E2e().AddPartner(exchangeAliceId, exchangeAliceContext.Session.E2e().GetDHPublicKey(),
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())


	Start(exchangeAliceContext.Switchboard, exchangeAliceContext.Session,
		exchangeAliceContext.Manager, nil)

	Start(exchangeBobContext.Switchboard, exchangeBobContext.Session,
		exchangeBobContext.Manager, nil)



	sessionID := GeneratePartnerID(exchangeAliceContext, exchangeBobContext, genericGroup)

	pubKey := exchangeBobContext.Session.E2e().GetDHPrivateKey().Bytes()
	rekeyTrigger, _ := proto.Marshal(&RekeyTrigger{
		SessionID: sessionID.Marshal(),
		PublicKey: pubKey,
	})
	payload := make([]byte, 0)

	payload = append(payload, rekeyTrigger...)

	triggerMsg := message.Receive{
		Payload:     payload,
		MessageType: message.KeyExchangeTrigger,
		Sender:      exchangeBobId,
		Timestamp:   time.Now(),
		Encryption:  message.E2E,
	}
	exchangeAliceContext.Switchboard.Speak(triggerMsg)



	time.Sleep(1*time.Second)


}
