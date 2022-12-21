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
const integrationChannelMessage = channels.MessageType(0)

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

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		user.GetCmix().AddHealthCallback(
			func(isConnected bool) {
				connected <- isConnected
			})
		waitUntilConnected(connected)

		// After connection, wait until registered with at least 85% of nodes
		for numReg, total := 1, 100; numReg < (total*3)/4; {
			time.Sleep(1 * time.Second)

			numReg, total, err = user.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf(
					"Failed to get node registration status: %+v", err)
			}

			jww.INFO.Printf("Registering with nodes (%d/%d)...", numReg, total)
		}

		rng := user.GetRng().GetStream()
		defer rng.Close()

		/* Set up underlying crypto broadcast.Channel */
		var channelIdentity cryptoChannel.PrivateIdentity

		// Construct mock event model builder
		mockEventModel := &eventModel{}
		mockEventModelBuilder := func(path string) (channels.EventModel, error) {
			return mockEventModel, nil
		}

		// Load path of channel identity
		path, err := utils.ExpandPath(viper.GetString(channelsChanIdentityPathFlag))
		if err != nil {
			jww.FATAL.Panicf("Failed to expand file path: %+v",
				err)
		}

		// Read or create new channel identity
		if utils.Exists(path) {
			// Load channel identity
			channelIdentity, err = readChannelIdentity(path)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to read channel identity: %+v",
					channelsPrintHeader, err)
			}

		} else {
			// Generate channel identity if extant one does not exist
			channelIdentity, err = cryptoChannel.GenerateIdentity(rng)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to generate identity for channel: %+v",
					channelsPrintHeader, err)
			}

			err = utils.WriteFileDef(path, channelIdentity.Marshal())
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to write channel identity to file: %+v",
					channelsPrintHeader, err)
			}
		}

		// Construct channels manager
		chanManager, err := channels.NewManager(channelIdentity,
			user.GetStorage().GetKV(), user.GetCmix(), user.GetRng(),
			mockEventModelBuilder, user.AddService)
		if err != nil {
			jww.FATAL.Panicf("[%s] Failed to create channels manager: %+v",
				channelsPrintHeader, err)
		}

		mockEventModel.api = chanManager

		// Load in channel info
		var channel *cryptoBroadcast.Channel
		chanPath := viper.GetString(channelsChanPathFlag)
		// Create new channel
		if viper.GetBool(channelsNewFlag) {
			channel, err = createNewChannel(chanPath, user)
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

		// Only join if we created it or are told to join
		if viper.GetBool(channelsNewFlag) || viper.GetBool(channelsJoinFlag) {
			// Join channel
			err = chanManager.JoinChannel(channel)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to join channel: %+v",
					channelsPrintHeader, err)
			}
			fmt.Printf("Successfully joined channel %s\n", channel.Name)
		}

		// Register a callback for the expected message to be received.
		err = makeChannelReceptionHandler(integrationChannelMessage,
			chanManager)
		if err != nil {
			jww.FATAL.Panicf("[%s] Failed to create reception handler for message type %s: %+v",
				channelsPrintHeader, channels.Text, err)
		}

		// Send message
		if viper.GetBool(channelsSendFlag) {
			msgBody := []byte(viper.GetString(messageFlag))
			err = sendMessageToChannel(chanManager, channel, msgBody)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to send message: %+v",
					channelsPrintHeader, err)
			}
		}

		// Leave channel
		if viper.IsSet(channelsLeaveFlag) {
			err = chanManager.LeaveChannel(channel.ReceptionID)
			if err != nil {
				jww.FATAL.Panicf("[%s] Failed to leave channel %s (ID %s): %+v",
					channelsPrintHeader, channel.Name, channel.ReceptionID, err)
			}

			fmt.Printf("Successfully left channel %s\n", channel.Name)
		}

	},
}

// createNewChannel is a helper function which creates a new channel.
func createNewChannel(chanPath string, user *xxdk.E2e) (
	*cryptoBroadcast.Channel, error) {

	keyPath := viper.GetString(channelsKeyPathFlag)
	name := viper.GetString(channelsNameFlag)
	desc := viper.GetString(channelsDescriptionFlag)
	if name == "" {
		jww.FATAL.Panicf("[%s] Name cannot be empty", channelsPrintHeader)
	} else if desc == "" {
		jww.FATAL.Panicf("[%s] Description cannot be empty", channelsPrintHeader)
	}

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
		jww.INFO.Printf("Private key generated for channel: %+v\n", pk.MarshalPem())
	}
	fmt.Printf("New channel generated\n")

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

// sendMessageToChannel is a helper function which will send a message to a
// channel.
func sendMessageToChannel(chanManager channels.Manager,
	channel *cryptoBroadcast.Channel, msgBody []byte) error {
	jww.INFO.Printf("[%s] Sending message (%s) to channel %s", channelsPrintHeader, msgBody,
		channel.Name)
	chanMsgId, round, _, err := chanManager.SendGeneric(
		channel.ReceptionID, integrationChannelMessage, msgBody, 5*time.Second,
		true, cmix.GetDefaultCMIXParams())
	if err != nil {
		return errors.Errorf("%+v", err)
	}

	jww.INFO.Printf("[%s] Sent message (%s) to channel %s (ID %s) with message ID %s on round %d",
		channelsPrintHeader, msgBody,
		channel.Name, channel.ReceptionID, chanMsgId, round.ID)
	fmt.Printf("Sent message (%s) to channel %s\n", msgBody, channel.Name)

	return nil
}

// makeChannelReceptionHandler is a helper function which will register with the
// channels.Manager a reception callback for the given message type.
func makeChannelReceptionHandler(msgType channels.MessageType,
	chanManager channels.Manager) error {
	// Construct receiver callback
	cb := func(channelID *id.ID, _ cryptoChannel.MessageID,
		_ channels.MessageType, _ string, content, _ []byte,
		_ ed25519.PublicKey, _ uint8, _, _ time.Time, _ time.Duration,
		_ rounds.Round, _ channels.SentStatus, _, _ bool) uint64 {
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
		channels.NewReceiveMessageHandler("", cb, true, true, true))
}

// readChannelIdentity is a helper function to read a channel identity.
func readChannelIdentity(path string) (cryptoChannel.PrivateIdentity, error) {
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

// eventModel is the CLI implementation of the channels.EventModel interface.
type eventModel struct {
	api channels.Manager
}

func (m *eventModel) JoinChannel(*cryptoBroadcast.Channel) {
	jww.WARN.Printf("JoinChannel is unimplemented in the CLI event model!")
}

func (m *eventModel) LeaveChannel(*id.ID) {
	jww.WARN.Printf("LeaveChannel is unimplemented in the CLI event model!")
}

func (m *eventModel) ReceiveMessage(_ *id.ID, _ cryptoChannel.MessageID, _,
	text string, _ ed25519.PublicKey, _ uint8, _ time.Time, _ time.Duration,
	_ rounds.Round, _ channels.MessageType, _ channels.SentStatus,
	_ bool) uint64 {
	jww.INFO.Printf("[%s] Received message (%s) from channel",
		channelsPrintHeader, text)
	fmt.Printf("Received message (%s) from channel\n", text)
	return 0
}

func (m *eventModel) ReceiveReply(channelID *id.ID, _ cryptoChannel.MessageID,
	_ cryptoChannel.MessageID, _, _ string, _ ed25519.PublicKey, _ uint8,
	_ time.Time, _ time.Duration, _ rounds.Round, _ channels.MessageType,
	_ channels.SentStatus, _ bool) uint64 {
	c, err := m.api.GetChannel(channelID)
	if err != nil {
		jww.FATAL.Panicf("[%s] Failed to get channel with ID %s",
			channelsPrintHeader, channelID)
	}
	fmt.Printf("Received reply for channel %s\n", c.Name)
	return 0
}

func (m *eventModel) ReceiveReaction(channelID *id.ID,
	_ cryptoChannel.MessageID, _ cryptoChannel.MessageID, _, _ string,
	_ ed25519.PublicKey, _ uint8, _ time.Time, _ time.Duration, _ rounds.Round,
	_ channels.MessageType, _ channels.SentStatus, _ bool) uint64 {
	c, err := m.api.GetChannel(channelID)
	if err != nil {
		jww.FATAL.Panicf("[%s] Failed to get channel with ID %s",
			channelsPrintHeader, channelID)
	}
	fmt.Printf("Received reaction for channel %s\n", c.Name)
	return 0
}

func (m *eventModel) UpdateFromUUID(uint64, *cryptoChannel.MessageID,
	*time.Time, *rounds.Round, *bool, *bool, *channels.SentStatus) {
	jww.WARN.Printf("UpdateFromUUID is unimplemented in the CLI event model!")
}

func (m *eventModel) UpdateFromMessageID(cryptoChannel.MessageID, *time.Time,
	*rounds.Round, *bool, *bool, *channels.SentStatus) uint64 {
	jww.WARN.Printf("UpdateFromMessageID is unimplemented in the CLI event model!")
	return 0
}

func (m *eventModel) GetMessage(cryptoChannel.MessageID) (channels.ModelMessage, error) {
	jww.WARN.Printf("GetMessage is unimplemented in the CLI event model!")
	return channels.ModelMessage{}, nil
}

func (m *eventModel) DeleteMessage(cryptoChannel.MessageID) error {
	jww.WARN.Printf("DeleteMessage is unimplemented in the CLI event model!")
	return nil
}

func init() {
	channelsCmd.Flags().String(channelsNameFlag, "ChannelName",
		"The name of the new channel to create.")
	bindFlagHelper(channelsNameFlag, channelsCmd)

	channelsCmd.Flags().String(channelsChanIdentityPathFlag, "",
		"The file path for the channel identity to be written to.")
	bindFlagHelper(channelsChanIdentityPathFlag, channelsCmd)

	channelsCmd.Flags().String(channelsChanPathFlag, "",
		"The file path for the channel information to be written to.")
	bindFlagHelper(channelsChanPathFlag, channelsCmd)

	channelsCmd.Flags().String(channelsDescriptionFlag, "Channel Description",
		"The description for the channel which will be created.")
	bindFlagHelper(channelsDescriptionFlag, channelsCmd)

	channelsCmd.Flags().String(channelsKeyPathFlag, "",
		"The file path for the channel identity's key to be written to.")
	bindFlagHelper(channelsKeyPathFlag, channelsCmd)

	channelsCmd.Flags().Bool(channelsJoinFlag, false,
		"Determines if the channel created from the 'newChannel' or loaded " +
		"from 'channelPath' flag will be joined.")
	bindFlagHelper(channelsJoinFlag, channelsCmd)

	channelsCmd.Flags().Bool(channelsLeaveFlag, false,
		"Determines if the channel created from the 'newChannel' or loaded " +
		"from 'channelPath' flag will be left.")
	bindFlagHelper(channelsLeaveFlag, channelsCmd)

	channelsCmd.Flags().Bool(channelsNewFlag, false,
		"Determines if a new channel will be constructed.")
	bindFlagHelper(channelsNewFlag, channelsCmd)

	channelsCmd.Flags().Bool(channelsSendFlag, false,
		"Determines if a message will be sent to the channel.")
	bindFlagHelper(channelsSendFlag, channelsCmd)

	rootCmd.AddCommand(channelsCmd)
}
