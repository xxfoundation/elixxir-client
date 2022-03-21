///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/primitives/format"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

// Unit test.
func TestNewFingerprints(t *testing.T) {
	expected := &Fingerprints{
		fingerprints: make(map[format.Fingerprint]*Processor),
		RWMutex:      sync.RWMutex{},
	}

	received := NewFingerprints()

	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("NewFingerprint error: Did not construct expected object."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}
}

// Unit test.
func TestFingerprints_Get(t *testing.T) {
	// Construct fingerprint map
	fpTracker := NewFingerprints()

	// Construct fingerprint and processor values
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)

	// Add the values to the tracker
	fpTracker.AddFingerprint(fp, mp)

	// Attempt to retrieve value from map
	received, exists := fpTracker.Get(fp)
	if !exists {
		t.Fatalf("Get error: Did not retrieve fingerprint (%s) that "+
			"should have been in map.", fp)
	}

	// Check that received value contains the expected data
	expected := newProcessor(mp)
	if !reflect.DeepEqual(received, expected) {
		t.Fatalf("Get error: Map does not contain expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}

}

// Unit test.
func TestFingerprints_AddFingerprint(t *testing.T) {
	// Construct fingerprint map
	fpTracker := NewFingerprints()

	// Construct fingerprint and processor values
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)

	// Add the values to the tracker
	fpTracker.AddFingerprint(fp, mp)

	// Check that the fingerprint key has a map entry
	received, exists := fpTracker.fingerprints[fp]
	if !exists {
		t.Fatalf("AddFingerprint did not write to map as expected. "+
			"Fingerprint %s not found in map", fp)
	}

	// Check that received value contains the expected data
	expected := newProcessor(mp)
	if !reflect.DeepEqual(received, expected) {
		t.Fatalf("AddFingerprint error: Map does not contain expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}
}

// Unit test.
func TestFingerprints_AddFingerprints(t *testing.T) {
	// Construct fingerprints map
	fpTracker := NewFingerprints()

	// Construct slices of fingerprints and processors
	numTests := 100
	fingerprints := make([]format.Fingerprint, 0, numTests)
	processors := make([]interfaces.MessageProcessorFP, 0, numTests)
	for i := 0; i < numTests; i++ {
		fp := format.NewFingerprint([]byte(strconv.Itoa(i)))
		mp := NewMockMsgProcessor(t)

		fingerprints = append(fingerprints, fp)
		processors = append(processors, mp)
	}

	// Add slices to map
	err := fpTracker.AddFingerprints(fingerprints, processors)
	if err != nil {
		t.Fatalf("AddFingerprints unexpected error: %v", err)
	}

	// Make sure every fingerprint is mapped to it's expected processor
	for i, expected := range fingerprints {
		received, exists := fpTracker.fingerprints[expected]
		if !exists {
			t.Errorf("AddFingerprints did not write to map as expected. "+
				"Fingerprint number %d (value: %s) not found in map", i, expected)
		}

		if !reflect.DeepEqual(received, expected) {
			t.Fatalf("AddFingerprints error: Map does not contain expected data for "+
				"fingerprint number %d."+
				"\nExpected: %v"+
				"\nReceived: %v", i, expected, received)
		}
	}

}

// Error case: Call Fingerprints.AddFingerprints with fingerprint and processor
// slices of different lengths.
func TestFingerprints_AddFingerprints_Error(t *testing.T) {
	// Construct fingerprint map
	fpTracker := NewFingerprints()

	// Construct 2 slices of different lengths
	fingerprints := []format.Fingerprint{
		format.NewFingerprint([]byte("1")),
		format.NewFingerprint([]byte("2")),
		format.NewFingerprint([]byte("3")),
	}
	processors := []interfaces.MessageProcessorFP{
		NewMockMsgProcessor(t),
	}

	// Attempt to add fingerprints
	err := fpTracker.AddFingerprints(fingerprints, processors)
	if err == nil {
		t.Fatalf("AddFingerprints should have received an error with mismatched " +
			"slices length")
	}

}

func TestFingerprints_RemoveFingerprint(t *testing.T) {

	// Construct fingerprint map
	fpTracker := NewFingerprints()

	// Construct fingerprint and processor values
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)

	// Add the values to the tracker
	fpTracker.AddFingerprint(fp, mp)

	// Remove value from tracker
	fpTracker.RemoveFingerprint(fp)

	// Check that value no longer exists within the map
	if _, exists := fpTracker.fingerprints[fp]; exists {
		t.Fatalf("RemoveFingerprint error: "+
			"Fingerprint %s exists in map after a RemoveFingerprint call", fp)
	}
}

// Unit test.
func TestFingerprints_RemoveFingerprints(t *testing.T) {
	// Construct fingerprints map
	fpTracker := NewFingerprints()

	// Construct slices of fingerprints and processors
	numTests := 100
	fingerprints := make([]format.Fingerprint, 0, numTests)
	processors := make([]interfaces.MessageProcessorFP, 0, numTests)
	for i := 0; i < numTests; i++ {
		fp := format.NewFingerprint([]byte(strconv.Itoa(i)))
		mp := NewMockMsgProcessor(t)

		fingerprints = append(fingerprints, fp)
		processors = append(processors, mp)
	}

	// Add slices to map
	err := fpTracker.AddFingerprints(fingerprints, processors)
	if err != nil {
		t.Fatalf("AddFingerprints unexpected error: %v", err)
	}

	fpTracker.RemoveFingerprints(fingerprints)

	// Make sure every fingerprint is mapped to it's expected processor
	for i, expected := range fingerprints {

		if received, exists := fpTracker.fingerprints[expected]; !exists {
			t.Fatalf("RemoveFingerprints error: Map does not contain "+
				"expected data for fingerprint number %d."+
				"\nExpected: %v"+
				"\nReceived: %v", i, expected, received)
		}

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

func (mock *MockMsgProcessor) MarkFingerprintUsed(fingerprint format.Fingerprint) {
	return
}

func (mock *MockMsgProcessor) Process(message format.Message, fingerprint format.Fingerprint) {
	return
}
