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
	"strconv"
	"testing"

	"github.com/pkg/errors"

	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store/cypher"
	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store/fileMessage"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Tests that newSentTransfer returns a new SentTransfer with the expected
// values.
func Test_newSentTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))
	numFps := uint16(24)
	parts := [][]byte{[]byte("hello"), []byte("hello"), []byte("hello")}
	stKv := kv.Prefix(makeSentTransferPrefix(fid))

	cypherManager, err := cypher.NewManager(&key, numFps, false, stKv)
	if err != nil {
		t.Errorf("Failed to make new cypher manager: %+v", err)
	}
	partStatus, err := utility.NewMultiStateVector(uint16(len(parts)),
		uint8(numSentStates), stateMap, sentTransferStatusKey, stKv)
	if err != nil {
		t.Errorf("Failed to make new state vector: %+v", err)
	}

	expected := &SentTransfer{
		cypherManager:            cypherManager,
		fid:                      fid,
		fileName:                 "file",
		recipient:                id.NewIdFromString("user", id.User, t),
		fileSize:                 calcFileSize(parts),
		numParts:                 uint16(len(parts)),
		status:                   Running,
		parts:                    parts,
		partStatus:               partStatus,
		currentCallbackID:        0,
		lastCallbackFingerprints: make(map[uint64]string),
		kv:                       stKv,
	}

	st, err := newSentTransfer(expected.recipient, &key, fid,
		expected.fileName, expected.fileSize, parts, numFps, kv)
	if err != nil {
		t.Errorf("newSentTransfer returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, st) {
		t.Errorf("New SentTransfer does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, st)
	}
}

// Tests that SentTransfer.GetUnsentParts returns the correct number of unsent
// parts.
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
			unsentParts[i].MarkSent()
		}
	}

	// Check that only have the number of parts is returned
	unsentParts = st.GetUnsentParts()
	if len(unsentParts) != int(numParts)/2 {
		t.Errorf("Number of unsent parts is not half original number after "+
			"half have been marked as sent.\nexpected: %d\nreceived: %d",
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
		unsentParts[i].MarkSent()
	}

	// Check that no sent parts are returned
	unsentParts = st.GetUnsentParts()
	if len(unsentParts) != 0 {
		t.Errorf("Number of unsent parts is not zero after all have been "+
			"marked as sent.\nexpected: %d\nreceived: %d",
			0, len(unsentParts))
	}
}

// Tests that SentTransfer.GetSentParts returns the correct number of sent
// parts.
func TestSentTransfer_GetSentParts(t *testing.T) {
	numParts := uint16(10)
	st, _, _, _, _ := newTestSentTransfer(numParts, t)

	// Check that there are not sent parts after initialisation
	sentParts := st.GetSentParts()
	if len(sentParts) != 0 {
		t.Errorf("Number of sent parts does not match original number of "+
			"parts when none have been sent.\nexpected: %d\nreceived: %d",
			numParts, 0)
	}

	// Use every other part
	for i, p := range st.GetUnsentParts() {
		if i%2 != 0 {
			p.MarkSent()
		}
	}

	// Check that only have the number of parts is returned
	sentParts = st.GetSentParts()
	if len(sentParts) != int(numParts)/2 {
		t.Errorf("Number of sent parts is not half original number after "+
			"half have been marked as sent.\nexpected: %d\nreceived: %d",
			numParts/2, len(sentParts))
	}

	// Ensure all parts have the proper part number
	for i, p := range sentParts {
		if int(p.partNum) != i*2+1 {
			t.Errorf("Part has incorrect part number."+
				"\nexpected: %d\nreceived: %d", i*2+1, p.partNum)
		}
	}

	// Use the rest of the parts
	for _, p := range st.GetUnsentParts() {
		p.MarkSent()
	}

	// Check that no sent parts are returned
	sentParts = st.GetSentParts()
	if len(sentParts) != int(numParts) {
		t.Errorf("Number of sent parts is not zero after all have been "+
			"marked as sent.\nexpected: %d\nreceived: %d",
			numParts, len(sentParts))
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
	expectedErr := fmt.Sprintf(errNoPartNum, invalidPartNum, st.fid, st.fileName)

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

// Tests that after setting all parts as sent via SentTransfer.markSent, that
// there are no unsent parts left.
func TestSentTransfer_markSent(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	// Mark all parts as sent
	for i := range parts {
		st.markSent(uint16(i))
	}

	// Check that no parts are marked unsent
	if unsentParts := st.GetUnsentParts(); len(unsentParts) != 0 {
		t.Errorf("There are %d unsent parts.", len(unsentParts))
	}

	// Check that all parts are marked as sent
	if unsentParts := st.GetSentParts(); len(unsentParts) != len(parts) {
		t.Errorf("Unexpected number of sent parts."+
			"\nexpected: %d\nreceived: %d", len(parts), len(unsentParts))
	}
}

// Tests that after setting all parts as received via SentTransfer.markReceived,
// that there are no unsent parts left and the transfer is marked as Completed.
func TestSentTransfer_markReceived(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	// Mark all parts as sent
	for i := range parts {
		st.markSent(uint16(i))
	}

	// Mark all parts as received
	for i := range parts {
		st.markReceived(uint16(i))
	}

	// Check that all parts are marked as sent
	if unsentParts := st.GetUnsentParts(); len(unsentParts) != 0 {
		t.Errorf("There are %d unreceived parts.", len(unsentParts))
	}

	if st.status != Completed {
		t.Errorf("Status not correctly marked.\nexpected: %s\nreceived: %s",
			Completed, st.status)
	}
}

// Tests that after setting all parts as unsent via SentTransfer.markForResend,
// that there are no sent parts left.
func TestSentTransfer_markForResend(t *testing.T) {
	const numParts = 16
	st, parts, _, _, _ := newTestSentTransfer(numParts, t)

	// Mark all parts as sent and then received
	for i := range parts {
		st.markSent(uint16(i))
		st.markForResend(uint16(i))
	}

	// Check that there are no sent parts
	if sentParts := st.GetSentParts(); len(sentParts) != 0 {
		t.Errorf("There are %d sent parts: %v", len(sentParts), sentParts)
	}

	// Check that all parts are marked as unsent
	if unsentParts := st.GetUnsentParts(); len(unsentParts) != numParts {
		t.Errorf(
			"Unexpected number of unsent parts: %v\nexpected: %d\nreceived: %d",
			unsentParts, numParts, len(unsentParts))
	}
}

// Tests that SentTransfer.getPartStatus returns the correct status for a part
// as its status is changed.
func TestSentTransfer_getPartStatus(t *testing.T) {
	const numParts = 16
	st, _, _, _, _ := newTestSentTransfer(numParts, t)

	partNum := uint16(5)
	status := st.getPartStatus(partNum)
	if status != UnsentPart {
		t.Errorf("Did not get expected status for part %d."+
			"\nexpected: %s\nreceived: %s", partNum, UnsentPart, status)
	}

	st.markSent(partNum)
	status = st.getPartStatus(partNum)
	if status != SentPart {
		t.Errorf("Did not get expected status for part %d."+
			"\nexpected: %s\nreceived: %s", partNum, SentPart, status)
	}

	st.markForResend(partNum)
	status = st.getPartStatus(partNum)
	if status != UnsentPart {
		t.Errorf("Did not get expected status for part %d."+
			"\nexpected: %s\nreceived: %s", partNum, UnsentPart, status)
	}

	st.markSent(partNum)
	status = st.getPartStatus(partNum)
	if status != SentPart {
		t.Errorf("Did not get expected status for part %d."+
			"\nexpected: %s\nreceived: %s", partNum, SentPart, status)
	}

	st.markReceived(partNum)
	status = st.getPartStatus(partNum)
	if status != ReceivedPart {
		t.Errorf("Did not get expected status for part %d."+
			"\nexpected: %s\nreceived: %s", partNum, ReceivedPart, status)
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

	// Mark all parts as sent
	for i := range parts {
		st.markSent(uint16(i))
	}

	// Mark all parts as received
	for i := range parts {
		st.markReceived(uint16(i))
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

// Tests that SentTransfer.FileID returns the correct file ID.
func TestSentTransfer_FileID(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	if st.FileID() != st.fid {
		t.Errorf("Incorrect file ID.\nexpected: %s\nreceived: %s",
			st.fid, st.FileID())
	}
}

// Tests that SentTransfer.FileName returns the correct file name.
func TestSentTransfer_FileName(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	if st.FileName() != st.fileName {
		t.Errorf("Incorrect file ID.\nexpected: %s\nreceived: %s",
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

// Tests that SentTransfer.NumSent returns the correct number of sent parts.
func TestSentTransfer_NumSent(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	if st.NumSent() != 0 {
		t.Errorf("Incorrect number of sent parts.\nexpected: %d\nreceived: %d",
			0, st.NumSent())
	}

	// Mark all parts as sent
	for i := range parts {
		st.markSent(uint16(i))
	}

	if st.NumSent() != st.partStatus.GetNumKeys() {
		t.Errorf("Incorrect number of sent parts.\nexpected: %d\nreceived: %d",
			uint32(st.NumSent()), st.partStatus.GetNumKeys())
	}
}

// Tests that SentTransfer.NumReceived returns the correct number of received
// parts.
func TestSentTransfer_NumReceived(t *testing.T) {
	st, parts, _, _, _ := newTestSentTransfer(16, t)

	if st.NumReceived() != 0 {
		t.Errorf("Incorrect number of sent parts.\nexpected: %d\nreceived: %d",
			0, st.NumSent())
	}

	// Mark all parts as received
	for i := range parts {
		st.markSent(uint16(i))
		st.markReceived(uint16(i))
	}

	if st.NumReceived() != st.partStatus.GetNumKeys() {
		t.Errorf("Incorrect number of received parts.\nexpected: %d\nreceived: %d",
			uint32(st.NumReceived()), st.partStatus.GetNumKeys())
	}
}

// Tests that the state vector returned by SentTransfer.CopyPartStatusVector
// has the same values as the original but is a copy.
func TestSentTransfer_CopyPartStatusVector(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	// Check that the vectors have the same unused parts
	partStatus := st.CopyPartStatusVector()
	expectedUnsentKeys := partStatus.GetKeys(uint8(UnsentPart))
	receivedUnsentKeys := st.partStatus.GetKeys(uint8(UnsentPart))
	if !reflect.DeepEqual(expectedUnsentKeys, receivedUnsentKeys) {
		t.Errorf("Copied part status does not match original."+
			"\nexpected: %v\nreceived: %v",
			expectedUnsentKeys, receivedUnsentKeys)
	}

	// Modify the state
	st.markSent(5)

	// Check that the copied state is different
	expectedUnsentKeys = partStatus.GetKeys(uint8(UnsentPart))
	receivedUnsentKeys = st.partStatus.GetKeys(uint8(UnsentPart))
	if reflect.DeepEqual(expectedUnsentKeys, receivedUnsentKeys) {
		t.Errorf("Old copied part status matches new status."+
			"\nexpected: %v\nreceived: %v",
			expectedUnsentKeys, receivedUnsentKeys)
	}
}

// Tests that SentTransfer.CompareAndSwapCallbackFps correctly swaps the
// fingerprints only when they differ.
func TestSentTransfer_CompareAndSwapCallbackFps(t *testing.T) {
	st, _, _, _, _ := newTestSentTransfer(16, t)

	expected := generateSentFp(true, 1, 2, 3, nil)
	if !st.CompareAndSwapCallbackFps(5, true, 1, 2, 3, nil) {
		t.Error("Did not swap when there is a new fingerprint.")
	} else if expected != st.lastCallbackFingerprints[5] {
		t.Errorf("lastCallbackFingerprint not correctly set."+
			"\nexpected: %s\nreceived: %s",
			expected, st.lastCallbackFingerprints[5])
	}

	if st.CompareAndSwapCallbackFps(5, true, 1, 2, 3, nil) {
		t.Error("Compared and swapped fingerprints when there was no change.")
	}

	expected = generateSentFp(false, 4, 5, 15, errors.New("Error"))
	if !st.CompareAndSwapCallbackFps(5, false, 4, 5, 15, errors.New("Error")) {
		t.Error("Did not swap when there is a new fingerprint.")
	} else if expected != st.lastCallbackFingerprints[5] {
		t.Errorf("lastCallbackFingerprint not correctly set."+
			"\nexpected: %s\nreceived: %s",
			expected, st.lastCallbackFingerprints[5])
	}
}

// Consistency test of generateSentFp.
func Test_generateSentFp_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	type test struct {
		completed             bool
		sent, received, total int
		err                   error
		expected              string
	}
	tests := []test{{
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500), prng.Intn(500),
		nil, "false487168250<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500), prng.Intn(500),
		nil, "false345357176<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500), prng.Intn(500),
		nil, "true143429467<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500), prng.Intn(500),
		nil, "false328152153<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500), prng.Intn(500),
		nil, "true101354<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500), prng.Intn(500),
		nil, "true434444275<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500), prng.Intn(500),
		errors.New("Bad error"), "true3612419Bad error",
	},
	}

	for i, tt := range tests {
		fp := generateSentFp(tt.completed, uint16(tt.sent), uint16(tt.received),
			uint16(tt.total), tt.err)

		if fp != tt.expected {
			t.Errorf("Fingerprint %d does not match expected."+
				"\nexpected: %s\nreceived: %s", i, tt.expected, fp)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that a SentTransfer loaded via loadSentTransfer matches the original.
func Test_loadSentTransfer(t *testing.T) {
	st, _, _, _, kv := newTestSentTransfer(64, t)

	loadedSt, err := loadSentTransfer(st.fid, st.parts, kv)
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

	_, err = loadSentTransfer(st.fid, st.parts, kv)
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
	}
	expected := sentTransferDisk{st.fileName, st.recipient, st.status}

	data, err := st.marshal()
	if err != nil {
		t.Errorf("marshal returned an error: %+v", err)
	}

	info, err := unmarshalSentTransfer(data)
	if err != nil {
		t.Errorf("Failed to unmarshal SentTransfer: %+v", err)
	}

	if !reflect.DeepEqual(expected, info) {
		t.Errorf("Incorrect sent file info.\nexpected: %+v\nreceived: %+v",
			expected, info)
	}
}

// Consistency test of makeSentTransferPrefix.
func Test_makeSentTransferPrefix_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	fileData := make([]byte, 64)
	expectedPrefixes := []string{
		"SentFileTransferStore/GmeTCfxGOqRqeIDPGDFroTglaY5zUwwxc9aRbeIf3Co=",
		"SentFileTransferStore/gbpJjHd3tIe8BKykHzm9/WUu/Fp38P6sPp0A8yORIfQ=",
		"SentFileTransferStore/2/ZdG+WNzODJBiFWbJzZAsuMEP0HPNuP0Ogq3LUcxJM=",
		"SentFileTransferStore/TFOBbdqMHgDtzFk9zxIkxbulljnjT4pRXsT5uFCDmLo=",
		"SentFileTransferStore/23OMC+rBmCk+gsutXRThSUNScqEOefqQEs7pu1p3KrI=",
		"SentFileTransferStore/qHu5MUVs83oMqy829cLN6ybTaWT8XvLPT+1r1JDA4Hc=",
		"SentFileTransferStore/kuXqxsezI0kS9Bc5QcSOOCJ7aJzUirqa84LcuNPLZWA=",
		"SentFileTransferStore/MSscKJ0w5yoWsB1Uoq3opFTk3hNEHd35hidPwouBe6I=",
		"SentFileTransferStore/VhdbiYnEpLIet2wCD9KkwGMzGu9IPvoOwDnpu/uPwZU=",
		"SentFileTransferStore/j01ZSSm762TH7mjPimhuASOl7nLxsf1sh0/Yed8MwoE=",
	}

	for i, expected := range expectedPrefixes {
		prng.Read(fileData)
		fid := ftCrypto.NewID(fileData)
		prefix := makeSentTransferPrefix(fid)

		if expected != prefix {
			t.Errorf("Prefix #%d does not match expected."+
				"\nexpected: %q\nreceived: %q", i, expected, prefix)
		}
	}
}

const numPrimeBytes = 512

// newTestSentTransfer creates a new SentTransfer for testing.
func newTestSentTransfer(numParts uint16, t *testing.T) (st *SentTransfer,
	parts [][]byte, key *ftCrypto.TransferKey, numFps uint16, kv *versioned.KV) {
	kv = versioned.NewKV(ekv.MakeMemstore())
	recipient := id.NewIdFromString("recipient", id.User, t)
	keyTmp, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))
	numFps = 2 * numParts
	fileName := "helloFile"
	parts, file := generateTestParts(numParts)

	st, err := newSentTransfer(
		recipient, &keyTmp, fid, fileName, uint32(len(file)), parts, numFps, kv)
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
