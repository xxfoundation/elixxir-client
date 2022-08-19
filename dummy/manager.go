////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

// Package dummy allows for the sending of dummy messages to dummy recipients
// via SendCmix at randomly generated intervals.

package dummy

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/fastRNG"
	"sync/atomic"
	"time"
)

// Manager related thread handling constants.
const (
	// The name of the Manager's stoppable.Stoppable
	dummyTrafficStoppableName = "DummyTraffic"

	// The amount of statuses in queue that can be placed
	// by Manager.SetStatus.
	statusChanLen = 100
)

// The thread status values.
const (
	notStarted uint32 = iota // Sending thread has not been started
	running                  // Sending thread is currently operating
	paused                   // Sending thread is temporarily halted.
	stopped                  // Sending thread is halted.
)

// Error messages for Manager.
const (
	setStatusErr = "Failed to change status of dummy traffic send thread to %t: channel full"
)

// Manager manages the sending of dummy messages.
type Manager struct {
	// The maximum number of messages to send each send
	maxNumMessages int

	// Average duration to wait between message sends
	avgSendDelta time.Duration

	// Upper limit for random duration that modified avgSendDelta
	randomRange time.Duration

	// Indicates the current status of the thread (0 = paused, 1 = running)
	status uint32

	// Pauses/Resumes the dummy send thread when triggered
	statusChan chan bool

	// Interfaces
	net   cmix.Client
	store storage.Session

	// Generates
	rng *fastRNG.StreamGenerator
}

// NewManager creates a Manager object and initialises the
// dummy traffic sending thread. Note that the Manager does not start sending dummy
// traffic until `True` is passed into Manager.SetStatus. The time duration
// between each sending operation and the amount of messages sent each interval
// are randomly generated values with bounds defined by the
// given parameters below.
//
// Params:
//  - maxNumMessages - the upper bound of the random number of messages sent
//    each sending cycle.
//  - avgSendDeltaMS - the average duration, in milliseconds, to wait
//    between sends.
//  - randomRangeMS - the upper bound of the interval between sending cycles,
//    in milliseconds. Sends occur every avgSendDeltaMS +/- a random duration
//    with an upper bound of randomRangeMS.
func NewManager(maxNumMessages int,
	avgSendDelta, randomRange time.Duration,
	net *xxdk.Cmix) *Manager {

	return newManager(maxNumMessages, avgSendDelta, randomRange, net.GetCmix(),
		net.GetStorage(), net.GetRng())
}

// newManager builds a new dummy Manager from fields explicitly passed in. This
// function is a helper function for NewManager.
func newManager(maxNumMessages int, avgSendDelta, randomRange time.Duration,
	net cmix.Client, store storage.Session, rng *fastRNG.StreamGenerator) *Manager {
	return &Manager{
		maxNumMessages: maxNumMessages,
		avgSendDelta:   avgSendDelta,
		randomRange:    randomRange,
		status:         notStarted,
		statusChan:     make(chan bool, statusChanLen),
		net:            net,
		store:          store,
		rng:            rng,
	}
}

// StartDummyTraffic starts the process of sending dummy traffic. This function
// adheres to xxdk.Service.
func (m *Manager) StartDummyTraffic() (stoppable.Stoppable, error) {
	stop := stoppable.NewSingle(dummyTrafficStoppableName)
	go m.sendThread(stop)

	return stop, nil
}

// SetStatus sets the state of the dummy traffic send thread by passing in
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
//  - error - if the Manager.SetStatus is called too frequently, causing the
//    internal status channel to fill.
func (m *Manager) SetStatus(status bool) error {
	select {
	case m.statusChan <- status:
		return nil
	default:
		return errors.Errorf(setStatusErr, status)
	}
}

// GetStatus returns the current state of the Manager's sending thread.
// Note that this function does not return the status set by the most recent call to
// SetStatus. Instead, this call returns the current status of the sending thread.
// This is due to the small delay that may occur between calling SetStatus and the
// sending thread taking into effect that status change.
//
// Returns:
//   - boolean - Returns true if sending thread is sending dummy messages.
//  	         Returns false if sending thread is paused/stopped and is
// 	             not sending dummy messages.
func (m *Manager) GetStatus() bool {
	switch atomic.LoadUint32(&m.status) {
	case running:
		return true
	default:
		return false
	}
}
