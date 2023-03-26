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
	"gitlab.com/elixxir/client/v4/fileTransfer/store/cypher"
	"gitlab.com/elixxir/client/v4/fileTransfer/store/fileMessage"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"strconv"
	"testing"
)

// Tests that newSentTransfer returns a new SentTransfer with the expected
// values.
func Test_newSentTransfer(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
	numFps := uint16(24)
	parts := [][]byte{[]byte("hello"), []byte("hello"), []byte("hello")}

	cypherManager, err := cypher.NewManager(&key, numFps, kv, makeSentTransferPrefix(&tid))
	if err != nil {
		t.Errorf("Failed to make new cypher manager: %+v", err)
	}
	partStatus, err := utility.NewStateVector(
		kv, sentTransferStatusKey, uint32(len(parts)))
	if err != nil {
		t.Errorf("Failed to make new state vector: %+v", err)
	}

	expected := &SentTransfer{
		cypherManager: cypherManager,
		tid:           &tid,
		fileName:      "file",
		recipient:     id.NewIdFromString("user", id.User, t),
		fileSize:      calcFileSize(parts),
		numParts:      uint16(len(parts)),
		status:        Running,
		parts:         parts,
		partStatus:    partStatus,
		kv:            kv,
	}

	st, err := newSentTransfer(expected.recipient, &key, &tid,
		expected.fileName, expected.fileSize, parts, numFps, kv)
	if err != nil {
		t.Errorf("newSentTransfer returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, st) {
		t.Errorf("New SentTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, st)
	}
}

// Tests that SentTransfer.GetUnsentParts returns the correct number of unsent
// parts
func TestSentTransfer_GetUnsentParts(t *testing.T) {
	numParts := uint16(10)
	st, _, _, _, _ := newTestSentTransfer(numParts, t)

	// Check that all parts are returned after initialisation
	unsentParts := st.GetUnsentParts()
	if len(unsentParts) != int(numParts) {
		t.Errorf("Number of unsent parts does not match original number of "+
			"parts when none have been sent.\nexpected: %d\nreceived: %d",
			numParts, len(unsentParts))
	}

	// Ensure all parts have the proper part number
	for i, p := range unsentParts {
		if int(p.partNum) != i {
			t.Errorf("Part has incorrect part number."+
				"\nexpected: %d\nreceived: %d", i, p.partNum)
		}
	}

	// Use every other part
	for i := range unsentParts {
		if i%2 == 0 {
			unsentParts[i].MarkArrived()
		}
	}

	// Check that only have the number of parts is returned
	unsentParts = st.GetUnsentParts()
	if len(unsentParts) != int(numParts)/2 {
		t.Errorf("Number of unsent parts is not half original number after "+
			"half have been marked as arrived.\nexpected: %d\nreceived: %d",
			numParts/2, len(unsentParts))
	}

	// Ensure all parts have the proper part number
	for i, p := range unsentParts {
		if int(p.partNum) != i*2+1 {
			t.Errorf("Part has incorrect part number."+
				"\nexpected: %d\nreceived: %d", i*2+1, p.partNum)
		}
	}

	// Use the rest of the parts
	for i := range unsentParts {
		unsentParts[i].MarkArrived()
	}

	// Check that no sent parts are returned
	unsentParts = st.GetUnsentParts()
	if len(unsentParts) != 0 {
		t.Errorf("Number of unsent parts is not zero after all have been "+
			"marked as arrived.\nexpected: %d\nreceived: %d",
			0, len(unsentParts))
	}
}

// Tests that SentTransfer.getPartData returns all the correct parts at their
// expected indexes.
func TestSentTransfer_getPartData(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	for i, part := range parts {
		partData := st.getPartData(uint16(i))

		if !bytes.Equal(part, partData) {
			t.Errorf("Incorrect part #%d.\nexpected: %q\nreceived: %q",
				i, part, partData)
		}
	}
}

// Tests that SentTransfer.getPartData panics when the part number is not within
// the range of part numbers.
func TestSentTransfer_getPartData_OutOfRangePanic(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	invalidPartNum := uint16(len(parts) + 1)
	expectedErr := fmt.Sprintf(errNoPartNum, invalidPartNum, st.tid, st.fileName)

	defer func() {
		r := recover()
		if r == nil || r != expectedErr {
			t.Errorf("getPartData did not return the expected error when the "+
				"part number %d is out of range.\nexpected: %s\nreceived: %+v",
				invalidPartNum, expectedErr, r)
		}
	}()

	_ = st.getPartData(invalidPartNum)
}

// Tests that after setting all parts as arrived via SentTransfer.markArrived,
// there are no unsent parts left and the transfer is marked as Completed.
func TestSentTransfer_markArrived(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	// Mark all parts as arrived
	for i := range parts {
		st.markArrived(uint16(i))
	}

	// Check that all parts are marked as arrived
	unsentParts := st.GetUnsentParts()
	if len(unsentParts) != 0 {
		t.Errorf("There are %d unsent parts.", len(unsentParts))
	}

	if st.status != Completed {
		t.Errorf("Status not correctly marked.\nexpected: %s\nreceived: %s",
			Completed, st.status)
	}
}

// Tests that SentTransfer.markTransferFailed changes the status of the transfer
// to Failed.
func TestSentTransfer_markTransferFailed(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	st.markTransferFailed()

	if st.status != Failed {
		t.Errorf("Status not correctly marked.\nexpected: %s\nreceived: %s",
			Failed, st.status)
	}
}

// Tests that SentTransfer.Status returns the correct status of the transfer.
func TestSentTransfer_Status(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	// Check that it is Running
	if st.Status() != Running {
		t.Errorf("Status returned incorrect status.\nexpected: %s\nreceived: %s",
			Running, st.Status())
	}

	// Mark all parts as arrived
	for i := range parts {
		st.markArrived(uint16(i))
	}

	// Check that it is Completed
	if st.Status() != Completed {
		t.Errorf("Status returned incorrect status.\nexpected: %s\nreceived: %s",
			Completed, st.Status())
	}

	// Mark transfer failed
	st.markTransferFailed()

	// Check that it is Failed
	if st.Status() != Failed {
		t.Errorf("Status returned incorrect status.\nexpected: %s\nreceived: %s",
			Failed, st.Status())
	}
}

// Tests that SentTransfer.TransferID returns the correct transfer ID.
func TestSentTransfer_TransferID(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	if st.TransferID() != st.tid {
		t.Errorf("Incorrect transfer ID.\nexpected: %s\nreceived: %s",
			st.tid, st.TransferID())
	}
}

// Tests that SentTransfer.FileName returns the correct file name.
func TestSentTransfer_FileName(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	if st.FileName() != st.fileName {
		t.Errorf("Incorrect transfer ID.\nexpected: %s\nreceived: %s",
			st.fileName, st.FileName())
	}
}

// Tests that SentTransfer.Recipient returns the correct recipient ID.
func TestSentTransfer_Recipient(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	if !st.Recipient().Cmp(st.recipient) {
		t.Errorf("Incorrect recipient ID.\nexpected: %s\nreceived: %s",
			st.recipient, st.Recipient())
	}
}

// Tests that SentTransfer.FileSize returns the correct file size.
func TestSentTransfer_FileSize(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)
	fileSize := calcFileSize(parts)

	if st.FileSize() != fileSize {
		t.Errorf("Incorrect file size.\nexpected: %d\nreceived: %d",
			fileSize, st.FileSize())
	}
}

// Tests that SentTransfer.NumParts returns the correct number of parts.
func TestSentTransfer_NumParts(t *testing.T) {
	numParts := uint16(16)
	st, _, _, _, _ := newTestSentTransfer(numParts, t)

	if st.NumParts() != numParts {
		t.Errorf("Incorrect number of parts.\nexpected: %d\nreceived: %d",
			numParts, st.NumParts())
	}
}

// Tests that SentTransfer.NumArrived returns the correct number of arrived
// parts.
func TestSentTransfer_NumArrived(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	if st.NumArrived() != 0 {
		t.Errorf("Incorrect number of arrived parts."+
			"\nexpected: %d\nreceived: %d", 0, st.NumArrived())
	}

	// Mark all parts as arrived
	for i := range parts {
		st.markArrived(uint16(i))
	}

	if uint32(st.NumArrived()) != st.partStatus.GetNumKeys() {
		t.Errorf("Incorrect number of arrived parts."+
			"\nexpected: %d\nreceived: %d",
			uint32(st.NumArrived()), st.partStatus.GetNumKeys())
	}
}

// Tests that the state vector returned by SentTransfer.CopyPartStatusVector
// has the same values as the original but is a copy.
func TestSentTransfer_CopyPartStatusVector(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	// Check that the vectors have the same unused parts
	partStatus := st.CopyPartStatusVector()
	if !reflect.DeepEqual(
		partStatus.GetUnusedKeyNums(), st.partStatus.GetUnusedKeyNums()) {
		t.Errorf("Copied part status does not match original."+
			"\nexpected: %v\nreceived: %v",
			st.partStatus.GetUnusedKeyNums(), partStatus.GetUnusedKeyNums())
	}

	// Modify the state
	st.markArrived(5)

	// Check that the copied state is different
	if reflect.DeepEqual(
		partStatus.GetUnusedKeyNums(), st.partStatus.GetUnusedKeyNums()) {
		t.Errorf("Old copied part status matches new status."+
			"\nexpected: %v\nreceived: %v",
			st.partStatus.GetUnusedKeyNums(), partStatus.GetUnusedKeyNums())
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that a SentTransfer loaded via loadSentTransfer matches the original.
func Test_loadSentTransfer(t *testing.T) {
	st, _, _, _, kv := newTestSentTransfer(64, t)

	loadedSt, err := loadSentTransfer(st.tid, kv)
	if err != nil {
		t.Errorf("Failed to load SentTransfer: %+v", err)
	}

	if !reflect.DeepEqual(st, loadedSt) {
		t.Errorf("Loaded SentTransfer does not match original."+
			"\nexpected: %+v\nreceived: %+v", st, loadedSt)
	}
}

// Tests that SentTransfer.Delete deletes the storage backend of the
// SentTransfer and that it cannot be loaded again.
func TestSentTransfer_Delete(t *testing.T) {
	st, _, _, _, kv := newTestSentTransfer(64, t)

	err := st.Delete()
	if err != nil {
		t.Errorf("Delete returned an error: %+v", err)
	}

	_, err = loadSentTransfer(st.tid, kv)
	if err == nil {
		t.Errorf("Loaded sent transfer that was deleted.")
	}
}

// Tests that the fields saved by SentTransfer.save can be loaded from storage.
func TestSentTransfer_save(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(64, t)

	err := st.save()
	if err != nil {
		t.Errorf("save returned an error: %+v", err)
	}

	_, err = st.kv.Get(sentTransferStoreKey, sentTransferStoreVersion)
	if err != nil {
		t.Errorf("Failed to load saved SentTransfer: %+v", err)
	}
}

// Tests that a SentTransfer marshalled via SentTransfer.marshal and
// unmarshalled via unmarshalSentTransfer matches the original.
func TestSentTransfer_marshal_unmarshalSentTransfer(t *testing.T) {
	st := &SentTransfer{
		fileName:  "transferName",
		recipient: id.NewIdFromString("user", id.User, t),
		status:    Failed,
		parts:     [][]byte{[]byte("Message"), []byte("Part")},
	}

	data, err := st.marshal()
	if err != nil {
		t.Errorf("marshal returned an error: %+v", err)
	}

	fileName, recipient, status, parts, err := unmarshalSentTransfer(data)
	if err != nil {
		t.Errorf("Failed to unmarshal SentTransfer: %+v", err)
	}

	if st.fileName != fileName {
		t.Errorf("Incorrect file name.\nexpected: %q\nreceived: %q",
			st.fileName, fileName)
	}

	if !st.recipient.Cmp(recipient) {
		t.Errorf("Incorrect recipient.\nexpected: %s\nreceived: %s",
			st.recipient, recipient)
	}

	if status != status {
		t.Errorf("Incorrect status.\nexpected: %s\nreceived: %s",
			status, status)
	}

	if !reflect.DeepEqual(st.parts, parts) {
		t.Errorf("Incorrect parts.\nexpected: %q\nreceived: %q",
			st.parts, parts)
	}
}

// Consistency test of makeSentTransferPrefix.
func Test_makeSentTransferPrefix_Consistency(t *testing.T) {
	expectedPrefixes := []string{
		"SentFileTransferStore/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/AwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/BAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/BQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/BgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/BwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"SentFileTransferStore/CQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}

	for i, expected := range expectedPrefixes {
		tid := ftCrypto.TransferID{byte(i)}
		prefix := makeSentTransferPrefix(&tid)

		if expected != prefix {
			t.Errorf("Prefix #%d does not match expected."+
				"\nexpected: %q\nreceived: %q", i, expected, prefix)
		}
	}
}

const numPrimeBytes = 512

// newTestSentTransfer creates a new SentTransfer for testing.
func newTestSentTransfer(numParts uint16, t *testing.T) (st *SentTransfer,
	parts [][]byte, key *ftCrypto.TransferKey, numFps uint16, kv *utility.KV) {
	kv = &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	recipient := id.NewIdFromString("recipient", id.User, t)
	keyTmp, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	tid, _ := ftCrypto.NewTransferID(csprng.NewSystemRNG())
	numFps = 2 * numParts
	fileName := "helloFile"
	parts, file := generateTestParts(numParts)

	st, err := newSentTransfer(
		recipient, &keyTmp, &tid, fileName, uint32(len(file)), parts, numFps, kv)
	if err != nil {
		t.Errorf("Failed to make new SentTransfer: %+v", err)
	}

	return st, parts, &keyTmp, numFps, kv
}

// generateTestParts generates a list of file parts of the correct size to be
// encrypted/decrypted.
func generateTestParts(numParts uint16) (parts [][]byte, file []byte) {
	// Calculate part size
	partSize := fileMessage.NewPartMessage(
		format.NewMessage(numPrimeBytes).ContentsSize()).GetPartSize()

	// Create list of parts and fill
	parts = make([][]byte, numParts)
	var buff bytes.Buffer
	buff.Grow(int(numParts) * partSize)
	for i := range parts {
		parts[i] = make([]byte, partSize)
		copy(parts[i], "Hello "+strconv.Itoa(i))
		buff.Write(parts[i])
	}

	return parts, buff.Bytes()
}
