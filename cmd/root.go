////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	cryptoE2e "gitlab.com/elixxir/crypto/e2e"

	"github.com/pkg/profile"

	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/elixxir/client/v4/backup"
	"gitlab.com/elixxir/client/v4/xxdk"

	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	backupCrypto "gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
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
		cpuProfileOut := viper.GetString(profileCpuFlag)
		if cpuProfileOut != "" {
			defer profile.Start(profile.CPUProfile,
				profile.ProfilePath(cpuProfileOut),
				profile.NoShutdownHook).Stop()
		}
		memProfileOut := viper.GetString(profileMemFlag)
		if memProfileOut != "" {
			defer profile.Start(profile.MemProfile,
				profile.ProfilePath(memProfileOut),
				profile.NoShutdownHook).Stop()
		}

		cmixParams, e2eParams := initParams()

		autoConfirm := viper.GetBool(unsafeChannelCreationFlag)
		acceptChannels := viper.GetBool(acceptChannelFlag)
		if acceptChannels {
			autoConfirm = false
		}

		authCbs := makeAuthCallbacks(autoConfirm, e2eParams)
		user := initE2e(cmixParams, e2eParams, authCbs)

		jww.INFO.Printf("Client Initialized...")

		receptionIdentity := user.GetReceptionIdentity()
		jww.INFO.Printf("User: %s", receptionIdentity.ID)
		writeContact(receptionIdentity.GetContact())

		var recipientContact contact.Contact
		var recipientID *id.ID

		destFile := viper.GetString(destFileFlag)
		destId := viper.GetString(destIdFlag)
		sendId := viper.GetString(sendIdFlag)
		if destFile != "" {
			recipientContact = readContact(destFile)
			recipientID = recipientContact.ID
		} else if destId == "0" || sendId == destId {
			jww.INFO.Printf("Sending message to self, " +
				"this will timeout unless authrequest is sent")
			recipientID = receptionIdentity.ID
			recipientContact = receptionIdentity.GetContact()
		} else {
			recipientID = parseRecipient(destId)
			jww.INFO.Printf("destId: %v\nrecipientId: %v", destId,
				recipientID)

		}
		isPrecanPartner := isPrecanID(recipientID)

		jww.INFO.Printf("Client: %s, Partner: %s", receptionIdentity.ID,
			recipientID)

		user.GetE2E().EnableUnsafeReception()
		recvCh := registerMessageListener(user)

		jww.INFO.Printf("Starting Network followers...")

		err := user.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("Network followers started!")

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		user.GetCmix().AddHealthCallback(
			func(isConnected bool) {
				connected <- isConnected
			})
		waitUntilConnected(connected)

		// After connection, make sure we have registered with enough of the
		// network to continue
		numReg := 1
		total := 100
		jww.INFO.Printf("Registering with nodes...")
		// If ephemeral registration is enabled, lower required nodes
		// registered before starting normal operations
		threshold := 3 / 4
		if cmixParams.Network.EnableImmediateSending {
			threshold = 4 / 10
		}

		for !cmixParams.Network.DisableNodeRegistration && numReg < total*threshold {
			time.Sleep(1 * time.Second)
			numReg, total, err = user.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			jww.INFO.Printf("Registering with nodes (%d/%d)...",
				numReg, total)
		}

		user.GetBackupContainer().TriggerBackup("Integration test.")

		jww.INFO.Printf("Client backup triggered...")

		// Send Messages
		msgBody := viper.GetString(messageFlag)
		hasMsgs := true
		if msgBody == "" {
			hasMsgs = false
		}
		time.Sleep(10 * time.Second)

		// Accept auth request for this recipient
		authSecs := viper.GetUint(authTimeoutFlag)
		authConfirmed := false
		jww.INFO.Printf("Preexisting E2e partners: %+v", user.GetE2E().GetAllPartnerIDs())
		if user.GetE2E().HasAuthenticatedChannel(recipientID) {
			jww.INFO.Printf("Authenticated channel already in "+
				"place for %s", recipientID)
			authConfirmed = true
		} else {
			jww.INFO.Printf("No authenticated channel in "+
				"place for %s", recipientID)
		}

		if acceptChannels && !authConfirmed {
			for reqDone := false; !reqDone; {
				select {
				case reqID := <-authCbs.reqCh:
					if recipientID.Cmp(reqID) {
						reqDone = true
					} else {
						fmt.Printf(
							"unexpected request:"+
								" %s", reqID)
					}
				case <-time.After(time.Duration(authSecs) *
					time.Second):
					fmt.Print("timed out on auth request")
					reqDone = true
				}
			}
			// Verify that the confirmation message makes it to the
			// original sender
			if viper.GetBool(verifySendFlag) {
				acceptChannelVerified(user, recipientID,
					e2eParams)
			} else {
				// Accept channel, agnostic of round result
				acceptChannel(user, recipientID)
			}

			// Do not wait for channel confirmations if we
			// accepted one
			authConfirmed = true
		}

		// Send unsafe messages or not?
		unsafe := viper.GetBool(unsafeFlag)
		sendAuthReq := viper.GetBool(sendAuthRequestFlag)
		if !unsafe && !authConfirmed && !isPrecanPartner &&
			sendAuthReq {
			addAuthenticatedChannel(user, recipientID,
				recipientContact, e2eParams)
		} else if !unsafe && !authConfirmed && isPrecanPartner {
			addPrecanAuthenticatedChannel(user,
				recipientID, recipientContact)
			authConfirmed = true
		} else if !unsafe && authConfirmed && !isPrecanPartner &&
			sendAuthReq {
			jww.WARN.Printf("Resetting negotiated auth channel")
			resetAuthenticatedChannel(user, recipientID,
				recipientContact, e2eParams)
			authConfirmed = false
		}

		if !unsafe && !authConfirmed && hasMsgs {
			// Signal for authConfirm callback in a separate thread
			go func() {
				for {
					authID := <-authCbs.confCh
					if authID.Cmp(recipientID) {
						authConfirmed = true
					}
				}
			}()

			jww.INFO.Printf("Waiting for authentication channel"+
				" confirmation with partner %s", recipientID)
			scnt := uint(0)

			// Wait until authConfirmed
			for !authConfirmed && scnt < authSecs {
				time.Sleep(1 * time.Second)
				scnt++
			}
			if scnt == authSecs {
				jww.FATAL.Panicf("Could not confirm "+
					"authentication channel for %s, "+
					"waited %d seconds.", recipientID,
					authSecs)
			}
			jww.INFO.Printf("Authentication channel confirmation"+
				" took %d seconds", scnt)
			jww.INFO.Printf("Authenticated partners saved: %v\n    PartnersList: %+v",
				!user.GetStorage().GetKV().IsMemStore(), user.GetE2E().GetAllPartnerIDs())
		}

		// DeleteFingerprint this recipient
		if viper.GetBool(deleteChannelFlag) {
			deleteChannel(user, recipientID)
		}

		if viper.GetBool(deleteReceiveRequestsFlag) {
			err = user.GetAuth().DeleteReceiveRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete received requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(deleteSentRequestsFlag) {
			err = user.GetAuth().DeleteSentRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete sent requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(deleteAllRequestsFlag) {
			err = user.GetAuth().DeleteAllRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete all requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(deleteRequestFlag) {
			err = user.GetAuth().DeleteRequest(recipientID)
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
		sendCnt := int(viper.GetUint(sendCountFlag))
		if !hasMsgs && sendCnt != 0 {
			msg := "No message to send, please set your message" +
				"or set sendCount to 0 to suppress this warning"
			jww.WARN.Printf(msg)
			fmt.Print(msg)
			sendCnt = 0
		}
		wg.Add(sendCnt)
		go func() {
			sendDelay := time.Duration(viper.GetUint(sendDelayFlag))
			for i := 0; i < sendCnt; i++ {
				go func(i int) {
					defer wg.Done()
					fmt.Printf("Sending to %s: %s\n", recipientID, msgBody)
					for {
						// Send messages
						var roundIDs []id.Round
						if unsafe {
							e2eParams.Base.DebugTag = "cmd.Unsafe"
							roundIDs, _, err = user.GetE2E().SendUnsafe(
								mt, recipient, payload,
								e2eParams.Base)
						} else {
							e2eParams.Base.DebugTag = "cmd.E2E"
							var sendReport cryptoE2e.SendReport
							sendReport, err = user.GetE2E().SendE2E(mt,
								recipient, payload, e2eParams.Base)
							roundIDs = sendReport.RoundList
						}
						if err != nil {
							jww.FATAL.Panicf("%+v", err)
						}

						// Verify message sends were successful
						if viper.GetBool(verifySendFlag) {
							if !verifySendSuccess(user, e2eParams.Base,
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
		expectedCnt := viper.GetUint(receiveCountFlag)
		receiveCnt := uint(0)
		waitSecs := viper.GetUint(waitTimeoutFlag)
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
			roundsNotepad.INFO.Printf("\n%s", user.GetCmix().GetVerboseRounds())
		}
		wg.Wait()
		err = user.StopNetworkFollower()
		if err != nil {
			jww.WARN.Printf(
				"Failed to cleanly close threads: %+v\n",
				err)
		}
		jww.INFO.Printf("Client exiting!")
	},
}

func initParams() (xxdk.CMIXParams, xxdk.E2EParams) {
	e2eParams := xxdk.GetDefaultE2EParams()
	e2eParams.Session.MinKeys = uint16(viper.GetUint(e2eMinKeysFlag))
	e2eParams.Session.MaxKeys = uint16(viper.GetUint(e2eMaxKeysFlag))
	e2eParams.Session.NumRekeys = uint16(viper.GetUint(e2eNumReKeysFlag))
	e2eParams.Session.RekeyThreshold = viper.GetFloat64(e2eRekeyThresholdFlag)

	if viper.GetBool(splitSendsFlag) {
		e2eParams.Base.ExcludedRounds = excludedRounds.NewSet()
	}

	cmixParams := xxdk.GetDefaultCMixParams()
	cmixParams.Network.Pickup.ForceHistoricalRounds = viper.GetBool(
		forceHistoricalRoundsFlag)
	cmixParams.Network.FastPolling = !viper.GetBool(slowPollingFlag)
	cmixParams.Network.Pickup.ForceMessagePickupRetry = viper.GetBool(
		forceMessagePickupRetryFlag)
	if cmixParams.Network.Pickup.ForceMessagePickupRetry {
		period := 3 * time.Second
		jww.INFO.Printf("Setting Uncheck Round Period to %v", period)
		cmixParams.Network.Pickup.UncheckRoundPeriod = period
	}
	cmixParams.Network.VerboseRoundTracking = viper.GetBool(
		verboseRoundTrackingFlag)

	cmixParams.Network.DisableNodeRegistration = viper.GetBool(disableNodeRegistrationFlag)
	cmixParams.Network.EnableImmediateSending = viper.GetBool(enableImmediateSendingFlag)

	cmixParams.Network.WhitelistedGateways = viper.GetStringSlice(gatewayWhitelistFlag)

	cmixParams.Network.Pickup.BatchMessageRetrieval = viper.GetBool(batchMessagePickupFlag)
	cmixParams.Network.Pickup.MaxBatchSize = viper.GetInt(maxPickupBatchSizeFlag)
	cmixParams.Network.Pickup.BatchPickupTimeout = viper.GetInt(batchPickupTimeoutFlag)
	cmixParams.Network.Pickup.BatchDelay = viper.GetInt(batchPickupDelayFlag)

	return cmixParams, e2eParams
}

// initE2e returns a fully-formed xxdk.E2e object
func initE2e(cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams,
	callbacks *authCallbacks) *xxdk.E2e {
	initLog(viper.GetUint(logLevelFlag), viper.GetString(logFlag))
	jww.INFO.Printf(Version())

	// Intake parameters for user initialization
	precanId := viper.GetUint(sendIdFlag)
	protoUserPath := viper.GetString(protoUserPathFlag)
	userIdPrefix := viper.GetString(userIdPrefixFlag)
	backupPath := viper.GetString(backupInFlag)
	backupPass := viper.GetString(backupPassFlag)
	storePassword := parsePassword(viper.GetString(passwordFlag))
	storeDir := viper.GetString(sessionFlag)
	regCode := viper.GetString(regCodeFlag)
	forceLegacy := viper.GetBool(forceLegacyFlag)
	jww.DEBUG.Printf("sessionDir: %v", storeDir)

	// Initialize the user of the proper type
	var user *xxdk.E2e
	if precanId != 0 {
		user = loadOrInitPrecan(precanId, storePassword, storeDir, cmixParams, e2eParams, callbacks)
	} else if protoUserPath != "" {
		user = loadOrInitProto(protoUserPath, storePassword, storeDir, cmixParams, e2eParams, callbacks)
	} else if userIdPrefix != "" {
		user = loadOrInitVanity(storePassword, storeDir, regCode, userIdPrefix, cmixParams, e2eParams, callbacks)
	} else if backupPath != "" {
		user = loadOrInitBackup(backupPath, backupPass, storePassword, storeDir, cmixParams, e2eParams, callbacks)
	} else {
		user = loadOrInitUser(forceLegacy, storePassword, storeDir, regCode, cmixParams, e2eParams, callbacks)
	}

	// Handle protoUser output
	if protoUser := viper.GetString(protoUserOutFlag); protoUser != "" {
		jsonBytes, err := user.ConstructProtoUserFile()
		if err != nil {
			jww.FATAL.Panicf("cannot construct proto user file: %v",
				err)
		}

		err = utils.WriteFileDef(protoUser, jsonBytes)
		if err != nil {
			jww.FATAL.Panicf("cannot write proto user to file: %v",
				err)
		}
	}

	// Handle backup output
	if backupOut := viper.GetString(backupOutFlag); backupOut != "" {
		if !forceLegacy {
			jww.FATAL.Panicf("Unable to make backup for non-legacy sender!")
		}
		updateBackupCb := func(encryptedBackup []byte) {
			jww.INFO.Printf("Backup update received, size %d",
				len(encryptedBackup))
			fmt.Println("Backup update received.")
			err := utils.WriteFileDef(backupOut, encryptedBackup)
			if err != nil {
				jww.FATAL.Panicf("cannot write backup: %+v",
					err)
			}

			backupJsonPath := viper.GetString(backupJsonOutFlag)

			if backupJsonPath != "" {
				var b backupCrypto.Backup
				err = b.Decrypt(backupPass, encryptedBackup)
				if err != nil {
					jww.ERROR.Printf("cannot decrypt backup: %+v", err)
				}

				backupJson, err := json.Marshal(b)
				if err != nil {
					jww.ERROR.Printf("Failed to JSON unmarshal backup: %+v", err)
				}

				err = utils.WriteFileDef(backupJsonPath, backupJson)
				if err != nil {
					jww.FATAL.Panicf("Failed to write backup to file: %+v", err)
				}
			}
		}
		_, err := backup.InitializeBackup(backupPass, updateBackupCb,
			user.GetBackupContainer(), user.GetE2E(), user.GetStorage(),
			nil, user.GetStorage().GetKV(), user.GetRng())
		if err != nil {
			jww.FATAL.Panicf("Failed to initialize backup with key %q: %+v",
				backupPass, err)
		}
	}

	cmixParams.Network.WhitelistedGateways = viper.GetStringSlice(gatewayWhitelistFlag)

	return user
}

func acceptChannel(user *xxdk.E2e, recipientID *id.ID) id.Round {
	recipientContact, err := user.GetAuth().GetReceivedRequest(
		recipientID)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	rid, err := user.GetAuth().Confirm(
		recipientContact)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return rid
}

func deleteChannel(user *xxdk.E2e, partnerId *id.ID) {
	err := user.DeleteContact(partnerId)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

func addAuthenticatedChannel(user *xxdk.E2e, recipientID *id.ID,
	recipient contact.Contact, e2eParams xxdk.E2EParams) {
	var allowed bool
	if viper.GetBool(unsafeChannelCreationFlag) {
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
	fmt.Print(msg)

	recipientContact := recipient

	if recipientContact.ID != nil && recipientContact.DhPubKey != nil {
		me := user.GetReceptionIdentity().GetContact()
		jww.INFO.Printf("Requesting auth channel from: %s",
			recipientID)

		// Verify that the auth request makes it to the recipient
		// by monitoring the round result
		if viper.GetBool(verifySendFlag) {
			requestChannelVerified(user, recipientContact, me, e2eParams)
		} else {
			// Just call Request, agnostic of round result
			_, err := user.GetAuth().Request(recipientContact,
				me.Facts)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
		}

	} else {
		jww.ERROR.Printf("Could not add auth channel for %s",
			recipientID)
	}
}

func resetAuthenticatedChannel(user *xxdk.E2e, recipientID *id.ID,
	recipient contact.Contact, e2eParams xxdk.E2EParams) {
	var allowed bool
	if viper.GetBool(unsafeChannelCreationFlag) {
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
		// Verify that the auth request makes it to the recipient
		// by monitoring the round result
		if viper.GetBool(verifySendFlag) {
			resetChannelVerified(user, recipientContact,
				e2eParams)
		} else {
			_, err := user.GetAuth().Reset(recipientContact)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
		}
	} else {
		jww.ERROR.Printf("Could not reset auth channel for %s",
			recipientID)
	}
}

func acceptChannelVerified(user *xxdk.E2e, recipientID *id.ID,
	params xxdk.E2EParams) {
	roundTimeout := params.Base.CMIXParams.SendTimeout

	done := make(chan struct{}, 1)
	retryChan := make(chan struct{}, 1)
	for {
		rid := acceptChannel(user, recipientID)

		// Monitor rounds for results
		user.GetCmix().GetRoundResults(roundTimeout,
			makeVerifySendsCallback(retryChan, done), rid)

		select {
		case <-retryChan:
			// On a retry, go to the top of the loop
			jww.DEBUG.Printf("Confirmation message for relationship"+
				" with %s were not sent successfully, resending "+
				"messages...", recipientID)
			continue
		case <-done:
			// Close channels on verification success
			close(done)
			close(retryChan)
			break
		}
		break
	}
}

func requestChannelVerified(user *xxdk.E2e,
	recipientContact, me contact.Contact,
	params xxdk.E2EParams) {
	roundTimeout := params.Base.CMIXParams.SendTimeout

	retryChan := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	for {
		rid, err := user.GetAuth().Request(recipientContact,
			me.Facts)
		if err != nil {
			continue
		}

		// Monitor rounds for results
		user.GetCmix().GetRoundResults(roundTimeout,
			makeVerifySendsCallback(retryChan, done),
			rid)

		select {
		case <-retryChan:
			// On a retry, go to the top of the loop
			jww.DEBUG.Printf("Auth request was not sent " +
				"successfully, resending...")
			continue
		case <-done:
			// Close channels on verification success
			close(done)
			close(retryChan)
			break
		}
		break
	}
}

func resetChannelVerified(user *xxdk.E2e, recipientContact contact.Contact,
	params xxdk.E2EParams) {
	roundTimeout := params.Base.CMIXParams.SendTimeout

	retryChan := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	for {

		rid, err := user.GetAuth().Reset(recipientContact)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Monitor rounds for results
		user.GetCmix().GetRoundResults(roundTimeout,
			makeVerifySendsCallback(retryChan, done),
			rid)

		select {
		case <-retryChan:
			// On a retry, go to the top of the loop
			jww.DEBUG.Printf("Auth request was not sent " +
				"successfully, resending...")
			continue
		case <-done:
			// Close channels on verification success
			close(done)
			close(retryChan)
			break
		}
		break

	}

}

func waitUntilConnected(connected chan bool) {
	waitTimeout := time.Duration(viper.GetUint(waitTimeoutFlag))
	timeoutTimer := time.NewTimer(waitTimeout * time.Second)
	isConnected := false
	// Wait until we connect or panic if we can't by a timeout
	for !isConnected {
		select {
		case isConnected = <-connected:
			jww.INFO.Printf("Network Status: %v\n",
				isConnected)
			break
		case <-timeoutTimer.C:
			jww.FATAL.Panicf("timeout on connection after %s",
				waitTimeout*time.Second)
		}
	}

	// Now start a thread to empty this channel and update us
	// on connection changes for debugging purposes.
	go func() {
		prev := true
		for {
			select {
			case isConnected = <-connected:
				if isConnected != prev {
					prev = isConnected
					jww.INFO.Printf(
						"Network Status Changed: %v\n",
						isConnected)
				}
				break
			}
		}
	}()
}

func waitForRegistration(user *xxdk.Cmix, threshhold float32) {
	// After connection, make sure we have registered with
	// at least 85% of the nodes
	var err error
	for numReg, total := 0, 100; numReg < int(threshhold*float32(total)); {
		jww.INFO.Printf("%d < %d", numReg,
			int(threshhold*float32(total)))
		time.Sleep(1 * time.Second)
		numReg, total, err = user.GetNodeRegistrationStatus()
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("Registering with nodes (%d/%d)...",
			numReg, total)
	}

}

func parseRecipient(idStr string) *id.ID {
	if idStr == "0" {
		jww.FATAL.Panicf("No recipient specified")
	}

	if strings.HasPrefix(idStr, "0x") {
		return getUIDFromHexString(idStr[2:])
	} else if strings.HasPrefix(idStr, "b64:") {
		return getUIDFromb64String(idStr[4:])
	} else {
		return getUIDFromString(idStr)
	}
}

func getUIDFromHexString(idStr string) *id.ID {
	idBytes, err := hex.DecodeString(fmt.Sprintf("%0*d%s",
		66-len(idStr), 0, idStr))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	ID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return ID
}

func getUIDFromb64String(idStr string) *id.ID {
	idBytes, err := base64.StdEncoding.DecodeString(idStr)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	ID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return ID
}

func getPWFromHexString(pwStr string) []byte {
	pwBytes, err := hex.DecodeString(fmt.Sprintf("%0*d%s",
		66-len(pwStr), 0, pwStr))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return pwBytes
}

func getPWFromb64String(pwStr string) []byte {
	pwBytes, err := base64.StdEncoding.DecodeString(pwStr)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return pwBytes
}

func getUIDFromString(idStr string) *id.ID {
	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	if idInt > 255 {
		jww.FATAL.Panicf("cannot convert integers above 255. Use 0x " +
			"or b64: representation")
	}
	idBytes := make([]byte, 33)
	binary.BigEndian.PutUint64(idBytes, uint64(idInt))
	idBytes[32] = byte(id.User)
	ID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return ID
}

func initLog(threshold uint, logPath string) {
	if logPath != "-" && logPath != "" {
		// Disable stdout output
		jww.SetStdoutOutput(ioutil.Discard)
		// Use log file
		logOutput, err := os.OpenFile(logPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		jww.SetLogOutput(logOutput)
	}

	if threshold > 1 {
		jww.SetStdoutThreshold(jww.LevelTrace)
		jww.SetLogThreshold(jww.LevelTrace)
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
		jww.INFO.Printf("log level set to: TRACE")
	} else if threshold == 1 {
		jww.SetStdoutThreshold(jww.LevelDebug)
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
		jww.INFO.Printf("log level set to: DEBUG")
	} else {
		jww.SetStdoutThreshold(jww.LevelInfo)
		jww.SetLogThreshold(jww.LevelInfo)
		jww.INFO.Printf("log level set to: INFO")
	}

	if viper.GetBool(verboseRoundTrackingFlag) {
		initRoundLog(logPath)
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

// this the the nodepad used for round logging.
var roundsNotepad *jww.Notepad

// initRoundLog creates the log output for round tracking. In debug mode,
// the client will keep track of all rounds it evaluates if it has
// messages in, and then will dump them to this log on client exit
func initRoundLog(logPath string) {
	parts := strings.Split(logPath, ".")
	path := parts[0] + "-rounds." + parts[1]
	logOutput, err := os.OpenFile(path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
	roundsNotepad = jww.NewNotepad(jww.LevelInfo, jww.LevelInfo,
		ioutil.Discard, logOutput, "", log.Ldate|log.Ltime)
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
	rootCmd.PersistentFlags().UintP(logLevelFlag, "v", 0,
		"Verbose mode for debugging")
	viper.BindPFlag(logLevelFlag, rootCmd.PersistentFlags().
		Lookup(logLevelFlag))

	rootCmd.PersistentFlags().Bool(verboseRoundTrackingFlag, false,
		"Verbose round tracking, keeps track and prints all rounds the "+
			"client was aware of while running. Defaults to false if not set.")
	viper.BindPFlag(verboseRoundTrackingFlag, rootCmd.PersistentFlags().Lookup(
		verboseRoundTrackingFlag))

	rootCmd.PersistentFlags().StringP(sessionFlag, "s",
		"", "Sets the initial storage directory for "+
			"client session data")
	viper.BindPFlag(sessionFlag, rootCmd.PersistentFlags().Lookup(sessionFlag))

	rootCmd.PersistentFlags().StringP(writeContactFlag, "w",
		"-", "Write contact information, if any, to this file, "+
			" defaults to stdout")
	viper.BindPFlag(writeContactFlag, rootCmd.PersistentFlags().Lookup(
		writeContactFlag))

	rootCmd.PersistentFlags().StringP(passwordFlag, "p", "",
		"Password to the session file")
	viper.BindPFlag(passwordFlag, rootCmd.PersistentFlags().Lookup(
		passwordFlag))

	rootCmd.PersistentFlags().StringArrayP(gatewayWhitelistFlag, "", []string{}, "")
	viper.BindPFlag(gatewayWhitelistFlag, rootCmd.PersistentFlags().Lookup(gatewayWhitelistFlag))

	rootCmd.PersistentFlags().StringP(ndfFlag, "n", "ndf.json",
		"Path to the network definition JSON file")
	viper.BindPFlag(ndfFlag, rootCmd.PersistentFlags().Lookup(ndfFlag))

	rootCmd.PersistentFlags().StringP(logFlag, "l", "-",
		"Path to the log output path (- is stdout)")
	viper.BindPFlag(logFlag, rootCmd.PersistentFlags().Lookup(logFlag))

	rootCmd.Flags().StringP(regCodeFlag, "", "",
		"ReceptionIdentity code (optional)")
	viper.BindPFlag(regCodeFlag, rootCmd.Flags().Lookup(regCodeFlag))

	rootCmd.PersistentFlags().StringP(messageFlag, "m", "",
		"Message to send")
	viper.BindPFlag(messageFlag, rootCmd.PersistentFlags().Lookup(messageFlag))

	rootCmd.Flags().UintP(sendIdFlag, "", 0,
		"Use precanned user id (must be between 1 and 40, inclusive)")
	viper.BindPFlag(sendIdFlag, rootCmd.Flags().Lookup(sendIdFlag))

	rootCmd.Flags().StringP(destIdFlag, "d", "0",
		"ID to send message to (if below 40, will be precanned. Use "+
			"'0x' or 'b64:' for hex and base64 representations)")
	viper.BindPFlag(destIdFlag, rootCmd.Flags().Lookup(destIdFlag))
	rootCmd.PersistentFlags().Bool("force-legacy", false,
		"Force client to operate using legacy identities.")
	viper.BindPFlag("force-legacy", rootCmd.PersistentFlags().Lookup("force-legacy"))

	rootCmd.PersistentFlags().StringP(destFileFlag, "",
		"", "Read this contact file for the destination id")
	viper.BindPFlag(destFileFlag, rootCmd.PersistentFlags().Lookup(destFileFlag))

	rootCmd.PersistentFlags().UintP(sendCountFlag,
		"", 1, "The number of times to send the message")
	viper.BindPFlag(sendCountFlag, rootCmd.PersistentFlags().Lookup(sendCountFlag))
	rootCmd.PersistentFlags().UintP(sendDelayFlag,
		"", 500, "The delay between sending the messages in ms")
	viper.BindPFlag(sendDelayFlag, rootCmd.PersistentFlags().Lookup(sendDelayFlag))
	rootCmd.Flags().BoolP(splitSendsFlag,
		"", false, "Force sends to go over multiple rounds if possible")
	viper.BindPFlag(splitSendsFlag, rootCmd.Flags().Lookup(splitSendsFlag))

	rootCmd.PersistentFlags().BoolP(verifySendFlag, "", false,
		"Ensure successful message sending by checking for round completion")
	viper.BindPFlag(verifySendFlag, rootCmd.PersistentFlags().Lookup(verifySendFlag))

	rootCmd.PersistentFlags().UintP(receiveCountFlag,
		"", 1, "How many messages we should wait for before quitting")
	viper.BindPFlag(receiveCountFlag, rootCmd.PersistentFlags().Lookup(receiveCountFlag))
	rootCmd.PersistentFlags().UintP(waitTimeoutFlag, "", 15,
		"The number of seconds to wait for messages to arrive")
	viper.BindPFlag(waitTimeoutFlag,
		rootCmd.PersistentFlags().Lookup(waitTimeoutFlag))

	rootCmd.Flags().BoolP(unsafeFlag, "", false,
		"Send raw, unsafe messages without e2e encryption.")
	viper.BindPFlag(unsafeFlag, rootCmd.Flags().Lookup(unsafeFlag))

	rootCmd.PersistentFlags().BoolP(unsafeChannelCreationFlag, "", false,
		"Turns off the user identity authenticated channel check, "+
			"automatically approving authenticated channels")
	viper.BindPFlag(unsafeChannelCreationFlag,
		rootCmd.PersistentFlags().Lookup(unsafeChannelCreationFlag))

	rootCmd.Flags().BoolP(acceptChannelFlag, "", false,
		"Accept the channel request for the corresponding recipient ID")
	viper.BindPFlag(acceptChannelFlag,
		rootCmd.Flags().Lookup(acceptChannelFlag))

	rootCmd.PersistentFlags().Bool(deleteChannelFlag, false,
		"DeleteFingerprint the channel information for the corresponding recipient ID")
	viper.BindPFlag(deleteChannelFlag,
		rootCmd.PersistentFlags().Lookup(deleteChannelFlag))

	rootCmd.PersistentFlags().Bool(deleteReceiveRequestsFlag, false,
		"DeleteFingerprint the all received contact requests.")
	viper.BindPFlag(deleteReceiveRequestsFlag,
		rootCmd.PersistentFlags().Lookup(deleteReceiveRequestsFlag))

	rootCmd.PersistentFlags().Bool(deleteSentRequestsFlag, false,
		"DeleteFingerprint the all sent contact requests.")
	viper.BindPFlag(deleteSentRequestsFlag,
		rootCmd.PersistentFlags().Lookup(deleteSentRequestsFlag))

	rootCmd.PersistentFlags().Bool(deleteAllRequestsFlag, false,
		"DeleteFingerprint the all contact requests, both sent and received.")
	viper.BindPFlag(deleteAllRequestsFlag,
		rootCmd.PersistentFlags().Lookup(deleteAllRequestsFlag))

	rootCmd.PersistentFlags().Bool(deleteRequestFlag, false,
		"DeleteFingerprint the request for the specified ID given by the "+
			"destfile flag's contact file.")
	viper.BindPFlag(deleteRequestFlag,
		rootCmd.PersistentFlags().Lookup(deleteRequestFlag))

	rootCmd.Flags().BoolP(sendAuthRequestFlag, "", false,
		"Send an auth request to the specified destination and wait"+
			"for confirmation")
	viper.BindPFlag(sendAuthRequestFlag,
		rootCmd.Flags().Lookup(sendAuthRequestFlag))
	rootCmd.Flags().UintP(authTimeoutFlag, "", 60,
		"The number of seconds to wait for an authentication channel"+
			"to confirm")
	viper.BindPFlag(authTimeoutFlag,
		rootCmd.Flags().Lookup(authTimeoutFlag))

	rootCmd.Flags().BoolP(forceHistoricalRoundsFlag, "", false,
		"Force all rounds to be sent to historical round retrieval")
	viper.BindPFlag(forceHistoricalRoundsFlag,
		rootCmd.Flags().Lookup(forceHistoricalRoundsFlag))

	// Network params
	rootCmd.Flags().BoolP(slowPollingFlag, "", false,
		"Enables polling for unfiltered network updates with RSA signatures")
	viper.BindPFlag(slowPollingFlag,
		rootCmd.Flags().Lookup(slowPollingFlag))

	rootCmd.Flags().Bool(forceMessagePickupRetryFlag, false,
		"Enable a mechanism which forces a 50% chance of no message pickup, "+
			"instead triggering the message pickup retry mechanism")
	viper.BindPFlag(forceMessagePickupRetryFlag,
		rootCmd.Flags().Lookup(forceMessagePickupRetryFlag))

	rootCmd.PersistentFlags().Bool(batchMessagePickupFlag, false,
		"Enables alternate message pickup logic which processes batches")
	viper.BindPFlag(batchMessagePickupFlag,
		rootCmd.PersistentFlags().Lookup(batchMessagePickupFlag))

	rootCmd.PersistentFlags().Int(maxPickupBatchSizeFlag, 20,
		"Set the maximum number of requests in a batch pickup message")
	viper.BindPFlag(maxPickupBatchSizeFlag,
		rootCmd.PersistentFlags().Lookup(maxPickupBatchSizeFlag))

	rootCmd.PersistentFlags().Int(batchPickupDelayFlag, 50,
		"Sets the delay (in MS) before a batch pickup request is sent, even if the batch is not full")
	viper.BindPFlag(batchPickupDelayFlag,
		rootCmd.PersistentFlags().Lookup(batchPickupDelayFlag))

	rootCmd.PersistentFlags().Int(batchPickupTimeoutFlag, 250,
		"Sets the timeout duration (in MS) sent to gateways that proxy batch message pickup requests")
	viper.BindPFlag(batchPickupTimeoutFlag,
		rootCmd.PersistentFlags().Lookup(batchPickupTimeoutFlag))

	// E2E Params
	defaultE2EParams := session.GetDefaultParams()
	rootCmd.Flags().UintP(e2eMinKeysFlag,
		"", uint(defaultE2EParams.MinKeys),
		"Minimum number of keys used before requesting rekey")
	viper.BindPFlag(e2eMinKeysFlag, rootCmd.Flags().Lookup(e2eMinKeysFlag))
	rootCmd.Flags().UintP(e2eMaxKeysFlag,
		"", uint(defaultE2EParams.MaxKeys),
		"Max keys used before blocking until a rekey completes")
	viper.BindPFlag(e2eMaxKeysFlag, rootCmd.Flags().Lookup(e2eMaxKeysFlag))
	rootCmd.Flags().UintP(e2eNumReKeysFlag,
		"", uint(defaultE2EParams.NumRekeys),
		"Number of rekeys reserved for rekey operations")
	viper.BindPFlag(e2eNumReKeysFlag, rootCmd.Flags().Lookup(e2eNumReKeysFlag))
	rootCmd.Flags().Float64P(e2eRekeyThresholdFlag,
		"", defaultE2EParams.RekeyThreshold,
		"Number between 0 an 1. Percent of keys used before a rekey is started")
	viper.BindPFlag(e2eRekeyThresholdFlag, rootCmd.Flags().Lookup(e2eRekeyThresholdFlag))

	rootCmd.Flags().String(profileCpuFlag, "",
		"Enable cpu profiling to this file")
	viper.BindPFlag(profileCpuFlag, rootCmd.Flags().Lookup(profileCpuFlag))

	rootCmd.Flags().String(profileMemFlag, "",
		"Enable memory profiling to this file")
	viper.BindPFlag(profileMemFlag, rootCmd.Flags().Lookup(profileMemFlag))

	// Proto user flags
	rootCmd.Flags().String(protoUserPathFlag, "",
		"Path to proto user JSON file containing cryptographic primitives "+
			"the client will load")
	viper.BindPFlag(protoUserPathFlag, rootCmd.Flags().Lookup(protoUserPathFlag))
	rootCmd.Flags().String(protoUserOutFlag, "",
		"Path to which a normally constructed client "+
			"will write proto user JSON file")
	viper.BindPFlag(protoUserOutFlag, rootCmd.Flags().Lookup(protoUserOutFlag))

	// Backup flags
	rootCmd.Flags().String(backupOutFlag, "",
		"Path to output encrypted client backup. If no path is supplied, the "+
			"backup system is not started.")
	viper.BindPFlag(backupOutFlag, rootCmd.Flags().Lookup(backupOutFlag))

	rootCmd.Flags().String(backupJsonOutFlag, "",
		"Path to output unencrypted client JSON backup.")
	viper.BindPFlag(backupJsonOutFlag, rootCmd.Flags().Lookup(backupJsonOutFlag))

	rootCmd.Flags().String(backupInFlag, "",
		"Path to load backup client from")
	viper.BindPFlag(backupInFlag, rootCmd.Flags().Lookup(backupInFlag))

	rootCmd.Flags().String(backupPassFlag, "",
		"Passphrase to encrypt/decrypt backup")
	viper.BindPFlag(backupPassFlag, rootCmd.Flags().Lookup(backupPassFlag))

	rootCmd.Flags().String(backupIdListFlag, "",
		"JSON file containing the backed up partner IDs")
	viper.BindPFlag(backupIdListFlag, rootCmd.Flags().Lookup(backupIdListFlag))

	rootCmd.Flags().BoolP(disableNodeRegistrationFlag, "", false,
		"Use to disable registering with nodes.  This should be used FOR TESTING PURPOSES ONLY.")
	viper.BindPFlag(disableNodeRegistrationFlag, rootCmd.Flags().Lookup(disableNodeRegistrationFlag))

	rootCmd.Flags().BoolP(enableImmediateSendingFlag, "", false,
		"Toggle to use ephemeral ED keys when attempting to send on a round with nodes you have not registered with")
	viper.BindPFlag(enableImmediateSendingFlag, rootCmd.Flags().Lookup(enableImmediateSendingFlag))

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}
