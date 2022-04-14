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
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"sync/atomic"
	"time"
)

// Request contains the information to respond to a single-use contact.
type Request struct {
	sender         *id.ID      // ID of the person to respond to
	senderPubKey   *cyclic.Int // Public key of the sender
	dhKey          *cyclic.Int // DH key
	tag            string      // Identifies which callback to use
	maxParts       uint8       // Max number of messages allowed in reply
	used           *uint32     // Atomic variable
	requestPayload []byte
	net            cmix.Client
}

// GetMaxParts returns the maximum number of message parts that can be sent in a
// reply.
func (r Request) GetMaxParts() uint8 {
	return r.maxParts
}

func (r Request) GetMaxResponseLength() int {
	responseMsg := message.NewResponsePart(r.net.GetMaxMessageLength())

	// Maximum payload size is the maximum amount of room in each message
	// multiplied by the number of messages
	return responseMsg.GetMaxContentsSize() * int(r.GetMaxParts())
}

// GetPartner returns a copy of the sender ID.
func (r Request) GetPartner() *id.ID {
	return r.sender.DeepCopy()
}

// GetTag returns the tag for the request.
func (r Request) GetTag() string {
	return r.tag
}

// GetPayload returns the payload that came in the request
func (r Request) GetPayload() []byte {
	return r.requestPayload
}

// String returns a string of the Contact structure.
func (r Request) String() string {
	format := "Request{sender:%s  senderPubKey:%s  dhKey:%s  tagFP:%s  " +
		"maxParts:%d}"
	return fmt.Sprintf(format, r.sender, r.senderPubKey.Text(10),
		r.dhKey.Text(10), r.tag, r.maxParts)
}

// Respond is used to respond to the request. It sends a payload up to
// r.GetMaxResponseLength(). It will chunk the message into multiple
// cmix messages if it is too long for a single message. It will fail
// If a single cmix message cannot be sent.
func (r Request) Respond(payload []byte, cmixParams cmix.CMIXParams,
	timeout time.Duration) ([]id.Round, error) {
	// make sure this has only been run once
	newRun := atomic.CompareAndSwapUint32(r.used, 0, 1)
	if !newRun {
		return nil, errors.Errorf("cannot respond to " +
			"single-use response that has already been responded to.")
	}

	//check that the payload isn't too long
	if len(payload) > r.GetMaxResponseLength() {
		return nil, errors.Errorf("length of provided "+
			"payload too long for message payload capacity, max: %d, "+
			"received: %d", r.GetMaxResponseLength(),
			len(payload))
	}

	//partition the payload
	parts := partitionResponse(payload, r.net.GetMaxMessageLength(), r.maxParts)

	//encrypt and send the partitions
	cyphers := makeCyphers(r.dhKey, uint8(len(parts)))
	rounds := make([]id.Round, len(parts))
	sendResults := make(chan ds.EventReturn, len(parts))

	wg := sync.WaitGroup{}
	wg.Add(len(parts))

	if cmixParams.DebugTag != cmix.DefaultDebugTag {
		cmixParams.DebugTag = "single.Response"
	}

	svc := cmixMsg.Service{
		Identifier: r.dhKey.Bytes(),
		Tag:        "single.response-dummyservice",
		Metadata:   nil,
	}

	failed := uint32(0)

	for i := 0; i < len(parts); i++ {
		go func(j int) {
			defer wg.Done()
			partFP, ecrPart, mac := cyphers[j].Encrypt(parts[j])
			// Send Message
			round, ephID, err := r.net.Send(r.sender, partFP, svc, ecrPart, mac,
				cmixParams)
			if err != nil {
				atomic.AddUint32(&failed, 1)
				jww.ERROR.Printf("Failed to send single-use response CMIX "+
					"message part %d: %+v", j, err)
			}
			jww.DEBUG.Printf("Sending single-use response CMIX message part "+
				"%d on round %d to address ID %d.", j, round, ephID.Int64())
			rounds[j] = round

			r.net.GetInstance().GetRoundEvents().AddRoundEventChan(round, sendResults,
				timeout, states.COMPLETED, states.FAILED)
		}(i)
	}

	// Wait for all go routines to finish
	wg.Wait()

	if failed > 0 {
		return nil, errors.Errorf("One or more send failed for the " +
			"response, the response will be handleable and will timeout")
	}

	jww.DEBUG.Printf("Sent %d single-use response CMIX messages to %s.",
		len(parts), r.sender)

	// Count the number of rounds
	roundMap := map[id.Round]struct{}{}
	for _, roundID := range rounds {
		if roundID != 0 {
			roundMap[roundID] = struct{}{}
		}
	}

	// Wait until the result tracking responds
	success, numRoundFail, numTimeOut := cmix.TrackResults(sendResults, len(roundMap))
	if !success {

		return nil, errors.Errorf("tracking results of %d rounds: %d round "+
			"failures, %d round event time outs; the send cannot be retried.",
			len(rounds), numRoundFail, numTimeOut)
	}

	jww.DEBUG.Printf("Tracked %d single-use response message round(s).", len(roundMap))

	return rounds, nil
}

// partitionResponse breaks a payload into its sub payloads for sending
func partitionResponse(payload []byte, cmixMessageLength int, maxParts uint8) []message.ResponsePart {
	responseMsg := message.NewResponsePart(cmixMessageLength)

	// Split payloads
	payloadParts := splitPayload(payload, responseMsg.GetMaxContentsSize(),
		int(maxParts))

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
	var parts [][]byte
	buff := bytes.NewBuffer(payload)

	for i := 0; i < maxParts && buff.Len() > 0; i++ {
		parts = append(parts, buff.Next(maxSize))
	}
	return parts
}
