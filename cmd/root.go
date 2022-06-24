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
	"gitlab.com/elixxir/client/e2e"
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
		profileOut := viper.GetString("profile-cpu")
		if profileOut != "" {
			f, err := os.Create(profileOut)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			pprof.StartCPUProfile(f)
		}

		client := initClient()

		jww.INFO.Printf("Client Initialized...")

		user := client.GetUser()
		jww.INFO.Printf("USERPUBKEY: %s",
			user.E2eDhPublicKey.TextVerbose(16, 0))
		jww.INFO.Printf("User: %s", user.ReceptionID)
		writeContact(user.GetContact())

		// get Recipient and/or set it to myself
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

		jww.INFO.Printf("Client: %s, Partner: %s", user.ReceptionID,
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
			func(isconnected bool) {
				connected <- isconnected
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
		paramsE2E := e2e.GetDefaultParams()
		if viper.GetBool("accept-channel") {
			// Verify that the confirmation message makes it to the
			// original sender
			if viper.GetBool(verifySendFlag) {
				acceptChannelVerified(client, recipientID)
			} else {
				// Accept channel, agnostic of round result
				acceptChannel(client, recipientID)
			}

			// Do not wait for channel confirmations if we
			// accepted one
			authConfirmed = true
		}

		if client.GetE2E().HasAuthenticatedChannel(recipientID) {
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
		} else if !unsafe && authConfirmed && !isPrecanPartner &&
			sendAuthReq {
			jww.WARN.Printf("Resetting negotiated auth channel")
			resetAuthenticatedChannel(client, recipientID,
				recipientContact)
			authConfirmed = false
		}

		go func() {
			for {
				authID := <-authCbs.confCh
				if authID.Cmp(recipientID) {
					authConfirmed = true
				}
			}
		}()

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

		// DeleteFingerprint this recipient
		if viper.GetBool("delete-channel") {
			deleteChannel(client, recipientID)
		}

		if viper.GetBool("delete-receive-requests") {
			err = client.GetAuth().DeleteReceiveRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete received requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool("delete-sent-requests") {
			err = client.GetAuth().DeleteSentRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete sent requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool("delete-all-requests") {
			err = client.GetAuth().DeleteAllRequests()
			if err != nil {
				jww.FATAL.Panicf("Failed to delete all requests:"+
					" %+v", err)
			}
		}

		if viper.GetBool("delete-request") {
			err = client.GetAuth().DeleteRequest(recipientID)
			if err != nil {
				jww.FATAL.Panicf("Failed to delete request for %s:"+
					" %+v", recipientID, err)
			}
		}

		mt := catalog.MessageType(catalog.XxMessage)
		payload := []byte(msgBody)
		recipient := recipientID

		if viper.GetBool("splitSends") {
			paramsE2E.ExcludedRounds = excludedRounds.NewSet()
		}

		jww.INFO.Printf("Client Sending messages...")

		wg := &sync.WaitGroup{}
		sendCnt := int(viper.GetUint("sendCount"))
		wg.Add(sendCnt)
		go func() {
			sendDelay := time.Duration(viper.GetUint("sendDelay"))
			for i := 0; i < sendCnt; i++ {
				go func(i int) {
					defer wg.Done()
					fmt.Printf("Sending to %s: %s\n", recipientID, msgBody)
					for {
						// Send messages
						var roundIDs []id.Round
						if unsafe {
							paramsE2E.CMIXParams.DebugTag = "cmd.Unsafe"
							roundIDs, _, err = client.GetE2E().SendUnsafe(
								mt, recipient, payload,
								paramsE2E)
						} else {
							paramsE2E.CMIXParams.DebugTag = "cmd.E2E"
							roundIDs, _, _, err = client.GetE2E().SendE2E(mt,
								recipient, payload, paramsE2E)
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
		expectedCnt := viper.GetUint("receiveCount")
		receiveCnt := uint(0)
		waitSecs := viper.GetUint("waitTimeout")
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
				fmt.Printf("Message received: %s\n", string(
					m.Payload))
				// fmt.Printf("%s", m.Timestamp)
				receiveCnt++
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

func createClient() *xxdk.Cmix {
	logLevel := viper.GetUint(logLevelFlag)
	initLog(logLevel, viper.GetString(logFlag))
	jww.INFO.Printf(Version())

	pass := parsePassword(viper.GetString("password"))
	storeDir := viper.GetString(sessionFlag)
	regCode := viper.GetString("regcode")
	precannedID := viper.GetUint("sendid")
	userIDprefix := viper.GetString("userid-prefix")
	protoUserPath := viper.GetString("protoUserPath")
	backupPath := viper.GetString("backupIn")
	backupPass := []byte(viper.GetString("backupPass"))

	// create a new client if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Load NDF
		ndfJSON, err := ioutil.ReadFile(viper.GetString("ndf"))
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}

		if precannedID != 0 {
			err = xxdk.NewPrecannedClient(precannedID,
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

			err = xxdk.NewProtoClient_Unsafe(string(ndfJSON), storeDir,
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
			err = utils.WriteFileDef(viper.GetString("backupJsonOut"), backupJson)
			if err != nil {
				jww.FATAL.Panicf("Failed to write backup to file: %+v", err)
			}

			// Construct client from backup data
			backupIdList, _, err := backup.NewClientFromBackup(string(ndfJSON), storeDir,
				pass, backupPass, backupFile)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			backupIdListPath := viper.GetString("backupIdList")
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
			err = xxdk.NewCmix(string(ndfJSON), storeDir,
				pass, regCode)
		}

		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	params := initParams()

	client, err := xxdk.OpenCmix(storeDir, pass, params)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return client
}

func initParams() xxdk.Params {
	p := xxdk.GetDefaultParams()
	p.Session.MinKeys = uint16(viper.GetUint("e2eMinKeys"))
	p.Session.MaxKeys = uint16(viper.GetUint("e2eMaxKeys"))
	p.Session.NumRekeys = uint16(viper.GetUint("e2eNumReKeys"))
	p.Session.RekeyThreshold = viper.GetFloat64("e2eRekeyThreshold")
	p.CMix.Pickup.ForceHistoricalRounds = viper.GetBool(
		"forceHistoricalRounds")
	p.CMix.FastPolling = !viper.GetBool("slowPolling")
	p.CMix.Pickup.ForceMessagePickupRetry = viper.GetBool(
		"forceMessagePickupRetry")
	if p.CMix.Pickup.ForceMessagePickupRetry {
		period := 3 * time.Second
		jww.INFO.Printf("Setting Uncheck Round Period to %v", period)
		p.CMix.Pickup.UncheckRoundPeriod = period
	}
	p.CMix.VerboseRoundTracking = viper.GetBool(
		"verboseRoundTracking")
	return p
}

func initClient() *xxdk.E2e {
	createClient()

	pass := parsePassword(viper.GetString("password"))
	storeDir := viper.GetString(sessionFlag)
	jww.DEBUG.Printf("sessionDur: %v", storeDir)

	params := initParams()

	// load the client
	baseclient, err := xxdk.LoadCmix(storeDir, pass, params)

	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	authCbs = makeAuthCallbacks(nil,
		viper.GetBool("unsafe-channel-creation"))

	client, err := xxdk.LoginLegacy(baseclient, authCbs)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	authCbs.client = client

	if protoUser := viper.GetString("protoUserOut"); protoUser != "" {

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

	if backupOut := viper.GetString("backupOut"); backupOut != "" {
		backupPass := viper.GetString("backupPass")
		updateBackupCb := func(encryptedBackup []byte) {
			jww.INFO.Printf("Backup update received, size %d",
				len(encryptedBackup))
			fmt.Println("Backup update received.")
			err = utils.WriteFileDef(backupOut, encryptedBackup)
			if err != nil {
				jww.FATAL.Panicf("cannot write backup: %+v",
					err)
			}

			backupJsonPath := viper.GetString("backupJsonOut")

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
	recipient contact.Contact) {
	var allowed bool
	if viper.GetBool("unsafe-channel-creation") {
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
		me := client.GetUser().GetContact()
		jww.INFO.Printf("Requesting auth channel from: %s",
			recipientID)

		// Verify that the auth request makes it to the recipient
		// by monitoring the round result
		if viper.GetBool(verifySendFlag) {
			requestChannelVerified(client, recipientContact, me)
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
	recipient contact.Contact) {
	var allowed bool
	if viper.GetBool("unsafe-channel-creation") {
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
			resetChannelVerified(client, recipientContact)
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

func acceptChannelVerified(client *xxdk.E2e, recipientID *id.ID) {
	paramsE2E := e2e.GetDefaultParams()
	roundTimeout := paramsE2E.CMIXParams.SendTimeout

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
	recipientContact, me contact.Contact) {
	paramsE2E := e2e.GetDefaultParams()
	roundTimeout := paramsE2E.CMIXParams.SendTimeout

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

func resetChannelVerified(client *xxdk.E2e, recipientContact contact.Contact) {
	paramsE2E := e2e.GetDefaultParams()
	roundTimeout := paramsE2E.CMIXParams.SendTimeout

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
	waitTimeout := time.Duration(viper.GetUint("waitTimeout"))
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

func parseRecipient(idStr string) (*id.ID, bool) {
	if idStr == "0" {
		return nil, false
	}

	var recipientID *id.ID
	if strings.HasPrefix(idStr, "0x") {
		recipientID = getUIDFromHexString(idStr[2:])
	} else if strings.HasPrefix(idStr, "b64:") {
		recipientID = getUIDFromb64String(idStr[4:])
	} else {
		recipientID = getUIDFromString(idStr)
	}
	return recipientID, isPrecanID(recipientID)
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

	if viper.GetBool("verboseRoundTracking") {
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

	rootCmd.PersistentFlags().Bool("verboseRoundTracking", false,
		"Verbose round tracking, keeps track and prints all rounds the "+
			"client was aware of while running. Defaults to false if not set.")
	viper.BindPFlag("verboseRoundTracking", rootCmd.PersistentFlags().Lookup(
		"verboseRoundTracking"))

	rootCmd.PersistentFlags().StringP(sessionFlag, "s",
		"", "Sets the initial storage directory for "+
			"client session data")
	viper.BindPFlag(sessionFlag, rootCmd.PersistentFlags().Lookup(sessionFlag))

	rootCmd.PersistentFlags().StringP("writeContact", "w",
		"-", "Write contact information, if any, to this file, "+
			" defaults to stdout")
	viper.BindPFlag("writeContact", rootCmd.PersistentFlags().Lookup(
		"writeContact"))

	rootCmd.PersistentFlags().StringP("password", "p", "",
		"Password to the session file")
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup(
		"password"))

	rootCmd.PersistentFlags().StringP("ndf", "n", "ndf.json",
		"Path to the network definition JSON file")
	viper.BindPFlag("ndf", rootCmd.PersistentFlags().Lookup("ndf"))

	rootCmd.PersistentFlags().StringP(logFlag, "l", "-",
		"Path to the log output path (- is stdout)")
	viper.BindPFlag(logFlag, rootCmd.PersistentFlags().Lookup(logFlag))

	rootCmd.Flags().StringP("regcode", "", "",
		"ReceptionIdentity code (optional)")
	viper.BindPFlag("regcode", rootCmd.Flags().Lookup("regcode"))

	rootCmd.PersistentFlags().StringP(messageFlag, "m", "",
		"Message to send")
	viper.BindPFlag(messageFlag, rootCmd.PersistentFlags().Lookup(messageFlag))

	rootCmd.Flags().UintP("sendid", "", 0,
		"Use precanned user id (must be between 1 and 40, inclusive)")
	viper.BindPFlag("sendid", rootCmd.Flags().Lookup("sendid"))

	rootCmd.Flags().StringP("destid", "d", "0",
		"ID to send message to (if below 40, will be precanned. Use "+
			"'0x' or 'b64:' for hex and base64 representations)")
	viper.BindPFlag("destid", rootCmd.Flags().Lookup("destid"))

	rootCmd.Flags().StringP(destFileFlag, "",
		"", "Read this contact file for the destination id")
	viper.BindPFlag(destFileFlag, rootCmd.Flags().Lookup(destFileFlag))

	rootCmd.Flags().UintP("sendCount",
		"", 1, "The number of times to send the message")
	viper.BindPFlag("sendCount", rootCmd.Flags().Lookup("sendCount"))
	rootCmd.Flags().UintP("sendDelay",
		"", 500, "The delay between sending the messages in ms")
	viper.BindPFlag("sendDelay", rootCmd.Flags().Lookup("sendDelay"))
	rootCmd.Flags().BoolP("splitSends",
		"", false, "Force sends to go over multiple rounds if possible")
	viper.BindPFlag("splitSends", rootCmd.Flags().Lookup("splitSends"))

	rootCmd.Flags().BoolP(verifySendFlag, "", false,
		"Ensure successful message sending by checking for round completion")
	viper.BindPFlag(verifySendFlag, rootCmd.Flags().Lookup(verifySendFlag))

	rootCmd.PersistentFlags().UintP("receiveCount",
		"", 1, "How many messages we should wait for before quitting")
	viper.BindPFlag("receiveCount", rootCmd.PersistentFlags().Lookup("receiveCount"))
	rootCmd.PersistentFlags().UintP("waitTimeout", "", 15,
		"The number of seconds to wait for messages to arrive")
	viper.BindPFlag("waitTimeout",
		rootCmd.PersistentFlags().Lookup("waitTimeout"))

	rootCmd.Flags().BoolP("unsafe", "", false,
		"Send raw, unsafe messages without e2e encryption.")
	viper.BindPFlag("unsafe", rootCmd.Flags().Lookup("unsafe"))

	rootCmd.PersistentFlags().BoolP("unsafe-channel-creation", "", false,
		"Turns off the user identity authenticated channel check, "+
			"automatically approving authenticated channels")
	viper.BindPFlag("unsafe-channel-creation",
		rootCmd.PersistentFlags().Lookup("unsafe-channel-creation"))

	rootCmd.Flags().BoolP("accept-channel", "", false,
		"Accept the channel request for the corresponding recipient ID")
	viper.BindPFlag("accept-channel",
		rootCmd.Flags().Lookup("accept-channel"))

	rootCmd.PersistentFlags().Bool("delete-channel", false,
		"DeleteFingerprint the channel information for the corresponding recipient ID")
	viper.BindPFlag("delete-channel",
		rootCmd.PersistentFlags().Lookup("delete-channel"))

	rootCmd.PersistentFlags().Bool("delete-receive-requests", false,
		"DeleteFingerprint the all received contact requests.")
	viper.BindPFlag("delete-receive-requests",
		rootCmd.PersistentFlags().Lookup("delete-receive-requests"))

	rootCmd.PersistentFlags().Bool("delete-sent-requests", false,
		"DeleteFingerprint the all sent contact requests.")
	viper.BindPFlag("delete-sent-requests",
		rootCmd.PersistentFlags().Lookup("delete-sent-requests"))

	rootCmd.PersistentFlags().Bool("delete-all-requests", false,
		"DeleteFingerprint the all contact requests, both sent and received.")
	viper.BindPFlag("delete-all-requests",
		rootCmd.PersistentFlags().Lookup("delete-all-requests"))

	rootCmd.PersistentFlags().Bool("delete-request", false,
		"DeleteFingerprint the request for the specified ID given by the "+
			"destfile flag's contact file.")
	viper.BindPFlag("delete-request",
		rootCmd.PersistentFlags().Lookup("delete-request"))

	rootCmd.Flags().BoolP("send-auth-request", "", false,
		"Send an auth request to the specified destination and wait"+
			"for confirmation")
	viper.BindPFlag("send-auth-request",
		rootCmd.Flags().Lookup("send-auth-request"))
	rootCmd.Flags().UintP("auth-timeout", "", 120,
		"The number of seconds to wait for an authentication channel"+
			"to confirm")
	viper.BindPFlag("auth-timeout",
		rootCmd.Flags().Lookup("auth-timeout"))

	rootCmd.Flags().BoolP("forceHistoricalRounds", "", false,
		"Force all rounds to be sent to historical round retrieval")
	viper.BindPFlag("forceHistoricalRounds",
		rootCmd.Flags().Lookup("forceHistoricalRounds"))

	// Network params
	rootCmd.Flags().BoolP("slowPolling", "", false,
		"Enables polling for unfiltered network updates with RSA signatures")
	viper.BindPFlag("slowPolling",
		rootCmd.Flags().Lookup("slowPolling"))

	rootCmd.Flags().Bool("forceMessagePickupRetry", false,
		"Enable a mechanism which forces a 50% chance of no message pickup, "+
			"instead triggering the message pickup retry mechanism")
	viper.BindPFlag("forceMessagePickupRetry",
		rootCmd.Flags().Lookup("forceMessagePickupRetry"))

	// E2E Params
	defaultE2EParams := session.GetDefaultParams()
	rootCmd.Flags().UintP("e2eMinKeys",
		"", uint(defaultE2EParams.MinKeys),
		"Minimum number of keys used before requesting rekey")
	viper.BindPFlag("e2eMinKeys", rootCmd.Flags().Lookup("e2eMinKeys"))
	rootCmd.Flags().UintP("e2eMaxKeys",
		"", uint(defaultE2EParams.MaxKeys),
		"Max keys used before blocking until a rekey completes")
	viper.BindPFlag("e2eMaxKeys", rootCmd.Flags().Lookup("e2eMaxKeys"))
	rootCmd.Flags().UintP("e2eNumReKeys",
		"", uint(defaultE2EParams.NumRekeys),
		"Number of rekeys reserved for rekey operations")
	viper.BindPFlag("e2eNumReKeys", rootCmd.Flags().Lookup("e2eNumReKeys"))
	rootCmd.Flags().Float64P("e2eRekeyThreshold",
		"", defaultE2EParams.RekeyThreshold,
		"Number between 0 an 1. Percent of keys used before a rekey is started")
	viper.BindPFlag("e2eRekeyThreshold", rootCmd.Flags().Lookup("e2eRekeyThreshold"))

	rootCmd.Flags().String("profile-cpu", "",
		"Enable cpu profiling to this file")
	viper.BindPFlag("profile-cpu", rootCmd.Flags().Lookup("profile-cpu"))

	// Proto user flags
	rootCmd.Flags().String("protoUserPath", "",
		"Path to proto user JSON file containing cryptographic primitives "+
			"the client will load")
	viper.BindPFlag("protoUserPath", rootCmd.Flags().Lookup("protoUserPath"))
	rootCmd.Flags().String("protoUserOut", "",
		"Path to which a normally constructed client "+
			"will write proto user JSON file")
	viper.BindPFlag("protoUserOut", rootCmd.Flags().Lookup("protoUserOut"))

	// Backup flags
	rootCmd.Flags().String("backupOut", "",
		"Path to output encrypted client backup. If no path is supplied, the "+
			"backup system is not started.")
	viper.BindPFlag("backupOut", rootCmd.Flags().Lookup("backupOut"))

	rootCmd.Flags().String("backupJsonOut", "",
		"Path to output unencrypted client JSON backup.")
	viper.BindPFlag("backupJsonOut", rootCmd.Flags().Lookup("backupJsonOut"))

	rootCmd.Flags().String("backupIn", "",
		"Path to load backup client from")
	viper.BindPFlag("backupIn", rootCmd.Flags().Lookup("backupIn"))

	rootCmd.Flags().String("backupPass", "",
		"Passphrase to encrypt/decrypt backup")
	viper.BindPFlag("backupPass", rootCmd.Flags().Lookup("backupPass"))

	rootCmd.Flags().String("backupIdList", "",
		"JSON file containing the backed up partner IDs")
	viper.BindPFlag("backupIdList", rootCmd.Flags().Lookup("backupIdList"))

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}
