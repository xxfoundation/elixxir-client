package partition

import (
	"bytes"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Tests happy path of New().
func TestNew(t *testing.T) {
	rootKv := versioned.NewKV(make(ekv.Memstore))
	expectedStore := &Store{
		multiParts: make(map[multiPartID]*multiPartMessage),
		kv:         rootKv.Prefix(packagePrefix),
	}

	store := New(rootKv)

	if !reflect.DeepEqual(expectedStore, store) {
		t.Errorf("New() did not return the expecte Store."+
			"\n\texpected: %v\n\treceived: %v", expectedStore, store)
	}
}

// Tests happy path of Store.AddFirst().
func TestStore_AddFirst(t *testing.T) {
	part := []byte("Test message.")
	s := New(versioned.NewKV(ekv.Memstore{}))

	msg, complete := s.AddFirst(id.NewIdFromString("User", id.User, t),
		message.Text, 5, 0, 1, time.Now(), part,
		[]byte{0})

	if !complete {
		t.Errorf("AddFirst() returned that the message was not complete.")
	}

	if !bytes.Equal(part, msg.Payload) {
		t.Errorf("AddFirst() returned message with invalid payload."+
			"\n\texpected: %v\n\treceived: %v", part, msg.Payload)
	}
}

// Tests happy path of Store.Add().
func TestStore_Add(t *testing.T) {
	part1 := []byte("Test message.")
	part2 := []byte("Second Sentence.")
	s := New(versioned.NewKV(ekv.Memstore{}))

	msg, complete := s.AddFirst(id.NewIdFromString("User", id.User, t),
		message.Text, 5, 0, 2, time.Now(), part1,
		[]byte{0})

	if complete {
		t.Errorf("AddFirst() returned that the message was complete.")
	}

	msg, complete = s.Add(id.NewIdFromString("User", id.User, t),
		5, 1, part2, []byte{0})
	if !complete {
		t.Errorf("AddFirst() returned that the message was not complete.")
	}

	part := append(part1, part2...)
	if !bytes.Equal(part, msg.Payload) {
		t.Errorf("AddFirst() returned message with invalid payload."+
			"\n\texpected: %v\n\treceived: %v", part, msg.Payload)
	}
}
