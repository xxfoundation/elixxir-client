////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// Tests that NewOrLoadSent returns a new Sent when none exist in storage and
// that the list of unsent parts is nil.
func TestNewOrLoadSent_New(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	expected := &Sent{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv,
	}

	s, unsentParts, err := NewOrLoadSent(kv)
	if err != nil {
		t.Errorf("NewOrLoadSent returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, s) {
		t.Errorf("New Sent does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, s)
	}

	if unsentParts != nil {
		t.Errorf("List of parts should be nil when not loading: %+v",
			unsentParts)
	}
}

// Tests that NewOrLoadSent returns a loaded Sent when one exist in storage and
// that the list of unsent parts is correct.
func TestNewOrLoadSent_Load(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	s, _, err := NewOrLoadSent(kv)
	if err != nil {
		t.Errorf("Failed to make new Sent: %+v", err)
	}
	var expectedUnsentParts []Part

	// Create and add transfers to map and save
	for i := 0; i < 10; i++ {
		key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
		tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
		parts, file := generateTestParts(uint16(10 + i))
		st, err2 := s.AddTransfer(
			id.NewIdFromString("recipient"+strconv.Itoa(i), id.User, t),
			&key, &tid, "file"+strconv.Itoa(i), uint32(len(file)), parts,
			uint16(2*(10+i)))
		if err2 != nil {
			t.Errorf("Failed to add transfer #%d: %+v", i, err2)
		}
		expectedUnsentParts = append(expectedUnsentParts, st.GetUnsentParts()...)
	}

	if err = s.save(); err != nil {
		t.Errorf("Failed to make save filled Sent: %+v", err)
	}

	// Load Sent
	loadedSent, unsentParts, err := NewOrLoadSent(kv)
	if err != nil {
		t.Errorf("Failed to load Sent: %+v", err)
	}

	// Check that the loaded Sent matches original
	require.Len(t, loadedSent.transfers, len(s.transfers))
	for key, val := range s.transfers {
		loaded, ok := loadedSent.transfers[key]
		require.True(t, ok)
		require.Equal(t, val.numParts, loaded.numParts)
		require.Equal(t, val.partStatus, loaded.partStatus)
		require.Equal(t, val.cypherManager, val.cypherManager)

	}

	sort.Slice(unsentParts, func(i, j int) bool {
		switch bytes.Compare(unsentParts[i].TransferID()[:],
			unsentParts[j].TransferID()[:]) {
		case -1:
			return true
		case 1:
			return false
		default:
			return unsentParts[i].partNum < unsentParts[j].partNum
		}
	})

	sort.Slice(expectedUnsentParts, func(i, j int) bool {
		switch bytes.Compare(expectedUnsentParts[i].TransferID()[:],
			expectedUnsentParts[j].TransferID()[:]) {
		case -1:
			return true
		case 1:
			return false
		default:
			return expectedUnsentParts[i].partNum < expectedUnsentParts[j].partNum
		}
	})
	require.Len(t, unsentParts, len(expectedUnsentParts))

	expectedMap := make(map[string]struct{})
	for _, part := range expectedUnsentParts {
		expectedMap[part.transfer.tid.String()] = struct{}{}
	}

	for _, part := range unsentParts {
		_, ok := expectedMap[part.transfer.tid.String()]
		require.True(t, ok)
	}
}

// Tests that Sent.AddTransfer makes a new transfer and adds it to the list.
func TestSent_AddTransfer(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	s, _, _ := NewOrLoadSent(kv)

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
	parts, file := generateTestParts(10)

	st, err := s.AddTransfer(id.NewIdFromString("recipient", id.User, t),
		&key, &tid, "file", uint32(len(file)), parts, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Check that the transfer was added
	if _, exists := s.transfers[*st.tid]; !exists {
		t.Errorf("No transfer with ID %s exists.", st.tid)
	}
}

// Tests that Sent.AddTransfer returns an error when adding a transfer ID that
// already exists.
func TestSent_AddTransfer_TransferAlreadyExists(t *testing.T) {
	tid := &ftCrypto.TransferID{0}
	s := &Sent{
		transfers: map[ftCrypto.TransferID]*SentTransfer{*tid: nil},
	}

	expectedErr := fmt.Sprintf(errAddExistingSentTransfer, tid)
	_, err := s.AddTransfer(nil, nil, tid, "", 0, nil, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Received unexpected error when adding transfer that already "+
			"exists.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Sent.GetTransfer returns the expected transfer.
func TestSent_GetTransfer(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	s, _, _ := NewOrLoadSent(kv)

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
	parts, file := generateTestParts(10)

	st, err := s.AddTransfer(id.NewIdFromString("recipient", id.User, t),
		&key, &tid, "file", uint32(len(file)), parts, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Check that the transfer was added
	receivedSt, exists := s.GetTransfer(st.tid)
	if !exists {
		t.Errorf("No transfer with ID %s exists.", st.tid)
	}

	if !reflect.DeepEqual(st, receivedSt) {
		t.Errorf("Received SentTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", st, receivedSt)
	}
}

// Tests that Sent.RemoveTransfer removes the transfer from the list.
func TestSent_RemoveTransfer(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	s, _, _ := NewOrLoadSent(kv)

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
	parts, file := generateTestParts(10)

	st, err := s.AddTransfer(id.NewIdFromString("recipient", id.User, t),
		&key, &tid, "file", uint32(len(file)), parts, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Delete the transfer
	err = s.RemoveTransfer(st.tid)
	if err != nil {
		t.Errorf("RemoveTransfer returned an error: %+v", err)
	}

	// Check that the transfer was deleted
	_, exists := s.GetTransfer(st.tid)
	if exists {
		t.Errorf("Transfer %s exists.", st.tid)
	}

	// Remove transfer that was already removed
	err = s.RemoveTransfer(st.tid)
	if err != nil {
		t.Errorf("RemoveTransfer returned an error: %+v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that Sent.save saves the transfer ID list to storage by trying to load
// it after a save.
func TestSent_save(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	s, _, _ := NewOrLoadSent(kv)
	s.transfers = map[ftCrypto.TransferID]*SentTransfer{
		{0}: nil, {1}: nil,
		{2}: nil, {3}: nil,
	}

	err := s.save()
	if err != nil {
		t.Errorf("Failed to save transfer ID list: %+v", err)
	}

	_, err = s.kv.Get(makeSentTransferKvKey(), sentTransfersStoreVersion)
	if err != nil {
		t.Errorf("Failed to load transfer ID list: %+v", err)
	}
}

// Tests that the transfer IDs keys in the map marshalled by
// marshalSentTransfersMap and unmarshalled by unmarshalTransferIdList match the
// original.
func Test_marshalSentTransfersMap_unmarshalTransferIdList(t *testing.T) {
	// Build map of transfer IDs
	transfers := make(map[ftCrypto.TransferID]*SentTransfer, 10)
	for i := 0; i < 10; i++ {
		tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
		transfers[tid] = nil
	}

	data, err := marshalSentTransfersMap(transfers)
	if err != nil {
		t.Errorf("marshalSentTransfersMap returned an error: %+v", err)
	}

	tidList, err := unmarshalTransferIdList(data)
	if err != nil {
		t.Errorf("unmarshalSentTransfer returned an error: %+v", err)
	}

	for _, tid := range tidList {
		if _, exists := transfers[tid]; exists {
			delete(transfers, tid)
		} else {
			t.Errorf("Transfer %s does not exist in list.", tid)
		}
	}

	if len(transfers) != 0 {
		t.Errorf("%d transfers not in unmarshalled list: %v",
			len(transfers), transfers)
	}
}
