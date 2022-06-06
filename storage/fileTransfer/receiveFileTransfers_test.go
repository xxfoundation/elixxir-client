////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Tests that NewReceivedFileTransfersStore creates a new object with empty maps
// and that it is saved to storage
func TestNewReceivedFileTransfersStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedRFT := &ReceivedFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*ReceivedTransfer),
		info:      make(map[format.Fingerprint]*partInfo),
		kv:        kv.Prefix(receivedFileTransfersStorePrefix),
	}

	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Errorf("NewReceivedFileTransfersStore returned an error: %+v", err)
	}

	// Check that the new ReceivedFileTransfersStore matches the expected
	if !reflect.DeepEqual(expectedRFT, rft) {
		t.Errorf("New ReceivedFileTransfersStore does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedRFT, rft)
	}

	// Ensure that the transfer list is saved to storage
	_, err = expectedRFT.kv.Get(
		receivedFileTransfersStoreKey, receivedFileTransfersStoreVersion)
	if err != nil {
		t.Errorf("Failed to load transfer list from storage: %+v", err)
	}
}

// Tests that ReceivedFileTransfersStore.AddTransfer adds a new transfer and
// adds all the fingerprints to the info map.
func TestReceivedFileTransfersStore_AddTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Generate info for new transfer
	key, _ := ftCrypto.NewTransferKey(prng)
	mac := []byte("transferMAC")
	fileSize := uint32(256)
	numParts, numFps := uint16(16), uint16(24)

	// Add the transfer
	tid, err := rft.AddTransfer(key, mac, fileSize, numParts, numFps, prng)
	if err != nil {
		t.Errorf("AddTransfer returned an error: %+v", err)
	}

	// Check that the transfer was added to the map
	transfer, exists := rft.transfers[tid]
	if !exists {
		t.Errorf("Transfer with ID %s not found.", tid)
	} else {
		if transfer.GetFileSize() != fileSize {
			t.Errorf("New transfer has incorrect file size."+
				"\nexpected: %d\nreceived: %d", fileSize, transfer.GetFileSize())
		}
		if transfer.GetNumParts() != numParts {
			t.Errorf("New transfer has incorrect number of parts."+
				"\nexpected: %d\nreceived: %d", numParts, transfer.GetNumParts())
		}
		if transfer.GetTransferKey() != key {
			t.Errorf("New transfer has incorrect transfer key."+
				"\nexpected: %s\nreceived: %s", key, transfer.GetTransferKey())
		}
		if transfer.GetNumFps() != numFps {
			t.Errorf("New transfer has incorrect number of fingerprints."+
				"\nexpected: %d\nreceived: %d", numFps, transfer.GetNumFps())
		}
	}

	// Check that the transfer was added to storage
	_, err = loadReceivedTransfer(tid, rft.kv)
	if err != nil {
		t.Errorf("Transfer with ID %s not found in storage: %+v", tid, err)
	}

	// Check that all the fingerprints are in the info map
	for fpNum, fp := range ftCrypto.GenerateFingerprints(key, numFps) {
		info, exists := rft.info[fp]
		if !exists {
			t.Errorf("Part fingerprint %s (#%d) not found.", fp, fpNum)
		}

		if int(info.fpNum) != fpNum {
			t.Errorf("Fingerprint %s has incorrect fingerprint number."+
				"\nexpected: %d\nreceived: %d", fp, fpNum, info.fpNum)
		}

		if info.id != tid {
			t.Errorf("Fingerprint %s has incorrect transfer ID."+
				"\nexpected: %s\nreceived: %s", fp, tid, info.id)
		}
	}
}

// Error path: tests that ReceivedFileTransfersStore.AddTransfer returns the
// expected error when the PRNG returns an error.
func TestReceivedFileTransfersStore_AddTransfer_NewTransferIdRngError(t *testing.T) {
	prng := NewPrngErr()
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Add the transfer
	expectedErr := strings.Split(addTransferNewIdErr, "%")[0]
	_, err = rft.AddTransfer(ftCrypto.TransferKey{}, nil, 0, 0, 0, prng)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("AddTransfer did not return the expected error when the PRNG "+
			"should have errored.\nexpected: %s\nrecieved: %+v", expectedErr, err)
	}

}

// Tests that ReceivedFileTransfersStore.addFingerprints adds all the
// fingerprints to the map
func TestReceivedFileTransfersStore_addFingerprints(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	key, _ := ftCrypto.NewTransferKey(prng)
	tid, _ := ftCrypto.NewTransferID(prng)
	numFps := uint16(24)

	rft.addFingerprints(key, tid, numFps)

	// Check that all the fingerprints are in the info map
	for fpNum, fp := range ftCrypto.GenerateFingerprints(key, numFps) {
		info, exists := rft.info[fp]
		if !exists {
			t.Errorf("Part fingerprint %s (#%d) not found.", fp, fpNum)
		}

		if int(info.fpNum) != fpNum {
			t.Errorf("Fingerprint %s has incorrect fingerprint number."+
				"\nexpected: %d\nreceived: %d", fp, fpNum, info.fpNum)
		}

		if info.id != tid {
			t.Errorf("Fingerprint %s has incorrect transfer ID."+
				"\nexpected: %s\nreceived: %s", fp, tid, info.id)
		}
	}
}

// Tests that ReceivedFileTransfersStore.GetTransfer returns the newly added
// transfer.
func TestReceivedFileTransfersStore_GetTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Generate random info for new transfer
	key, _ := ftCrypto.NewTransferKey(prng)
	mac := []byte("transferMAC")
	fileSize := uint32(256)
	numParts, numFps := uint16(16), uint16(24)

	// Add the transfer
	tid, err := rft.AddTransfer(key, mac, fileSize, numParts, numFps, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Get the transfer
	transfer, err := rft.GetTransfer(tid)
	if err != nil {
		t.Errorf("GetTransfer returned an error: %+v", err)
	}

	if transfer.GetFileSize() != fileSize {
		t.Errorf("New transfer has incorrect file size."+
			"\nexpected: %d\nreceived: %d", fileSize, transfer.GetFileSize())
	}

	if transfer.GetNumParts() != numParts {
		t.Errorf("New transfer has incorrect number of parts."+
			"\nexpected: %d\nreceived: %d", numParts, transfer.GetNumParts())
	}

	if transfer.GetNumFps() != numFps {
		t.Errorf("New transfer has incorrect number of fingerprints."+
			"\nexpected: %d\nreceived: %d", numFps, transfer.GetNumFps())
	}

	if transfer.GetTransferKey() != key {
		t.Errorf("New transfer has incorrect transfer key."+
			"\nexpected: %s\nreceived: %s", key, transfer.GetTransferKey())
	}
}

// Error path: tests that ReceivedFileTransfersStore.GetTransfer returns the
// expected error when the provided transfer ID does not correlate to any saved
// transfer.
func TestReceivedFileTransfersStore_GetTransfer_NoTransferError(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Generate random info for new transfer
	key, _ := ftCrypto.NewTransferKey(prng)
	mac := []byte("transferMAC")

	// Add the transfer
	_, err = rft.AddTransfer(key, mac, 256, 16, 24, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Get the transfer
	invalidTid, _ := ftCrypto.NewTransferID(prng)
	expectedErr := fmt.Sprintf(getReceivedTransferErr, invalidTid)
	_, err = rft.GetTransfer(invalidTid)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("GetTransfer did not return the expected error when no "+
			"transfer for the ID exists.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests that DeleteTransfer removed a transfer from memory and storage.
func TestReceivedFileTransfersStore_DeleteTransfer(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Add the transfer
	key, _ := ftCrypto.NewTransferKey(prng)
	mac := []byte("transferMAC")
	numFps := uint16(24)
	tid, err := rft.AddTransfer(key, mac, 256, 16, numFps, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Delete the transfer
	err = rft.DeleteTransfer(tid)
	if err != nil {
		t.Errorf("DeleteTransfer returned an error: %+v", err)
	}

	// Check that the transfer was deleted from the map
	_, exists := rft.transfers[tid]
	if exists {
		t.Errorf("Transfer with ID %s found in map when it should have been "+
			"deleted.", tid)
	}

	// Check that the transfer was deleted from storage
	_, err = loadReceivedTransfer(tid, rft.kv)
	if err == nil {
		t.Errorf("Transfer with ID %s found in storage when it should have "+
			"been deleted.", tid)
	}

	// Check that all the fingerprints in the info map were deleted
	for fpNum, fp := range ftCrypto.GenerateFingerprints(key, numFps) {
		_, exists = rft.info[fp]
		if exists {
			t.Errorf("Part fingerprint %s (#%d) found in map when it should "+
				"have been deleted.", fp, fpNum)
		}
	}
}

// Error path: tests that ReceivedFileTransfersStore.DeleteTransfer returns the
// expected error when the provided transfer ID does not correlate to any saved
// transfer.
func TestReceivedFileTransfersStore_DeleteTransfer_NoTransferError(t *testing.T) {
	prng := NewPrng(42)
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Delete the transfer
	invalidTid, _ := ftCrypto.NewTransferID(prng)
	expectedErr := fmt.Sprintf(getReceivedTransferErr, invalidTid)
	err = rft.DeleteTransfer(invalidTid)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("DeleteTransfer did not return the expected error when no "+
			"transfer for the ID exists.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests that ReceivedFileTransfersStore.AddPart modifies the expected transfer
// in memory.
func TestReceivedFileTransfersStore_AddPart(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	prng := NewPrng(42)
	key, _ := ftCrypto.NewTransferKey(prng)
	mac := []byte("transferMAC")

	tid, err := rft.AddTransfer(key, mac, 256, 16, 24, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Create encrypted part

	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	expectedData := []byte("test")

	partNum, fpNum := uint16(1), uint16(1)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(expectedData)

	fp := ftCrypto.GenerateFingerprint(key, fpNum)
	encryptedPart, mac, err := ftCrypto.EncryptPart(key, partData.Marshal(), fpNum, fp)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(encryptedPart)
	cmixMsg.SetMac(mac)

	// Add encrypted part
	rt, _, _, err := rft.AddPart(cmixMsg)
	if err != nil {
		t.Errorf("AddPart returned an error: %+v", err)
	}

	// Make sure its fingerprint was removed from the map
	_, exists := rft.info[fp]
	if exists {
		t.Errorf("Fingerprints %s for added part found when it should have "+
			"been deleted.", fp)
	}

	// Check that the transfer is correct
	expectedRT := rft.transfers[tid]
	if !reflect.DeepEqual(expectedRT, rt) {
		t.Errorf("Returned transfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedRT, rt)
	}

	// Check that the correct part was stored
	receivedPart := expectedRT.receivedParts.parts[partNum]
	if !bytes.Equal(receivedPart[:len(expectedData)], expectedData) {
		t.Errorf("Part in memory is not expected."+
			"\nexpected: %q\nreceived: %q", expectedData, receivedPart[:len(expectedData)])
	}
}

// Error path: tests that ReceivedFileTransfersStore.AddPart returns the
// expected error when the provided fingerprint does not correlate to any part.
func TestReceivedFileTransfersStore_AddPart_NoFingerprintError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Create encrypted part
	fp := format.NewFingerprint([]byte("invalidTransferKey"))

	msg := format.NewMessage(1000)
	msg.SetKeyFP(fp)

	// Add encrypted part
	expectedErr := fmt.Sprintf(noFingerprintErr, fp)
	_, _, _, err = rft.AddPart(msg)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("AddPart did not return the expected error when no part for "+
			"the fingerprint exists.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Error path: tests that ReceivedFileTransfersStore.AddPart returns the
// expected error when the provided transfer ID does not correlate to any saved
// transfer.
func TestReceivedFileTransfersStore_AddPart_NoTransferError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	prng := NewPrng(42)
	key, _ := ftCrypto.NewTransferKey(prng)
	mac := []byte("transferMAC")

	_, err = rft.AddTransfer(key, mac, 256, 16, 24, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Create encrypted part
	fp := ftCrypto.GenerateFingerprint(key, 1)
	invalidTid, _ := ftCrypto.NewTransferID(prng)
	rft.info[fp].id = invalidTid

	msg := format.NewMessage(1000)
	msg.SetKeyFP(fp)

	// Add encrypted part
	expectedErr := fmt.Sprintf(getReceivedTransferErr, invalidTid)
	_, _, _, err = rft.AddPart(msg)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("AddPart did not return the expected error when no transfer "+
			"for the ID exists.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that ReceivedFileTransfersStore.AddPart returns the
// expected error when the encrypted part data, MAC, and padding are invalid.
func TestReceivedFileTransfersStore_AddPart_AddPartError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	prng := NewPrng(42)
	key, _ := ftCrypto.NewTransferKey(prng)
	mac := []byte("transferMAC")
	numParts := uint16(16)

	tid, err := rft.AddTransfer(key, mac, 256, numParts, 24, prng)
	if err != nil {
		t.Errorf("Failed to add new transfer: %+v", err)
	}

	// Create encrypted part
	partNum, fpNum := uint16(1), uint16(1)
	part := []byte("invalidPart")
	mac = make([]byte, format.MacLen)
	fp := ftCrypto.GenerateFingerprint(key, fpNum)

	// Add encrypted part
	expectedErr := fmt.Sprintf(addPartErr, tid, "")

	cmixMsg := format.NewMessage(format.MinimumPrimeSize)

	partData, _ := NewPartMessage(cmixMsg.ContentsSize())
	partData.SetPartNum(partNum)
	_ = partData.SetPart(part)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetContents(partData.Marshal())
	cmixMsg.SetMac(mac)

	_, _, _, err = rft.AddPart(cmixMsg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("AddPart did not return the expected error when the "+
			"encrypted part, padding, and MAC are invalid."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that the ReceivedFileTransfersStore loaded from storage by
// LoadReceivedFileTransfersStore matches the original in memory.
func TestLoadReceivedFileTransfersStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new ReceivedFileTransfersStore: %+v", err)
	}

	// Add 10 transfers to map in memory
	list := make([]ftCrypto.TransferID, 10)
	for i := range list {
		tid, rt, _ := newRandomReceivedTransfer(16, 24, rft.kv, t)

		// Add to transfer
		rft.transfers[tid] = rt

		// Add unused fingerprints
		for n := uint32(0); n < rt.fpVector.GetNumKeys(); n++ {
			if !rt.fpVector.Used(n) {
				fp := ftCrypto.GenerateFingerprint(rt.key, uint16(n))
				rft.info[fp] = &partInfo{tid, uint16(n)}
			}
		}

		// Add ID to list
		list[i] = tid
	}

	// Save list to storage
	if err = rft.saveTransfersList(); err != nil {
		t.Errorf("Faileds to save transfers list: %+v", err)
	}

	// Load ReceivedFileTransfersStore from storage
	loadedRFT, err := LoadReceivedFileTransfersStore(kv)
	if err != nil {
		t.Errorf("LoadReceivedFileTransfersStore returned an error: %+v", err)
	}

	if !reflect.DeepEqual(rft, loadedRFT) {
		t.Errorf("Loaded ReceivedFileTransfersStore does not match original"+
			"in  memory.\nexpected: %+v\nreceived: %+v", rft, loadedRFT)
	}
}

// Error path: tests that ReceivedFileTransfersStore returns the expected error
// when the transfer list cannot be loaded from storage.
func TestLoadReceivedFileTransfersStore_NoListInStorageError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadReceivedTransfersListErr, "%")[0]

	// Load ReceivedFileTransfersStore from storage
	_, err := LoadReceivedFileTransfersStore(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("LoadReceivedFileTransfersStore did not return the expected "+
			"error  when there is no transfer list saved in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that ReceivedFileTransfersStore returns the expected error
// when the first transfer loaded from storage does not exist.
func TestLoadReceivedFileTransfersStore_NoTransferInStorageError(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedErr := strings.Split(loadReceivedFileTransfersErr, "%")[0]

	// Save list of one transfer ID to storage
	obj := &versioned.Object{
		Version:   receivedFileTransfersStoreVersion,
		Timestamp: netTime.Now(),
		Data:      ftCrypto.UnmarshalTransferID([]byte("testID_01")).Bytes(),
	}
	err := kv.Prefix(receivedFileTransfersStorePrefix).Set(
		receivedFileTransfersStoreKey, receivedFileTransfersStoreVersion, obj)

	// Load ReceivedFileTransfersStore from storage
	_, err = LoadReceivedFileTransfersStore(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("LoadReceivedFileTransfersStore did not return the expected "+
			"error when there is no transfer saved in storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the ReceivedFileTransfersStore loaded from storage by
// NewOrLoadReceivedFileTransfersStore matches the original in memory.
func TestNewOrLoadReceivedFileTransfersStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new ReceivedFileTransfersStore: %+v", err)
	}

	// Add 10 transfers to map in memory
	list := make([]ftCrypto.TransferID, 10)
	for i := range list {
		tid, rt, _ := newRandomReceivedTransfer(16, 24, rft.kv, t)

		// Add to transfer
		rft.transfers[tid] = rt

		// Add unused fingerprints
		for n := uint32(0); n < rt.fpVector.GetNumKeys(); n++ {
			if !rt.fpVector.Used(n) {
				fp := ftCrypto.GenerateFingerprint(rt.key, uint16(n))
				rft.info[fp] = &partInfo{tid, uint16(n)}
			}
		}

		// Add ID to list
		list[i] = tid
	}

	// Save list to storage
	if err = rft.saveTransfersList(); err != nil {
		t.Errorf("Faileds to save transfers list: %+v", err)
	}

	// Load ReceivedFileTransfersStore from storage
	loadedRFT, err := NewOrLoadReceivedFileTransfersStore(kv)
	if err != nil {
		t.Errorf("NewOrLoadReceivedFileTransfersStore returned an error: %+v",
			err)
	}

	if !reflect.DeepEqual(rft, loadedRFT) {
		t.Errorf("Loaded ReceivedFileTransfersStore does not match original "+
			"in memory.\nexpected: %+v\nreceived: %+v", rft, loadedRFT)
	}
}

// Tests that NewOrLoadReceivedFileTransfersStore returns a new
// ReceivedFileTransfersStore when there is none in storage.
func TestNewOrLoadReceivedFileTransfersStore_NewReceivedFileTransfersStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	// Load ReceivedFileTransfersStore from storage
	loadedRFT, err := NewOrLoadReceivedFileTransfersStore(kv)
	if err != nil {
		t.Errorf("NewOrLoadReceivedFileTransfersStore returned an error: %+v",
			err)
	}

	newRFT, _ := NewReceivedFileTransfersStore(kv)

	if !reflect.DeepEqual(newRFT, loadedRFT) {
		t.Errorf("Returned ReceivedFileTransfersStore does not match new."+
			"\nexpected: %+v\nreceived: %+v", newRFT, loadedRFT)
	}
}

// Tests that the list saved by ReceivedFileTransfersStore.saveTransfersList
// matches the list loaded by ReceivedFileTransfersStore.load.
func TestReceivedFileTransfersStore_saveTransfersList_loadTransfersList(t *testing.T) {
	rft, err := NewReceivedFileTransfersStore(versioned.NewKV(make(ekv.Memstore)))
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Fill map with transfers
	expectedList := make([]ftCrypto.TransferID, 7)
	for i := range expectedList {
		prng := NewPrng(int64(i))
		key, _ := ftCrypto.NewTransferKey(prng)
		mac := []byte("transferMAC")
		numParts := uint16(i)
		numFps := uint16(float32(i) * 2.5)

		expectedList[i], err = rft.AddTransfer(
			key, mac, 256, numParts, numFps, prng)
		if err != nil {
			t.Errorf("Failed to add new transfer #%d: %+v", i, err)
		}
	}

	// Save the list
	err = rft.saveTransfersList()
	if err != nil {
		t.Errorf("saveTransfersList returned an error: %+v", err)
	}

	// Load the list
	loadedList, err := rft.loadTransfersList()
	if err != nil {
		t.Errorf("loadTransfersList returned an error: %+v", err)
	}

	// Sort slices so they can be compared
	sort.SliceStable(expectedList, func(i, j int) bool {
		return bytes.Compare(expectedList[i].Bytes(), expectedList[j].Bytes()) == -1
	})
	sort.SliceStable(loadedList, func(i, j int) bool {
		return bytes.Compare(loadedList[i].Bytes(), loadedList[j].Bytes()) == -1
	})

	if !reflect.DeepEqual(expectedList, loadedList) {
		t.Errorf("Loaded transfer list does not match expected."+
			"\nexpected: %v\nreceived: %v", expectedList, loadedList)
	}
}

// Tests that the list loaded by ReceivedFileTransfersStore.load matches the
// original saved to storage.
func TestReceivedFileTransfersStore_load(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Fill map with transfers
	idList := make([]ftCrypto.TransferID, 7)
	for i := range idList {
		prng := NewPrng(int64(i))
		key, _ := ftCrypto.NewTransferKey(prng)
		mac := []byte("transferMAC")

		idList[i], err = rft.AddTransfer(key, mac, 256, 16, 24, prng)
		if err != nil {
			t.Errorf("Failed to add new transfer #%d: %+v", i, err)
		}
	}

	// Save the list
	err = rft.saveTransfersList()
	if err != nil {
		t.Errorf("saveTransfersList returned an error: %+v", err)
	}

	// Build new ReceivedFileTransfersStore
	newRFT := &ReceivedFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*ReceivedTransfer),
		info:      make(map[format.Fingerprint]*partInfo),
		kv:        kv.Prefix(receivedFileTransfersStorePrefix),
	}

	// Load saved transfers from storage
	err = newRFT.load(idList)
	if err != nil {
		t.Errorf("load returned an error: %+v", err)
	}

	// Check that all transfer were loaded from storage
	for _, id := range idList {
		transfer, exists := newRFT.transfers[id]
		if !exists {
			t.Errorf("Transfer %s not loaded from storage.", id)
		}

		// Check that the loaded transfer matches the original in memory
		if !reflect.DeepEqual(transfer, rft.transfers[id]) {
			t.Errorf("Loaded transfer does not match original."+
				"\noriginal: %+v\nreceived: %+v", rft.transfers[id], transfer)
		}

		// Make sure all fingerprints are present
		for fpNum, fp := range ftCrypto.GenerateFingerprints(transfer.key, 24) {
			info, exists := newRFT.info[fp]
			if !exists {
				t.Errorf("Fingerprint %d for transfer %s does not exist in "+
					"the map.", fpNum, id)
			}

			if info.id != id {
				t.Errorf("Fingerprint has wrong transfer ID."+
					"\nexpected: %s\nreceived: %s", id, info.id)
			}
			if int(info.fpNum) != fpNum {
				t.Errorf("Fingerprint has wrong number."+
					"\nexpected: %d\nreceived: %d", fpNum, info.fpNum)
			}
		}
	}
}

// Error path: tests that ReceivedFileTransfersStore.load returns an error when
// all file transfers fail to load from storage.
func TestReceivedFileTransfersStore_load_AllFail(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	rft, err := NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to create new ReceivedFileTransfersStore: %+v", err)
	}

	// Fill map with transfers
	idList := make([]ftCrypto.TransferID, 7)
	for i := range idList {
		prng := NewPrng(int64(i))
		key, _ := ftCrypto.NewTransferKey(prng)
		mac := []byte("transferMAC")

		idList[i], err = rft.AddTransfer(key, mac, 256, 16, 24, prng)
		if err != nil {
			t.Errorf("Failed to add new transfer #%d: %+v", i, err)
		}

		err = rft.DeleteTransfer(idList[i])
		if err != nil {
			t.Errorf("Failed to delete transfer: %+v", err)
		}
	}

	// Save the list
	err = rft.saveTransfersList()
	if err != nil {
		t.Errorf("saveTransfersList returned an error: %+v", err)
	}

	// Build new ReceivedFileTransfersStore
	newRFT := &ReceivedFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*ReceivedTransfer),
		info:      make(map[format.Fingerprint]*partInfo),
		kv:        kv.Prefix(receivedFileTransfersStorePrefix),
	}

	expectedErr := fmt.Sprintf(loadReceivedTransfersAllErr, len(idList))

	// Load saved transfers from storage
	err = newRFT.load(idList)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("load did not return the expected error when none of the "+
			"transfer could be loaded from storage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that a transfer list marshalled with
// ReceivedFileTransfersStore.marshalTransfersList and unmarshalled with
// unmarshalTransfersList matches the original.
func TestReceivedFileTransfersStore_marshalTransfersList_unmarshalTransfersList(t *testing.T) {
	prng := NewPrng(42)
	rft := &ReceivedFileTransfersStore{
		transfers: make(map[ftCrypto.TransferID]*ReceivedTransfer),
	}

	// Add 10 transfers to map in memory
	for i := 0; i < 10; i++ {
		tid, _ := ftCrypto.NewTransferID(prng)
		rft.transfers[tid] = &ReceivedTransfer{}
	}

	// Marshal into byte slice
	marshalledBytes := rft.marshalTransfersList()

	// Unmarshal marshalled bytes into transfer ID list
	list := unmarshalTransfersList(marshalledBytes)

	// Check that the list has all the transfer IDs in memory
	for _, tid := range list {
		if _, exists := rft.transfers[tid]; !exists {
			t.Errorf("No transfer for ID %s exists.", tid)
		} else {
			delete(rft.transfers, tid)
		}
	}
}
