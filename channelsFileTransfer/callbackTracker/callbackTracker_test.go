////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package callbackTracker

import (
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/elixxir/client/v4/stoppable"
)

// Tests that newCallbackTracker returns a new callbackTracker with all the
// expected values.
func Test_newCallbackTracker(t *testing.T) {
	expected := &callbackTracker{
		period:    time.Millisecond,
		lastCall:  time.Time{},
		scheduled: false,
		complete:  false,
		stop:      stoppable.NewSingle("Test_newCallbackTracker"),
	}

	newCT := newCallbackTracker(nil, expected.period, expected.stop)
	newCT.cb = nil

	if !reflect.DeepEqual(expected, newCT) {
		t.Errorf("New callbackTracker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, newCT)
	}
}

// Tests four test cases of callbackTracker.call:
//  1. An initial call is not scheduled.
//  2. A second call within the periods is only called after the period.
//  3. An error sets the callback to complete.
//  4. No more callbacks will be called after set to complete.
func Test_callbackTracker_call(t *testing.T) {
	cbChan := make(chan error, 10)
	cb := func(err error) { cbChan <- err }
	stop := stoppable.NewSingle("Test_callbackTracker_call")
	ct := newCallbackTracker(cb, 250*time.Millisecond, stop)

	// Test that the initial call is unscheduled and is called before the period
	go ct.call(nil)

	select {
	case r := <-cbChan:
		if r != nil {
			t.Errorf("Received error: %+v", r)
		}
	case <-time.After(35 * time.Millisecond):
		t.Error("Timed out waiting for callback.")
	}

	// Test that another call within the period is called only after the period
	// is reached
	go ct.call(nil)

	select {
	case <-cbChan:
		t.Error("Callback called too soon.")

	case <-time.After(35 * time.Millisecond):
		ct.mux.RLock()
		if !ct.scheduled {
			t.Error("Callback is not scheduled when it should be.")
		}
		ct.mux.RUnlock()
		select {
		case r := <-cbChan:
			if r != nil {
				t.Errorf("Received error: %+v", r)
			}
		case <-time.After(ct.period):
			t.Errorf("Callback not called after period %s.", ct.period)

			if ct.scheduled {
				t.Error("Callback is scheduled when it should not be.")
			}
		}
	}

	// Test that calling with an error sets the callback to complete
	expectedErr := errors.New("test error")
	go ct.call(expectedErr)

	select {
	case r := <-cbChan:
		if r != expectedErr {
			t.Errorf("Received incorrect error.\nexpected: %v\nreceived: %v",
				expectedErr, r)
		}
		if !ct.complete {
			t.Error("Callback is not marked complete when it should be.")
		}
	case <-time.After(ct.period + 25*time.Millisecond):
		t.Errorf("Callback not called after period %s.",
			ct.period+15*time.Millisecond)
	}

	// Tests that all callback calls after an error are blocked
	go ct.call(nil)

	select {
	case r := <-cbChan:
		t.Errorf("Received callback when it should have been completed: %+v", r)
	case <-time.After(ct.period):
	}
}

// TODO: fix test. It was disabled since stop() now calls all remaining callbacks before closing.
// // Tests that callbackTracker.call does not call on the callback when the
// // stoppable is triggered.
// func Test_callbackTracker_call_stop(t *testing.T) {
// 	cbChan := make(chan error, 10)
// 	cb := func(err error) { cbChan <- err }
// 	stop := stoppable.NewSingle("Test_callbackTracker_call")
// 	ct := newCallbackTracker(cb, 250*time.Millisecond, stop)
//
// 	go ct.call(nil)
//
// 	select {
// 	case r := <-cbChan:
// 		if r != nil {
// 			t.Errorf("Received error: %+v", r)
// 		}
// 	case <-time.After(25 * time.Millisecond):
// 		t.Error("Timed out waiting for callback.")
// 	}
//
// 	go ct.call(nil)
//
// 	err := stop.Close()
// 	if err != nil {
// 		t.Errorf("Failed closing stoppable: %+v", err)
// 	}
//
// 	select {
// 	case <-cbChan:
// 		t.Error("Callback called.")
// 	case <-time.After(ct.period * 2):
// 	}
// }
