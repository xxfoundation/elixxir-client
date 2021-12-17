////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"strings"
)

// Error messages.
const (
	// Manager.readMessage
	unmarshalPartMessageErr = "failed to unmarshal cMix message contents into file part message: %+v"
)

// receive runs a loop that receives file message parts and stores them in their
// appropriate transfer.
func (m *Manager) receive(rawMsgs chan message.Receive, stop *stoppable.Single) {
	jww.DEBUG.Print("Starting file part reception thread.")

	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Print("Stopping file part reception thread: stoppable " +
				"triggered.")
			stop.ToStopped()
			return
		case receiveMsg := <-rawMsgs:
			cMixMsg, err := m.readMessage(receiveMsg)
			if err != nil {
				// Print error as warning unless the fingerprint does not match,
				// which means this message is not of the correct type and will
				// be ignored
				if strings.Contains(err.Error(), "fingerprint") {
					jww.INFO.Print(err)
				} else {
					jww.WARN.Print(err)
				}
				continue
			}

			// Denote that the message is a file part
			m.store.GetGarbledMessages().Remove(cMixMsg)
		}
	}
}

// readMessage unmarshal the payload in the message.Receive and stores it with
// the appropriate received transfer. The cMix message is returned so that, on
// error, it can be either marked as used not used.
func (m *Manager) readMessage(msg message.Receive) (format.Message, error) {
	// Unmarshal payload into cMix message
	cMixMsg := format.Unmarshal(msg.Payload)

	// Unmarshal cMix message contents into a file part message
	partMsg, err := unmarshalPartMessage(cMixMsg.GetContents())
	if err != nil {
		return cMixMsg, errors.Errorf(unmarshalPartMessageErr, err)
	}

	// Add part to received transfer
	transfer, _, err := m.received.AddPart(partMsg.getPart(),
		partMsg.getPadding(), cMixMsg.GetMac(), partMsg.getPartNum(),
		cMixMsg.GetKeyFP())
	if err != nil {
		return cMixMsg, err
	}

	// Call callback with updates
	transfer.CallProgressCB(nil)

	return cMixMsg, nil
}
