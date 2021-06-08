///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
)

// receiveTransmissionHandler waits to receive single-use transmissions. When
// a message is received, its is returned via its registered callback.
func (m *Manager) receiveTransmissionHandler(rawMessages chan message.Receive,
	stop *stoppable.Single) {
	fp := singleUse.NewTransmitFingerprint(m.store.E2e().GetDHPublicKey())
	jww.DEBUG.Print("Waiting to receive single-use transmission messages.")
	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Printf("Stopping waiting to receive single-use " +
				"transmission message.")
			stop.ToStopped()
			return
		case msg := <-rawMessages:
			jww.DEBUG.Printf("Received CMIX message; checking if it is a " +
				"single-use transmission.")

			// Check if message is a single-use transmit message
			cmixMsg := format.Unmarshal(msg.Payload)
			if fp != cmixMsg.GetKeyFP() {
				// If the verification fails, then ignore the message as it is
				// likely garbled or for a different protocol
				jww.INFO.Print("Failed to read single-use CMIX message: " +
					"fingerprint verification failed.")
				continue
			}

			// Denote that the message is not garbled
			jww.DEBUG.Printf("Received single-use transmission message.")
			m.store.GetGarbledMessages().Remove(cmixMsg)

			// Handle message
			payload, c, err := m.processTransmission(cmixMsg, fp)
			if err != nil {
				jww.WARN.Printf("Failed to read single-use CMIX message: %+v",
					err)
				continue
			}
			jww.DEBUG.Printf("Successfully processed single-use transmission message.")

			// Lookup the registered callback for the message's tag fingerprint
			callback, err := m.callbackMap.getCallback(c.tagFP)
			if err != nil {
				jww.WARN.Printf("Failed to find module to pass single-use "+
					"payload: %+v", err)
				continue
			}

			jww.DEBUG.Printf("Calling single-use callback with tag "+
				"fingerprint %s.", c.tagFP)

			go callback(payload, c)
		}
	}
}

// processTransmission unmarshalls and decrypts the message payload and
// returns the decrypted payload and the contact information for the sender.
func (m *Manager) processTransmission(msg format.Message,
	fp format.Fingerprint) ([]byte, Contact, error) {
	grp := m.store.E2e().GetGroup()
	dhPrivKey := m.store.E2e().GetDHPrivateKey()

	// Unmarshal the CMIX message contents to a transmission message
	transmitMsg, err := unmarshalTransmitMessage(msg.GetContents(),
		grp.GetP().ByteLen())
	if err != nil {
		return nil, Contact{}, errors.Errorf("failed to unmarshal contents: %+v", err)
	}

	// Generate DH key and symmetric key
	dhKey := grp.Exp(transmitMsg.GetPubKey(grp), dhPrivKey, grp.NewInt(1))
	key := singleUse.NewTransmitKey(dhKey)

	// Verify the MAC
	if !singleUse.VerifyMAC(key, transmitMsg.GetPayload(), msg.GetMac()) {
		return nil, Contact{}, errors.New("failed to verify MAC.")
	}

	// Decrypt the transmission message payload
	decryptedPayload := cAuth.Crypt(key, fp[:24], transmitMsg.GetPayload())

	// Unmarshal the decrypted payload to a transmission message payload
	payload, err := unmarshalTransmitMessagePayload(decryptedPayload)
	if err != nil {
		return nil, Contact{}, errors.Errorf("failed to unmarshal payload: %+v", err)
	}

	c := NewContact(payload.GetRID(transmitMsg.GetPubKey(grp)),
		transmitMsg.GetPubKey(grp), dhKey, payload.GetTagFP(), payload.GetMaxParts())

	return payload.GetContents(), c, nil
}
