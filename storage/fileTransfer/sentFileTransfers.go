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
	sentFileTransfersPrefix  = "SentFileTransfersStore"
	sentFileTransfersKey     = "SentFileTransfers"
	sentFileTransfersVersion = 0
)

// Error messages.
const (
	saveSentTransfersListErr = "failed to save list of sent items in transfer map to storage: %+v"
	loadSentTransfersListErr = "failed to load list of sent items in transfer map from storage: %+v"
	loadSentTransfersErr     = "failed to load sent transfers from storage: %+v"

	newSentTransferErr    = "failed to create new sent transfer: %+v"
	getSentTransferErr    = "sent file transfer not found"
	cancelCallbackErr     = "Transfer with ID %s: %+v"
	deleteSentTransferErr = "failed to delete sent transfer with ID %s from store: %+v"
)

// SentFileTransfers contains information for tracking sent file transfers.
type SentFileTransfers struct {
	transfers map[ftCrypto.TransferID]*SentTransfer
	mux       sync.Mutex
	kv        *versioned.KV
}

// NewSentFileTransfers creates a new SentFileTransfers with an empty map.
func NewSentFileTransfers(kv *versioned.KV) (*SentFileTransfers, error) {
	sft := &SentFileTransfers{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersPrefix),
	}

	return sft, sft.saveTransfersList()
}

// AddTransfer creates a new empty SentTransfer and adds it to the transfers
// map.
func (sft *SentFileTransfers) AddTransfer(recipient *id.ID,
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
func (sft *SentFileTransfers) GetTransfer(tid ftCrypto.TransferID) (
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
func (sft *SentFileTransfers) DeleteTransfer(tid ftCrypto.TransferID) error {
	sft.mux.Lock()
	defer sft.mux.Unlock()

	// Return an error if the transfer does not exist
	st, exists := sft.transfers[tid]
	if !exists {
		return errors.New(getSentTransferErr)
	}

	// Cancel any scheduled callbacks
	err := st.StopScheduledProgressCB()
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

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadSentFileTransfers loads all SentFileTransfers from storage.
func LoadSentFileTransfers(kv *versioned.KV) (*SentFileTransfers, error) {
	sft := &SentFileTransfers{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersPrefix),
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

// NewOrLoadSentFileTransfers loads all SentFileTransfers from storage, if they
// exist. Otherwise, a new SentFileTransfers is returned.
func NewOrLoadSentFileTransfers(kv *versioned.KV) (*SentFileTransfers, error) {
	sft := &SentFileTransfers{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersPrefix),
	}

	// If the transfer list cannot be loaded from storage, then create a new
	// SentFileTransfers
	vo, err := sft.kv.Get(sentFileTransfersKey, sentFileTransfersVersion)
	if err != nil {
		return NewSentFileTransfers(kv)
	}

	// Unmarshal data into list of saved transfer IDs
	transfersList := unmarshalTransfersList(vo.Data)

	// Load each transfer in the list from storage into the map
	err = sft.loadTransfers(transfersList)
	if err != nil {
		return nil, errors.Errorf(loadSentTransfersErr, err)
	}

	return sft, nil
}

// saveTransfersList saves a list of items in the transfers map to storage.
func (sft *SentFileTransfers) saveTransfersList() error {
	// Create new versioned object with a list of items in the transfers map
	obj := &versioned.Object{
		Version:   sentFileTransfersVersion,
		Timestamp: netTime.Now(),
		Data:      sft.marshalTransfersList(),
	}

	// Save list of items in the transfers map to storage
	return sft.kv.Set(sentFileTransfersKey, sentFileTransfersVersion, obj)
}

// loadTransfersList gets the list of transfer IDs corresponding to each saved
// sent transfer from storage.
func (sft *SentFileTransfers) loadTransfersList() ([]ftCrypto.TransferID, error) {
	// Get transfers list from storage
	vo, err := sft.kv.Get(sentFileTransfersKey, sentFileTransfersVersion)
	if err != nil {
		return nil, err
	}

	// Unmarshal data into list of saved transfer IDs
	return unmarshalTransfersList(vo.Data), nil
}

// loadTransfers loads each SentTransfer from the list and adds them to the map.
func (sft *SentFileTransfers) loadTransfers(list []ftCrypto.TransferID) error {
	var err error

	// Load each sentTransfer from storage into the map
	for _, tid := range list {
		sft.transfers[tid], err = loadSentTransfer(tid, sft.kv)
		if err != nil {
			return err
		}
	}

	return nil
}

// marshalTransfersList creates a list of all transfer IDs in the transfers map
// and serialises it.
func (sft *SentFileTransfers) marshalTransfersList() []byte {
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
