////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"crypto/ed25519"
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	rsa2 "gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"time"
)

// mockEventModel is the CLI implementation of the channels.EventModel interface.
type mockEventModel struct{}

func (m mockEventModel) JoinChannel(channel *cryptoBroadcast.Channel) {
	jww.WARN.Printf("JoinChannel is unimplemented in the CLI event model!")
}

func (m mockEventModel) LeaveChannel(channelID *id.ID) {
	jww.WARN.Printf("LeaveChannel is unimplemented in the CLI event model!")
}

func (m mockEventModel) ReceiveMessage(channelID *id.ID,
	messageID cryptoChannel.MessageID, nickname, text string, pubKey ed25519.PublicKey,
	codeset uint8, timestamp time.Time, lease time.Duration, round rounds.Round,
	mType channels.MessageType, status channels.SentStatus) uint64 {
	jww.WARN.Printf("ReceiveMessage is unimplemented in the CLI event model!")
	return 0
}

func (m mockEventModel) ReceiveReply(channelID *id.ID,
	messageID cryptoChannel.MessageID, reactionTo cryptoChannel.MessageID,
	nickname, text string, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	mType channels.MessageType, status channels.SentStatus) uint64 {
	jww.WARN.Printf("ReceiveReply is unimplemented in the CLI event model!")
	return 0
}

func (m mockEventModel) ReceiveReaction(channelID *id.ID,
	messageID cryptoChannel.MessageID, reactionTo cryptoChannel.MessageID,
	nickname, reaction string, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	mType channels.MessageType, status channels.SentStatus) uint64 {
	jww.WARN.Printf("ReceiveReaction is unimplemented in the CLI event model!")
	return 0
}

func (m mockEventModel) UpdateSentStatus(uuid uint64,
	messageID cryptoChannel.MessageID, timestamp time.Time, round rounds.Round,
	status channels.SentStatus) {
	jww.WARN.Printf("UpdateSentStatus is unimplemented in the CLI event model!")
}

// connectionCmd handles the operation of connection operations within the CLI.
var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "Runs clients using the channels API.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Create client
		cmixParams, e2eParams := initParams()
		authCbs := makeAuthCallbacks(
			viper.GetBool(unsafeChannelCreationFlag), e2eParams)
		user := initE2e(cmixParams, e2eParams, authCbs)

		// Print user's reception ID
		identity := user.GetReceptionIdentity()
		jww.INFO.Printf("User: %s", identity.ID)

		// Wait for user to be connected to network
		err := user.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("[CHANNELS] %+v", err)
		}

		rng := user.GetRng().GetStream()
		defer rng.Close()

		/* Set up underlying crypto broadcast.Channel */
		var channelIdentity cryptoChannel.PrivateIdentity
		if viper.IsSet(channelsChanIdentityPathFlag) {
			path, err := utils.ExpandPath(viper.GetString(channelsChanIdentityPathFlag))
			if err != nil {
				jww.FATAL.Panicf("[CHANNELS] Failed to load identity: %+v", err)
			}
			// Load channel identity from path if given from path
			cBytes, err := utils.ReadFile(path)
			if err != nil {
				jww.FATAL.Panicf("[CHANNELS] Failed to read channel identity from file at %s: %+v", path, err)
			}
			channelIdentity, err = cryptoChannel.UnmarshalPrivateIdentity(cBytes)
			if err != nil {
				jww.FATAL.Panicf("[CHANNELS] Failed to unmarshal channel data %+v: %+v", cBytes, err)
			}
		} else {
			// Generate channel identity if extant one does not exist
			channelIdentity, err = cryptoChannel.GenerateIdentity(rng)
			if err != nil {
				jww.FATAL.Panicf("[CHANNELS] Failed to generate identity for channel: %+v", err)
			}
		}

		// Construct mock event model builder
		mockEventModelBuilder := func(path string) (channels.EventModel, error) {
			return mockEventModel{}, nil
		}

		// Construct channels manager
		chanManager, err := channels.NewManager(channelIdentity, user.GetStorage().GetKV(),
			user.GetCmix(), user.GetRng(), mockEventModelBuilder)
		if err != nil {
			jww.FATAL.Panicf("[CHANNELS] Failed to create channels manager: %+v", err)
		}

		// Load in channel info
		name := viper.GetString(channelsNameFlag)
		desc := viper.GetString(channelsDescriptionFlag)
		if name == "" {
			jww.FATAL.Panicf("[CHANNELS] Name cannot be empty")
		} else if desc == "" {
			jww.FATAL.Panicf("[CHANNELS] Description cannot be empty")
		}

		var channel *cryptoBroadcast.Channel
		var pk rsa2.PrivateKey
		keyPath := viper.GetString(channelsKeyPathFlag)
		chanPath := viper.GetString(channelsChanPathFlag)
		// Create new channel
		if viper.GetBool(channelsNewFlag) {
			// Create a new  channel
			channel, pk, err = cryptoBroadcast.NewChannel(name, desc,
				cryptoBroadcast.Public,
				user.GetCmix().GetMaxMessageLength(), user.GetRng().GetStream())
			if err != nil {
				jww.FATAL.Panicf("Failed to create new channel: %+v", err)
			}

			if keyPath != "" {
				err = utils.WriteFile(keyPath, pk.MarshalPem(), os.ModePerm, os.ModeDir)
				if err != nil {
					jww.ERROR.Printf("Failed to write private key to path %s: %+v", keyPath, err)
				}
			} else {
				fmt.Printf("Private key generated for channel: %+v", pk.MarshalPem())
			}
			fmt.Printf("New channel generated")

			// Write channel to file
			marshalledChan, err := channel.Marshal()
			if err != nil {
				jww.FATAL.Panicf("[CHANNELS] Failed to marshal channel: %+v", err)
			}

			err = utils.WriteFileDef(chanPath, marshalledChan)
			if err != nil {
				jww.FATAL.Panicf("[CHANNELS] Failed to write channel to file: %+v", err)
			}

		} else {
			// Load channel
			marshalledChan, err := utils.ReadFile(chanPath)
			if err != nil {
				jww.FATAL.Panicf("[CHANNELS] Failed to read channel from file: %+v", err)
			}

			channel, err = cryptoBroadcast.UnmarshalChannel(marshalledChan)
			if err != nil {
				jww.FATAL.Panicf("[CHANNELS] Failed to unmarshal channel: %+v", err)
			}

		}

		// Join channel
		err = chanManager.JoinChannel(channel)
		if err != nil {
			jww.FATAL.Panicf("[CHANNELS] Failed to join channel: %+v", err)
		}

	},
}
