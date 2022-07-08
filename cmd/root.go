///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/elixxir/client/storage/user"

	"gitlab.com/elixxir/client/backup"
	"gitlab.com/elixxir/client/xxdk"

	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	backupCrypto "gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
)

var authCbs *authCallbacks

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
		profileOut := viper.GetString(profileCpuFlag)
		if profileOut != "" {
			f, err := os.Create(profileOut)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			pprof.StartCPUProfile(f)
		}

		cmixParams, e2eParams := initParams()

		client := initE2e(cmixParams, e2eParams)

		jww.INFO.Printf("Client Initialized...")

		receptionIdentity := client.GetReceptionIdentity()
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
			jww.INFO.Printf("Sending message to self")
			recipientID = receptionIdentity.ID
			recipientContact = receptionIdentity.GetContact()
		} else {
			recipientID = parseRecipient(destId)
		}
		isPrecanPartner := isPrecanID(recipientID)

		jww.INFO.Printf("Client: %s, Partner: %s", receptionIdentity.ID,
			recipientID)

		client.GetE2E().EnableUnsafeReception()
		recvCh := registerMessageListener(client)

		jww.INFO.Printf("Starting Network followers...")

		err := client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("Network followers started!")

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetCmix().AddHealthCallback(
			func(isConnected bool) {
				connected <- isConnected
			})
		waitUntilConnected(connected)

		// err = client.RegisterForNotifications("dJwuGGX3KUyKldWK5PgQH8:APA91bFjuvimRc4LqOyMDiy124aLedifA8DhldtaB_b76ggphnFYQWJc_fq0hzQ-Jk4iYp2wPpkwlpE1fsOjs7XWBexWcNZoU-zgMiM0Mso9vTN53RhbXUferCbAiEylucEOacy9pniN")
		// if err != nil {
		//	jww.FATAL.Panicf("Failed to register for notifications: %+v", err)
		// }

		// After connection, make sure we have registered with at least
		// 85% of the nodes
		numReg := 1
		total := 100
		jww.INFO.Printf("Registering with nodes...")

		for numReg < (total*3)/4 {
			time.Sleep(1 * time.Second)
			numReg, total, err = client.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			jww.INFO.Printf("Registering with nodes (%d/%d)...",
				numReg, total)
		}

		client.GetBackupContainer().TriggerBackup("Integration test.")

		jww.INFO.Printf("Client backup triggered...")

		// Send Messages
		msgBody := viper.GetString(messageFlag)
		time.Sleep(10 * time.Second)

		// Accept auth request for this recipient
		authConfirmed := false
		if viper.GetBool(acceptChannelFlag) {
			// Verify that the confirmation message makes it to the
			// original sender
			if viper.GetBool(verifySendFlag) {
				acceptChannelVerified(client, recipientID,
					e2eParams)
			} else {
				// Accept channel, agnostic of round result
				acceptChannel(client, recipientID)
			}

			// Do not wait for channel confirmations if we
			// accepted one
			authConfirmed = true
		}

		jww.INFO.Printf("Preexisting E2e partners: %+v", client.GetE2E().GetAllPartnerIDs())
		if client.GetE2E().HasAuthenticatedChannel(recipientID) {
			jww.INFO.Printf("Authenticated channel already in "+
				"place for %s", recipientID)
			authConfirmed = true
		} else {
			jww.INFO.Printf("No authenticated channel in "+
				"place for %s", recipientID)
		}

		// Send unsafe messages or not?
		unsafe := viper.GetBool(unsafeFlag)
		sendAuthReq := viper.GetBool(sendAuthRequestFlag)
		if !unsafe && !authConfirmed && !isPrecanPartner &&
			sendAuthReq {
			addAuthenticatedChannel(client, recipientID,
				recipientContact, e2eParams)
		} else if !unsafe && !authConfirmed && isPrecanPartner {
			addPrecanAuthenticatedChannel(client,
				recipientID, recipientContact)
			authConfirmed = true
		} else if !unsafe && authConfirmed && !isPrecanPartner &&
			sendAuthReq {
			jww.WARN.Printf("Resetting negotiated auth channel")
			resetAuthenticatedChannel(client, recipientID,
				recipientContact, e2eParams)
			authConfirmed = false
		}

		if !unsafe && !authConfirmed {
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
			waitSecs := viper.GetUint(authTimeoutFlag)
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
				!client.GetStorage().GetKV().IsMemStore(), client.GetE2E().GetAllPartnerIDs())
		}

		// DeleteFingerprint this recipient
		if viper.GetBool(deleteChannelFlag) {
			deleteChannel(client, recipientID)
		}

		if viper.GetBool(deleteReceiveRequestsFlag) {
			err = client.GetAuth().DeleteReceiveRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete received requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(deleteSentRequestsFlag) {
			err = client.GetAuth().DeleteSentRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete sent requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(deleteAllRequestsFlag) {
			err = client.GetAuth().DeleteAllRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete all requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool(deleteRequestFlag) {
			err = client.GetAuth().DeleteRequest(recipientID)
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
							roundIDs, _, err = client.GetE2E().SendUnsafe(
								mt, recipient, payload,
								e2eParams.Base)
						} else {
							e2eParams.Base.DebugTag = "cmd.E2E"
							roundIDs, _, _, err = client.GetE2E().SendE2E(mt,
								recipient, payload, e2eParams.Base)
						}
						if err != nil {
							jww.FATAL.Panicf("%+v", err)
						}

						// Verify message sends were successful
						if viper.GetBool(verifySendFlag) {
							if !verifySendSuccess(client, paramsE2E,
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
		waitTime := time.Duration(5 * time.Second)
		if expectedCnt == 0 {
			// Wait longer if we didn't expect to receive anything
			waitTime = time.Duration(15 * time.Second)
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
			roundsNotepad.INFO.Printf("\n%s", client.GetCmix().GetVerboseRounds())
		}
		wg.Wait()
		err = client.StopNetworkFollower()
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

// initCmix returns a newly-initialized xxdk.Cmix object and its stored xxdk.ReceptionIdentity
func initCmix() (*xxdk.Cmix, xxdk.ReceptionIdentity) {
	logLevel := viper.GetUint(logLevelFlag)
	initLog(logLevel, viper.GetString(logFlag))
	jww.INFO.Printf(Version())

	pass := parsePassword(viper.GetString(passwordFlag))
	storeDir := viper.GetString(sessionFlag)
	regCode := viper.GetString(regCodeFlag)
	precannedID := viper.GetUint(sendIdFlag)
	userIDprefix := viper.GetString("userid-prefix")
	protoUserPath := viper.GetString(protoUserPathFlag)
	backupPath := viper.GetString(backupInFlag)
	backupPass := []byte(viper.GetString(backupPassFlag))

	// FIXME: All branches of the upcoming path
	var knownReception xxdk.ReceptionIdentity

	// create a new client if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Load NDF
		ndfJSON, err := ioutil.ReadFile(viper.GetString(ndfFlag))
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}

		if precannedID != 0 {
			knownReception, err = xxdk.NewPrecannedClient(precannedID,
				string(ndfJSON), storeDir, pass)
		} else if protoUserPath != "" {
			protoUserJson, err := utils.ReadFile(protoUserPath)
			if err != nil {
				jww.FATAL.Panicf("%v", err)
			}

			protoUser := &user.Proto{}
			err = json.Unmarshal(protoUserJson, protoUser)
			if err != nil {
				jww.FATAL.Panicf("%v", err)
			}

			knownReception, err = xxdk.NewProtoClient_Unsafe(string(ndfJSON), storeDir,
				pass, protoUser)
		} else if userIDprefix != "" {
			err = xxdk.NewVanityClient(string(ndfJSON), storeDir,
				pass, regCode, userIDprefix)
		} else if backupPath != "" {

			b, backupFile := loadBackup(backupPath, string(backupPass))

			// Marshal the backup object in JSON
			backupJson, err := json.Marshal(b)
			if err != nil {
				jww.ERROR.Printf("Failed to JSON Marshal backup: %+v", err)
			}

			// Write the backup JSON to file
			err = utils.WriteFileDef(viper.GetString(backupJsonOutFlag), backupJson)
			if err != nil {
				jww.FATAL.Panicf("Failed to write backup to file: %+v", err)
			}

			// Construct client from backup data
			backupIdList, _, err := backup.NewClientFromBackup(string(ndfJSON), storeDir,
				pass, backupPass, backupFile)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			backupIdListPath := viper.GetString(backupIdListFlag)
			if backupIdListPath != "" {
				// Marshal backed up ID list to JSON
				backedUpIdListJson, err := json.Marshal(backupIdList)
				if err != nil {
					jww.ERROR.Printf("Failed to JSON Marshal backed up IDs: %+v", err)
				}

				// Write backed up ID list to file
				err = utils.WriteFileDef(backupIdListPath, backedUpIdListJson)
				if err != nil {
					jww.FATAL.Panicf("Failed to write backed up IDs to file %q: %+v",
						backupIdListPath, err)
				}
			}

		} else {
			jww.INFO.Printf("loading from existing session")
			err = xxdk.NewCmix(string(ndfJSON), storeDir,
				pass, regCode)
		}

		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	cmixParams, _ := initParams()

	client, err := xxdk.OpenCmix(storeDir, pass, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// If there is a known xxdk.ReceptionIdentity, store it now
	if knownReception.ID != nil {
		err = xxdk.StoreReceptionIdentity(identityStorageKey, knownReception, client)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	// Attempt to load extant xxdk.ReceptionIdentity
	identity, err := xxdk.LoadReceptionIdentity(identityStorageKey, client)
	if err != nil {
		// If no extant xxdk.ReceptionIdentity, generate and store a new one
		identity, err = xxdk.MakeReceptionIdentity(client)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, client)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}
	return client, identity
}

func initParams() (xxdk.CMIXParams, xxdk.E2EParams) {
	e2eParams := xxdk.GetDefaultE2EParams()
	e2eParams.Session.MinKeys = uint16(viper.GetUint(e2eMinKeysFlag))
	e2eParams.Session.MaxKeys = uint16(viper.GetUint(e2eMaxKeysFlag))
	e2eParams.Session.NumRekeys = uint16(viper.GetUint(e2eNumReKeysFlag))
	e2eParams.Session.RekeyThreshold = viper.GetFloat64(e2eRekeyThresholdFlag)

	if viper.GetBool("splitSends") {
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
	return cmixParams, e2eParams
}

// initE2e returns a fully-formed xxdk.E2e object
func initE2e(cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) *xxdk.E2e {
	_, receptionIdentity := initCmix()

	pass := parsePassword(viper.GetString(passwordFlag))
	storeDir := viper.GetString(sessionFlag)
	jww.DEBUG.Printf("sessionDur: %v", storeDir)

	// load the client
	baseClient, err := xxdk.LoadCmix(storeDir, pass, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	authCbs = makeAuthCallbacks(
		viper.GetBool(unsafeChannelCreationFlag), e2eParams)

	// Force LoginLegacy for precanned senderID
	var client *xxdk.E2e
	if isPrecanID(receptionIdentity.ID) {
		jww.INFO.Printf("Using LoginLegacy for precan sender")
		client, err = xxdk.LoginLegacy(baseClient, e2eParams, authCbs)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	} else {
		jww.INFO.Printf("Using Login for non-precan sender")
		client, err = xxdk.Login(baseClient, authCbs, receptionIdentity,
			e2eParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	if protoUser := viper.GetString(protoUserOutFlag); protoUser != "" {

		jsonBytes, err := client.ConstructProtoUserFile()
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

	if backupOut := viper.GetString(backupOutFlag); backupOut != "" {
		backupPass := viper.GetString(backupPassFlag)
		updateBackupCb := func(encryptedBackup []byte) {
			jww.INFO.Printf("Backup update received, size %d",
				len(encryptedBackup))
			fmt.Println("Backup update received.")
			err = utils.WriteFileDef(backupOut, encryptedBackup)
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
		_, err = backup.InitializeBackup(backupPass, updateBackupCb,
			client.GetBackupContainer(), client.GetE2E(), client.GetStorage(),
			nil, client.GetStorage().GetKV(), client.GetRng())
		if err != nil {
			jww.FATAL.Panicf("Failed to initialize backup with key %q: %+v",
				backupPass, err)
		}
	}

	return client
}

func acceptChannel(client *xxdk.E2e, recipientID *id.ID) id.Round {
	recipientContact, err := client.GetAuth().GetReceivedRequest(
		recipientID)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	rid, err := client.GetAuth().Confirm(
		recipientContact)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return rid
}

func deleteChannel(client *xxdk.E2e, partnerId *id.ID) {
	err := client.DeleteContact(partnerId)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

func addAuthenticatedChannel(client *xxdk.E2e, recipientID *id.ID,
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
	fmt.Printf(msg)

	recipientContact := recipient

	if recipientContact.ID != nil && recipientContact.DhPubKey != nil {
		me := client.GetReceptionIdentity().GetContact()
		jww.INFO.Printf("Requesting auth channel from: %s",
			recipientID)

		// Verify that the auth request makes it to the recipient
		// by monitoring the round result
		if viper.GetBool(verifySendFlag) {
			requestChannelVerified(client, recipientContact, me, e2eParams)
		} else {
			// Just call Request, agnostic of round result
			_, err := client.GetAuth().Request(recipientContact,
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

func resetAuthenticatedChannel(client *xxdk.E2e, recipientID *id.ID,
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
			resetChannelVerified(client, recipientContact,
				e2eParams)
		} else {
			_, err := client.GetAuth().Reset(recipientContact)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
		}
	} else {
		jww.ERROR.Printf("Could not reset auth channel for %s",
			recipientID)
	}
}

func acceptChannelVerified(client *xxdk.E2e, recipientID *id.ID,
	params xxdk.E2EParams) {
	roundTimeout := params.Base.CMIXParams.SendTimeout

	done := make(chan struct{}, 1)
	retryChan := make(chan struct{}, 1)
	for {
		rid := acceptChannel(client, recipientID)

		// Monitor rounds for results
		err := client.GetCmix().GetRoundResults(roundTimeout,
			makeVerifySendsCallback(retryChan, done), rid)
		if err != nil {
			jww.DEBUG.Printf("Could not verify "+
				"confirmation message for relationship with %s were sent "+
				"successfully, resending messages...", recipientID)
			continue
		}

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

func requestChannelVerified(client *xxdk.E2e,
	recipientContact, me contact.Contact,
	params xxdk.E2EParams) {
	roundTimeout := params.Base.CMIXParams.SendTimeout

	retryChan := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	for {
		rid, err := client.GetAuth().Request(recipientContact,
			me.Facts)
		if err != nil {
			continue
		}

		// Monitor rounds for results
		err = client.GetCmix().GetRoundResults(roundTimeout,
			makeVerifySendsCallback(retryChan, done),
			rid)
		if err != nil {
			jww.DEBUG.Printf("Could not verify auth request was sent " +
				"successfully, resending...")
			continue
		}

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

func resetChannelVerified(client *xxdk.E2e, recipientContact contact.Contact,
	params xxdk.E2EParams) {
	roundTimeout := params.Base.CMIXParams.SendTimeout

	retryChan := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	for {

		rid, err := client.GetAuth().Reset(recipientContact)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Monitor rounds for results
		err = client.GetCmix().GetRoundResults(roundTimeout,
			makeVerifySendsCallback(retryChan, done),
			rid)
		if err != nil {
			jww.DEBUG.Printf("Could not verify auth request was sent " +
				"successfully, resending...")
			continue
		}

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
			jww.FATAL.Panicf("timeout on connection after %s", waitTimeout*time.Second)
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

func parsePassword(pwStr string) []byte {
	if strings.HasPrefix(pwStr, "0x") {
		return getPWFromHexString(pwStr[2:])
	} else if strings.HasPrefix(pwStr, "b64:") {
		return getPWFromb64String(pwStr[4:])
	} else {
		return []byte(pwStr)
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
			panic(err.Error())
		}
		jww.SetLogOutput(logOutput)
	}

	if threshold > 1 {
		jww.INFO.Printf("log level set to: TRACE")
		jww.SetStdoutThreshold(jww.LevelTrace)
		jww.SetLogThreshold(jww.LevelTrace)
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
	} else if threshold == 1 {
		jww.INFO.Printf("log level set to: DEBUG")
		jww.SetStdoutThreshold(jww.LevelDebug)
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
	} else {
		jww.INFO.Printf("log level set to: INFO")
		jww.SetStdoutThreshold(jww.LevelInfo)
		jww.SetLogThreshold(jww.LevelInfo)
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

	rootCmd.PersistentFlags().StringP(destFileFlag, "",
		"", "Read this contact file for the destination id")
	viper.BindPFlag(destFileFlag, rootCmd.PersistentFlags().Lookup(destFileFlag))

	rootCmd.PersistentFlags().UintP(sendCountFlag,
		"", 1, "The number of times to send the message")
	viper.BindPFlag(sendCountFlag, rootCmd.PersistentFlags().Lookup(sendCountFlag))
	rootCmd.Flags().UintP(sendDelayFlag,
		"", 500, "The delay between sending the messages in ms")
	viper.BindPFlag(sendDelayFlag, rootCmd.Flags().Lookup(sendDelayFlag))
	rootCmd.Flags().BoolP(splitSendsFlag,
		"", false, "Force sends to go over multiple rounds if possible")
	viper.BindPFlag(splitSendsFlag, rootCmd.Flags().Lookup(splitSendsFlag))

	rootCmd.Flags().BoolP(verifySendFlag, "", false,
		"Ensure successful message sending by checking for round completion")
	viper.BindPFlag(verifySendFlag, rootCmd.Flags().Lookup(verifySendFlag))

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

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}
