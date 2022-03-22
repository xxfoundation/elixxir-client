///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"strings"
)

// receiveResponseHandler handles the reception of single-use response messages.
func (m *Manager) receiveResponseHandler(rawMessages chan message.Receive,
	stop *stoppable.Single) {
	jww.DEBUG.Print("Waiting to receive single-use response messages.")
	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Printf("Stopping waiting to receive single-use " +
				"response message.")
			stop.ToStopped()
			return
		case msg := <-rawMessages:
			jww.TRACE.Printf("Received CMIX message; checking if it is a " +
				"single-use response.")

			// Process CMIX message
			err := m.processesResponse(msg.RecipientID, msg.EphemeralID, msg.Payload)
			if err != nil {
				em := fmt.Sprintf("Failed to read single-use "+
					"CMIX message response: %+v", err)
				if strings.Contains(err.Error(), "no state exists for the reception ID") {
					jww.TRACE.Print(em)
				} else {
					if m.client != nil {
						m.client.ReportEvent(9, "SingleUse",
							"Error", em)
					}
				}
			}
		}
	}
}

// processesResponse processes the CMIX message and collates its payload. If the
// message is invalid, an error is returned.
func (m *Manager) processesResponse(rid *id.ID, ephID ephemeral.Id,
	msgBytes []byte) error {

	// get the state from the map
	m.p.RLock()
	state, exists := m.p.singleUse[*rid]
	m.p.RUnlock()

	// Check that the state exists
	if !exists {
		return errors.Errorf("no state exists for the reception ID %s.", rid)
	}

	// Unmarshal CMIX message
	cmixMsg, err := format.Unmarshal(msgBytes)
	if err != nil {
		return err
	}

	// Ensure the fingerprints match
	fp := cmixMsg.GetKeyFP()
	key, exists := state.fpMap.getKey(state.dhKey, fp)
	if !exists {
		return errors.New("message fingerprint does not correspond to the " +
			"expected fingerprint.")
	}

	// Verify the CMIX message MAC
	if !singleUse.VerifyMAC(key, cmixMsg.GetContents(), cmixMsg.GetMac()) {
		return errors.New("failed to verify the CMIX message MAC.")
	}

	// Denote that the message is not garbled
	jww.DEBUG.Print("Received single-use response message.")
	m.store.GetGarbledMessages().Remove(cmixMsg)

	// Decrypt and collate the payload
	decryptedPayload := auth.Crypt(key, fp[:24], cmixMsg.GetContents())
	collatedPayload, collated, err := state.c.collate(decryptedPayload)
	if err != nil {
		return errors.Errorf("failed to collate payload: %+v", err)
	}
	jww.DEBUG.Print("Successfully processed single-use response message part.")

	// Once all message parts have been received delete and close everything
	if collated {
		if m.client != nil {
			m.client.ReportEvent(1, "SingleUse", "MessageReceived",
				fmt.Sprintf("Single use response received "+
					"from %s", rid))
		}
		jww.DEBUG.Print("Received all parts of single-use response message.")
		// Exit the timeout handler
		state.quitChan <- struct{}{}

		// Remove identity
		m.reception.RemoveIdentity(ephID)

		// Remove state from map
		m.p.Lock()
		delete(m.p.singleUse, *rid)
		m.p.Unlock()

		// Call in separate routine to prevent blocking
		jww.DEBUG.Print("Calling single-use response message callback.")
		go state.callback(collatedPayload, nil)
	}

	return nil
}
