///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partition

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests the creation part of loadOrCreateMultiPartMessage().
func Test_loadOrCreateMultiPartMessage_Create(t *testing.T) {
	// Set up expected test value
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	expectedMpm := &multiPartMessage{
		Sender:          id.NewIdFromUInt(prng.Uint64(), id.User, t),
		MessageID:       prng.Uint64(),
		NumParts:        0,
		PresentParts:    0,
		SenderTimestamp: time.Time{},
		MessageType:     0,
		kv:              versioned.NewKV(make(ekv.Memstore)),
	}
	expectedData, err := json.Marshal(expectedMpm)
	if err != nil {
		t.Fatalf("Failed to marshal expected multiPartMessage: %v", err)
	}

	// Make new multiPartMessage
	mpm := loadOrCreateMultiPartMessage(expectedMpm.Sender,
		expectedMpm.MessageID, expectedMpm.kv)

	CheckMultiPartMessages(expectedMpm, mpm, t)

	obj, err := mpm.kv.Get(messageKey, 0)
	if err != nil {
		t.Errorf("get() failed to get multiPartMessage from key value store: %v", err)
	}

	if !bytes.Equal(expectedData, obj.Data) {
		t.Errorf("loadOrCreateMultiPartMessage() did not save the "+
			"multiPartMessage correctly.\n\texpected: %+v\n\treceived: %+v",
			expectedData, obj.Data)
	}
}

// Tests the loading part of loadOrCreateMultiPartMessage().
func Test_loadOrCreateMultiPartMessage_Load(t *testing.T) {
	// Set up expected test value
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	expectedMpm := &multiPartMessage{
		Sender:          id.NewIdFromUInt(prng.Uint64(), id.User, t),
		MessageID:       prng.Uint64(),
		NumParts:        0,
		PresentParts:    0,
		SenderTimestamp: time.Time{},
		MessageType:     0,
		kv:              versioned.NewKV(make(ekv.Memstore)),
	}
	err := expectedMpm.save()
	if err != nil {
		t.Fatalf("Failed to save multiPartMessage: %v", err)
	}

	// Make new multiPartMessage
	mpm := loadOrCreateMultiPartMessage(expectedMpm.Sender,
		expectedMpm.MessageID, expectedMpm.kv)

	CheckMultiPartMessages(expectedMpm, mpm, t)
}

func CheckMultiPartMessages(expectedMpm *multiPartMessage, mpm *multiPartMessage, t *testing.T) {
	// The kv differs because it has prefix called, so we compare fields individually
	if expectedMpm.SenderTimestamp != mpm.SenderTimestamp {
		t.Errorf("timestamps mismatch: expected %v, got %v", expectedMpm.SenderTimestamp, mpm.SenderTimestamp)
	}
	if expectedMpm.MessageType != mpm.MessageType {
		t.Errorf("messagetype mismatch: expected %v, got %v", expectedMpm.MessageID, mpm.MessageID)
	}
	if expectedMpm.MessageID != mpm.MessageID {
		t.Errorf("messageid mismatch: expected %v, got %v", expectedMpm.MessageID, mpm.MessageID)
	}
	if expectedMpm.NumParts != mpm.NumParts {
		t.Errorf("numparts mismatch: expected %v, got %v", expectedMpm.NumParts, mpm.NumParts)
	}
	if expectedMpm.PresentParts != mpm.PresentParts {
		t.Errorf("presentparts mismatch: expected %v, got %v", expectedMpm.PresentParts, mpm.PresentParts)
	}
	if !expectedMpm.Sender.Cmp(mpm.Sender) {
		t.Errorf("sender mismatch: expected %v, got %v", expectedMpm.Sender, mpm.Sender)
	}
	if len(expectedMpm.parts) != len(mpm.parts) {
		t.Error("parts different length")
	}
	for i := range expectedMpm.parts {
		if !bytes.Equal(expectedMpm.parts[i], mpm.parts[i]) {
			t.Errorf("parts differed at index %v", i)
		}
	}
}

// Tests happy path of multiPartMessage.AddFingerprint().
func TestMultiPartMessage_Add(t *testing.T) {
	// Generate test values
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	mpm := loadOrCreateMultiPartMessage(id.NewIdFromUInt(prng.Uint64(), id.User, t),
		prng.Uint64(), versioned.NewKV(make(ekv.Memstore)))
	partNums, parts := generateParts(prng, 0)

	for i := range partNums {
		mpm.Add(partNums[i], parts[i])
	}

	for i, p := range partNums {
		if !bytes.Equal(mpm.parts[p], parts[i]) {
			t.Errorf("Incorrect part at index %d (%d)."+
				"\n\texpected: %v\n\treceived: %v", p, i, parts[i], mpm.parts[p])
		}
	}

	if len(partNums) != int(mpm.PresentParts) {
		t.Errorf("Incorrect PresentParts.\n\texpected: %d\n\treceived: %d",
			len(partNums), int(mpm.PresentParts))
	}

	expectedData, err := json.Marshal(mpm)
	if err != nil {
		t.Fatalf("Failed to marshal expected multiPartMessage: %v", err)
	}

	obj, err := mpm.kv.Get(messageKey, 0)
	if err != nil {
		t.Errorf("get() failed to get multiPartMessage from key value store: %v", err)
	}

	if !bytes.Equal(expectedData, obj.Data) {
		t.Errorf("loadOrCreateMultiPartMessage() did not save the "+
			"multiPartMessage correctly.\n\texpected: %+v\n\treceived: %+v",
			expectedData, obj.Data)
	}
}

// Tests happy path of multiPartMessage.AddFirst().
func TestMultiPartMessage_AddFirst(t *testing.T) {
	// Generate test values
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	expectedMpm := &multiPartMessage{
		Sender:          id.NewIdFromUInt(prng.Uint64(), id.User, t),
		MessageID:       prng.Uint64(),
		NumParts:        uint8(prng.Uint32()),
		PresentParts:    1,
		SenderTimestamp: netTime.Now(),
		MessageType:     message.NoType,
		parts:           make([][]byte, 3),
		kv:              versioned.NewKV(make(ekv.Memstore)),
	}
	expectedMpm.parts[2] = []byte{5, 8, 78, 9}
	npm := loadOrCreateMultiPartMessage(expectedMpm.Sender,
		expectedMpm.MessageID, expectedMpm.kv)

	npm.AddFirst(expectedMpm.MessageType, 2, expectedMpm.NumParts,
		expectedMpm.SenderTimestamp, netTime.Now(), expectedMpm.parts[2])

	CheckMultiPartMessages(expectedMpm, npm, t)

	data, err := loadPart(npm.kv, 2)
	if err != nil {
		t.Errorf("loadPart() produced an error: %v", err)
	}

	if !bytes.Equal(data, expectedMpm.parts[2]) {
		t.Errorf("AddFirst() did not save multiPartMessage correctly."+
			"\n\texpected: %#v\n\treceived: %#v", expectedMpm.parts[2], data)
	}
}

// Tests happy path of multiPartMessage.IsComplete().
func TestMultiPartMessage_IsComplete(t *testing.T) {
	// Create multiPartMessage and fill with random parts
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	mid := prng.Uint64()
	mpm := loadOrCreateMultiPartMessage(id.NewIdFromUInt(prng.Uint64(), id.User, t),
		mid, versioned.NewKV(make(ekv.Memstore)))
	partNums, parts := generateParts(prng, 75)

	// Check that IsComplete() is false where there are no parts
	msg, complete := mpm.IsComplete([]byte{0})
	if complete {
		t.Error("IsComplete() returned true when NumParts == 0.")
	}

	mpm.AddFirst(message.XxMessage, partNums[0], 75, netTime.Now(), netTime.Now(), parts[0])
	for i := range partNums {
		if i > 0 {
			mpm.Add(partNums[i], parts[i])
		}
	}

	msg, complete = mpm.IsComplete([]byte{0})
	if !complete {
		t.Error("IsComplete() returned false when the message should be complete.")
	}

	var payload []byte
	for _, b := range mpm.parts {
		payload = append(payload, b...)
	}

	expectedMsg := message.Receive{
		Payload:     payload,
		MessageType: mpm.MessageType,
		Sender:      mpm.Sender,
		Timestamp:   msg.Timestamp,
		Encryption:  0,
		ID:          e2e.NewMessageID([]byte{0}, mid),
	}

	if !reflect.DeepEqual(expectedMsg, msg) {
		t.Errorf("IsComplete() did not return the expected message."+
			"\n\texpected: %v\n\treceived: %v", expectedMsg, msg)
	}

}

// Tests happy path of multiPartMessage.delete().
func TestMultiPartMessage_delete(t *testing.T) {
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	kv := versioned.NewKV(make(ekv.Memstore))
	mpm := loadOrCreateMultiPartMessage(id.NewIdFromUInt(prng.Uint64(), id.User, t),
		prng.Uint64(), kv)

	mpm.delete()
	obj, err := kv.Get(messageKey, 0)
	if ekv.Exists(err) {
		t.Errorf("delete() did not properly delete key %s."+
			"\n\tobject received: %+v", messageKey, obj)
	}
}

// generateParts generates a list of test part numbers and a list of test parts.
func generateParts(r *rand.Rand, numParts uint8) ([]uint8, [][]byte) {
	if numParts == 0 {
		numParts = uint8(25 + r.Intn(150))
	}
	partNums := make([]uint8, numParts)
	parts := make([][]byte, len(partNums))
	nums := map[uint8]bool{}

	for i := range partNums {
		n := uint8(r.Intn(int(numParts)))
		for ; nums[n] == true; n = uint8(r.Intn(int(numParts))) {
		}
		nums[n] = true
		partNums[i] = n
		parts[i] = make([]byte, r.Int31n(100))
		_, _ = r.Read(parts[i])
	}

	return partNums, parts
}
