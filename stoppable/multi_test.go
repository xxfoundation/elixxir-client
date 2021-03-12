///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"reflect"
	"testing"
	"time"
)

// Tests happy path of NewMulti().
func TestNewMulti(t *testing.T) {
	name := "test name"
	multi := NewMulti(name)

	if multi.name != name || multi.running != 1 {
		t.Errorf("NewMulti() returned Multi with incorrect values."+
			"\n\texpected:  name: %s  running: %d\n\treceived:  name: %s  running: %d",
			name, 1, multi.name, multi.running)
	}
}

// Tests happy path of Multi.IsRunning().
func TestMulti_IsRunning(t *testing.T) {
	multi := NewMulti("name")

	if !multi.IsRunning() {
		t.Errorf("IsRunning() returned false when it should be running.")
	}

	multi.running = 0
	if multi.IsRunning() {
		t.Errorf("IsRunning() returned true when it should not be running.")
	}
}

// Tests happy path of Multi.Add().
func TestMulti_Add(t *testing.T) {
	multi := NewMulti("multi name")
	singles := []*Single{
		NewSingle("single name 1"),
		NewSingle("single name 2"),
		NewSingle("single name 3"),
	}

	for _, single := range singles {
		multi.Add(single)
	}

	for i, single := range singles {
		if !reflect.DeepEqual(single, multi.stoppables[i]) {
			t.Errorf("Add() did not add the correct Stoppables."+
				"\n\texpected: %#v\n\treceived: %#v", single, multi.stoppables[i])
		}
	}
}

// Tests happy path of Multi.Name().
func TestMulti_Name(t *testing.T) {
	name := "test name"
	multi := NewMulti(name)
	singles := []*Single{
		NewSingle("single name 1"),
		NewSingle("single name 2"),
		NewSingle("single name 3"),
	}
	expectedNames := []string{
		name + ": {}",
		name + ": {" + singles[0].name + "}",
		name + ": {" + singles[0].name + ", " + singles[1].name + "}",
		name + ": {" + singles[0].name + ", " + singles[1].name + ", " + singles[2].name + "}",
	}

	for i, single := range singles {
		if expectedNames[i] != multi.Name() {
			t.Errorf("Name() returned the incorrect string."+
				"\n\texpected: %s\n\treceived: %s", expectedNames[0], multi.Name())
		}
		multi.Add(single)
	}
}

// Tests happy path of Multi.Close().
func TestMulti_Close(t *testing.T) {
	// Create new Multi and add Singles to it
	multi := NewMulti("name")
	singles := []*Single{
		NewSingle("single name 1"),
		NewSingle("single name 2"),
		NewSingle("single name 3"),
	}
	for _, single := range singles {
		multi.Add(single)
	}

	go func() {
		select {
		case <-singles[0].quit:
		}
		select {
		case <-singles[1].quit:
		}
		select {
		case <-singles[2].quit:
		}
	}()

	err := multi.Close(5 * time.Millisecond)
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}

	err = multi.Close(0)
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
}
