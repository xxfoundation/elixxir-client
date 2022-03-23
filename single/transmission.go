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
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	ephemeral2 "gitlab.com/elixxir/client/network/identity/receptionID"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	contact2 "gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"sync/atomic"
	"time"
)

// GetMaxTransmissionPayloadSize returns the maximum payload size for a
// transmission message.
func (m *Manager) GetMaxTransmissionPayloadSize() int {
	// Generate empty messages to determine the available space for the payload
	cmixPrimeSize := m.store.Cmix().GetGroup().GetP().ByteLen()
	e2ePrimeSize := m.store.E2e().GetGroup().GetP().ByteLen()
	cmixMsg := format.NewMessage(cmixPrimeSize)
	transmitMsg := newTransmitMessage(cmixMsg.ContentsSize(), e2ePrimeSize)
	msgPayload := newTransmitMessagePayload(transmitMsg.GetPayloadSize())

	return msgPayload.GetMaxContentsSize()
}

// TransmitSingleUse creates a CMIX message, sends it, and waits for delivery.
func (m *Manager) TransmitSingleUse(partner contact2.Contact, payload []byte,
	tag string, maxMsgs uint8, callback ReplyComm, timeout time.Duration) error {

	rngReader := m.rng.GetStream()
	defer rngReader.Close()

	return m.transmitSingleUse(partner, payload, tag, maxMsgs, rngReader, callback, timeout)
}

// roundEvents interface allows custom round events to be passed in for testing.
type roundEvents interface {
	AddRoundEventChan(id.Round, chan ds.EventReturn, time.Duration,
		...states.Round) *ds.EventCallback
}

// transmitSingleUse has the fields passed in for easier testing.
func (m *Manager) transmitSingleUse(partner contact2.Contact, payload []byte,
	tag string, MaxMsgs uint8, rng io.Reader, callback ReplyComm, timeout time.Duration) error {

	// get address ID address space size; this blocks until the address space
	// size is set for the first time
	addressSize := m.net.GetAddressSize()
	timeStart := netTime.Now()

	// Create new CMIX message containing the transmission payload
	cmixMsg, dhKey, rid, ephID, err := m.makeTransmitCmixMessage(partner,
		payload, tag, MaxMsgs, addressSize, timeout, timeStart, rng)
	if err != nil {
		return errors.Errorf("failed to create new CMIX message: %+v", err)
	}

	startValid := timeStart.Add(-2 * timeout)
	endValid := timeStart.Add(2 * timeout)
	jww.DEBUG.Printf("Created single-use transmission CMIX message with new ID "+
		"%s and EphID %d (Valid %s - %s)", rid, ephID.Int64(), startValid.String(), endValid.String())

	// Add message state to map
	quitChan, quit, err := m.p.addState(rid, dhKey, MaxMsgs, callback, timeout)
	if err != nil {
		return errors.Errorf("failed to add pending state: %+v", err)
	}

	// Add identity for newly generated ID
	err = m.reception.AddIdentity(ephemeral2.Identity{
		EphId:       ephID,
		Source:      rid,
		AddressSize: addressSize,
		End:         endValid,
		ExtraChecks: interfaces.DefaultExtraChecks,
		StartValid:  startValid,
		EndValid:    endValid,
		Ephemeral:   true,
	})
	if err != nil {
		errorString := fmt.Sprintf("failed to add new identity to "+
			"reception storage for single-use: %+v", err)
		jww.ERROR.Print(errorString)

		// Exit the state timeout handler, delete the state from map, and
		// return an error on the callback
		quitChan <- struct{}{}
		m.p.Lock()
		delete(m.p.singleUse, *rid)
		m.p.Unlock()
		go callback(nil, errors.New(errorString))
	}

	go func() {
		// Send Message
		jww.DEBUG.Printf("Sending single-use transmission CMIX "+
			"message to %s.", partner.ID)
		p := params.GetDefaultCMIX()
		p.DebugTag = "single.Transmit"
		round, _, err := m.net.SendCMIX(cmixMsg, partner.ID, p)
		if err != nil {
			errorString := fmt.Sprintf("failed to send single-use transmission "+
				"CMIX message: %+v", err)
			jww.ERROR.Printf(errorString)

			// Exit the state timeout handler, delete the state from map, and
			// return an error on the callback
			quitChan <- struct{}{}
			m.p.Lock()
			delete(m.p.singleUse, *rid)
			m.p.Unlock()
			go callback(nil, errors.New(errorString))
		}

		// Check if the state timeout handler has quit
		if atomic.LoadInt32(quit) == 1 {
			jww.ERROR.Print("Stopping to send single-use transmission CMIX " +
				"message because the timeout handler quit.")
			return
		}
		jww.DEBUG.Printf("Sent single-use transmission CMIX "+
			"message to %s and address ID %d on round %d.",
			partner.ID, ephID.Int64(), round)
	}()

	return nil
}

// makeTransmitCmixMessage generates a CMIX message containing the transmission message,
// which contains the encrypted payload.
func (m *Manager) makeTransmitCmixMessage(partner contact2.Contact,
	payload []byte, tag string, maxMsgs uint8, addressSize uint8,
	timeout time.Duration, timeNow time.Time, rng io.Reader) (format.Message,
	*cyclic.Int, *id.ID, ephemeral.Id, error) {
	e2eGrp := m.store.E2e().GetGroup()

	// Generate internal payloads based off key size to determine if the passed
	// in payload is too large to fit in the available contents
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())
	transmitMsg := newTransmitMessage(cmixMsg.ContentsSize(), e2eGrp.GetP().ByteLen())
	msgPayload := newTransmitMessagePayload(transmitMsg.GetPayloadSize())

	if msgPayload.GetMaxContentsSize() < len(payload) {
		return format.Message{}, nil, nil, ephemeral.Id{},
			errors.Errorf("length of provided payload (%d) too long for message "+
				"payload capacity (%d).", len(payload), len(msgPayload.contents))
	}

	// Generate DH key and public key
	dhKey, publicKey, err := generateDhKeys(e2eGrp, partner.DhPubKey, rng)
	if err != nil {
		return format.Message{}, nil, nil, ephemeral.Id{}, err
	}

	// Compose payload
	msgPayload.SetTagFP(singleUse.NewTagFP(tag))
	msgPayload.SetMaxParts(maxMsgs)
	msgPayload.SetContents(payload)

	// Generate new user ID and address ID
	rid, ephID, err := makeIDs(&msgPayload, publicKey, addressSize, timeout,
		timeNow, rng)
	if err != nil {
		return format.Message{}, nil, nil, ephemeral.Id{},
			errors.Errorf("failed to generate IDs: %+v", err)
	}

	// Encrypt payload
	fp := singleUse.NewTransmitFingerprint(partner.DhPubKey)
	key := singleUse.NewTransmitKey(dhKey)
	encryptedPayload := auth.Crypt(key, fp[:24], msgPayload.Marshal())

	// Generate CMIX message MAC
	mac := singleUse.MakeMAC(key, encryptedPayload)

	// Compose transmission message
	transmitMsg.SetPubKey(publicKey)
	transmitMsg.SetPayload(encryptedPayload)

	// Compose CMIX message contents
	cmixMsg.SetContents(transmitMsg.Marshal())
	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetMac(mac)

	return cmixMsg, dhKey, rid, ephID, nil
}

// generateDhKeys generates a new public key and DH key.
func generateDhKeys(grp *cyclic.Group, dhPubKey *cyclic.Int,
	rng io.Reader) (*cyclic.Int, *cyclic.Int, error) {
	// Generate private key
	privKeyBytes, err := csprng.GenerateInGroup(grp.GetP().Bytes(),
		grp.GetP().ByteLen(), rng)
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate key in group: %+v",
			err)
	}
	privKey := grp.NewIntFromBytes(privKeyBytes)

	// Generate public key and DH key
	publicKey := grp.ExpG(privKey, grp.NewInt(1))
	dhKey := grp.Exp(dhPubKey, privKey, grp.NewInt(1))

	return dhKey, publicKey, nil
}

// makeIDs generates a new user ID and address ID with a start and end within
// the given timout. The ID is generated from the unencrypted msg payload, which
// contains a nonce. If the generated address ID has a window that is not
// within +/- the given 2*timeout from now, then the IDs are generated again
// using a new nonce.
func makeIDs(msg *transmitMessagePayload, publicKey *cyclic.Int,
	addressSize uint8, timeout time.Duration, timeNow time.Time,
	rng io.Reader) (*id.ID, ephemeral.Id, error) {
	var rid *id.ID
	var ephID ephemeral.Id

	// Generate acceptable window for the address ID to exist in
	windowStart, windowEnd := timeNow.Add(-2*timeout), timeNow.Add(2*timeout)
	start, end := timeNow, timeNow

	// Loop until the address ID's start and end are within bounds
	for windowStart.Before(start) || windowEnd.After(end) {
		// Generate new nonce
		err := msg.SetNonce(rng)
		if err != nil {
			return nil, ephemeral.Id{},
				errors.Errorf("failed to generate nonce: %+v", err)
		}

		// Generate ID from unencrypted payload
		rid = msg.GetRID(publicKey)

		// Generate the address ID
		ephID, start, end, err = ephemeral.GetId(rid, uint(addressSize), timeNow.UnixNano())
		if err != nil {
			return nil, ephemeral.Id{}, errors.Errorf("failed to generate "+
				"address ID from newly generated ID: %+v", err)
		}
		jww.DEBUG.Printf("address.GetId(%s, %d, %d) = %d", rid, addressSize, timeNow.UnixNano(), ephID.Int64())
	}

	jww.INFO.Printf("generated by singe use sender reception id for single use: %s, "+
		"ephId: %d, pubkey: %x, msg: %s", rid, ephID.Int64(), publicKey.Bytes(), msg)

	return rid, ephID, nil
}
