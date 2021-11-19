////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"reflect"
	"testing"
	"time"
)

// Tests that newReceivedCallbackTracker returns the expected
// receivedCallbackTracker and that the callback triggers correctly.
func Test_newReceivedCallbackTracker(t *testing.T) {
	type cbFields struct {
		completed       bool
		received, total uint16
		err             error
	}

	cbChan := make(chan cbFields)
	cbFunc := func(completed bool, received, total uint16, err error) {
		cbChan <- cbFields{completed, received, total, err}
	}

	expectedRCT := &receivedCallbackTracker{
		period:    time.Millisecond,
		lastCall:  time.Time{},
		scheduled: false,
		cb:        cbFunc,
	}

	receivedSCT := newReceivedCallbackTracker(expectedRCT.cb, expectedRCT.period)

	go receivedSCT.cb(false, 0, 0, nil)

	select {
	case <-time.NewTimer(time.Millisecond).C:
		t.Error("Timed out waiting for callback to be called.")
	case r := <-cbChan:
		err := checkReceivedProgress(r.completed, r.received, r.total, false, 0, 0)
		if err != nil {
			t.Error(err)
		}
	}

	// Nil the callbacks so that DeepEqual works
	receivedSCT.cb = nil
	expectedRCT.cb = nil

	if !reflect.DeepEqual(expectedRCT, receivedSCT) {
		t.Errorf("New receivedCallbackTracker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedRCT, receivedSCT)
	}
}

// Tests that
func Test_receivedCallbackTracker_call(t *testing.T) {
	type cbFields struct {
		completed       bool
		received, total uint16
		err             error
	}

	cbChan := make(chan cbFields)
	cbFunc := func(completed bool, received, total uint16, err error) {
		cbChan <- cbFields{completed, received, total, err}
	}

	sct := newReceivedCallbackTracker(cbFunc, 50*time.Millisecond)

	tracker := testReceiveTrack{false, 1, 3}
	sct.call(tracker, nil)

	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for callback to be called.")
	case r := <-cbChan:
		err := checkReceivedProgress(r.completed, r.received, r.total, false, 1, 3)
		if err != nil {
			t.Error(err)
		}
	}

	tracker = testReceiveTrack{true, 3, 3}
	sct.call(tracker, nil)

	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		if !sct.scheduled {
			t.Error("Callback should be scheduled.")
		}
	case r := <-cbChan:
		t.Errorf("Received message when period of %s should not have been "+
			"reached: %+v", sct.period, r)
	}

	sct.call(tracker, nil)

	select {
	case <-time.NewTimer(60 * time.Millisecond).C:
		t.Error("Timed out waiting for callback to be called.")
	case r := <-cbChan:
		err := checkReceivedProgress(r.completed, r.received, r.total, true, 3, 3)
		if err != nil {
			t.Error(err)
		}
	}
}

// Tests that ReceivedTransfer satisfies the receivedProgressTracker interface.
func TestReceivedTransfer_ReceivedProgressTrackerInterface(t *testing.T) {
	var _ receivedProgressTracker = &ReceivedTransfer{}
}

// testReceiveTrack is a test structure that satisfies the
// receivedProgressTracker interface.
type testReceiveTrack struct {
	completed       bool
	received, total uint16
}

// GetProgress returns the values in the testTrack.
func (trt testReceiveTrack) GetProgress() (completed bool, received, total uint16) {
	return trt.completed, trt.received, trt.total
}
