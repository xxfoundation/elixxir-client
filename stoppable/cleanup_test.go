///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"testing"
)

// Tests that NewCleanup returns a Cleanup that is stopped with the given
// Stoppable.
func TestNewCleanup(t *testing.T) {
	single := NewSingle("testSingle")
	cleanup := NewCleanup(single, single.Close)

	if cleanup.stop != single {
		t.Errorf("NewCleanup returned cleanup with incorrect Stoppable."+
			"\nexpected: %+v\nreceived: %+v", single, cleanup.stop)
	}

	if cleanup.running != stopped {
		t.Errorf("NewMulti returned Multi with incorrect running."+
			"\nexpected: %d\nreceived: %d", stopped, cleanup.running)
	}
}

// Tests that Cleanup.IsRunning returns the expected value when the Cleanup is
// marked as both running and not running.
func TestCleanup_IsRunning(t *testing.T) {
	single := NewSingle("threadName")
	cleanup := NewCleanup(single, single.Close)

	if cleanup.IsRunning() {
		t.Errorf("IsRunning returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, cleanup.IsRunning())
	}

	cleanup.running = running
	if !single.IsRunning() {
		t.Errorf("IsRunning returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", false, single.IsRunning())
	}
}

// Unit test of Cleanup.Name.
func TestCleanup_Name(t *testing.T) {
	name := "threadName"
	single := NewSingle(name)
	cleanup := NewCleanup(single, single.Close)

	if name+nameTag != cleanup.Name() {
		t.Errorf("Name did not return the expected name."+
			"\nexpected: %s\nreceived: %s", name+nameTag, cleanup.Name())
	}
}

// Tests happy path of Cleanup.Close().
func TestCleanup_Close(t *testing.T) {
	single := NewSingle("threadName")
	cleanup := NewCleanup(single, single.Close)

	// go func() {
	// 	select {
	// 	case <-time.NewTimer(10 * time.Millisecond).C:
	// 		t.Error("Timed out waiting for quit channel.")
	// 	case <-single.Quit():
	// 	}
	// }()

	err := cleanup.Close(0)
	if err != nil {
		t.Errorf("Close() returned an error: %+v", err)
	}
}
