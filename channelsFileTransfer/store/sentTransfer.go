////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/channelsFileTransfer/store/cypher"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
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
	errNoPartNum = "no part with part number %d exists in file %s"

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
	fid ftCrypto.ID

	// ID of the recipient of the file transfer
	recipient *id.ID

	// The time when the file was first queued to send
	sentTimestamp time.Time

	// The transfer MAC used to verify a file is complete
	mac []byte

	// The size of the entire file
	fileSize uint32

	// The number of file parts in the file
	numParts uint16

	// Determines number of sen retries on failure
	retry float32

	// Indicates the status of the transfer
	status TransferStatus

	// List of all file parts in order to send
	parts [][]byte

	// Stores the status of each part in a bitstream format
	partStatus *utility.MultiStateVector

	// Current ID to assign to a callback
	currentCallbackID uint64

	// Unique identifier of the last progress callback called (used to prevent
	// callback calls with duplicate data)
	lastCallbackFingerprints map[uint64]string

	mux sync.RWMutex
	kv  *versioned.KV
}

// newSentTransfer generates a new SentTransfer with the specified transfer key,
// file ID, and parts.
func newSentTransfer(recipient *id.ID, sentTimestamp time.Time,
	key *ftCrypto.TransferKey, mac []byte, fid ftCrypto.ID, fileSize uint32,
	parts [][]byte, numFps uint16, retry float32, kv *versioned.KV) (
	*SentTransfer, error) {
	kv = kv.Prefix(makeSentTransferPrefix(fid))

	// Create new cypher manager
	cypherManager, err := cypher.NewManager(key, numFps, false, kv)
	if err != nil {
		return nil, errors.Errorf(errStNewCypherManager, err)
	}

	// Create new state vector for storing statuses of sent parts
	partStatus, err := utility.NewMultiStateVector(uint16(len(parts)),
		uint8(numSentStates), stateMap, sentTransferStatusKey, kv)
	if err != nil {
		return nil, errors.Errorf(errStNewPartStatusVector, err)
	}

	st := &SentTransfer{
		cypherManager:            cypherManager,
		fid:                      fid,
		recipient:                recipient,
		sentTimestamp:            sentTimestamp.Round(0),
		mac:                      mac,
		fileSize:                 fileSize,
		numParts:                 uint16(len(parts)),
		retry:                    retry,
		status:                   Running,
		parts:                    parts,
		partStatus:               partStatus,
		currentCallbackID:        0,
		lastCallbackFingerprints: make(map[uint64]string),
		kv:                       kv,
	}

	return st, st.save()
}

// GetUnsentParts builds a list of all unsent parts, each in a Part object.
func (st *SentTransfer) GetUnsentParts() []*Part {
	unusedPartNumbers := st.partStatus.GetKeys(uint8(UnsentPart))
	partList := make([]*Part, len(unusedPartNumbers))

	for i, partNum := range unusedPartNumbers {
		partList[i] = &Part{
			transfer:      st,
			cypherManager: st.cypherManager,
			partNum:       partNum,
		}
	}

	return partList
}

// GetSentParts builds a list of all sent parts, each in a Part object.
func (st *SentTransfer) GetSentParts() []*Part {
	unusedPartNumbers := st.partStatus.GetKeys(uint8(SentPart))
	partList := make([]*Part, len(unusedPartNumbers))

	for i, partNum := range unusedPartNumbers {
		partList[i] = &Part{
			transfer:      st,
			cypherManager: st.cypherManager,
			partNum:       partNum,
		}
	}

	return partList
}

// getPartData returns the part data from the given part number.
func (st *SentTransfer) getPartData(partNum uint16) []byte {
	if int(partNum) > len(st.parts)-1 {
		jww.FATAL.Panicf(errNoPartNum, partNum, st.fid)
	}

	return st.parts[partNum]
}

// markSent marks the SentPartStatus of the given part numbers as sent
// (SentPart). The part must already have a SentPartStatus of UnsentPart.
func (st *SentTransfer) markSent(partNum uint16) {
	st.partStatus.CompareNotAndSwap(partNum, uint8(ReceivedPart), uint8(SentPart))
}

// markReceived marks the SentPartStatus of the given part numbers as received
// (ReceivedPart). The part must already have a SentPartStatus of SentPart.
//
// When the last part is marked received, the TransferStatus of the entire sent
// transfer as Completed. This transfer can be retrieved publicly using
// SentTransfer.Status.
func (st *SentTransfer) markReceived(partNum uint16) {
	st.partStatus.Set(partNum, uint8(ReceivedPart))

	// Mark transfer completed if all parts are received
	if st.partStatus.GetCount(uint8(ReceivedPart)) == st.numParts {
		st.mux.Lock()
		defer st.mux.Unlock()
		st.status = Completed
	}
}

// markForResend marks the SentPartStatus of the given part numbers as unsent
// (UnsentPart). The part must already have a SentPartStatus of SentPart.
func (st *SentTransfer) markForResend(partNum uint16) {
	st.partStatus.Set(partNum, uint8(UnsentPart))
}

// getPartStatus returns the SentPartStatus of the given part.
func (st *SentTransfer) getPartStatus(partNum uint16) SentPartStatus {
	return SentPartStatus(st.partStatus.Get(partNum))
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

// GetFileID returns the file's ID.
func (st *SentTransfer) GetFileID() ftCrypto.ID {
	return st.fid
}

// GetRecipient returns the transfer's recipient ID.
func (st *SentTransfer) GetRecipient() *id.ID {
	return st.recipient
}

// SentTimestamp returns the time when the file was first queued to send.
func (st *SentTransfer) SentTimestamp() time.Time {
	return st.sentTimestamp
}

// GetKey returns the transfer key used for encrypting/decrypting.
// TODO: test
func (st *SentTransfer) GetKey() *ftCrypto.TransferKey {
	return st.cypherManager.GetKey()
}

// GetMAC returns the transfer MAC used to verify the file.
// TODO: test
func (st *SentTransfer) GetMAC() []byte {
	return st.mac
}

// GetFileSize returns the size of the entire file transfer.
func (st *SentTransfer) GetFileSize() uint32 {
	return st.fileSize
}

// GetNumParts returns the total number of file parts in the transfer.
func (st *SentTransfer) GetNumParts() uint16 {
	return st.numParts
}

// NumSent returns the number of parts that have been sent.
func (st *SentTransfer) NumSent() uint16 {
	return st.partStatus.GetCount(uint8(SentPart))
}

// NumReceived returns the number of parts that have been received.
func (st *SentTransfer) NumReceived() uint16 {
	return st.partStatus.GetCount(uint8(ReceivedPart))
}

// GetRetry returns the retry number.
// TODO: test
func (st *SentTransfer) GetRetry() float32 {
	return st.retry
}

// CopyPartStatusVector returns a copy of the part status vector that can be
// used to look up the current status of parts. Note that the statuses are from
// when this function is called and not realtime.
func (st *SentTransfer) CopyPartStatusVector() *utility.MultiStateVector {
	return st.partStatus.DeepCopy()
}

// GetNewCallbackID issues a new unique for a callback.
// TODO: test
func (st *SentTransfer) GetNewCallbackID() uint64 {
	st.mux.Lock()
	defer st.mux.Unlock()
	newID := st.currentCallbackID
	st.currentCallbackID++
	return newID
}

// CompareAndSwapCallbackFps compares the fingerprint to the previous callback
// call's fingerprint. If they are different, the new one is stored, and it
// returns true. Returns fall if they are the same.
func (st *SentTransfer) CompareAndSwapCallbackFps(callbackID uint64,
	completed bool, sent, received, total uint16, err error) bool {
	fp := generateSentFp(completed, sent, received, total, err)
	st.mux.Lock()
	defer st.mux.Unlock()

	if fp != st.lastCallbackFingerprints[callbackID] {
		st.lastCallbackFingerprints[callbackID] = fp
		return true
	}

	return false
}

// generateSentFp generates a fingerprint for a sent progress callback.
func generateSentFp(
	completed bool, sent, received, total uint16, err error) string {
	errString := "<nil>"
	if err != nil {
		errString = err.Error()
	}

	return strconv.FormatBool(completed) +
		strconv.FormatUint(uint64(sent), 10) +
		strconv.FormatUint(uint64(received), 10) +
		strconv.FormatUint(uint64(total), 10) +
		errString
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// loadSentTransfer loads the SentTransfer with the given file ID from storage.
func loadSentTransfer(
	fid ftCrypto.ID, parts [][]byte, kv *versioned.KV) (*SentTransfer, error) {
	kv = kv.Prefix(makeSentTransferPrefix(fid))

	// Load cypher manager
	cypherManager, err := cypher.LoadManager(kv)
	if err != nil {
		return nil, errors.Errorf(errStLoadCypherManager, err)
	}

	// Load fileName, recipient ID, status, and file parts
	obj, err := kv.Get(sentTransferStoreKey, sentTransferStoreVersion)
	if err != nil {
		// TODO: test
		return nil, errors.Errorf(errStLoadFields, err)
	}

	info, err := unmarshalSentTransfer(obj.Data)
	if err != nil {
		return nil, errors.Errorf(errStUnmarshalFields, err)
	}

	// Load state vector for storing statuses of sent parts
	partStatus, err := utility.LoadMultiStateVector(
		stateMap, sentTransferStatusKey, kv)
	if err != nil {
		return nil, errors.Errorf(errStLoadPartStatusVector, err)
	}

	st := &SentTransfer{
		cypherManager:            cypherManager,
		fid:                      fid,
		recipient:                info.Recipient,
		sentTimestamp:            info.SentTimestamp.Round(0),
		mac:                      info.Mac,
		fileSize:                 calcFileSize(parts),
		numParts:                 uint16(len(parts)),
		retry:                    info.Retry,
		status:                   info.Status,
		parts:                    parts,
		partStatus:               partStatus,
		currentCallbackID:        0,
		lastCallbackFingerprints: make(map[uint64]string),
		kv:                       kv,
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
	Recipient     *id.ID         `json:"recipient"`
	SentTimestamp time.Time      `json:"sentTimestamp"`
	Mac           []byte         `json:"mac"`
	Retry         float32        `json:"retry"`
	Status        TransferStatus `json:"status"`
}

// marshal serialises the SentTransfer's file information.
func (st *SentTransfer) marshal() ([]byte, error) {
	disk := sentTransferDisk{
		Recipient:     st.recipient,
		SentTimestamp: st.sentTimestamp,
		Mac:           st.mac,
		Retry:         st.retry,
		Status:        st.status,
	}

	return json.Marshal(disk)
}

// unmarshalSentTransfer deserializes the SentTransfer's file information.
func unmarshalSentTransfer(data []byte) (sentTransferDisk, error) {
	var disk sentTransferDisk
	return disk, json.Unmarshal(data, &disk)
}

// makeSentTransferPrefix generates the unique prefix used on the key value
// store to store sent transfers for the given file ID.
func makeSentTransferPrefix(fid ftCrypto.ID) string {
	return sentTransferStorePrefix +
		base64.StdEncoding.EncodeToString(fid.Marshal())
}
