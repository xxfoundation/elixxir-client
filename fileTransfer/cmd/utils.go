package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	ft "gitlab.com/elixxir/client/fileTransfer"
	ftE2e "gitlab.com/elixxir/client/fileTransfer/e2e"
	"gitlab.com/elixxir/client/xxdk"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/utils"
	"time"
)

const callbackPeriod = 25 * time.Millisecond

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
func initFileTransferManager(messenger *xxdk.E2e, maxThroughput int) (
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
	manager, err := ft.NewManager(p,
		messenger.GetReceptionIdentity().ID,
		messenger.GetCmix(),
		messenger.GetStorage(),
		messenger.GetRng())
	if err != nil {
		jww.FATAL.Panicf(
			"[FT] Failed to create new file transfer manager: %+v", err)
	}

	// Start the file transfer sending and receiving threads
	err = messenger.AddService(manager.StartProcesses)
	if err != nil {
		jww.FATAL.Panicf("[FT] Failed to start file transfer threads: %+v", err)
	}

	e2eParams := ftE2e.DefaultParams()
	e2eFt, err := ftE2e.NewWrapper(receiveCB, e2eParams, manager,
		messenger.GetReceptionIdentity().ID, messenger.GetE2E(), messenger.GetCmix())
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
	recipient := cmdUtils.GetContactFromFile(recipientContactPath)

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
