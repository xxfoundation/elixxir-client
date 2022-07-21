package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"time"
)

// response adheres to the single.Response interface. This is a
// simple implementation for CLI and CI/CD integration testing purposes.
type response struct {
	callbackChan chan struct {
		payload []byte
		err     error
	}
}

func (r *response) Callback(payload []byte, receptionID receptionID.EphemeralIdentity,
	round []rounds.Round, err error) {
	jww.DEBUG.Printf("Payload: %v, receptionID: %v, round: %v, err: %v",
		payload, receptionID, round, err)
	r.callbackChan <- struct {
		payload []byte
		err     error
	}{payload: payload, err: err}
}

// sendSingleUse sends a single use message.
func sendSingleUse(net *xxdk.Cmix, partner contact.Contact, payload []byte,
	maxMessages uint8, timeout time.Duration, tag string) {
	// Construct callback
	callback := &response{
		callbackChan: make(chan struct {
			payload []byte
			err     error
		}),
	}

	jww.INFO.Printf("Sending single-use message to contact: %+v", partner)
	jww.INFO.Printf("Payload: \"%s\"", payload)
	jww.INFO.Printf("Max number of replies: %d", maxMessages)
	jww.INFO.Printf("Timeout: %s", timeout)

	// Send single-use message
	fmt.Printf("Sending single-use transmission message: %s\n", payload)
	jww.DEBUG.Printf("Sending single-use transmission to %s: %s",
		partner.ID, payload)
	params := single.GetDefaultRequestParams()
	params.MaxResponseMessages = maxMessages
	rng := net.GetRng().GetStream()
	defer rng.Close()

	e2eGrp := net.GetStorage().GetE2EGroup()
	rnd, ephID, err := single.TransmitRequest(partner, tag, payload, callback, params,
		net.GetCmix(), rng, e2eGrp)
	if err != nil {
		jww.FATAL.Panicf("Failed to transmit single-use message: %+v", err)
	}

	jww.INFO.Printf("Single Use request sent on round %v with id %v", rnd,
		ephID)

	// Wait for callback to be called
	fmt.Println("Waiting for response.")
	results := <-callback.callbackChan
	if results.payload != nil {
		fmt.Printf("Message received: %s\n", results.payload)
		jww.DEBUG.Printf("Received single-use reply payload: %s", results.payload)
	} else {
		jww.ERROR.Print("Failed to receive single-use reply payload.")
	}

	if results.err != nil {
		jww.FATAL.Panicf("Received error when waiting for reply: %+v", results.err)
	}
}
