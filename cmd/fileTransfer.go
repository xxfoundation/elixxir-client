////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"os"
	"time"

	"gitlab.com/elixxir/client/v4/xxdk"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	ft "gitlab.com/elixxir/client/v4/fileTransfer"
	ftE2e "gitlab.com/elixxir/client/v4/fileTransfer/e2e"
	"gitlab.com/elixxir/crypto/contact"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/utils"
)

const callbackPeriod = 25 * time.Millisecond

// ftCmd starts the file transfer manager and allows the sending and receiving
// of files.
var ftCmd = &cobra.Command{
	Use:   "fileTransfer",
	Short: "Send and receive file for cMix client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmixParams, e2eParams := initParams()
		authCbs := makeAuthCallbacks(
			viper.GetBool(unsafeChannelCreationFlag), e2eParams)
		user := initE2e(cmixParams, e2eParams, authCbs)

		// Print user's reception ID and save contact file
		identity := user.GetReceptionIdentity()
		jww.INFO.Printf("User: %s", identity.ID)
		writeContact(identity.GetContact())

		// Start the network follower
		err := user.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("Failed to start the network follower: %+v", err)
		}

		// Initialize the file transfer manager
		maxThroughput := viper.GetInt(fileMaxThroughputFlag)
		m, receiveChan := initFileTransferManager(user, maxThroughput)

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

		// Start thread that receives new file transfers and prints them to log
		receiveQuit := make(chan struct{})
		receiveDone := make(chan struct{})
		go receiveNewFileTransfers(receiveChan, receiveDone, receiveQuit, m)

		// If set, send the file to the recipient
		sendDone := make(chan struct{})
		if viper.IsSet(fileSendFlag) {
			recipientContactPath := viper.GetString(fileSendFlag)
			filePath := viper.GetString(filePathFlag)
			fileType := viper.GetString(fileTypeFlag)
			filePreviewPath := viper.GetString(filePreviewPathFlag)
			filePreviewString := viper.GetString(filePreviewStringFlag)
			retry := float32(viper.GetFloat64(fileRetry))

			sendFile(filePath, fileType, filePreviewPath, filePreviewString,
				recipientContactPath, retry, m, sendDone)
		}

		// Wait until either the file finishes sending or the file finishes
		// being received, stop the receiving thread, and exit
		select {
		case <-sendDone:
			jww.INFO.Printf("[FT] Finished sending file. Stopping threads " +
				"and network follower.")
		case <-receiveDone:
			jww.INFO.Printf("[FT] Finished receiving file. Stopping threads " +
				"and network follower.")
		}

		// Stop reception thread
		receiveQuit <- struct{}{}

		// Stop network follower
		err = user.StopNetworkFollower()
		if err != nil {
			jww.WARN.Printf("[FT] Failed to stop network follower: %+v", err)
		}

		jww.INFO.Print("[FT] File transfer finished stopping threads and " +
			"network follower.")
	},
}

// receivedFtResults is used to return received new file transfer results on a
// channel from a callback.
type receivedFtResults struct {
	tid      *ftCrypto.TransferID
	fileName string
	fileType string
	sender   *id.ID
	size     uint32
	preview  []byte
}

// initFileTransferManager creates a new file transfer manager with a new
// reception callback. Returns the file transfer manager and the channel that
// will be triggered when the callback is called.
func initFileTransferManager(user *xxdk.E2e, maxThroughput int) (
	*ftE2e.Wrapper, chan receivedFtResults) {

	// Create interfaces.ReceiveCallback that returns the results on a channel
	receiveChan := make(chan receivedFtResults, 100)
	receiveCB := func(tid *ftCrypto.TransferID, fileName, fileType string,
		sender *id.ID, size uint32, preview []byte) {
		receiveChan <- receivedFtResults{
			tid, fileName, fileType, sender, size, preview}
	}

	// Create new parameters
	p := ft.DefaultParams()
	p.SendTimeout = 10 * time.Second
	if maxThroughput != 0 {
		p.MaxThroughput = maxThroughput
	}

	// Create new manager
	manager, err := ft.NewManager(p, user)
	if err != nil {
		jww.FATAL.Panicf(
			"[FT] Failed to create new file transfer manager: %+v", err)
	}

	// Start the file transfer sending and receiving threads
	err = user.AddService(manager.StartProcesses)
	if err != nil {
		jww.FATAL.Panicf("[FT] Failed to start file transfer threads: %+v", err)
	}

	e2eParams := ftE2e.DefaultParams()
	e2eFt, err := ftE2e.NewWrapper(receiveCB, e2eParams, manager, user)
	if err != nil {
		jww.FATAL.Panicf(
			"[FT] Failed to create new e2e file transfer wrapper: %+v", err)
	}

	return e2eFt, receiveChan
}

// sendFile sends the file to the recipient and prints the progress.
func sendFile(filePath, fileType, filePreviewPath, filePreviewString,
	recipientContactPath string, retry float32, m *ftE2e.Wrapper,
	done chan struct{}) {

	// Get file from path
	fileData, err := utils.ReadFile(filePath)
	if err != nil {
		jww.FATAL.Panicf("[FT] Failed to read file %q: %+v", filePath, err)
	}

	// Get file preview from path
	filePreviewData := []byte(filePreviewString)
	if filePreviewPath != "" {
		filePreviewData, err = utils.ReadFile(filePreviewPath)
		if err != nil {
			jww.FATAL.Panicf("[FT] Failed to read file preview %q: %+v",
				filePreviewPath, err)
		}
	}

	// Truncate file path if it is too long to be a file name
	fileName := filePath
	if len(fileName) > ft.FileNameMaxLen {
		fileName = fileName[:ft.FileNameMaxLen]
	}

	// Get recipient contact from file
	recipient := getContactFromFile(recipientContactPath)

	jww.INFO.Printf("[FT] Going to start sending file %q to %s {type: %q, "+
		"size: %d, retry: %f, path: %q, previewPath: %q, preview: %q}",
		fileName, recipient.ID, fileType, len(fileData), retry, filePath,
		filePreviewPath, filePreviewData)

	var sendStart time.Time

	// Create sent progress callback that prints the results
	progressCB := func(completed bool, arrived, total uint16,
		st ft.SentTransfer, _ ft.FilePartTracker, err error) {
		jww.INFO.Printf("[FT] Sent progress callback for %q "+
			"{completed: %t, arrived: %d, total: %d, err: %v}",
			fileName, completed, arrived, total, err)
		if arrived == 0 || (arrived == total) || completed ||
			err != nil {
			fmt.Printf("Sent progress callback for %q "+
				"{completed: %t, arrived: %d, total: %d, err: %v}\n",
				fileName, completed, arrived, total, err)
		}

		if completed {
			fileSize := len(fileData)
			sendTime := netTime.Since(sendStart)
			fileSizeKb := float32(fileSize) * .001
			speed := fileSizeKb * float32(time.Second) / (float32(sendTime))
			jww.INFO.Printf("[FT] Completed sending file %q in %s (%.2f kb @ %.2f kb/s).",
				fileName, sendTime, fileSizeKb, speed)
			fmt.Printf("Completed sending file.\n")
			done <- struct{}{}
		} else if err != nil {
			jww.ERROR.Printf("[FT] Failed sending file %q in %s: %+v",
				fileName, netTime.Since(sendStart), err)
			fmt.Printf("Failed sending file: %+v\n", err)
			done <- struct{}{}
		}
	}

	sendStart = netTime.Now()

	// Send the file
	tid, err := m.Send(recipient.ID, fileName, fileType, fileData, retry,
		filePreviewData, progressCB, callbackPeriod)
	if err != nil {
		jww.FATAL.Panicf("[FT] Failed to send file %q to %s: %+v",
			fileName, recipient.ID, err)
	}

	jww.INFO.Printf("[FT] Sending new file transfer %s to %s {name: %s, "+
		"type: %q, size: %d, retry: %f}",
		tid, recipient, fileName, fileType, len(fileData), retry)
}

// receiveNewFileTransfers waits to receive new file transfers and prints its
// information to the log.
func receiveNewFileTransfers(receive chan receivedFtResults, done,
	quit chan struct{}, m *ftE2e.Wrapper) {
	jww.INFO.Print("[FT] Starting thread waiting to receive NewFileTransfer " +
		"E2E message.")
	for {
		select {
		case <-quit:
			jww.INFO.Print("[FT] Quitting thread waiting for NewFileTransfer " +
				"E2E message.")
			return
		case r := <-receive:
			receiveStart := netTime.Now()
			jww.INFO.Printf("[FT] Received new file %q transfer %s of type "+
				"%q from %s of size %d bytes with preview: %q",
				r.fileName, r.tid, r.fileType, r.sender, r.size, r.preview)
			fmt.Printf("Received new file transfer %q of size %d "+
				"bytes with preview: %q\n", r.fileName, r.size, r.preview)

			cb := newReceiveProgressCB(r.tid, r.fileName, done, receiveStart, m)
			err := m.RegisterReceivedProgressCallback(r.tid, cb, callbackPeriod)
			if err != nil {
				jww.FATAL.Panicf("[FT] Failed to register new receive "+
					"progress callback for transfer %s: %+v", r.tid, err)
			}
		}
	}
}

// newReceiveProgressCB creates a new reception progress callback that prints
// the results to the log.
func newReceiveProgressCB(tid *ftCrypto.TransferID, fileName string,
	done chan struct{}, receiveStart time.Time,
	m *ftE2e.Wrapper) ft.ReceivedProgressCallback {
	return func(completed bool, received, total uint16,
		rt ft.ReceivedTransfer, t ft.FilePartTracker, err error) {
		jww.INFO.Printf("[FT] Received progress callback for transfer %s "+
			"{completed: %t, received: %d, total: %d, err: %v}",
			tid, completed, received, total, err)

		if received == total || completed || err != nil {
			fmt.Printf("Received progress callback "+
				"{completed: %t, received: %d, total: %d, err: %v}\n",
				completed, received, total, err)
		}

		if completed {
			receivedFile, err2 := m.Receive(tid)
			if err2 != nil {
				jww.FATAL.Panicf(
					"[FT] Failed to receive file %s: %+v", tid, err2)
			}
			jww.INFO.Printf("[FT] Completed receiving file %q in %s.",
				fileName, netTime.Since(receiveStart))
			fmt.Printf("Completed receiving file:\n%s\n", receivedFile)
			done <- struct{}{}
		} else if err != nil {
			jww.INFO.Printf("[FT] Failed receiving file %q in %s.",
				fileName, netTime.Since(receiveStart))
			fmt.Printf("Failed sending file: %+v\n", err)
			done <- struct{}{}
		}
	}
}

// getContactFromFile loads the contact from the given file path.
func getContactFromFile(path string) contact.Contact {
	data, err := os.ReadFile(path)
	jww.INFO.Printf("Read in contact file of size %d bytes", len(data))
	if err != nil {
		jww.FATAL.Panicf("Failed to read contact file: %+v", err)
	}

	c, err := contact.Unmarshal(data)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal contact: %+v", err)
	}

	return c
}

////////////////////////////////////////////////////////////////////////////////
// Command Line Flags                                                         //
////////////////////////////////////////////////////////////////////////////////

// init initializes commands and flags for Cobra.
func init() {
	ftCmd.Flags().String(fileSendFlag, "",
		"Sends a file to a recipient with the contact file at this path.")
	bindFlagHelper(fileSendFlag, ftCmd)

	ftCmd.Flags().String(filePathFlag, "",
		"The path to the file to send. Also used as the file name.")
	bindFlagHelper(filePathFlag, ftCmd)

	ftCmd.Flags().String(fileTypeFlag, "txt",
		"8-byte file type.")
	bindFlagHelper(fileTypeFlag, ftCmd)

	ftCmd.Flags().String(filePreviewPathFlag, "",
		"The path to the file preview to send. Set either this flag or "+
			"filePreviewString.")
	bindFlagHelper(filePreviewPathFlag, ftCmd)

	ftCmd.Flags().String(filePreviewStringFlag, "",
		"File preview data. Set either this flag or filePreviewPath.")
	bindFlagHelper(filePreviewStringFlag, ftCmd)

	ftCmd.Flags().Int(fileMaxThroughputFlag, 1000,
		"Maximum data transfer speed to send file parts (in bytes per second)")
	bindFlagHelper(fileMaxThroughputFlag, ftCmd)

	ftCmd.Flags().Float64(fileRetry, 0.5,
		"Retry rate.")
	bindFlagHelper(fileRetry, ftCmd)

	rootCmd.AddCommand(ftCmd)
}
