////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/client/fileTransfer2/store/cypher"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"reflect"
	"testing"
)

// Tests that newReceivedTransfer returns a new ReceivedTransfer with the
// expected values.
func Test_newReceivedTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
	numFps := uint16(24)
	parts, _ := generateTestParts(16)
	fileSize := uint32(len(parts) * len(parts[0]))
	numParts := uint16(len(parts))
	rtKv := kv.Prefix(makeReceivedTransferPrefix(&tid))

	cypherManager, err := cypher.NewManager(&key, numFps, rtKv)
	if err != nil {
		t.Errorf("Failed to make new cypher manager: %+v", err)
	}
	partStatus, err := utility.NewStateVector(
		rtKv, receivedTransferStatusKey, uint32(numParts))
	if err != nil {
		t.Errorf("Failed to make new state vector: %+v", err)
	}

	expected := &ReceivedTransfer{
		cypherManager: cypherManager,
		tid:           &tid,
		fileName:      "fileName",
		transferMAC:   []byte("transferMAC"),
		fileSize:      fileSize,
		numParts:      numParts,
		parts:         make([][]byte, numParts),
		partStatus:    partStatus,
		kv:            rtKv,
	}

	rt, err := newReceivedTransfer(&key, &tid, expected.fileName,
		expected.transferMAC, fileSize, numParts, numFps, kv)
	if err != nil {
		t.Errorf("newReceivedTransfer returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, rt) {
		t.Errorf("New ReceivedTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, rt)
	}
}

// Tests that ReceivedTransfer.AddPart adds the part to the part list and marks
// it as received
func TestReceivedTransfer_AddPart(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	part := []byte("Part")
	partNum := 6

	err := rt.AddPart(part, partNum)
	if err != nil {
		t.Errorf("Failed to add part: %+v", err)
	}

	if !bytes.Equal(rt.parts[partNum], part) {
		t.Errorf("Found incorrect part in list.\nexpected: %q\nreceived: %q",
			part, rt.parts[partNum])
	}

	if !rt.partStatus.Used(uint32(partNum)) {
		t.Errorf("Part #%d not marked as received.", partNum)
	}
}

// Tests that ReceivedTransfer.AddPart returns an error if the part number is
// not within the range of part numbers
func TestReceivedTransfer_AddPart_PartOutOfRangeError(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	expectedErr := fmt.Sprintf(errPartOutOfRange, rt.partStatus.GetNumKeys(),
		rt.partStatus.GetNumKeys()-1)

	err := rt.AddPart([]byte("Part"), int(rt.partStatus.GetNumKeys()))
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Failed to get expected error when part number is out of range."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that ReceivedTransfer.GetFile returns the expected file after all the
// parts are added to the transfer.
func TestReceivedTransfer_GetFile(t *testing.T) {
	// Generate parts and make last file part smaller than the rest
	parts, _ := generateTestParts(16)
	lastPartLen := 6
	rt, _, _, _, _ := newTestReceivedTransfer(uint16(len(parts)), t)
	rt.fileSize = uint32((len(parts)-1)*len(parts[0]) + lastPartLen)

	for i, p := range parts {
		err := rt.AddPart(p, i)
		if err != nil {
			t.Errorf("Failed to add part #%d: %+v", i, err)
		}
	}

	parts[len(parts)-1] = parts[len(parts)-1][:lastPartLen]
	combinedParts := bytes.Join(parts, nil)

	file := rt.GetFile()

	if !bytes.Equal(file, combinedParts) {
		t.Errorf("Received file does not match expected."+
			"\nexpected: %q\nreceived: %q", combinedParts, file)
	}

}

// Tests that ReceivedTransfer.GetUnusedCyphers returns the correct number of
// unused cyphers.
func TestReceivedTransfer_GetUnusedCyphers(t *testing.T) {
	numParts := uint16(10)
	rt, _, _, numFps, _ := newTestReceivedTransfer(numParts, t)

	// Check that all cyphers are returned after initialisation
	unsentCyphers := rt.GetUnusedCyphers()
	if len(unsentCyphers) != int(numFps) {
		t.Errorf("Number of unused cyphers does not match original number of "+
			"fingerprints when none have been used.\nexpected: %d\nreceived: %d",
			numFps, len(unsentCyphers))
	}

	// Use every other part
	for i := range unsentCyphers {
		if i%2 == 0 {
			_, _ = unsentCyphers[i].PopCypher()
		}
	}

	// Check that only have the number of parts is returned
	unsentCyphers = rt.GetUnusedCyphers()
	if len(unsentCyphers) != int(numFps)/2 {
		t.Errorf("Number of unused cyphers is not half original number after "+
			"half have been marked as received.\nexpected: %d\nreceived: %d",
			numFps/2, len(unsentCyphers))
	}

	// Use the rest of the parts
	for i := range unsentCyphers {
		_, _ = unsentCyphers[i].PopCypher()
	}

	// Check that no sent parts are returned
	unsentCyphers = rt.GetUnusedCyphers()
	if len(unsentCyphers) != 0 {
		t.Errorf("Number of unused cyphers is not zero after all have been "+
			"marked as received.\nexpected: %d\nreceived: %d",
			0, len(unsentCyphers))
	}
}

// Tests that ReceivedTransfer.TransferID returns the correct transfer ID.
func TestReceivedTransfer_TransferID(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	if rt.TransferID() != rt.tid {
		t.Errorf("Incorrect transfer ID.\nexpected: %s\nreceived: %s",
			rt.tid, rt.TransferID())
	}
}

// Tests that ReceivedTransfer.FileName returns the correct file name.
func TestReceivedTransfer_FileName(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	if rt.FileName() != rt.fileName {
		t.Errorf("Incorrect transfer ID.\nexpected: %s\nreceived: %s",
			rt.fileName, rt.FileName())
	}
}

// Tests that ReceivedTransfer.FileSize returns the correct file size.
func TestReceivedTransfer_FileSize(t *testing.T) {
	rt, file, _, _, _ := newTestReceivedTransfer(16, t)
	fileSize := uint32(len(file))

	if rt.FileSize() != fileSize {
		t.Errorf("Incorrect file size.\nexpected: %d\nreceived: %d",
			fileSize, rt.FileSize())
	}
}

// Tests that ReceivedTransfer.NumParts returns the correct number of parts.
func TestReceivedTransfer_NumParts(t *testing.T) {
	numParts := uint16(16)
	rt, _, _, _, _ := newTestReceivedTransfer(numParts, t)

	if rt.NumParts() != numParts {
		t.Errorf("Incorrect number of parts.\nexpected: %d\nreceived: %d",
			numParts, rt.NumParts())
	}
}

// Tests that ReceivedTransfer.NumReceived returns the correct number of
// received parts.
func TestReceivedTransfer_NumReceived(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	if rt.NumReceived() != 0 {
		t.Errorf("Incorrect number of received parts."+
			"\nexpected: %d\nreceived: %d", 0, rt.NumReceived())
	}

	// Add all parts as received
	for i := 0; i < int(rt.numParts); i++ {
		_ = rt.AddPart(nil, i)
	}

	if uint32(rt.NumReceived()) != rt.partStatus.GetNumKeys() {
		t.Errorf("Incorrect number of received parts."+
			"\nexpected: %d\nreceived: %d",
			uint32(rt.NumReceived()), rt.partStatus.GetNumKeys())
	}
}

// Tests that the state vector returned by ReceivedTransfer.CopyPartStatusVector
// has the same values as the original but is a copy.
func TestReceivedTransfer_CopyPartStatusVector(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(64, t)

	// Check that the vectors have the same unused parts
	partStatus := rt.CopyPartStatusVector()
	if !reflect.DeepEqual(
		partStatus.GetUnusedKeyNums(), rt.partStatus.GetUnusedKeyNums()) {
		t.Errorf("Copied part status does not match original."+
			"\nexpected: %v\nreceived: %v",
			rt.partStatus.GetUnusedKeyNums(), partStatus.GetUnusedKeyNums())
	}

	// Modify the state
	_ = rt.AddPart([]byte("hello"), 5)

	// Check that the copied state is different
	if reflect.DeepEqual(
		partStatus.GetUnusedKeyNums(), rt.partStatus.GetUnusedKeyNums()) {
		t.Errorf("Old copied part status matches new status."+
			"\nexpected: %v\nreceived: %v",
			rt.partStatus.GetUnusedKeyNums(), partStatus.GetUnusedKeyNums())
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that a ReceivedTransfer loaded via loadReceivedTransfer matches the
// original.
func Test_loadReceivedTransfer(t *testing.T) {
	parts, _ := generateTestParts(16)
	rt, _, _, _, kv := newTestReceivedTransfer(uint16(len(parts)), t)

	for i, p := range parts {
		if i%2 == 0 {

			err := rt.AddPart(p, i)
			if err != nil {
				t.Errorf("Failed to add part #%d: %+v", i, err)
			}
		}
	}

	loadedRt, err := loadReceivedTransfer(rt.tid, kv)
	if err != nil {
		t.Errorf("Failed to load ReceivedTransfer: %+v", err)
	}

	if !reflect.DeepEqual(rt, loadedRt) {
		t.Errorf("Loaded ReceivedTransfer does not match original."+
			"\nexpected: %+v\nreceived: %+v", rt, loadedRt)
	}
}

// Tests that ReceivedTransfer.Delete deletes the storage backend of the
// ReceivedTransfer and that it cannot be loaded again.
func TestReceivedTransfer_Delete(t *testing.T) {
	rt, _, _, _, kv := newTestReceivedTransfer(64, t)

	err := rt.Delete()
	if err != nil {
		t.Errorf("Delete returned an error: %+v", err)
	}

	_, err = loadSentTransfer(rt.tid, kv)
	if err == nil {
		t.Errorf("Loaded received transfer that was deleted.")
	}
}

// Tests that the fields saved by ReceivedTransfer.save can be loaded from
// storage.
func TestReceivedTransfer_save(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(64, t)

	err := rt.save()
	if err != nil {
		t.Errorf("save returned an error: %+v", err)
	}

	_, err = rt.kv.Get(receivedTransferStoreKey, receivedTransferStoreVersion)
	if err != nil {
		t.Errorf("Failed to load saved ReceivedTransfer: %+v", err)
	}
}

// newTestReceivedTransfer creates a new ReceivedTransfer for testing.
func newTestReceivedTransfer(numParts uint16, t *testing.T) (
	rt *ReceivedTransfer, file []byte, key *ftCrypto.TransferKey,
	numFps uint16, kv *versioned.KV) {
	kv = versioned.NewKV(ekv.MakeMemstore())
	keyTmp, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
	transferMAC := []byte("I am a transfer MAC")
	numFps = 2 * numParts
	fileName := "helloFile"
	_, file = generateTestParts(numParts)
	fileSize := uint32(len(file))

	st, err := newReceivedTransfer(
		&keyTmp, &tid, fileName, transferMAC, fileSize, numParts, numFps, kv)
	if err != nil {
		t.Errorf("Failed to make new SentTransfer: %+v", err)
	}

	return st, file, &keyTmp, numFps, kv
}

// Tests that a ReceivedTransfer marshalled via ReceivedTransfer.marshal and
// unmarshalled via unmarshalReceivedTransfer matches the original.
func TestReceivedTransfer_marshal_unmarshalReceivedTransfer(t *testing.T) {
	rt := &ReceivedTransfer{
		fileName:    "transferName",
		transferMAC: []byte("I am a transfer MAC"),
		fileSize:    735,
		numParts:    153,
	}

	data, err := rt.marshal()
	if err != nil {
		t.Errorf("marshal returned an error: %+v", err)
	}

	fileName, transferMac, numParts, fileSize, err :=
		unmarshalReceivedTransfer(data)
	if err != nil {
		t.Errorf("Failed to unmarshal SentTransfer: %+v", err)
	}

	if rt.fileName != fileName {
		t.Errorf("Incorrect file name.\nexpected: %q\nreceived: %q",
			rt.fileName, fileName)
	}

	if !bytes.Equal(rt.transferMAC, transferMac) {
		t.Errorf("Incorrect transfer MAC.\nexpected: %s\nreceived: %s",
			rt.transferMAC, transferMac)
	}

	if rt.numParts != numParts {
		t.Errorf("Incorrect number of parts.\nexpected: %d\nreceived: %d",
			rt.numParts, numParts)
	}

	if rt.fileSize != fileSize {
		t.Errorf("Incorrect file size.\nexpected: %d\nreceived: %d",
			rt.fileSize, fileSize)
	}
}

// Tests that the part saved to storage via savePart can be loaded.
func Test_savePart_loadPart(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	part := []byte("I am a part.")
	partNum := 18

	err := savePart(part, partNum, kv)
	if err != nil {
		t.Errorf("Failed to save part: %+v", err)
	}

	loadedPart, err := loadPart(partNum, kv)
	if err != nil {
		t.Errorf("Failed to load part: %+v", err)
	}

	if !bytes.Equal(part, loadedPart) {
		t.Errorf("Loaded part does not match original."+
			"\nexpected: %q\nreceived: %q", part, loadedPart)
	}
}

// Consistency test of makeReceivedTransferPrefix.
func Test_makeReceivedTransferPrefix_Consistency(t *testing.T) {
	expectedPrefixes := []string{
		"ReceivedFileTransferStore/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/AwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/BAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/BQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/BgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/BwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"ReceivedFileTransferStore/CQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}

	for i, expected := range expectedPrefixes {
		tid := ftCrypto.TransferID{byte(i)}
		prefix := makeReceivedTransferPrefix(&tid)

		if expected != prefix {
			t.Errorf("Prefix #%d does not match expected."+
				"\nexpected: %q\nreceived: %q", i, expected, prefix)
		}
	}
}

// Consistency test of makeReceivedPartKey.
func Test_makeReceivedPartKey_Consistency(t *testing.T) {
	expectedKeys := []string{
		"receivedPart#0", "receivedPart#1", "receivedPart#2", "receivedPart#3",
		"receivedPart#4", "receivedPart#5", "receivedPart#6", "receivedPart#7",
		"receivedPart#8", "receivedPart#9",
	}

	for i, expected := range expectedKeys {
		key := makeReceivedPartKey(i)

		if expected != key {
			t.Errorf("Key #%d does not match expected."+
				"\nexpected: %q\nreceived: %q", i, expected, key)
		}
	}
}
