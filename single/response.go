///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/interfaces/params"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"sync/atomic"
	"time"
)

// GetMaxResponsePayloadSize returns the maximum payload size for a response
// message.
func (m *Manager) GetMaxResponsePayloadSize() int {
	// Generate empty messages to determine the available space for the payload
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())
	responseMsg := newResponseMessagePart(cmixMsg.ContentsSize())

	return responseMsg.GetMaxContentsSize()
}

// RespondSingleUse creates the single-use response messages with the given
// payload and sends them to the given partner.
func (m *Manager) RespondSingleUse(partner Contact, payload []byte,
	timeout time.Duration) error {
	return m.respondSingleUse(partner, payload, timeout,
		m.net.GetInstance().GetRoundEvents())
}

// respondSingleUse allows for easier testing.
func (m *Manager) respondSingleUse(partner Contact, payload []byte,
	timeout time.Duration, roundEvents roundEvents) error {
	// Ensure that only a single reply can be sent in response
	firstUse := atomic.CompareAndSwapInt32(partner.used, 0, 1)
	if !firstUse {
		return errors.Errorf("cannot send to single-use contact that has " +
			"already been sent to.")
	}

	// Generate messages from payload
	msgs, err := m.makeReplyCmixMessages(partner, payload)
	if err != nil {
		return errors.Errorf("failed to create new CMIX messages: %+v", err)
	}

	jww.DEBUG.Printf("Created %d single-use response CMIX message parts.", len(msgs))

	// Tracks the numbers of rounds that messages are sent on
	rounds := make([]id.Round, len(msgs))

	sendResults := make(chan ds.EventReturn, len(msgs))

	// Send CMIX messages
	wg := sync.WaitGroup{}
	for i, cmixMsg := range msgs {
		wg.Add(1)
		cmixMsgFunc := cmixMsg
		j := i
		go func() {
			defer wg.Done()
			// Send Message
			p := params.GetDefaultCMIX()
			p.DebugTag = "single.Response"
			round, ephID, err := m.net.SendCMIX(cmixMsgFunc, partner.partner, p)
			if err != nil {
				jww.ERROR.Printf("Failed to send single-use response CMIX "+
					"message part %d: %+v", j, err)
			}
			jww.DEBUG.Printf("Sending single-use response CMIX message part "+
				"%d on round %d to address ID %d.", j, round, ephID.Int64())
			rounds[j] = round

			roundEvents.AddRoundEventChan(round, sendResults, timeout,
				states.COMPLETED, states.FAILED)
		}()
	}

	// Wait for all go routines to finish
	wg.Wait()
	jww.DEBUG.Printf("Sent %d single-use response CMIX messages to %s.", len(msgs), partner.partner)

	// Count the number of rounds
	roundMap := map[id.Round]struct{}{}
	for _, roundID := range rounds {
		roundMap[roundID] = struct{}{}
	}

	// Wait until the result tracking responds
	success, numRoundFail, numTimeOut := cmix.TrackResults(sendResults, len(roundMap))
	if !success {
		return errors.Errorf("tracking results of %d rounds: %d round "+
			"failures, %d round event time outs; the send cannot be retried.",
			len(rounds), numRoundFail, numTimeOut)
	}
	jww.DEBUG.Printf("Tracked %d single-use response message round(s).", len(roundMap))

	return nil
}

// makeReplyCmixMessages
func (m *Manager) makeReplyCmixMessages(partner Contact, payload []byte) ([]format.Message, error) {
	// Generate internal payloads based off key size to determine if the passed
	// in payload is too large to fit in the available contents
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())
	responseMsg := newResponseMessagePart(cmixMsg.ContentsSize())

	// Maximum payload size is the maximum amount of room in each message
	// multiplied by the number of messages
	maxPayloadSize := responseMsg.GetMaxContentsSize() * int(partner.GetMaxParts())

	if maxPayloadSize < len(payload) {
		return nil, errors.Errorf("length of provided payload (%d) too long "+
			"for message payload capacity (%d = %d byte payload * %d messages).",
			len(payload), maxPayloadSize, responseMsg.GetMaxContentsSize(),
			partner.GetMaxParts())
	}

	// Split payloads
	payloadParts := m.splitPayload(payload, responseMsg.GetMaxContentsSize(),
		int(partner.GetMaxParts()))

	// Create CMIX messages
	cmixMsgs := make([]format.Message, len(payloadParts))
	wg := sync.WaitGroup{}
	for i, contents := range payloadParts {
		wg.Add(1)
		go func(partner Contact, contents []byte, i uint8) {
			defer wg.Done()
			cmixMsgs[i] = m.makeMessagePart(partner, contents, uint8(len(payloadParts)), i)
		}(partner, contents, uint8(i))
	}

	// Wait for all go routines to finish
	wg.Wait()

	return cmixMsgs, nil
}

// makeMessagePart generates a CMIX message containing a responseMessagePart.
func (m *Manager) makeMessagePart(partner Contact, contents []byte, maxPart, i uint8) format.Message {
	cmixMsg := format.NewMessage(m.store.Cmix().GetGroup().GetP().ByteLen())
	responseMsg := newResponseMessagePart(cmixMsg.ContentsSize())

	// Compose response message
	responseMsg.SetMaxParts(maxPart)
	responseMsg.SetPartNum(i)
	responseMsg.SetContents(contents)

	// Encrypt payload
	fp := singleUse.NewResponseFingerprint(partner.dhKey, uint64(i))
	key := singleUse.NewResponseKey(partner.dhKey, uint64(i))
	encryptedPayload := cAuth.Crypt(key, fp[:24], responseMsg.Marshal())

	// Generate CMIX message MAC
	mac := singleUse.MakeMAC(key, encryptedPayload)

	// Compose CMIX message contents
	cmixMsg.SetContents(encryptedPayload)
	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetMac(mac)

	return cmixMsg
}

// splitPayload splits the given payload into separate payload parts and returns
// them in a slice. Each part's size is less than or equal to maxSize. Any extra
// data in the payload is not used if it is longer than the maximum capacity.
func (m *Manager) splitPayload(payload []byte, maxSize, maxParts int) [][]byte {
	var parts [][]byte
	buff := bytes.NewBuffer(payload)

	for i := 0; i < maxParts && buff.Len() > 0; i++ {
		parts = append(parts, buff.Next(maxSize))
	}
	return parts
}
