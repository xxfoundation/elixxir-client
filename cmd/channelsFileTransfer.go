////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"

	"gitlab.com/elixxir/client/v4/channels"
	channelsFT "gitlab.com/elixxir/client/v4/channelsFileTransfer"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/utils"
)

// connectionCmd handles the operation of connection operations within the CLI.
var channelsFileTransferCmd = &cobra.Command{
	Use:   "channelsFileTransfer",
	Short: "Runs clients using the channels and file transfer API.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Create client
		cmixParams, e2eParams := initParams()
		authCbs := makeAuthCallbacks(
			viper.GetBool(unsafeChannelCreationFlag), e2eParams)
		user := initE2e(cmixParams, e2eParams, authCbs)

		// Print user's reception ID
		jww.INFO.Printf("User: %s", user.GetReceptionIdentity().ID)

		// Wait for user to be connected to network
		err := user.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("[FT] %+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		user.GetCmix().AddHealthCallback(func(c bool) { connected <- c })
		waitUntilConnected(connected)

		// After connection, wait until registered with at least 85% of nodes
		for numReg, total := 1, 100; numReg < (total*3)/4; {
			time.Sleep(1 * time.Second)

			numReg, total, err = user.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf("Node registration status error: %+v", err)
			}

			jww.INFO.Printf("Registering with nodes (%d/%d)...", numReg, total)
		}

		// Load or make channel private identity
		channelsChanIdentityPath := viper.GetString(channelsChanIdentityPathFlag)
		rng := user.GetRng().GetStream()
		chanID, err := getOrMakeChannelIdentity(channelsChanIdentityPath, rng)
		rng.Close()
		if err != nil {
			jww.FATAL.Panicf("[FT] Failed to get or make channel private "+
				"identity %q: %+v", channelsChanIdentityPath, err)
		}

		// Create event model that will contain the channels event model and the
		// file transfer event model
		em := newFtEventModel()

		// Construct file transfer wrapper
		p := channelsFT.DefaultParams()
		if viper.IsSet(channelsFtMaxThroughputFlag) {
			p.MaxThroughput = viper.GetInt(channelsFtMaxThroughputFlag)
		}
		var extensions = make([]channels.ExtensionBuilder, 1)
		em.FileTransfer, extensions[0], err = channelsFT.NewWrapper(user, p)
		if err != nil {
			jww.FATAL.Panicf(
				"[FT] Failed to create new file transfer manager: %+v", err)
		}

		// Construct channels manager
		em.eventModel.api, err = channels.NewManager(chanID,
			user.GetStorage().GetKV(), user.GetCmix(), user.GetRng(), em,
			extensions, user.AddService)
		if err != nil {
			jww.FATAL.Panicf("[FT] Failed to create channels manager: %+v", err)
		}

		err = user.Cmix.AddService(em.FileTransfer.StartProcesses)
		if err != nil {
			jww.FATAL.Panicf(
				"[FT] Failed to register file transfer service: %+v", err)
		}

		// Create or join channel
		chanPath := viper.GetString(channelsChanPathFlag)
		channel, err := em.createOrJoinChannel(chanPath,
			viper.GetBool(channelsNewFlag), viper.GetBool(channelsJoinFlag),
			user)
		if err != nil {
			jww.FATAL.Panicf("[FT] Failed create or join channel: %+v", err)
		}

		// Start thread that receives new file downloads and prints them to log
		receiveDone := make(chan struct{})
		outputPath := viper.GetString(channelsFtOutputPath)
		go em.receiveFileLink(outputPath, receiveDone)

		// Send message
		if viper.GetBool(channelsSendFlag) {

			// Upload file and wait for it to complete
			filePath := viper.GetString(channelsFtFilePath)
			retry := float32(viper.GetFloat64(channelsFtRetry))
			start := netTime.Now()
			var fid ftCrypto.ID
			select {
			case fid = <-em.uploadChannelFile(filePath, retry):
				jww.INFO.Printf("[FT] Finished uploading file %s.", fid)
				jww.INFO.Printf("[FT] Upload took %s", netTime.Since(start))
			}

			// Get file from event model
			var f channelsFT.ModelFile
			for f.Link == nil {
				select {
				case f = <-em.fileUpdate:
				}
			}

			// Send file to channel
			msgID, rounds, _, err := em.Send(channel.ReceptionID, f.Link,
				filePath, viper.GetString(channelsFtTypeFlag),
				[]byte(viper.GetString(channelsFtPreviewStringFlag)),
				channels.ValidForever, xxdk.GetDefaultCMixParams())
			if err != nil {
				jww.FATAL.Panicf("[FT] Failed to send file %s to channel %s: %+v",
					fid, channel.ReceptionID, err)
			} else {

				jww.INFO.Printf("[FT] Sent file %s to channel %s (ID %s) with "+
					"message ID %s on round %d",
					fid, channel.Name, channel.ReceptionID, msgID, rounds.ID)
			}
		}

		// Wait until either the file finishes sending or the file finishes
		// being received, stop the receiving thread, and exit
		select {
		case <-receiveDone:
			jww.INFO.Printf("[FT] Finished downloading file. Closing test.")
		}

		// Stop network follower
		if err = user.StopNetworkFollower(); err != nil {
			jww.WARN.Printf("[FT] Failed to stop network follower: %+v", err)
		}

		jww.INFO.Printf("[FT] Completed execution.")
	},
}

// getOrMakeChannelIdentity loads the channel private identity from the path if
// it exists. Otherwise, it creates a new one and saves it to the supplied path.
func getOrMakeChannelIdentity(path string, rng io.Reader) (
	cryptoChannel.PrivateIdentity, error) {
	var privID cryptoChannel.PrivateIdentity
	// Load channel identity from path if given from path
	data, err := utils.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Generate channel identity if extant one does not exist
			if privID, err = cryptoChannel.GenerateIdentity(rng); err != nil {
				return cryptoChannel.PrivateIdentity{}, errors.Errorf(
					"failed to generate identity: %+v", err)
			}
			if err = utils.WriteFileDef(path, privID.Marshal()); err != nil {
				return cryptoChannel.PrivateIdentity{}, errors.Errorf(
					"failed to write to file: %+v", err)
			}
			return privID, nil
		} else {
			return cryptoChannel.PrivateIdentity{}, errors.Errorf(
				"failed to read from file %s: %+v", path, err)
		}
	}

	// Unmarshal the private identity
	if privID, err = cryptoChannel.UnmarshalPrivateIdentity(data); err != nil {
		return cryptoChannel.PrivateIdentity{},
			errors.Errorf("failed to unmarshal channel identity: %+v", err)
	}

	return privID, nil
}

// createOrJoinChannel creates or joins a channel depending on the flags set.
func (em *ftEventModel) createOrJoinChannel(chanPath string, newChannel,
	joinChannel bool, user *xxdk.E2e) (*cryptoBroadcast.Channel, error) {

	var channel *cryptoBroadcast.Channel
	var err error

	if newChannel {
		// Create new channel
		if channel, err = createNewChannel(chanPath, user); err != nil {
			return nil, errors.Errorf("failed to create new channel: %+v", err)
		}
	} else {
		// Load channel
		marshalledChan, err2 := utils.ReadFile(chanPath)
		if err2 != nil {
			return nil,
				errors.Errorf("failed to read channel from file: %+v", err2)
		}

		channel, err = cryptoBroadcast.UnmarshalChannel(marshalledChan)
		if err != nil {
			return nil, errors.Errorf("failed to unmarshal channel: %+v", err)
		}
	}

	// Only join if we created it or are told to join
	if newChannel || joinChannel {
		// Join channel
		if err = em.eventModel.api.JoinChannel(channel); err != nil {
			return nil, errors.Errorf("failed to join channel: %+v", err)
		}
		fmt.Printf("Successfully joined channel %q\n", channel.Name)
	}

	return channel, nil
}

// uploadChannelFile uploads the file.
func (em *ftEventModel) uploadChannelFile(
	filePath string, retry float32) chan ftCrypto.ID {
	done := make(chan ftCrypto.ID)

	// Get file from path
	fileData, err := utils.ReadFile(filePath)
	if err != nil {
		jww.FATAL.Panicf("[FT] Failed to read file %q: %+v", filePath, err)
	}

	jww.INFO.Printf("[FT] Uploading file %q {size: %d, retry: %f, path: %q}",
		fileData[:32], len(fileData), retry, filePath)

	uploadStart := netTime.Now()
	uploadCB := func(completed bool, sent, received, total uint16,
		st channelsFT.SentTransfer, _ channelsFT.FilePartTracker, err error) {
		jww.INFO.Printf("[FT] Upload progress for %q {completed: %t, "+
			"sent: %d, received: %d, total: %d, err: %v}",
			st.GetFileID(), completed, sent, received, total, err)
		if (sent == 0 && received == 0) || completed || err != nil {
			fmt.Printf("Upload progress for %q {completed: %t, sent: %d, "+
				"received: %d, total: %d, err: %v}\n",
				st.GetFileID(), completed, sent, received, total, err)
		}

		if completed {
			sendTime := netTime.Since(uploadStart)
			fileSizeKb := float32(len(fileData)) * .001
			speed := fileSizeKb * float32(time.Second) / (float32(sendTime))
			jww.INFO.Printf("[FT] Completed sending file %s in %s "+
				"(%.2f kb @ %.2f kb/s).",
				st.GetFileID(), sendTime, fileSizeKb, speed)
			fmt.Printf("Completed sending file.\n")
			done <- st.GetFileID()
		} else if err != nil {
			jww.ERROR.Printf("[FT] Failed sending file %q in %s: %+v",
				st.GetFileID(), netTime.Since(uploadStart), err)
			fmt.Printf("Failed sending file: %+v\n", err)
			done <- st.GetFileID()
		}
	}

	fid, err := em.Upload(fileData, retry, uploadCB, callbackPeriod)
	if err != nil {
		jww.FATAL.Panicf("[FT] Failed to upload file: %+v", err)
	}

	jww.INFO.Printf("[FT] Uploading new file %s {size: %d, retry: %f}",
		fid, len(fileData), retry)

	return done
}

// receiveFileLink waits on the channel from the event model for a new message
// containing a file link. The file is downloaded and the done channel is
// triggered on completion.
func (em *ftEventModel) receiveFileLink(outputPath string, done chan struct{}) {
	jww.INFO.Print(
		"[FT] Starting thread waiting to receive file links to download.")
	for {
		select {
		case mm := <-em.newMessage:
			var fi channelsFT.FileInfo
			err := json.Unmarshal(mm.Content, &fi)
			if err != nil {
				jww.FATAL.Panicf(
					"[FT] Failed to unmarshal contents into file info: %+v", err)
			}

			jww.INFO.Printf("[FT] Received new file link for file %q (ID %s) "+
				"of type %q from %X of size %d bytes with preview: %q",
				fi.Name, fi.FileID, fi.Type, mm.PubKey, fi.Size, fi.Preview)
			fmt.Printf("Received new file link for file %q (ID %s) of size %d "+
				"bytes with preview: %q\n",
				fi.Name, fi.FileID, fi.Size, fi.Preview)

			var downloadStart time.Time
			progressCB := func(completed bool, received, total uint16,
				rt channelsFT.ReceivedTransfer, _ channelsFT.FilePartTracker,
				err error) {
				jww.INFO.Printf("[FT] Download progress for file %s "+
					"{completed: %t, received: %d, total: %d, err: %v}",
					rt.GetFileID(), completed, received, total, err)

				if received == 0 || completed || err != nil {
					fmt.Printf("Download progress for %s "+
						"{completed: %t, received: %d, total: %d, err: %v}\n",
						rt.GetFileID(), completed, received, total, err)
				}

				if completed {
					// Get file from event model
					f, err2 := em.GetFile(rt.GetFileID())
					if err2 != nil && !channels.CheckNoMessageErr(err) {
						jww.FATAL.Panicf("[FT] Failed to get file %s: %+v",
							rt.GetFileID(), err2)
					} else if channels.CheckNoMessageErr(err) || f.Data == nil {
						for f.Data == nil {
							select {
							case f = <-em.fileUpdate:
							}
						}
					}

					dur := netTime.Since(downloadStart)
					fileSizeKb := float32(len(f.Data)) * .001
					speed := fileSizeKb * float32(time.Second) / (float32(dur))
					jww.INFO.Printf("[FT] Completed downloading file %s in %s "+
						"(%.2f kb @ %.2f kb/s).",
						f.ID, dur, fileSizeKb, speed)
					if err = utils.WriteFileDef(outputPath, f.Data); err != nil {
						jww.FATAL.Panicf(
							"[FT] Failed to save downloaded file: %+v", err)
					}
					fmt.Printf("Completed receiving file. Saved to output path.\n")
					done <- struct{}{}
				} else if err != nil {
					jww.INFO.Printf("[FT] Failed receiving file %s in %s.",
						rt.GetFileID(), netTime.Since(downloadStart))
					fmt.Printf("Failed sending file: %+v\n", err)
					done <- struct{}{}
				}
			}

			downloadStart = netTime.Now()
			_, err = em.FileTransfer.Download(mm.Content, progressCB, callbackPeriod)
			if err != nil {
				jww.FATAL.Panicf("[FT] Failed to initiate download: %+v", err)
			}
		}
	}
}

// makeChannelReceptionHandler is a helper function which will register with the
// channels.Manager a reception callback for the given message type.
func (em *ftEventModel) makeChannelReceptionHandler(receiveMessage chan receivedMessage) error {
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
	return em.api.RegisterReceiveHandler(channels.FileTransfer,
		channels.NewReceiveMessageHandler("", cb, true, true, true))
}

// ftEventModel is the CLI implementation of the channelsFileTransfer.EventModel
// interface.
type ftEventModel struct {
	channelsFT.FileTransfer
	eventModel

	files      map[ftCrypto.ID]channelsFT.ModelFile
	fileUpdate chan channelsFT.ModelFile
	newMessage chan channels.ModelMessage
	mux        sync.Mutex
}

func newFtEventModel() *ftEventModel {
	return &ftEventModel{
		files:      make(map[ftCrypto.ID]channelsFT.ModelFile),
		fileUpdate: make(chan channelsFT.ModelFile, 100),
		newMessage: make(chan channels.ModelMessage, 100),
	}
}

func (em *ftEventModel) ReceiveFile(fileID ftCrypto.ID, fileLink, fileData []byte,
	timestamp time.Time, status channelsFT.Status) error {
	jww.INFO.Printf("[FT] Received file %s at %s with status %s."+
		"\nlink: %q\ndata: %q", fileID, timestamp, status, fileLink, fileData)
	fmt.Printf("Received file %s with status %s\n", fileID, status)

	em.mux.Lock()
	defer em.mux.Unlock()
	em.files[fileID] = channelsFT.ModelFile{
		ID:        fileID,
		Link:      fileLink,
		Data:      fileData,
		Timestamp: timestamp,
		Status:    status,
	}
	return nil
}
func (em *ftEventModel) UpdateFile(fileID ftCrypto.ID, fileLink, fileData []byte,
	timestamp *time.Time, status *channelsFT.Status) error {
	jww.INFO.Printf("[FT] Updating file %s at %s with status %s."+
		"\nlink: %q\ndata: %q", fileID, timestamp, status, fileLink, fileData)

	em.mux.Lock()
	f, exists := em.files[fileID]
	if !exists {
		return channels.NoMessageErr
	}

	if fileLink != nil {
		f.Link = fileLink
	}
	if fileData != nil {
		f.Data = fileData
	}
	if timestamp != nil {
		f.Timestamp = *timestamp
	}
	if status != nil {
		f.Status = *status
	}
	em.files[fileID] = f
	em.mux.Unlock()

	em.fileUpdate <- f

	return nil
}

func (em *ftEventModel) GetFile(fileID ftCrypto.ID) (channelsFT.ModelFile, error) {
	em.mux.Lock()
	defer em.mux.Unlock()
	f, exists := em.files[fileID]
	if !exists {
		return channelsFT.ModelFile{}, channels.NoMessageErr
	}
	return f, nil
}

func (em *ftEventModel) DeleteFile(fileID ftCrypto.ID) error {
	jww.INFO.Printf("[FT] Deleting file %s", fileID)
	em.mux.Lock()
	defer em.mux.Unlock()
	if _, exists := em.files[fileID]; !exists {
		return channels.NoMessageErr
	}
	delete(em.files, fileID)
	return nil
}

func (em *ftEventModel) ReceiveMessage(channelID *id.ID, messageID message.ID,
	nickname, text string, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus, hidden bool) uint64 {
	jww.INFO.Printf(
		"[FT] Received message %s on channel %s", messageID, channelID)
	em.newMessage <- channels.ModelMessage{
		Nickname:       nickname,
		MessageID:      messageID,
		ChannelID:      channelID,
		Timestamp:      timestamp,
		Lease:          lease,
		Status:         status,
		Hidden:         hidden,
		Content:        []byte(text),
		Type:           messageType,
		Round:          round.ID,
		PubKey:         pubKey,
		CodesetVersion: codeset,
		DmToken:        dmToken,
	}
	return 0
}

func init() {

	channelsFileTransferCmd.Flags().String(channelsChanIdentityPathFlag, "",
		"The file path for the channel identity to be written to.")
	bindFlagHelper(channelsChanIdentityPathFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().String(channelsChanPathFlag, "",
		"The file path for the channel information to be written to.")
	bindFlagHelper(channelsChanPathFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().String(channelsNameFlag, "ChannelName",
		"The name of the new channel to create.")
	bindFlagHelper(channelsNameFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().String(channelsDescriptionFlag, "Channel Description",
		"The description for the channel which will be created.")
	bindFlagHelper(channelsDescriptionFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().String(channelsFtTypeFlag, "txt",
		"8-byte file type.")
	bindFlagHelper(channelsFtTypeFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().String(channelsFtPreviewStringFlag, "",
		"File preview data.")
	bindFlagHelper(channelsFtPreviewStringFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().String(channelsKeyPathFlag, "",
		"The file path for the channel identity's key to be written to.")
	bindFlagHelper(channelsKeyPathFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().Bool(channelsJoinFlag, false,
		"Determines if the channel created from the 'newChannel' or loaded "+
			"from 'channelPath' flag will be joined.")
	bindFlagHelper(channelsJoinFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().Bool(channelsNewFlag, false,
		"Determines if a new channel will be constructed.")
	bindFlagHelper(channelsNewFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().Bool(channelsSendFlag, false,
		"Determines if a message will be sent to the channel.")
	bindFlagHelper(channelsSendFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().String(channelsFtFilePath, "",
		"The path to the file to send. Also used as the file name.")
	bindFlagHelper(channelsFtFilePath, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().Int(channelsFtMaxThroughputFlag, 1000,
		"Maximum data transfer speed to send file parts (in bytes per second)")
	bindFlagHelper(channelsFtMaxThroughputFlag, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().Float64(channelsFtRetry, 0.5, "Retry rate.")
	bindFlagHelper(channelsFtRetry, channelsFileTransferCmd)

	channelsFileTransferCmd.Flags().String(channelsFtOutputPath, "",
		"The path to save the received file")
	bindFlagHelper(channelsFtOutputPath, channelsFileTransferCmd)

	rootCmd.AddCommand(channelsFileTransferCmd)
}
