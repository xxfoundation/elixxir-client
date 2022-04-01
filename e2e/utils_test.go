package e2e

import (
	"bytes"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
)

func e2eMessagesEqual(received, expected e2eMessage, t *testing.T) bool {
	equals := true
	if !bytes.Equal(received.Recipient, expected.Recipient) {
		t.Errorf("Receipient values for messages are not equivalent")
		equals = false
	}

	if !bytes.Equal(received.Payload, expected.Payload) {
		equals = false
		t.Errorf("Payload values for messages are not equivalent")
	}

	if received.MessageType != expected.MessageType {
		equals = false
		t.Errorf("MessageType values for messages are not equivalent")
	}

	return equals

}

// makeTestE2EMessages creates a list of messages with random data and the
// expected map after they are added to the buffer.
func makeTestE2EMessages(n int, t *testing.T) []e2eMessage {
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	msgs := make([]e2eMessage, n)
	for i := range msgs {
		rngBytes := make([]byte, 128)
		prng.Read(rngBytes)
		msgs[i].Recipient = id.NewIdFromBytes(rngBytes, t).Bytes()
		prng.Read(rngBytes)
		msgs[i].Payload = rngBytes
		prng.Read(rngBytes)
		msgs[i].MessageType = uint32(rngBytes[0])
	}

	return msgs
}
