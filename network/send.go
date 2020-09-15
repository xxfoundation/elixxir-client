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
func (m *Manager) SendUnsafe(msg message.Send) ([]id.Round, error) {
	return nil, nil
}

