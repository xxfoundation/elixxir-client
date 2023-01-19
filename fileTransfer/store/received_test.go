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
	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// Tests that NewOrLoadReceived returns a new Received when none exist in
// storage and that the list of incomplete transfers is nil.
func TestNewOrLoadReceived_New(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &Received{
		transfers: make(map[ftCrypto.TransferID]*ReceivedTransfer),
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
	r, _, err := NewOrLoadReceived(kv)
	if err != nil {
		t.Errorf("Failed to make new Received: %+v", err)
	}
	var expectedIncompleteTransfers []*ReceivedTransfer

	// Create and add transfers to map and save
	for i := 0; i < 2; i++ {
		key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
		tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
		rt, err2 := r.AddTransfer(&key, &tid, "file"+strconv.Itoa(i),
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
		return bytes.Compare(incompleteTransfers[i].TransferID()[:],
			incompleteTransfers[j].TransferID()[:]) == -1
	})

	sort.Slice(expectedIncompleteTransfers, func(i, j int) bool {
		return bytes.Compare(expectedIncompleteTransfers[i].TransferID()[:],
			expectedIncompleteTransfers[j].TransferID()[:]) == -1
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

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())

	rt, err := r.AddTransfer(
		&key, &tid, "file", []byte("transferMAC"), 128, 10, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Check that the transfer was added
	if _, exists := r.transfers[*rt.tid]; !exists {
		t.Errorf("No transfer with ID %s exists.", rt.tid)
	}
}

// Tests that Received.AddTransfer returns an error when adding a transfer ID
// that already exists.
func TestReceived_AddTransfer_TransferAlreadyExists(t *testing.T) {
	tid := &ftCrypto.TransferID{0}
	r := &Received{
		transfers: map[ftCrypto.TransferID]*ReceivedTransfer{*tid: nil},
	}

	expectedErr := fmt.Sprintf(errAddExistingReceivedTransfer, tid)
	_, err := r.AddTransfer(nil, tid, "", nil, 0, 0, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Received unexpected error when adding transfer that already "+
			"exists.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Received.GetTransfer returns the expected transfer.
func TestReceived_GetTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	r, _, _ := NewOrLoadReceived(kv)

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())

	rt, err := r.AddTransfer(
		&key, &tid, "file", []byte("transferMAC"), 128, 10, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Check that the transfer was added
	receivedRt, exists := r.GetTransfer(rt.tid)
	if !exists {
		t.Errorf("No transfer with ID %s exists.", rt.tid)
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

	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())

	rt, err := r.AddTransfer(
		&key, &tid, "file", []byte("transferMAC"), 128, 10, 20)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Delete the transfer
	err = r.RemoveTransfer(rt.tid)
	if err != nil {
		t.Errorf("RemoveTransfer returned an error: %+v", err)
	}

	// Check that the transfer was deleted
	_, exists := r.GetTransfer(rt.tid)
	if exists {
		t.Errorf("Transfer %s exists.", rt.tid)
	}

	// Remove transfer that was already removed
	err = r.RemoveTransfer(rt.tid)
	if err != nil {
		t.Errorf("RemoveTransfer returned an error: %+v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that Received.save saves the transfer ID list to storage by trying to
// load it after a save.
func TestReceived_save(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	r, _, _ := NewOrLoadReceived(kv)
	r.transfers = map[ftCrypto.TransferID]*ReceivedTransfer{
		{0}: nil, {1}: nil,
		{2}: nil, {3}: nil,
	}

	err := r.save()
	if err != nil {
		t.Errorf("Failed to save transfer ID list: %+v", err)
	}

	_, err = r.kv.Get(receivedTransfersStoreKey, receivedTransfersStoreVersion)
	if err != nil {
		t.Errorf("Failed to load transfer ID list: %+v", err)
	}
}

// Tests that the transfer IDs keys in the map marshalled by
// marshalReceivedTransfersMap and unmarshalled by unmarshalTransferIdList match
// the original.
func Test_marshalReceivedTransfersMap_unmarshalTransferIdList(t *testing.T) {
	// Build map of transfer IDs
	transfers := make(map[ftCrypto.TransferID]*ReceivedTransfer, 10)
	for i := 0; i < 10; i++ {
		tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
		transfers[tid] = nil
	}

	data, err := marshalReceivedTransfersMap(transfers)
	if err != nil {
		t.Errorf("marshalReceivedTransfersMap returned an error: %+v", err)
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
