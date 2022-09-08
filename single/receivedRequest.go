////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	cmixMsg "gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/single/message"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Error messages.
const (
	// Request.Respond
	errSendResponse      = "%d responses failed to send, the response will be handleable and will time out"
	errUsed              = "cannot respond to single-use response that has already been responded to"
	errMaxResponseLength = "length of provided payload %d greater than max payload %d"
	errTrackResults      = "tracking results of %d rounds: %d round failures, %d round event time-outs; the send cannot be retried."
)

// Request contains the information contained in a single-use request message.
type Request struct {
	sender         *id.ID      // ID of the sender/ID to send response to
	senderPubKey   *cyclic.Int // Public key of the sender
	dhKey          *cyclic.Int // DH key
	tag            string      // Identifies which callback to use
	maxParts       uint8       // Max number of messages allowed in reply
	used           *uint32     // Set when response is sent
	requestPayload []byte      // Request message payload

	net RequestCmix
}

// Respond is used to respond to the request. It sends a payload up to
// Request.GetMaxResponseLength. It will chunk the message into multiple cMix
// messages if it is too long for a single message. It will fail if a single
// cMix message cannot be sent.
func (r *Request) Respond(payload []byte, cMixParams cmix.CMIXParams,
	timeout time.Duration) ([]id.Round, error) {
	// Make sure this has only been run once
	newRun := atomic.CompareAndSwapUint32(r.used, 0, 1)
	if !newRun {
		return nil, errors.New(errUsed)
	}

	// Check that the payload is not too long
	if len(payload) > r.GetMaxResponseLength() {
		return nil, errors.Errorf(
			errMaxResponseLength, len(payload), r.GetMaxResponseLength())
	}

	// Partition the payload
	parts := partitionResponse(payload, r.net.GetMaxMessageLength(), r.maxParts)

	// Encrypt and send the partitions
	cyphers := makeCyphers(r.dhKey, uint8(len(parts)),
		singleUse.NewResponseKey, singleUse.NewResponseFingerprint)
	rounds := make([]id.Round, len(parts))
	sendResults := make(chan ds.EventReturn, len(parts))

	if cMixParams.DebugTag == cmix.DefaultDebugTag || cMixParams.DebugTag == "" {
		cMixParams.DebugTag = "single-use.Response"
	}

	var wg sync.WaitGroup
	wg.Add(len(parts))
	failed := uint32(0)

	jww.INFO.Printf("[SU] Sending single-use response cMix message with %d "+
		"parts to %s (%s)", len(parts), r.sender, r.tag)

	for i := 0; i < len(parts); i++ {
		go func(i int, part []byte) {
			defer wg.Done()
			partFP, ecrPart, mac := cyphers[i].encrypt(part)

			// Send Message
			round, ephID, err := r.net.Send(
				r.sender, partFP, cmixMsg.Service{}, ecrPart, mac, cMixParams)
			if err != nil {
				atomic.AddUint32(&failed, 1)
				jww.ERROR.Printf("[SU] Failed to send single-use response "+
					"cMix message part %d of %d to %s (%s): %+v",
					i, len(parts), r.sender, r.tag, err)
				return
			}

			jww.DEBUG.Printf("[SU] Sent single-use response cMix message part "+
				"%d of %d on round %d to %s (eph ID %d) (%s).",
				i, len(parts), round, r.sender, ephID.Int64(), r.tag)
			rounds[i] = round.ID

			r.net.GetInstance().GetRoundEvents().AddRoundEventChan(
				round.ID, sendResults, timeout, states.COMPLETED, states.FAILED)
		}(i, parts[i].Marshal())
	}

	// Wait for all go routines to finish
	wg.Wait()

	if failed > 0 {
		return nil, errors.Errorf(errSendResponse, failed)
	}

	jww.INFO.Printf("[SU] Sent single-use response cMix message with %d "+
		"parts to %s (%s).", len(parts), r.sender, r.tag)

	// Count the number of rounds
	roundMap := map[id.Round]struct{}{}
	for _, roundID := range rounds {
		if roundID != 0 {
			roundMap[roundID] = struct{}{}
		}
	}

	// Wait until the result tracking responds
	success, numRoundFail, numTimeOut := cmix.TrackResults(
		sendResults, len(roundMap))
	if !success {
		return nil, errors.Errorf(
			errTrackResults, len(rounds), numRoundFail, numTimeOut)
	}

	jww.DEBUG.Printf("[SU] Tracked %d single-use response message round(s).",
		len(roundMap))

	return rounds, nil
}

// GetMaxParts returns the maximum number of messages allowed to send in the
// reply.
func (r *Request) GetMaxParts() uint8 {
	return r.maxParts
}

// GetMaxResponseLength returns the maximum size of the entire response message.
func (r *Request) GetMaxResponseLength() int {
	return r.GetMaxResponsePartSize() * int(r.GetMaxParts())
}

// GetMaxResponsePartSize returns maximum payload size for an individual part of
// the response message.
func (r *Request) GetMaxResponsePartSize() int {
	responseMsg := message.NewResponsePart(r.net.GetMaxMessageLength())
	return responseMsg.GetMaxContentsSize()
}

// GetPartner returns a copy of the sender ID.
func (r *Request) GetPartner() *id.ID {
	return r.sender.DeepCopy()
}

// GetTag returns the tag for the request.
func (r *Request) GetTag() string {
	return r.tag
}

// GetPayload returns the payload that came in the request
func (r *Request) GetPayload() []byte {
	return r.requestPayload
}

// GoString returns string showing the values of all the fields of Request.
// Adheres to the fmt.GoStringer interface.
func (r *Request) GoString() string {
	return fmt.Sprintf(
		"{sender:%s senderPubKey:%s dhKey:%s tag:%q maxParts:%d used:%p(%d) "+
			"requestPayload:%q net:%p}",
		r.sender, r.senderPubKey.Text(10), r.dhKey.Text(10), r.tag, r.maxParts,
		r.used, atomic.LoadUint32(r.used), r.requestPayload, r.net)
}

// partitionResponse breaks a payload into its sub payloads for sending.
func partitionResponse(payload []byte, cmixMessageLength int,
	maxParts uint8) []message.ResponsePart {
	responseMsg := message.NewResponsePart(cmixMessageLength)

	// Split payloads
	payloadParts := splitPayload(
		payload, responseMsg.GetMaxContentsSize(), int(maxParts))

	// Create messages
	parts := make([]message.ResponsePart, len(payloadParts))
	for i := range payloadParts {
		nrp := message.NewResponsePart(cmixMessageLength)
		nrp.SetPartNum(uint8(i))
		nrp.SetContents(payloadParts[i])
		nrp.SetNumParts(uint8(len(payloadParts)))
		parts[i] = nrp
	}

	return parts
}

// splitPayload splits the given payload into separate payload parts and returns
// them in a slice. Each part's size is less than or equal to maxSize. Any extra
// data in the payload is not used if it is longer than the maximum capacity.
func splitPayload(payload []byte, maxSize, maxParts int) [][]byte {
	parts := make([][]byte, 0, len(payload)/maxSize)
	buff := bytes.NewBuffer(payload)

	for i := 0; i < maxParts && buff.Len() > 0; i++ {
		parts = append(parts, buff.Next(maxSize))
	}
	return parts
}

// BuildTestRequest can be used for mocking a Request. Should only be used for
// tests.
func BuildTestRequest(payload []byte, _ testing.TB) *Request {
	return &Request{
		sender:         nil,
		senderPubKey:   nil,
		dhKey:          nil,
		tag:            "",
		maxParts:       0,
		used:           nil,
		requestPayload: payload,
		net:            nil,
	}
}
