////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/v4/dummy"
	"time"
)

// DummyTraffic is the bindings-layer dummy (or "cover") traffic manager. T
// The manager can be used to set and get the status of the thread responsible for
// sending dummy messages.
type DummyTraffic struct {
	m *dummy.Manager
}

// NewDummyTrafficManager creates a DummyTraffic manager and initialises the
// dummy traffic sending thread. Note that the manager is by default paused,
// and as such the sending thread must be started by calling DummyTraffic.Start.
// The time duration between each sending operation and the amount of messages
// sent each interval are randomly generated values with bounds defined by the
// given parameters below.
//
// Parameters:
//  - cmixId - a Cmix object ID in the tracker.
//  - maxNumMessages - the upper bound of the random number of messages sent
//    each sending cycle.  Suggested value: 5.
//  - avgSendDeltaMS - the average duration, in milliseconds, to wait between
//    sends.  Suggested value: 60000.
//  - randomRangeMS - the upper bound of the interval between sending cycles, in
//    milliseconds. Sends occur every avgSendDeltaMS +/- a random duration with
//    an upper bound of randomRangeMS.  Suggested value: 1000.
func NewDummyTrafficManager(cmixId, maxNumMessages, avgSendDeltaMS,
	randomRangeMS int) (*DummyTraffic, error) {

	// Get user from singleton
	net, err := cmixTrackerSingleton.get(cmixId)
	if err != nil {
		return nil, err
	}

	avgSendDelta := time.Duration(avgSendDeltaMS) * time.Millisecond
	randomRange := time.Duration(randomRangeMS) * time.Millisecond

	m := dummy.NewManager(
		maxNumMessages, avgSendDelta, randomRange, net.api)

	return &DummyTraffic{m}, net.api.AddService(m.StartDummyTraffic)
}

// Pause will pause the Manager's sending thread, meaning messages will no
// longer be sent. After calling Pause, the sending thread may only be resumed
// by calling Resume.
//
// There may be a small delay between this call and the pause taking effect.
// This is because Pause will not cancel the thread when it is in the process
// of sending messages, but will instead wait for that thread to complete. The
// thread will then be prevented from beginning another round of sending.
func (dt *DummyTraffic) Pause() error {
	return dt.m.Pause()
}

// Resume will resume the Manager's sending thread, meaning messages will
// continue to be sent. This should typically be called only if the thread
// has been paused by calling Pause previously.
//
// This will re-initialize the sending thread with a new randomly generated
// interval between sending dummy messages. This means that there is zero
// guarantee that the sending interval prior to pausing will be the same
// sending interval after a call to Resume.
func (dt *DummyTraffic) Resume() error {
	return dt.m.Resume()
}

// Start will start the sending thread. This is meant to be called after
// NewDummyTrafficManager.
//
// This will initialize the sending thread with a randomly generated interval
// in between sending dummy messages.
func (dt *DummyTraffic) Start() error {
	return dt.m.Start()
}

// GetStatus returns the current state of the DummyTraffic manager's sending
// thread. Note that the status returned here may lag behind a user's earlier
// call to pause the sending thread. This is a result of a small delay (see
// DummyTraffic.Pause for more details)
//
// Returns:
//   - bool - Returns true (dummy.Running) if the sending thread is sending
//     messages and false (dummy.Paused) if the sending thread is not sending
//     messages.
func (dt *DummyTraffic) GetStatus() bool {
	return dt.m.GetStatus()
}
