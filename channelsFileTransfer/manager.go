////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
)

// TODO: Fix the way received messages are checked. The current way is done
//     through the received progress callback, which is inefficient. It also
//     sometimes causes a crash where a part is marked received twice. This
//     crash can be prevented without any other issue by modifying the stateMap,
//     but it is probably best to fix the underlying inefficiency.

const (
	// The maximum size, in bytes, for a file name.
	fileNameMaxLen = 48

	// The maximum size, in bytes, for a file type.
	fileTypeMaxLen = 8

	// The maximum file size that can be transferred.
	fileMaxSize = 250_000

	// The maximum size, in bytes, for a file preview.
	previewMaxSize = 297

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

	// manager.send
	errFileNameSize      = "received %d-byte file name; max allowed %d bytes"
	errFileTypeSize      = "received %d-byte file type; max allowed %d bytes"
	fileSizeMaxErr       = "received %d-byte file; maximum size is %d bytes"
	fileSizeMinErr       = "received %d-byte file; file cannot be empty"
	errPreviewSize       = "received %d-byte preview type; max allowed %d bytes"
	errSendNetworkHealth = "network not healthy"
	errNewKey            = "could not generate new transfer key: %+v"
	errNewRecipientID    = "could not generate new recipient ID: %+v"
	errAddSentTransfer   = "failed to add transfer: %+v"

	// manager.CloseSend
	errDeleteIncompleteTransfer = "file not completed or failed"
	errDeleteSentTransfer       = "could not delete sent file %s: %+v"
	errRemoveSentTransfer       = "could not remove file %s from list: %+v"

	// manager.HandleIncomingTransfer
	errAddNewRt = "failed to add new file transfer %s: %+v"

	// manager.Receive
	errIncompleteFile         = "missing %d of %d parts"
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
	kv        versioned.KV
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

// String prints the sentPartPacket in a human-readable form for logging and
// debugging. This function adheres to the fmt.Stringer interface.
func (spp *sentPartPacket) String() string {
	fields := []string{
		fmt.Sprintf("packets: %s", spp.packet),
		"sentTime:" + spp.sentTime.String(),
		"loaded:" + strconv.FormatBool(spp.loaded),
	}

	return "{" + strings.Join(fields, " ") + "}"
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

// newManager creates a new file transfer manager object. If sent or received
// transfers already existed, they are loaded from storage and queued to resume
// once manager.startProcesses is called.
func newManager(user FtE2e, params Params) (m *manager,
	inProgressSends, inProgressReceives []ftCrypto.ID, err error) {
	kv := user.GetStorage().GetKV()

	// Create a new list of sent file transfers or load one if it exists
	sent, inProgressSends, err := store.NewOrLoadSent(kv)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create a new list of received file transfers or load one if it exists
	received, inProgressReceives, err := store.NewOrLoadReceived(false, kv)
	if err != nil {
		return nil, nil, nil, err
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

// loadInProgressUploads loads the file for each in-progress upload into the
// manager and restarts their upload from where they left off. Any stale file
// uploads (files that no longer appear in the event model) are removed from the
// transfer list.
func (m *manager) loadInProgressUploads(uploads map[ftCrypto.ID]ModelFile,
	staleUploads []ftCrypto.ID, progressCB SentProgressCallback,
	completeCB sendCompleteCallback) error {
	partSize := fileMessage.
		NewPartMessage(m.cmix.GetMaxMessageLength()).GetPartSize()

	var i int
	for fid, file := range uploads {
		i++

		// Load transfer from storage into sent transfer list
		parts := partitionFile(file.Data, partSize)
		st, err := m.sent.LoadTransfer(fid, parts)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to load file %s from uploads "+
				"(%d/%d): %+v", fid, i, len(uploads), err)
			continue
		}

		m.registerSentProgressCallback(st, progressCB, 0)

		fl := FileLink{
			FileID:        st.GetFileID(),
			RecipientID:   st.GetRecipient(),
			SentTimestamp: st.SentTimestamp(),
			Key:           *st.GetKey(),
			Mac:           st.GetMAC(),
			Size:          st.GetFileSize(),
			NumParts:      st.GetNumParts(),
			Retry:         st.GetRetry(),
		}

		// Start tracking the received file parts for the SentTransfer
		callbacks := []receivedProgressCBs{{
			m.checkedReceivedParts(st, &fl, completeCB), 0}}
		_, err = m.handleIncomingTransfer(&fl, callbacks)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to initiate tracking of received "+
				"parts for file upload %s (%d/%d): %+v",
				fid, i, len(uploads), err)
			continue
		}

		// Add all unsent file parts to queue
		for _, p := range st.GetUnsentParts() {
			m.batchQueue <- p
		}

		// Add all sent file parts to recheck queue
		if len(st.GetSentParts()) > 0 {
			m.sentQueue <- &sentPartPacket{st.GetSentParts(), time.Time{}, true}
		}
	}

	// Remove all stale transfers
	if err := m.sent.RemoveTransfers(staleUploads...); err != nil {
		return err
	}

	return nil
}

// loadInProgressDownloads loads the file for each in-progress download into the
// manager and restarts their download from the beginning. Any stale file
// downloads (files that no longer appear in the event model) are removed from
// the transfer list.
func (m *manager) loadInProgressDownloads(downloads map[ftCrypto.ID]ModelFile,
	staleDownloads []ftCrypto.ID, progressCB ReceivedProgressCallback) error {
	var i int
	for fid, file := range downloads {
		i++

		// Unmarshal file link first; if this fails, skip the entire file
		var fl FileLink
		if err := json.Unmarshal(file.Link, &fl); err != nil {
			jww.ERROR.Printf("[FT] Failed to JSON unmarshal %T for file "+
				"download %s (%d/%d): %+v", fl, fid, i, len(downloads), err)
			continue
		}

		// Add file to be downloaded
		var err error
		callbacks := []receivedProgressCBs{{progressCB, 0}}
		_, err = m.handleIncomingTransfer(&fl, callbacks)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to initiate download of file %s "+
				"(%d/%d): %+v", fid, i, len(downloads), err)
			continue
		}
	}

	// Remove all stale transfers
	if err := m.received.RemoveTransfers(staleDownloads...); err != nil {
		return err
	}

	return nil
}

// startProcesses starts the sending threads. Adheres to the xxdk.Service type.
func (m *manager) startProcesses() (stoppable.Stoppable, error) {
	// Construct stoppables
	senderPoolStop := stoppable.NewMulti(workerPoolStoppable)
	batchBuilderStop := stoppable.NewSingle(batchBuilderThreadStoppable)
	resendPartsStop := stoppable.NewSingle(resendPartThreadStoppable)

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
	return fileNameMaxLen
}

// maxFileTypeLen returns the max number of bytes allowed for a file type.
func (m *manager) maxFileTypeLen() int {
	return fileTypeMaxLen
}

// maxFileSize returns the max number of bytes allowed for a file.
func (m *manager) maxFileSize() int {
	return fileMaxSize
}

// maxPreviewSize returns the max number of bytes allowed for a file preview.
func (m *manager) maxPreviewSize() int {
	return previewMaxSize
}

/* === Sending ============================================================== */

// verifyFile verifies that the data is within the size range allowed.
func (m *manager) verifyFile(fileData []byte) error {
	// Return an error if the file is too large or empty
	if fileLen := len(fileData); fileLen > m.maxFileSize() {
		return errors.Errorf(fileSizeMaxErr, fileLen, fileMaxSize)
	} else if fileLen == 0 {
		return errors.Errorf(fileSizeMinErr, fileLen)
	}

	return nil
}

// verifyFile checks all the lengths of the file and its metadata and ensures
// they are of the valid lengths.
func (m *manager) verifyFileInfo(fileName, fileType string, preview []byte) error {
	// Return an error if the file name is too long
	if len(fileName) > fileNameMaxLen {
		return errors.Errorf(errFileNameSize, len(fileName), fileNameMaxLen)
	}

	// Return an error if the file type is too long
	if len(fileType) > fileTypeMaxLen {
		return errors.Errorf(errFileTypeSize, len(fileType), fileTypeMaxLen)
	}

	// Return an error if the preview is too large
	if len(preview) > previewMaxSize {
		return errors.Errorf(errPreviewSize, len(preview), previewMaxSize)
	}

	return nil
}

// sendCompleteCallback is called when a file transfer has successfully sent.
// The returned FileLink can be marshalled and sent to others so that they can
// receive the file.
type sendCompleteCallback func(fi FileLink)

type sentProgressCBs struct {
	cb     SentProgressCallback
	period time.Duration
}

// send initiates the sending of a file to the recipient and returns a file
// ID that uniquely identifies this file transfer.
//
// In-progress transfers are restored when closing and reopening; however, a
// SentProgressCallback must be registered again.
func (m *manager) send(fid ftCrypto.ID, fileData []byte, retry float32,
	completeCB sendCompleteCallback, progressCBs []sentProgressCBs) (
	*store.SentTransfer, error) {

	// Return an error if the file is too large
	if err := m.verifyFile(fileData); err != nil {
		return nil, err
	}

	// Return an error if the network is not healthy
	if !m.cmix.IsHealthy() {
		return nil, errors.New(errSendNetworkHealth)
	}

	// Generate new transfer key and file ID
	rng := m.rng.GetStream()
	key, err := ftCrypto.NewTransferKey(rng)
	if err != nil {
		rng.Close()
		return nil, errors.Errorf(errNewKey, err)
	}

	// Generate random identity to send the file to that will be used later for
	// others to receive the file
	newID, err := id.NewRandomID(rng, id.User)
	if err != nil {
		rng.Close()
		return nil, errors.Errorf(errNewRecipientID, err)
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
	sentTimestamp := netTime.Now()
	st, err := m.sent.AddTransfer(
		newID, sentTimestamp, &key, mac, fid, fileSize, parts, numFps, retry)
	if err != nil {
		return nil, errors.Errorf(errAddSentTransfer, err)
	}

	// Build FileInfo that will be returned to the user on completion
	fl := &FileLink{
		FileID:        fid,
		RecipientID:   st.GetRecipient(),
		SentTimestamp: sentTimestamp,
		Key:           key,
		Mac:           mac,
		Size:          fileSize,
		NumParts:      numParts,
		Retry:         retry,
	}

	jww.DEBUG.Printf("[FT] Created new sent file transfer %s (size %d bytes, "+
		"%d parts, retry %f)", st.GetFileID(), fileSize, numParts, retry)

	// Register the progress callback
	for _, p := range progressCBs {
		m.registerSentProgressCallback(st, p.cb, p.period)
	}

	callbacks :=
		[]receivedProgressCBs{{m.checkedReceivedParts(st, fl, completeCB), 0}}

	// Start tracking the received file parts for the SentTransfer
	_, err = m.handleIncomingTransfer(fl, callbacks)
	if err != nil {
		return nil, err
	}

	// Add all parts to the send queue
	for _, p := range st.GetUnsentParts() {
		m.batchQueue <- p
	}

	return nil, nil
}

// registerSentProgressCallback adds the given callback to the callback manager
// for the given transfer.
func (m *manager) registerSentProgressCallback(st *store.SentTransfer,
	progressCB SentProgressCallback, period time.Duration) {
	if progressCB == nil {
		return
	}

	// Build callback
	cbID := st.GetNewCallbackID()
	cb := func(err error) {
		// Get transfer progress
		sent, received, total := st.NumSent(), st.NumReceived(), st.GetNumParts()
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
	m.callbacks.AddCallback(st.GetFileID(), cb, period)
}

// closeSend deletes the sent transfer from storage and the sent transfer list.
// Also stops any scheduled progress callbacks and deletes them from the manager
// to prevent any further calls. Deletion only occurs if the transfer has either
// completed or failed.
//
// This function should be called once a transfer completes or errors out (as
// reported by the progress callback). Returns an error if the transfer has not
// run out of retries.
func (m *manager) closeSend(st *store.SentTransfer) error {
	// Check that the transfer is either completed or failed
	if st.Status() != store.Completed && st.Status() != store.Failed {
		return errors.New(errDeleteIncompleteTransfer)
	}

	// Delete from storage
	err := st.Delete()
	if err != nil {
		return errors.Errorf(errDeleteSentTransfer, st.GetFileID(), err)
	}

	// Delete from transfers list
	err = m.sent.RemoveTransfer(st.GetFileID())
	if err != nil {
		return errors.Errorf(errRemoveSentTransfer, st.GetFileID(), err)
	}

	// Stop and delete all progress callbacks
	m.callbacks.Delete(st.GetFileID())

	jww.DEBUG.Printf("[FT] Sent file %s has been closed.", st.GetFileID())

	return nil
}

/* === Receiving ============================================================ */

type receivedProgressCBs struct {
	cb     ReceivedProgressCallback
	period time.Duration
}

// handleIncomingTransfer starts tracking the received file parts for the given
// file information and returns the ReceivedTransfer object for the file and its
// FileInfo.
//
// In-progress transfers are restored when closing and reopening; however, a
// ReceivedProgressCallback must be registered again.
func (m *manager) handleIncomingTransfer(fl *FileLink,
	progressCBs []receivedProgressCBs) (
	*store.ReceivedTransfer, error) {

	// Calculate the number of fingerprints based on the retry rate
	numFps := calcNumberOfFingerprints(int(fl.NumParts), fl.Retry)

	// Store the transfer
	rt, err := m.received.AddTransfer(fl.RecipientID, &fl.Key, fl.FileID,
		fl.Mac, fl.Size, fl.NumParts, numFps)
	if err != nil {
		return nil, errors.Errorf(errAddNewRt, fl.FileID, err)
	}

	jww.DEBUG.Printf("[FT] Added new received file %s "+
		"(size %d bytes, %d parts, %d fingerprints)",
		rt.GetFileID(), fl.Size, fl.NumParts, numFps)

	// Register the progress callback
	for _, c := range progressCBs {
		m.registerReceivedProgressCallback(rt, c.cb, c.period)
	}

	// Start tracking fingerprints for each file part
	m.addFingerprints(rt)

	return rt, nil
}

// receive concatenates the received file and returns it. Only returns the file
// if all file parts have been received and returns an error otherwise. Also
// deletes the transfer from storage. Once Receive has been called on a file, it
// cannot be received again.
func (m *manager) receive(rt *store.ReceivedTransfer) ([]byte, error) {
	// Return an error if the transfer is not complete
	if rt.NumReceived() != rt.GetNumParts() {
		return nil, errors.Errorf(
			errIncompleteFile, rt.GetNumParts()-rt.NumReceived(), rt.GetNumParts())
	}

	// Get the file
	file := rt.GetFile()

	// Delete all unused fingerprints
	m.cmix.DeleteClientFingerprints(rt.GetRecipient())

	// Removed the tracked identity
	m.cmix.RemoveIdentity(rt.GetRecipient())

	// Delete from storage
	err := rt.Delete()
	if err != nil {
		return nil, errors.Errorf(errDeleteReceivedTransfer, rt.GetFileID(), err)
	}

	// Delete from transfers list
	err = m.received.RemoveTransfer(rt.GetFileID())
	if err != nil {
		return nil, errors.Errorf(errRemoveReceivedTransfer, rt.GetFileID(), err)
	}

	// Stop and delete all progress callbacks
	m.callbacks.Delete(rt.GetFileID())

	jww.DEBUG.Printf("[FT] Received file %s has been received.", rt.GetFileID())

	return file, nil
}

// receiveFromID looks up the transfer from the ID and then receives the file.
// Returns an error if the transfer cannot be found.
func (m *manager) receiveFromID(fid ftCrypto.ID) ([]byte, error) {
	rt, exists := m.received.GetTransfer(fid)
	if !exists {
		return nil, errors.Errorf(errNoReceivedTransfer, fid)
	}

	return m.receive(rt)
}

// RegisterReceivedProgressCallback adds the given callback to the callback
// manager for the given transfer.
func (m *manager) registerReceivedProgressCallback(rt *store.ReceivedTransfer,
	progressCB ReceivedProgressCallback, period time.Duration) {
	if progressCB == nil {
		return
	}

	// Build callback
	cbID := rt.GetNewCallbackID()
	cb := func(err error) {
		// Get transfer progress
		received, total := rt.NumReceived(), rt.GetNumParts()
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
	m.callbacks.AddCallback(rt.GetFileID(), cb, period)
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
	m.cmix.AddIdentity(rt.GetRecipient(), identity.Forever, false, nil)

	// Build processor for each file part and add its fingerprint to receive on
	for _, c := range rt.GetUnusedCyphers() {
		p := &processor{
			Cypher:           c,
			ReceivedTransfer: rt,
			manager:          m,
		}

		err := m.cmix.AddFingerprint(rt.GetRecipient(), c.GetFingerprint(), p)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to add fingerprint for file %s: %+v",
				rt.GetFileID(), err)
		}
	}

	m.cmix.CheckInProgressMessages()
}
