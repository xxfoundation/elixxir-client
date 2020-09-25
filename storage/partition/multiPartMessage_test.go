package partition

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests the creation part of loadOrCreateMultiPartMessage().
func Test_loadOrCreateMultiPartMessage_Create(t *testing.T) {
	// Set up expected test value
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	expectedMpm := &multiPartMessage{
		Sender:       id.NewIdFromUInt(prng.Uint64(), id.User, t),
		MessageID:    prng.Uint64(),
		NumParts:     0,
		PresentParts: 0,
		Timestamp:    time.Time{},
		MessageType:  0,
		kv:           versioned.NewKV(make(ekv.Memstore)),
	}
	expectedData, err := json.Marshal(expectedMpm)
	if err != nil {
		t.Fatalf("Failed to marshal expected multiPartMessage: %v", err)
	}

	// Make new multiPartMessage
	mpm := loadOrCreateMultiPartMessage(expectedMpm.Sender,
		expectedMpm.MessageID, expectedMpm.kv)

	if !reflect.DeepEqual(expectedMpm, mpm) {
		t.Errorf("loadOrCreateMultiPartMessage() did not create the correct "+
			"multiPartMessage.\n\texpected: %+v\n\treceived: %+v",
			expectedMpm, mpm)
	}

	obj, err := expectedMpm.kv.Get(makeMultiPartMessageKey(expectedMpm.Sender,
		expectedMpm.MessageID))
	if err != nil {
		t.Errorf("Get() failed to get multiPartMessage from key value store: %v", err)
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
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	expectedMpm := &multiPartMessage{
		Sender:       id.NewIdFromUInt(prng.Uint64(), id.User, t),
		MessageID:    prng.Uint64(),
		NumParts:     0,
		PresentParts: 0,
		Timestamp:    time.Time{},
		MessageType:  0,
		kv:           versioned.NewKV(make(ekv.Memstore)),
	}
	err := expectedMpm.save()
	if err != nil {
		t.Fatalf("Failed to save multiPartMessage: %v", err)
	}

	// Make new multiPartMessage
	mpm := loadOrCreateMultiPartMessage(expectedMpm.Sender,
		expectedMpm.MessageID, expectedMpm.kv)

	if !reflect.DeepEqual(expectedMpm, mpm) {
		t.Errorf("loadOrCreateMultiPartMessage() did not create the correct "+
			"multiPartMessage.\n\texpected: %+v\n\treceived: %+v",
			expectedMpm, mpm)
	}
}

// Tests happy path of multiPartMessage.Add().
func TestMultiPartMessage_Add(t *testing.T) {
	// Generate test values
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
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

	obj, err := mpm.kv.Get(makeMultiPartMessageKey(mpm.Sender, mpm.MessageID))
	if err != nil {
		t.Errorf("Get() failed to get multiPartMessage from key value store: %v", err)
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
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	expectedMpm := &multiPartMessage{
		Sender:       id.NewIdFromUInt(prng.Uint64(), id.User, t),
		MessageID:    prng.Uint64(),
		NumParts:     uint8(prng.Uint32()),
		PresentParts: 1,
		Timestamp:    time.Now(),
		MessageType:  message.NoType,
		parts:        make([][]byte, 3),
		kv:           versioned.NewKV(make(ekv.Memstore)),
	}
	expectedMpm.parts[2] = []byte{5, 8, 78, 9}
	npm := loadOrCreateMultiPartMessage(expectedMpm.Sender,
		expectedMpm.MessageID, expectedMpm.kv)

	npm.AddFirst(expectedMpm.MessageType, 2, expectedMpm.NumParts,
		expectedMpm.Timestamp, expectedMpm.parts[2])

	if !reflect.DeepEqual(expectedMpm, npm) {
		t.Errorf("AddFirst() did not produce correct multiPartMessage."+
			"\n\texpected: %#v\n\treceived: %#v", expectedMpm, npm)
	}

	data, err := loadPart(npm.kv, npm.Sender, npm.MessageID, 2)
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
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mpm := loadOrCreateMultiPartMessage(id.NewIdFromUInt(prng.Uint64(), id.User, t),
		prng.Uint64(), versioned.NewKV(make(ekv.Memstore)))
	partNums, parts := generateParts(prng, 75)

	// Check that IsComplete() is false where there are no parts
	msg, complete := mpm.IsComplete()
	if complete {
		t.Error("IsComplete() returned true when NumParts == 0.")
	}

	mpm.AddFirst(message.Text, partNums[0], 75, time.Now(), parts[0])
	for i := range partNums {
		if i > 0 {
			mpm.Add(partNums[i], parts[i])
		}
	}

	msg, complete = mpm.IsComplete()
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
		Timestamp:   time.Time{},
		Encryption:  0,
	}

	if !reflect.DeepEqual(expectedMsg, msg) {
		t.Errorf("IsComplete() did not return the expected message."+
			"\n\texpected: %v\n\treceived: %v", expectedMsg, msg)
	}

}

// Tests happy path of multiPartMessage.delete().
func TestMultiPartMessage_delete(t *testing.T) {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	kv := versioned.NewKV(make(ekv.Memstore))
	mpm := loadOrCreateMultiPartMessage(id.NewIdFromUInt(prng.Uint64(), id.User, t),
		prng.Uint64(), kv)
	key := makeMultiPartMessageKey(mpm.Sender, mpm.MessageID)

	mpm.delete()
	obj, err := kv.Get(key)
	if ekv.Exists(err) {
		t.Errorf("delete() did not properly delete key %s."+
			"\n\tobject received: %+v", key, obj)
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
