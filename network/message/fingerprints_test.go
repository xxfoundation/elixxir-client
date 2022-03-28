///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"strconv"
	"testing"
)

// Unit test.
func Test_newFingerprints(t *testing.T) {
	expected := &FingerprintsManager{
		fpMap: make(map[id.ID]map[format.Fingerprint]Processor),
	}

	received := newFingerprints()
	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("New FingerprintsManager did not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Unit test.
func TestFingerprintsManager_pop(t *testing.T) {
	// Construct fingerprint map
	fpTracker := newFingerprints()

	// Construct fingerprint and handler values
	cid := id.NewIdFromString("clientID", id.User, t)
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)

	// Add the values to the tracker
	err := fpTracker.AddFingerprint(cid, fp, mp)
	if err != nil {
		t.Errorf("Failed to add fingerprint: %+v", err)
	}

	// Attempt to retrieve value from map
	received, exists := fpTracker.pop(cid, fp)
	if !exists {
		t.Fatalf("get error: Did not retrieve fingerprint (%s) that "+
			"should have been in map.", fp)
	}

	// Check that received value contains the expected data
	expected := NewMockMsgProcessor(t)
	if !reflect.DeepEqual(received, expected) {
		t.Fatalf("get error: Map does not contain expected data."+
			"\nexpected: %#v\nreceived: %#v", expected, received)
	}

}

// Unit test.
func TestFingerprintsManager_AddFingerprint(t *testing.T) {
	// Construct fingerprint map
	fpTracker := newFingerprints()

	// Construct fingerprint and handler values
	cid := id.NewIdFromString("clientID", id.User, t)
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)

	// Add the values to the tracker
	err := fpTracker.AddFingerprint(cid, fp, mp)
	if err != nil {
		t.Errorf("Failed to add fingerprint: %+v", err)
	}

	// Check that the fingerprint key has a map entry
	received, exists := fpTracker.fpMap[*cid]
	if !exists {
		t.Fatalf("Add did not write to map as expected. "+
			"Fingerprint %s not found in map", fp)
	}

	// Check that received value contains the expected data
	expected := map[format.Fingerprint]Processor{fp: mp}
	if !reflect.DeepEqual(received, expected) {
		t.Fatalf("Add error: Map does not contain expected data."+
			"\nexpected: %v\nreceived: %v", expected, received)
	}
}

func TestFingerprintsManager_DeleteFingerprint(t *testing.T) {

	// Construct fingerprint map
	fpTracker := newFingerprints()

	// Construct fingerprint and handler values
	cid := id.NewIdFromString("clientID", id.User, t)
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)

	// Add the values to the tracker
	err := fpTracker.AddFingerprint(cid, fp, mp)
	if err != nil {
		t.Errorf("Failed to add fingerprint: %+v", err)
	}

	// Remove value from tracker
	fpTracker.DeleteFingerprint(cid, fp)

	// Check that value no longer exists within the map
	if _, exists := fpTracker.fpMap[*cid][fp]; exists {
		t.Fatalf("RemoveFingerprint error: "+
			"Fingerprint %s exists in map after a RemoveFingerprint call", fp)
	}
}

// Unit test.
func TestFingerprintsManager_DeleteClientFingerprints(t *testing.T) {
	// Construct fingerprints map
	fpTracker := newFingerprints()

	// Construct slices of fingerprints and processors
	numTests := 100
	cid := id.NewIdFromString("clientID", id.User, t)
	fingerprints := make([]format.Fingerprint, 0, numTests)
	processors := make([]Processor, 0, numTests)
	for i := 0; i < numTests; i++ {
		fp := format.NewFingerprint([]byte(strconv.Itoa(i)))
		mp := NewMockMsgProcessor(t)

		// Add the values to the tracker
		err := fpTracker.AddFingerprint(cid, fp, mp)
		if err != nil {
			t.Errorf("Failed to add fingerprint: %+v", err)
		}

		fingerprints = append(fingerprints, fp)
		processors = append(processors, mp)
	}

	fpTracker.DeleteClientFingerprints(cid)

	// Make sure every fingerprint is mapped to it's expected handler
	if _, exists := fpTracker.fpMap[*cid]; exists {
		t.Fatalf("RemoveFingerprints error: failed to delete client.")
	}
}

// todo: consider moving this to a test utils somewhere else.. maybe in the interfaces package?
type MockMsgProcessor struct{}

func NewMockMsgProcessor(face interface{}) *MockMsgProcessor {
	switch face.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("NewMockMsgProcessor is restricted to testing only. Got %T", face)
	}

	return &MockMsgProcessor{}
}

func (mock *MockMsgProcessor) MarkFingerprintUsed(_ format.Fingerprint) {
	return
}

func (mock *MockMsgProcessor) Process(format.Message, receptionID.EphemeralIdentity,
	*mixmessages.RoundInfo) {
	return
}
