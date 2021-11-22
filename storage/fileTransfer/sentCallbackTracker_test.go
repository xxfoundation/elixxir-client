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

// Tests that newSentCallbackTracker returns the expected sentCallbackTracker
// and that the callback triggers correctly.
func Test_newSentCallbackTracker(t *testing.T) {
	type cbFields struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}

	cbChan := make(chan cbFields)
	cbFunc := func(completed bool, sent, arrived, total uint16, err error) {
		cbChan <- cbFields{completed, sent, arrived, total, err}
	}

	expectedSCT := &sentCallbackTracker{
		period:    time.Millisecond,
		lastCall:  time.Time{},
		scheduled: false,
		cb:        cbFunc,
	}

	receivedSCT := newSentCallbackTracker(expectedSCT.cb, expectedSCT.period)

	go receivedSCT.cb(false, 0, 0, 0, nil)

	select {
	case <-time.NewTimer(time.Millisecond).C:
		t.Error("Timed out waiting for callback to be called.")
	case r := <-cbChan:
		err := checkSentProgress(
			r.completed, r.sent, r.arrived, r.total, false, 0, 0, 0)
		if err != nil {
			t.Error(err)
		}
	}

	// Nil the callbacks so that DeepEqual works
	receivedSCT.cb = nil
	expectedSCT.cb = nil

	if !reflect.DeepEqual(expectedSCT, receivedSCT) {
		t.Errorf("New sentCallbackTracker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedSCT, receivedSCT)
	}
}

// Tests that
func Test_sentCallbackTracker_call(t *testing.T) {
	type cbFields struct {
		completed            bool
		sent, arrived, total uint16
		err                  error
	}

	cbChan := make(chan cbFields)
	cbFunc := func(completed bool, sent, arrived, total uint16, err error) {
		cbChan <- cbFields{completed, sent, arrived, total, err}
	}

	sct := newSentCallbackTracker(cbFunc, 50*time.Millisecond)

	tracker := testSentTrack{false, 1, 2, 3}
	sct.call(tracker, nil)

	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for callback to be called.")
	case r := <-cbChan:
		err := checkSentProgress(
			r.completed, r.sent, r.arrived, r.total, false, 1, 2, 3)
		if err != nil {
			t.Error(err)
		}
	}

	tracker = testSentTrack{false, 1, 2, 3}
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
		err := checkSentProgress(
			r.completed, r.sent, r.arrived, r.total, false, 1, 2, 3)
		if err != nil {
			t.Error(err)
		}
	}
}

// Tests that SentTransfer satisfies the sentProgressTracker interface.
func TestSentTransfer_SentProgressTrackerInterface(t *testing.T) {
	var _ sentProgressTracker = &SentTransfer{}
}

// testSentTrack is a test structure that satisfies the sentProgressTracker
// interface.
type testSentTrack struct {
	completed            bool
	sent, arrived, total uint16
}

func (tst testSentTrack) getProgress() (completed bool, sent, arrived, total uint16) {
	return tst.completed, tst.sent, tst.arrived, tst.total
}

// GetProgress returns the values in the testTrack.
func (tst testSentTrack) GetProgress() (completed bool, sent, arrived, total uint16) {
	return tst.completed, tst.sent, tst.arrived, tst.total
}
