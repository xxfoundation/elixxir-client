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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/xxdk"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"time"
)

const channelsPrintHeader = "CHANNELS"

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
			channelIdentity, err = readChannelIdentity()
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to read channel identity: %+v", err)
			}
		} else {
			// Generate channel identity if extant one does not exist
			channelIdentity, err = cryptoChannel.GenerateIdentity(rng)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to generate identity for channel: %+v",
					channelsPrintHeader, err)
			}
		}

		// Construct mock event model builder
		mockEventModelBuilder := func(path string) (channels.EventModel, error) {
			return mockEventModel{}, nil
		}

		// Construct channels manager
		chanManager, err := channels.NewManager(channelIdentity,
			user.GetStorage().GetKV(), user.GetCmix(), user.GetRng(),
			mockEventModelBuilder)
		if err != nil {
			jww.FATAL.Panicf("[%s] Failed to create channels manager: %+v",
				channelsPrintHeader, err)
		}

		// Load in channel info
		name := viper.GetString(channelsNameFlag)
		desc := viper.GetString(channelsDescriptionFlag)
		if name == "" {
			jww.FATAL.Panicf("[%s] Name cannot be empty", channelsPrintHeader)
		} else if desc == "" {
			jww.FATAL.Panicf("[%s] Description cannot be empty", channelsPrintHeader)
		}

		var channel *cryptoBroadcast.Channel
		keyPath := viper.GetString(channelsKeyPathFlag)
		chanPath := viper.GetString(channelsChanPathFlag)
		// Create new channel
		if viper.GetBool(channelsNewFlag) {
			channel, err = createNewChannel(name, desc, keyPath, chanPath, user)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to create new channel: %+v",
					channelsPrintHeader, err)
			}
		} else {
			// Load channel
			marshalledChan, err := utils.ReadFile(chanPath)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to read channel from file: %+v",
					channelsPrintHeader, err)
			}

			channel, err = cryptoBroadcast.UnmarshalChannel(marshalledChan)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to unmarshal channel: %+v",
					channelsPrintHeader, err)
			}
		}

		// Join channel
		err = chanManager.JoinChannel(channel)
		if err != nil {
			jww.FATAL.Panicf("[%s] Failed to join channel: %+v",
				channelsPrintHeader, err)
		}

		err = makeChannelReceptionHandler(channels.Text, chanManager)
		if err != nil {
			jww.FATAL.Panicf("[%s] Failed to create reception handler for message type %s: %+v",
				channels.Text, err)
		}

		if viper.IsSet(channelsSendFlag) {
			message := viper.GetString(channelsSendFlag)
			chanMsgId, round, _, err := chanManager.SendMessage(
				channel.ReceptionID, message, 5*time.Second,
				cmix.GetDefaultCMIXParams())
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to send message to channel: %+v",
					channelsPrintHeader, err)
			}

			jww.INFO.Printf("[%s] Sent message (%s) to channel %s (ID %s) with message ID %s on round %d",
				channelsPrintHeader, message, channel.Name, channel.ReceptionID, chanMsgId, round.ID)
			fmt.Printf("Sent message (%s) to channel %s\n", message, channel.Name)
		}

		if viper.IsSet(channelsLeaveFlag) {
			err = chanManager.LeaveChannel(channel.ReceptionID)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to leave channel %s (ID %s): %+v",
					channelsPrintHeader, channel.Name, channel.ReceptionID)
			}

			fmt.Printf("Successfully left channel %s\n", channel.Name)
		}

	},
}

// createNewChannel is a helper function which creates a new channel.
func createNewChannel(name, desc, keyPath, chanPath string,
	user *xxdk.E2e) (*cryptoBroadcast.Channel, error) {
	// Create a new  channel
	channel, pk, err := cryptoBroadcast.NewChannel(name, desc,
		cryptoBroadcast.Public,
		user.GetCmix().GetMaxMessageLength(), user.GetRng().GetStream())
	if err != nil {
		return nil, errors.Errorf("failed to create new channel: %+v", err)
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
		return nil, errors.Errorf("failed to marshal channel: %+v", err)
	}

	err = utils.WriteFileDef(chanPath, marshalledChan)
	if err != nil {
		return nil, errors.Errorf("failed to write channel to file: %+v",
			err)
	}

	return channel, nil

}

func makeChannelReceptionHandler(msgType channels.MessageType,
	chanManager channels.Manager) error {
	// Construct receiver callback
	messageReceptionCb := func(channelID *id.ID,
		messageID cryptoChannel.MessageID,
		messageType channels.MessageType,
		nickname string, content []byte, pubKey ed25519.PublicKey,
		codeset uint8, timestamp time.Time, lease time.Duration,
		round rounds.Round, status channels.SentStatus) uint64 {
		channelReceivedMessage, err := chanManager.GetChannel(channelID)
		if err != nil {
			jww.FATAL.Panicf("[%s] Failed to find channel for %s: %+v",
				channelsPrintHeader, channelID, err)
		}
		jww.INFO.Printf("[%s] Received message (%s) from %s",
			channelsPrintHeader, content, channelReceivedMessage.Name)
		fmt.Printf("Received message (%s) from %s\n",
			content, channelReceivedMessage.Name)
		return 0
	}
	return chanManager.RegisterReceiveHandler(msgType,
		messageReceptionCb)
}

// readChannelIdentity is a helper function to read a channel identity.
func readChannelIdentity() (cryptoChannel.PrivateIdentity, error) {
	path, err := utils.ExpandPath(viper.GetString(channelsChanIdentityPathFlag))
	if err != nil {
		return cryptoChannel.PrivateIdentity{},
			errors.Errorf("Failed to expand file path: %+v",
				err)
	}
	// Load channel identity from path if given from path
	cBytes, err := utils.ReadFile(path)
	if err != nil {
		return cryptoChannel.PrivateIdentity{},
			errors.Errorf("failed to read channel identity from file at %s: %+v",
				path, err)
	}
	channelIdentity, err := cryptoChannel.UnmarshalPrivateIdentity(cBytes)
	if err != nil {
		return cryptoChannel.PrivateIdentity{},
			errors.Errorf("Failed to unmarshal channel data %+v: %+v",
				string(cBytes), err)
	}

	return channelIdentity, nil
}

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

func init() {
	channelsCmd.Flags().String(channelsNameFlag, "Channel Name",
		"The name of the new channel to create.")
	bindFlagHelper(channelsNameFlag, channelsCmd)

	channelsCmd.Flags().String(channelsChanIdentityPathFlag, "",
		"The file path for the channel identity to be written to.")
	bindFlagHelper(channelsChanIdentityPathFlag, channelsCmd)

	channelsCmd.Flags().String(channelsChanPathFlag, "",
		"The file path for the channel information to be written to.")
	bindFlagHelper(channelsChanPathFlag, channelsCmd)

}
