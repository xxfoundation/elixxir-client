///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"errors"
	"gitlab.com/elixxir/client/stoppable"
	"reflect"
	"testing"
	"time"
)

// Unit test
func TestNewServices(t *testing.T) {
	expected := &services{
		services:  make([]Service, 0),
		stoppable: stoppable.NewMulti("services"),
		state:     Stopped,
	}

	received := newServices()

	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("Unexpected value in constructor (newServices): "+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received)
	}
}

// Unit test
func TestServices_Add(t *testing.T) {
	mockService := func() (stoppable.Stoppable, error) {
		return nil, nil
	}

	mockServices := newServices()

	err := mockServices.add(mockService)
	if err != nil {
		t.Fatalf("Failed to add mock service to services: %v", err)
	}

	err = mockServices.start(500 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to start mock services: %v", err)
	}

	// Add a doomed to fail process
	mockServiceErr := func() (stoppable.Stoppable, error) {
		return nil, errors.New("Expected failure case")
	}

	err = mockServices.add(mockServiceErr)
	if err == nil {
		t.Fatalf("Expected error case: " +
			"Service should have started and failed")
	}
}

func TestServices_Start(t *testing.T) {
	mockService := func() (stoppable.Stoppable, error) {
		return nil, nil
	}

	mockServices := newServices()

	err := mockServices.add(mockService)
	if err != nil {
		t.Fatalf("Failed to add mock service to services: %v", err)
	}

	err = mockServices.start(500)
	if err != nil {
		t.Fatalf("Failed to start mock services: %v", err)
	}

	// Try and start again should error
	err = mockServices.start(500 * time.Millisecond)
	if err == nil {
		t.Fatalf("Should fail when calling start with running processes")
	}
}

func TestServices_Stop(t *testing.T) {
	mockService := func() (stoppable.Stoppable, error) {
		return stoppable.NewSingle("test"), nil
	}

	mockServices := newServices()

	err := mockServices.add(mockService)
	if err != nil {
		t.Fatalf("Failed to add mock service to services: %v", err)
	}

	err = mockServices.stop()
	if err == nil {
		t.Fatalf("Should error when calling " +
			"stop on non-running service")
	}

	err = mockServices.start(500 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to start mock services: %v", err)
	}

	err = mockServices.stop()
	if err != nil {
		t.Fatalf("Should not error when calling stop; %v", err)
	}

	err = mockServices.stop()
	if err == nil {
		t.Fatalf("Should error when stopping a stopped service")
	}

}
