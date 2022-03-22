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
	"sort"
	"strings"
	"testing"
)

// Tests that NewSentFileTransfersStore creates a new object with empty maps and
// that it is saved to storage
func TestNewSentFileTransfersStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedSFT := &SentFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersStorePrefix),
	}

	sft, err := NewSentFileTransfersStore(kv)

	// Check that the new SentFileTransfersStore matches the expected
	if !reflect.DeepEqual(expectedSFT, sft) {
		t.Errorf("New SentFileTransfersStore does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedSFT, sft)
	}

	// Ensure that the transfer list is saved to storage
	_, err = expectedSFT.kv.Get(
		sentFileTransfersStoreKey, sentFileTransfersStoreVersion)
	if err != nil {
		t.Errorf("Failed to load transfer list from storage: %+v", err)
	}
}

// Tests that SentFileTransfersStore.AddTransfer adds a new transfer to the map
// and that its ID is saved to the list in storage.
func TestSentFileTransfersStore_AddTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
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

// Error path: tests that SentFileTransfersStore.AddTransfer returns the
// expected error when the PRNG returns an error.
func TestSentFileTransfersStore_AddTransfer_NewTransferIdRngError(t *testing.T) {
	prng := NewPrngErr()
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
	}

	// Add the transfer
	expectedErr := strings.Split(addTransferNewIdErr, "%")[0]
	_, err = sft.AddTransfer(nil, ftCrypto.TransferKey{}, nil, 0, nil, 0, prng)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("AddTransfer did not return the expected error when the PRNG "+
			"should have errored.\nexpected: %s\nrecieved: %+v", expectedErr, err)
	}
}

// Tests that SentFileTransfersStore.GetTransfer returns the expected transfer
// from the map.
func TestSentFileTransfersStore_GetTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
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

// Error path: tests that SentFileTransfersStore.GetTransfer returns the
// expected error when the map is empty/there is no transfer with the given
// transfer ID.
func TestSentFileTransfersStore_GetTransfer_NoTransferError(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
	}

	tid, _ := ftCrypto.NewTransferID(prng)

	_, err = sft.GetTransfer(tid)
	if err == nil || err.Error() != getSentTransferErr {
		t.Errorf("GetTransfer did not return the expected error when it is "+
			"empty.\nexpected: %s\nreceived: %+v", getSentTransferErr, err)
	}
}

// Tests that SentFileTransfersStore.DeleteTransfer removes the transfer from
// the map in memory and from the list in storage.
func TestSentFileTransfersStore_DeleteTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
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
		t.Errorf("No error getting transfer that should be deleted: %+v",
			transfer)
	}

	list, err := sft.loadTransfersList()
	if err != nil {
		t.Errorf("Failed to load transfer list from storage: %+v", err)
	}

	if len(list) > 0 {
		t.Errorf("Transfer ID list in storage not empty: %+v", list)
	}
}

// Error path: tests that SentFileTransfersStore.DeleteTransfer returns the
// expected error when the map is empty/there is no transfer with the given
// transfer ID.
func TestSentFileTransfersStore_DeleteTransfer_NoTransferError(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
	}

	tid, _ := ftCrypto.NewTransferID(prng)

	err = sft.DeleteTransfer(tid)
	if err == nil || err.Error() != getSentTransferErr {
		t.Errorf("DeleteTransfer did not return the expected error when it is "+
			"empty.\nexpected: %s\nreceived: %+v", getSentTransferErr, err)
	}
}

// Tests that SentFileTransfersStore.GetUnsentParts returns the expected unsent
// parts for each transfer. Transfers are created with increasing number of
// parts. Each part of each transfer is set as either in-progress, finished, or
// unsent.
func TestSentFileTransfersStore_GetUnsentParts(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
	}

	n := uint16(3)
	expectedParts := make(map[ftCrypto.TransferID][]uint16, n)

	// Add new transfers
	for i := uint16(0); i < n; i++ {
		recipient := id.NewIdFromUInt(uint64(i), id.User, t)
		key, _ := ftCrypto.NewTransferKey(prng)
		parts, _ := newRandomPartSlice((i+1)*6, prng, t)
		numParts := uint16(len(parts))
		numFps := numParts * 3 / 2

		tid, err := sft.AddTransfer(recipient, key, parts, numFps, nil, 0, prng)
		if err != nil {
			t.Errorf("Failed to add transfer %d: %+v", i, err)
		}

		// Loop through each part and set it individually
		for j := uint16(0); j < numParts; j++ {
			switch ((j + i) % numParts) % 3 {
			case 0:
				// Part is sent (in-progress)
				_, _ = sft.transfers[tid].SetInProgress(id.Round(j), j)
			case 1:
				// Part is sent and arrived (finished)
				_, _ = sft.transfers[tid].SetInProgress(id.Round(j), j)
				_, _ = sft.transfers[tid].FinishTransfer(id.Round(j))
			case 2:
				// Part is unsent (neither in-progress nor arrived)
				expectedParts[tid] = append(expectedParts[tid], j)
			}
		}
	}

	unsentParts, err := sft.GetUnsentParts()
	if err != nil {
		t.Errorf("GetUnsentParts returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedParts, unsentParts) {
		t.Errorf("Unexpected unsent parts map.\nexpected: %+v\nreceived: %+v",
			expectedParts, unsentParts)
	}
}

// Tests that SentFileTransfersStore.GetSentRounds returns the expected transfer
// ID for each unfinished round. Transfers are created with increasing number of
// parts. Each part of each transfer is set as either in-progress, finished, or
// unsent.
func TestSentFileTransfersStore_GetSentRounds(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
	}

	n := uint16(3)
	expectedRounds := make(map[id.Round][]ftCrypto.TransferID)

	// Add new transfers
	for i := uint16(0); i < n; i++ {
		recipient := id.NewIdFromUInt(uint64(i), id.User, t)
		key, _ := ftCrypto.NewTransferKey(prng)
		parts, _ := newRandomPartSlice((i+1)*6, prng, t)
		numParts := uint16(len(parts))
		numFps := numParts * 3 / 2

		tid, err := sft.AddTransfer(recipient, key, parts, numFps, nil, 0, prng)
		if err != nil {
			t.Errorf("Failed to add transfer %d: %+v", i, err)
		}

		// Loop through each part and set it individually
		for j := uint16(0); j < numParts; j++ {
			rid := id.Round(j)
			switch j % 3 {
			case 0:
				// Part is sent (in-progress)
				_, _ = sft.transfers[tid].SetInProgress(id.Round(j), j)
				expectedRounds[rid] = append(expectedRounds[rid], tid)
			case 1:
				// Part is sent and arrived (finished)
				_, _ = sft.transfers[tid].SetInProgress(id.Round(j), j)
				_, _ = sft.transfers[tid].FinishTransfer(id.Round(j))
			case 2:
				// Part is unsent (neither in-progress nor arrived)
			}
		}
	}

	// Sort expected rounds map transfer IDs
	for _, tIDs := range expectedRounds {
		sort.Slice(tIDs,
			func(i, j int) bool { return tIDs[i].String() < tIDs[j].String() })
	}

	sentRounds := sft.GetSentRounds()

	// Sort sent rounds map transfer IDs
	for _, tIDs := range sentRounds {
		sort.Slice(tIDs,
			func(i, j int) bool { return tIDs[i].String() < tIDs[j].String() })
	}

	if !reflect.DeepEqual(expectedRounds, sentRounds) {
		t.Errorf("Unexpected sent rounds map.\nexpected: %+v\nreceived: %+v",
			expectedRounds, sentRounds)
	}
}

// Tests that SentFileTransfersStore.GetUnsentPartsAndSentRounds
func TestSentFileTransfersStore_GetUnsentPartsAndSentRounds(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new SentFileTransfersStore: %+v", err)
	}

	n := uint16(3)
	expectedParts := make(map[ftCrypto.TransferID][]uint16, n)
	expectedRounds := make(map[id.Round][]ftCrypto.TransferID)

	// Add new transfers
	for i := uint16(0); i < n; i++ {
		recipient := id.NewIdFromUInt(uint64(i), id.User, t)
		key, _ := ftCrypto.NewTransferKey(prng)
		parts, _ := newRandomPartSlice((i+1)*6, prng, t)
		numParts := uint16(len(parts))
		numFps := numParts * 3 / 2

		tid, err := sft.AddTransfer(recipient, key, parts, numFps, nil, 0, prng)
		if err != nil {
			t.Errorf("Failed to add transfer %d: %+v", i, err)
		}

		// Loop through each part and set it individually
		for j := uint16(0); j < numParts; j++ {
			rid := id.Round(j)
			switch j % 3 {
			case 0:
				// Part is sent (in-progress)
				_, _ = sft.transfers[tid].SetInProgress(rid, j)
				expectedRounds[rid] = append(expectedRounds[rid], tid)
			case 1:
				// Part is sent and arrived (finished)
				_, _ = sft.transfers[tid].SetInProgress(rid, j)
				_, _ = sft.transfers[tid].FinishTransfer(rid)
			case 2:
				// Part is unsent (neither in-progress nor arrived)
				expectedParts[tid] = append(expectedParts[tid], j)
			}
		}
	}

	// Sort expected rounds map transfer IDs
	for _, tIDs := range expectedRounds {
		sort.Slice(tIDs,
			func(i, j int) bool { return tIDs[i].String() < tIDs[j].String() })
	}

	unsentParts, sentRounds, err := sft.GetUnsentPartsAndSentRounds()
	if err != nil {
		t.Errorf("GetUnsentPartsAndSentRounds returned an error: %+v", err)
	}

	// Sort sent rounds map transfer IDs
	for _, tIDs := range sentRounds {
		sort.Slice(tIDs,
			func(i, j int) bool { return tIDs[i].String() < tIDs[j].String() })
	}

	if !reflect.DeepEqual(expectedParts, unsentParts) {
		t.Errorf("Unexpected unsent parts map.\nexpected: %+v\nreceived: %+v",
			expectedParts, unsentParts)
	}

	if !reflect.DeepEqual(expectedRounds, sentRounds) {
		t.Errorf("Unexpected sent rounds map.\nexpected: %+v\nreceived: %+v",
			expectedRounds, sentRounds)
	}

	unsentParts2, err := sft.GetUnsentParts()
	if err != nil {
		t.Errorf("GetUnsentParts returned an error: %+v", err)
	}

	if !reflect.DeepEqual(unsentParts, unsentParts2) {
		t.Errorf("Unsent parts from GetUnsentParts and "+
			"GetUnsentPartsAndSentRounds do not match."+
			"\nGetUnsentParts:              %+v"+
			"\nGetUnsentPartsAndSentRounds: %+v",
			unsentParts, unsentParts2)
	}

	sentRounds2 := sft.GetSentRounds()

	// Sort sent rounds map transfer IDs
	for _, tIDs := range sentRounds2 {
		sort.Slice(tIDs,
			func(i, j int) bool { return tIDs[i].String() < tIDs[j].String() })
	}

	if !reflect.DeepEqual(sentRounds, sentRounds2) {
		t.Errorf("Sent rounds map from GetSentRounds and "+
			"GetUnsentPartsAndSentRounds do not match."+
			"\nGetSentRounds:               %+v"+
			"\nGetUnsentPartsAndSentRounds: %+v",
			sentRounds, sentRounds2)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that the SentFileTransfersStore loaded from storage by
// LoadSentFileTransfersStore matches the original in memory.
func TestLoadSentFileTransfersStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new SentFileTransfersStore: %+v", err)
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

	// Load SentFileTransfersStore from storage
	loadedSFT, err := LoadSentFileTransfersStore(kv)
	if err != nil {
		t.Errorf("LoadSentFileTransfersStore returned an error: %+v", err)
	}

	// Equalize all progressCallbacks because reflect.DeepEqual does not seem to
	// work on function pointers
	for _, tid := range list {
		loadedSFT.transfers[tid].progressCallbacks =
			sft.transfers[tid].progressCallbacks
	}

	if !reflect.DeepEqual(sft, loadedSFT) {
		t.Errorf("Loaded SentFileTransfersStore does not match original in "+
			"memory.\nexpected: %+v\nreceived: %+v", sft, loadedSFT)
	}
}

// Error path: tests that LoadSentFileTransfersStore returns the expected error
// when the transfer list cannot be loaded from storage.
func TestLoadSentFileTransfersStore_NoListInStorageError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadSentTransfersListErr, "%")[0]

	// Load SentFileTransfersStore from storage
	_, err := LoadSentFileTransfersStore(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("LoadSentFileTransfersStore did not return the expected "+
			"error when there is no transfer list saved in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that LoadSentFileTransfersStore returns the expected error
// when the first transfer loaded from storage does not exist.
func TestLoadSentFileTransfersStore_NoTransferInStorageError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadSentTransfersErr, "%")[0]

	// Save list of one transfer ID to storage
	obj := &versioned.Object{
		Version:   sentFileTransfersStoreVersion,
		Timestamp: netTime.Now(),
		Data:      ftCrypto.UnmarshalTransferID([]byte("testID_01")).Bytes(),
	}
	err := kv.Prefix(sentFileTransfersStorePrefix).Set(
		sentFileTransfersStoreKey, sentFileTransfersStoreVersion, obj)

	// Load SentFileTransfersStore from storage
	_, err = LoadSentFileTransfersStore(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("LoadSentFileTransfersStore did not return the expected "+
			"error when there is no transfer saved in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the SentFileTransfersStore loaded from storage by
// NewOrLoadSentFileTransfersStore matches the original in memory.
func TestNewOrLoadSentFileTransfersStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	sft, err := NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new SentFileTransfersStore: %+v", err)
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

	// Load SentFileTransfersStore from storage
	loadedSFT, err := NewOrLoadSentFileTransfersStore(kv)
	if err != nil {
		t.Errorf("NewOrLoadSentFileTransfersStore returned an error: %+v", err)
	}

	// Equalize all progressCallbacks because reflect.DeepEqual does not seem
	// to work on function pointers
	for _, tid := range list {
		loadedSFT.transfers[tid].progressCallbacks =
			sft.transfers[tid].progressCallbacks
	}

	if !reflect.DeepEqual(sft, loadedSFT) {
		t.Errorf("Loaded SentFileTransfersStore does not match original in "+
			"memory.\nexpected: %+v\nreceived: %+v", sft, loadedSFT)
	}
}

// Tests that NewOrLoadSentFileTransfersStore returns a new
// SentFileTransfersStore when there is none in storage.
func TestNewOrLoadSentFileTransfersStore_NewSentFileTransfersStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	// Load SentFileTransfersStore from storage
	loadedSFT, err := NewOrLoadSentFileTransfersStore(kv)
	if err != nil {
		t.Errorf("NewOrLoadSentFileTransfersStore returned an error: %+v", err)
	}

	newSFT, _ := NewSentFileTransfersStore(kv)

	if !reflect.DeepEqual(newSFT, loadedSFT) {
		t.Errorf("Returned SentFileTransfersStore does not match new."+
			"\nexpected: %+v\nreceived: %+v", newSFT, loadedSFT)
	}
}

// Error path: tests that the NewOrLoadSentFileTransfersStore returns the
// expected error when the first transfer loaded from storage does not exist.
func TestNewOrLoadSentFileTransfersStore_NoTransferInStorageError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadSentTransfersErr, "%")[0]

	// Save list of one transfer ID to storage
	obj := &versioned.Object{
		Version:   sentFileTransfersStoreVersion,
		Timestamp: netTime.Now(),
		Data:      ftCrypto.UnmarshalTransferID([]byte("testID_01")).Bytes(),
	}
	err := kv.Prefix(sentFileTransfersStorePrefix).Set(
		sentFileTransfersStoreKey, sentFileTransfersStoreVersion, obj)

	// Load SentFileTransfersStore from storage
	_, err = NewOrLoadSentFileTransfersStore(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("NewOrLoadSentFileTransfersStore did not return the expected "+
			"error when there is no transfer saved in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that SentFileTransfersStore.saveTransfersList saves all the transfer
// IDs to storage by loading them from storage via
// SentFileTransfersStore.loadTransfersList and comparing the list to the list
// in memory.
func TestSentFileTransfersStore_saveTransfersList_loadTransfersList(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	sft := &SentFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersStorePrefix),
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

	// get list from storage
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

// Tests that the transfer loaded by SentFileTransfersStore.loadTransfers from
// storage matches the original in memory
func TestSentFileTransfersStore_loadTransfers(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	sft := &SentFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersStorePrefix),
	}

	// Add 10 transfers to map in memory
	list := make([]ftCrypto.TransferID, 10)
	for i := range list {
		tid, st := newRandomSentTransfer(16, 24, sft.kv, t)
		sft.transfers[tid] = st
		list[i] = tid
	}

	// Load the transfers into a new SentFileTransfersStore
	loadedSft := &SentFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*SentTransfer),
		kv:        kv.Prefix(sentFileTransfersStorePrefix),
	}
	err := loadedSft.loadTransfers(list)
	if err != nil {
		t.Errorf("loadTransfers returned an error: %+v", err)
	}

	// Equalize all progressCallbacks because reflect.DeepEqual does not seem
	// to work on function pointers
	for _, tid := range list {
		loadedSft.transfers[tid].progressCallbacks =
			sft.transfers[tid].progressCallbacks
	}

	if !reflect.DeepEqual(sft.transfers, loadedSft.transfers) {
		t.Errorf("Transfers loaded from storage does not match transfers in "+
			"memory.\nexpected: %+v\nreceived: %+v",
			sft.transfers, loadedSft.transfers)
	}
}

// Tests that a transfer list marshalled with
// SentFileTransfersStore.marshalTransfersList and unmarshalled with
// unmarshalTransfersList matches the original.
func TestSentFileTransfersStore_marshalTransfersList_unmarshalTransfersList(t *testing.T) {
	prng := NewPrng(42)
	sft := &SentFileTransfersStore{
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
