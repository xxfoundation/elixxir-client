////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"reflect"
	"testing"
)

// Happy path
func TestRatchet_AddService(t *testing.T) {
	// Initialize ratchet
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	// Add service to ratchet
	tag := "youreIt"
	procName := "mario"
	expected := &mockProcessor{name: procName}
	err = r.AddService(tag, expected)
	if err != nil {
		t.Fatalf("AddService error: %+v", err)
	}

	// Ensure service exists within the map
	received, ok := r.services[tag]
	if !ok {
		t.Fatalf("Could not find processor in map")
	}

	// Ensure the entry in the map is what was added
	if !reflect.DeepEqual(received, expected) {
		t.Fatalf("Did not receive expected service."+
			"\nExpected: %+v"+
			"\nReceived: %+v", expected, received)
	}

}

// Error path: attempting to add to an already existing tag
// should result in an error
func TestRatchet_AddService_DuplicateAddErr(t *testing.T) {
	// Initialize ratchet
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	// Add a mock service
	tag := "youreIt"
	err = r.AddService(tag, &mockProcessor{})
	if err != nil {
		t.Fatalf("AddService error: %+v", err)
	}

	// Add a mock service with the same tag should result in an error
	err = r.AddService(tag, &mockProcessor{})
	if err == nil {
		t.Fatalf("Expected error: " +
			"Should not be able to add more than one service")
	}

}

// Happy path
func TestRatchet_RemoveService(t *testing.T) {
	// Initialize ratchet
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	// Add a mock service
	tag := "youreIt"
	err = r.AddService(tag, &mockProcessor{})
	if err != nil {
		t.Fatalf("AddService error: %+v", err)
	}

	// Remove the service
	err = r.RemoveService(tag)
	if err != nil {
		t.Fatalf("RemoveService error: %+v", err)
	}

	// Ensure service does not exist within the map
	_, ok := r.services[tag]
	if ok {
		t.Fatalf("Entry with tag %s should not be in map", tag)
	}

}

// Error path: removing a service that does not exist
func TestRatchet_RemoveService_DoesNotExistError(t *testing.T) {
	// Initialize ratchet
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	// Remove the service that does not exist
	tag := "youreIt"
	err = r.RemoveService(tag)
	if err == nil {
		t.Fatalf("Expected error: RemoveService should return an error when " +
			"trying to move a service that was not added")
	}

	// Ensure service does not exist within the map
	_, ok := r.services[tag]
	if ok {
		t.Fatalf("Entry with tag %s should not be in map", tag)
	}

}
