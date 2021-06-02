///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"fmt"
	"testing"
	"time"
)

// Tests that NewSingle returns a Single with the correct name and running.
func TestNewSingle(t *testing.T) {
	name := "threadName"
	single := NewSingle(name)

	if single.name != name {
		t.Errorf("NewSingle returned Single with incorrect name."+
			"\nexpected: %s\nreceived: %s", name, single.name)
	}

	if single.running != running {
		t.Errorf("NewSingle returned Single with incorrect running."+
			"\nexpected: %d\nreceived: %d", running, single.running)
	}
}

// Tests that Single.IsRunning returns the expected value when the Single is
// marked as both running and not running.
func TestSingle_IsRunning(t *testing.T) {
	single := NewSingle("threadName")

	if !single.IsRunning() {
		t.Errorf("IsRunning returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, single.IsRunning())
	}

	single.running = stopped
	if single.IsRunning() {
		t.Errorf("IsRunning returned the wrong value when not running."+
			"\nexpected: %t\nreceived: %t", false, single.IsRunning())
	}
}

// Tests that Single.Quit returns a channel that is triggered when the Single
// quit channel is triggered.
func TestSingle_Quit(t *testing.T) {
	single := NewSingle("threadName")

	go func() {
		select {
		case <-time.NewTimer(5 * time.Millisecond).C:
			t.Error("Timed out waiting for quit channel.")
		case <-single.Quit():
		}
	}()

	single.quit <- struct{}{}
}

// Unit test of Single.Name.
func TestSingle_Name(t *testing.T) {
	name := "threadName"
	single := NewSingle(name)

	if name != single.Name() {
		t.Errorf("Name did not return the expected name."+
			"\nexpected: %s\nreceived: %s", name, single.Name())
	}
}

// Test happy path of Single.Close().
func TestSingle_Close(t *testing.T) {
	single := NewSingle("threadName")

	go func() {
		select {
		case <-time.NewTimer(10 * time.Millisecond).C:
			t.Error("Timed out waiting for quit channel.")
		case <-single.Quit():
		}
	}()

	err := single.Close(5 * time.Millisecond)
	if err != nil {
		t.Errorf("Close returned an error: %v", err)
	}
}

// Error path: tests that Single.Close returns an error when the timeout is
// reached.
func TestSingle_Close_Error(t *testing.T) {
	single := NewSingle("threadName")
	timeout := time.Millisecond
	expectedErr := fmt.Sprintf(closeTimeoutErr, single.Name(), timeout)

	go func() {
		time.Sleep(5 * time.Millisecond)
		<-single.Quit()
	}()

	err := single.Close(timeout)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Close did not return the expected error."+
			"\nexpected: %s\nreceived: %v", expectedErr, err)
	}
}
