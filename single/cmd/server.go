package cmd

import (
	"bytes"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single"
	"time"
)

// receiver adheres to the single.Receiver interface. This is a
// simple implementation for CLI and CI/CD integration testing purposes.
type receiver struct {
	recvCh chan struct {
		request *single.Request
		ephID   receptionID.EphemeralIdentity
		round   []rounds.Round
	}
}

func (r *receiver) Callback(req *single.Request, ephID receptionID.EphemeralIdentity,
	round []rounds.Round) {
	r.recvCh <- struct {
		request *single.Request
		ephID   receptionID.EphemeralIdentity
		round   []rounds.Round
	}{
		request: req,
		ephID:   ephID,
		round:   round,
	}
}

// replySingleUse responds to any single-use message it receives by replying\
// with the same payload.
func replySingleUse(timeout time.Duration, receiver *receiver) {
	// Wait to receive a message or stop after timeout occurs
	fmt.Println("Waiting for single-use message.")
	timer := time.NewTimer(timeout)
	select {
	case results := <-receiver.recvCh:
		payload := results.request.GetPayload()
		if payload != nil {
			fmt.Printf("Single-use transmission received: %s\n", payload)
			jww.DEBUG.Printf("Received single-use transmission from %s: %s",
				results.request.GetPartner(), payload)
		} else {
			jww.ERROR.Print("Failed to receive single-use payload.")
		}

		// Create new payload from repeated received payloads so that each
		// message part contains the same payload
		resPayload := makeResponsePayload(payload, results.request.GetMaxParts(),
			results.request.GetMaxResponsePartSize())

		fmt.Printf("Sending single-use response message: %s\n", payload)
		jww.DEBUG.Printf("Sending single-use response to %s: %s",
			results.request.GetPartner(), payload)
		roundId, err := results.request.Respond(resPayload, cmix.GetDefaultCMIXParams(),
			30*time.Second)
		if err != nil {
			jww.FATAL.Panicf("Failed to send response: %+v", err)
		}

		jww.INFO.Printf("response sent on roundID: %v", roundId)

	case <-timer.C:
		fmt.Println("Timed out!")
		jww.FATAL.Panicf("Failed to receive transmission after %s.", timeout)
	}
}

// makeResponsePayload generates a new payload that will span the max number of
// message parts in the contact. Each resulting message payload will contain a
// copy of the supplied payload with spaces taking up any remaining data.
func makeResponsePayload(payload []byte, maxParts uint8, maxSizePerPart int) []byte {
	payloads := make([][]byte, maxParts)
	payloadPart := makeResponsePayloadPart(payload, maxSizePerPart)
	for i := range payloads {
		payloads[i] = make([]byte, maxSizePerPart)
		copy(payloads[i], payloadPart)
	}
	return bytes.Join(payloads, []byte{})
}

// makeResponsePayloadPart creates a single response payload by coping the given
// payload and filling the rest with spaces.
func makeResponsePayloadPart(payload []byte, maxSize int) []byte {
	payloadPart := make([]byte, maxSize)
	for i := range payloadPart {
		payloadPart[i] = ' '
	}
	copy(payloadPart, payload)

	return payloadPart
}
