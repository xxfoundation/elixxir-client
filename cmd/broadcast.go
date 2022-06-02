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
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"time"
)

// singleCmd is the single-use subcommand that allows for sending and responding
// to single-use messages.
var broadcastCmd = &cobra.Command{
	Use:   "broadcast",
	Short: "Send broadcast messages",
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
			jww.FATAL.Panicf("Failed to start network follower: %+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetNetworkInterface().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		/* Set up underlying crypto broadcast.Channel */
		var channel *crypto.Channel
		var pk *rsa.PrivateKey
		keyPath := viper.GetString("keyPath")
		path, err := utils.ExpandPath(viper.GetString("chanPath"))
		if utils.Exists(path) {
			// Load symmetric from path
			cBytes, err := utils.ReadFile(path)
			if err != nil {
				jww.FATAL.Panicf("Failed to read channel from file at %s: %+v", path, err)
			}
			channel, err = crypto.UnmarshalChannel(cBytes)
			if err != nil {
				jww.FATAL.Panicf("Failed to unmarshal channel data %+v: %+v", cBytes, err)
			}
		} else {
			// Load in broadcast channel info
			name := viper.GetString("name")
			desc := viper.GetString("description")
			if name == "" {
				jww.FATAL.Panicf("Name cannot be empty")
			} else if desc == "" {
				jww.FATAL.Panicf("description cannot be empty")
			}

			var channel *crypto.Channel
			if viper.GetBool("new") {
				// Create a new broadcast channel
				channel, pk, err = crypto.NewChannel(name, desc, client.GetRng().GetStream())
				if err != nil {
					jww.FATAL.Panicf("Failed to create new channel: %+v", err)
				}

				if keyPath != "" {
					err = utils.WriteFile(path, rsa.CreatePrivateKeyPem(pk), os.ModePerm, os.ModeDir)
					if err != nil {
						jww.ERROR.Printf("Failed to write private key to path %s: %+v", path, err)
					}
				} else {
					fmt.Printf("Private key generated for channel: %+v", rsa.CreatePrivateKeyPem(pk))
				}
			} else {
				// Read rest of info from config & build object manually
				pubKeyBytes := []byte(viper.GetString("rsaPub"))
				pubKey, err := rsa.LoadPublicKeyFromPem(pubKeyBytes)
				if err != nil {
					jww.FATAL.Panicf("Failed to load public key at path: %+v", err)
				}
				salt := []byte(viper.GetString("salt"))

				rid, err := crypto.NewChannelID(name, desc, salt, pubKeyBytes)
				if err != nil {
					jww.FATAL.Panicf("Failed to generate channel ID: %+v", err)
				}

				channel = &crypto.Channel{
					ReceptionID: rid,
					Name:        name,
					Description: desc,
					Salt:        salt,
					RsaPubKey:   pubKey,
				}

				// Load key if it's there
				if keyPath != "" {
					if ep, err := utils.ExpandPath(keyPath); err == nil {
						keyBytes, err := utils.ReadFile(ep)
						if err != nil {
							jww.ERROR.Printf("Failed to read private key from %s: %+v", ep, err)
						}
						pk, err = rsa.LoadPrivateKeyFromPem(keyBytes)
						if err != nil {
							jww.ERROR.Printf("Failed to load private key %+v: %+v", keyBytes, err)
						}
					} else {
						jww.ERROR.Printf("Failed to expand private key path: %+v", err)
					}

				}
			}

			// Save channel to disk
			cBytes, err := channel.Marshal()
			if err != nil {
				jww.ERROR.Printf("Failed to marshal channel to bytes: %+v", err)
			}
			// Write to file if there
			if path != "" {
				err = utils.WriteFile(path, cBytes, os.ModePerm, os.ModeDir)
				if err != nil {
					jww.ERROR.Printf("Failed to write channel to file %s: %+v", path, err)
				}
			} else {
				fmt.Printf("Channel marshalled: %+v", cBytes)
			}
		}

		/* Broadcast client setup */

		// Create receiver callback
		receiveChan := make(chan []byte, 100)
		cb := func(payload []byte,
			receptionID receptionID.EphemeralIdentity, round rounds.Round) {
			jww.INFO.Printf("Received symmetric message from %s over round %d", receptionID, round)
			receiveChan <- payload
		}

		// Select broadcast method
		var method broadcast.Method
		symmetric := viper.GetBool("symmetric")
		asymmetric := viper.GetBool("asymmetric")
		if symmetric && asymmetric {
			jww.FATAL.Panicf("Cannot simultaneously broadcast symmetric & asymmetric")
		}
		if symmetric {
			method = broadcast.Symmetric
		} else if asymmetric {
			method = broadcast.Asymmetric
		}

		// Connect to broadcast channel
		bcl, err := broadcast.NewBroadcastChannel(*channel, cb, client.GetNetworkInterface(), client.GetRng(), broadcast.Param{Method: method})

		/* Create properly sized broadcast message */
		message := viper.GetString("broadcast")
		fmt.Println(message)
		var broadcastMessage []byte
		if message != "" {
			broadcastMessage, err = broadcast.NewSizedBroadcast(bcl.MaxPayloadSize(), []byte(message))
			if err != nil {
				jww.ERROR.Printf("Failed to create sized broadcast: %+v", err)
			}

		}

		/* Broadcast message to the channel */
		switch method {
		case broadcast.Symmetric:
			rid, eid, err := bcl.Broadcast(broadcastMessage, cmix.GetDefaultCMIXParams())
			if err != nil {
				jww.ERROR.Printf("Failed to send symmetric broadcast message: %+v", err)
			}
			jww.INFO.Printf("Sent symmetric broadcast message to %s over round %d", eid, rid)
		case broadcast.Asymmetric:
			if pk == nil {
				jww.FATAL.Panicf("CANNOT SEND ASYMMETRIC BROADCAST WITHOUT PRIVATE KEY")
			}
			rid, eid, err := bcl.BroadcastAsymmetric(pk, broadcastMessage, cmix.GetDefaultCMIXParams())
			if err != nil {
				jww.ERROR.Printf("Failed to send asymmetric broadcast message: %+v", err)
			}
			jww.INFO.Printf("Sent asymmetric broadcast message to %s over round %d", eid, rid)
		default:
			jww.WARN.Printf("Unknown broadcast type (this should not happen)")
		}

		/* Receive broadcast messages over the channel */
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
		bcl.Stop()
		err = client.StopNetworkFollower()
		if err != nil {
			jww.WARN.Printf("Failed to cleanly close threads: %+v\n", err)
		}
	},
}

func init() {
	// Single-use subcommand options
	broadcastCmd.Flags().StringP("name", "", "",
		"Symmetric channel name")
	_ = viper.BindPFlag("name", broadcastCmd.Flags().Lookup("name"))

	broadcastCmd.Flags().StringP("rsaPub", "", "",
		"Broadcast channel rsa pub key")
	_ = viper.BindPFlag("rsaPub", broadcastCmd.Flags().Lookup("rsaPub"))

	broadcastCmd.Flags().StringP("salt", "", "",
		"Broadcast channel salt")
	_ = viper.BindPFlag("salt", broadcastCmd.Flags().Lookup("salt"))

	broadcastCmd.Flags().StringP("description", "", "",
		"Broadcast channel description")
	_ = viper.BindPFlag("description", broadcastCmd.Flags().Lookup("description"))

	broadcastCmd.Flags().StringP("chanPath", "", "",
		"Broadcast channel output path")
	_ = viper.BindPFlag("chanPath", broadcastCmd.Flags().Lookup("chanPath"))

	broadcastCmd.Flags().StringP("keyPath", "", "",
		"Broadcast channel private key output path")
	_ = viper.BindPFlag("keyPath", broadcastCmd.Flags().Lookup("keyPath"))

	broadcastCmd.Flags().BoolP("new", "", false,
		"Create new broadcast channel")
	_ = viper.BindPFlag("new", broadcastCmd.Flags().Lookup("new"))

	broadcastCmd.Flags().StringP("broadcast", "", "",
		"Message contents for broadcast")
	_ = viper.BindPFlag("broadcast", broadcastCmd.Flags().Lookup("broadcast"))

	broadcastCmd.Flags().BoolP("symmetric", "", false,
		"Set broadcast method to symmetric")
	_ = viper.BindPFlag("symmetric", broadcastCmd.Flags().Lookup("symmetric"))

	broadcastCmd.Flags().BoolP("asymmetric", "", false,
		"Set broadcast method to asymmetric")
	_ = viper.BindPFlag("asymmetric", broadcastCmd.Flags().Lookup("asymmetric"))

	rootCmd.AddCommand(broadcastCmd)
}
