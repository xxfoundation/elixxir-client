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
	"os"
	"time"

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
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
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
		cbs := &channelCbs{}
		chanManager, err := channels.NewManagerBuilder(channelIdentity,
			user.GetStorage().GetKV(), user.GetCmix(), user.GetRng(),
			mockEventModelBuilder, nil, user.AddService, nil, cbs)
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
			keyPath := viper.GetString(channelsKeyPathFlag)
			name := viper.GetString(channelsNameFlag)
			desc := viper.GetString(channelsDescriptionFlag)
			channel, err = createNewChannel(chanPath, keyPath, name, desc, user)
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
		receiveMessage := make(chan receivedMessage)
		err = makeChannelReceptionHandler(integrationChannelMessage,
			chanManager, receiveMessage)
		if err != nil {
			jww.FATAL.Panicf("[%s] Failed to create reception handler for "+
				"message type %s: %+v",
				channelsPrintHeader, integrationChannelMessage, err)
		}

		// Send message
		sendDone := make(chan error)
		if viper.GetBool(channelsSendFlag) {
			go func() {
				msgBody := []byte(viper.GetString(messageFlag))
				sendDone <- sendMessageToChannel(chanManager, channel, msgBody)
			}()
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

		// Wait for reception. There should be 4 operations for the
		// integration test:
		waitTime := viper.GetDuration(waitTimeoutFlag) * time.Second
		maxReceiveCnt := viper.GetInt(receiveCountFlag)
		receiveCnt := 0
		for done := false; viper.IsSet(channelsSendFlag) && !done; {
			if maxReceiveCnt != 0 && receiveCnt >= maxReceiveCnt {
				done = true
				continue
			}

			select {
			case m := <-receiveMessage:
				channelID, content := m.chanId, m.content
				channelReceivedMessage, err := chanManager.GetChannel(channelID)
				if err != nil {
					jww.FATAL.Panicf("[%s] Failed to find channel for %s: %+v",
						channelsPrintHeader, channelID, err)
				}
				jww.INFO.Printf("[%s] Received message (%s) from %s",
					channelsPrintHeader, content, channelReceivedMessage.Name)
				fmt.Printf("Received from %s this message: %s\n",
					channelReceivedMessage.Name, content)

				receiveCnt++
			case <-time.After(waitTime):
				done = true
			}
		}

		if maxReceiveCnt == 0 {
			maxReceiveCnt = receiveCnt
		}
		fmt.Printf("Received %d/%d messages\n", receiveCnt,
			maxReceiveCnt)

		// Ensure send is completed before looking closing. Note that sending
		// to yourself does not go through cMix, so sending does not block the
		// above loop.
		for done := false; viper.IsSet(channelsSendFlag) && !done; {
			select {
			case err = <-sendDone:
				if err != nil {
					jww.FATAL.Panicf("[%s] Failed to send message: %+v",
						channelsPrintHeader, err)
				}
				done = true
			case <-time.After(waitTime):
				done = true
			}

		}

		// Stop network follower
		err = user.StopNetworkFollower()
		if err != nil {
			jww.WARN.Printf("[CHAN] Failed to stop network follower: %+v", err)
		}

		jww.INFO.Printf("[CHAN] Completed execution...")

	},
}

// createNewChannel creates a new channel with the name and description. If a
// key path is set, then the private key is saved to that path in PEM format;
// otherwise, it is only printed to the log. The marshalled channel is written
// to the chanPath.
//
// This function prints to stdout when a new channel is successfully generated.
func createNewChannel(chanPath, keyPath, name, desc string, user *xxdk.E2e) (
	*cryptoBroadcast.Channel, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	} else if desc == "" {
		return nil, errors.New("description cannot be empty")
	}

	// Create a new  channel
	channel, pk, err := cryptoBroadcast.NewChannel(
		name, desc, cryptoBroadcast.Public, user.GetCmix().GetMaxMessageLength(),
		user.GetRng().GetStream())
	if err != nil {
		return nil, errors.Errorf("failed to create new channel: %+v", err)
	}

	if keyPath != "" {
		err = utils.WriteFile(keyPath, pk.MarshalPem(), os.ModePerm, os.ModeDir)
		if err != nil {
			jww.ERROR.Printf("[%s] Failed to write private key to path %s: %+v",
				channelsPrintHeader, keyPath, err)
		}
	} else {
		jww.INFO.Printf(
			"Private key generated for channel: %+v\n", pk.MarshalPem())
	}
	fmt.Printf("New channel generated\n")

	// Write channel to file
	marshalledChan, err := channel.Marshal()
	if err != nil {
		return nil, errors.Errorf("failed to marshal channel: %+v", err)
	}

	err = utils.WriteFileDef(chanPath, marshalledChan)
	if err != nil {
		return nil, errors.Errorf("failed to write channel to file: %+v", err)
	}

	return channel, nil
}

// sendMessageToChannel is a helper function which will send a message to a
// channel.
func sendMessageToChannel(chanManager channels.Manager,
	channel *cryptoBroadcast.Channel, msgBody []byte) error {
	jww.INFO.Printf("[%s] Sending message (%s) to channel %s",
		channelsPrintHeader, msgBody, channel.Name)
	chanMsgId, round, _, err := chanManager.SendGeneric(
		channel.ReceptionID, integrationChannelMessage, msgBody, 5*time.Second,
		true, cmix.GetDefaultCMIXParams(), nil)
	if err != nil {
		return errors.Errorf("%+v", err)
	}
	jww.INFO.Printf("[%s] Sent message (%s) to channel %s (ID %s) with "+
		"message ID %s on round %d", channelsPrintHeader, msgBody, channel.Name,
		channel.ReceptionID, chanMsgId, round.ID)

	return nil
}

// makeChannelReceptionHandler is a helper function which will register with the
// channels.Manager a reception callback for the given message type.
func makeChannelReceptionHandler(msgType channels.MessageType,
	chanManager channels.Manager, receiveMessage chan receivedMessage) error {
	// Construct receiver callback
	cb := func(channelID *id.ID, _ message.ID, _ channels.MessageType, _ string,
		content, _ []byte, _ ed25519.PublicKey, _ uint32, _ uint8, _,
		_ time.Time, _ time.Duration, _ id.Round, _ rounds.Round,
		_ channels.SentStatus, _, _ bool) uint64 {
		receiveMessage <- receivedMessage{
			chanId:  channelID,
			content: content,
		}
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

// receivedMessage is a structure containing the information for a received
// message. This is passed from the channel's reception callback to
// the main thread which waits to receive receiveCountFlag messages, or
// until waitTimeoutFlag seconds have passed.
type receivedMessage struct {
	chanId  *id.ID
	content []byte
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

func (m *eventModel) ReceiveMessage(_ *id.ID, _ message.ID, _, text string,
	_ ed25519.PublicKey, _ uint32, _ uint8, _ time.Time, _ time.Duration,
	_ rounds.Round, _ channels.MessageType, _ channels.SentStatus, _ bool) uint64 {
	jww.INFO.Printf("[%s] Received message (%s) from channel",
		channelsPrintHeader, text)
	fmt.Printf("Received message (%s) from channel\n", text)
	return 0
}

func (m *eventModel) ReceiveReply(channelID *id.ID, _, _ message.ID, _,
	_ string, _ ed25519.PublicKey, _ uint32, _ uint8, _ time.Time,
	_ time.Duration, _ rounds.Round, _ channels.MessageType,
	_ channels.SentStatus, _ bool) uint64 {
	c, err := m.api.GetChannel(channelID)
	if err != nil {
		jww.FATAL.Panicf("[%s] Failed to get channel with ID %s",
			channelsPrintHeader, channelID)
	}
	fmt.Printf("Received reply for channel %s\n", c.Name)
	return 0
}

func (m *eventModel) ReceiveReaction(channelID *id.ID, _, _ message.ID, _,
	_ string, _ ed25519.PublicKey, _ uint32, _ uint8, _ time.Time,
	_ time.Duration, _ rounds.Round, _ channels.MessageType,
	_ channels.SentStatus, _ bool) uint64 {
	c, err := m.api.GetChannel(channelID)
	if err != nil {
		jww.FATAL.Panicf("[%s] Failed to get channel with ID %s",
			channelsPrintHeader, channelID)
	}
	fmt.Printf("Received reaction for channel %s\n", c.Name)
	return 0
}

func (m *eventModel) UpdateFromUUID(uint64, *message.ID, *time.Time,
	*rounds.Round, *bool, *bool, *channels.SentStatus) error {
	jww.WARN.Printf("UpdateFromUUID is unimplemented in the CLI event model!")
	return nil
}

func (m *eventModel) UpdateFromMessageID(message.ID, *time.Time, *rounds.Round,
	*bool, *bool, *channels.SentStatus) (uint64, error) {
	jww.WARN.Printf("UpdateFromMessageID is unimplemented in the CLI event model!")
	return 0, nil
}

func (m *eventModel) GetMessage(message.ID) (channels.ModelMessage, error) {
	jww.WARN.Printf("GetMessage is unimplemented in the CLI event model!")
	return channels.ModelMessage{}, nil
}

func (m *eventModel) DeleteMessage(message.ID) error {
	jww.WARN.Printf("DeleteMessage is unimplemented in the CLI event model!")
	return nil
}

func (m *eventModel) MuteUser(*id.ID, ed25519.PublicKey, bool) {
	jww.WARN.Printf("MuteUser is unimplemented in the CLI event model!")
}

type channelCbs struct{}

func (c *channelCbs) NicknameUpdate(channelID *id.ID, nickname string,
	exists bool) {
	jww.INFO.Printf("NickNameUpdate(%s, %s, %v)", channelID,
		nickname, exists)
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
		"Determines if the channel created from the 'newChannel' or loaded "+
			"from 'channelPath' flag will be joined.")
	bindFlagHelper(channelsJoinFlag, channelsCmd)

	channelsCmd.Flags().Bool(channelsLeaveFlag, false,
		"Determines if the channel created from the 'newChannel' or loaded "+
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
