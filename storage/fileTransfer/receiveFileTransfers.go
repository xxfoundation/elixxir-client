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
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	receivedFileTransfersStorePrefix  = "FileTransferReceivedFileTransfersStore"
	receivedFileTransfersStoreKey     = "ReceivedFileTransfers"
	receivedFileTransfersStoreVersion = 0
)

// Error messages for ReceivedFileTransfersStore.
const (
	saveReceivedTransfersListErr = "failed to save list of received items in transfer map to storage: %+v"
	loadReceivedTransfersListErr = "failed to load list of received items in transfer map from storage: %+v"
	loadReceivedFileTransfersErr = "[FT] Failed to load received transfers from storage: %+v"

	newReceivedTransferErr    = "failed to create new received transfer: %+v"
	getReceivedTransferErr    = "received transfer with ID %s not found"
	addTransferNewIdErr       = "could not generate new transfer ID: %+v"
	noFingerprintErr          = "no part found with fingerprint %s"
	addPartErr                = "failed to add part to transfer %s: %+v"
	deleteReceivedTransferErr = "failed to delete received transfer with ID %s from store: %+v"

	// ReceivedFileTransfersStore.load
	loadReceivedTransferWarn    = "[FT] Failed to load received file transfer %d of %d with ID %s: %v"
	loadReceivedTransfersAllErr = "failed to load all %d transfers"
)

// ReceivedFileTransfersStore contains information for tracking a received
// ReceivedTransfer to its transfer ID. It also maps a received part, partInfo,
// to the message fingerprint.
type ReceivedFileTransfersStore struct {
	transfers map[ftCrypto.TransferID]*ReceivedTransfer
	info      map[format.Fingerprint]*partInfo
	mux       sync.Mutex
	kv        *versioned.KV
}

// NewReceivedFileTransfersStore creates a new ReceivedFileTransfersStore with
// empty maps.
func NewReceivedFileTransfersStore(kv *versioned.KV) (
	*ReceivedFileTransfersStore, error) {
	rft := &ReceivedFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*ReceivedTransfer),
		info:      make(map[format.Fingerprint]*partInfo),
		kv:        kv.Prefix(receivedFileTransfersStorePrefix),
	}

	return rft, rft.saveTransfersList()
}

// AddTransfer creates a new empty ReceivedTransfer, adds it to the transfers
// map, and adds an entry for each file part fingerprint to the fingerprint map.
func (rft *ReceivedFileTransfersStore) AddTransfer(key ftCrypto.TransferKey,
	transferMAC []byte, fileSize uint32, numParts, numFps uint16,
	rng csprng.Source) (ftCrypto.TransferID, error) {

	rft.mux.Lock()
	defer rft.mux.Unlock()

	// Generate new transfer ID
	tid, err := ftCrypto.NewTransferID(rng)
	if err != nil {
		return tid, errors.Errorf(addTransferNewIdErr, err)
	}

	// Generate a new ReceivedTransfer and add it to the map
	rft.transfers[tid], err = NewReceivedTransfer(
		tid, key, transferMAC, fileSize, numParts, numFps, rft.kv)
	if err != nil {
		return tid, errors.Errorf(newReceivedTransferErr, err)
	}

	// Add part info for each file part to the map
	rft.addFingerprints(key, tid, numFps)

	// Update list of transfers in storage
	err = rft.saveTransfersList()
	if err != nil {
		return tid, errors.Errorf(saveReceivedTransfersListErr, err)
	}

	return tid, nil
}

// addFingerprints generates numFps fingerprints, creates a new partInfo
// for each, and adds each to the info map.
func (rft *ReceivedFileTransfersStore) addFingerprints(key ftCrypto.TransferKey,
	tid ftCrypto.TransferID, numFps uint16) {

	// Generate list of fingerprints
	fps := ftCrypto.GenerateFingerprints(key, numFps)

	// Add fingerprints to map
	for fpNum, fp := range fps {
		rft.info[fp] = newPartInfo(tid, uint16(fpNum))
	}
}

// GetTransfer returns the ReceivedTransfer with the given transfer ID. An error
// is returned if no corresponding transfer is found.
func (rft *ReceivedFileTransfersStore) GetTransfer(tid ftCrypto.TransferID) (
	*ReceivedTransfer, error) {
	rft.mux.Lock()
	defer rft.mux.Unlock()

	rt, exists := rft.transfers[tid]
	if !exists {
		return nil, errors.Errorf(getReceivedTransferErr, tid)
	}

	return rt, nil
}

// DeleteTransfer removes the ReceivedTransfer with the associated transfer ID
// from memory and storage.
func (rft *ReceivedFileTransfersStore) DeleteTransfer(tid ftCrypto.TransferID) error {
	rft.mux.Lock()
	defer rft.mux.Unlock()

	// Return an error if the transfer does not exist
	rt, exists := rft.transfers[tid]
	if !exists {
		return errors.Errorf(getReceivedTransferErr, tid)
	}

	// Cancel any scheduled callbacks
	err := rt.stopScheduledProgressCB()
	if err != nil {
		jww.WARN.Print(errors.Errorf(cancelCallbackErr, tid, err))
	}

	// Remove all unused fingerprints from map
	for n, err := rt.fpVector.Next(); err == nil; n, err = rt.fpVector.Next() {
		// Generate fingerprint
		fp := ftCrypto.GenerateFingerprint(rt.key, uint16(n))

		// Delete fingerprint from map
		delete(rft.info, fp)
	}

	// Delete all data the transfer saved to storage
	err = rft.transfers[tid].delete()
	if err != nil {
		return errors.Errorf(deleteReceivedTransferErr, tid, err)
	}

	// Delete the transfer from memory
	delete(rft.transfers, tid)

	// Update the transfers list for the removed transfer
	err = rft.saveTransfersList()
	if err != nil {
		return errors.Errorf(saveReceivedTransfersListErr, err)
	}

	return nil
}

// AddPart adds the file part to its corresponding transfer. The fingerprint
// number and transfer ID are looked up using the fingerprint. Then the part is
// added to the transfer with the corresponding transfer ID. Returns the
// transfer that the part was added to so that a progress callback can be
// called. Returns the transfer ID so that it can be used for logging. Also
// returns of the transfer is complete after adding the part.
func (rft *ReceivedFileTransfersStore) AddPart(cmixMsg format.Message) (*ReceivedTransfer,
	ftCrypto.TransferID, bool, error) {
	rft.mux.Lock()
	defer rft.mux.Unlock()

	keyfp := cmixMsg.GetKeyFP()

	// Lookup the part info for the given fingerprint
	info, exists := rft.info[cmixMsg.GetKeyFP()]
	if !exists {
		return nil, ftCrypto.TransferID{}, false,
			errors.Errorf(noFingerprintErr, keyfp)
	}

	// Lookup the transfer with the ID in the part info
	transfer, exists := rft.transfers[info.id]
	if !exists {
		return nil, info.id, false,
			errors.Errorf(getReceivedTransferErr, info.id)
	}

	// Add the part to the transfer
	completed, err := transfer.AddPart(cmixMsg, info.fpNum)
	if err != nil {
		return transfer, info.id, false, errors.Errorf(
			addPartErr, info.id, err)
	}

	// Remove the part info from the map
	delete(rft.info, keyfp)

	return transfer, info.id, completed, nil
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadReceivedFileTransfersStore loads all ReceivedFileTransfersStore from
// storage.
func LoadReceivedFileTransfersStore(kv *versioned.KV) (
	*ReceivedFileTransfersStore, error) {
	rft := &ReceivedFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*ReceivedTransfer),
		info:      make(map[format.Fingerprint]*partInfo),
		kv:        kv.Prefix(receivedFileTransfersStorePrefix),
	}

	// Get the list of transfer IDs corresponding to each received transfer from
	// storage
	transfersList, err := rft.loadTransfersList()
	if err != nil {
		return nil, errors.Errorf(loadReceivedTransfersListErr, err)
	}

	// Load transfers and fingerprints into the maps
	err = rft.load(transfersList)
	if err != nil {
		return nil, errors.Errorf(loadReceivedFileTransfersErr, err)
	}

	return rft, nil
}

// NewOrLoadReceivedFileTransfersStore loads all ReceivedFileTransfersStore from
// storage, if they exist. Otherwise, a new ReceivedFileTransfersStore is
// returned.
func NewOrLoadReceivedFileTransfersStore(kv *versioned.KV) (
	*ReceivedFileTransfersStore, error) {
	rft := &ReceivedFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*ReceivedTransfer),
		info:      make(map[format.Fingerprint]*partInfo),
		kv:        kv.Prefix(receivedFileTransfersStorePrefix),
	}

	// If the transfer list cannot be loaded from storage, then create a new
	// ReceivedFileTransfersStore
	vo, err := rft.kv.Get(
		receivedFileTransfersStoreKey, receivedFileTransfersStoreVersion)
	if err != nil {
		return NewReceivedFileTransfersStore(kv)
	}

	// Unmarshal data into list of saved transfer IDs
	transfersList := unmarshalTransfersList(vo.Data)

	// Load transfers and fingerprints into the maps
	err = rft.load(transfersList)
	if err != nil {
		jww.ERROR.Printf(loadReceivedFileTransfersErr, err)
		return NewReceivedFileTransfersStore(kv)
	}

	return rft, nil
}

// saveTransfersList saves a list of items in the transfers map to storage.
func (rft *ReceivedFileTransfersStore) saveTransfersList() error {
	// Create new versioned object with a list of items in the transfers map
	obj := &versioned.Object{
		Version:   receivedFileTransfersStoreVersion,
		Timestamp: netTime.Now(),
		Data:      rft.marshalTransfersList(),
	}

	// Save list of items in the transfers map to storage
	return rft.kv.Set(
		receivedFileTransfersStoreKey, receivedFileTransfersStoreVersion, obj)
}

// loadTransfersList gets the list of transfer IDs corresponding to each saved
// received transfer from storage.
func (rft *ReceivedFileTransfersStore) loadTransfersList() (
	[]ftCrypto.TransferID, error) {
	// Get transfers list from storage
	vo, err := rft.kv.Get(
		receivedFileTransfersStoreKey, receivedFileTransfersStoreVersion)
	if err != nil {
		return nil, err
	}

	// Unmarshal data into list of saved transfer IDs
	return unmarshalTransfersList(vo.Data), nil
}

// load gets each ReceivedTransfer in the list from storage and adds them to the
// map. Also adds all unused fingerprints in each ReceivedTransfer to the info
// map.
func (rft *ReceivedFileTransfersStore) load(list []ftCrypto.TransferID) error {
	var errCount int

	// Load each sentTransfer from storage into the map
	for i, tid := range list {
		// Load the transfer with the given transfer ID from storage
		rt, err := loadReceivedTransfer(tid, rft.kv)
		if err != nil {
			jww.WARN.Printf(loadReceivedTransferWarn, i, len(list), tid, err)
			errCount++
			continue
		}

		// Add transfer to transfer map
		rft.transfers[tid] = rt

		// Load all unused fingerprints into the info map
		for n := uint32(0); n < rt.fpVector.GetNumKeys(); n++ {
			if !rt.fpVector.Used(n) {
				fpNum := uint16(n)

				// Generate fingerprint
				fp := ftCrypto.GenerateFingerprint(rt.key, fpNum)

				// Add to map
				rft.info[fp] = &partInfo{tid, fpNum}
			}
		}
	}

	// Return an error if all transfers failed to load
	if errCount == len(list) {
		return errors.Errorf(loadReceivedTransfersAllErr, len(list))
	}

	return nil
}

// marshalTransfersList creates a list of all transfer IDs in the transfers map
// and serialises it.
func (rft *ReceivedFileTransfersStore) marshalTransfersList() []byte {
	buff := bytes.NewBuffer(nil)
	buff.Grow(ftCrypto.TransferIdLength * len(rft.transfers))

	for tid := range rft.transfers {
		buff.Write(tid.Bytes())
	}

	return buff.Bytes()
}
