////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"reflect"
	"testing"
	"time"
)

// Tests that newManager returns the expected Manager.
func Test_newManager(t *testing.T) {
	expected := &Manager{
		maxNumMessages: 10,
		avgSendDelta:   time.Minute,
		randomRange:    time.Second,
	}

	received := newManager(expected.maxNumMessages, expected.avgSendDelta,
		expected.randomRange, nil, nil, nil, nil)

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("New manager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that Manager.StartDummyTraffic sends dummy messages and that it stops
// when the stoppable is closed.
func TestManager_StartDummyTraffic(t *testing.T) {
	m := newTestManager(10, 50*time.Millisecond, 10*time.Millisecond, false, t)

	stop, err := m.StartDummyTraffic()
	if err != nil {
		t.Errorf("StartDummyTraffic returned an error: %+v", err)
	}

	msgChan := make(chan bool)
	go func() {
		for m.net.(*testNetworkManager).GetMsgListLen() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		msgChan <- true
	}()

	var numReceived int
	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:
		t.Errorf("Timed out after %s waiting for messages to be sent.",
			3*m.avgSendDelta)
	case <-msgChan:
		numReceived += m.net.(*testNetworkManager).GetMsgListLen()
	}

	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to close stoppable: %+v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if !stop.IsStopped() {
		t.Error("Stoppable never stopped.")
	}

	msgChan = make(chan bool)
	go func() {
		for m.net.(*testNetworkManager).GetMsgListLen() == numReceived {
			time.Sleep(5 * time.Millisecond)
		}
		msgChan <- true
	}()

	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:

	case <-msgChan:
		t.Error("Received new messages after stoppable was stopped.")
	}
}
