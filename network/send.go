////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (m *Manager) SendE2E(msg message.Send, e2eP params.E2E, cmixP params.CMIX) (
	[]id.Round, error) {

	if !m.health.IsRunning() {
		return nil, errors.New("Cannot send e2e message when the " +
			"network is not healthy")
	}


	return nil, nil
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (m *Manager) SendUnsafe(m message.Send) ([]id.Round, error) {
	return nil, nil
}

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (m *Manager) SendCMIX(msg format.Message) (id.Round, error) {
	if !m.health.IsRunning() {
		return 0, errors.New("Cannot send cmix message when the " +
			"network is not healthy")
	}

	return m.sendCMIX(msg)
}

// Internal send e2e which bypasses the network check, for use in SendE2E and
// SendUnsafe which do their own network checks
func (m *Manager) sendCMIX(message format.Message) (id.Round, error) {

	//get the round to send on
	m.

	return 0, nil
}