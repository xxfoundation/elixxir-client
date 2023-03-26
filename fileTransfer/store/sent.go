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
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
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
	errLoadSent            = "error loading sent transfer list from storage: %+v"
	errUnmarshalSent       = "could not unmarshal sent transfer list: %+v"
	warnLoadSentTransfer   = "[FT] Failed to load sent transfer %d of %d with ID %s: %+v"
	errLoadAllSentTransfer = "failed to load all %d sent transfers"

	// Sent.AddTransfer
	errAddExistingSentTransfer = "sent transfer with ID %s already exists in map."
	errNewSentTransfer         = "failed to make new sent transfer: %+v"
)

// Sent contains a list of all sent transfers.
type Sent struct {
	transfers map[ftCrypto.TransferID]*SentTransfer

	mux sync.RWMutex
	kv  *utility.KV
}

// NewOrLoadSent attempts to load Sent from storage. Or if none exist, then a
// new Sent is returned. If running transfers were loaded from storage, a list
// of unsent parts is returned.
func NewOrLoadSent(kv *utility.KV) (*Sent, []Part, error) {
	s := &Sent{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv,
	}

	data, err := s.kv.Get(makeSentTransferKvKey(), sentTransfersStoreVersion)
	if err != nil {
		if !kv.Exists(err) {
			// Return the new Sent if none exists in storage
			return s, nil, nil
		} else {
			// Return other errors
			return nil, nil, errors.Errorf(errLoadSent, err)
		}
	}

	fmt.Println("loaded: ", base64.StdEncoding.EncodeToString(data))

	// Load list of saved sent transfers from storage
	tidList, err := unmarshalTransferIdList(data)
	if err != nil {
		return nil, nil, errors.Errorf(errUnmarshalSent, err)
	}
	fmt.Println("tid list ", tidList)
	// Load sent transfers from storage
	var errCount int
	var unsentParts []Part
	for i := range tidList {
		tid := tidList[i]
		s.transfers[tid], err = loadSentTransfer(&tid, s.kv)
		if err != nil {
			jww.WARN.Printf(warnLoadSentTransfer, i, len(tidList), tid, err)
			errCount++
			continue
		}

		if s.transfers[tid].Status() == Running {
			unsentParts =
				append(unsentParts, s.transfers[tid].GetUnsentParts()...)
		}
	}

	// Return an error if all transfers failed to load
	if len(tidList) > 0 && errCount == len(tidList) {
		return nil, nil, errors.Errorf(errLoadAllSentTransfer, len(tidList))
	}

	return s, unsentParts, nil
}

// AddTransfer creates a SentTransfer and adds it to the map keyed on its
// transfer ID.
func (s *Sent) AddTransfer(recipient *id.ID, key *ftCrypto.TransferKey,
	tid *ftCrypto.TransferID, fileName string, fileSize uint32, parts [][]byte,
	numFps uint16) (*SentTransfer, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	_, exists := s.transfers[*tid]
	if exists {
		return nil, errors.Errorf(errAddExistingSentTransfer, tid)
	}

	st, err := newSentTransfer(
		recipient, key, tid, fileName, fileSize, parts, numFps, s.kv)
	if err != nil {
		return nil, errors.Errorf(errNewSentTransfer, tid)
	}

	s.transfers[*tid] = st

	return st, s.save()
}

// GetTransfer returns the SentTransfer with the desiccated transfer ID or false
// if none exists.
func (s *Sent) GetTransfer(tid *ftCrypto.TransferID) (*SentTransfer, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	st, exists := s.transfers[*tid]
	return st, exists
}

// RemoveTransfer removes the transfer from the map. If no transfer exists,
// returns nil. Only errors due to saving to storage are returned.
func (s *Sent) RemoveTransfer(tid *ftCrypto.TransferID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	_, exists := s.transfers[*tid]
	if !exists {
		return nil
	}

	delete(s.transfers, *tid)
	return s.save()
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// save stores a list of transfer IDs in the map to storage.
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

	fmt.Println("saved: ", base64.StdEncoding.EncodeToString(data))
	fmt.Println("saved transfers: ", s.transfers)

	return s.kv.Set(makeSentTransferKvKey(), obj.Marshal())
}

// marshalSentTransfersMap serialises the list of transfer IDs from a
// SentTransfer map.
func marshalSentTransfersMap(transfers map[ftCrypto.TransferID]*SentTransfer) (
	[]byte, error) {
	tidList := make([]ftCrypto.TransferID, 0, len(transfers))

	for tid := range transfers {
		tidList = append(tidList, tid)
	}

	return json.Marshal(tidList)
}

// unmarshalTransferIdList deserializes the data into a list of transfer IDs.
func unmarshalTransferIdList(data []byte) ([]ftCrypto.TransferID, error) {
	var tidList []ftCrypto.TransferID
	err := json.Unmarshal(data, &tidList)
	if err != nil {
		return nil, err
	}

	return tidList, nil
}

func makeSentTransferKvKey() string {
	return sentTransfersStorePrefix + sentTransfersStoreKey
}
