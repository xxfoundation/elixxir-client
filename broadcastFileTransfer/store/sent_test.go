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
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Tests that NewOrLoadSent returns a new Sent when none exist in storage and
// that the list of unsent parts is nil.
func TestNewOrLoadSent_New(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &Sent{
		transfers: make(map[ftCrypto.ID]*SentTransfer),
		kv:        kv.Prefix(sentTransfersStorePrefix),
	}

	s, unsentParts, sentParts, err := NewOrLoadSent(kv)
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

	if sentParts != nil {
		t.Errorf("List of parts should be nil when not loading: %+v",
			sentParts)
	}
}

// Tests that NewOrLoadSent returns a loaded Sent when one exist in storage and
// that the list of unsent parts is correct.
func TestNewOrLoadSent_Load(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	prng := rand.New(rand.NewSource(42))
	s, _, _, err := NewOrLoadSent(kv)
	if err != nil {
		t.Errorf("Failed to make new Sent: %+v", err)
	}
	var expectedUnsentParts, expectedSentParts []*Part
	fileData := make([]byte, 64)

	// Create and add transfers to map and save
	for i := 0; i < 10; i++ {
		key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
		prng.Read(fileData)
		fid := ftCrypto.NewID(fileData)
		parts, file := generateTestParts(uint16(10 + i))
		st, err2 := s.AddTransfer(
			id.NewIdFromString("recipient"+strconv.Itoa(i), id.User, t), &key,
			fid, "file"+strconv.Itoa(i), uint32(len(file)), parts,
			uint16(2*(10+i)))
		if err2 != nil {
			t.Errorf("Failed to add transfer #%d: %+v", i, err2)
		}
		expectedUnsentParts = append(expectedUnsentParts, st.GetUnsentParts()...)
		expectedSentParts = append(expectedSentParts, st.GetSentParts()...)
	}
	if err = s.save(); err != nil {
		t.Errorf("Failed to make save filled Sent: %+v", err)
	}

	// Load Sent
	loadedSent, unsentParts, sentParts, err := NewOrLoadSent(kv)
	if err != nil {
		t.Errorf("Failed to load Sent: %+v", err)
	}

	// Check that the loaded Sent matches original
	if !reflect.DeepEqual(s, loadedSent) {
		t.Errorf("Loaded Sent does not match original."+
			"\nexpected: %v\nreceived: %v", s, loadedSent)
	}

	sort.Slice(unsentParts, func(i, j int) bool {
		switch bytes.Compare(unsentParts[i].FileID().Marshal(),
			unsentParts[j].FileID().Marshal()) {
		case -1:
			return true
		case 1:
			return false
		default:
			return unsentParts[i].partNum < unsentParts[j].partNum
		}
	})

	sort.Slice(expectedUnsentParts, func(i, j int) bool {
		switch bytes.Compare(expectedUnsentParts[i].FileID().Marshal(),
			expectedUnsentParts[j].FileID().Marshal()) {
		case -1:
			return true
		case 1:
			return false
		default:
			return expectedUnsentParts[i].partNum < expectedUnsentParts[j].partNum
		}
	})

	// Check that the unsent parts matches expected
	if !reflect.DeepEqual(expectedUnsentParts, unsentParts) {
		t.Errorf("Incorrect unsent parts.\nexpected: %v\nreceived: %v",
			expectedUnsentParts, unsentParts)
	}

	sort.Slice(sentParts, func(i, j int) bool {
		switch bytes.Compare(sentParts[i].FileID().Marshal(),
			sentParts[j].FileID().Marshal()) {
		case -1:
			return true
		case 1:
			return false
		default:
			return sentParts[i].partNum < sentParts[j].partNum
		}
	})

	sort.Slice(expectedSentParts, func(i, j int) bool {
		switch bytes.Compare(expectedSentParts[i].FileID().Marshal(),
			expectedSentParts[j].FileID().Marshal()) {
		case -1:
			return true
		case 1:
			return false
		default:
			return expectedSentParts[i].partNum < expectedSentParts[j].partNum
		}
	})

	// Check that the sent parts matches expected
	if !reflect.DeepEqual(expectedSentParts, sentParts) {
		t.Errorf("Incorrect sent parts.\nexpected: %v\nreceived: %v",
			expectedSentParts, sentParts)
	}
}

// Tests that Sent.AddTransfer makes a new transfer and adds it to the list.
func TestSent_AddTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s, _, _, _ := NewOrLoadSent(kv)

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))
	parts, file := generateTestParts(10)

	st, err := s.AddTransfer(id.NewIdFromString("recipient", id.User, t), &key,
		fid, "file", uint32(len(file)), parts, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Check that the transfer was added
	if _, exists := s.transfers[st.fid]; !exists {
		t.Errorf("No transfer with ID %s exists.", st.fid)
	}
}

// Tests that Sent.AddTransfer returns an error when adding a file ID that
// already exists.
func TestSent_AddTransfer_TransferAlreadyExists(t *testing.T) {
	fid := ftCrypto.ID{0}
	s := &Sent{
		transfers: map[ftCrypto.ID]*SentTransfer{fid: nil},
	}

	expectedErr := fmt.Sprintf(errAddExistingSentTransfer, fid)
	_, err := s.AddTransfer(nil, nil, fid, "", 0, nil, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Received unexpected error when adding transfer that already "+
			"exists.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Sent.GetTransfer returns the expected transfer.
func TestSent_GetTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s, _, _, _ := NewOrLoadSent(kv)

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))
	parts, file := generateTestParts(10)

	st, err := s.AddTransfer(id.NewIdFromString("recipient", id.User, t),
		&key, fid, "file", uint32(len(file)), parts, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Check that the transfer was added
	receivedSt, exists := s.GetTransfer(st.fid)
	if !exists {
		t.Errorf("No transfer with ID %s exists.", st.fid)
	}

	if !reflect.DeepEqual(st, receivedSt) {
		t.Errorf("Received SentTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", st, receivedSt)
	}
}

// Tests that Sent.RemoveTransfer removes the transfer from the list.
func TestSent_RemoveTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s, _, _, _ := NewOrLoadSent(kv)

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))
	parts, file := generateTestParts(10)

	st, err := s.AddTransfer(id.NewIdFromString("recipient", id.User, t), &key,
		fid, "file", uint32(len(file)), parts, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Delete the transfer
	err = s.RemoveTransfer(st.fid)
	if err != nil {
		t.Errorf("RemoveTransfer returned an error: %+v", err)
	}

	// Check that the transfer was deleted
	_, exists := s.GetTransfer(st.fid)
	if exists {
		t.Errorf("File %s exists.", st.fid)
	}

	// Remove transfer that was already removed
	err = s.RemoveTransfer(st.fid)
	if err != nil {
		t.Errorf("RemoveTransfer returned an error: %+v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that Sent.save saves the file ID list to storage by trying to load it
// after a save.
func TestSent_save(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s, _, _, _ := NewOrLoadSent(kv)
	s.transfers = map[ftCrypto.ID]*SentTransfer{
		{0}: nil, {1}: nil,
		{2}: nil, {3}: nil,
	}

	err := s.save()
	if err != nil {
		t.Errorf("Failed to save file ID list: %+v", err)
	}

	_, err = s.kv.Get(sentTransfersStoreKey, sentTransfersStoreVersion)
	if err != nil {
		t.Errorf("Failed to load file ID list: %+v", err)
	}
}

// Tests that the file IDs keys in the map marshalled by marshalSentTransfersMap
// and unmarshalled by unmarshalFileIdList match the original.
func Test_marshalSentTransfersMap_unmarshalFileIdList(t *testing.T) {
	// Build map of file IDs
	transfers := make(map[ftCrypto.ID]*SentTransfer, 10)
	for i := 0; i < 10; i++ {
		fid := ftCrypto.NewID([]byte("fileData"))
		transfers[fid] = nil
	}

	data, err := marshalSentTransfersMap(transfers)
	if err != nil {
		t.Errorf("marshalSentTransfersMap returned an error: %+v", err)
	}

	fidList, err := unmarshalFileIdList(data)
	if err != nil {
		t.Errorf("unmarshalFileIdList returned an error: %+v", err)
	}

	for _, fid := range fidList {
		if _, exists := transfers[fid]; exists {
			delete(transfers, fid)
		} else {
			t.Errorf("File %s does not exist in list.", fid)
		}
	}

	if len(transfers) != 0 {
		t.Errorf("%d transfers not in unmarshalled list: %v",
			len(transfers), transfers)
	}
}
