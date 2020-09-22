package network

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	jww "github.com/spf13/jwalterweatherman"
)

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (m *Manager) SendCMIX(msg format.Message, param params.CMIX) (id.Round, error) {
	if !m.m.Health.IsRunning() {
		return 0, errors.New("Cannot send cmix message when the " +
			"network is not healthy")
	}

	return m.m.message.SendCMIX(msg, param)
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (m *Manager) SendUnsafe(msg message.Send, param params.Unsafe) ([]id.Round, error) {
	if !m.m.Health.IsRunning() {
		return nil, errors.New("cannot send unsafe message when the " +
			"network is not healthy")
	}

	jww.WARN.Println("Sending unsafe message. Unsafe payloads have no end" +
		" to end encryption, they have limited security and privacy " +
		"preserving properties")

	return m.SendUnsafe(msg, param)
}

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (m *Manager) SendE2E(msg message.Send, e2eP params.E2E) (
	[]id.Round, error) {

	if !m.m.Health.IsRunning() {
		return nil, errors.New("Cannot send e2e message when the " +
			"network is not healthy")
	}

	return m.m.message.SendE2E(msg, e2eP)
}
