///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"fmt"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"os"
	"runtime/pprof"
	"sync"
	"time"

	"gitlab.com/elixxir/client/xxdk"

	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

// Key used for storing xxdk.ReceptionIdentity objects
const identityStorageKey = "identityStorageKey"

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "Runs a client for cMix anonymous communication platform",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		profileOut := viper.GetString(cmdUtils.ProfileCpuFlag)
		if profileOut != "" {
			f, err := os.Create(profileOut)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			pprof.StartCPUProfile(f)
		}

		cmixParams, e2eParams := cmdUtils.InitParams()
		roundsNotepad := cmdUtils.InitLog(viper.GetUint(cmdUtils.LogLevelFlag),
			viper.GetString(cmdUtils.LogFlag))

		authCbs := cmdUtils.MakeAuthCallbacks(
			viper.GetBool(cmdUtils.UnsafeChannelCreationFlag), e2eParams)
		messenger := cmdUtils.InitE2e(cmixParams, e2eParams, authCbs)

		jww.INFO.Printf("Client Initialized...")

		receptionIdentity := messenger.GetReceptionIdentity()
		jww.INFO.Printf("User: %s", receptionIdentity.ID)
		writeContact(receptionIdentity.GetContact())

		var recipientContact contact.Contact
		var recipientID *id.ID

		destFile := viper.GetString(cmdUtils.DestFileFlag)
		destId := viper.GetString(cmdUtils.DestIdFlag)
		sendId := viper.GetString(cmdUtils.SendIdFlag)
		if destFile != "" {
			recipientContact = readContact(destFile)
			recipientID = recipientContact.ID
		} else if destId == "0" || sendId == destId {
			jww.INFO.Printf("Sending message to self")
			recipientID = receptionIdentity.ID
			recipientContact = receptionIdentity.GetContact()
		} else {
			recipientID = cmdUtils.ParseRecipient(destId)
		}
		isPrecanPartner := isPrecanID(recipientID)

		jww.INFO.Printf("Client: %s, Partner: %s", receptionIdentity.ID,
			recipientID)

		messenger.GetE2E().EnableUnsafeReception()
		recvCh := cmdUtils.RegisterMessageListener(messenger)

		jww.INFO.Printf("Starting Network followers...")

		err := messenger.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("Network followers started!")

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		messenger.GetCmix().AddHealthCallback(
			func(isConnected bool) {
				connected <- isConnected
			})
		cmdUtils.WaitUntilConnected(connected)

		// After connection, make sure we have registered with at least
		// 85% of the nodes
		numReg := 1
		total := 100
		jww.INFO.Printf("Registering with nodes...")

		for numReg < (total*3)/4 {
			time.Sleep(1 * time.Second)
			numReg, total, err = messenger.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			jww.INFO.Printf("Registering with nodes (%d/%d)...",
				numReg, total)
		}

		messenger.GetBackupContainer().TriggerBackup("Integration test.")

		jww.INFO.Printf("Client backup triggered...")

		// Send Messages
		msgBody := viper.GetString(cmdUtils.MessageFlag)
		time.Sleep(10 * time.Second)

		// Accept auth request for this recipient
		authConfirmed := false
		if viper.GetBool(cmdUtils.AcceptChannelFlag) {
			// Verify that the confirmation message makes it to the
			// original sender

			for {
				// Verify message sends were successful
				recipientContact, err := messenger.GetAuth().GetReceivedRequest(
					recipientID)
				if err != nil {
					jww.FATAL.Panicf("%+v", err)
				}
				rid, err := messenger.GetAuth().Confirm(
					recipientContact)
				if err != nil {
					jww.FATAL.Panicf("%+v", err)
				}

				if viper.GetBool(cmdUtils.VerifySendFlag) {
					if !cmdUtils.VerifySendSuccess(messenger, e2eParams.Base,
						[]id.Round{rid}, recipientID, nil) {
						continue
					}
				}
				break
			}

			// Do not wait for channel confirmations if we
			// accepted one
			authConfirmed = true
		}

		jww.INFO.Printf("Preexisting E2e partners: %+v", messenger.GetE2E().GetAllPartnerIDs())
		if messenger.GetE2E().HasAuthenticatedChannel(recipientID) {
			jww.INFO.Printf("Authenticated channel already in "+
				"place for %s", recipientID)
			authConfirmed = true
		} else {
			jww.INFO.Printf("No authenticated channel in "+
				"place for %s", recipientID)
		}

		// Send unsafe messages or not?
		unsafe := viper.GetBool(cmdUtils.UnsafeFlag)
		sendAuthReq := viper.GetBool(cmdUtils.SendAuthRequestFlag)
		if !unsafe && !authConfirmed && !isPrecanPartner &&
			sendAuthReq {
			addAuthenticatedChannel(messenger, recipientID,
				recipientContact, e2eParams)
		} else if !unsafe && !authConfirmed && isPrecanPartner {
			addPrecanAuthenticatedChannel(messenger,
				recipientID, recipientContact)
			authConfirmed = true
		} else if !unsafe && authConfirmed && !isPrecanPartner &&
			sendAuthReq {
			jww.WARN.Printf("Resetting negotiated auth channel")
			resetAuthenticatedChannel(messenger, recipientID,
				recipientContact, e2eParams)
			authConfirmed = false
		}

		if !unsafe && !authConfirmed {
			// Signal for authConfirm callback in a separate thread
			go func() {
				for {
					authID := authCbs.ReceiveConfirmation()
					if authID.Cmp(recipientID) {
						authConfirmed = true
					}
				}
			}()

			jww.INFO.Printf("Waiting for authentication channel"+
				" confirmation with partner %s", recipientID)
			scnt := uint(0)

			// Wait until authConfirmed
			waitSecs := viper.GetUint(cmdUtils.AuthTimeoutFlag)
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
			jww.INFO.Printf("Authenticated partners saved: %v\n    PartnersList: %+v",
				!messenger.GetStorage().GetKV().IsMemStore(), messenger.GetE2E().GetAllPartnerIDs())
		}

		// DeleteFingerprint this recipient
		if viper.GetBool(cmdUtils.DeleteChannelFlag) {
			deleteChannel(messenger, recipientID)
		}

		if viper.GetBool(cmdUtils.DeleteReceiveRequestsFlag) {
			err = messenger.GetAuth().DeleteReceiveRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete received requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(cmdUtils.DeleteSentRequestsFlag) {
			err = messenger.GetAuth().DeleteSentRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete sent requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(cmdUtils.DeleteAllRequestsFlag) {
			err = messenger.GetAuth().DeleteAllRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete all requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(cmdUtils.DeleteRequestFlag) {
			err = messenger.GetAuth().DeleteRequest(recipientID)
			if err != nil {
				jww.FATAL.Panicf("Failed to delete request for %s:"+
					" %+v", recipientID, err)
			}
		}

		mt := catalog.MessageType(catalog.XxMessage)
		payload := []byte(msgBody)
		recipient := recipientID

		jww.INFO.Printf("Client Sending messages...")

		wg := &sync.WaitGroup{}
		sendCnt := int(viper.GetUint(cmdUtils.SendCountFlag))
		wg.Add(sendCnt)
		go func() {
			sendDelay := time.Duration(viper.GetUint(cmdUtils.SendDelayFlag))
			for i := 0; i < sendCnt; i++ {
				go func(i int) {
					defer wg.Done()
					fmt.Printf("Sending to %s: %s\n", recipientID, msgBody)
					for {
						// Send messages
						var roundIDs []id.Round
						if unsafe {
							e2eParams.Base.DebugTag = "cmd.Unsafe"
							roundIDs, _, err = messenger.GetE2E().SendUnsafe(
								mt, recipient, payload,
								e2eParams.Base)
						} else {
							e2eParams.Base.DebugTag = "cmd.E2E"
							roundIDs, _, _, err = messenger.GetE2E().SendE2E(mt,
								recipient, payload, e2eParams.Base)
						}
						if err != nil {
							jww.FATAL.Panicf("%+v", err)
						}

						// Verify message sends were successful
						if viper.GetBool(cmdUtils.VerifySendFlag) {
							if !cmdUtils.VerifySendSuccess(messenger, e2eParams.Base,
								roundIDs, recipientID, payload) {
								continue
							}

						}

						break
					}
				}(i)
				time.Sleep(sendDelay * time.Millisecond)
			}
		}()

		// Wait until message timeout or we receive enough then exit
		// TODO: Actually check for how many messages we've received
		expectedCnt := viper.GetUint(cmdUtils.ReceiveCountFlag)
		receiveCnt := uint(0)
		waitSecs := viper.GetUint(cmdUtils.WaitTimeoutFlag)
		waitTimeout := time.Duration(waitSecs) * time.Second
		done := false

		jww.INFO.Printf("Client receiving messages...")

		for !done && expectedCnt != 0 {
			timeoutTimer := time.NewTimer(waitTimeout)
			select {
			case <-timeoutTimer.C:
				fmt.Println("Timed out!")
				jww.ERROR.Printf("Timed out on message reception after %s!", waitTimeout)
				done = true
				break
			case m := <-recvCh:
				strToPrint := string(m.Payload)
				if m.MessageType != catalog.XxMessage {
					strToPrint = fmt.Sprintf("type is %s",
						m.MessageType)
				} else {
					receiveCnt++
				}

				fmt.Printf("Message received: %s\n",
					strToPrint)

				// fmt.Printf("%s", m.Timestamp)
				if receiveCnt == expectedCnt {
					done = true
					break
				}
			}
		}

		// wait an extra 5 seconds to make sure no messages were missed
		done = false
		waitTime := 5 * time.Second
		if expectedCnt == 0 {
			// Wait longer if we didn't expect to receive anything
			waitTime = 15 * time.Second
		}
		timer := time.NewTimer(waitTime)
		for !done {
			select {
			case <-timer.C:
				done = true
				break
			case m := <-recvCh:
				fmt.Printf("Message received: %s\n", string(
					m.Payload))
				// fmt.Printf("%s", m.Timestamp)
				receiveCnt++
			}
		}

		jww.INFO.Printf("Received %d/%d Messages!", receiveCnt, expectedCnt)
		fmt.Printf("Received %d\n", receiveCnt)
		if roundsNotepad != nil {
			roundsNotepad.INFO.Printf("\n%s", messenger.GetCmix().GetVerboseRounds())
		}
		wg.Wait()
		err = messenger.StopNetworkFollower()
		if err != nil {
			jww.WARN.Printf(
				"Failed to cleanly close threads: %+v\n",
				err)
		}
		if profileOut != "" {
			pprof.StopCPUProfile()
		}
		jww.INFO.Printf("Client exiting!")
	},
}

func acceptChannel(messenger *xxdk.E2e, recipientID *id.ID) id.Round {
	recipientContact, err := messenger.GetAuth().GetReceivedRequest(
		recipientID)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	rid, err := messenger.GetAuth().Confirm(
		recipientContact)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return rid
}

func deleteChannel(messenger *xxdk.E2e, partnerId *id.ID) {
	err := messenger.DeleteContact(partnerId)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

func addAuthenticatedChannel(messenger *xxdk.E2e, recipientID *id.ID,
	recipient contact.Contact, e2eParams xxdk.E2EParams) {
	var allowed bool
	if viper.GetBool(cmdUtils.UnsafeChannelCreationFlag) {
		msg := "unsafe channel creation enabled\n"
		jww.WARN.Printf(msg)
		fmt.Printf("WARNING: %s", msg)
		allowed = true
	} else {
		allowed = askToCreateChannel(recipientID)
	}
	if !allowed {
		jww.FATAL.Panicf("User did not allow channel creation!")
	}

	msg := fmt.Sprintf("Adding authenticated channel for: %s\n",
		recipientID)
	jww.INFO.Printf(msg)
	fmt.Printf(msg)

	recipientContact := recipient

	if recipientContact.ID != nil && recipientContact.DhPubKey != nil {
		me := messenger.GetReceptionIdentity().GetContact()
		jww.INFO.Printf("Requesting auth channel from: %s",
			recipientID)
		for {
			// Just call Request, agnostic of round result
			rid, err := messenger.GetAuth().Request(recipientContact,
				me.Facts)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			// Verify sends if requested
			if viper.GetBool(cmdUtils.VerifySendFlag) {
				if !cmdUtils.VerifySendSuccess(messenger, e2eParams.Base,
					[]id.Round{rid}, recipientID, nil) {
					continue
				}
			}
			break
		}
	} else {
		jww.ERROR.Printf("Could not add auth channel for %s",
			recipientID)
	}
}

func resetAuthenticatedChannel(messenger *xxdk.E2e, recipientID *id.ID,
	recipient contact.Contact, e2eParams xxdk.E2EParams) {
	var allowed bool
	if viper.GetBool(cmdUtils.UnsafeChannelCreationFlag) {
		msg := "unsafe channel creation enabled\n"
		jww.WARN.Printf(msg)
		fmt.Printf("WARNING: %s", msg)
		allowed = true
	} else {
		allowed = askToCreateChannel(recipientID)
	}
	if !allowed {
		jww.FATAL.Panicf("User did not allow channel reset!")
	}

	msg := fmt.Sprintf("Resetting authenticated channel for: %s\n",
		recipientID)
	jww.INFO.Printf(msg)
	fmt.Printf(msg)

	recipientContact := recipient

	if recipientContact.ID != nil && recipientContact.DhPubKey != nil {
		jww.INFO.Printf("Requesting auth channel from: %s",
			recipientID)

		for {
			// Verify that the auth request makes it to the recipient
			// by monitoring the round result
			rid, err := messenger.GetAuth().Reset(recipientContact)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			if viper.GetBool(cmdUtils.VerifySendFlag) {
				if !cmdUtils.VerifySendSuccess(messenger, e2eParams.Base,
					[]id.Round{rid}, recipientID, nil) {
					continue
				}
			}
			break
		}

	} else {
		jww.ERROR.Printf("Could not reset auth channel for %s",
			recipientID)
	}
}

func askToCreateChannel(recipientID *id.ID) bool {
	for {
		fmt.Printf("This is the first time you have messaged %v, "+
			"are you sure? (yes/no) ", recipientID)
		var input string
		fmt.Scanln(&input)
		if input == "yes" {
			return true
		}
		if input == "no" {
			return false
		}
		fmt.Printf("Please answer 'yes' or 'no'\n")
	}
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// NOTE: The point of init() is to be declarative.  There is
	// one init in each sub command. Do not put variable
	// declarations here, and ensure all the Flags are of the *P
	// variety, unless there's a very good reason not to have them
	// as local params to sub command."
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().UintP(cmdUtils.LogLevelFlag, "v", 0,
		"Verbose mode for debugging")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.LogLevelFlag, rootCmd)

	rootCmd.PersistentFlags().Bool(cmdUtils.VerboseRoundTrackingFlag, false,
		"Verbose round tracking, keeps track and prints all rounds the "+
			"client was aware of while running. Defaults to false if not set.")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.VerboseRoundTrackingFlag, rootCmd)

	rootCmd.PersistentFlags().StringP(cmdUtils.SessionFlag, "s",
		"", "Sets the initial storage directory for "+
			"client session data")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.SessionFlag, rootCmd)

	rootCmd.PersistentFlags().StringP(cmdUtils.WriteContactFlag, "w",
		"-", "Write contact information, if any, to this file, "+
			" defaults to stdout")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.WriteContactFlag, rootCmd)

	rootCmd.PersistentFlags().StringP(cmdUtils.PasswordFlag, "p", "",
		"Password to the session file")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.PasswordFlag, rootCmd)

	rootCmd.PersistentFlags().StringP(cmdUtils.NdfFlag, "n", "ndf.json",
		"Path to the network definition JSON file")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.NdfFlag, rootCmd)

	rootCmd.PersistentFlags().StringP(cmdUtils.LogFlag, "l", "-",
		"Path to the log output path (- is stdout)")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.LogFlag, rootCmd)

	rootCmd.PersistentFlags().StringP(cmdUtils.MessageFlag, "m", "",
		"Message to send")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.MessageFlag, rootCmd)

	rootCmd.PersistentFlags().Bool(cmdUtils.ForceLegacyFlag, false,
		"Force client to operate using legacy identities.")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.ForceLegacyFlag, rootCmd)

	rootCmd.PersistentFlags().StringP(cmdUtils.DestFileFlag, "",
		"", "Read this contact file for the destination id")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.DestFileFlag, rootCmd)

	rootCmd.PersistentFlags().UintP(cmdUtils.SendCountFlag,
		"", 1, "The number of times to send the message")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.SendCountFlag, rootCmd)

	rootCmd.PersistentFlags().UintP(cmdUtils.SendDelayFlag,
		"", 500, "The delay between sending the messages in ms")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.SendDelayFlag, rootCmd)

	rootCmd.Flags().StringP(cmdUtils.RegCodeFlag, "", "",
		"ReceptionIdentity code (optional)")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.RegCodeFlag, rootCmd)

	rootCmd.PersistentFlags().UintP(cmdUtils.ReceiveCountFlag,
		"", 1, "How many messages we should wait for before quitting")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.ReceiveCountFlag, rootCmd)

	rootCmd.PersistentFlags().UintP(cmdUtils.WaitTimeoutFlag, "", 15,
		"The number of seconds to wait for messages to arrive")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.WaitTimeoutFlag, rootCmd)

	rootCmd.PersistentFlags().BoolP(cmdUtils.UnsafeChannelCreationFlag, "", false,
		"Turns off the user identity authenticated channel check, "+
			"automatically approving authenticated channels")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.UnsafeChannelCreationFlag, rootCmd)

	rootCmd.PersistentFlags().Bool(cmdUtils.DeleteChannelFlag, false,
		"DeleteFingerprint the channel information for the corresponding recipient ID")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.DeleteChannelFlag, rootCmd)

	rootCmd.PersistentFlags().Bool(cmdUtils.DeleteReceiveRequestsFlag, false,
		"DeleteFingerprint the all received contact requests.")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.DeleteReceiveRequestsFlag, rootCmd)

	rootCmd.PersistentFlags().Bool(cmdUtils.DeleteSentRequestsFlag, false,
		"DeleteFingerprint the all sent contact requests.")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.DeleteSentRequestsFlag, rootCmd)

	rootCmd.PersistentFlags().Bool(cmdUtils.DeleteAllRequestsFlag, false,
		"DeleteFingerprint the all contact requests, both sent and received.")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.DeleteAllRequestsFlag, rootCmd)

	rootCmd.PersistentFlags().Bool(cmdUtils.DeleteRequestFlag, false,
		"DeleteFingerprint the request for the specified ID given by the "+
			"destfile flag's contact file.")
	cmdUtils.BindPersistentFlagHelper(cmdUtils.DeleteRequestFlag, rootCmd)

	/////////////////////////////////////////////////////////////////////////////////////////
	// Non-Persistant Flags
	////////////////////////////////////////////////////////////////////////////////////////

	rootCmd.Flags().UintP(cmdUtils.SendIdFlag, "", 0,
		"Use precanned user id (must be between 1 and 40, inclusive)")
	cmdUtils.BindFlagHelper(cmdUtils.SendIdFlag, rootCmd)

	rootCmd.Flags().StringP(cmdUtils.DestIdFlag, "d", "0",
		"ID to send message to (if below 40, will be precanned. Use "+
			"'0x' or 'b64:' for hex and base64 representations)")
	cmdUtils.BindFlagHelper(cmdUtils.DestIdFlag, rootCmd)

	rootCmd.Flags().BoolP(cmdUtils.SplitSendsFlag,
		"", false, "Force sends to go over multiple rounds if possible")
	cmdUtils.BindFlagHelper(cmdUtils.SplitSendsFlag, rootCmd)

	rootCmd.Flags().BoolP(cmdUtils.VerifySendFlag, "", false,
		"Ensure successful message sending by checking for round completion")
	cmdUtils.BindFlagHelper(cmdUtils.VerifySendFlag, rootCmd)

	rootCmd.Flags().BoolP(cmdUtils.UnsafeFlag, "", false,
		"Send raw, unsafe messages without e2e encryption.")
	cmdUtils.BindFlagHelper(cmdUtils.UnsafeFlag, rootCmd)

	rootCmd.Flags().BoolP(cmdUtils.AcceptChannelFlag, "", false,
		"Accept the channel request for the corresponding recipient ID")
	cmdUtils.BindFlagHelper(cmdUtils.AcceptChannelFlag, rootCmd)

	rootCmd.Flags().BoolP(cmdUtils.SendAuthRequestFlag, "", false,
		"Send an auth request to the specified destination and wait"+
			"for confirmation")
	cmdUtils.BindFlagHelper(cmdUtils.SendAuthRequestFlag, rootCmd)

	rootCmd.Flags().UintP(cmdUtils.AuthTimeoutFlag, "", 60,
		"The number of seconds to wait for an authentication channel"+
			"to confirm")
	cmdUtils.BindFlagHelper(cmdUtils.AuthTimeoutFlag, rootCmd)

	rootCmd.Flags().BoolP(cmdUtils.ForceHistoricalRoundsFlag, "", false,
		"Force all rounds to be sent to historical round retrieval")
	cmdUtils.BindFlagHelper(cmdUtils.ForceHistoricalRoundsFlag, rootCmd)

	// Network params
	rootCmd.Flags().BoolP(cmdUtils.SlowPollingFlag, "", false,
		"Enables polling for unfiltered network updates with RSA signatures")
	cmdUtils.BindFlagHelper(cmdUtils.SlowPollingFlag, rootCmd)

	rootCmd.Flags().Bool(cmdUtils.ForceMessagePickupRetryFlag, false,
		"Enable a mechanism which forces a 50% chance of no message pickup, "+
			"instead triggering the message pickup retry mechanism")
	cmdUtils.BindFlagHelper(cmdUtils.ForceMessagePickupRetryFlag, rootCmd)

	rootCmd.Flags().String(cmdUtils.ProfileCpuFlag, "",
		"Enable cpu profiling to this file")
	cmdUtils.BindFlagHelper(cmdUtils.ProfileCpuFlag, rootCmd)

	// E2E Params
	defaultE2EParams := session.GetDefaultParams()
	rootCmd.Flags().UintP(cmdUtils.E2eMinKeysFlag, "", uint(defaultE2EParams.MinKeys),
		"Minimum number of keys used before requesting rekey")
	cmdUtils.BindFlagHelper(cmdUtils.E2eMinKeysFlag, rootCmd)

	rootCmd.Flags().UintP(cmdUtils.E2eMaxKeysFlag,
		"", uint(defaultE2EParams.MaxKeys),
		"Max keys used before blocking until a rekey completes")
	cmdUtils.BindFlagHelper(cmdUtils.E2eMaxKeysFlag, rootCmd)

	rootCmd.Flags().UintP(cmdUtils.E2eNumReKeysFlag,
		"", uint(defaultE2EParams.NumRekeys),
		"Number of rekeys reserved for rekey operations")
	cmdUtils.BindFlagHelper(cmdUtils.E2eNumReKeysFlag, rootCmd)

	rootCmd.Flags().Float64P(cmdUtils.E2eRekeyThresholdFlag,
		"", defaultE2EParams.RekeyThreshold,
		"Number between 0 an 1. Percent of keys used before a rekey is started")
	cmdUtils.BindFlagHelper(cmdUtils.E2eRekeyThresholdFlag, rootCmd)

	// Proto user flags
	rootCmd.Flags().String(cmdUtils.ProtoUserPathFlag, "",
		"Path to proto user JSON file containing cryptographic primitives "+
			"the client will load")
	cmdUtils.BindFlagHelper(cmdUtils.ProtoUserPathFlag, rootCmd)

	rootCmd.Flags().String(cmdUtils.ProtoUserOutFlag, "",
		"Path to which a normally constructed client "+
			"will write proto user JSON file")
	cmdUtils.BindFlagHelper(cmdUtils.ProtoUserOutFlag, rootCmd)

	// Backup flags
	rootCmd.Flags().String(cmdUtils.BackupOutFlag, "",
		"Path to output encrypted client backup. "+
			"If no path is supplied, the backup system is not started.")
	cmdUtils.BindFlagHelper(cmdUtils.BackupOutFlag, rootCmd)

	rootCmd.Flags().String(cmdUtils.BackupJsonOutFlag, "",
		"Path to output unencrypted client JSON backup.")
	cmdUtils.BindFlagHelper(cmdUtils.BackupJsonOutFlag, rootCmd)

	rootCmd.Flags().String(cmdUtils.BackupInFlag, "",
		"Path to load backup client from")
	cmdUtils.BindFlagHelper(cmdUtils.BackupInFlag, rootCmd)

	rootCmd.Flags().String(cmdUtils.BackupPassFlag, "",
		"Passphrase to encrypt/decrypt backup")
	cmdUtils.BindFlagHelper(cmdUtils.BackupPassFlag, rootCmd)

	rootCmd.Flags().String(cmdUtils.BackupIdListFlag, "",
		"JSON file containing the backed up partner IDs")
	cmdUtils.BindFlagHelper(cmdUtils.BackupIdListFlag, rootCmd)

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}
