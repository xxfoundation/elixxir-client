////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

// Storage keys and versions.
const (
	sentTransferPrefix      = "FileTransferSentTransferStore"
	sentTransferKey         = "SentTransfer"
	sentTransferVersion     = 0
	sentFpVectorKey         = "SentFingerprintVector"
	sentInProgressVectorKey = "SentInProgressStatusVector"
	sentFinishedVectorKey   = "SentFinishedStatusVector"
)

// Error messages.
const (
	// NewSentTransfer
	newSentTransferFpVectorErr  = "failed to create new state vector for fingerprints: %+v"
	newSentTransferPartStoreErr = "failed to create new part store: %+v"
	newInProgressTransfersErr   = "failed to create new in-progress transfers bundle: %+v"
	newFinishedTransfersErr     = "failed to create new finished transfers bundle: %+v"
	newSentInProgressVectorErr  = "failed to create new state vector for in-progress status: %+v"
	newSentFinishedVectorErr    = "failed to create new state vector for finished status: %+v"

	// SentTransfer.ReInit
	reInitSentTransferFpVectorErr = "failed to overwrite fingerprint state vector with new vector: %+v"
	reInitInProgressTransfersErr  = "failed to overwrite in-progress transfers bundle: %+v"
	reInitFinishedTransfersErr    = "failed to overwrite finished transfers bundle: %+v"
	reInitSentInProgressVectorErr = "failed to overwrite in-progress state vector with new vector: %+v"
	reInitSentFinishedVectorErr   = "failed to overwrite finished state vector with new vector: %+v"

	// loadSentTransfer
	loadSentStoreErr            = "failed to load sent transfer info from storage: %+v"
	loadSentFpVectorErr         = "failed to load sent fingerprint vector from storage: %+v"
	loadSentPartStoreErr        = "failed to load sent part store from storage: %+v"
	loadInProgressTransfersErr  = "failed to load in-progress transfers bundle from storage: %+v"
	loadFinishedTransfersErr    = "failed to load finished transfers bundle from storage: %+v"
	loadSentInProgressVectorErr = "failed to load new in-progress status state vector from storage: %+v"
	loadSentFinishedVectorErr   = "failed to load new finished status state vector from storage: %+v"

	// SentTransfer.delete
	deleteSentTransferInfoErr     = "failed to delete sent transfer info from storage: %+v"
	deleteSentFpVectorErr         = "failed to delete sent fingerprint vector from storage: %+v"
	deleteSentFilePartsErr        = "failed to delete sent file parts from storage: %+v"
	deleteInProgressTransfersErr  = "failed to delete in-progress transfers from storage: %+v"
	deleteFinishedTransfersErr    = "failed to delete finished transfers from storage: %+v"
	deleteSentInProgressVectorErr = "failed to delete in-progress status state vector from storage: %+v"
	deleteSentFinishedVectorErr   = "failed to delete finished status state vector from storage: %+v"

	// SentTransfer.FinishTransfer
	noPartsForRoundErr       = "no file parts in-progress on round %d"
	deleteInProgressPartsErr = "failed to remove file parts on round %d from in-progress: %+v"

	// SentTransfer.GetEncryptedPart
	noPartNumErr   = "no part with part number %d exists"
	maxRetriesErr  = "maximum number of retries reached"
	fingerprintErr = "could not get fingerprint: %+v"
	encryptPartErr = "failed to encrypt file part #%d: %+v"
)

// MaxRetriesErr is returned as an error when number of file part sending
// retries runs out. This occurs when all the fingerprints in a transfer have
// been used.
var MaxRetriesErr = errors.New(maxRetriesErr)

// SentTransfer contains information and progress data for sending and in-
// progress file transfer.
type SentTransfer struct {
	// ID of the recipient of the file transfer
	recipient *id.ID

	// The transfer key is a randomly generated key created by the sender and
	// used to generate MACs and fingerprints
	key ftCrypto.TransferKey

	// The number of file parts in the file
	numParts uint16

	// The number of fingerprints to generate (function of numParts and the
	// retry rate)
	numFps uint16

	// Stores the state of a fingerprint (used/unused) in a bitstream format
	// (has its own storage backend)
	fpVector *utility.StateVector

	// List of all file parts in order to send (has its own storage backend)
	sentParts *partStore

	// List of parts per round that are currently transferring
	inProgressTransfers *transferredBundle

	// List of parts per round that finished transferring
	finishedTransfers *transferredBundle

	// Stores the in-progress status for each file part in a bitstream format
	inProgressStatus *utility.StateVector

	// Stores the finished status for each file part in a bitstream format
	finishedStatus *utility.StateVector

	// List of callbacks to call for every send
	progressCallbacks []*sentCallbackTracker

	// status indicates that the transfer is either done or errored out and
	// that no more callbacks should be called
	status transferStatus

	mux sync.RWMutex
	kv  *versioned.KV
}

type transferStatus int

const (
	running transferStatus = iota
	stopping
	stopped
)

// NewSentTransfer generates a new SentTransfer with the specified transfer key,
// transfer ID, and number of parts.
func NewSentTransfer(recipient *id.ID, tid ftCrypto.TransferID,
	key ftCrypto.TransferKey, parts [][]byte, numFps uint16,
	progressCB interfaces.SentProgressCallback, period time.Duration,
	kv *versioned.KV) (*SentTransfer, error) {

	// Create the SentTransfer object
	st := &SentTransfer{
		recipient:         recipient,
		key:               key,
		numParts:          uint16(len(parts)),
		numFps:            numFps,
		progressCallbacks: []*sentCallbackTracker{},
		status:            running,
		kv:                kv.Prefix(makeSentTransferPrefix(tid)),
	}

	var err error

	// Create new StateVector for storing fingerprint usage
	st.fpVector, err = utility.NewStateVector(
		st.kv, sentFpVectorKey, uint32(numFps))
	if err != nil {
		return nil, errors.Errorf(newSentTransferFpVectorErr, err)
	}

	// Create new part store
	st.sentParts, err = newPartStoreFromParts(st.kv, parts...)
	if err != nil {
		return nil, errors.Errorf(newSentTransferPartStoreErr, err)
	}

	// Create new in-progress transfer bundle
	st.inProgressTransfers, err = newTransferredBundle(inProgressKey, st.kv)
	if err != nil {
		return nil, errors.Errorf(newInProgressTransfersErr, err)
	}

	// Create new finished transfer bundle
	st.finishedTransfers, err = newTransferredBundle(finishedKey, st.kv)
	if err != nil {
		return nil, errors.Errorf(newFinishedTransfersErr, err)
	}

	// Create new StateVector for storing in-progress status
	st.inProgressStatus, err = utility.NewStateVector(
		st.kv, sentInProgressVectorKey, uint32(st.numParts))
	if err != nil {
		return nil, errors.Errorf(newSentInProgressVectorErr, err)
	}

	// Create new StateVector for storing in-progress status
	st.finishedStatus, err = utility.NewStateVector(
		st.kv, sentFinishedVectorKey, uint32(st.numParts))
	if err != nil {
		return nil, errors.Errorf(newSentFinishedVectorErr, err)
	}

	// Add first progress callback
	if progressCB != nil {
		st.AddProgressCB(progressCB, period)
	}

	return st, st.saveInfo()
}

// ReInit resets the SentTransfer to its initial state so that sending can
// restart from the beginning. ReInit is used when the sent transfer runs out of
// retries and a user wants to attempt to resend the entire file again.
func (st *SentTransfer) ReInit(numFps uint16,
	progressCB interfaces.SentProgressCallback, period time.Duration) error {
	st.mux.Lock()
	defer st.mux.Unlock()
	var err error

	// Mark the status as running
	st.status = running

	// Update number of fingerprints and overwrite old fingerprint vector
	st.numFps = numFps
	st.fpVector, err = utility.NewStateVector(
		st.kv, sentFpVectorKey, uint32(numFps))
	if err != nil {
		return errors.Errorf(reInitSentTransferFpVectorErr, err)
	}

	// Overwrite in-progress transfer bundle
	st.inProgressTransfers, err = newTransferredBundle(inProgressKey, st.kv)
	if err != nil {
		return errors.Errorf(reInitInProgressTransfersErr, err)
	}

	// Overwrite finished transfer bundle
	st.finishedTransfers, err = newTransferredBundle(finishedKey, st.kv)
	if err != nil {
		return errors.Errorf(reInitFinishedTransfersErr, err)
	}

	// Overwrite in-progress status StateVector
	st.inProgressStatus, err = utility.NewStateVector(
		st.kv, sentInProgressVectorKey, uint32(st.numParts))
	if err != nil {
		return errors.Errorf(reInitSentInProgressVectorErr, err)
	}

	// Overwrite finished status StateVector
	st.finishedStatus, err = utility.NewStateVector(
		st.kv, sentFinishedVectorKey, uint32(st.numParts))
	if err != nil {
		return errors.Errorf(reInitSentFinishedVectorErr, err)
	}

	// Clear callbacks
	st.progressCallbacks = []*sentCallbackTracker{}

	// Add first progress callback
	if progressCB != nil {
		// Add callback
		sct := newSentCallbackTracker(progressCB, period)
		st.progressCallbacks = append(st.progressCallbacks, sct)

		// Trigger the initial call
		sct.callNowUnsafe(st, nil)
	}

	return nil
}

// GetRecipient returns the ID of the recipient of the transfer.
func (st *SentTransfer) GetRecipient() *id.ID {
	st.mux.RLock()
	defer st.mux.RUnlock()

	return st.recipient
}

// GetTransferKey returns the transfer Key for this sent transfer.
func (st *SentTransfer) GetTransferKey() ftCrypto.TransferKey {
	st.mux.RLock()
	defer st.mux.RUnlock()

	return st.key
}

// GetNumParts returns the number of file parts in this transfer.
func (st *SentTransfer) GetNumParts() uint16 {
	st.mux.RLock()
	defer st.mux.RUnlock()

	return st.numParts
}

// GetNumFps returns the number of fingerprints.
func (st *SentTransfer) GetNumFps() uint16 {
	st.mux.RLock()
	defer st.mux.RUnlock()

	return st.numFps
}

// GetNumAvailableFps returns the number of unused fingerprints.
func (st *SentTransfer) GetNumAvailableFps() uint16 {
	st.mux.RLock()
	defer st.mux.RUnlock()

	return uint16(st.fpVector.GetNumAvailable())
}

// IsPartInProgress returns true if the part has successfully been sent. Returns
// false if the part is unsent or finished sending or if the part number is
// invalid.
func (st *SentTransfer) IsPartInProgress(partNum uint16) bool {
	return st.inProgressStatus.Used(uint32(partNum))
}

// IsPartFinished returns true if the part has successfully arrived. Returns
// false if the part is unsent or in the process of sending or if the part
// number is invalid.
func (st *SentTransfer) IsPartFinished(partNum uint16) bool {
	return st.finishedStatus.Used(uint32(partNum))
}

// GetProgress returns the current progress of the transfer. Completed is true
// when all parts have arrived, sent is the number of in-progress parts, arrived
// is the number of finished parts, total is the total number of parts being
// sent, and t is a part status tracker that can be used to get the status of
// individual file parts.
func (st *SentTransfer) GetProgress() (completed bool, sent, arrived,
	total uint16, t SentPartTracker) {
	st.mux.RLock()
	defer st.mux.RUnlock()

	return st.getProgress()
}

// getProgress is the thread-unsafe helper function for GetProgress.
func (st *SentTransfer) getProgress() (completed bool, sent, arrived,
	total uint16, t SentPartTracker) {
	arrived = st.finishedTransfers.getNumParts()
	sent = st.inProgressTransfers.getNumParts()
	total = st.numParts

	if sent == 0 && arrived == total {
		completed = true
	}

	return completed, sent, arrived, total,
		NewSentPartTracker(st.inProgressStatus, st.finishedStatus)
}

// CallProgressCB calls all the progress callbacks with the most recent progress
// information.
func (st *SentTransfer) CallProgressCB(err error) {
	st.mux.Lock()

	switch st.status {
	case stopped:
		st.mux.Unlock()
		return
	case stopping:
		st.status = stopped
	}

	st.mux.Unlock()
	st.mux.RLock()
	defer st.mux.RUnlock()

	for _, cb := range st.progressCallbacks {
		cb.call(st, err)
	}
}

// AddProgressCB appends a new interfaces.SentProgressCallback to the list of
// progress callbacks to be called and calls it. The period is how often the
// callback should be called when there are updates.
func (st *SentTransfer) AddProgressCB(cb interfaces.SentProgressCallback,
	period time.Duration) {
	st.mux.Lock()

	// Add callback
	sct := newSentCallbackTracker(cb, period)
	st.progressCallbacks = append(st.progressCallbacks, sct)

	st.mux.Unlock()

	// Trigger the initial call
	sct.callNow(st, nil)
}

// GetEncryptedPart gets the specified part, encrypts it, and returns the
// encrypted part along with its MAC, padding, and fingerprint.
func (st *SentTransfer) GetEncryptedPart(partNum uint16, partSize int,
	rng csprng.Source) (encPart, mac, padding []byte, fp format.Fingerprint,
	err error) {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Lookup part
	part, exists := st.sentParts.getPart(partNum)
	if !exists {
		return nil, nil, nil, format.Fingerprint{},
			errors.Errorf(noPartNumErr, partNum)
	}

	// If all fingerprints have been used but parts still remain, then change
	// the status to stopping and return an error specifying that all the
	// retries have been used
	if st.fpVector.GetNumAvailable() < 1 {
		st.status = stopping
		return nil, nil, nil, format.Fingerprint{}, MaxRetriesErr
	}

	// Get next unused fingerprint number and mark it as used
	nextKey, err := st.fpVector.Next()
	if err != nil {
		return nil, nil, nil, format.Fingerprint{},
			errors.Errorf(fingerprintErr, err)
	}
	fpNum := uint16(nextKey)

	// Generate fingerprint
	fp = ftCrypto.GenerateFingerprint(st.key, fpNum)

	// Encrypt the file part and generate the file part MAC and padding (nonce)
	maxLengthPart := make([]byte, partSize)
	copy(maxLengthPart, part)
	encPart, mac, padding, err = ftCrypto.EncryptPart(
		st.key, maxLengthPart, fpNum, rng)
	if err != nil {
		return nil, nil, nil, format.Fingerprint{},
			errors.Errorf(encryptPartErr, partNum, err)
	}

	return encPart, mac, padding, fp, err
}

// SetInProgress adds the specified file part numbers to the in-progress
// transfers for the given round ID. Returns whether the round already exists in
// the list.
func (st *SentTransfer) SetInProgress(rid id.Round, partNums ...uint16) (error, bool) {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Set as in-progress in bundle
	_, exists := st.inProgressTransfers.getPartNums(rid)

	// Set parts as in-progress in status vector
	st.inProgressStatus.UseMany(uint16SliceToUint32Slice(partNums)...)

	return st.inProgressTransfers.addPartNums(rid, partNums...), exists
}

// GetInProgress returns a list of all part number in the in-progress transfers
// list.
func (st *SentTransfer) GetInProgress(rid id.Round) ([]uint16, bool) {
	st.mux.Lock()
	defer st.mux.Unlock()

	return st.inProgressTransfers.getPartNums(rid)
}

// UnsetInProgress removed the file part numbers from the in-progress transfers
// for the given round ID. Returns the list of part numbers that were removed
// from the list.
func (st *SentTransfer) UnsetInProgress(rid id.Round) ([]uint16, error) {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Get the list of part numbers to be removed from list
	partNums, _ := st.inProgressTransfers.getPartNums(rid)

	// Unset parts as in-progress in status vector
	st.inProgressStatus.UnuseMany(uint16SliceToUint32Slice(partNums)...)

	return partNums, st.inProgressTransfers.deletePartNums(rid)
}

// FinishTransfer moves the in-progress file parts for the given round to the
// finished list.
func (st *SentTransfer) FinishTransfer(rid id.Round) error {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Get the parts in-progress for the round ID or return an error if none
	// exist
	partNums, exists := st.inProgressTransfers.getPartNums(rid)
	if !exists {
		return errors.Errorf(noPartsForRoundErr, rid)
	}

	// Delete the parts from the in-progress list
	err := st.inProgressTransfers.deletePartNums(rid)
	if err != nil {
		return errors.Errorf(deleteInProgressPartsErr, rid, err)
	}

	// Unset parts as in-progress in status vector
	st.inProgressStatus.UnuseMany(uint16SliceToUint32Slice(partNums)...)

	// Add the parts to the finished list
	err = st.finishedTransfers.addPartNums(rid, partNums...)
	if err != nil {
		return err
	}

	// Set parts as finished in status vector
	st.finishedStatus.UseMany(uint16SliceToUint32Slice(partNums)...)

	// If all parts have been moved to the finished list, then set the status
	// to stopping
	if st.finishedTransfers.getNumParts() == st.numParts &&
		st.inProgressTransfers.getNumParts() == 0 {
		st.status = stopping
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// loadSentTransfer loads the SentTransfer with the given transfer ID from
// storage.
func loadSentTransfer(tid ftCrypto.TransferID, kv *versioned.KV) (*SentTransfer,
	error) {
	st := &SentTransfer{
		kv: kv.Prefix(makeSentTransferPrefix(tid)),
	}

	// Load transfer key and number of sent parts from storage
	err := st.loadInfo()
	if err != nil {
		return nil, errors.Errorf(loadSentStoreErr, err)
	}

	// Load the fingerprint vector from storage
	st.fpVector, err = utility.LoadStateVector(st.kv, sentFpVectorKey)
	if err != nil {
		return nil, errors.Errorf(loadSentFpVectorErr, err)
	}

	// Load sent part store from storage
	st.sentParts, err = loadPartStore(st.kv)
	if err != nil {
		return nil, errors.Errorf(loadSentPartStoreErr, err)
	}

	// Load in-progress transfer bundle from storage
	st.inProgressTransfers, err = loadTransferredBundle(inProgressKey, st.kv)
	if err != nil {
		return nil, errors.Errorf(loadInProgressTransfersErr, err)
	}

	// Load finished transfer bundle from storage
	st.finishedTransfers, err = loadTransferredBundle(finishedKey, st.kv)
	if err != nil {
		return nil, errors.Errorf(loadFinishedTransfersErr, err)
	}

	// Load the in-progress status StateVector from storage
	st.inProgressStatus, err = utility.LoadStateVector(
		st.kv, sentInProgressVectorKey)
	if err != nil {
		return nil, errors.Errorf(loadSentInProgressVectorErr, err)
	}

	// Load the finished status StateVector from storage
	st.finishedStatus, err = utility.LoadStateVector(
		st.kv, sentFinishedVectorKey)
	if err != nil {
		return nil, errors.Errorf(loadSentFinishedVectorErr, err)
	}

	return st, nil
}

// saveInfo saves all fields in SentTransfer that do not have their own storage
// (recipient ID, transfer key, number of file parts, number of fingerprints,
// and transfer status) to storage.
func (st *SentTransfer) saveInfo() error {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Create new versioned object for the SentTransfer
	obj := &versioned.Object{
		Version:   sentTransferVersion,
		Timestamp: netTime.Now(),
		Data:      st.marshal(),
	}

	// Save versioned object
	return st.kv.Set(sentTransferKey, sentTransferVersion, obj)
}

// loadInfo gets the recipient ID, transfer key, number of part, number of
// fingerprints, and transfer status from storage and saves it to the
// SentTransfer.
func (st *SentTransfer) loadInfo() error {
	vo, err := st.kv.Get(sentTransferKey, sentTransferVersion)
	if err != nil {
		return err
	}

	// Unmarshal the transfer key and numParts
	st.recipient, st.key, st.numParts, st.numFps, st.status =
		unmarshalSentTransfer(vo.Data)

	return nil
}

// delete deletes all data in the SentTransfer from storage.
func (st *SentTransfer) delete() error {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Delete sent transfer info from storage
	err := st.deleteInfo()
	if err != nil {
		return errors.Errorf(deleteSentTransferInfoErr, err)
	}

	// Delete fingerprint vector from storage
	err = st.fpVector.Delete()
	if err != nil {
		return errors.Errorf(deleteSentFpVectorErr, err)
	}

	// Delete sent file parts from storage
	err = st.sentParts.delete()
	if err != nil {
		return errors.Errorf(deleteSentFilePartsErr, err)
	}

	// Delete in-progress transfer bundles from storage
	err = st.inProgressTransfers.delete()
	if err != nil {
		return errors.Errorf(deleteInProgressTransfersErr, err)
	}

	// Delete finished transfer bundles from storage
	err = st.finishedTransfers.delete()
	if err != nil {
		return errors.Errorf(deleteFinishedTransfersErr, err)
	}

	// Delete the in-progress status StateVector from storage
	err = st.inProgressStatus.Delete()
	if err != nil {
		return errors.Errorf(deleteSentInProgressVectorErr, err)
	}

	// Delete the finished status StateVector from storage
	err = st.finishedStatus.Delete()
	if err != nil {
		return errors.Errorf(deleteSentFinishedVectorErr, err)
	}

	return nil
}

// deleteInfo removes received transfer info (recipient, transfer key,  number
// of parts, and number of fingerprints) from storage.
func (st *SentTransfer) deleteInfo() error {
	return st.kv.Delete(sentTransferKey, sentTransferVersion)
}

// marshal serializes the transfer key, numParts, and numFps.

// marshal serializes all primitive fields in SentTransfer (recipient, key,
// numParts, numFps, and status).
func (st *SentTransfer) marshal() []byte {
	// Construct the buffer to the correct size
	// (size of ID + size of key + numParts (2 bytes) + numFps (2 bytes))
	buff := bytes.NewBuffer(nil)
	buff.Grow(id.ArrIDLen + ftCrypto.TransferKeyLength + 2 + 2)

	// Write the recipient ID to the buffer
	if st.recipient != nil {
		buff.Write(st.recipient.Marshal())
	} else {
		buff.Write((&id.ID{}).Marshal())
	}

	// Write the key to the buffer
	buff.Write(st.key.Bytes())

	// Write the number of parts to the buffer
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, st.numParts)
	buff.Write(b)

	// Write the number of fingerprints to the buffer
	b = make([]byte, 2)
	binary.LittleEndian.PutUint16(b, st.numFps)
	buff.Write(b)

	// Write the transfer status to the buffer
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(st.status))
	buff.Write(b)

	// Return the serialized data
	return buff.Bytes()
}

// unmarshalSentTransfer deserializes a byte slice into the primitive fields
// of SentTransfer (recipient, key, numParts, numFps, and status).
func unmarshalSentTransfer(b []byte) (recipient *id.ID,
	key ftCrypto.TransferKey, numParts, numFps uint16, status transferStatus) {

	buff := bytes.NewBuffer(b)

	// Read the recipient ID from the buffer
	recipient = &id.ID{}
	copy(recipient[:], buff.Next(id.ArrIDLen))

	// Read the transfer key from the buffer
	key = ftCrypto.UnmarshalTransferKey(buff.Next(ftCrypto.TransferKeyLength))

	// Read the number of part from the buffer
	numParts = binary.LittleEndian.Uint16(buff.Next(2))

	// Read the number of fingerprints from the buffer
	numFps = binary.LittleEndian.Uint16(buff.Next(2))

	// Read the transfer status from the buffer
	status = transferStatus(binary.LittleEndian.Uint64(buff.Next(8)))

	return recipient, key, numParts, numFps, status
}

// makeSentTransferPrefix generates the unique prefix used on the key value
// store to store sent transfers for the given transfer ID.
func makeSentTransferPrefix(tid ftCrypto.TransferID) string {
	return sentTransferPrefix + tid.String()
}

// uint16SliceToUint32Slice converts a slice of uint16 to a slice of uint32.
func uint16SliceToUint32Slice(slice []uint16) []uint32 {
	newSlice := make([]uint32, len(slice))
	for i, val := range slice {
		newSlice[i] = uint32(val)
	}
	return newSlice
}
