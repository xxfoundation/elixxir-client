////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/dummy"
	"time"
)

// DummyTraffic is the bindings-layer dummy (or "cover") traffic manager. T
// The manager can be used to set and get the status of the thread responsible for
// sending dummy messages.
type DummyTraffic struct {
	m *dummy.Manager
}

// NewDummyTrafficManager creates a DummyTraffic manager and initialises the
// dummy traffic sending thread. Note that the manager does not start sending dummy
// traffic until `True` is passed into DummyTraffic.SetStatus. The time duration
// between each sending operation and the amount of messages sent each interval
// are randomly generated values with bounds defined by the
// given parameters below.
//
// Params:
//  - cmixId - a Cmix object ID in the tracker.
//  - maxNumMessages - the upper bound of the random number of messages sent
//    each sending cycle.
//  - avgSendDeltaMS - the average duration, in milliseconds, to wait
//    between sends.
//  - randomRangeMS - the upper bound of the interval between sending cycles,
//    in milliseconds. Sends occur every avgSendDeltaMS +/- a random duration
//    with an upper bound of randomRangeMS.
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

// SetStatus sets the state of the DummyTraffic manager's send thread by passing in
// a boolean parameter. There may be a small delay in between this call
// and the status of the sending thread to change accordingly. For example,
// passing False into this call while the sending thread is currently sending messages
// will not cancel nor halt the sending operation, but will pause the thread once that
// operation has completed.
//
// Params:
//  - boolean - Input should be true if you want to send dummy messages.
//  			Input should be false if you want to pause dummy messages.
// Returns:
//  - error - if the DummyTraffic.SetStatus is called too frequently, causing the
//    internal status channel to fill.
func (dt *DummyTraffic) SetStatus(status bool) error {
	return dt.m.SetStatus(status)
}

// GetStatus returns the current state of the DummyTraffic manager's sending thread.
// Note that this function does not return the status set by the most recent call to
// SetStatus. Instead, this call returns the current status of the sending thread.
// This is due to the small delay that may occur between calling SetStatus and the
// sending thread taking into effect that status change.
//
// Returns:
//   - boolean - Returns true if sending thread is sending dummy messages.
//  	         Returns false if sending thread is paused/stopped and is
// 	             not sending dummy messages.
func (dt *DummyTraffic) GetStatus() bool {
	return dt.m.GetStatus()
}
