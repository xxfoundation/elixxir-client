////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/fileTransfer/store/cypher"
	"gitlab.com/elixxir/client/v4/storage/utility"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage keys and versions.
const (
	receivedTransferStorePrefix  = "ReceivedFileTransferStore/"
	receivedTransferStoreKey     = "ReceivedTransfer"
	receivedTransferStoreVersion = 0
	receivedTransferStatusKey    = "ReceivedPartStatusVector"
	receivedPartStoreKey         = "receivedPart#"
	receivedPartStoreVersion     = 0
)

// Error messages.
const (
	// newReceivedTransfer
	errRtNewCypherManager       = "failed to create new cypher manager: %+v"
	errRtNewPartStatusVectorErr = "failed to create new state vector for part statuses: %+v"

	// ReceivedTransfer.AddPart
	errPartOutOfRange   = "part number %d out of range of max %d"
	errReceivedPartSave = "failed to save part #%d to storage: %+v"

	// loadReceivedTransfer
	errRtLoadCypherManager    = "failed to load cypher manager from storage: %+v"
	errRtLoadFields           = "failed to load transfer MAC, number of parts, and file size: %+v"
	errRtUnmarshalFields      = "failed to unmarshal transfer MAC, number of parts, and file size: %+v"
	errRtLoadPartStatusVector = "failed to load state vector for part statuses: %+v"
	errRtLoadPart             = "[FT] Failed to load part #%d from storage: %+v"

	// ReceivedTransfer.Delete
	errRtDeleteCypherManager = "failed to delete cypher manager: %+v"
	errRtDeleteSentTransfer  = "failed to delete transfer MAC, number of parts, and file size: %+v"
	errRtDeletePartStatus    = "failed to delete part status state vector: %+v"

	// ReceivedTransfer.save
	errMarshalReceivedTransfer = "failed to marshal: %+v"
)

// ReceivedTransfer contains information and progress data for a receiving or
// received file transfer.
type ReceivedTransfer struct {
	// Tracks file part cyphers
	cypherManager *cypher.Manager

	// The ID of the transfer
	tid *ftCrypto.TransferID

	// User given name to file
	fileName string

	// The MAC for the entire file; used to verify the integrity of all parts
	transferMAC []byte

	// Size of the entire file in bytes
	fileSize uint32

	// The number of file parts in the file
	numParts uint16

	// Saves each part in order (has its own storage backend)
	parts [][]byte

	// Stores the received status for each file part in a bitstream format
	partStatus *utility.StateVector

	// Unique identifier of the last progress callback called (used to prevent
	// callback calls with duplicate data)
	lastCallbackFingerprint string

	mux sync.RWMutex
	kv  versioned.KV
}

// newReceivedTransfer generates a ReceivedTransfer with the specified transfer
// key, transfer ID, and a number of parts.
func newReceivedTransfer(key *ftCrypto.TransferKey, tid *ftCrypto.TransferID,
	fileName string, transferMAC []byte, fileSize uint32, numParts,
	numFps uint16, kv versioned.KV) (*ReceivedTransfer, error) {
	kv, err := kv.Prefix(makeReceivedTransferPrefix(tid))
	if err != nil {
		return nil, err
	}

	// Create new cypher manager
	cypherManager, err := cypher.NewManager(key, numFps, kv)
	if err != nil {
		return nil, errors.Errorf(errRtNewCypherManager, err)
	}

	// Create new state vector for storing statuses of received parts
	partStatus, err := utility.NewStateVector(
		uint32(numParts), false, receivedTransferStatusKey, kv)
	if err != nil {
		return nil, errors.Errorf(errRtNewPartStatusVectorErr, err)
	}

	rt := &ReceivedTransfer{
		cypherManager: cypherManager,
		tid:           tid,
		fileName:      fileName,
		transferMAC:   transferMAC,
		fileSize:      fileSize,
		numParts:      numParts,
		parts:         make([][]byte, numParts),
		partStatus:    partStatus,
		kv:            kv,
	}

	return rt, rt.save()
}

// AddPart adds the file part to the list of file parts at the index of partNum.
func (rt *ReceivedTransfer) AddPart(part []byte, partNum int) error {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	if partNum > len(rt.parts)-1 {
		return errors.Errorf(errPartOutOfRange, partNum, len(rt.parts)-1)
	}

	// Save part
	rt.parts[partNum] = part
	err := savePart(part, partNum, rt.kv)
	if err != nil {
		return errors.Errorf(errReceivedPartSave, partNum, err)
	}

	// Mark part as received
	rt.partStatus.Use(uint32(partNum))

	return nil
}

// GetFile concatenates all file parts and returns it as a single complete file.
// Note that this function does not care for the completeness of the file and
// returns all parts it has.
func (rt *ReceivedTransfer) GetFile() []byte {
	rt.mux.RLock()
	defer rt.mux.RUnlock()

	file := bytes.Join(rt.parts, nil)

	// Strip off trailing padding from last part
	if len(file) > int(rt.fileSize) {
		file = file[:rt.fileSize]
	}

	return file
}

// GetUnusedCyphers returns a list of cyphers with unused fingerprint numbers.
func (rt *ReceivedTransfer) GetUnusedCyphers() []cypher.Cypher {
	return rt.cypherManager.GetUnusedCyphers()
}

// TransferID returns the transfer's ID.
func (rt *ReceivedTransfer) TransferID() *ftCrypto.TransferID {
	return rt.tid
}

// FileName returns the transfer's file name.
func (rt *ReceivedTransfer) FileName() string {
	return rt.fileName
}

// FileSize returns the size of the entire file transfer.
func (rt *ReceivedTransfer) FileSize() uint32 {
	return rt.fileSize
}

// NumParts returns the total number of file parts in the transfer.
func (rt *ReceivedTransfer) NumParts() uint16 {
	return rt.numParts
}

// NumReceived returns the number of parts that have been received.
func (rt *ReceivedTransfer) NumReceived() uint16 {
	rt.mux.RLock()
	defer rt.mux.RUnlock()
	return uint16(rt.partStatus.GetNumUsed())
}

// CopyPartStatusVector returns a copy of the part status vector that can be
// used to look up the current status of parts. Note that the statuses are from
// when this function is called and not realtime.
func (rt *ReceivedTransfer) CopyPartStatusVector() *utility.StateVector {
	return rt.partStatus.DeepCopy()
}

// CompareAndSwapCallbackFps compares the fingerprint to the previous callback
// call's fingerprint. If they are different, the new one is stored, and it
// returns true. Returns fall if they are the same.
func (rt *ReceivedTransfer) CompareAndSwapCallbackFps(
	completed bool, received, total uint16, err error) bool {
	fp := generateReceivedFp(completed, received, total, err)

	rt.mux.Lock()
	defer rt.mux.Unlock()

	if fp != rt.lastCallbackFingerprint {
		rt.lastCallbackFingerprint = fp
		return true
	}

	return false
}

// generateReceivedFp generates a fingerprint for a received progress callback.
func generateReceivedFp(completed bool, received, total uint16, err error) string {
	errString := "<nil>"
	if err != nil {
		errString = err.Error()
	}

	return strconv.FormatBool(completed) +
		strconv.FormatUint(uint64(received), 10) +
		strconv.FormatUint(uint64(total), 10) +
		errString
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// loadReceivedTransfer loads the ReceivedTransfer with the given transfer ID
// from storage.
func loadReceivedTransfer(tid *ftCrypto.TransferID, kv versioned.KV) (
	*ReceivedTransfer, error) {
	kv, err := kv.Prefix(makeReceivedTransferPrefix(tid))
	if err != nil {
		return nil, err
	}

	// Load cypher manager
	cypherManager, err := cypher.LoadManager(kv)
	if err != nil {
		return nil, errors.Errorf(errRtLoadCypherManager, err)
	}

	// Load transfer MAC, number of parts, and file size
	obj, err := kv.Get(receivedTransferStoreKey, receivedTransferStoreVersion)
	if err != nil {
		return nil, errors.Errorf(errRtLoadFields, err)
	}

	fileName, transferMAC, numParts, fileSize, err :=
		unmarshalReceivedTransfer(obj.Data)
	if err != nil {
		return nil, errors.Errorf(errRtUnmarshalFields, err)
	}

	// Load StateVector for storing statuses of received parts
	partStatus, err := utility.LoadStateVector(kv, receivedTransferStatusKey)
	if err != nil {
		return nil, errors.Errorf(errRtLoadPartStatusVector, err)
	}

	// Load parts from storage
	parts := make([][]byte, numParts)
	for i := range parts {
		if partStatus.Used(uint32(i)) {
			parts[i], err = loadPart(i, kv)
			if err != nil {
				jww.ERROR.Printf(errRtLoadPart, i, err)
			}
		}
	}

	rt := &ReceivedTransfer{
		cypherManager: cypherManager,
		tid:           tid,
		fileName:      fileName,
		transferMAC:   transferMAC,
		fileSize:      fileSize,
		numParts:      numParts,
		parts:         parts,
		partStatus:    partStatus,
		kv:            kv,
	}

	return rt, nil
}

// Delete deletes all data in the ReceivedTransfer from storage.
func (rt *ReceivedTransfer) Delete() error {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	// Delete cypher manager
	err := rt.cypherManager.Delete()
	if err != nil {
		return errors.Errorf(errRtDeleteCypherManager, err)
	}

	// Delete transfer MAC, number of parts, and file size
	err = rt.kv.Delete(receivedTransferStoreKey, receivedTransferStoreVersion)
	if err != nil {
		return errors.Errorf(errRtDeleteSentTransfer, err)
	}

	// Delete part status state vector
	err = rt.partStatus.Delete()
	if err != nil {
		return errors.Errorf(errRtDeletePartStatus, err)
	}

	return nil
}

// save stores all fields in ReceivedTransfer that do not have their own storage
// (transfer MAC, file size, and number of file parts) to storage.
func (rt *ReceivedTransfer) save() error {
	data, err := rt.marshal()
	if err != nil {
		return errors.Errorf(errMarshalReceivedTransfer, err)
	}

	// Create new versioned object for the ReceivedTransfer
	vo := &versioned.Object{
		Version:   receivedTransferStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	// Save versioned object
	return rt.kv.Set(receivedTransferStoreKey, vo)
}

// receivedTransferDisk structure is used to marshal and unmarshal
// ReceivedTransfer fields to/from storage.
type receivedTransferDisk struct {
	FileName    string
	TransferMAC []byte
	NumParts    uint16
	FileSize    uint32
}

// marshal serialises the ReceivedTransfer's fileName, transferMAC, numParts,
// and fileSize.
func (rt *ReceivedTransfer) marshal() ([]byte, error) {
	disk := receivedTransferDisk{
		FileName:    rt.fileName,
		TransferMAC: rt.transferMAC,
		NumParts:    rt.numParts,
		FileSize:    rt.fileSize,
	}

	return json.Marshal(disk)
}

// unmarshalReceivedTransfer deserializes the data into the fileName,
// transferMAC, numParts, and fileSize.
func unmarshalReceivedTransfer(data []byte) (fileName string,
	transferMAC []byte, numParts uint16, fileSize uint32, err error) {
	var disk receivedTransferDisk
	err = json.Unmarshal(data, &disk)
	if err != nil {
		return "", nil, 0, 0, err
	}

	return disk.FileName, disk.TransferMAC, disk.NumParts, disk.FileSize, nil
}

// savePart saves the given part to storage keying on its part number.
func savePart(part []byte, partNum int, kv versioned.KV) error {
	obj := &versioned.Object{
		Version:   receivedPartStoreVersion,
		Timestamp: netTime.Now(),
		Data:      part,
	}

	return kv.Set(makeReceivedPartKey(partNum), obj)
}

// loadPart loads the part with the given part number from storage.
func loadPart(partNum int, kv versioned.KV) ([]byte, error) {
	obj, err := kv.Get(makeReceivedPartKey(partNum), receivedPartStoreVersion)
	if err != nil {
		return nil, err
	}
	return obj.Data, nil
}

// makeReceivedTransferPrefix generates the unique prefix used on the key value
// store to store received transfers for the given transfer ID.
func makeReceivedTransferPrefix(tid *ftCrypto.TransferID) string {
	return receivedTransferStorePrefix +
		base64.StdEncoding.EncodeToString(tid.Bytes())
}

// makeReceivedPartKey generates a storage key for the given part number.
func makeReceivedPartKey(partNum int) string {
	return receivedPartStoreKey + strconv.Itoa(partNum)
}
