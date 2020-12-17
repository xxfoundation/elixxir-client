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

// Tests happy path of NewCleanup().
func TestNewCleanup(t *testing.T) {
	single := NewSingle("test name")
	cleanup := NewCleanup(single, single.Close)

	if cleanup.stop != single || cleanup.running != 0 {
		t.Errorf("NewCleanup() returned Single with incorrect values."+
			"\n\texpected:  stop: %v  running: %d\n\treceived:  stop: %v  running: %d",
			single, cleanup.stop, 0, cleanup.running)
	}
}

// Tests happy path of Cleanup.IsRunning().
func TestCleanup_IsRunning(t *testing.T) {
	single := NewSingle("test name")
	cleanup := NewCleanup(single, single.Close)

	if cleanup.IsRunning() {
		t.Errorf("IsRunning() returned false when it should be running.")
	}

	cleanup.running = 1
	if !cleanup.IsRunning() {
		t.Errorf("IsRunning() returned true when it should not be running.")
	}
}

// Tests happy path of Cleanup.Name().
func TestCleanup_Name(t *testing.T) {
	name := "test name"
	single := NewSingle(name)
	cleanup := NewCleanup(single, single.Close)

	if name+" with cleanup" != cleanup.Name() {
		t.Errorf("Name() returned the incorrect string."+
			"\n\texpected: %s\n\treceived: %s", name+" with cleanup", cleanup.Name())
	}
}

// Tests happy path of Cleanup.Close().
func TestCleanup_Close(t *testing.T) {
	single := NewSingle("test name")
	cleanup := NewCleanup(single, single.Close)

	err := cleanup.Close(0)
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
}
