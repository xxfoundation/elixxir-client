////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	ftStorage "gitlab.com/elixxir/client/storage/fileTransfer"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const (
	// PreviewMaxSize is the maximum size, in bytes, for a file preview.
	// Currently, it is set to 4 kB.
	PreviewMaxSize = 4_000

	// FileNameMaxLen is the maximum size, in bytes, for a file name. Currently,
	// it is set to 48 bytes.
	FileNameMaxLen = 48

	// FileTypeMaxLen is the maximum size, in bytes, for a file type. Currently,
	// it is set to 8 bytes.
	FileTypeMaxLen = 8

	// FileMaxSize is the maximum file size that can be transferred. Currently,
	// it is set to 4 mB.
	FileMaxSize = 4_000_000

	// minPartsSendPerRound is the minimum number of file parts sent each round.
	minPartsSendPerRound = 1

	// maxPartsSendPerRound is the maximum number of file parts sent each round.
	maxPartsSendPerRound = 11

	// Size of the buffered channel that queues file parts to send
	sendQueueBuffLen = 10_000

	// Size of the buffered channel that reports if the network is healthy
	networkHealthBuffLen = 10_000
)

// Part status constants.
const (
	unsent   = 0
	sent     = 1
	arrived  = 2
	received = 1
)

// Error messages.
const (
	// newManager
	newManagerSentErr     = "failed to load or create new list of sent file transfers: %+v"
	newManagerReceivedErr = "failed to load or create new list of received file transfers: %+v"

	// Manager.Send
	fileNameSizeErr = "length of filename (%d) greater than max allowed length (%d)"
	fileTypeSizeErr = "length of file type (%d) greater than max allowed length (%d)"
	fileSizeErr     = "size of file (%d bytes) greater than max allowed size (%d bytes)"
	previewSizeErr  = "size of preview (%d bytes) greater than max allowed size (%d bytes)"
	getPartSizeErr  = "failed to get file part size: %+v"
	sendInitMsgErr  = "failed to send initial file transfer message: %+v"

	// Manager.Resend
	transferNotFailedErr = "transfer %s has not failed"

	// Manager.CloseSend
	transferInProgressErr = "transfer %s has not completed or failed"
)

// Stoppable and listener values.
const (
	rawMessageBuffSize        = 10_000
	sendStoppableName         = "FileTransferSend"
	newFtStoppableName        = "FileTransferNew"
	newFtListenerName         = "FileTransferNewListener"
	filePartStoppableName     = "FilePart"
	filePartListenerName      = "FilePartListener"
	fileTransferStoppableName = "FileTransfer"
)

// Manager is used to manage the sending and receiving of all file transfers.
type Manager struct {
	// Callback that is called every time a new file transfer is received
	receiveCB interfaces.ReceiveCallback

	// Storage-backed structure for tracking sent file transfers
	sent *ftStorage.SentFileTransfers

	// Storage-backed structure for tracking received file transfers
	received *ftStorage.ReceivedFileTransfers

	// Queue of parts to send
	sendQueue chan queuedPart

	// Maximum data transfer speed in bytes per second
	maxThroughput int

	// Client interfaces
	client          *api.Client
	store           *storage.Session
	swb             interfaces.Switchboard
	net             interfaces.NetworkManager
	healthy         chan bool
	rng             *fastRNG.StreamGenerator
	getRoundResults getRoundResultsFunc
}

// getRoundResultsFunc is a function that matches client.GetRoundResults. It is
// used to pass in an alternative function for testing.
type getRoundResultsFunc func(roundList []id.Round, timeout time.Duration,
	roundCallback api.RoundEventCallback) error

// queuedPart contains the unique information identifying a file part.
type queuedPart struct {
	tid     ftCrypto.TransferID
	partNum uint16
}

// NewManager produces a new empty file transfer Manager. Does not start sending
// and receiving services.
func NewManager(client *api.Client, receiveCB interfaces.ReceiveCallback,
	p Params) (*Manager, error) {
	return newManager(client, client.GetStorage(), client.GetSwitchboard(),
		client.GetNetworkInterface(), client.GetRng(), client.GetRoundResults,
		client.GetStorage().GetKV(), receiveCB, p)
}

// newManager builds the manager from fields explicitly passed in. This function
// is a helper function for NewManager to make it easier to test.
func newManager(client *api.Client, store *storage.Session,
	swb interfaces.Switchboard, net interfaces.NetworkManager,
	rng *fastRNG.StreamGenerator, getRoundResults getRoundResultsFunc,
	kv *versioned.KV, receiveCB interfaces.ReceiveCallback, p Params) (
	*Manager, error) {

	// Create a new list of sent file transfers or load one if it exists in
	// storage
	sent, err := ftStorage.NewOrLoadSentFileTransfers(kv)
	if err != nil {
		return nil, errors.Errorf(newManagerSentErr, err)
	}

	// Create a new list of received file transfers or load one if it exists in
	// storage
	received, err := ftStorage.NewOrLoadReceivedFileTransfers(kv)
	if err != nil {
		return nil, errors.Errorf(newManagerReceivedErr, err)
	}

	return &Manager{
		receiveCB:       receiveCB,
		sent:            sent,
		received:        received,
		sendQueue:       make(chan queuedPart, sendQueueBuffLen),
		maxThroughput:   p.MaxThroughput,
		client:          client,
		store:           store,
		swb:             swb,
		net:             net,
		healthy:         make(chan bool, networkHealthBuffLen),
		rng:             rng,
		getRoundResults: getRoundResults,
	}, nil
}

// StartProcesses starts the processes needed to send and receive file parts. It
// starts three threads that (1) receives the initial NewFileTransfer E2E
// message; (2) receives each file part; and (3) sends file parts. It also
// registers the network health channel.
func (m *Manager) StartProcesses() (stoppable.Stoppable, error) {
	// Create the two reception channels
	newFtChan := make(chan message.Receive, rawMessageBuffSize)
	filePartChan := make(chan message.Receive, rawMessageBuffSize)

	return m.startProcesses(newFtChan, filePartChan)
}

// startProcesses starts the sending and receiving processes with the provided
// channels.
func (m *Manager) startProcesses(newFtChan, filePartChan chan message.Receive) (
	stoppable.Stoppable, error) {

	// Register network health channel that is used by the sending thread to
	// ensure the network is healthy before sending
	m.net.GetHealthTracker().AddChannel(m.healthy)

	// Start the new file transfer message reception thread
	newFtStop := stoppable.NewSingle(newFtStoppableName)
	m.swb.RegisterChannel(newFtListenerName, &id.ID{},
		message.NewFileTransfer, newFtChan)
	go m.receiveNewFileTransfer(newFtChan, newFtStop)

	// Start the file part message reception thread
	filePartStop := stoppable.NewSingle(filePartStoppableName)
	m.swb.RegisterChannel(filePartListenerName, &id.ID{}, message.Raw,
		filePartChan)
	go m.receive(filePartChan, filePartStop)

	// Start the file part sending thread
	sendStop := stoppable.NewSingle(sendStoppableName)
	go m.sendThread(sendStop, getRandomNumParts)

	// Create a multi stoppable
	multiStoppable := stoppable.NewMulti(fileTransferStoppableName)
	multiStoppable.Add(newFtStop)
	multiStoppable.Add(filePartStop)
	multiStoppable.Add(sendStop)

	return multiStoppable, nil
}

// Send starts the sending of a file transfer to the recipient. It sends the
// initial NewFileTransfer E2E message to the recipient to inform them of the
// incoming file parts. It partitions the file, puts it into storage, and queues
// each file for sending. Returns a unique ID identifying the file transfer.
func (m Manager) Send(fileName, fileType string, fileData []byte,
	recipient *id.ID, retry float32, preview []byte,
	progressCB interfaces.SentProgressCallback, period time.Duration) (
	ftCrypto.TransferID, error) {

	// Return an error if the file name is too long
	if len(fileName) > FileNameMaxLen {
		return ftCrypto.TransferID{}, errors.Errorf(
			fileNameSizeErr, len(fileName), FileNameMaxLen)
	}

	// Return an error if the file type is too long
	if len(fileType) > FileTypeMaxLen {
		return ftCrypto.TransferID{}, errors.Errorf(
			fileTypeSizeErr, len(fileType), FileTypeMaxLen)
	}

	// Return an error if the file is too large
	if len(fileData) > FileMaxSize {
		return ftCrypto.TransferID{}, errors.Errorf(
			fileSizeErr, len(fileData), FileMaxSize)
	}

	// Return an error if the preview is too large
	if len(preview) > PreviewMaxSize {
		return ftCrypto.TransferID{}, errors.Errorf(
			previewSizeErr, len(preview), PreviewMaxSize)
	}

	// Generate new transfer key
	rng := m.rng.GetStream()
	transferKey, err := ftCrypto.NewTransferKey(rng)
	if err != nil {
		rng.Close()
		return ftCrypto.TransferID{}, err
	}
	rng.Close()

	// Get the size of each file part
	partSize, err := m.getPartSize()
	if err != nil {
		return ftCrypto.TransferID{}, errors.Errorf(getPartSizeErr, err)
	}

	// Generate transfer MAC
	mac := ftCrypto.CreateTransferMAC(fileData, transferKey)

	// Partition the file into parts
	parts := partitionFile(fileData, partSize)
	numParts := uint16(len(parts))
	fileSize := uint32(len(fileData))

	// Send the initial file transfer message over E2E
	err = m.sendNewFileTransfer(recipient, fileName, fileType, transferKey, mac,
		numParts, fileSize, retry, preview)
	if err != nil {
		return ftCrypto.TransferID{}, errors.Errorf(sendInitMsgErr, err)
	}

	// Calculate the number of fingerprints to generate
	numFps := calcNumberOfFingerprints(numParts, retry)

	// Add the transfer to storage
	rng = m.rng.GetStream()
	transferID, err := m.sent.AddTransfer(
		recipient, transferKey, parts, numFps, progressCB, period, rng)
	if err != nil {
		return ftCrypto.TransferID{}, err
	}
	rng.Close()

	m.queueParts(transferID, numParts)

	return transferID, nil
}

// RegisterSendProgressCallback adds the sent progress callback to the sent
// transfer so that it will be called when updates for the transfer occur. The
// progress callback is called when initially added and on transfer updates, at
// most once per period.
func (m Manager) RegisterSendProgressCallback(tid ftCrypto.TransferID,
	progressCB interfaces.SentProgressCallback, period time.Duration) error {
	// Get the transfer for the given ID
	transfer, err := m.sent.GetTransfer(tid)
	if err != nil {
		return err
	}

	// Add the progress callback
	transfer.AddProgressCB(progressCB, period)

	return nil
}

// GetSentPartStatus returns the status of the sent file part number for the
// given transfer ID. An error is returned if the sent transfer does not
// exist. The possible values for the status are:
// 0 = unsent
// 1 = sent
// 2 = arrived
// TODO: test
func (m Manager) GetSentPartStatus(tid ftCrypto.TransferID, partNum uint16) (int, error) {
	// Get the transfer for the given ID
	transfer, err := m.sent.GetTransfer(tid)
	if err != nil {
		return unsent, err
	}

	if transfer.IsPartInProgress(partNum) {
		return sent, nil
	} else if transfer.IsPartFinished(partNum) {
		return arrived, nil
	} else {
		return unsent, nil
	}
}

// Resend resends a file if sending fails. Returns an error if CloseSend
// was already called or if the transfer did not run out of retries. This
// function should only be called if the interfaces.SentProgressCallback returns
// an error.
// TODO: add test
// TODO: write test
// TODO: can you resend? Can you reuse fingerprints?
// TODO: what to do if sendE2E fails?
func (m Manager) Resend(tid ftCrypto.TransferID) error {
	// Get the transfer for the given ID
	transfer, err := m.sent.GetTransfer(tid)
	if err != nil {
		return err
	}

	// Check if the transfer has run out of fingerprints, which occurs when the
	// retry limit is reached
	if transfer.GetNumAvailableFps() > 0 {
		return errors.Errorf(transferNotFailedErr, tid)
	}

	return nil
}

// CloseSend deletes a sent file transfer from the sent transfer map and from
// storage once a transfer has completed or reached the retry limit. Returns an
// error if the transfer has not run out of retries.
func (m Manager) CloseSend(tid ftCrypto.TransferID) error {
	// Get the transfer for the given ID
	transfer, err := m.sent.GetTransfer(tid)
	if err != nil {
		return err
	}

	// Check if the transfer has completed or run out of fingerprints, which
	// occurs when the retry limit is reached
	completed, _, _, _, _ := transfer.GetProgress()
	if transfer.GetNumAvailableFps() > 0 && !completed {
		return errors.Errorf(transferInProgressErr, tid)
	}

	// Delete the transfer from storage
	return m.sent.DeleteTransfer(tid)
}

// Receive returns the fully assembled file on the completion of the transfer.
// It deletes the transfer from the received transfer map and from storage.
// Returns an error if the transfer is not complete, the full file cannot be
// verified, or if the transfer cannot be found.
func (m Manager) Receive(tid ftCrypto.TransferID) ([]byte, error) {
	// Get the transfer for the given ID
	transfer, err := m.received.GetTransfer(tid)
	if err != nil {
		return nil, err
	}

	// Get the file from the transfer
	file, err := transfer.GetFile()
	if err != nil {
		return nil, err
	}

	// Return the file and delete the transfer from storage
	return file, m.received.DeleteTransfer(tid)
}

// RegisterReceiveProgressCallback adds the reception progress callback to the
// received transfer so that it will be called when updates for the transfer
// occur. The progress callback is called when initially added and on transfer
// updates, at most once per period.
func (m Manager) RegisterReceiveProgressCallback(tid ftCrypto.TransferID,
	progressCB interfaces.ReceivedProgressCallback, period time.Duration) error {
	// Get the transfer for the given ID
	transfer, err := m.received.GetTransfer(tid)
	if err != nil {
		return err
	}

	// Add the progress callback
	transfer.AddProgressCB(progressCB, period)

	return nil
}

// GetReceivedPartStatus returns the status of the received file part number
// for the given transfer ID. An error is returned if the received transfer
// does not exist. The possible values for the status are:
// 0 = unsent
// 1 = received
// TODO: test
func (m Manager) GetReceivedPartStatus(tid ftCrypto.TransferID, partNum uint16) (int, error) {
	// Get the transfer for the given ID
	transfer, err := m.received.GetTransfer(tid)
	if err != nil {
		return unsent, err
	}

	if transfer.IsPartReceived(partNum) {
		return received, nil
	} else {
		return unsent, nil
	}
}

// calcNumberOfFingerprints is the formula used to calculate the number of
// fingerprints to generate, which is based off the number of file parts and the
// retry float.
func calcNumberOfFingerprints(numParts uint16, retry float32) uint16 {
	return uint16(float32(numParts) * (1 + retry))
}
