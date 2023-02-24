////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"bytes"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/channelsFileTransfer/callbackTracker"
	"gitlab.com/elixxir/client/v4/channelsFileTransfer/store"
	"gitlab.com/elixxir/client/v4/channelsFileTransfer/store/fileMessage"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// TODO:
//  1. Fix the way received messages are checked. The current way is done
//     through the received progress callback, which is inefficient. It also
//     sometimes causes a crash where a part is marked received twice. This
//     crash can be prevented without any other issue by modifying the stateMap,
//     but it is probably best to fix the underlying inefficiency.

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
	// Currently, it is set to 590 bytes.
	PreviewMaxSize = 590

	// minPartsSendPerRound is the minimum number of file parts sent each round.
	minPartsSendPerRound = 1

	// maxPartsSendPerRound is the maximum number of file parts sent each round.
	maxPartsSendPerRound = 11

	// Size of the buffered channel that queues file parts to package
	batchQueueBuffLen = 10_000

	// Size of the buffered channel that queues file parts to send
	sendQueueBuffLen = 10_000

	// Size of the buffered channel that queues the sent file parts.
	sentQueueBuffLen = 10_000
)

// Stoppable and listener values.
const (
	fileTransferStoppable       = "FileTransfer"
	workerPoolStoppable         = "FilePartSendingWorkerPool"
	batchBuilderThreadStoppable = "FilePartBatchBuilderThread"
	resendPartThreadStoppable   = "FilePartResendThread"
)

// Error messages.
const (
	errNoSentTransfer     = "could not find sent transfer with ID %s"
	errNoReceivedTransfer = "could not find received transfer with ID %s"

	// newManager
	errNewOrLoadSent     = "failed to load or create new list of sent file transfers: %+v"
	errNewOrLoadReceived = "failed to load or create new list of received file transfers: %+v"

	// manager.Send
	errFileNameSize      = "length of filename (%d) greater than max allowed length (%d)"
	errFileTypeSize      = "length of file type (%d) greater than max allowed length (%d)"
	errFileSize          = "size of file (%d bytes) greater than max allowed size (%d bytes)"
	errPreviewSize       = "size of preview (%d bytes) greater than max allowed size (%d bytes)"
	errSendNetworkHealth = "cannot initiate file transfer of %q when network is not healthy."
	errNewKey            = "could not generate new transfer key: %+v"
	errNewRecipientID    = "could not generate new recipient ID: %+v"
	errMarshalInfo       = "could not marshal file info: %+v"
	errAddSentTransfer   = "failed to add transfer: %+v"

	// manager.CloseSend
	errDeleteIncompleteTransfer = "cannot delete file %s that has not completed or failed"
	errDeleteSentTransfer       = "could not delete sent file %s: %+v"
	errRemoveSentTransfer       = "could not remove file %s from list: %+v"

	// manager.HandleIncomingTransfer
	errUnmarshalInfo = "failed to unmarshal incoming file info: %+v"
	errAddNewRt      = "failed to add new file transfer %s (%q): %+v"

	// manager.Receive
	errIncompleteFile         = "cannot get incomplete file: missing %d of %d parts"
	errDeleteReceivedTransfer = "could not delete received file %s: %+v"
	errRemoveReceivedTransfer = "could not remove file %s from list: %+v"
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
	batchQueue chan *store.Part

	// Queue of batches of parts to send
	sendQueue chan []*store.Part

	// Queue of parts that were sent and their status needs to be checked
	sentQueue chan *sentPartPacket

	// File transfer parameters
	params Params

	myID      *id.ID
	cmix      Cmix
	cmixGroup *cyclic.Group
	kv        *versioned.KV
	rng       *fastRNG.StreamGenerator
}

// sentPartPacket contains packets of sent parts and the time they were sent. It
// is used to track sent parts and resend any parts that were not received.
type sentPartPacket struct {
	// List of sent parts
	packet []*store.Part

	// The time the send finished at (when the round completed)
	sentTime time.Time

	// If loaded is true, it means the file packet was loaded from storage and
	// the sentTime should be ignored when calculating the time to wait to check
	// if the files need to be resent.
	loaded bool
}

// FtE2e interface matches a subset of the xxdk.E2e methods used by the file
// transfer manager for easier testing.
type FtE2e interface {
	GetStorage() storage.Session
	GetReceptionIdentity() xxdk.ReceptionIdentity
	GetCmix() cmix.Client
	GetRng() *fastRNG.StreamGenerator
	GetE2E() e2e.Handler
}

// Cmix interface matches a subset of the cmix.Client methods used by the file
// transfer manager for easier testing.
type Cmix interface {
	GetMaxMessageLength() int
	SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (
		rounds.Round, []ephemeral.Id, error)
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool,
		fallthroughProcessor message.Processor)
	RemoveIdentity(id *id.ID)
	AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
		mp message.Processor) error
	DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint)
	DeleteClientFingerprints(identity *id.ID)
	CheckInProgressMessages()
	IsHealthy() bool
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
	GetRoundResults(timeout time.Duration,
		roundCallback cmix.RoundEventCallback, roundList ...id.Round)
}

// Storage interface matches a subset of the storage.Session methods used by the
// manager for easier testing.
type Storage interface {
	GetKV() *versioned.KV
	GetCmixGroup() *cyclic.Group
}

// newManager creates a new file transfer manager object. If sent or received
// transfers already existed, they are loaded from storage and queued to resume
// once manager.startProcesses is called.
func newManager(user FtE2e, params Params) (m *manager,
	inProgressSends, inProgressReceives []ftCrypto.ID, err error) {
	kv := user.GetStorage().GetKV()

	// Create a new list of sent file transfers or load one if it exists
	sent, inProgressSends, err := store.NewOrLoadSent(kv)
	if err != nil {
		return nil, nil, nil, errors.Errorf(errNewOrLoadSent, err)
	}

	// Create a new list of received file transfers or load one if it exists
	received, inProgressReceives, err := store.NewOrLoadReceived(false, kv)
	if err != nil {
		return nil, nil, nil, errors.Errorf(errNewOrLoadReceived, err)
	}

	// Construct manager
	m = &manager{
		sent:       sent,
		received:   received,
		callbacks:  callbackTracker.NewManager(),
		batchQueue: make(chan *store.Part, batchQueueBuffLen),
		sendQueue:  make(chan []*store.Part, sendQueueBuffLen),
		sentQueue:  make(chan *sentPartPacket, sentQueueBuffLen),
		params:     params,
		myID:       user.GetReceptionIdentity().ID,
		cmix:       user.GetCmix(),
		cmixGroup:  user.GetStorage().GetCmixGroup(),
		kv:         kv,
		rng:        user.GetRng(),
	}

	return m, inProgressSends, inProgressReceives, nil
}

// LoadInProgressTransfers queues the in-progress sent and received transfers.
func (m *manager) LoadInProgressTransfers(
	sentFileParts map[ftCrypto.ID][][]byte, sentToRemove []ftCrypto.ID) error {
	unsentParts, sentParts, err := m.sent.LoadTransfers(sentFileParts)
	if err != nil {
		return err
	}

	// Add all unsent file parts to queue
	for _, p := range unsentParts {
		m.batchQueue <- p
	}

	// Add all sent file parts to recheck queue
	if len(sentParts) > 0 {
		m.sentQueue <- &sentPartPacket{packet: sentParts, loaded: true}
	}

	if err = m.sent.RemoveTransfers(sentToRemove...); err != nil {
		return err
	}

	return nil
}

// StartProcesses starts the sending threads. Adheres to the xxdk.Service type.
func (m *manager) StartProcesses() (stoppable.Stoppable, error) {
	// Construct stoppables
	senderPoolStop := stoppable.NewMulti(workerPoolStoppable)
	batchBuilderStop := stoppable.NewSingle(batchBuilderThreadStoppable)
	resendPartsStop := stoppable.NewMulti(resendPartThreadStoppable)

	// Start sending threads
	// Note that the startSendingWorkerPool already creates thread for every
	// worker. As a result, there is no need to run it asynchronously. In fact,
	// running this asynchronously could result in a race condition where
	// some worker threads are not added to senderPoolStop before that stoppable
	// is added to the multiStoppable.
	m.startSendingWorkerPool(senderPoolStop)
	go m.batchBuilderThread(batchBuilderStop)
	go m.resendUnreceived(resendPartsStop)

	// Create a multi stoppable
	multiStoppable := stoppable.NewMulti(fileTransferStoppable)
	multiStoppable.Add(senderPoolStop)
	multiStoppable.Add(batchBuilderStop)
	multiStoppable.Add(resendPartsStop)

	return multiStoppable, nil
}

// maxFileNameLen returns the max number of bytes allowed for a file name.
func (m *manager) maxFileNameLen() int {
	return FileNameMaxLen
}

// maxFileTypeLen returns the max number of bytes allowed for a file type.
func (m *manager) maxFileTypeLen() int {
	return FileTypeMaxLen
}

// maxFileSize returns the max number of bytes allowed for a file.
func (m *manager) maxFileSize() int {
	return FileMaxSize
}

// maxPreviewSize returns the max number of bytes allowed for a file preview.
func (m *manager) maxPreviewSize() int {
	return PreviewMaxSize
}

/* === Sending ============================================================== */

// Send initiates the sending of a file to the recipient and returns a file
// ID that uniquely identifies this file transfer.
//
// In-progress transfers are restored when closing and reopening; however, a
// SentProgressCallback must be registered again.
func (m *manager) Send(fileName, fileType string, fileData []byte,
	retry float32, preview []byte, completeCB SendCompleteCallback,
	progressCB SentProgressCallback, period time.Duration) (ftCrypto.ID, error) {

	// Return an error if the file name is too long
	if len(fileName) > FileNameMaxLen {
		return ftCrypto.ID{},
			errors.Errorf(errFileNameSize, len(fileName), FileNameMaxLen)
	}

	// Return an error if the file type is too long
	if len(fileType) > FileTypeMaxLen {
		return ftCrypto.ID{},
			errors.Errorf(errFileTypeSize, len(fileType), FileTypeMaxLen)
	}

	// Return an error if the file is too large
	if len(fileData) > FileMaxSize {
		return ftCrypto.ID{},
			errors.Errorf(errFileSize, len(fileData), FileMaxSize)
	}

	// Return an error if the preview is too large
	if len(preview) > PreviewMaxSize {
		return ftCrypto.ID{},
			errors.Errorf(errPreviewSize, len(preview), PreviewMaxSize)
	}

	// Return an error if the network is not healthy
	if !m.cmix.IsHealthy() {
		return ftCrypto.ID{}, errors.Errorf(errSendNetworkHealth, fileName)
	}

	// Generate new transfer key and file ID
	rng := m.rng.GetStream()
	key, err := ftCrypto.NewTransferKey(rng)
	if err != nil {
		rng.Close()
		return ftCrypto.ID{}, errors.Errorf(errNewKey, err)
	}
	fid := ftCrypto.NewID(fileData)

	// Generate random identity to send the file to that will be used later for
	// others to receive the file
	newID, err := id.NewRandomID(rng, id.User)
	if err != nil {
		rng.Close()
		return ftCrypto.ID{}, errors.Errorf(errNewRecipientID, err)
	}
	rng.Close()

	// Generate transfer MAC
	mac := ftCrypto.CreateTransferMAC(fileData, key)

	// Get size of each part and partition file into equal length parts
	partMessage := fileMessage.NewPartMessage(m.cmix.GetMaxMessageLength())
	parts := partitionFile(fileData, partMessage.GetPartSize())
	numParts := uint16(len(parts))
	fileSize := uint32(len(fileData))

	// Calculate the number of fingerprints to generate
	numFps := calcNumberOfFingerprints(len(parts), retry)

	// Create new sent transfer
	// TODO: how to handle differing filename and file type
	st, err := m.sent.AddTransfer(
		newID, &key, fid, fileName, fileSize, parts, numFps)
	if err != nil {
		return ftCrypto.ID{}, errors.Errorf(errAddSentTransfer, err)
	}

	// Build FileInfo that will be returned to the user on completion
	info := &FileInfo{fid, st.Recipient(), fileName, fileType, key, mac, numParts,
		fileSize, retry, preview}
	fileInfo, err := info.Marshal()
	if err != nil {
		return ftCrypto.ID{}, errors.Errorf(errMarshalInfo, err)
	}

	// Add all parts to the send queue
	for _, p := range st.GetUnsentParts() {
		m.batchQueue <- p
	}

	jww.DEBUG.Printf("[FT] Created new sent file transfer %s for %q "+
		"(type %s, size %d bytes, %d parts, retry %f)",
		st.FileID(), fileName, fileType, fileSize, numParts, retry)

	// Register the progress callback
	m.registerSentProgressCallback(st, progressCB, period)

	// Start tracking the received file parts for the SentTransfer
	_, _, err = m.handleIncomingTransfer(
		fileInfo, m.checkedReceivedParts(st, info, completeCB), 0)
	if err != nil {
		return ftCrypto.ID{}, err
	}

	return fid, nil
}

// RegisterSentProgressCallback adds the given callback to the callback manager
// for the given file ID. Returns an error if the transfer cannot be found.
func (m *manager) RegisterSentProgressCallback(fid ftCrypto.ID,
	progressCB SentProgressCallback, period time.Duration) error {
	st, exists := m.sent.GetTransfer(fid)
	if !exists {
		return errors.Errorf(errNoSentTransfer, fid)
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
	cbID := st.GetNewCallbackID()
	cb := func(err error) {
		// Get transfer progress
		sent, received, total := st.NumSent(), st.NumReceived(), st.NumParts()
		completed := received == total

		// Build part tracker from copy of part statuses vector
		tracker := &sentFilePartTracker{st.CopyPartStatusVector()}

		// If the callback data is the same as the last call, skip the call
		if !st.CompareAndSwapCallbackFps(
			cbID, completed, sent, received, total, err) {
			return
		}

		// Call the progress callback
		progressCB(completed, sent, received, total, st, tracker, err)
	}

	// Add the callback to the callback tracker
	m.callbacks.AddCallback(st.FileID(), cb, period)
}

// CloseSend deletes the sent transfer from storage and the sent transfer list.
// Also stops any scheduled progress callbacks and deletes them from the manager
// to prevent any further calls. Deletion only occurs if the transfer has either
// completed or failed.
//
// This function should be called once a transfer completes or errors out (as
// reported by the progress callback). Returns an error if the transfer has not
// run out of retries.
func (m *manager) CloseSend(fid ftCrypto.ID) error {
	st, exists := m.sent.GetTransfer(fid)
	if !exists {
		return errors.Errorf(errNoSentTransfer, fid)
	}

	// Check that the transfer is either completed or failed
	if st.Status() != store.Completed && st.Status() != store.Failed {
		return errors.Errorf(errDeleteIncompleteTransfer, fid)
	}

	// Delete from storage
	err := st.Delete()
	if err != nil {
		return errors.Errorf(errDeleteSentTransfer, fid, err)
	}

	// Delete from transfers list
	err = m.sent.RemoveTransfer(fid)
	if err != nil {
		return errors.Errorf(errRemoveSentTransfer, fid, err)
	}

	// Stop and delete all progress callbacks
	m.callbacks.Delete(fid)

	jww.DEBUG.Printf("[FT] Sent file %s has been closed.", fid)

	return nil
}

/* === Receiving ============================================================ */

// HandleIncomingTransfer starts tracking the received file parts for the
// given marshalled FileInfo and returns the file's ID and FileInfo.
//
// In-progress transfers are restored when closing and reopening; however, a
// ReceivedProgressCallback must be registered again.
func (m *manager) HandleIncomingTransfer(fileInfo []byte,
	progressCB ReceivedProgressCallback, period time.Duration) (
	ftCrypto.ID, *FileInfo, error) {

	_, fi, err := m.handleIncomingTransfer(fileInfo, progressCB, period)
	if err != nil {
		return ftCrypto.ID{}, nil, err
	}
	return fi.FID, fi, nil
}

// handleIncomingTransfer starts tracking the received file parts for the given
// file information and returns the ReceivedTransfer object for the file and its
// FileInfo.
func (m *manager) handleIncomingTransfer(fileInfo []byte,
	progressCB ReceivedProgressCallback, period time.Duration) (
	*store.ReceivedTransfer, *FileInfo, error) {

	// Unmarshal the payload
	t, err := UnmarshalFileInfo(fileInfo)
	if err != nil {
		return nil, nil, errors.Errorf(errUnmarshalInfo, err)
	}

	// Calculate the number of fingerprints based on the retry rate
	numFps := calcNumberOfFingerprints(int(t.NumParts), t.Retry)

	// Store the transfer
	rt, err := m.received.AddTransfer(t.RecipientID, &t.Key, t.FID, t.FileName,
		t.Mac, t.Size, t.NumParts, numFps)
	if err != nil {
		return nil, nil, errors.Errorf(errAddNewRt, t.FID, t.FileName, err)
	}

	jww.DEBUG.Printf("[FT] Added new received file %s named %q "+
		"(type %s, size %d bytes, %d parts, %d fingerprints)",
		rt.FileID(), t.FileName, t.FileType, t.Size, t.NumParts, numFps)

	// Start tracking fingerprints for each file part
	m.addFingerprints(rt)

	// Register the progress callback
	m.registerReceivedProgressCallback(rt, progressCB, period)

	return rt, t, nil
}

// Receive concatenates the received file and returns it. Only returns the file
// if all file parts have been received and returns an error otherwise. Also
// deletes the transfer from storage. Once Receive has been called on a file, it
// cannot be received again.
func (m *manager) Receive(fid ftCrypto.ID) ([]byte, error) {
	rt, exists := m.received.GetTransfer(fid)
	if !exists {
		return nil, errors.Errorf(errNoReceivedTransfer, fid)
	}

	// Return an error if the transfer is not complete
	if rt.NumReceived() != rt.NumParts() {
		return nil, errors.Errorf(
			errIncompleteFile, rt.NumParts()-rt.NumReceived(), rt.NumParts())
	}

	// Get the file
	file := rt.GetFile()

	// Delete all unused fingerprints
	m.cmix.DeleteClientFingerprints(rt.Recipient())

	// Delete from storage
	err := rt.Delete()
	if err != nil {
		return nil, errors.Errorf(errDeleteReceivedTransfer, fid, err)
	}

	// Delete from transfers list
	err = m.received.RemoveTransfer(fid)
	if err != nil {
		return nil, errors.Errorf(errRemoveReceivedTransfer, fid, err)
	}

	// Stop and delete all progress callbacks
	m.callbacks.Delete(fid)

	jww.DEBUG.Printf("[FT] Received file %s has been received.", fid)

	return file, nil
}

func (m *manager) receive(rt *store.ReceivedTransfer) ([]byte, error) {
	// Return an error if the transfer is not complete
	if rt.NumReceived() != rt.NumParts() {
		return nil, errors.Errorf(
			errIncompleteFile, rt.NumParts()-rt.NumReceived(), rt.NumParts())
	}

	// Get the file
	file := rt.GetFile()

	// Delete all unused fingerprints
	m.cmix.DeleteClientFingerprints(rt.Recipient())

	// Removed the tracked identity
	m.cmix.RemoveIdentity(rt.Recipient())

	// Delete from storage
	err := rt.Delete()
	if err != nil {
		return nil, errors.Errorf(errDeleteReceivedTransfer, rt.FileID(), err)
	}

	// Delete from transfers list
	err = m.received.RemoveTransfer(rt.FileID())
	if err != nil {
		return nil, errors.Errorf(errRemoveReceivedTransfer, rt.FileID(), err)
	}

	// Stop and delete all progress callbacks
	m.callbacks.Delete(rt.FileID())

	jww.DEBUG.Printf("[FT] Received file %s has been received.", rt.FileID())

	return file, nil
}

// RegisterReceivedProgressCallback adds the given callback to the callback
// manager for the given file ID. Returns an error if the transfer cannot be
// found.
func (m *manager) RegisterReceivedProgressCallback(fid ftCrypto.ID,
	progressCB ReceivedProgressCallback, period time.Duration) error {
	rt, exists := m.received.GetTransfer(fid)
	if !exists {
		return errors.Errorf(errNoReceivedTransfer, fid)
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
	cbID := rt.GetNewCallbackID()
	cb := func(err error) {
		// Get transfer progress
		received, total := rt.NumReceived(), rt.NumParts()
		completed := received == total

		// Build part tracker from copy of part statuses vector
		tracker := &receivedFilePartTracker{rt.CopyPartStatusVector()}

		// If the callback data is the same as the last call, skip the call
		if !rt.CompareAndSwapCallbackFps(cbID, completed, received, total, err) {
			return
		}

		// Call the progress callback
		progressCB(completed, received, total, rt, tracker, err)
	}

	// Add the callback to the callback tracker
	m.callbacks.AddCallback(rt.FileID(), cb, period)
}

// ReceivedPartsCallback is a callback function that is called every time there
// are new received parts to be saved.
type ReceivedPartsCallback func(fileData []byte, err error)

func (m *manager) registerReceivedPartsCallback(fid ftCrypto.ID,
	partsCB ReceivedPartsCallback, period time.Duration) error {
	rt, exists := m.received.GetTransfer(fid)
	if !exists {
		return errors.Errorf(errNoReceivedTransfer, fid)
	}

	// Build callback
	cbID := rt.GetNewCallbackID()
	cb := func(err error) {
		// Get transfer progress
		received, total := rt.NumReceived(), rt.NumParts()
		completed := received == total

		// If the callback data is the same as the last call, skip the call
		if !rt.CompareAndSwapCallbackFps(cbID, completed, received, total, err) {
			return
		}

		if err != nil {
			partsCB(nil, err)
		}

		data, err := rt.MarshalPartialFile()
		if err != nil {
			partsCB(nil, errors.Wrap(err,
				"failed to get partial file for saving to the event model"))
		}

		partsCB(data, nil)
	}

	// Add the callback to the callback tracker
	m.callbacks.AddCallback(rt.FileID(), cb, period)

	return nil
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
	// Start tracking messages for the recipient ID with persistence == false so
	// that all messages are re-downloaded on client restart
	m.cmix.AddIdentity(rt.Recipient(), identity.Forever, false, nil)

	// Build processor for each file part and add its fingerprint to receive on
	for _, c := range rt.GetUnusedCyphers() {
		p := &processor{
			Cypher:           c,
			ReceivedTransfer: rt,
			manager:          m,
		}

		err := m.cmix.AddFingerprint(rt.Recipient(), c.GetFingerprint(), p)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to add fingerprint for file %s: %+v",
				rt.FileID(), err)
		}
	}

	m.cmix.CheckInProgressMessages()
}
