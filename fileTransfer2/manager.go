////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/fileTransfer2/callbackTracker"
	"gitlab.com/elixxir/client/fileTransfer2/store"
	"gitlab.com/elixxir/client/fileTransfer2/store/fileMessage"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

const (
	// FileNameMaxLen is the maximum size, in bytes, for a file name. Currently,
	// it is set to 48 bytes.
	FileNameMaxLen = 48

	// FileTypeMaxLen is the maximum size, in bytes, for a file type. Currently,
	// it is set to 8 bytes.
	FileTypeMaxLen = 8

	// FileMaxSize is the maximum file size that can be transferred. Currently,
	// it is set to 250 kB.
	FileMaxSize = 250_000

	// PreviewMaxSize is the maximum size, in bytes, for a file preview.
	// Currently, it is set to 4 kB.
	PreviewMaxSize = 4_000

	// minPartsSendPerRound is the minimum number of file parts sent each round.
	minPartsSendPerRound = 1

	// maxPartsSendPerRound is the maximum number of file parts sent each round.
	maxPartsSendPerRound = 11

	// Size of the buffered channel that queues file parts to package
	batchQueueBuffLen = 10_000

	// Size of the buffered channel that queues file packets to send
	sendQueueBuffLen = 10_000
)

// Stoppable and listener values.
const (
	fileTransferStoppable       = "FileTransfer"
	workerPoolStoppable         = "FilePartSendingWorkerPool"
	batchBuilderThreadStoppable = "BatchBuilderThread"
)

// Error messages.
const (
	errNoSentTransfer     = "could not find sent transfer with ID %s"
	errNoReceivedTransfer = "could not find received transfer with ID %s"

	// NewManager
	errNewOrLoadSent     = "failed to load or create new list of sent file transfers: %+v"
	errNewOrLoadReceived = "failed to load or create new list of received file transfers: %+v"

	// manager.Send
	errFileNameSize      = "length of filename (%d) greater than max allowed length (%d)"
	errFileTypeSize      = "length of file type (%d) greater than max allowed length (%d)"
	errFileSize          = "size of file (%d bytes) greater than max allowed size (%d bytes)"
	errPreviewSize       = "size of preview (%d bytes) greater than max allowed size (%d bytes)"
	errSendNetworkHealth = "cannot initiate file transfer of %q when network is not healthy."
	errNewKey            = "could not generate new transfer key: %+v"
	errNewID             = "could not generate new transfer ID: %+v"
	errAddSentTransfer   = "failed to add transfer: %+v"

	// manager.CloseSend
	errDeleteIncompleteTransfer = "cannot delete transfer %s that has not completed or failed"
	errDeleteSentTransfer       = "could not delete sent transfer %s: %+v"
	errRemoveSentTransfer       = "could not remove transfer %s from list: %+v"

	// manager.AddNew
	errNewRtTransferID = "failed to generate transfer ID for new received file transfer %q: %+v"
	errAddNewRt        = "failed to add new file transfer %s (%q): %+v"

	// manager.Receive
	errIncompleteFile         = "cannot get incomplete file: missing %d of %d parts"
	errDeleteReceivedTransfer = "could not delete received transfer %s: %+v"
	errRemoveReceivedTransfer = "could not remove transfer %s from list: %+v"
)

// manager handles the sending and receiving of file, their storage, and their
// callbacks.
type manager struct {
	// Storage-backed structure for tracking sent file transfers
	sent *store.Sent

	// Storage-backed structure for tracking received file transfers
	received *store.Received

	// Progress callback tracker
	callbacks *callbackTracker.Manager

	// Queue of parts to batch and send
	batchQueue chan store.Part

	// Queue of batches of parts to send
	sendQueue chan []store.Part

	// Function to call to send new file transfer information to recipient
	sendNewCb SendNew

	// Function to call to send notification that file transfer has completed
	sendEndCb SendEnd

	// File transfer parameters
	params Params

	myID *id.ID
	cmix Cmix
	kv   *versioned.KV
	rng  *fastRNG.StreamGenerator
}

type Cmix interface {
	GetMaxMessageLength() int
	SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (id.Round,
		[]ephemeral.Id, error)
	AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
		mp message.Processor) error
	DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint)
	IsHealthy() bool
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
	GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback,
		roundList ...id.Round) error
}

// NewManager creates a new file transfer manager object. If sent or received
// transfers already existed, they are loaded from storage and queued to resume
// once manager.startProcesses is called.
func NewManager(sendNewCb SendNew, sendEndCb SendEnd, params Params,
	myID *id.ID, cmix Cmix, kv *versioned.KV,
	rng *fastRNG.StreamGenerator) (FileTransfer, error) {

	// Create a new list of sent file transfers or load one if it exists
	sent, unsentParts, err := store.NewOrLoadSent(kv)
	if err != nil {
		return nil, errors.Errorf(errNewOrLoadSent, err)
	}

	// Create a new list of received file transfers or load one if it exists
	received, incompleteTransfers, err := store.NewOrLoadReceived(kv)
	if err != nil {
		return nil, errors.Errorf(errNewOrLoadReceived, err)
	}

	// Construct manager
	m := &manager{
		sent:       sent,
		received:   received,
		callbacks:  callbackTracker.NewManager(),
		batchQueue: make(chan store.Part, batchQueueBuffLen),
		sendQueue:  make(chan []store.Part, sendQueueBuffLen),
		sendNewCb:  sendNewCb,
		sendEndCb:  sendEndCb,
		params:     params,
		myID:       myID,
		cmix:       cmix,
		kv:         kv,
		rng:        rng,
	}

	// Add all unsent file parts to queue
	for _, p := range unsentParts {
		m.batchQueue <- p
	}

	// Add all fingerprints for unreceived parts
	for _, rt := range incompleteTransfers {
		m.addFingerprints(rt)
	}

	return m, nil
}

// StartProcesses starts the sending threads. Adheres to the api.Service type.
func (m *manager) StartProcesses() (stoppable.Stoppable, error) {

	// Construct stoppables
	multiStop := stoppable.NewMulti(workerPoolStoppable)
	batchBuilderStop := stoppable.NewSingle(batchBuilderThreadStoppable)

	// Start sending threads
	go m.startSendingWorkerPool(multiStop)
	go m.batchBuilderThread(batchBuilderStop)

	// Create a multi stoppable
	multiStoppable := stoppable.NewMulti(fileTransferStoppable)
	multiStoppable.Add(multiStop)
	multiStoppable.Add(batchBuilderStop)

	return multiStoppable, nil
}

// MaxFileNameLen returns the max number of bytes allowed for a file name.
func (m *manager) MaxFileNameLen() int {
	return FileNameMaxLen
}

// MaxFileTypeLen returns the max number of bytes allowed for a file type.
func (m *manager) MaxFileTypeLen() int {
	return FileTypeMaxLen
}

// MaxFileSize returns the max number of bytes allowed for a file.
func (m *manager) MaxFileSize() int {
	return FileMaxSize
}

// MaxPreviewSize returns the max number of bytes allowed for a file preview.
func (m *manager) MaxPreviewSize() int {
	return PreviewMaxSize
}

/* === Sending ============================================================== */

// Send partitions the given file into cMix message sized chunks and sends them
// via cmix.SendMany.
func (m *manager) Send(fileName, fileType string, fileData []byte,
	recipient *id.ID, retry float32, preview []byte,
	progressCB SentProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {

	// Return an error if the file name is too long
	if len(fileName) > FileNameMaxLen {
		return nil, errors.Errorf(errFileNameSize, len(fileName), FileNameMaxLen)
	}

	// Return an error if the file type is too long
	if len(fileType) > FileTypeMaxLen {
		return nil, errors.Errorf(errFileTypeSize, len(fileType), FileTypeMaxLen)
	}

	// Return an error if the file is too large
	if len(fileData) > FileMaxSize {
		return nil, errors.Errorf(errFileSize, len(fileData), FileMaxSize)
	}

	// Return an error if the preview is too large
	if len(preview) > PreviewMaxSize {
		return nil, errors.Errorf(errPreviewSize, len(preview), PreviewMaxSize)
	}

	// Return an error if the network is not healthy
	if !m.cmix.IsHealthy() {
		return nil, errors.Errorf(errSendNetworkHealth, fileName)
	}

	// Generate new transfer key and transfer ID
	rng := m.rng.GetStream()
	key, err := ftCrypto.NewTransferKey(rng)
	if err != nil {
		rng.Close()
		return nil, errors.Errorf(errNewKey, err)
	}
	tid, err := ftCrypto.NewTransferID(rng)
	if err != nil {
		rng.Close()
		return nil, errors.Errorf(errNewID, err)
	}
	rng.Close()

	// Generate transfer MAC
	mac := ftCrypto.CreateTransferMAC(fileData, key)

	// Get size of each part and partition file into equal length parts
	partMessage := fileMessage.NewPartMessage(m.cmix.GetMaxMessageLength())
	parts := partitionFile(fileData, partMessage.GetPartSize())
	numParts := uint16(len(parts))
	fileSize := uint32(len(fileData))

	// Send the initial file transfer message over E2E
	info := &TransferInfo{
		FileName: fileName,
		FileType: fileType,
		Key:      key,
		Mac:      mac,
		NumParts: numParts,
		Size:     fileSize,
		Retry:    retry,
		Preview:  preview,
	}
	go m.sendNewCb(recipient, info)

	// Calculate the number of fingerprints to generate
	numFps := calcNumberOfFingerprints(len(parts), retry)

	// Create new sent transfer
	st, err := m.sent.AddTransfer(recipient, &key, &tid, fileName, parts, numFps)
	if err != nil {
		return nil, errors.Errorf(errAddSentTransfer, err)
	}

	// Add all parts to the send queue
	for _, p := range st.GetUnsentParts() {
		m.batchQueue <- p
	}

	// Register the progress callback
	m.registerSentProgressCallback(st, progressCB, period)

	return &tid, nil
}

// RegisterSentProgressCallback adds the given callback to the callback manager
// for the given transfer ID. Returns an error if the transfer cannot be found.
func (m *manager) RegisterSentProgressCallback(tid *ftCrypto.TransferID,
	progressCB SentProgressCallback, period time.Duration) error {
	st, exists := m.sent.GetTransfer(tid)
	if !exists {
		return errors.Errorf(errNoSentTransfer, tid)
	}

	m.registerSentProgressCallback(st, progressCB, period)

	return nil
}

// registerSentProgressCallback creates a callback for the sent transfer that
// will get the most recent progress and send it on the progress callback.
func (m *manager) registerSentProgressCallback(st *store.SentTransfer,
	progressCB SentProgressCallback, period time.Duration) {
	if progressCB == nil {
		return
	}

	// Build callback
	cb := func(err error) {
		// Get transfer progress
		arrived, total := st.NumArrived(), st.NumParts()
		completed := arrived == total

		// If the transfer is completed, send last message informing recipient
		if completed && m.params.NotifyUponCompletion {
			go m.sendEndCb(st.Recipient())
		}

		// Build part tracker from copy of part statuses vector
		tracker := &sentFilePartTracker{st.CopyPartStatusVector()}

		// Call the progress callback
		progressCB(completed, arrived, total, tracker, err)
	}

	// Add the callback to the callback tracker
	m.callbacks.AddCallback(st.TransferID(), cb, period)
}

// CloseSend deletes the sent transfer from storage and the sent transfer list.
// Also stops any scheduled progress callbacks and deletes them from the manager
// to prevent any further calls. Deletion only occurs if the transfer has either
// completed or failed.
func (m *manager) CloseSend(tid *ftCrypto.TransferID) error {
	st, exists := m.sent.GetTransfer(tid)
	if !exists {
		return errors.Errorf(errNoSentTransfer, tid)
	}

	// Check that the transfer is either completed or failed
	if st.Status() != store.Completed && st.Status() != store.Failed {
		return errors.Errorf(errDeleteIncompleteTransfer, tid)
	}

	// Delete from storage
	err := st.Delete()
	if err != nil {
		return errors.Errorf(errDeleteSentTransfer, tid, err)
	}

	// Delete from transfers list
	err = m.sent.RemoveTransfer(tid)
	if err != nil {
		return errors.Errorf(errRemoveSentTransfer, tid, err)
	}

	// Stop and delete all progress callbacks
	m.callbacks.Delete(tid)

	return nil
}

/* === Receiving ============================================================ */

func (m *manager) AddNew(fileName string, key *ftCrypto.TransferKey,
	transferMAC []byte, numParts uint16, size uint32, retry float32,
	progressCB ReceivedProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {

	// Generate new transfer ID
	rng := m.rng.GetStream()
	tid, err := ftCrypto.NewTransferID(rng)
	if err != nil {
		rng.Close()
		return nil, errors.Errorf(errNewRtTransferID, fileName, err)
	}
	rng.Close()

	// Calculate the number of fingerprints based on the retry rate
	numFps := calcNumberOfFingerprints(int(numParts), retry)

	// Store the transfer
	rt, err := m.received.AddTransfer(
		key, &tid, fileName, transferMAC, numParts, numFps, size)
	if err != nil {
		return nil, errors.Errorf(errAddNewRt, tid, fileName, err)
	}

	// Start tracking fingerprints for each file part
	m.addFingerprints(rt)

	// Register the progress callback
	m.registerReceivedProgressCallback(rt, progressCB, period)

	return &tid, nil
}

// Receive concatenates the received file and returns it. Only returns the file
// if all file parts have been received and returns an error otherwise. Also
// deletes the transfer from storage. Once Receive has been called on a file, it
// cannot be received again.
func (m *manager) Receive(tid *ftCrypto.TransferID) ([]byte, error) {
	rt, exists := m.received.GetTransfer(tid)
	if !exists {
		return nil, errors.Errorf(errNoReceivedTransfer, tid)
	}

	// Return an error if the transfer is not complete
	if rt.NumReceived() != rt.NumParts() {
		return nil, errors.Errorf(
			errIncompleteFile, rt.NumParts()-rt.NumReceived(), rt.NumParts())
	}

	// Get the file
	file := rt.GetFile()

	// Delete all unused fingerprints
	for _, c := range rt.GetUnusedCyphers() {
		m.cmix.DeleteFingerprint(m.myID, c.GetFingerprint())
	}

	// Delete from storage
	err := rt.Delete()
	if err != nil {
		return nil, errors.Errorf(errDeleteReceivedTransfer, tid, err)
	}

	// Delete from transfers list
	err = m.received.RemoveTransfer(tid)
	if err != nil {
		return nil, errors.Errorf(errRemoveReceivedTransfer, tid, err)
	}

	// Stop and delete all progress callbacks
	m.callbacks.Delete(tid)

	return file, nil
}

// RegisterReceivedProgressCallback adds the given callback to the callback
// manager for the given transfer ID. Returns an error if the transfer cannot be
// found.
func (m *manager) RegisterReceivedProgressCallback(tid *ftCrypto.TransferID,
	progressCB ReceivedProgressCallback, period time.Duration) error {
	rt, exists := m.received.GetTransfer(tid)
	if !exists {
		return errors.Errorf(errNoReceivedTransfer, tid)
	}

	m.registerReceivedProgressCallback(rt, progressCB, period)

	return nil
}

// registerReceivedProgressCallback creates a callback for the received transfer
// that will get the most recent progress and send it on the progress callback.
func (m *manager) registerReceivedProgressCallback(rt *store.ReceivedTransfer,
	progressCB ReceivedProgressCallback, period time.Duration) {
	if progressCB == nil {
		return
	}

	// Build callback
	cb := func(err error) {
		// Get transfer progress
		received, total := rt.NumReceived(), rt.NumParts()
		completed := received == total

		// Build part tracker from copy of part statuses vector
		tracker := &receivedFilePartTracker{rt.CopyPartStatusVector()}

		// Call the progress callback
		progressCB(completed, received, total, tracker, err)
	}

	// Add the callback to the callback tracker
	m.callbacks.AddCallback(rt.TransferID(), cb, period)
}

/* === Utility ============================================================== */

// partitionFile splits the file into parts of the specified part size.
func partitionFile(file []byte, partSize int) [][]byte {
	// Initialize part list to the correct size
	numParts := (len(file) + partSize - 1) / partSize
	parts := make([][]byte, 0, numParts)
	buff := bytes.NewBuffer(file)

	for n := buff.Next(partSize); len(n) > 0; n = buff.Next(partSize) {
		newPart := make([]byte, partSize)
		copy(newPart, n)
		parts = append(parts, newPart)
	}

	return parts
}

// calcNumberOfFingerprints is the formula used to calculate the number of
// fingerprints to generate, which is based off the number of file parts and the
// retry float.
func calcNumberOfFingerprints(numParts int, retry float32) uint16 {
	return uint16(float32(numParts) * (1 + retry))
}

// addFingerprints adds all fingerprints for unreceived parts in the received
// transfer.
func (m *manager) addFingerprints(rt *store.ReceivedTransfer) {
	// Build processor for each file part and add its fingerprint to receive on
	for _, c := range rt.GetUnusedCyphers() {
		p := &processor{
			Cypher:           c,
			ReceivedTransfer: rt,
			manager:          m,
		}

		err := m.cmix.AddFingerprint(m.myID, c.GetFingerprint(), p)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to add fingerprint for transfer "+
				"%s: %+v", rt.TransferID(), err)
		}
	}
}
