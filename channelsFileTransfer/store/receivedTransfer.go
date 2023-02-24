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
	"encoding/binary"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"gitlab.com/elixxir/client/v4/channelsFileTransfer/store/cypher"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
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
	errRtUnmarshalPartialFile = "failed to unmarshal partial file: %+v"

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
	fid ftCrypto.ID

	// User given name to file
	fileName string

	// ID of the recipient of the file transfer
	recipient *id.ID

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

	// Current ID to assign to a callback
	currentCallbackID uint64

	// Unique identifier of the last progress callback called (used to prevent
	// callback calls with duplicate data)
	lastCallbackFingerprints map[uint64]string

	mux       sync.RWMutex
	disableKV bool // Toggles use of KV storage
	kv        *versioned.KV
}

// newReceivedTransfer generates a ReceivedTransfer with the specified transfer
// key, file ID, and a number of parts.
func newReceivedTransfer(recipient *id.ID, key *ftCrypto.TransferKey,
	fid ftCrypto.ID, fileName string, transferMAC []byte, fileSize uint32,
	numParts, numFps uint16, disableKV bool, kv *versioned.KV) (
	*ReceivedTransfer, error) {
	kv = kv.Prefix(makeReceivedTransferPrefix(fid))

	// Create new cypher manager
	cypherManager, err := cypher.NewManager(key, numFps, false, kv)
	if err != nil {
		return nil, errors.Errorf(errRtNewCypherManager, err)
	}

	// Create new state vector for storing statuses of received parts
	partStatus, err := utility.NewStateVector(
		uint32(numParts), disableKV, receivedTransferStatusKey, kv)
	if err != nil {
		return nil, errors.Errorf(errRtNewPartStatusVectorErr, err)
	}

	rt := &ReceivedTransfer{
		cypherManager:            cypherManager,
		fid:                      fid,
		fileName:                 fileName,
		recipient:                recipient,
		transferMAC:              transferMAC,
		fileSize:                 fileSize,
		numParts:                 numParts,
		parts:                    make([][]byte, numParts),
		partStatus:               partStatus,
		currentCallbackID:        0,
		lastCallbackFingerprints: make(map[uint64]string),
		disableKV:                disableKV,
		kv:                       kv,
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
	if !rt.disableKV {
		err := savePart(part, partNum, rt.kv)
		if err != nil {
			return errors.Errorf(errReceivedPartSave, partNum, err)
		}
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
	return rt.getFile()
}

func (rt *ReceivedTransfer) getFile() []byte {
	file := bytes.Join(rt.parts, nil)

	// Strip off trailing padding from last part
	if len(file) > int(rt.fileSize) {
		file = file[:rt.fileSize]
	}

	return file
}

// MarshalPartialFile returns the file as-is with data on which parts have been
// received.
func (rt *ReceivedTransfer) MarshalPartialFile() ([]byte, error) {
	rt.mux.RLock()
	defer rt.mux.RUnlock()

	buff := bytes.NewBuffer(nil)

	// Get partial file
	partialFile := rt.getFile()

	// Write length of partial file to buffer
	err := binary.Write(buff, binary.LittleEndian, uint64(len(partialFile)))
	if err != nil {
		return nil, errors.Errorf(
			"failed to write file length to buffer: %+v", err)
	}

	// Write partial file to buffer
	buff.Write(partialFile)

	// Write JSON encoded part status vector to buffer
	err = json.NewEncoder(buff).Encode(rt.partStatus)
	if err != nil {
		return nil, errors.Errorf(
			"failed to JSON marshal part statuses: %+v", err)
	}

	return buff.Bytes(), nil
}

// unmarshalPartialFile unmarshalls the data into a list of parts and the
// accompanying part statuses vector.
func (rt *ReceivedTransfer) unmarshalPartialFile(data []byte, partSize int) (
	[][]byte, *utility.StateVector, error) {

	buff := bytes.NewBuffer(data)

	// Read file length from buffer
	var fileLen uint64
	err := binary.Read(buff, binary.LittleEndian, &fileLen)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read file length")
	}

	// Read partial file from buffer
	partialFile := make([]byte, fileLen)
	n, err := buff.Read(partialFile)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read file")
	} else if n != int(fileLen) {
		return nil, nil, errors.Errorf(
			"read %d bytes for file; %d bytes expected", n, fileLen)
	}

	// JSON decode part status vector from buffer
	partStatus, _ := utility.NewStateVector(
		0, rt.disableKV, receivedTransferStatusKey, rt.kv)
	err = json.NewDecoder(buff).Decode(partStatus)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read part status vector")
	}

	// Split partial file into file parts
	parts := make([][]byte, partStatus.GetNumKeys())
	buff = bytes.NewBuffer(partialFile)
	for i := uint32(0); i < partStatus.GetNumKeys(); i++ {
		if partStatus.Used(i) {
			parts[i] = buff.Next(partSize)
		}
	}

	return parts, partStatus, nil
}

// GetUnusedCyphers returns a list of cyphers with unused fingerprint numbers.
func (rt *ReceivedTransfer) GetUnusedCyphers() []cypher.Cypher {
	return rt.cypherManager.GetUnusedCyphers()
}

// FileID returns the file's ID.
func (rt *ReceivedTransfer) FileID() ftCrypto.ID {
	return rt.fid
}

// FileName returns the transfer's file name.
func (rt *ReceivedTransfer) FileName() string {
	return rt.fileName
}

// Recipient returns the transfer's recipient ID.
func (rt *ReceivedTransfer) Recipient() *id.ID {
	return rt.recipient
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

// GetNewCallbackID issues a new unique for a callback.
func (rt *ReceivedTransfer) GetNewCallbackID() uint64 {
	rt.mux.Lock()
	defer rt.mux.Unlock()
	newID := rt.currentCallbackID
	rt.currentCallbackID++
	return newID
}

// CompareAndSwapCallbackFps compares the fingerprint to the previous callback
// call's fingerprint. If they are different, the new one is stored, and it
// returns true. Returns false if they are the same.
func (rt *ReceivedTransfer) CompareAndSwapCallbackFps(callbackID uint64,
	completed bool, received, total uint16, err error) bool {
	fp := GenerateReceivedFp(completed, received, total, err)

	rt.mux.Lock()
	defer rt.mux.Unlock()

	if fp != rt.lastCallbackFingerprints[callbackID] {
		rt.lastCallbackFingerprints[callbackID] = fp
		return true
	}

	return false
}

// GenerateReceivedFp generates a fingerprint for a received progress callback.
func GenerateReceivedFp(completed bool, received, total uint16, err error) string {
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

// loadReceivedTransfer loads the ReceivedTransfer with the given file ID from
// storage.
func loadReceivedTransfer(fid ftCrypto.ID, partialFile []byte, partSize int,
	kv *versioned.KV) (*ReceivedTransfer, error) {
	kv = kv.Prefix(makeReceivedTransferPrefix(fid))

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

	info, err := unmarshalReceivedTransfer(obj.Data)
	if err != nil {
		return nil, errors.Errorf(errRtUnmarshalFields, err)
	}

	rt := &ReceivedTransfer{
		cypherManager:            cypherManager,
		fid:                      fid,
		fileName:                 info.FileName,
		recipient:                info.Recipient,
		transferMAC:              info.TransferMAC,
		fileSize:                 info.FileSize,
		numParts:                 info.NumParts,
		currentCallbackID:        0,
		lastCallbackFingerprints: make(map[uint64]string),
		disableKV:                false,
		kv:                       kv,
	}

	// Load StateVector for storing statuses of received parts
	rt.parts, rt.partStatus, err = rt.unmarshalPartialFile(partialFile, partSize)
	if err != nil {
		return nil, errors.Errorf(errRtUnmarshalPartialFile, err)
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

	if !rt.disableKV {
		// Delete transfer MAC, number of parts, and file size
		err = rt.kv.Delete(receivedTransferStoreKey, receivedTransferStoreVersion)
		if err != nil {
			return errors.Errorf(errRtDeleteSentTransfer, err)
		}
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
	if rt.disableKV {
		return nil
	}

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
	FileName    string `json:"fileName"`
	Recipient   *id.ID `json:"recipient"`
	TransferMAC []byte `json:"transferMAC"`
	NumParts    uint16 `json:"numParts"`
	FileSize    uint32 `json:"fileSize"`
}

// marshal serialises the ReceivedTransfer's file information.
func (rt *ReceivedTransfer) marshal() ([]byte, error) {
	disk := receivedTransferDisk{
		FileName:    rt.fileName,
		Recipient:   rt.recipient,
		TransferMAC: rt.transferMAC,
		NumParts:    rt.numParts,
		FileSize:    rt.fileSize,
	}

	return json.Marshal(disk)
}

// unmarshalReceivedTransfer deserializes the ReceivedTransfer's file
// information.
func unmarshalReceivedTransfer(data []byte) (receivedTransferDisk, error) {
	var disk receivedTransferDisk
	return disk, json.Unmarshal(data, &disk)
}

// savePart saves the given part to storage keying on its part number.
func savePart(part []byte, partNum int, kv *versioned.KV) error {
	obj := &versioned.Object{
		Version:   receivedPartStoreVersion,
		Timestamp: netTime.Now(),
		Data:      part,
	}

	return kv.Set(makeReceivedPartKey(partNum), obj)
}

// loadPart loads the part with the given part number from storage.
func loadPart(partNum int, kv *versioned.KV) ([]byte, error) {
	obj, err := kv.Get(makeReceivedPartKey(partNum), receivedPartStoreVersion)
	if err != nil {
		return nil, err
	}
	return obj.Data, nil
}

// makeReceivedTransferPrefix generates the unique prefix used on the key value
// store to store received transfers for the given file ID.
func makeReceivedTransferPrefix(fid ftCrypto.ID) string {
	return receivedTransferStorePrefix +
		base64.StdEncoding.EncodeToString(fid.Marshal())
}

// makeReceivedPartKey generates a storage key for the given part number.
func makeReceivedPartKey(partNum int) string {
	return receivedPartStoreKey + strconv.Itoa(partNum)
}
