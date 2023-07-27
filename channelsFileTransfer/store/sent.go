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
	"time"

	"github.com/pkg/errors"

	"gitlab.com/elixxir/client/v4/collective/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage keys and versions.
const (
	sentTransfersStorePrefix  = "SentFileTransfersPrefix"
	sentTransfersStoreKey     = "SentFileTransfers"
	sentTransfersStoreVersion = 0
)

// Error messages.
const (
	// NewOrLoadSent
	errLoadSent      = "error loading sent transfer list from storage: %+v"
	errUnmarshalSent = "could not unmarshal sent transfer list: %+v"

	// Sent.AddTransfer
	errAddExistingSentTransfer = "upload already exists in map"
	errNewSentTransfer         = "failed to make new sent transfer: %+v"
)

// Sent contains a list of all sent transfers.
type Sent struct {
	transfers map[ftCrypto.ID]*SentTransfer

	mux sync.RWMutex
	kv  versioned.KV
}

// NewOrLoadSent attempts to load Sent from storage. Or if none exist, then a
// new Sent is returned. A list of file IDs for all incomplete sends is also
// returned.
func NewOrLoadSent(kv versioned.KV) (*Sent, []ftCrypto.ID, error) {
	stkv, err := kv.Prefix(sentTransfersStorePrefix)
	if err != nil {
		return nil, nil, err
	}
	s := &Sent{
		transfers: make(map[ftCrypto.ID]*SentTransfer),
		kv:        stkv,
	}

	obj, err := s.kv.Get(sentTransfersStoreKey, sentTransfersStoreVersion)
	if err != nil {
		if !kv.Exists(err) {
			// Return the new Sent if none exists in storage
			return s, nil, nil
		} else {
			// Return other errors
			return nil, nil, errors.Errorf(errLoadSent, err)
		}
	}

	// Load list of saved sent transfers from storage
	fidList, err := unmarshalFileIdList(obj.Data)
	if err != nil {
		return nil, nil, errors.Errorf(errUnmarshalSent, err)
	}

	return s, fidList, nil
}

// LoadTransfer loads the sent transfer with the file ID from storage into Sent
// and returns it.
func (s *Sent) LoadTransfer(
	fid ftCrypto.ID, parts [][]byte) (*SentTransfer, error) {
	// Load sent transfers from storage
	st, err := loadSentTransfer(fid, parts, s.kv)
	if err != nil {
		return nil, err
	}

	// Add sent transfer to transfer list
	if _, exists := s.transfers[fid]; exists {
		return nil, errors.New(errAddExistingSentTransfer)
	}
	s.transfers[fid] = st

	return st, nil
}

// AddTransfer creates a SentTransfer and adds it to the map keyed on its file
// ID.
func (s *Sent) AddTransfer(recipient *id.ID, sentTimestamp time.Time,
	key *ftCrypto.TransferKey, mac []byte, fid ftCrypto.ID, fileSize uint32,
	parts [][]byte, numFps uint16, retry float32) (*SentTransfer, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	st, exists := s.transfers[fid]
	if exists {
		return nil, errors.New(errAddExistingSentTransfer)
	}

	st, err := newSentTransfer(recipient, sentTimestamp, key, mac, fid,
		fileSize, parts, numFps, retry, s.kv)
	if err != nil {
		return nil, errors.Errorf(errNewSentTransfer, fid)
	}

	s.transfers[fid] = st

	return st, s.save()
}

// GetTransfer returns the SentTransfer with the given file ID or false if none
// exists.
func (s *Sent) GetTransfer(fid ftCrypto.ID) (*SentTransfer, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	st, exists := s.transfers[fid]
	return st, exists
}

// RemoveTransfer removes the transfer from the map. If no transfer exists,
// returns nil. Only errors due to saving to storage are returned.
func (s *Sent) RemoveTransfer(fid ftCrypto.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	_, exists := s.transfers[fid]
	if !exists {
		return nil
	}

	delete(s.transfers, fid)
	return s.save()
}

// RemoveTransfers removes the transfers from the map.
func (s *Sent) RemoveTransfers(fidList ...ftCrypto.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, fid := range fidList {
		delete(s.transfers, fid)
	}

	return s.save()
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// save stores a list of file IDs in the map to storage.
func (s *Sent) save() error {
	data, err := marshalSentTransfersMap(s.transfers)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   sentTransfersStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return s.kv.Set(sentTransfersStoreKey, obj)
}

// marshalSentTransfersMap serialises the list of file IDs from a SentTransfer
// map.
func marshalSentTransfersMap(transfers map[ftCrypto.ID]*SentTransfer) (
	[]byte, error) {
	fidList := make([]ftCrypto.ID, 0, len(transfers))

	for fid := range transfers {
		fidList = append(fidList, fid)
	}

	return json.Marshal(fidList)
}

// unmarshalFileIdList deserializes the data into a list of file IDs.
func unmarshalFileIdList(data []byte) ([]ftCrypto.ID, error) {
	var fidList []ftCrypto.ID
	err := json.Unmarshal(data, &fidList)
	if err != nil {
		return nil, err
	}

	return fidList, nil
}
