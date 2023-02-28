////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage keys and versions.
const (
	receivedTransfersStorePrefix  = "ReceivedFileTransfersPrefix"
	receivedTransfersStoreKey     = "ReceivedFileTransfers"
	receivedTransfersStoreVersion = 0
)

// Error messages.
const (
	// NewOrLoadReceived
	errLoadReceived            = "error loading received transfer list from storage: %+v"
	errUnmarshalReceived       = "could not unmarshal received transfer list: %+v"
	warnLoadReceivedTransfer   = "[FT] failed to load received transfer %d of %d with ID %s: %+v"
	errLoadAllReceivedTransfer = "failed to load all %d received transfers"

	// Received.AddTransfer
	errAddExistingReceivedTransfer = "received transfer with ID %s already exists in map."
)

// Received contains a list of all received transfers.
type Received struct {
	transfers map[ftCrypto.ID]*ReceivedTransfer

	mux       sync.RWMutex
	disableKV bool // Toggles use of KV storage
	kv        *versioned.KV
}

// NewOrLoadReceived attempts to load a Received from storage. Or if none exist,
// then a new Received is returned. A list of file IDs for all incomplete
// receives is also returned.
func NewOrLoadReceived(
	disableKV bool, kv *versioned.KV) (*Received, []ftCrypto.ID, error) {
	r := &Received{
		transfers: make(map[ftCrypto.ID]*ReceivedTransfer),
		disableKV: disableKV,
		kv:        kv.Prefix(receivedTransfersStorePrefix),
	}

	if disableKV {
		return r, nil, nil
	}

	obj, err := r.kv.Get(receivedTransfersStoreKey, receivedTransfersStoreVersion)
	if err != nil {
		if kv.Exists(err) {
			return nil, nil, errors.Errorf(errLoadReceived, err)
		} else {
			return r, nil, nil
		}
	}

	fidList, err := unmarshalFileIdList(obj.Data)
	if err != nil {
		return nil, nil, errors.Errorf(errUnmarshalReceived, err)
	}

	return r, fidList, nil
}

// LoadTransfers loads all received transfers in the list from storage into
// Received  It returns a list of all incomplete transfers so that their
// fingerprints can be re-added to the listener.
func (r *Received) LoadTransfers(partialFiles map[ftCrypto.ID][]byte,
	partSize int) ([]*ReceivedTransfer, error) {

	var errCount, i int
	var err error
	incompleteTransfer := make([]*ReceivedTransfer, 0, len(partialFiles))
	for fid, partialFile := range partialFiles {
		i++
		r.transfers[fid], err =
			loadReceivedTransfer(fid, partialFile, partSize, r.kv)
		if err != nil {
			jww.WARN.Printf(
				warnLoadReceivedTransfer, i, len(partialFiles), fid, err)
			errCount++
		}

		if r.transfers[fid].NumReceived() != r.transfers[fid].GetNumParts() {
			incompleteTransfer = append(incompleteTransfer, r.transfers[fid])
		}
	}

	// Return an error if all transfers failed to load
	if len(partialFiles) > 0 && errCount == len(partialFiles) {
		return nil, errors.Errorf(errLoadAllReceivedTransfer, len(partialFiles))
	}

	return incompleteTransfer, nil
}

// AddTransfer adds the ReceivedTransfer to the map keyed on its file ID.
func (r *Received) AddTransfer(recipient *id.ID, key *ftCrypto.TransferKey,
	fid ftCrypto.ID, fileName string, transferMAC []byte,
	fileSize uint32, numParts, numFps uint16) (*ReceivedTransfer, error) {

	r.mux.Lock()
	defer r.mux.Unlock()

	_, exists := r.transfers[fid]
	if exists {
		return nil, errors.Errorf(errAddExistingReceivedTransfer, fid)
	}

	rt, err := newReceivedTransfer(recipient, key, fid, fileName, transferMAC,
		fileSize, numParts, numFps, r.disableKV, r.kv)
	if err != nil {
		return nil, err
	}

	r.transfers[fid] = rt

	return rt, r.save()
}

// GetTransfer returns the ReceivedTransfer with the provided file ID or false
// if none exists.
func (r *Received) GetTransfer(fid ftCrypto.ID) (*ReceivedTransfer, bool) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	rt, exists := r.transfers[fid]
	return rt, exists
}

// RemoveTransfer removes the transfer from the map. If no transfer exists,
// returns nil. Only errors due to saving to storage are returned.
func (r *Received) RemoveTransfer(fid ftCrypto.ID) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	_, exists := r.transfers[fid]
	if !exists {
		return nil
	}

	delete(r.transfers, fid)
	return r.save()
}

// RemoveTransfers removes the transfers from the map.
func (r *Received) RemoveTransfers(fidList ...ftCrypto.ID) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	for _, fid := range fidList {
		delete(r.transfers, fid)
	}

	return r.save()
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// save stores a list of file IDs in the map to storage.
func (r *Received) save() error {
	if r.disableKV {
		return nil
	}

	data, err := marshalReceivedTransfersMap(r.transfers)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   receivedTransfersStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return r.kv.Set(receivedTransfersStoreKey, obj)
}

// marshalReceivedTransfersMap serialises the list of file IDs from a
// ReceivedTransfer map.
func marshalReceivedTransfersMap(
	transfers map[ftCrypto.ID]*ReceivedTransfer) ([]byte, error) {
	fidList := make([]ftCrypto.ID, 0, len(transfers))

	for fid := range transfers {
		fidList = append(fidList, fid)
	}

	return json.Marshal(fidList)
}
