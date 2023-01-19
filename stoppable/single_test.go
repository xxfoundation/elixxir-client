////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"fmt"
	"sync/atomic"
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

	if single.status != Running {
		t.Errorf("NewSingle returned Single with incorrect status."+
			"\nexpected: %s\nreceived: %s", Running, single.status)
	}
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

// Tests that Single.GetStatus returns the expected Status.
func TestSingle_GetStatus(t *testing.T) {
	single := NewSingle("threadName")

	status := single.GetStatus()
	if status != Running {
		t.Errorf("GetStatus returned the wrong status."+
			"\nexpected: %s\nreceived: %s", Running, status)
	}

	atomic.StoreUint32((*uint32)(&single.status), uint32(Stopping))
	status = single.GetStatus()
	if status != Stopping {
		t.Errorf("GetStatus returned the wrong status."+
			"\nexpected: %s\nreceived: %s", Stopping, status)
	}

	atomic.StoreUint32((*uint32)(&single.status), uint32(Stopped))
	status = single.GetStatus()
	if status != Stopped {
		t.Errorf("GetStatus returned the wrong status."+
			"\nexpected: %s\nreceived: %s", Stopped, status)
	}
}

// Tests that Single.IsRunning returns the expected value when the Single is
// marked as running, stopping, and stopped.
func TestSingle_IsRunning(t *testing.T) {
	single := NewSingle("threadName")

	if result := single.IsRunning(); !result {
		t.Errorf("IsRunning returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, result)
	}

	single.status = Stopping
	if result := single.IsRunning(); result {
		t.Errorf("IsRunning returned the wrong value when stopping."+
			"\nexpected: %t\nreceived: %t", false, result)
	}

	single.status = Stopped
	if result := single.IsRunning(); result {
		t.Errorf("IsRunning returned the wrong value when stopped."+
			"\nexpected: %t\nreceived: %t", false, result)
	}
}

// Tests that Single.IsStopping returns the expected value when the Single is
// marked as running, stopping, and stopped.
func TestSingle_IsStopping(t *testing.T) {
	single := NewSingle("threadName")

	if result := single.IsStopping(); result {
		t.Errorf("IsStopping returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, result)
	}

	single.status = Stopping
	if result := single.IsStopping(); !result {
		t.Errorf("IsStopping returned the wrong value when stopping."+
			"\nexpected: %t\nreceived: %t", false, result)
	}

	single.status = Stopped
	if result := single.IsStopping(); result {
		t.Errorf("IsStopping returned the wrong value when stopped."+
			"\nexpected: %t\nreceived: %t", false, result)
	}
}

// Tests that Single.IsStopped returns the expected value when the Single is
// marked as running, stopping, and stopped.
func TestSingle_IsStopped(t *testing.T) {
	single := NewSingle("threadName")

	if result := single.IsStopped(); result {
		t.Errorf("IsStopped returned the wrong value when running."+
			"\nexpected: %t\nreceived: %t", true, result)
	}

	single.status = Stopping
	if result := single.IsStopped(); result {
		t.Errorf("IsStopped returned the wrong value when stopping."+
			"\nexpected: %t\nreceived: %t", false, result)
	}

	single.status = Stopped
	if result := single.IsStopped(); !result {
		t.Errorf("IsStopped returned the wrong value when stopped."+
			"\nexpected: %t\nreceived: %t", false, result)
	}
}

// Tests that Single.toStopping changes the status to stopping.
func TestSingle_toStopping(t *testing.T) {
	single := NewSingle("threadName")

	err := single.toStopping()
	if err != nil {
		t.Errorf("toStopping returned an error: %+v", err)
	}

	if single.status != Stopping {
		t.Errorf("toStopping failed to set the status correctly."+
			"\nexpected: %s\nreceived: %s", Stopping, single.status)
	}
}

// Error path: tests that Single.toStopping returns an error when failing to
// change the status to stopping when the current status is not running.
func TestSingle_toStopping_StatusError(t *testing.T) {
	single := NewSingle("threadName")
	single.status = Stopped
	expectedErr := fmt.Sprintf(
		toStoppingErr, single.Name(), single.GetStatus(), Running)

	err := single.toStopping()
	if err == nil || err.Error() != expectedErr {
		t.Errorf("toStopping failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

	if single.status != Stopped {
		t.Errorf("toStopping changed the status when the compare failed."+
			"\nexpected: %s\nreceived: %s", Stopped, single.status)
	}
}

// Tests that Single.ToStopped changes the status to stopped.
func TestSingle_ToStopped(t *testing.T) {
	single := NewSingle("threadName")

	single.status = Stopping
	single.ToStopped()

	if single.status != Stopped {
		t.Errorf("ToStopped failed to set the status correctly."+
			"\nexpected: %s\nreceived: %s", Stopped, single.status)
	}
}

// Panic path: tests that Single.ToStopped panics when failing to change the
// status to stopped when the current status is not stopping.
func TestSingle_ToStopped_StatusPanic(t *testing.T) {
	single := NewSingle("threadName")

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("ToStopped failed to panic when the status should not " +
				"have changed.")
		} else {
			if single.status != Running {
				t.Errorf("ToStopped changed the status when the compare failed."+
					"\nexpected: %s\nreceived: %s", Running, single.status)
			}
		}
	}()

	single.status = Running
	single.ToStopped()
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

// Test happy path of Single.Close().
func TestSingle_Close(t *testing.T) {
	single := NewSingle("threadName")
	timeout := 10 * time.Millisecond

	go func() {
		select {
		case <-time.NewTimer(timeout).C:
			t.Errorf("Timed out waiting to receive on quit channel after %s.",
				timeout)
		case <-single.Quit():
			if !single.IsStopping() {
				t.Errorf("Status of stoppable incorrect."+
					"\nexpected: %s\nreceived: %s", Stopping, single.status)
			}
			atomic.StoreUint32((*uint32)(&single.status), uint32(Stopped))
		}
	}()

	err := single.Close()
	if err != nil {
		t.Errorf("Close returned an error: %v", err)
	}
}

// Error path: tests that Single.Close returns an error when the status fails
// to change to stopping.
func TestSingle_Close_Error(t *testing.T) {
	single := NewSingle("threadName")
	single.status = Stopped
	expectedErr := fmt.Sprintf(
		toStoppingErr, single.Name(), single.GetStatus(), Running)

	err := single.Close()
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Close did not return the expected error."+
			"\nexpected: %s\nreceived: %v", expectedErr, err)
	}
}
