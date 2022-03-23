////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
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
	jww.DEBUG.Print("[FT] Starting file part reception thread.")

	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Print("[FT] Stopping file part reception thread: stoppable " +
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
					jww.TRACE.Printf("[FT] %v", err)
				} else {
					jww.WARN.Printf("[FT] %v", err)
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
	cMixMsg, err := format.Unmarshal(msg.Payload)
	if err != nil {
		return cMixMsg, err
	}

	// Add part to received transfer
	rt, tid, completed, err := m.received.AddPart(cMixMsg)
	if err != nil {
		return cMixMsg, err
	}

	// Print debug message on completion
	if completed {
		jww.DEBUG.Printf("[FT] Received last part for file transfer %s from "+
			"%s {size: %d, parts: %d, numFps: %d/%d}", tid, msg.Sender,
			rt.GetFileSize(), rt.GetNumParts(),
			rt.GetNumFps()-rt.GetNumAvailableFps(), rt.GetNumFps())
	}

	// Call callback with updates
	rt.CallProgressCB(nil)

	return cMixMsg, nil
}
