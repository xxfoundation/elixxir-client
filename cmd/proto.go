///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"io/ioutil"
	"sync"
	"time"
)

var protoCmd = &cobra.Command{
	Use:   "proto",
	Short: "Load client with a proto client JSON file.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// If output path is specified, only write to file
		var client *api.Client
		protoOutputPath := viper.GetString("protoUserOut")
		if protoOutputPath != "" {
			client = initClient()

			jsonBytes, err := client.ConstructProtoUerFile()
			if err != nil {
				jww.FATAL.Panicf("Failed to construct proto user file: %v", err)
			}

			err = utils.WriteFileDef(protoOutputPath, jsonBytes)
			if err != nil {
				jww.FATAL.Panicf("Failed to write proto user to file: %v", err)
			}
		} else {

			client = loadProtoClient()
		}

		// Write user to contact file
		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ReceptionID)
		jww.INFO.Printf("User Transmission: %s", user.TransmissionID)
		writeContact(user.GetContact())

		// Get Recipient and/or set it to myself
		isPrecanPartner := false
		recipientContact := readContact()
		recipientID := recipientContact.ID

		// Try to get recipientID from destid
		if recipientID == nil {
			recipientID, isPrecanPartner = parseRecipient(
				viper.GetString("destid"))
		}

		// Set it to myself
		if recipientID == nil {
			jww.INFO.Printf("sending message to self")
			recipientID = user.ReceptionID
			recipientContact = user.GetContact()
		}

		confCh, recvCh := initClientCallbacks(client)
		// The following block is used to check if the request from
		// a channel authorization is from the recipient we intend in
		// this run.
		authConfirmed := false
		go func() {
			for {
				requestor := <-confCh
				authConfirmed = recipientID.Cmp(requestor)
			}
		}()

		err := client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetHealth().AddChannel(connected)
		waitUntilConnected(connected)

		// After connection, make sure we have registered with at least
		// 85% of the nodes
		numReg := 1
		total := 100
		for numReg < (total*3)/4 {
			time.Sleep(1 * time.Second)
			numReg, total, err = client.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			jww.INFO.Printf("Registering with nodes (%d/%d)...",
				numReg, total)
		}

		// Send Messages
		msgBody := viper.GetString("message")

		time.Sleep(10 * time.Second)

		// Accept auth request for this recipient
		if viper.GetBool("accept-channel") {
			acceptChannel(client, recipientID)
			// Do not wait for channel confirmations if we
			// accepted one
			authConfirmed = true
		}

		if client.HasAuthenticatedChannel(recipientID) {
			jww.INFO.Printf("Authenticated channel already in "+
				"place for %s", recipientID)
			authConfirmed = true
		}

		// Send unsafe messages or not?
		unsafe := viper.GetBool("unsafe")

		sendAuthReq := viper.GetBool("send-auth-request")
		if !unsafe && !authConfirmed && !isPrecanPartner &&
			sendAuthReq {
			addAuthenticatedChannel(client, recipientID,
				recipientContact)
		} else if !unsafe && !authConfirmed && isPrecanPartner {
			addPrecanAuthenticatedChannel(client,
				recipientID, recipientContact)
			authConfirmed = true
		}

		if !unsafe && !authConfirmed {
			jww.INFO.Printf("Waiting for authentication channel"+
				" confirmation with partner %s", recipientID)
			scnt := uint(0)
			waitSecs := viper.GetUint("auth-timeout")
			for !authConfirmed && scnt < waitSecs {
				time.Sleep(1 * time.Second)
				scnt++
			}
			if scnt == waitSecs {
				jww.FATAL.Panicf("Could not confirm "+
					"authentication channel for %s, "+
					"waited %d seconds.", recipientID,
					waitSecs)
			}
			jww.INFO.Printf("Authentication channel confirmation"+
				" took %d seconds", scnt)
		}

		// Delete this recipient
		if viper.GetBool("delete-channel") {
			deleteChannel(client, recipientID)
		}

		msg := message.Send{
			Recipient:   recipientID,
			Payload:     []byte(msgBody),
			MessageType: message.Text,
		}
		paramsE2E := params.GetDefaultE2E()
		paramsUnsafe := params.GetDefaultUnsafe()
		wg := &sync.WaitGroup{}
		sendCnt := int(viper.GetUint("sendCount"))
		wg.Add(sendCnt)
		go func() {
			//sendDelay := time.Duration(viper.GetUint("sendDelay"))
			for i := 0; i < sendCnt; i++ {
				go func(i int) {
					defer wg.Done()
					fmt.Printf("Sending to %s: %s\n", recipientID, msgBody)
					var roundIDs []id.Round
					var roundTimeout time.Duration
					if unsafe {
						roundIDs, err = client.SendUnsafe(msg,
							paramsUnsafe)
						roundTimeout = paramsUnsafe.Timeout
					} else {
						roundIDs, _, _, err = client.SendE2E(msg,
							paramsE2E)
						roundTimeout = paramsE2E.Timeout
					}
					if err != nil {
						jww.FATAL.Panicf("%+v", err)
					}

					// Construct the callback function which prints out the rounds' results
					f := func(allRoundsSucceeded, timedOut bool,
						rounds map[id.Round]api.RoundResult) {
						printRoundResults(allRoundsSucceeded, timedOut, rounds, roundIDs, msg)
					}

					// Have the client report back the round results
					err = errors.New("derp")
					for j := 0; j < 5 && err != nil; j++ {
						err = client.GetRoundResults(roundIDs, roundTimeout, f)
					}

					if err != nil {
						jww.FATAL.Panicf("Message sending for send %d failed: %+v", i, err)
					}
				}(i)
			}
		}()

		// Wait until message timeout or we receive enough then exit
		// TODO: Actually check for how many messages we've received
		expectedCnt := viper.GetUint("receiveCount")
		receiveCnt := uint(0)
		waitSecs := viper.GetUint("waitTimeout")
		waitTimeout := time.Duration(waitSecs) * time.Second
		done := false

		for !done && expectedCnt != 0 {
			timeoutTimer := time.NewTimer(waitTimeout)
			select {
			case <-timeoutTimer.C:
				fmt.Println("Timed out!")
				jww.ERROR.Printf("Timed out on message reception after %s!", waitTimeout)
				done = true
				break
			case m := <-recvCh:
				fmt.Printf("Message received: %s\n", string(
					m.Payload))
				//fmt.Printf("%s", m.Timestamp)
				receiveCnt++
				if receiveCnt == expectedCnt {
					done = true
					break
				}
			}
		}

		//wait an extra 5 seconds to make sure no messages were missed
		done = false
		timer := time.NewTimer(5 * time.Second)
		for !done {
			select {
			case <-timer.C:
				done = true
				break
			case m := <-recvCh:
				fmt.Printf("Message received: %s\n", string(
					m.Payload))
				//fmt.Printf("%s", m.Timestamp)
				receiveCnt++
			}
		}

		jww.INFO.Printf("Received %d/%d Messages!", receiveCnt, expectedCnt)
		fmt.Printf("Received %d\n", receiveCnt)
		if roundsNotepad != nil {
			roundsNotepad.INFO.Printf("\n%s", client.GetNetworkInterface().GetVerboseRounds())
		}
		wg.Wait()
		err = client.StopNetworkFollower()
		if err != nil {
			jww.WARN.Printf(
				"Failed to cleanly close threads: %+v\n",
				err)
		}

	},
}

func loadProtoClient() *api.Client {
	protoUserPath := viper.GetString("protoUserPath")

	protoUserFile, err := utils.ReadFile(protoUserPath)
	if err != nil {
		jww.FATAL.Panicf("Failed to read proto user: %v", err)
	}

	pass := viper.GetString("password")
	storeDir := viper.GetString("session")

	netParams := params.GetDefaultNetwork()
	netParams.E2EParams.MinKeys = uint16(viper.GetUint("e2eMinKeys"))
	netParams.E2EParams.MaxKeys = uint16(viper.GetUint("e2eMaxKeys"))
	netParams.E2EParams.NumRekeys = uint16(
		viper.GetUint("e2eNumReKeys"))
	netParams.ForceHistoricalRounds = viper.GetBool("forceHistoricalRounds")
	netParams.FastPolling = viper.GetBool(" slowPolling")
	netParams.ForceMessagePickupRetry = viper.GetBool("forceMessagePickupRetry")
	if netParams.ForceMessagePickupRetry {
		period := 3 * time.Second
		jww.INFO.Printf("Setting Uncheck Round Period to %v", period)
		netParams.UncheckRoundPeriod = period
	}
	netParams.VerboseRoundTracking = viper.GetBool("verboseRoundTracking")

	// Load NDF
	ndfPath := viper.GetString("ndf")
	ndfJSON, err := ioutil.ReadFile(ndfPath)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	client, err := api.LoginWithProtoClient(storeDir, []byte(pass),
		protoUserFile, string(ndfJSON), netParams)
	if err != nil {
		jww.FATAL.Panicf("Failed to login: %v", err)
	}

	return client
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Proto user flags
	protoCmd.Flags().String("protoUserPath", "protoUser.json",
		"Path to proto user JSON file containing cryptographic primitives "+
			"the client will load")
	viper.BindPFlag("protoUserPath", protoCmd.Flags().Lookup("protoUserPath"))
	protoCmd.Flags().String("protoUserOut", "protoUser.json",
		"Path to which a normally constructed client "+
			"will write proto user JSON file")
	viper.BindPFlag("protoUserOut", protoCmd.Flags().Lookup("protoUserOut"))

	protoCmd.Flags().UintP("logLevel", "v", 0,
		"Verbose mode for debugging")
	viper.BindPFlag("logLevel", protoCmd.Flags().Lookup("logLevel"))

	protoCmd.Flags().Bool("verboseRoundTracking", false,
		"Verbose round tracking, keeps track and prints all rounds the "+
			"client was aware of while running. Defaults to false if not set.")
	viper.BindPFlag("verboseRoundTracking", protoCmd.Flags().Lookup("verboseRoundTracking"))

	protoCmd.Flags().StringP("session", "s",
		"", "Sets the initial storage directory for "+
			"client session data")
	viper.BindPFlag("session", protoCmd.Flags().Lookup("session"))

	protoCmd.Flags().StringP("writeContact", "w",
		"-", "Write contact information, if any, to this file, "+
			" defaults to stdout")
	viper.BindPFlag("writeContact", protoCmd.Flags().Lookup(
		"writeContact"))

	protoCmd.Flags().StringP("password", "p", "",
		"Password to the session file")
	viper.BindPFlag("password", protoCmd.Flags().Lookup(
		"password"))

	protoCmd.Flags().StringP("ndf", "n", "ndf.json",
		"Path to the network definition JSON file")
	viper.BindPFlag("ndf", protoCmd.Flags().Lookup("ndf"))

	protoCmd.Flags().StringP("log", "l", "-",
		"Path to the log output path (- is stdout)")
	viper.BindPFlag("log", protoCmd.Flags().Lookup("log"))

	protoCmd.Flags().StringP("regcode", "", "",
		"Identity code (optional)")
	viper.BindPFlag("regcode", protoCmd.Flags().Lookup("regcode"))

	protoCmd.Flags().StringP("message", "m", "",
		"Message to send")
	viper.BindPFlag("message", protoCmd.Flags().Lookup("message"))

	protoCmd.Flags().UintP("sendid", "", 0,
		"Use precanned user id (must be between 1 and 40, inclusive)")
	viper.BindPFlag("sendid", protoCmd.Flags().Lookup("sendid"))

	protoCmd.Flags().StringP("destid", "d", "0",
		"ID to send message to (if below 40, will be precanned. Use "+
			"'0x' or 'b64:' for hex and base64 representations)")
	viper.BindPFlag("destid", protoCmd.Flags().Lookup("destid"))

	protoCmd.Flags().StringP("destfile", "",
		"", "Read this contact file for the destination id")
	viper.BindPFlag("destfile", protoCmd.Flags().Lookup("destfile"))

	protoCmd.Flags().UintP("sendCount",
		"", 1, "The number of times to send the message")
	viper.BindPFlag("sendCount", protoCmd.Flags().Lookup("sendCount"))
	protoCmd.Flags().UintP("sendDelay",
		"", 500, "The delay between sending the messages in ms")
	viper.BindPFlag("sendDelay", protoCmd.Flags().Lookup("sendDelay"))

	protoCmd.Flags().UintP("receiveCount",
		"", 1, "How many messages we should wait for before quitting")
	viper.BindPFlag("receiveCount", protoCmd.Flags().Lookup("receiveCount"))
	protoCmd.Flags().UintP("waitTimeout", "", 15,
		"The number of seconds to wait for messages to arrive")
	viper.BindPFlag("waitTimeout",
		protoCmd.Flags().Lookup("waitTimeout"))

	protoCmd.Flags().BoolP("unsafe", "", false,
		"Send raw, unsafe messages without e2e encryption.")
	viper.BindPFlag("unsafe", protoCmd.Flags().Lookup("unsafe"))

	protoCmd.Flags().BoolP("unsafe-channel-creation", "", false,
		"Turns off the user identity authenticated channel check, "+
			"automatically approving authenticated channels")
	viper.BindPFlag("unsafe-channel-creation",
		protoCmd.Flags().Lookup("unsafe-channel-creation"))

	protoCmd.Flags().BoolP("accept-channel", "", false,
		"Accept the channel request for the corresponding recipient ID")
	viper.BindPFlag("accept-channel",
		protoCmd.Flags().Lookup("accept-channel"))

	protoCmd.Flags().Bool("delete-channel", false,
		"Delete the channel information for the corresponding recipient ID")
	viper.BindPFlag("delete-channel",
		protoCmd.Flags().Lookup("delete-channel"))

	protoCmd.Flags().BoolP("send-auth-request", "", false,
		"Send an auth request to the specified destination and wait"+
			"for confirmation")
	viper.BindPFlag("send-auth-request",
		protoCmd.Flags().Lookup("send-auth-request"))
	protoCmd.Flags().UintP("auth-timeout", "", 120,
		"The number of seconds to wait for an authentication channel"+
			"to confirm")
	viper.BindPFlag("auth-timeout",
		protoCmd.Flags().Lookup("auth-timeout"))

	protoCmd.Flags().BoolP("forceHistoricalRounds", "", false,
		"Force all rounds to be sent to historical round retrieval")
	viper.BindPFlag("forceHistoricalRounds",
		protoCmd.Flags().Lookup("forceHistoricalRounds"))

	// Network params
	protoCmd.Flags().BoolP("slowPolling", "", false,
		"Enables polling for unfiltered network updates with RSA signatures")
	viper.BindPFlag("slowPolling",
		protoCmd.Flags().Lookup("slowPolling"))
	protoCmd.Flags().Bool("forceMessagePickupRetry", false,
		"Enable a mechanism which forces a 50% chance of no message pickup, "+
			"instead triggering the message pickup retry mechanism")
	viper.BindPFlag("forceMessagePickupRetry",
		protoCmd.Flags().Lookup("forceMessagePickupRetry"))

	// E2E Params
	defaultE2EParams := params.GetDefaultE2ESessionParams()
	protoCmd.Flags().UintP("e2eMinKeys",
		"", uint(defaultE2EParams.MinKeys),
		"Minimum number of keys used before requesting rekey")
	viper.BindPFlag("e2eMinKeys", protoCmd.Flags().Lookup("e2eMinKeys"))
	protoCmd.Flags().UintP("e2eMaxKeys",
		"", uint(defaultE2EParams.MaxKeys),
		"Max keys used before blocking until a rekey completes")
	viper.BindPFlag("e2eMaxKeys", protoCmd.Flags().Lookup("e2eMaxKeys"))
	protoCmd.Flags().UintP("e2eNumReKeys",
		"", uint(defaultE2EParams.NumRekeys),
		"Number of rekeys reserved for rekey operations")
	viper.BindPFlag("e2eNumReKeys", protoCmd.Flags().Lookup("e2eNumReKeys"))

	protoCmd.Flags().String("profile-cpu", "",
		"Enable cpu profiling to this file")
	viper.BindPFlag("profile-cpu", protoCmd.Flags().Lookup("profile-cpu"))

	rootCmd.AddCommand(protoCmd)
}
