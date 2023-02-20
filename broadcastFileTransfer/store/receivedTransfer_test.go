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

// Tests that newReceivedTransfer returns a new ReceivedTransfer with the
// expected values.
func Test_newReceivedTransfer(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	key, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))
	numFps := uint16(24)
	parts, _ := generateTestParts(16)
	fileSize := uint32(len(parts) * len(parts[0]))
	numParts := uint16(len(parts))
	rtKv := kv.Prefix(makeReceivedTransferPrefix(fid))

	cypherManager, err := cypher.NewManager(&key, numFps, false, rtKv)
	if err != nil {
		t.Errorf("Failed to make new cypher manager: %+v", err)
	}
	partStatus, err := utility.NewStateVector(
		uint32(numParts), false, receivedTransferStatusKey, rtKv)
	if err != nil {
		t.Errorf("Failed to make new state vector: %+v", err)
	}

	expected := &ReceivedTransfer{
		cypherManager:            cypherManager,
		fid:                      fid,
		fileName:                 "fileName",
		recipient:                id.NewIdFromString("blob", id.User, t),
		transferMAC:              []byte("transferMAC"),
		fileSize:                 fileSize,
		numParts:                 numParts,
		parts:                    make([][]byte, numParts),
		partStatus:               partStatus,
		currentCallbackID:        0,
		lastCallbackFingerprints: make(map[uint64]string),
		disableKV:                false,
		kv:                       rtKv,
	}

	rt, err := newReceivedTransfer(expected.recipient, &key, fid,
		expected.fileName, expected.transferMAC, fileSize, numParts, numFps,
		false, kv)
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

// Tests that ReceivedTransfer.MarshalPartialFile returns bytes that can be
// unmarshalled by unmarshalPartialFile to result in the same part list and part
// status vector.
func TestReceivedTransfer_MarshalPartialFile_UnmarshalPartialFile(t *testing.T) {
	// Generate parts and make last file part smaller than the rest
	parts, _ := generateTestParts(16)
	rt, _, _, _, _ := newTestReceivedTransfer(uint16(len(parts)), t)

	for i, p := range parts {
		if i%2 == 0 {
			if err := rt.AddPart(p, i); err != nil {
				t.Errorf("Failed to add part #%d: %+v", i, err)
			}
		}
	}

	data, err := rt.MarshalPartialFile()
	if err != nil {
		t.Errorf("Failed to martial partial file: %+v", err)
	}

	partSize := fileMessage.NewPartMessage(
		format.NewMessage(numPrimeBytes).ContentsSize()).GetPartSize()
	parts, partStatus, err := rt.unmarshalPartialFile(data, partSize)
	if err != nil {
		t.Errorf("Failed to unmarshal partial data: %+v", err)
	}

	if !reflect.DeepEqual(rt.parts, parts) {
		t.Errorf("Incorrect parts.\nexpected:%#v\nreceived:%#v", rt.parts, parts)
	}

	if !reflect.DeepEqual(rt.partStatus, partStatus) {
		t.Errorf("Incorrect part status vector.\nexpected:%#v\nreceived:%#v",
			rt.partStatus, partStatus)
	}
}

// Tests that ReceivedTransfer.GetUnusedCyphers returns the correct number of
// unused cyphers.
func TestReceivedTransfer_GetUnusedCyphers(t *testing.T) {
	numParts := uint16(10)
	rt, _, _, numFps, _ := newTestReceivedTransfer(numParts, t)

	// Check that all cyphers are returned after initialisation
	unusedCyphers := rt.GetUnusedCyphers()
	if len(unusedCyphers) != int(numFps) {
		t.Errorf("Number of unused cyphers does not match original number of "+
			"fingerprints when none have been used.\nexpected: %d\nreceived: %d",
			numFps, len(unusedCyphers))
	}

	// Use every other part
	for i := range unusedCyphers {
		if i%2 == 0 {
			_, _ = unusedCyphers[i].PopCypher()
		}
	}

	// Check that only have the number of parts is returned
	unusedCyphers = rt.GetUnusedCyphers()
	if len(unusedCyphers) != int(numFps)/2 {
		t.Errorf("Number of unused cyphers is not half original number after "+
			"half have been marked as received.\nexpected: %d\nreceived: %d",
			numFps/2, len(unusedCyphers))
	}

	// Use the rest of the parts
	for i := range unusedCyphers {
		_, _ = unusedCyphers[i].PopCypher()
	}

	// Check that no unused parts are returned
	unusedCyphers = rt.GetUnusedCyphers()
	if len(unusedCyphers) != 0 {
		t.Errorf("Number of unused cyphers is not zero after all have been "+
			"marked as received.\nexpected: %d\nreceived: %d",
			0, len(unusedCyphers))
	}
}

// Tests that ReceivedTransfer.FileID returns the correct file ID.
func TestReceivedTransfer_FileID(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	if rt.FileID() != rt.fid {
		t.Errorf("Incorrect file ID.\nexpected: %s\nreceived: %s",
			rt.fid, rt.FileID())
	}
}

// Tests that ReceivedTransfer.FileName returns the correct file name.
func TestReceivedTransfer_FileName(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	if rt.FileName() != rt.fileName {
		t.Errorf("Incorrect file name.\nexpected: %s\nreceived: %s",
			rt.fileName, rt.FileName())
	}
}

// Tests that ReceivedTransfer.Recipient returns the correct recipient ID.
func TestReceivedTransfer_Recipient(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	if rt.Recipient() != rt.recipient {
		t.Errorf("Incorrect recipient ID.\nexpected: %s\nreceived: %s",
			rt.recipient, rt.Recipient())
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

// Tests that ReceivedTransfer.CompareAndSwapCallbackFps correctly swaps the
// fingerprints only when they differ.
func TestReceivedTransfer_CompareAndSwapCallbackFps(t *testing.T) {
	rt, _, _, _, _ := newTestReceivedTransfer(16, t)

	expected := GenerateReceivedFp(true, 1, 3, nil)
	if !rt.CompareAndSwapCallbackFps(5, true, 1, 3, nil) {
		t.Error("Did not swap when there is a new fingerprint.")
	} else if expected != rt.lastCallbackFingerprints[5] {
		t.Errorf("lastCallbackFingerprint not correctly set."+
			"\nexpected: %s\nreceived: %s",
			expected, rt.lastCallbackFingerprints[5])
	}

	if rt.CompareAndSwapCallbackFps(5, true, 1, 3, nil) {
		t.Error("Compared and swapped fingerprints when there was no change.")
	}

	expected = GenerateReceivedFp(false, 4, 15, errors.New("Error"))
	if !rt.CompareAndSwapCallbackFps(5, false, 4, 15, errors.New("Error")) {
		t.Error("Did not swap when there is a new fingerprint.")
	} else if expected != rt.lastCallbackFingerprints[5] {
		t.Errorf("lastCallbackFingerprint not correctly set."+
			"\nexpected: %s\nreceived: %s",
			expected, rt.lastCallbackFingerprints[5])
	}
}

// Consistency test of GenerateReceivedFp.
func Test_GenerateReceivedFp(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	type test struct {
		completed       bool
		received, total int
		err             error
		expected        string
	}
	tests := []test{{
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500),
		nil, "false487168<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500),
		nil, "true423345<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500),
		nil, "false176128<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500),
		nil, "false429467<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500),
		nil, "false328152<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500),
		nil, "false52101<nil>",
	}, {
		prng.Intn(2) == 0, prng.Intn(500), prng.Intn(500),
		errors.New("Bad error"), "false482Bad error",
	},
	}

	for i, tt := range tests {
		fp := GenerateReceivedFp(tt.completed, uint16(tt.received),
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

	partialFile, _ := rt.MarshalPartialFile()
	partSize := fileMessage.NewPartMessage(
		format.NewMessage(numPrimeBytes).ContentsSize()).GetPartSize()
	loadedRt, err := loadReceivedTransfer(rt.fid, partialFile, partSize, kv)
	if err != nil {
		t.Errorf("Failed to load ReceivedTransfer: %+v", err)
	}

	// loadedRt.partStatus = rt.partStatus
	if !reflect.DeepEqual(rt, loadedRt) {
		t.Errorf("Loaded ReceivedTransfer does not match original."+
			"\nexpected: %+v\nreceived: %+v", rt, loadedRt)
		t.Errorf("Loaded ReceivedTransfer does not match original."+
			"\nexpected: %+v\nreceived: %+v", rt.partStatus, loadedRt.partStatus)
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

	partialFile, _ := rt.MarshalPartialFile()
	partSize := fileMessage.NewPartMessage(
		format.NewMessage(numPrimeBytes).ContentsSize()).GetPartSize()
	_, err = loadReceivedTransfer(rt.fid, partialFile, partSize, kv)
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
	recipient := id.NewIdFromString("ftRecipient", id.User, t)
	keyTmp, _ := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	fid := ftCrypto.NewID([]byte("fileData"))
	transferMAC := []byte("I am a transfer MAC")
	numFps = 2 * numParts
	fileName := "helloFile"
	_, file = generateTestParts(numParts)
	fileSize := uint32(len(file))

	rt, err := newReceivedTransfer(recipient, &keyTmp, fid, fileName,
		transferMAC, fileSize, numParts, numFps, false, kv)
	if err != nil {
		t.Errorf("Failed to make new ReceivedTransfer: %+v", err)
	}

	return rt, file, &keyTmp, numFps, kv
}

// Tests that a ReceivedTransfer marshalled via ReceivedTransfer.marshal and
// unmarshalled via unmarshalReceivedTransfer matches the original.
func TestReceivedTransfer_marshal_unmarshalReceivedTransfer(t *testing.T) {
	rt := &ReceivedTransfer{
		fileName:    "transferName",
		recipient:   id.NewIdFromString("recipient", id.User, t),
		transferMAC: []byte("I am a transfer MAC"),
		fileSize:    735,
		numParts:    153,
	}
	expected := receivedTransferDisk{
		rt.fileName, rt.recipient, rt.transferMAC, rt.numParts, rt.fileSize}

	data, err := rt.marshal()
	if err != nil {
		t.Errorf("marshal returned an error: %+v", err)
	}

	info, err := unmarshalReceivedTransfer(data)
	if err != nil {
		t.Errorf("Failed to unmarshal ReceivedTransfer: %+v", err)
	}

	if !reflect.DeepEqual(expected, info) {
		t.Errorf("Incorrect received file info.\nexpected: %+v\nreceived: %+v",
			expected, info)
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
	prng := rand.New(rand.NewSource(42))
	fileData := make([]byte, 64)
	expectedPrefixes := []string{
		"ReceivedFileTransferStore/GmeTCfxGOqRqeIDPGDFroTglaY5zUwwxc9aRbeIf3Co=",
		"ReceivedFileTransferStore/gbpJjHd3tIe8BKykHzm9/WUu/Fp38P6sPp0A8yORIfQ=",
		"ReceivedFileTransferStore/2/ZdG+WNzODJBiFWbJzZAsuMEP0HPNuP0Ogq3LUcxJM=",
		"ReceivedFileTransferStore/TFOBbdqMHgDtzFk9zxIkxbulljnjT4pRXsT5uFCDmLo=",
		"ReceivedFileTransferStore/23OMC+rBmCk+gsutXRThSUNScqEOefqQEs7pu1p3KrI=",
		"ReceivedFileTransferStore/qHu5MUVs83oMqy829cLN6ybTaWT8XvLPT+1r1JDA4Hc=",
		"ReceivedFileTransferStore/kuXqxsezI0kS9Bc5QcSOOCJ7aJzUirqa84LcuNPLZWA=",
		"ReceivedFileTransferStore/MSscKJ0w5yoWsB1Uoq3opFTk3hNEHd35hidPwouBe6I=",
		"ReceivedFileTransferStore/VhdbiYnEpLIet2wCD9KkwGMzGu9IPvoOwDnpu/uPwZU=",
		"ReceivedFileTransferStore/j01ZSSm762TH7mjPimhuASOl7nLxsf1sh0/Yed8MwoE=",
	}

	for i, expected := range expectedPrefixes {
		prng.Read(fileData)
		fid := ftCrypto.NewID(fileData)
		prefix := makeReceivedTransferPrefix(fid)

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
