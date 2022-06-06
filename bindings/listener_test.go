package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

func TestMessage_Json(t *testing.T) {
	rng := csprng.NewSystemRNG()
	messageID := e2e.MessageID{}
	rng.Read(messageID[:])
	payload := make([]byte, 64)
	rng.Read(payload)
	sender := id.NewIdFromString("zezima", id.User, t)
	receiver := id.NewIdFromString("jakexx360", id.User, t)
	m := Message{
		MessageType: 1,
		ID:          messageID[:],
		Payload:     payload,
		Sender:      sender.Marshal(),
		RecipientID: receiver.Marshal(),
		EphemeralID: 17,
		Timestamp:   time.Now().UnixNano(),
		Encrypted:   false,
		RoundId:     19,
	}
	mm, _ := json.Marshal(m)
	t.Log("Marshalled Message")
	t.Log(string(mm))
}
