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

// Tests that NewOrLoadReceived returns a new Received when none exist in
// storage and that the list of incomplete transfers is nil.
func TestNewOrLoadReceived_New(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &Received{
		transfers: make(map[ftCrypto.ID]*ReceivedTransfer),
		kv:        kv.Prefix(receivedTransfersStorePrefix),
	}

	r, incompleteTransfers, err := NewOrLoadReceived(kv)
	if err != nil {
		t.Errorf("NewOrLoadReceived returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, r) {
		t.Errorf("New Received does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, r)
	}

	if incompleteTransfers != nil {
		t.Errorf("List of incomplete transfers should be nil when not "+
			"loading: %+v", incompleteTransfers)
	}
}

// Tests that NewOrLoadReceived returns a loaded Received when one exist in
// storage and that the list of incomplete transfers is correct.
func TestNewOrLoadReceived_Load(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	prng := rand.New(rand.NewSource(42))
	r, _, err := NewOrLoadReceived(kv)
	if err != nil {
		t.Errorf("Failed to make new Received: %+v", err)
	}
	var expectedIncompleteTransfers []*ReceivedTransfer
	fileData := make([]byte, 64)

	// Create and add transfers to map and save
	for i := 0; i < 2; i++ {
		recipient := id.NewIdFromUInt(uint64(i), id.User, t)
		key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
		prng.Read(fileData)
		fid := ftCrypto.NewID(fileData)
		rt, err2 := r.AddTransfer(recipient, &key, fid, "file"+strconv.Itoa(i),
			[]byte("transferMAC"+strconv.Itoa(i)), 128, 10, 20)
		if err2 != nil {
			t.Errorf("Failed to add transfer #%d: %+v", i, err2)
		}
		expectedIncompleteTransfers = append(expectedIncompleteTransfers, rt)
	}
	if err = r.save(); err != nil {
		t.Errorf("Failed to make save filled Receivced: %+v", err)
	}

	// Load Received
	loadedReceived, incompleteTransfers, err := NewOrLoadReceived(kv)
	if err != nil {
		t.Errorf("Failed to load Received: %+v", err)
	}

	// Check that the loaded Received matches original
	if !reflect.DeepEqual(r, loadedReceived) {
		t.Errorf("Loaded Received does not match original."+
			"\nexpected: %#v\nreceived: %#v", r, loadedReceived)
	}

	sort.Slice(incompleteTransfers, func(i, j int) bool {
		return bytes.Compare(incompleteTransfers[i].FileID().Marshal(),
			incompleteTransfers[j].FileID().Marshal()) == -1
	})

	sort.Slice(expectedIncompleteTransfers, func(i, j int) bool {
		return bytes.Compare(expectedIncompleteTransfers[i].FileID().Marshal(),
			expectedIncompleteTransfers[j].FileID().Marshal()) == -1
	})

	// Check that the incomplete transfers matches expected
	if !reflect.DeepEqual(expectedIncompleteTransfers, incompleteTransfers) {
		t.Errorf("Incorrect incomplete transfers.\nexpected: %v\nreceived: %v",
			expectedIncompleteTransfers, incompleteTransfers)
	}
}

// Tests that Received.AddTransfer makes a new transfer and adds it to the list.
func TestReceived_AddTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	r, _, _ := NewOrLoadReceived(kv)

	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))

	rt, err := r.AddTransfer(
		recipient, &key, fid, "file", []byte("transferMAC"), 128, 10, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Check that the transfer was added
	if _, exists := r.transfers[rt.fid]; !exists {
		t.Errorf("No transfer with ID %s exists.", rt.fid)
	}
}

// Tests that Received.AddTransfer returns an error when adding a file ID that
// already exists.
func TestReceived_AddTransfer_TransferAlreadyExists(t *testing.T) {
	fid := ftCrypto.ID{0}
	r := &Received{
		transfers: map[ftCrypto.ID]*ReceivedTransfer{fid: nil},
	}

	expectedErr := fmt.Sprintf(errAddExistingReceivedTransfer, fid)
	_, err := r.AddTransfer(nil, nil, fid, "", nil, 0, 0, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Received unexpected error when adding transfer that already "+
			"exists.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Received.GetTransfer returns the expected transfer.
func TestReceived_GetTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	r, _, _ := NewOrLoadReceived(kv)

	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))

	rt, err := r.AddTransfer(
		recipient, &key, fid, "file", []byte("transferMAC"), 128, 10, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Check that the transfer was added
	receivedRt, exists := r.GetTransfer(rt.fid)
	if !exists {
		t.Errorf("No transfer with ID %s exists.", rt.fid)
	}

	if !reflect.DeepEqual(rt, receivedRt) {
		t.Errorf("Received ReceivedTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", rt, receivedRt)
	}
}

// Tests that Sent.RemoveTransfer removes the transfer from the list.
func TestReceived_RemoveTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	r, _, _ := NewOrLoadReceived(kv)

	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))

	rt, err := r.AddTransfer(
		recipient, &key, fid, "file", []byte("transferMAC"), 128, 10, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Delete the transfer
	err = r.RemoveTransfer(rt.fid)
	if err != nil {
		t.Errorf("RemoveTransfer returned an error: %+v", err)
	}

	// Check that the transfer was deleted
	_, exists := r.GetTransfer(rt.fid)
	if exists {
		t.Errorf("File %s exists.", rt.fid)
	}

	// Remove transfer that was already removed
	err = r.RemoveTransfer(rt.fid)
	if err != nil {
		t.Errorf("RemoveTransfer returned an error: %+v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that Received.save saves the file ID list to storage by trying to load
// it after a save.
func TestReceived_save(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	r, _, _ := NewOrLoadReceived(kv)
	r.transfers = map[ftCrypto.ID]*ReceivedTransfer{
		{0}: nil, {1}: nil,
		{2}: nil, {3}: nil,
	}

	err := r.save()
	if err != nil {
		t.Errorf("Failed to save file ID list: %+v", err)
	}

	_, err = r.kv.Get(receivedTransfersStoreKey, receivedTransfersStoreVersion)
	if err != nil {
		t.Errorf("Failed to load file ID list: %+v", err)
	}
}

// Tests that the file IDs keys in the map marshalled by
// marshalReceivedTransfersMap and unmarshalled by unmarshalFileIdList match
// the original.
func Test_marshalReceivedTransfersMap_unmarshalFileIdList(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	fileData := make([]byte, 64)

	// Build map of file IDs
	transfers := make(map[ftCrypto.ID]*ReceivedTransfer, 10)
	for i := 0; i < 10; i++ {
		prng.Read(fileData)
		fid := ftCrypto.NewID(fileData)
		transfers[fid] = nil
	}

	data, err := marshalReceivedTransfersMap(transfers)
	if err != nil {
		t.Errorf("marshalReceivedTransfersMap returned an error: %+v", err)
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
