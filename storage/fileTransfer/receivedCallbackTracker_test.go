////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"gitlab.com/elixxir/client/interfaces"
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
	cbFunc := func(completed bool, received, total uint16,
		t interfaces.FilePartTracker, err error) {
		cbChan <- cbFields{completed, received, total, err}
	}

	expectedRCT := &receivedCallbackTracker{
		period:    time.Millisecond,
		lastCall:  time.Time{},
		scheduled: false,
		cb:        cbFunc,
	}

	receivedRCT := newReceivedCallbackTracker(expectedRCT.cb, expectedRCT.period)

	go receivedRCT.cb(false, 0, 0, nil, nil)

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
	receivedRCT.cb = nil
	expectedRCT.cb = nil

	receivedRCT.stop = expectedRCT.stop

	if !reflect.DeepEqual(expectedRCT, receivedRCT) {
		t.Errorf("New receivedCallbackTracker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedRCT, receivedRCT)
	}
}

// Tests that receivedCallbackTracker.call calls the tracker immediately when
// no other calls are scheduled and that it schedules a call to the tracker when
// one has been called recently.
func Test_receivedCallbackTracker_call(t *testing.T) {
	type cbFields struct {
		completed       bool
		received, total uint16
		err             error
	}

	cbChan := make(chan cbFields)
	cbFunc := func(completed bool, received, total uint16,
		t interfaces.FilePartTracker, err error) {
		cbChan <- cbFields{completed, received, total, err}
	}

	rct := newReceivedCallbackTracker(cbFunc, 50*time.Millisecond)

	tracker := testReceiveTrack{false, 1, 3, receivedPartTracker{}}
	rct.call(tracker, nil)

	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for callback to be called.")
	case r := <-cbChan:
		err := checkReceivedProgress(r.completed, r.received, r.total, false, 1, 3)
		if err != nil {
			t.Error(err)
		}
	}

	tracker = testReceiveTrack{true, 3, 3, receivedPartTracker{}}
	rct.call(tracker, nil)

	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		if !rct.scheduled {
			t.Error("Callback should be scheduled.")
		}
	case r := <-cbChan:
		t.Errorf("Received message when period of %s should not have been "+
			"reached: %+v", rct.period, r)
	}

	rct.call(tracker, nil)

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

// Tests that receivedCallbackTracker.stopThread prevents a scheduled call to
// the tracker from occurring.
func Test_receivedCallbackTracker_stopThread(t *testing.T) {
	type cbFields struct {
		completed       bool
		received, total uint16
		err             error
	}

	cbChan := make(chan cbFields)
	cbFunc := func(completed bool, received, total uint16,
		t interfaces.FilePartTracker, err error) {
		cbChan <- cbFields{completed, received, total, err}
	}

	rct := newReceivedCallbackTracker(cbFunc, 50*time.Millisecond)

	tracker := testReceiveTrack{false, 1, 3, receivedPartTracker{}}
	rct.call(tracker, nil)

	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for callback to be called.")
	case r := <-cbChan:
		err := checkReceivedProgress(r.completed, r.received, r.total, false, 1, 3)
		if err != nil {
			t.Error(err)
		}
	}

	tracker = testReceiveTrack{true, 3, 3, receivedPartTracker{}}
	rct.call(tracker, nil)

	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		if !rct.scheduled {
			t.Error("Callback should be scheduled.")
		}
	case r := <-cbChan:
		t.Errorf("Received message when period of %s should not have been "+
			"reached: %+v", rct.period, r)
	}

	rct.call(tracker, nil)

	err := rct.stopThread()
	if err != nil {
		t.Errorf("stopThread returned an error: %+v", err)
	}

	select {
	case <-time.NewTimer(60 * time.Millisecond).C:
	case r := <-cbChan:
		t.Errorf("Received message when period of %s should not have been "+
			"reached: %+v", rct.period, r)
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
	t               receivedPartTracker
}

func (trt testReceiveTrack) getProgress() (completed bool, received,
	total uint16, t interfaces.FilePartTracker) {
	return trt.completed, trt.received, trt.total, trt.t
}

// GetProgress returns the values in the testTrack.
func (trt testReceiveTrack) GetProgress() (completed bool, received,
	total uint16, t interfaces.FilePartTracker) {
	return trt.completed, trt.received, trt.total, trt.t
}
