////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

// Storage keys and versions.
const (
	sentFileTransfersStorePrefix  = "SentFileTransfersStore"
	sentFileTransfersStoreKey     = "SentFileTransfers"
	sentFileTransfersStoreVersion = 0
)

// Error messages.
const (
	saveSentTransfersListErr = "failed to save list of sent items in transfer map to storage: %+v"
	loadSentTransfersListErr = "failed to load list of sent items in transfer map from storage: %+v"
	loadSentTransfersErr     = "[FT] Failed to load sent transfers from storage: %+v"

	newSentTransferErr    = "failed to create new sent transfer: %+v"
	getSentTransferErr    = "sent file transfer not found"
	cancelCallbackErr     = "[FT] Transfer with ID %s: %+v"
	deleteSentTransferErr = "failed to delete sent transfer with ID %s from store: %+v"

	// SentFileTransfersStore.loadTransfers
	loadSentTransferWarn    = "[FT] Failed to load sent file transfer %d of %d with ID %s: %v"
	loadSentTransfersAllErr = "failed to load all %d transfers"
)

// SentFileTransfersStore contains information for tracking sent file transfers.
type SentFileTransfersStore struct {
	transfers map[ftCrypto.TransferID]*SentTransfer
	mux       sync.Mutex
	kv        *versioned.KV
}

// NewSentFileTransfersStore creates a new SentFileTransfersStore with an empty
// map.
func NewSentFileTransfersStore(kv *versioned.KV) (*SentFileTransfersStore, error) {
	sft := &SentFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersStorePrefix),
	}

	return sft, sft.saveTransfersList()
}

// AddTransfer creates a new empty SentTransfer and adds it to the transfers
// map.
func (sft *SentFileTransfersStore) AddTransfer(recipient *id.ID,
	key ftCrypto.TransferKey, parts [][]byte, numFps uint16,
	progressCB interfaces.SentProgressCallback, period time.Duration,
	rng csprng.Source) (ftCrypto.TransferID, error) {

	sft.mux.Lock()
	defer sft.mux.Unlock()

	// Generate new transfer ID
	tid, err := ftCrypto.NewTransferID(rng)
	if err != nil {
		return tid, errors.Errorf(addTransferNewIdErr, err)
	}

	// Generate a new SentTransfer and add it to the map
	sft.transfers[tid], err = NewSentTransfer(
		recipient, tid, key, parts, numFps, progressCB, period, sft.kv)
	if err != nil {
		return tid, errors.Errorf(newSentTransferErr, err)
	}

	// Update list of transfers in storage
	err = sft.saveTransfersList()
	if err != nil {
		return tid, errors.Errorf(saveSentTransfersListErr, err)
	}

	return tid, nil
}

// GetTransfer returns the SentTransfer with the given transfer ID. An error is
// returned if no corresponding transfer is found.
func (sft *SentFileTransfersStore) GetTransfer(tid ftCrypto.TransferID) (
	*SentTransfer, error) {
	sft.mux.Lock()
	defer sft.mux.Unlock()

	rt, exists := sft.transfers[tid]
	if !exists {
		return nil, errors.New(getSentTransferErr)
	}

	return rt, nil
}

// DeleteTransfer removes the SentTransfer with the associated transfer ID
// from memory and storage.
func (sft *SentFileTransfersStore) DeleteTransfer(tid ftCrypto.TransferID) error {
	sft.mux.Lock()
	defer sft.mux.Unlock()

	// Return an error if the transfer does not exist
	st, exists := sft.transfers[tid]
	if !exists {
		return errors.New(getSentTransferErr)
	}

	// Cancel any scheduled callbacks
	err := st.stopScheduledProgressCB()
	if err != nil {
		jww.WARN.Print(errors.Errorf(cancelCallbackErr, tid, err))
	}

	// Delete all data the transfer saved to storage
	err = st.delete()
	if err != nil {
		return errors.Errorf(deleteSentTransferErr, tid, err)
	}

	// Delete the transfer from memory
	delete(sft.transfers, tid)

	// Update the transfers list for the removed transfer
	err = sft.saveTransfersList()
	if err != nil {
		return errors.Errorf(saveSentTransfersListErr, err)
	}

	return nil
}

// GetUnsentParts returns a map of all transfers and a list of their parts that
// have not been sent (parts that were never marked as in-progress).
func (sft *SentFileTransfersStore) GetUnsentParts() (
	map[ftCrypto.TransferID][]uint16, error) {
	sft.mux.Lock()
	defer sft.mux.Unlock()
	unsentParts := map[ftCrypto.TransferID][]uint16{}

	// Get list of unsent part numbers for each transfer
	for tid, st := range sft.transfers {
		unsentPartNums, err := st.GetUnsentPartNums()
		if err != nil {
			return nil, err
		}
		unsentParts[tid] = unsentPartNums
	}

	return unsentParts, nil
}

// GetSentRounds returns a map of all round IDs and which transfers have parts
// sent on those rounds (parts marked in-progress).
func (sft *SentFileTransfersStore) GetSentRounds() map[id.Round][]ftCrypto.TransferID {
	sft.mux.Lock()
	defer sft.mux.Unlock()
	sentRounds := map[id.Round][]ftCrypto.TransferID{}

	// Get list of round IDs that transfers have in-progress rounds on
	for tid, st := range sft.transfers {
		for _, rid := range st.GetSentRounds() {
			sentRounds[rid] = append(sentRounds[rid], tid)
		}
	}

	return sentRounds
}

// GetUnsentPartsAndSentRounds returns two maps. The first is a map of all
// transfers and a list of their parts that have not been sent (parts that were
// never marked as in-progress). The second is a map of all round IDs and which
// transfers have parts sent on those rounds (parts marked in-progress). This
// function performs the same operations as GetUnsentParts and GetSentRounds but
// in a single loop.
func (sft *SentFileTransfersStore) GetUnsentPartsAndSentRounds() (
	map[ftCrypto.TransferID][]uint16, map[id.Round][]ftCrypto.TransferID, error) {
	sft.mux.Lock()
	defer sft.mux.Unlock()

	unsentParts := map[ftCrypto.TransferID][]uint16{}
	sentRounds := map[id.Round][]ftCrypto.TransferID{}

	for tid, st := range sft.transfers {
		// Get list of unsent part numbers for each transfer
		stUnsentParts, err := st.GetUnsentPartNums()
		if err != nil {
			return nil, nil, err
		}
		if len(stUnsentParts) > 0 {
			unsentParts[tid] = stUnsentParts
		}

		// Get list of round IDs that transfers have in-progress rounds on
		for _, rid := range st.GetSentRounds() {
			sentRounds[rid] = append(sentRounds[rid], tid)
		}
	}

	return unsentParts, sentRounds, nil
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadSentFileTransfersStore loads all SentFileTransfersStore from storage.
// Returns a list of unsent file parts.
func LoadSentFileTransfersStore(kv *versioned.KV) (*SentFileTransfersStore, error) {
	sft := &SentFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersStorePrefix),
	}

	// Get the list of transfer IDs corresponding to each sent transfer from
	// storage
	transfersList, err := sft.loadTransfersList()
	if err != nil {
		return nil, errors.Errorf(loadSentTransfersListErr, err)
	}

	// Load each transfer in the list from storage into the map
	err = sft.loadTransfers(transfersList)
	if err != nil {
		return nil, errors.Errorf(loadSentTransfersErr, err)
	}

	return sft, nil
}

// NewOrLoadSentFileTransfersStore loads all SentFileTransfersStore from storage
// and returns a list of unsent file parts, if they exist. Otherwise, a new
// SentFileTransfersStore is returned.
func NewOrLoadSentFileTransfersStore(kv *versioned.KV) (*SentFileTransfersStore,
	error) {
	sft := &SentFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersStorePrefix),
	}

	// If the transfer list cannot be loaded from storage, then create a new
	// SentFileTransfersStore
	vo, err := sft.kv.Get(
		sentFileTransfersStoreKey, sentFileTransfersStoreVersion)
	if err != nil {
		return NewSentFileTransfersStore(kv)
	}

	// Unmarshal data into list of saved transfer IDs
	transfersList := unmarshalTransfersList(vo.Data)

	// Load each transfer in the list from storage into the map
	err = sft.loadTransfers(transfersList)
	if err != nil {
		jww.WARN.Printf(loadSentTransfersErr, err)
		return NewSentFileTransfersStore(kv)
	}

	return sft, nil
}

// saveTransfersList saves a list of items in the transfers map to storage.
func (sft *SentFileTransfersStore) saveTransfersList() error {
	// Create new versioned object with a list of items in the transfers map
	obj := &versioned.Object{
		Version:   sentFileTransfersStoreVersion,
		Timestamp: netTime.Now(),
		Data:      sft.marshalTransfersList(),
	}

	// Save list of items in the transfers map to storage
	return sft.kv.Set(
		sentFileTransfersStoreKey, sentFileTransfersStoreVersion, obj)
}

// loadTransfersList gets the list of transfer IDs corresponding to each saved
// sent transfer from storage.
func (sft *SentFileTransfersStore) loadTransfersList() ([]ftCrypto.TransferID,
	error) {
	// Get transfers list from storage
	vo, err := sft.kv.Get(
		sentFileTransfersStoreKey, sentFileTransfersStoreVersion)
	if err != nil {
		return nil, err
	}

	// Unmarshal data into list of saved transfer IDs
	return unmarshalTransfersList(vo.Data), nil
}

// loadTransfers loads each SentTransfer from the list and adds them to the map.
// Returns a map of all transfers and their unsent file part numbers to be used
// to add them back into the queue.
func (sft *SentFileTransfersStore) loadTransfers(list []ftCrypto.TransferID) error {
	var err error
	var errCount int

	// Load each sentTransfer from storage into the map
	for i, tid := range list {
		sft.transfers[tid], err = loadSentTransfer(tid, sft.kv)
		if err != nil {
			jww.WARN.Printf(loadSentTransferWarn, i, len(list), tid, err)
			errCount++
		}
	}

	// Return an error if all transfers failed to load
	if errCount == len(list) {
		return errors.Errorf(loadSentTransfersAllErr, len(list))
	}

	return nil
}

// marshalTransfersList creates a list of all transfer IDs in the transfers map
// and serialises it.
func (sft *SentFileTransfersStore) marshalTransfersList() []byte {
	buff := bytes.NewBuffer(nil)
	buff.Grow(ftCrypto.TransferIdLength * len(sft.transfers))

	for tid := range sft.transfers {
		buff.Write(tid.Bytes())
	}

	return buff.Bytes()
}

// unmarshalTransfersList deserializes a byte slice into a list of transfer IDs.
func unmarshalTransfersList(b []byte) []ftCrypto.TransferID {
	buff := bytes.NewBuffer(b)
	list := make([]ftCrypto.TransferID, 0, buff.Len()/ftCrypto.TransferIdLength)

	const size = ftCrypto.TransferIdLength
	for n := buff.Next(size); len(n) == size; n = buff.Next(size) {
		list = append(list, ftCrypto.UnmarshalTransferID(n))
	}

	return list
}
