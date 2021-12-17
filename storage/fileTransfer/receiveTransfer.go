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
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

// Storage constants
const (
	receivedTransfersPrefix = "FileTransferReceivedTransferStore"
	receivedTransferKey     = "ReceivedTransfer"
	receivedTransferVersion = 0
	receivedFpVectorKey     = "ReceivedFingerprintVector"
	receivedVectorKey       = "ReceivedStatusVector"
)

// Error messages for ReceivedTransfer
const (
	// NewReceivedTransfer
	newReceivedTransferFpVectorErr  = "failed to create new StateVector for fingerprints: %+v"
	newReceivedTransferPartStoreErr = "failed to create new part store: %+v"
	newReceivedVectorErr            = "failed to create new state vector for received status: %+v"

	// ReceivedTransfer.GetFile
	getFileErr        = "missing %d/%d parts of the file"
	getTransferMacErr = "failed to verify transfer MAC"

	// loadReceivedTransfer
	loadReceivedStoreErr    = "failed to load received transfer info from storage: %+v"
	loadReceivePartStoreErr = "failed to load received part store from storage: %+v"
	loadReceivedVectorErr   = "failed to load new received status state vector from storage: %+v"
	loadReceiveFpVectorErr  = "failed to load received fingerprint vector from storage: %+v"

	// ReceivedTransfer.delete
	deleteReceivedTransferInfoErr = "failed to delete received transfer info from storage: %+v"
	deleteReceivedFpVectorErr     = "failed to delete received fingerprint vector from storage: %+v"
	deleteReceivedFilePartsErr    = "failed to delete received file parts from storage: %+v"
	deleteReceivedVectorErr       = "failed to delete received status state vector from storage: %+v"

	// ReceivedTransfer.StopScheduledProgressCB
	cancelReceivedCallbacksErr = "could not cancel %d out of %d received progress callbacks: %d"
)

// ReceivedTransfer contains information and progress data for receiving an in-
// progress file transfer.
type ReceivedTransfer struct {
	// The transfer key is a randomly generated key created by the sender and
	// used to generate MACs and fingerprints
	key ftCrypto.TransferKey

	// The MAC for the entire file; used to verify the integrity of all parts
	transferMAC []byte

	// Size of the entire file in bytes
	fileSize uint32

	// The number of file parts in the file
	numParts uint16

	// The number `of fingerprints to generate (function of numParts and the
	// retry rate)
	numFps uint16

	// Stores the state of a fingerprint (used/unused) in a bitstream format
	// (has its own storage backend)
	fpVector *utility.StateVector

	// Saves each part in order (has its own storage backend)
	receivedParts *partStore

	// Stores the received status for each file part in a bitstream format
	receivedStatus *utility.StateVector

	// List of callbacks to call for every send
	progressCallbacks []*receivedCallbackTracker

	mux sync.RWMutex
	kv  *versioned.KV
}

// NewReceivedTransfer generates a ReceivedTransfer with the specified
// transfer key, transfer ID, and a number of parts.
func NewReceivedTransfer(tid ftCrypto.TransferID, key ftCrypto.TransferKey,
	transferMAC []byte, fileSize uint32, numParts, numFps uint16,
	kv *versioned.KV) (*ReceivedTransfer, error) {

	// Create the ReceivedTransfer object
	rt := &ReceivedTransfer{
		key:               key,
		transferMAC:       transferMAC,
		fileSize:          fileSize,
		numParts:          numParts,
		numFps:            numFps,
		progressCallbacks: []*receivedCallbackTracker{},
		kv:                kv.Prefix(makeReceivedTransferPrefix(tid)),
	}

	var err error

	// Create new StateVector for storing fingerprint usage
	rt.fpVector, err = utility.NewStateVector(
		rt.kv, receivedFpVectorKey, uint32(numFps))
	if err != nil {
		return nil, errors.Errorf(newReceivedTransferFpVectorErr, err)
	}

	// Create new part store
	rt.receivedParts, err = newPartStore(rt.kv, numParts)
	if err != nil {
		return nil, errors.Errorf(newReceivedTransferPartStoreErr, err)
	}

	// Create new StateVector for storing received status
	rt.receivedStatus, err = utility.NewStateVector(
		rt.kv, receivedVectorKey, uint32(rt.numParts))
	if err != nil {
		return nil, errors.Errorf(newReceivedVectorErr, err)
	}

	// Save all fields without their own storage to storage
	return rt, rt.saveInfo()
}

// GetTransferKey returns the transfer Key for this received transfer.
func (rt *ReceivedTransfer) GetTransferKey() ftCrypto.TransferKey {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	return rt.key
}

// GetTransferMAC returns the transfer MAC for this received transfer.
func (rt *ReceivedTransfer) GetTransferMAC() []byte {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	return rt.transferMAC
}

// GetNumParts returns the number of file parts in this transfer.
func (rt *ReceivedTransfer) GetNumParts() uint16 {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	return rt.numParts
}

// GetNumFps returns the number of fingerprints.
func (rt *ReceivedTransfer) GetNumFps() uint16 {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	return rt.numFps
}

// GetNumAvailableFps returns the number of unused fingerprints.
func (rt *ReceivedTransfer) GetNumAvailableFps() uint16 {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	return uint16(rt.fpVector.GetNumAvailable())
}

// GetFileSize returns the file size in bytes.
func (rt *ReceivedTransfer) GetFileSize() uint32 {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	return rt.fileSize
}

// IsPartReceived returns true if the part has successfully been received.
// Returns false if the part has not been received or if the part number is
// invalid.
func (rt *ReceivedTransfer) IsPartReceived(partNum uint16) bool {
	_, exists := rt.receivedParts.getPart(partNum)
	return exists
}

// GetProgress returns the current progress of the transfer. Completed is true
// when all parts have been received, received is the number of parts received,
// total is the total number of parts excepted to be received, and t is a part
// status tracker that can be used to get the status of individual file parts.
func (rt *ReceivedTransfer) GetProgress() (completed bool, received,
	total uint16, t ReceivedPartTracker) {
	rt.mux.RLock()
	defer rt.mux.RUnlock()

	completed, received, total, t = rt.getProgress()
	return completed, received, total, t
}

// getProgress is the thread-unsafe helper function for GetProgress.
func (rt *ReceivedTransfer) getProgress() (completed bool, received,
	total uint16, t ReceivedPartTracker) {

	received = uint16(rt.receivedStatus.GetNumUsed())
	total = rt.numParts

	if received == total {
		completed = true
	}

	return completed, received, total, NewReceivedPartTracker(rt.receivedStatus)
}

// CallProgressCB calls all the progress callbacks with the most recent progress
// information.
func (rt *ReceivedTransfer) CallProgressCB(err error) {
	rt.mux.RLock()
	defer rt.mux.RUnlock()

	for _, cb := range rt.progressCallbacks {
		cb.call(rt, err)
	}
}

// StopScheduledProgressCB cancels all scheduled received progress callbacks
// calls.
func (rt *ReceivedTransfer) StopScheduledProgressCB() error {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	// Tracks the index of callbacks that failed to stop
	var failedCallbacks []int

	for i, cb := range rt.progressCallbacks {
		err := cb.stopThread()
		if err != nil {
			failedCallbacks = append(failedCallbacks, i)
			jww.WARN.Print(err.Error())
		}
	}

	if len(failedCallbacks) > 0 {
		return errors.Errorf(cancelReceivedCallbacksErr, len(failedCallbacks),
			len(rt.progressCallbacks), failedCallbacks)
	}

	return nil
}

// AddProgressCB appends a new interfaces.ReceivedProgressCallback to the list
// of progress callbacks to be called and calls it. The period is how often the
// callback should be called when there are updates.
func (rt *ReceivedTransfer) AddProgressCB(
	cb interfaces.ReceivedProgressCallback, period time.Duration) {
	rt.mux.Lock()

	// Add callback
	rct := newReceivedCallbackTracker(cb, period)
	rt.progressCallbacks = append(rt.progressCallbacks, rct)

	rt.mux.Unlock()

	// Trigger the initial call
	rct.callNow(rt, nil)
}

// AddPart decrypts an encrypted file part, adds it to the list of received
// parts and marks its fingerprint as used.
func (rt *ReceivedTransfer) AddPart(encryptedPart, padding, mac []byte, partNum,
	fpNum uint16) error {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	// Decrypt the encrypted file part
	decryptedPart, err := ftCrypto.DecryptPart(
		rt.key, encryptedPart, padding, mac, fpNum)
	if err != nil {
		return err
	}

	// Add the part to the list of parts
	err = rt.receivedParts.addPart(decryptedPart, partNum)
	if err != nil {
		return err
	}

	// Mark the fingerprint as used
	rt.fpVector.Use(uint32(fpNum))

	// Mark part as received
	rt.receivedStatus.Use(uint32(partNum))

	return nil
}

// GetFile returns all the file parts combined into a single byte slice. An
// error is returned if parts are missing or if the MAC cannot be verified. The
// incomplete or invalid file is returned despite errors.
func (rt *ReceivedTransfer) GetFile() ([]byte, error) {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	fileData, missingParts := rt.receivedParts.getFile()
	if missingParts != 0 {
		return fileData, errors.Errorf(getFileErr, missingParts, rt.numParts)
	}

	// Remove extra data added when sending as parts
	fileData = fileData[:rt.fileSize]

	if !ftCrypto.VerifyTransferMAC(fileData, rt.key, rt.transferMAC) {
		return fileData, errors.New(getTransferMacErr)
	}

	return fileData, nil
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// loadReceivedTransfer loads the ReceivedTransfer with the given transfer ID
// from storage.
func loadReceivedTransfer(tid ftCrypto.TransferID, kv *versioned.KV) (
	*ReceivedTransfer, error) {
	// Create the ReceivedTransfer object
	rt := &ReceivedTransfer{
		progressCallbacks: []*receivedCallbackTracker{},
		kv:                kv.Prefix(makeReceivedTransferPrefix(tid)),
	}

	// Load transfer key and number of received parts from storage
	err := rt.loadInfo()
	if err != nil {
		return nil, errors.Errorf(loadReceivedStoreErr, err)
	}

	// Load the fingerprint vector from storage
	rt.fpVector, err = utility.LoadStateVector(rt.kv, receivedFpVectorKey)
	if err != nil {
		return nil, errors.Errorf(loadReceiveFpVectorErr, err)
	}

	// Load received part store from storage
	rt.receivedParts, err = loadPartStore(rt.kv)
	if err != nil {
		return nil, errors.Errorf(loadReceivePartStoreErr, err)
	}

	// Load the received status StateVector from storage
	rt.receivedStatus, err = utility.LoadStateVector(rt.kv, receivedVectorKey)
	if err != nil {
		return nil, errors.Errorf(loadReceivedVectorErr, err)
	}

	return rt, nil
}

// saveInfo saves all fields in ReceivedTransfer that do not have their own
// storage (transfer key, transfer MAC, file size, number of file parts, and
// number of fingerprints) to storage.
func (rt *ReceivedTransfer) saveInfo() error {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	// Create new versioned object for the ReceivedTransfer
	vo := &versioned.Object{
		Version:   receivedTransferVersion,
		Timestamp: netTime.Now(),
		Data:      rt.marshal(),
	}

	// Save versioned object
	return rt.kv.Set(receivedTransferKey, receivedTransferVersion, vo)
}

// loadInfo gets the transfer key, transfer MAC, file size, number of part, and
// number of fingerprints from storage and saves it to the ReceivedTransfer.
func (rt *ReceivedTransfer) loadInfo() error {
	vo, err := rt.kv.Get(receivedTransferKey, receivedTransferVersion)
	if err != nil {
		return err
	}

	// Unmarshal the transfer key and numParts
	rt.key, rt.transferMAC, rt.fileSize, rt.numParts, rt.numFps =
		unmarshalReceivedTransfer(vo.Data)

	return nil
}

// delete deletes all data in the ReceivedTransfer from storage.
func (rt *ReceivedTransfer) delete() error {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	// Delete received transfer info from storage
	err := rt.deleteInfo()
	if err != nil {
		return errors.Errorf(deleteReceivedTransferInfoErr, err)
	}

	// Delete fingerprint vector from storage
	err = rt.fpVector.Delete()
	if err != nil {
		return errors.Errorf(deleteReceivedFpVectorErr, err)
	}

	// Delete received file parts from storage
	err = rt.receivedParts.delete()
	if err != nil {
		return errors.Errorf(deleteReceivedFilePartsErr, err)
	}

	// Delete the received status StateVector from storage
	err = rt.receivedStatus.Delete()
	if err != nil {
		return errors.Errorf(deleteReceivedVectorErr, err)
	}

	return nil
}

// deleteInfo removes received transfer info (transfer key, transfer MAC, file
// size, number of parts, and number of fingerprints) from storage.
func (rt *ReceivedTransfer) deleteInfo() error {
	return rt.kv.Delete(receivedTransferKey, receivedTransferVersion)
}

// marshal serializes all primitive fields in ReceivedTransfer (key,
// transferMAC, fileSize, numParts, and numFps).
func (rt *ReceivedTransfer) marshal() []byte {
	// Construct the buffer to the correct size (size of key + transfer MAC
	// length (2 bytes) + transfer MAC + fileSize (4 bytes) + numParts (2 bytes)
	// + numFps (2 bytes))
	buff := bytes.NewBuffer(nil)
	buff.Grow(ftCrypto.TransferKeyLength + 2 + len(rt.transferMAC) + 4 + 2 + 2)

	// Write the key to the buffer
	buff.Write(rt.key.Bytes())

	// Write the length of the transfer MAC to the buffer
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(len(rt.transferMAC)))
	buff.Write(b)

	// Write the transfer MAC to the buffer
	buff.Write(rt.transferMAC)

	// Write the file size to the buffer
	b = make([]byte, 4)
	binary.LittleEndian.PutUint32(b, rt.fileSize)
	buff.Write(b)

	// Write the number of parts to the buffer
	b = make([]byte, 2)
	binary.LittleEndian.PutUint16(b, rt.numParts)
	buff.Write(b)

	// Write the number of fingerprints to the buffer
	b = make([]byte, 2)
	binary.LittleEndian.PutUint16(b, rt.numFps)
	buff.Write(b)

	// Return the serialized data
	return buff.Bytes()
}

// unmarshalReceivedTransfer deserializes a byte slice into the primitive fields
// of ReceivedTransfer (key, transferMAC, fileSize, numParts, and numFps).
func unmarshalReceivedTransfer(data []byte) (key ftCrypto.TransferKey,
	transferMAC []byte, size uint32, numParts, numFps uint16) {

	buff := bytes.NewBuffer(data)

	// Read the transfer key from the buffer
	key = ftCrypto.UnmarshalTransferKey(buff.Next(ftCrypto.TransferKeyLength))

	// Read the size of the transfer MAC from the buffer
	transferMacSize := binary.LittleEndian.Uint16(buff.Next(2))

	// Read the transfer MAC from the buffer
	transferMAC = buff.Next(int(transferMacSize))

	// Read the file size from the buffer
	size = binary.LittleEndian.Uint32(buff.Next(4))

	// Read the number of part from the buffer
	numParts = binary.LittleEndian.Uint16(buff.Next(2))

	// Read the number of fingerprints from the buffer
	numFps = binary.LittleEndian.Uint16(buff.Next(2))

	return key, transferMAC, size, numParts, numFps
}

// makeReceivedTransferPrefix generates the unique prefix used on the key value
// store to store received transfers for the given transfer ID.
func makeReceivedTransferPrefix(tid ftCrypto.TransferID) string {
	return receivedTransfersPrefix + tid.String()
}
