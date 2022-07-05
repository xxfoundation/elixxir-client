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
	"gitlab.com/elixxir/client/cmix"
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

// Deployment environment constants for the download-ndf code path
const (
	mainnet = "mainnet"
	release = "release"
	dev     = "dev"
	testnet = "testnet"
)

// URL constants pointing to the NDF of the associated deployment environment
// requested for the download-ndf code path.
const (
	mainNetUrl = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/mainnet.json"
	releaseUrl = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/release.json"
	devUrl     = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/default.json"
	testNetUrl = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/testnet.json"
)

// Certificates for deployment environments. Used to verify NDF signatures.
const (
	mainNetCert = `-----BEGIN CERTIFICATE-----
MIIFqTCCA5GgAwIBAgIUO0qHXSeKrOMucO+Zz82Mf1Zlq4gwDQYJKoZIhvcNAQEL
BQAwgYAxCzAJBgNVBAYTAktZMRQwEgYDVQQHDAtHZW9yZ2UgVG93bjETMBEGA1UE
CgwKeHggbmV0d29yazEPMA0GA1UECwwGRGV2T3BzMRMwEQYDVQQDDAp4eC5uZXR3
b3JrMSAwHgYJKoZIhvcNAQkBFhFhZG1pbnNAeHgubmV0d29yazAeFw0yMTEwMzAy
MjI5MjZaFw0zMTEwMjgyMjI5MjZaMIGAMQswCQYDVQQGEwJLWTEUMBIGA1UEBwwL
R2VvcmdlIFRvd24xEzARBgNVBAoMCnh4IG5ldHdvcmsxDzANBgNVBAsMBkRldk9w
czETMBEGA1UEAwwKeHgubmV0d29yazEgMB4GCSqGSIb3DQEJARYRYWRtaW5zQHh4
Lm5ldHdvcmswggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQD08ixnPWwz
FtBIEWx2SnFjBsdrSWCp9NcWXRtGWeq3ACz+ixiflj/U9U4b57aULeOAvcoC7bwU
j5w3oYxRmXIV40QSevx1z9mNcW3xbbacQ+yCgPPhhj3/c285gVVOUzURLBTNAi9I
EA59zAb8Vy0E6zfq4HRAhH11Q/10QgDjEXuGXra1k3IlemVsouiJGNAtKojNDE1N
x9HnraSEiXzdnV2GDplEvMHaLd3s9vs4XsiLB3VwKyHv7EH9+LOIra6pr5BWw+kD
2qHKGmQMOQe0a7nCirW/k9axH0WiA0XWuQu3U1WfcMEfdC/xn1vtubrdYjtzpXUy
oUEX5eHfu4OlA/zoH+trocfARDyBmTVbDy0P9imH//a6GUKDui9r3fXwEy5YPMhb
dKaNc7QWLPHMh1n25h559z6PqxxPT6UqFFbZD2gTw1sbbpjyqhLbnYguurkxY3jZ
ztW337hROzQ1/abbg/P59JA95Pmhkl8nqqDEf0buOmvMazq3Lwg92nuZ8gsdMKXB
xaEtTTpxhTPOqzc1/XQgScZnc+092MBDh3C2GMxzylOIdk+yF2Gyb+VWPUe29dSa
azzxsDXzRy8y8jaOjdSUWaLa/MgS5Dg1AfHtD55bdvqYzw3NEXIVarpMlzl+Z+6w
jvuwz8GyoMSVe+YEGgvSDvlfY/z19aqneQIDAQABoxkwFzAVBgNVHREEDjAMggp4
eC5uZXR3b3JrMA0GCSqGSIb3DQEBCwUAA4ICAQCp0JDub2w5vZQvFREyA+utZ/+s
XT05j1iTgIRKMa3nofDGERYJUG7FcTd373I2baS70PGx8FF1QuXhn4DNNZlW/SZt
pa1d0pAerqFrIzwOuWVDponYHQ8ayvsT7awCbwZEZE4RhooqS4LqnvtgFu/g7LuM
zkFN8TER7HAUn3P7BujLvcgtqk2LMDz+AgBRszDp/Bw7+1EJDeG9d7hC/stXgDV/
vpD1YDpxSmW4zjezFJqV6OdMOwo9RWVIktK3RXbFc6I5UJZ5kmzPe/I2oPPCBQvD
G3VqFLQe5ik5rXP7SgAN1fL/7KuQna0s42hkV64Z2ymCX69G1ofpgpEFaQLaxLbj
QOun0r8A3NyKvHRIh4K0dFcc3FSOF60Y6k769HKbOPmSDjSSg0qO9GEONBJ8BxAT
IHcHoTAOQoqGehdzepXQSjHsPqTXv3ZFFwCCgO0toI0Qhqwo89X6R3k+i4Kaktr7
mLiPO8s0nq1PZ1XrybKE9BCHkYH1JkUDA+M0pn4QAEx/BuM0QnGXoi1sImW3pEUG
NP7fjkISrD48P8P/TLS45sx5pB8MNGEsRw0lBKmuOdWDmdfhOltB6JxmbhpstNZp
6LVLK6SEOwE76xnHiisR2KyhTTiroUq73BgPFWkWhoJDPbmL1DHgnbdKwwstG8Qu
UGb8k8vh6tzqYZAOKg==
-----END CERTIFICATE-----`
	releaseCert = `-----BEGIN CERTIFICATE-----
MIIFtjCCA56gAwIBAgIJAJnUcpLbGSQiMA0GCSqGSIb3DQEBCwUAMIGMMQswCQYD
VQQGEwJVUzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCUNsYXJlbW9udDEQMA4GA1UE
CgwHRWxpeHhpcjEUMBIGA1UECwwLRGV2ZWxvcG1lbnQxEzARBgNVBAMMCmVsaXh4
aXIuaW8xHzAdBgkqhkiG9w0BCQEWEGFkbWluQGVsaXh4aXIuaW8wHhcNMjAxMTE3
MTkwMTUyWhcNMjIxMTE3MTkwMTUyWjCBjDELMAkGA1UEBhMCVVMxCzAJBgNVBAgM
AkNBMRIwEAYDVQQHDAlDbGFyZW1vbnQxEDAOBgNVBAoMB0VsaXh4aXIxFDASBgNV
BAsMC0RldmVsb3BtZW50MRMwEQYDVQQDDAplbGl4eGlyLmlvMR8wHQYJKoZIhvcN
AQkBFhBhZG1pbkBlbGl4eGlyLmlvMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIIC
CgKCAgEAvtByOoSS8SeMLvvHIuOGfnx0VgweveJHX93LUyJxr1RlVBXCgC5/QOQN
N3dmKWzu4YwaA2jtwaAMhkgdfyOcw6kuqfvQjxv99XRIRKM4GZQkJiym2cnorNu7
hm2/bxmj5TjpP9+vFzbjkJrpRQ80hsV7I9+NKzIhMK4YTgte/F/q9URESlMZxTbb
MFh3s5iiBfBLRNFFsHVdy8OVH+Jv5901cLn+yowaMDLrBMOWGlRROg82ZeRAranX
9X1s+6BclJ/cBe/LcDxGso5sco6UzrWHzpDTnOTzHoamQHYCXtAZP4XbzcqI6A5i
GFM2akuG9Wv3XZZv/6eJRnKS2GLkvv7dtzv+nalxoBKtyIE8ICIVOrb+pVJvY1Id
HOXkK9MEJJ6sZhddipUaQw6hD4I0dNEt30Ugq9zTgFcEnM2R7qKpIDmxrRbcl280
TQGNYgdidzleNdZbjcTvsMVhcxPXCY+bVX1xICD1oJiZZbZFejBvPEfLYyzSdYp+
awX5OnLVSrQtTJu9yz5q3q5pHhxJnqS/CVGLTvzLfmk7BGwRZZuK87LnSixyYfpd
S23qI45AEUINEE0HDZsI+KBq0oVlDB0Z3AZpWauRDqY3o6JIbIOpqmZc6KntyL7j
YCAhbB1tchS47PpbIxUgMMGoR3MBkJutPqtTWCEE3l5jvv0CknUCAwEAAaMZMBcw
FQYDVR0RBA4wDIIKZWxpeHhpci5pbzANBgkqhkiG9w0BAQsFAAOCAgEACLoxE3nh
3VzXH2lQo1QzjKkG/+1m75T0l9Wn9uxa2W/90qBCfim1CPfWUstGdRLBi8gjDevV
zK5HN+Cpz2E22qByeN9fl6rJC4zd1vIdexEre5h7goWoV+qFPhOACElor1tF5UQ2
GD+NFH+Z0ALG1u8db0hBv8NCbtD4YzcQzzINEbs9gp/Sq3cRzkz1wCufFwJwr7+R
0YqZfPj/v/w9G9wSUys1s3i4xr2u87T/bPF68VRg6r1+kXRSRevXd99wKwap52jY
zOwsDGZF9BHMpFVYR/yZhfzSK3F1DmvwuqOsfwSFIjrUjfRlwS28zyZ8rjBq1suD
EAdvYCLDmBSGssNh8E20PHmk5UROYFGEEhlK5ZKj/f1HOmMiOX461XK6HODYyitq
Six2dPi1ZlBJW83DyFqSWJaUR/CluBYmqrWoBX+chv54bU2Y9j/sA/O98wa7trsk
ctzvAcXjhXm6ESRVVD/iZvkW5MP2mkgbDpW3RP9souK5JzbcpC7i3hEcAqPSPgzL
94kHDpYNY7jcGQC4CjPdfBi+Tf6il/QLFRFgyHm2ze3+qrlPT6SQ4hSSH1iXyf4v
tlqu6u77fbF9yaHtq7dvYxH1WioIUxMqbIC1CNgGC1Y/LhzgLRKPSTBCrbQyTcGc
0b5cTzVKxdP6v6WOAXVOEkXTcBPZ4nEZxY0=
-----END CERTIFICATE-----`
	devCert = `-----BEGIN CERTIFICATE-----
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
EnretBzQkeKeBwoB2u6NTiOmUjk=
-----END CERTIFICATE-----`
	testNetCert = ``
)

// Key used for storing xxdk.ReceptionIdentity objects
const identityStorageKey = "identityStorageKey"

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

		client := initE2e()

		jww.INFO.Printf("Client Initialized...")

		receptionIdentity := client.GetReceptionIdentity()
		jww.INFO.Printf("User: %s", receptionIdentity.ID)
		writeContact(receptionIdentity.GetContact())

		var recipientContact contact.Contact
		var recipientID *id.ID

		destFile := viper.GetString("destfile")
		destId := viper.GetString("destid")
		sendId := viper.GetString("sendid")
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
		msgBody := viper.GetString("message")
		time.Sleep(10 * time.Second)

		// Accept auth request for this recipient
		authConfirmed := false
		paramsE2E := e2e.GetDefaultParams()
		if viper.GetBool("accept-channel") {
			// Verify that the confirmation message makes it to the
			// original sender
			if viper.GetBool("verify-sends") {
				acceptChannelVerified(client, recipientID)
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
			jww.INFO.Printf("Authenticated partners saved: %v\n    PartnersList: %+v",
				!client.GetStorage().GetKV().IsMemStore(), client.GetE2E().GetAllPartnerIDs())
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

						if viper.GetBool("verify-sends") { // Verify message sends were successful
							retryChan := make(chan struct{})
							done := make(chan struct{}, 1)

							// Construct the callback function which
							// verifies successful message send or retries
							f := func(allRoundsSucceeded, timedOut bool,
								rounds map[id.Round]cmix.RoundResult) {
								printRoundResults(
									rounds, roundIDs, payload, recipientID)
								if !allRoundsSucceeded {
									retryChan <- struct{}{}
								} else {
									done <- struct{}{}
								}
							}

							// Monitor rounds for results
							err = client.GetCmix().GetRoundResults(paramsE2E.CMIXParams.Timeout, f, roundIDs...)
							if err != nil {
								jww.DEBUG.Printf("Could not verify messages were sent successfully, resending messages...")
								continue
							}

							select {
							case <-retryChan:
								// On a retry, go to the top of the loop
								jww.DEBUG.Printf("Messages were not sent successfully, resending messages...")
								continue
							case <-done:
								// Close channels on verification success
								close(done)
								close(retryChan)
								break
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

// initCmix returns a newly-initialized xxdk.Cmix object and its stored xxdk.ReceptionIdentity
func initCmix() (*xxdk.Cmix, xxdk.ReceptionIdentity) {
	logLevel := viper.GetUint("logLevel")
	initLog(logLevel, viper.GetString("log"))
	jww.INFO.Printf(Version())

	pass := parsePassword(viper.GetString("password"))
	storeDir := viper.GetString("session")
	regCode := viper.GetString("regcode")
	precannedID := viper.GetUint("sendid")
	userIDprefix := viper.GetString("userid-prefix")
	protoUserPath := viper.GetString("protoUserPath")
	backupPath := viper.GetString("backupIn")
	backupPass := []byte(viper.GetString("backupPass"))

	// FIXME: All branches of the upcoming path
	var knownReception xxdk.ReceptionIdentity

	// create a new client if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Load NDF
		ndfJSON, err := ioutil.ReadFile(viper.GetString("ndf"))
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

// initE2e returns a fully-formed xxdk.E2e object
func initE2e() *xxdk.E2e {
	_, receptionIdentity := initCmix()

	pass := parsePassword(viper.GetString("password"))
	storeDir := viper.GetString("session")
	jww.DEBUG.Printf("sessionDur: %v", storeDir)

	params := initParams()

	// load the client
	baseClient, err := xxdk.LoadCmix(storeDir, pass, params)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	authCbs = makeAuthCallbacks(
		viper.GetBool("unsafe-channel-creation"))

	// Force LoginLegacy for precanned senderID
	var client *xxdk.E2e
	if isPrecanID(receptionIdentity.ID) {
		jww.INFO.Printf("Using LoginLegacy for precan sender")
		client, err = xxdk.LoginLegacy(baseClient, authCbs)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	} else {
		jww.INFO.Printf("Using Login for non-precan sender")
		client, err = xxdk.Login(baseClient, authCbs, receptionIdentity)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

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
		me := client.GetReceptionIdentity().GetContact()
		jww.INFO.Printf("Requesting auth channel from: %s",
			recipientID)

		// Verify that the auth request makes it to the recipient
		// by monitoring the round result
		if viper.GetBool("verify-sends") {
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
		if viper.GetBool("verify-sends") {
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
	rootCmd.PersistentFlags().UintP("logLevel", "v", 0,
		"Verbose mode for debugging")
	viper.BindPFlag("logLevel", rootCmd.PersistentFlags().Lookup("logLevel"))

	rootCmd.PersistentFlags().Bool("verboseRoundTracking", false,
		"Verbose round tracking, keeps track and prints all rounds the "+
			"client was aware of while running. Defaults to false if not set.")
	viper.BindPFlag("verboseRoundTracking", rootCmd.PersistentFlags().Lookup("verboseRoundTracking"))

	rootCmd.PersistentFlags().StringP("session", "s",
		"", "Sets the initial storage directory for "+
			"client session data")
	viper.BindPFlag("session", rootCmd.PersistentFlags().Lookup("session"))

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

	rootCmd.PersistentFlags().StringP("log", "l", "-",
		"Path to the log output path (- is stdout)")
	viper.BindPFlag("log", rootCmd.PersistentFlags().Lookup("log"))

	rootCmd.Flags().StringP("regcode", "", "",
		"ReceptionIdentity code (optional)")
	viper.BindPFlag("regcode", rootCmd.Flags().Lookup("regcode"))

	rootCmd.PersistentFlags().StringP("message", "m", "",
		"Message to send")
	viper.BindPFlag("message", rootCmd.PersistentFlags().Lookup("message"))

	rootCmd.Flags().UintP("sendid", "", 0,
		"Use precanned user id (must be between 1 and 40, inclusive)")
	viper.BindPFlag("sendid", rootCmd.Flags().Lookup("sendid"))

	rootCmd.Flags().StringP("destid", "d", "0",
		"ID to send message to (if below 40, will be precanned. Use "+
			"'0x' or 'b64:' for hex and base64 representations)")
	viper.BindPFlag("destid", rootCmd.Flags().Lookup("destid"))

	rootCmd.Flags().StringP("destfile", "",
		"", "Read this contact file for the destination id")
	viper.BindPFlag("destfile", rootCmd.Flags().Lookup("destfile"))

	rootCmd.Flags().UintP("sendCount",
		"", 1, "The number of times to send the message")
	viper.BindPFlag("sendCount", rootCmd.Flags().Lookup("sendCount"))
	rootCmd.PersistentFlags().UintP("sendDelay",
		"", 500, "The delay between sending the messages in ms")
	viper.BindPFlag("sendDelay", rootCmd.PersistentFlags().Lookup("sendDelay"))
	rootCmd.Flags().BoolP("splitSends",
		"", false, "Force sends to go over multiple rounds if possible")
	viper.BindPFlag("splitSends", rootCmd.Flags().Lookup("splitSends"))

	rootCmd.Flags().BoolP("verify-sends", "", false,
		"Ensure successful message sending by checking for round completion")
	viper.BindPFlag("verify-sends", rootCmd.Flags().Lookup("verify-sends"))

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
	rootCmd.Flags().UintP("auth-timeout", "", 60,
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
