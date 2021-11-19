////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"strings"
	"testing"
)

// Tests that NewSentFileTransfers creates a new object with empty maps and that
// it is saved to storage
func TestNewSentFileTransfers(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedSFT := &SentFileTransfers{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersPrefix),
	}

	sft, err := NewSentFileTransfers(kv)

	// Check that the new SentFileTransfers matches the expected
	if !reflect.DeepEqual(expectedSFT, sft) {
		t.Errorf("New SentFileTransfers does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedSFT, sft)
	}

	// Ensure that the transfer list is saved to storage
	_, err = expectedSFT.kv.Get(sentFileTransfersKey, sentFileTransfersVersion)
	if err != nil {
		t.Errorf("Failed to load transfer list from storage: %+v", err)
	}
}

// Tests that SentFileTransfers.AddTransfer adds a new transfer to the map and
// that its ID is saved to the list in storage.
func TestSentFileTransfers_AddTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfers(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfers: %+v", err)
	}

	// Generate info for new transfer
	recipient := id.NewIdFromString("recipientID", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	parts, _ := newRandomPartSlice(16, prng, t)
	numFps := uint16(24)

	// Add the transfer
	tid, err := sft.AddTransfer(recipient, key, parts, numFps, nil, 0, prng)
	if err != nil {
		t.Errorf("AddTransfer returned an error: %+v", err)
	}

	_, exists := sft.transfers[tid]
	if !exists {
		t.Errorf("New transfer %s does not exist in map.", tid)
	}

	list, err := sft.loadTransfersList()
	if err != nil {
		t.Errorf("Failed to load transfer list from storage: %+v", err)
	}

	if list[0] != tid {
		t.Errorf("Transfer ID saved to storage does not match ID in memory."+
			"\nexpected: %s\nreceived: %s", tid, list[0])
	}
}

// Error path: tests that SentFileTransfers.AddTransfer returns the expected
// error when the PRNG returns an error.
func TestSentFileTransfers_AddTransfer_NewTransferIdRngError(t *testing.T) {
	prng := NewPrngErr()
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfers(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfers: %+v", err)
	}

	// Add the transfer
	expectedErr := strings.Split(addTransferNewIdErr, "%")[0]
	_, err = sft.AddTransfer(nil, ftCrypto.TransferKey{}, nil, 0, nil, 0, prng)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("AddTransfer did not return the expected error when the PRNG "+
			"should have errored.\nexpected: %s\nrecieved: %+v", expectedErr, err)
	}
}

// Tests that SentFileTransfers.GetTransfer returns the expected transfer from
// the map.
func TestSentFileTransfers_GetTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfers(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfers: %+v", err)
	}

	// Generate info for new transfer
	recipient := id.NewIdFromString("recipientID", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	parts, _ := newRandomPartSlice(16, prng, t)
	numFps := uint16(24)

	tid, err := sft.AddTransfer(recipient, key, parts, numFps, nil, 0, prng)
	if err != nil {
		t.Errorf("AddTransfer returned an error: %+v", err)
	}

	transfer, err := sft.GetTransfer(tid)
	if err != nil {
		t.Errorf("GetTransfer returned an error: %+v", err)
	}

	if !reflect.DeepEqual(sft.transfers[tid], transfer) {
		t.Errorf("Received transfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", sft.transfers[tid], transfer)
	}
}

// Error path: tests that SentFileTransfers.GetTransfer returns the expected
// error when the map is empty/there is no transfer with the given transfer ID.
func TestSentFileTransfers_GetTransfer_NoTransferError(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfers(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfers: %+v", err)
	}

	tid, _ := ftCrypto.NewTransferID(prng)

	_, err = sft.GetTransfer(tid)
	if err == nil || err.Error() != getSentTransferErr {
		t.Errorf("GetTransfer did not return the expected error when it is "+
			"empty.\nexpected: %s\nreceived: %+v", getSentTransferErr, err)
	}
}

// Tests that SentFileTransfers.DeleteTransfer removes the transfer from the map
// in memory and from the list in storage.
func TestSentFileTransfers_DeleteTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfers(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfers: %+v", err)
	}

	// Generate info for new transfer
	recipient := id.NewIdFromString("recipientID", id.User, t)
	key, _ := ftCrypto.NewTransferKey(prng)
	parts, _ := newRandomPartSlice(16, prng, t)
	numFps := uint16(24)

	tid, err := sft.AddTransfer(recipient, key, parts, numFps, nil, 0, prng)
	if err != nil {
		t.Errorf("AddTransfer returned an error: %+v", err)
	}

	err = sft.DeleteTransfer(tid)
	if err != nil {
		t.Errorf("DeleteTransfer returned an error: %+v", err)
	}

	transfer, err := sft.GetTransfer(tid)
	if err == nil {
		t.Errorf("No error getting transfer that should be deleted: %+v", transfer)
	}

	list, err := sft.loadTransfersList()
	if err != nil {
		t.Errorf("Failed to load transfer list from storage: %+v", err)
	}

	if len(list) > 0 {
		t.Errorf("Transfer ID list in storage not empty: %+v", list)
	}
}

// Error path: tests that SentFileTransfers.DeleteTransfer returns the expected
// error when the map is empty/there is no transfer with the given transfer ID.
func TestSentFileTransfers_DeleteTransfer_NoTransferError(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfers(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfers: %+v", err)
	}

	tid, _ := ftCrypto.NewTransferID(prng)

	err = sft.DeleteTransfer(tid)
	if err == nil || err.Error() != getSentTransferErr {
		t.Errorf("DeleteTransfer did not return the expected error when it is "+
			"empty.\nexpected: %s\nreceived: %+v", getSentTransferErr, err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that the SentFileTransfers loaded from storage by LoadSentFileTransfers
// matches the original in memory.
func TestLoadSentFileTransfers(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfers(kv)
	if err != nil {
		t.Fatalf("Failed to make new SentFileTransfers: %+v", err)
	}

	// Add 10 transfers to map in memory
	list := make([]ftCrypto.TransferID, 10)
	for i := range list {
		tid, st := newRandomSentTransfer(16, 24, sft.kv, t)
		sft.transfers[tid] = st
		list[i] = tid
	}

	// Save list to storage
	if err = sft.saveTransfersList(); err != nil {
		t.Errorf("Faileds to save transfers list: %+v", err)
	}

	// Load SentFileTransfers from storage
	loadedSFT, err := LoadSentFileTransfers(kv)
	if err != nil {
		t.Errorf("LoadSentFileTransfers returned an error: %+v", err)
	}

	// Equalize all progressCallbacks because reflect.DeepEqual does not seem
	// to work on function pointers
	for _, tid := range list {
		loadedSFT.transfers[tid].progressCallbacks = sft.transfers[tid].progressCallbacks
	}

	if !reflect.DeepEqual(sft, loadedSFT) {
		t.Errorf("Loaded SentFileTransfers does not match original in memory."+
			"\nexpected: %+v\nreceived: %+v", sft, loadedSFT)
	}
}

// Error path: tests that LoadSentFileTransfers returns the expected error when
// the transfer list cannot be loaded from storage.
func TestLoadSentFileTransfers_NoListInStorageError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadSentTransfersListErr, "%")[0]

	// Load SentFileTransfers from storage
	_, err := LoadSentFileTransfers(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("LoadSentFileTransfers did not return the expected error "+
			"when there is no transfer list saved in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that LoadSentFileTransfers returns the expected error when
// the first transfer loaded from storage does not exist.
func TestLoadSentFileTransfers_NoTransferInStorageError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadSentTransfersErr, "%")[0]

	// Save list of one transfer ID to storage
	obj := &versioned.Object{
		Version:   sentFileTransfersVersion,
		Timestamp: netTime.Now(),
		Data:      ftCrypto.UnmarshalTransferID([]byte("testID_01")).Bytes(),
	}
	err := kv.Prefix(sentFileTransfersPrefix).Set(
		sentFileTransfersKey, sentFileTransfersVersion, obj)

	// Load SentFileTransfers from storage
	_, err = LoadSentFileTransfers(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("LoadSentFileTransfers did not return the expected error "+
			"when there is no transfer saved in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the SentFileTransfers loaded from storage by
// NewOrLoadSentFileTransfers matches the original in memory.
func TestNewOrLoadSentFileTransfers(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfers(kv)
	if err != nil {
		t.Fatalf("Failed to make new SentFileTransfers: %+v", err)
	}

	// Add 10 transfers to map in memory
	list := make([]ftCrypto.TransferID, 10)
	for i := range list {
		tid, st := newRandomSentTransfer(16, 24, sft.kv, t)
		sft.transfers[tid] = st
		list[i] = tid
	}

	// Save list to storage
	if err = sft.saveTransfersList(); err != nil {
		t.Errorf("Faileds to save transfers list: %+v", err)
	}

	// Load SentFileTransfers from storage
	loadedSFT, err := NewOrLoadSentFileTransfers(kv)
	if err != nil {
		t.Errorf("NewOrLoadSentFileTransfers returned an error: %+v", err)
	}

	// Equalize all progressCallbacks because reflect.DeepEqual does not seem
	// to work on function pointers
	for _, tid := range list {
		loadedSFT.transfers[tid].progressCallbacks = sft.transfers[tid].progressCallbacks
	}

	if !reflect.DeepEqual(sft, loadedSFT) {
		t.Errorf("Loaded SentFileTransfers does not match original in memory."+
			"\nexpected: %+v\nreceived: %+v", sft, loadedSFT)
	}
}

// Tests that NewOrLoadSentFileTransfers returns a new SentFileTransfers when
// there is none in storage.
func TestNewOrLoadSentFileTransfers_NewSentFileTransfers(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	// Load SentFileTransfers from storage
	loadedSFT, err := NewOrLoadSentFileTransfers(kv)
	if err != nil {
		t.Errorf("NewOrLoadSentFileTransfers returned an error: %+v", err)
	}

	newSFT, _ := NewSentFileTransfers(kv)

	if !reflect.DeepEqual(newSFT, loadedSFT) {
		t.Errorf("Returned SentFileTransfers does not match new."+
			"\nexpected: %+v\nreceived: %+v", newSFT, loadedSFT)
	}
}

// Error path: tests that the NewOrLoadSentFileTransfers returns the expected
// error when the first transfer loaded from storage does not exist.
func TestNewOrLoadSentFileTransfers_NoTransferInStorageError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadSentTransfersErr, "%")[0]

	// Save list of one transfer ID to storage
	obj := &versioned.Object{
		Version:   sentFileTransfersVersion,
		Timestamp: netTime.Now(),
		Data:      ftCrypto.UnmarshalTransferID([]byte("testID_01")).Bytes(),
	}
	err := kv.Prefix(sentFileTransfersPrefix).Set(
		sentFileTransfersKey, sentFileTransfersVersion, obj)

	// Load SentFileTransfers from storage
	_, err = NewOrLoadSentFileTransfers(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("NewOrLoadSentFileTransfers did not return the expected "+
			"error when there is no transfer saved in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that SentFileTransfers.saveTransfersList saves all the transfer IDs to
// storage by loading them from storage via SentFileTransfers.loadTransfersList
// and comparing the list to the list in memory.
func TestSentFileTransfers_saveTransfersList_loadTransfersList(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	sft := &SentFileTransfers{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersPrefix),
	}

	// Add 10 transfers to map in memory
	for i := 0; i < 10; i++ {
		tid, st := newRandomSentTransfer(16, 24, sft.kv, t)
		sft.transfers[tid] = st
	}

	// Save transfer ID list to storage
	err := sft.saveTransfersList()
	if err != nil {
		t.Errorf("saveTransfersList returned an error: %+v", err)
	}

	// Get list from storage
	list, err := sft.loadTransfersList()
	if err != nil {
		t.Errorf("loadTransfersList returned an error: %+v", err)
	}

	// Check that the list has all the transfer IDs in memory
	for _, tid := range list {
		if _, exists := sft.transfers[tid]; !exists {
			t.Errorf("No transfer for ID %s exists.", tid)
		} else {
			delete(sft.transfers, tid)
		}
	}
}

// Tests that the transfer loaded by SentFileTransfers.loadTransfers from
// storage matches the original in memory
func TestSentFileTransfers_loadTransfers(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	sft := &SentFileTransfers{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersPrefix),
	}

	// Add 10 transfers to map in memory
	list := make([]ftCrypto.TransferID, 10)
	for i := range list {
		tid, st := newRandomSentTransfer(16, 24, sft.kv, t)
		sft.transfers[tid] = st
		list[i] = tid
	}

	// Load the transfers into a new SentFileTransfers
	loadedSft := &SentFileTransfers{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersPrefix),
	}
	err := loadedSft.loadTransfers(list)
	if err != nil {
		t.Errorf("loadTransfers returned an error: %+v", err)
	}

	// Equalize all progressCallbacks because reflect.DeepEqual does not seem
	// to work on function pointers
	for _, tid := range list {
		loadedSft.transfers[tid].progressCallbacks = sft.transfers[tid].progressCallbacks
	}

	if !reflect.DeepEqual(sft.transfers, loadedSft.transfers) {
		t.Errorf("Transfers loaded from storage does not match transfers in memory."+
			"\nexpected: %+v\nreceived: %+v", sft.transfers, loadedSft.transfers)
	}
}

// Tests that a transfer list marshalled with
// SentFileTransfers.marshalTransfersList and unmarshalled with
// unmarshalTransfersList matches the original.
func TestSentFileTransfers_marshalTransfersList_unmarshalTransfersList(t *testing.T) {
	prng := NewPrng(42)
	sft := &SentFileTransfers{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
	}

	// Add 10 transfers to map in memory
	for i := 0; i < 10; i++ {
		tid, _ := ftCrypto.NewTransferID(prng)
		sft.transfers[tid] = &SentTransfer{}
	}

	// Marshal into byte slice
	marshalledBytes := sft.marshalTransfersList()

	// Unmarshal marshalled bytes into transfer ID list
	list := unmarshalTransfersList(marshalledBytes)

	// Check that the list has all the transfer IDs in memory
	for _, tid := range list {
		if _, exists := sft.transfers[tid]; !exists {
			t.Errorf("No transfer for ID %s exists.", tid)
		} else {
			delete(sft.transfers, tid)
		}
	}
}
