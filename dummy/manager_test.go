////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/stoppable"
)

// Tests that newManager returns the expected Manager.
func Test_newManager(t *testing.T) {
	expected := &Manager{
		maxNumMessages: 10,
		avgSendDelta:   time.Minute,
		randomRange:    time.Second,
		status:         notStarted,
		statusChan:     make(chan bool, statusChanLen),
	}

	received := newManager(expected.maxNumMessages, expected.avgSendDelta,
		expected.randomRange, nil, nil, nil)

	if statusChanLen != cap(received.statusChan) {
		t.Errorf("Capacity of status channel unexpected."+
			"\nexpected: %d\nreceived: %d",
			statusChanLen, cap(received.statusChan))
	}
	received.statusChan = expected.statusChan
	received.totalSent = nil

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("New manager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that Manager.StartDummyTraffic sends dummy messages and that it stops
// when the stoppable is closed.
func TestManager_StartDummyTraffic(t *testing.T) {
	m := newTestManager(10, 50*time.Millisecond, 10*time.Millisecond, t)

	err := m.Start()
	if err != nil {
		t.Errorf("Failed to set status to true.")
	}

	stop, err := m.StartDummyTraffic()
	if err != nil {
		t.Errorf("StartDummyTraffic returned an error: %+v", err)
	}

	msgChan := make(chan bool)
	go func() {
		for m.net.(*mockCmix).GetMsgListLen() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		msgChan <- true
	}()

	var numReceived int
	select {
	case <-time.NewTimer(75 * m.avgSendDelta).C:
		t.Errorf("Timed out after %s waiting for messages to be sent.",
			75*m.avgSendDelta)
	case <-msgChan:
	}

	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to close stoppable: %+v", err)
	}

	err = stoppable.WaitForStopped(stop, 5*time.Second)
	if err != nil {
		t.Errorf("Failed to wait for stoppable to be stopped: %+v", err)
	}

	// NOTE: this test was a bit bugged originally, as you can't
	// stop waiting for message received updated, wait for a few
	// lines, then stop the process, then expect not to have ever
	// received a new message.
	// What we do instead is get the len immediately after stopping, then
	// wait for some time to see if it changes
	numReceived = m.net.(*mockCmix).GetMsgListLen()
	if !stop.IsStopped() {
		t.Error("Stoppable never stopped.")
	}

	msgChan = make(chan bool)
	go func() {
		for m.net.(*mockCmix).GetMsgListLen() == numReceived {
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

// Tests that Manager.Pause & Manager.Resume prevents messages from being sent and
// that either may be called multiple times with the same status without it affecting
// the process. Also tests that the thread quits even when paused.
func TestManager_PauseResume(t *testing.T) {
	m := newTestManager(10, 50*time.Millisecond, 10*time.Millisecond, t)

	err := m.Pause()
	if err != nil {
		t.Errorf("Pause returned an error: %+v", err)
	}

	stop := stoppable.NewSingle("sendThreadTest")
	go m.sendThread(stop)

	msgChan := make(chan bool, 10)
	go func() {
		var numReceived int
		for i := 0; i < 2; i++ {
			for m.net.(*mockCmix).GetMsgListLen() == numReceived {
				time.Sleep(5 * time.Millisecond)
			}
			numReceived = m.net.(*mockCmix).GetMsgListLen()
			msgChan <- true
		}
	}()

	time.Sleep(3 * time.Millisecond)
	if stat := atomic.LoadUint32(&m.status); stat != paused {
		t.Errorf("Unexpected thread status.\nexpected: %d\nreceived: %d",
			paused, stat)
	}

	// Setting status to false should cause the messages to not send
	err = m.Pause()
	if err != nil {
		t.Errorf("Pause returned an error: %+v", err)
	}

	var numReceived int
	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:
	case <-msgChan:
		t.Errorf("Should not have received messages when thread was pasued.")
	}

	err = m.Start()
	if err != nil {
		t.Errorf("Resume returned an error: %+v", err)
	}

	time.Sleep(3 * time.Millisecond)
	if stat := atomic.LoadUint32(&m.status); stat != running {
		t.Errorf("Unexpected thread status.\nexpected: %d\nreceived: %d",
			running, stat)
	}

	select {
	case <-time.NewTimer(75 * m.avgSendDelta).C:
		t.Errorf("Timed out after %s waiting for messages to be sent.",
			75*m.avgSendDelta)
	case <-msgChan:
		numReceived += m.net.(*mockCmix).GetMsgListLen()
	}

	// Setting status to true multiple times does not interrupt sending
	for i := 0; i < 3; i++ {
		err = m.Start()
		if err != nil {
			t.Errorf("Resume returned an error (%d): %+v", i, err)
		}
	}

	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:
		t.Errorf("Timed out after %s waiting for messages to be sent.",
			3*m.avgSendDelta)
	case <-msgChan:
		if m.net.(*mockCmix).GetMsgListLen() <= numReceived {
			t.Errorf("Failed to receive second send."+
				"\nmessages on last receive: %d\nmessages on this receive: %d",
				numReceived, m.net.(*mockCmix).GetMsgListLen())
		}
	}

	// Shows that the stoppable still stops when the thread is paused
	err = m.Pause()
	if err != nil {
		t.Errorf("Pause returned an error: %+v", err)
	}
	time.Sleep(3 * time.Millisecond)
	if stat := atomic.LoadUint32(&m.status); stat != paused {
		t.Errorf("Unexpected thread status.\nexpected: %d\nreceived: %d",
			paused, stat)
	}

	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to close stoppable: %+v", err)
	}

	time.Sleep(10 * time.Millisecond)
	if !stop.IsStopped() {
		t.Error("Stoppable never stopped.")
	}
	if stat := atomic.LoadUint32(&m.status); stat != stopped {
		t.Errorf("Unexpected thread status.\nexpected: %d\nreceived: %d",
			stopped, stat)
	}
}

// Error path: tests that Manager.Pause returns an error if the status
// cannot be set.
func TestManager_Pause_ChannelError(t *testing.T) {
	m := newTestManager(10, 50*time.Millisecond, 10*time.Millisecond, t)

	// Send the max number of status changes on the channel
	for i := 0; i < statusChanLen; i++ {
		err := m.Pause()
		if err != nil {
			t.Errorf("Pause returned an error (%d): %+v", i, err)
		}
	}

	// Calling one more time causes an error
	expectedErr := fmt.Sprintf(setStatusErr, true)
	err := m.Start()
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Resume returned unexpected error when channel is full."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

}

// Tests that Manager.GetStatus gets the correct status before the send thread
// starts, while sending, while paused, and after it is stopped.
func TestManager_GetStatus(t *testing.T) {
	m := newTestManager(10, 250*time.Millisecond, 10*time.Millisecond, t)

	require.NoError(t, m.Pause())

	stop := stoppable.NewSingle("sendThreadTest")
	go m.sendThread(stop)

	require.False(t, m.GetStatus(), "GetStatus reported thread as running.")

	msgChan := make(chan bool, 10)
	go func() {
		var numReceived int
		for i := 0; i < 2; i++ {
			for m.net.(*mockCmix).GetMsgListLen() == numReceived {
				time.Sleep(5 * time.Millisecond)
			}
			numReceived = m.net.(*mockCmix).GetMsgListLen()
			msgChan <- true
		}
	}()

	// Setting status to false should cause the messages to not send
	require.NoError(t, m.Pause())

	require.False(t, m.GetStatus(), "GetStatus reported thread as running.")

	var numReceived int
	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:
	case <-msgChan:
		t.Errorf("Should not have received messages when thread was pasued.")
	}

	require.NoError(t, m.Start())

	time.Sleep(3 * time.Millisecond)
	require.True(t, m.GetStatus(), "GetStatus reported thread as paused.")

	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:
		t.Errorf("Timed out after %s waiting for messages to be sent.",
			3*m.avgSendDelta)
	case <-msgChan:
		numReceived += m.net.(*mockCmix).GetMsgListLen()
	}

	// Setting status to true multiple times does not interrupt sending
	for i := 0; i < 3; i++ {
		require.NoError(t, m.Start())
	}

	require.False(t, !m.GetStatus(), "GetStatus reported thread as paused.")

	select {
	case <-time.NewTimer(3 * m.avgSendDelta).C:
		t.Errorf("Timed out after %s waiting for messages to be sent.",
			3*m.avgSendDelta)
	case <-msgChan:
		if m.net.(*mockCmix).GetMsgListLen() <= numReceived {
			t.Errorf("Failed to receive second send."+
				"\nmessages on last receive: %d\nmessages on this receive: %d",
				numReceived, m.net.(*mockCmix).GetMsgListLen())
		}
	}

	// Shows that the stoppable still stops when the thread is paused
	require.NoError(t, m.Pause())

	time.Sleep(3 * time.Millisecond)
	require.False(t, m.GetStatus(), "GetStatus reported thread as running.")

	require.NoError(t, stop.Close())

	time.Sleep(10 * time.Millisecond)
	require.True(t, stop.IsStopped(), "Stoppable never stopped.")

	require.False(t, m.GetStatus(), "GetStatus reported thread as running.")
}
