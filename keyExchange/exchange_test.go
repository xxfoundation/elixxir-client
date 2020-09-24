package keyExchange

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/client/switchboard"
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

	// Add Alice and Bob as partners
	aliceSession.E2e().AddPartner(exchangeBobId, bobPubKey,
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())
	bobSession.E2e().AddPartner(exchangeAliceId, alicePubKey,
		e2e.GetDefaultSessionParams(), e2e.GetDefaultSessionParams())

	// Start the listeners for alice and bob
	Start(aliceSwitchboard, aliceSession, aliceManager)
	Start(bobSwitchboard, bobSession, bobManager)

	// Generate a session ID, bypassing some business logic here
	sessionID := GeneratePartnerID(alicePrivKey, bobPubKey, genericGroup)

	// Generate the message
	rekeyTrigger, _ := proto.Marshal(&RekeyTrigger{
		SessionID: sessionID.Marshal(),
		PublicKey: bobPubKey.Bytes(),
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

	// Speak the message to Bob, triggers the SendE2E in utils_test
	aliceSwitchboard.Speak(triggerMsg)

	// Allow the test time to work it's goroutines
	time.Sleep(1 * time.Second)

}
