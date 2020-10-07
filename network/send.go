package network

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (m *manager) SendCMIX(msg format.Message, param params.CMIX) (id.Round, error) {
	if !m.Health.IsHealthy() {
		return 0, errors.New("Cannot send cmix message when the " +
			"network is not healthy")
	}

	return m.message.SendCMIX(msg, param)
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (m *manager) SendUnsafe(msg message.Send, param params.Unsafe) ([]id.Round, error) {
	if !m.Health.IsHealthy() {
		return nil, errors.New("cannot send unsafe message when the " +
			"network is not healthy")
	}

	jww.WARN.Println("Sending unsafe message. Unsafe payloads have no end" +
		" to end encryption, they have limited security and privacy " +
		"preserving properties")

	return m.message.SendUnsafe(msg, param)
}

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (m *manager) SendE2E(msg message.Send, e2eP params.E2E) (
	[]id.Round, e2e.MessageID, error) {

	if !m.Health.IsHealthy() {
		return nil, e2e.MessageID{}, errors.New("Cannot send e2e " +
			"message when the network is not healthy")
	}

	return m.message.SendE2E(msg, e2eP)
}
