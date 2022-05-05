package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"time"
)

// singleCmd is the single-use subcommand that allows for sending and responding
// to single-use messages.
var symmetricCmd = &cobra.Command{
	Use:   "symmetric",
	Short: "Send symmetric broadcast messages",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := initClient()

		// Write user contact to file
		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ReceptionID)
		jww.INFO.Printf("User Transmission: %s", user.TransmissionID)
		writeContact(user.GetContact())

		err := client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetNetworkInterface().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		// Create new symmetric or load from path if exists
		path, err := utils.ExpandPath(viper.GetString("path"))
		var symmetric *crypto.Symmetric
		if utils.Exists(path) {
			// Load symmetric from path
			symmBytes, err := utils.ReadFile(path)
			if err != nil {

			}
			symmetric, err = crypto.UnmarshalSymmetric(symmBytes)
			if err != nil {

			}
		} else {
			// New symmetric
			name := viper.GetString("name")
			desc := viper.GetString("description")
			if name == "" {
				jww.FATAL.Panicf("Name cannot be empty")
			} else if desc == "" {
				jww.FATAL.Panicf("description cannot be empty")
			}

			var pubKey *rsa.PublicKey
			var salt, pubKeyBytes []byte
			if viper.GetBool("new") {
				privKey, err := rsa.GenerateKey(client.GetRng().GetStream(), rsa.DefaultRSABitLen)
				if err != nil {

				}
				pubKey = privKey.GetPublic()
				pubKeyBytes = rsa.CreatePublicKeyPem(pubKey)
			} else {
				pubKeyBytes := []byte(viper.GetString("rsaPub"))
				pubKey, err = rsa.LoadPublicKeyFromPem(pubKeyBytes)
				if err != nil {

				}
				salt = []byte(viper.GetString("salt"))
			}

			h, err := hash.NewCMixHash()
			if err != nil {

			}
			h.Write([]byte(name))
			h.Write([]byte(desc))
			h.Write(salt)
			h.Write(pubKeyBytes)
			ridBytes := h.Sum(nil)

			rid := &id.ID{}
			copy(rid[:], ridBytes)
			rid.SetType(id.User)

			symmetric = &crypto.Symmetric{
				ReceptionID: rid,
				Name:        name,
				Description: desc,
				Salt:        salt,
				RsaPubKey:   pubKey,
			}

			symmBytes, err := symmetric.Marshal()
			if err != nil {

			}
			// Write to file if there
			if path != "" {
				err = utils.WriteFile(path, symmBytes, os.ModePerm, os.ModeDir)
				if err != nil {

				}
			} else {
				fmt.Printf("Symmetric marshalled: %+v", symmBytes)
			}
		}

		// Create receiver callback
		receiveChan := make(chan []byte, 100)
		cb := func(payload []byte,
			receptionID receptionID.EphemeralIdentity, round rounds.Round) {
			jww.INFO.Printf("Received symmetric message from %s over round %d", receptionID, round)
			receiveChan <- payload
		}

		// Connect to symmetric broadcast channel
		scl := broadcast.NewSymmetricClient(*symmetric, cb, client.GetNetworkInterface(), client.GetRng())
		message := viper.GetString("broadcast")
		fmt.Println(message)
		// Send a broadcast over the channel
		if message != "" {
			broadcastMessage, err := broadcast.NewSizedBroadcast(scl.MaxPayloadSize(), []byte(message))
			if err != nil {
				jww.ERROR.Printf("Failed to create sized broadcast: %+v", err)
			}
			rid, eid, err := scl.Broadcast(broadcastMessage, cmix.GetDefaultCMIXParams())
			if err != nil {
				jww.ERROR.Printf("Failed to send symmetric broadcast message: %+v", err)
			}
			jww.INFO.Printf("Sent symmetric broadcast message to %s over round %d", eid, rid)
		}

		// Receive messages over the channel
		waitSecs := viper.GetUint("waitTimeout")
		expectedCnt := viper.GetUint("receiveCount")
		waitTimeout := time.Duration(waitSecs) * time.Second
		receivedCount := uint(0)
		done := false
		for !done && expectedCnt != 0 {
			timeout := time.NewTimer(waitTimeout)
			select {
			case receivedPayload := <-receiveChan:
				receivedCount++
				receivedBroadcast, err := broadcast.DecodeSizedBroadcast(receivedPayload)
				if err != nil {
					jww.ERROR.Printf("Failed to decode sized broadcast: %+v", err)
					continue
				}
				fmt.Printf("Symmetric broadcast message %d/%d received: %s\n", receivedCount, expectedCnt, string(receivedBroadcast))
				if receivedCount == expectedCnt {
					done = true
				}
			case <-timeout.C:
				fmt.Println("Timed out")
				jww.ERROR.Printf("Timed out on message reception after %s!", waitTimeout)
				done = true
			}
		}

		jww.INFO.Printf("Received %d/%d Messages!", receivedCount, expectedCnt)
		scl.Stop()
		err = client.StopNetworkFollower()
		if err != nil {
			jww.WARN.Printf(
				"Failed to cleanly close threads: %+v\n",
				err)
		}
	},
}

func init() {
	// Single-use subcommand options
	symmetricCmd.Flags().StringP("name", "", "",
		"Symmetric channel name")
	_ = viper.BindPFlag("name", symmetricCmd.Flags().Lookup("name"))

	symmetricCmd.Flags().StringP("rsaPub", "", "",
		"Symmetric channel rsa pub key")
	_ = viper.BindPFlag("rsaPub", symmetricCmd.Flags().Lookup("rsaPub"))

	symmetricCmd.Flags().StringP("salt", "", "",
		"Symmetric channel salt")
	_ = viper.BindPFlag("salt", symmetricCmd.Flags().Lookup("salt"))

	symmetricCmd.Flags().StringP("description", "", "",
		"Symmetric channel description")
	_ = viper.BindPFlag("description", symmetricCmd.Flags().Lookup("description"))

	symmetricCmd.Flags().StringP("path", "", "",
		"Symmetric channel output path")
	_ = viper.BindPFlag("path", symmetricCmd.Flags().Lookup("path"))

	symmetricCmd.Flags().BoolP("new", "", false,
		"Create new symmetric channel")
	_ = viper.BindPFlag("new", symmetricCmd.Flags().Lookup("new"))

	symmetricCmd.Flags().StringP("broadcast", "", "",
		"Message to send via symmetric broadcast")
	_ = viper.BindPFlag("broadcast", symmetricCmd.Flags().Lookup("broadcast"))

	rootCmd.AddCommand(symmetricCmd)
}
