///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"bytes"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/utils"
)

// singleCmd is the single-use subcommand that allows for sending and responding
// to single-use messages.
var singleCmd = &cobra.Command{
	Use:   "single",
	Short: "Send and respond to single-use messages.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		cmixParams, e2eParams := initParams()
		authCbs := makeAuthCallbacks(
			viper.GetBool(unsafeChannelCreationFlag), e2eParams)
		initLog(viper.GetUint(logLevelFlag), viper.GetString(logFlag))
		client := initE2e(cmixParams, e2eParams, authCbs)

		// Write user contact to file
		user := client.GetReceptionIdentity()
		jww.INFO.Printf("User: %s", user.ID)
		writeContact(user.GetContact())

		err := client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		// get the tag
		tag := viper.GetString(singleTagFlag)

		// Register the callback
		receiver := &Receiver{
			recvCh: make(chan struct {
				request *single.Request
				ephID   receptionID.EphemeralIdentity
				round   []rounds.Round
			}),
		}

		dhKeyPriv, err := user.GetDHKeyPrivate()
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		myID := user.ID
		listener := single.Listen(tag, myID,
			dhKeyPriv,
			client.GetCmix(),
			client.GetStorage().GetE2EGroup(),
			receiver)

		for numReg, total := 1, 100; numReg < (total*3)/4; {
			time.Sleep(1 * time.Second)
			numReg, total, err = client.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			jww.INFO.Printf("Registering with nodes (%d/%d)...",
				numReg, total)
		}

		timeout := viper.GetDuration(singleTimeoutFlag)

		// If the send flag is set, then send a message
		if viper.GetBool(singleSendFlag) {
			// get message details
			payload := []byte(viper.GetString(messageFlag))
			partner := readSingleUseContact(singleContactFlag)
			maxMessages := uint8(viper.GetUint(singleMaxMessagesFlag))

			sendSingleUse(client.Cmix, partner, payload,
				maxMessages, timeout, tag)
		}

		// If the reply flag is set, then start waiting for a
		// message and reply when it is received
		if viper.GetBool(singleReplyFlag) {
			replySingleUse(timeout, receiver)
		}
		listener.Stop()
	},
}

func init() {
	// Single-use subcommand options

	singleCmd.Flags().Bool(singleSendFlag, false, "Sends a single-use message.")
	BindFlagHelper(singleSendFlag, singleCmd)

	singleCmd.Flags().Bool(singleReplyFlag, false,
		"Listens for a single-use message and sends a reply.")
	BindFlagHelper(singleReplyFlag, singleCmd)

	singleCmd.Flags().StringP(singleContactFlag, "c", "",
		"Path to contact file to send message to.")
	BindFlagHelper(singleContactFlag, singleCmd)

	singleCmd.Flags().StringP(singleTagFlag, "", "testTag",
		"The tag that specifies the callback to trigger on reception.")
	BindFlagHelper(singleTagFlag, singleCmd)

	singleCmd.Flags().Uint8(singleMaxMessagesFlag, 1,
		"The max number of single-use response messages.")
	BindFlagHelper(singleMaxMessagesFlag, singleCmd)

	singleCmd.Flags().DurationP(singleTimeoutFlag, "t", 30*time.Second,
		"Duration before stopping to wait for single-use message.")
	BindFlagHelper(singleTimeoutFlag, singleCmd)

	rootCmd.AddCommand(singleCmd)
}

type Response struct {
	callbackChan chan struct {
		payload []byte
		err     error
	}
}

func (r *Response) Callback(payload []byte, receptionID receptionID.EphemeralIdentity,
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
	callback := &Response{
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

// replySingleUse responds to any single-use message it receives by replying\
// with the same payload.
func replySingleUse(timeout time.Duration, receiver *Receiver) {
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

type Receiver struct {
	recvCh chan struct {
		request *single.Request
		ephID   receptionID.EphemeralIdentity
		round   []rounds.Round
	}
}

func (r *Receiver) Callback(req *single.Request, ephID receptionID.EphemeralIdentity,
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

// readSingleUseContact opens the contact specified in the CLI flags. Panics if
// no file provided or if an error occurs while reading or unmarshalling it.
func readSingleUseContact(key string) contact.Contact {
	// get path
	filePath := viper.GetString(key)
	if filePath == "" {
		jww.FATAL.Panicf("Failed to read contact file: no file path provided.")
	}

	// Read from file
	data, err := utils.ReadFile(filePath)
	jww.INFO.Printf("Contact file size read in: %d bytes", len(data))
	if err != nil {
		jww.FATAL.Panicf("Failed to read contact file: %+v", err)
	}

	// Unmarshal contact
	c, err := contact.Unmarshal(data)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal contact: %+v", err)
	}

	return c
}
