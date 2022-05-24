////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package old

import (
	"gitlab.com/elixxir/client/dummy"
	"time"
)

// DummyTraffic contains the file dummy traffic manager. The manager can be used
// to set and get the status of the send thread.
type DummyTraffic struct {
	m *dummy.Manager
}

// NewDummyTrafficManager creates a DummyTraffic manager and initialises the
// dummy traffic send thread. Note that the manager does not start sending dummy
// traffic until its status is set to true using DummyTraffic.SetStatus.
// The maxNumMessages is the upper bound of the random number of messages sent
// each send. avgSendDeltaMS is the average duration, in milliseconds, to wait
// between sends. Sends occur every avgSendDeltaMS +/- a random duration with an
// upper bound of randomRangeMS.
func NewDummyTrafficManager(client *Client, maxNumMessages, avgSendDeltaMS,
	randomRangeMS int) (*DummyTraffic, error) {

	avgSendDelta := time.Duration(avgSendDeltaMS) * time.Millisecond
	randomRange := time.Duration(randomRangeMS) * time.Millisecond

	m := dummy.NewManager(
		maxNumMessages, avgSendDelta, randomRange, &client.api)

	return &DummyTraffic{m}, client.api.AddService(m.StartDummyTraffic)
}

// SetStatus sets the state of the dummy traffic send thread, which determines
// if the thread is running or paused. The possible statuses are:
//  true  = send thread is sending dummy messages
//  false = send thread is paused/stopped and not sending dummy messages
// Returns an error if the channel is full.
// Note that this function cannot change the status of the send thread if it has
// yet to be started or stopped.
func (dt *DummyTraffic) SetStatus(status bool) error {
	return dt.m.SetStatus(status)
}

// GetStatus returns the current state of the dummy traffic send thread. It has
// the following return values:
//  true  = send thread is sending dummy messages
//  false = send thread is paused/stopped and not sending dummy messages
// Note that this function does not return the status set by SetStatus directly;
// it returns the current status of the send thread, which means any call to
// SetStatus will have a small delay before it is returned by GetStatus.
func (dt *DummyTraffic) GetStatus() bool {
	return dt.m.GetStatus()
}
