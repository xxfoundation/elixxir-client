////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/fileTransfer/store/cypher"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"strconv"
	"sync"
)

// Storage keys and versions.
const (
	sentTransferStorePrefix  = "SentFileTransferStore/"
	sentTransferStoreKey     = "SentTransfer"
	sentTransferStoreVersion = 0
	sentTransferStatusKey    = "SentPartStatusVector"
)

// Error messages.
const (
	// newSentTransfer
	errStNewCypherManager    = "failed to create new cypher manager: %+v"
	errStNewPartStatusVector = "failed to create new state vector for part statuses: %+v"

	// SentTransfer.getPartData
	errNoPartNum = "no part with part number %d exists in transfer %s (%q)"

	// loadSentTransfer
	errStLoadCypherManager    = "failed to load cypher manager from storage: %+v"
	errStLoadFields           = "failed to load recipient, status, and parts list: %+v"
	errStUnmarshalFields      = "failed to unmarshal recipient, status, and parts list: %+v"
	errStLoadPartStatusVector = "failed to load state vector for part statuses: %+v"

	// SentTransfer.Delete
	errStDeleteCypherManager = "failed to delete cypherManager: %+v"
	errStDeleteSentTransfer  = "failed to delete recipient ID, status, and file parts: %+v"
	errStDeletePartStatus    = "failed to delete part status multi state vector: %+v"

	// SentTransfer.save
	errMarshalSentTransfer = "failed to marshal: %+v"
)

// SentTransfer contains information and progress data for sending or sent file
// transfer.
type SentTransfer struct {
	// Tracks cyphers for each part
	cypherManager *cypher.Manager

	// The ID of the transfer
	tid *ftCrypto.TransferID

	// User given name to file
	fileName string

	// ID of the recipient of the file transfer
	recipient *id.ID

	// The size of the entire file
	fileSize uint32

	// The number of file parts in the file
	numParts uint16

	// Indicates the status of the transfer
	status TransferStatus

	// List of all file parts in order to send
	parts [][]byte

	// Stores the status of each part in a bitstream format
	partStatus *utility.StateVector

	// Unique identifier of the last progress callback called (used to prevent
	// callback calls with duplicate data)
	lastCallbackFingerprint string

	mux sync.RWMutex
	kv  *versioned.KV
}

// newSentTransfer generates a new SentTransfer with the specified transfer key,
// transfer ID, and parts.
func newSentTransfer(recipient *id.ID, key *ftCrypto.TransferKey,
	tid *ftCrypto.TransferID, fileName string, fileSize uint32, parts [][]byte,
	numFps uint16, kv *versioned.KV) (*SentTransfer, error) {
	kv = kv.Prefix(makeSentTransferPrefix(tid))

	// Create new cypher manager
	cypherManager, err := cypher.NewManager(key, numFps, kv)
	if err != nil {
		return nil, errors.Errorf(errStNewCypherManager, err)
	}

	// Create new state vector for storing statuses of arrived parts
	partStatus, err := utility.NewStateVector(
		kv, sentTransferStatusKey, uint32(len(parts)))
	if err != nil {
		return nil, errors.Errorf(errStNewPartStatusVector, err)
	}

	st := &SentTransfer{
		cypherManager: cypherManager,
		tid:           tid,
		fileName:      fileName,
		recipient:     recipient,
		fileSize:      fileSize,
		numParts:      uint16(len(parts)),
		status:        Running,
		parts:         parts,
		partStatus:    partStatus,
		kv:            kv,
	}

	return st, st.save()
}

// GetUnsentParts builds a list of all unsent parts, each in a Part object.
func (st *SentTransfer) GetUnsentParts() []Part {
	unusedPartNumbers := st.partStatus.GetUnusedKeyNums()
	partList := make([]Part, len(unusedPartNumbers))

	for i, partNum := range unusedPartNumbers {
		partList[i] = Part{
			transfer:      st,
			cypherManager: st.cypherManager,
			partNum:       uint16(partNum),
		}
	}

	return partList
}

// getPartData returns the part data from the given part number.
func (st *SentTransfer) getPartData(partNum uint16) []byte {
	if int(partNum) > len(st.parts)-1 {
		jww.FATAL.Panicf(errNoPartNum, partNum, st.tid, st.fileName)
	}

	return st.parts[partNum]
}

// markArrived marks the status of the given part numbers as arrived. When the
// last part is marked arrived, the transfer is marked as completed.
func (st *SentTransfer) markArrived(partNum uint16) {
	st.mux.Lock()
	defer st.mux.Unlock()

	st.partStatus.Use(uint32(partNum))

	// Mark transfer completed if all parts arrived
	if st.partStatus.GetNumUsed() == uint32(st.numParts) {
		st.status = Completed
	}
}

// markTransferFailed sets the transfer as failed. Only call this if no more
// retries are available.
func (st *SentTransfer) markTransferFailed() {
	st.mux.Lock()
	defer st.mux.Unlock()
	st.status = Failed
}

// Status returns the status of the transfer.
func (st *SentTransfer) Status() TransferStatus {
	st.mux.RLock()
	defer st.mux.RUnlock()
	return st.status
}

// TransferID returns the transfer's ID.
func (st *SentTransfer) TransferID() *ftCrypto.TransferID {
	return st.tid
}

// FileName returns the transfer's file name.
func (st *SentTransfer) FileName() string {
	return st.fileName
}

// Recipient returns the transfer's recipient ID.
func (st *SentTransfer) Recipient() *id.ID {
	return st.recipient
}

// FileSize returns the size of the entire file transfer.
func (st *SentTransfer) FileSize() uint32 {
	return st.fileSize
}

// NumParts returns the total number of file parts in the transfer.
func (st *SentTransfer) NumParts() uint16 {
	return st.numParts
}

// NumArrived returns the number of parts that have arrived.
func (st *SentTransfer) NumArrived() uint16 {
	return uint16(st.partStatus.GetNumUsed())
}

// CopyPartStatusVector returns a copy of the part status vector that can be
// used to look up the current status of parts. Note that the statuses are from
// when this function is called and not realtime.
func (st *SentTransfer) CopyPartStatusVector() *utility.StateVector {
	return st.partStatus.DeepCopy()
}

// CompareAndSwapCallbackFps compares the fingerprint to the previous callback
// call's fingerprint. If they are different, the new one is stored, and it
// returns true. Returns fall if they are the same.
func (st *SentTransfer) CompareAndSwapCallbackFps(
	completed bool, arrived, total uint16, err error) bool {
	fp := generateSentFp(completed, arrived, total, err)
	st.mux.Lock()
	defer st.mux.Unlock()

	if fp != st.lastCallbackFingerprint {
		st.lastCallbackFingerprint = fp
		return true
	}

	return false
}

// generateSentFp generates a fingerprint for a sent progress callback.
func generateSentFp(completed bool, arrived, total uint16, err error) string {
	errString := "<nil>"
	if err != nil {
		errString = err.Error()
	}

	return strconv.FormatBool(completed) +
		strconv.FormatUint(uint64(arrived), 10) +
		strconv.FormatUint(uint64(total), 10) +
		errString
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// loadSentTransfer loads the SentTransfer with the given transfer ID from
// storage.
func loadSentTransfer(tid *ftCrypto.TransferID, kv *versioned.KV) (
	*SentTransfer, error) {
	kv = kv.Prefix(makeSentTransferPrefix(tid))

	// Load cypher manager
	cypherManager, err := cypher.LoadManager(kv)
	if err != nil {
		return nil, errors.Errorf(errStLoadCypherManager, err)
	}

	// Load fileName, recipient ID, status, and file parts
	obj, err := kv.Get(sentTransferStoreKey, sentTransferStoreVersion)
	if err != nil {
		return nil, errors.Errorf(errStLoadFields, err)
	}

	fileName, recipient, status, parts, err := unmarshalSentTransfer(obj.Data)
	if err != nil {
		return nil, errors.Errorf(errStUnmarshalFields, err)
	}

	// Load state vector for storing statuses of arrived parts
	partStatus, err := utility.LoadStateVector(kv, sentTransferStatusKey)
	if err != nil {
		return nil, errors.Errorf(errStLoadPartStatusVector, err)
	}

	st := &SentTransfer{
		cypherManager: cypherManager,
		tid:           tid,
		fileName:      fileName,
		recipient:     recipient,
		fileSize:      calcFileSize(parts),
		numParts:      uint16(len(parts)),
		status:        status,
		parts:         parts,
		partStatus:    partStatus,
		kv:            kv,
	}

	return st, nil
}

// calcFileSize calculates the size of the entire file from a list of parts. All
// parts, except the last, are assumed to have the same length.
func calcFileSize(parts [][]byte) uint32 {
	lastPartSize := len(parts[len(parts)-1])
	otherPartsSize := len(parts[0]) * (len(parts) - 1)
	return uint32(lastPartSize + otherPartsSize)
}

// Delete deletes all data in the SentTransfer from storage.
func (st *SentTransfer) Delete() error {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Delete cypher manager
	err := st.cypherManager.Delete()
	if err != nil {
		return errors.Errorf(errStDeleteCypherManager, err)
	}

	// Delete recipient ID, status, and file parts
	err = st.kv.Delete(sentTransferStoreKey, sentTransferStoreVersion)
	if err != nil {
		return errors.Errorf(errStDeleteSentTransfer, err)
	}

	// Delete part status multi state vector
	err = st.partStatus.Delete()
	if err != nil {
		return errors.Errorf(errStDeletePartStatus, err)
	}

	return nil
}

// save stores all fields in SentTransfer that do not have their own storage
// (recipient ID, status, and file parts) to storage.
func (st *SentTransfer) save() error {
	data, err := st.marshal()
	if err != nil {
		return errors.Errorf(errMarshalSentTransfer, err)
	}

	obj := &versioned.Object{
		Version:   sentTransferStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return st.kv.Set(sentTransferStoreKey, obj)
}

// sentTransferDisk structure is used to marshal and unmarshal SentTransfer
// fields to/from storage.
type sentTransferDisk struct {
	FileName  string
	Recipient *id.ID
	Status    TransferStatus
	Parts     [][]byte
}

// marshal serialises the SentTransfer's fileName, recipient, status, and parts
// list.
func (st *SentTransfer) marshal() ([]byte, error) {
	disk := sentTransferDisk{
		FileName:  st.fileName,
		Recipient: st.recipient,
		Status:    st.status,
		Parts:     st.parts,
	}

	return json.Marshal(disk)
}

// unmarshalSentTransfer deserializes the data into a fileName, recipient,
// status, and parts list.
func unmarshalSentTransfer(data []byte) (fileName string, recipient *id.ID,
	status TransferStatus, parts [][]byte, err error) {
	var disk sentTransferDisk
	err = json.Unmarshal(data, &disk)
	if err != nil {
		return "", nil, 0, nil, err
	}

	return disk.FileName, disk.Recipient, disk.Status, disk.Parts, nil
}

// makeSentTransferPrefix generates the unique prefix used on the key value
// store to store sent transfers for the given transfer ID.
func makeSentTransferPrefix(tid *ftCrypto.TransferID) string {
	return sentTransferStorePrefix +
		base64.StdEncoding.EncodeToString(tid.Bytes())
}
