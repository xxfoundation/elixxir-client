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
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/utils"
	"time"
)

// singleCmd is the single-use subcommand that allows for sending and responding
// to single-use messages.
var singleCmd = &cobra.Command{
	Use:   "single",
	Short: "Send and respond to single-use messages.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		client := initClient()

		// Write user contact to file
		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ReceptionID)
		jww.INFO.Printf("User Transmission: %s", user.TransmissionID)
		writeContact(user.GetContact())

		// Set up reception handler
		swBoard := client.GetSwitchboard()
		recvCh := make(chan message.Receive, 10000)
		listenerID := swBoard.RegisterChannel("DefaultCLIReceiver",
			switchboard.AnyUser(), message.XxMessage, recvCh)
		jww.INFO.Printf("Message ListenerID: %v", listenerID)

		// Set up auth request handler, which simply prints the user ID of the
		// requester
		authMgr := client.GetAuthRegistrar()
		authMgr.AddGeneralRequestCallback(printChanRequest)

		// If unsafe channels, then add auto-acceptor
		if viper.GetBool("unsafe-channel-creation") {
			authMgr.AddGeneralRequestCallback(func(
				requester contact.Contact) {
				jww.INFO.Printf("Got request: %s", requester.ID)
				_, err := client.ConfirmAuthenticatedChannel(requester)
				if err != nil {
					jww.FATAL.Panicf("%+v", err)
				}
			})
		}

		err := client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetHealth().AddChannel(connected)
		waitUntilConnected(connected)

		// Make single-use manager and start receiving process
		singleMng := single.NewManager(client)

		// get the tag
		tag := viper.GetString("tag")

		// Register the callback
		callbackChan := make(chan responseCallbackChan)
		callback := func(payload []byte, c single.Contact) {
			callbackChan <- responseCallbackChan{payload, c}
		}
		singleMng.RegisterCallback(tag, callback)
		err = client.AddService(singleMng.StartProcesses)
		if err != nil {
			jww.FATAL.Panicf("Could not add single use process: %+v", err)
		}

		for numReg, total := 1, 100; numReg < (total*3)/4; {
			time.Sleep(1 * time.Second)
			numReg, total, err = client.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			jww.INFO.Printf("Registering with nodes (%d/%d)...",
				numReg, total)
		}

		timeout := viper.GetDuration("timeout")

		// If the send flag is set, then send a message
		if viper.GetBool("send") {
			// get message details
			payload := []byte(viper.GetString("message"))
			partner := readSingleUseContact("contact")
			maxMessages := uint8(viper.GetUint("maxMessages"))

			sendSingleUse(singleMng, partner, payload, maxMessages, timeout, tag)
		}

		// If the reply flag is set, then start waiting for a message and reply
		// when it is received
		if viper.GetBool("reply") {
			replySingleUse(singleMng, timeout, callbackChan)
		}
	},
}

func init() {
	// Single-use subcommand options

	singleCmd.Flags().Bool("send", false, "Sends a single-use message.")
	_ = viper.BindPFlag("send", singleCmd.Flags().Lookup("send"))

	singleCmd.Flags().Bool("reply", false,
		"Listens for a single-use message and sends a reply.")
	_ = viper.BindPFlag("reply", singleCmd.Flags().Lookup("reply"))

	singleCmd.Flags().StringP("contact", "c", "",
		"Path to contact file to send message to.")
	_ = viper.BindPFlag("contact", singleCmd.Flags().Lookup("contact"))

	singleCmd.Flags().StringP("tag", "", "testTag",
		"The tag that specifies the callback to trigger on reception.")
	_ = viper.BindPFlag("tag", singleCmd.Flags().Lookup("tag"))

	singleCmd.Flags().Uint8("maxMessages", 1,
		"The max number of single-use response messages.")
	_ = viper.BindPFlag("maxMessages", singleCmd.Flags().Lookup("maxMessages"))

	singleCmd.Flags().DurationP("timeout", "t", 30*time.Second,
		"Duration before stopping to wait for single-use message.")
	_ = viper.BindPFlag("timeout", singleCmd.Flags().Lookup("timeout"))

	rootCmd.AddCommand(singleCmd)
}

// sendSingleUse sends a single use message.
func sendSingleUse(m *single.Manager, partner contact.Contact, payload []byte,
	maxMessages uint8, timeout time.Duration, tag string) {
	// Construct callback
	callbackChan := make(chan struct {
		payload []byte
		err     error
	})
	callback := func(payload []byte, err error) {
		callbackChan <- struct {
			payload []byte
			err     error
		}{payload: payload, err: err}
	}

	jww.INFO.Printf("Sending single-use message to contact: %+v", partner)
	jww.INFO.Printf("Payload: \"%s\"", payload)
	jww.INFO.Printf("Max number of replies: %d", maxMessages)
	jww.INFO.Printf("Timeout: %s", timeout)

	// Send single-use message
	fmt.Printf("Sending single-use transmission message: %s\n", payload)
	jww.DEBUG.Printf("Sending single-use transmission to %s: %s", partner.ID, payload)
	err := m.TransmitSingleUse(partner, payload, tag, maxMessages, callback, timeout)
	if err != nil {
		jww.FATAL.Panicf("Failed to transmit single-use message: %+v", err)
	}

	// Wait for callback to be called
	fmt.Println("Waiting for response.")
	results := <-callbackChan
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
func replySingleUse(m *single.Manager, timeout time.Duration, callbackChan chan responseCallbackChan) {

	// Wait to receive a message or stop after timeout occurs
	fmt.Println("Waiting for single-use message.")
	timer := time.NewTimer(timeout)
	select {
	case results := <-callbackChan:
		if results.payload != nil {
			fmt.Printf("Single-use transmission received: %s\n", results.payload)
			jww.DEBUG.Printf("Received single-use transmission from %s: %s",
				results.c.GetPartner(), results.payload)
		} else {
			jww.ERROR.Print("Failed to receive single-use payload.")
		}

		// Create new payload from repeated received payloads so that each
		// message part contains the same payload
		payload := makeResponsePayload(m, results.payload, results.c.GetMaxParts())

		fmt.Printf("Sending single-use response message: %s\n", payload)
		jww.DEBUG.Printf("Sending single-use response to %s: %s", results.c.GetPartner(), payload)
		err := m.RespondSingleUse(results.c, payload, timeout)
		if err != nil {
			jww.FATAL.Panicf("Failed to send response: %+v", err)
		}

	case <-timer.C:
		fmt.Println("Timed out!")
		jww.FATAL.Panicf("Failed to receive transmission after %s.", timeout)
	}
}

// responseCallbackChan structure used to collect information sent to the
// response callback.
type responseCallbackChan struct {
	payload []byte
	c       single.Contact
}

// makeResponsePayload generates a new payload that will span the max number of
// message parts in the contact. Each resulting message payload will contain a
// copy of the supplied payload with spaces taking up any remaining data.
func makeResponsePayload(m *single.Manager, payload []byte, maxParts uint8) []byte {
	payloads := make([][]byte, maxParts)
	payloadPart := makeResponsePayloadPart(m, payload)
	for i := range payloads {
		payloads[i] = make([]byte, m.GetMaxResponsePayloadSize())
		copy(payloads[i], payloadPart)
	}
	return bytes.Join(payloads, []byte{})
}

// makeResponsePayloadPart creates a single response payload by coping the given
// payload and filling the rest with spaces.
func makeResponsePayloadPart(m *single.Manager, payload []byte) []byte {
	payloadPart := make([]byte, m.GetMaxResponsePayloadSize())
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
