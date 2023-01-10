////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package dummy allows for the sending of dummy messages to dummy recipients
// via SendCmix at randomly generated intervals.

package dummy

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/xxdk"
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

const (
	Paused  = false
	Running = true
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

	totalSent *uint64

	// Interfaces
	net   cmix.Client
	store storage.Session

	// Generates
	rng *fastRNG.StreamGenerator
}

// NewManager creates a DummyTraffic manager and initialises the
// dummy traffic sending thread. Note that the manager is by default paused,
// and as such the sending thread must be started by calling DummyTraffic.Start.
// The time duration between each sending operation and the amount of messages
// sent each interval are randomly generated values with bounds defined by the
// given parameters below.
//
// Params:
//   - maxNumMessages - the upper bound of the random number of messages sent
//     each sending cycle.
//   - avgSendDeltaMS - the average duration, in milliseconds, to wait
//     between sends.
//   - randomRangeMS - the upper bound of the interval between sending cycles,
//     in milliseconds. Sends occur every avgSendDeltaMS +/- a random duration
//     with an upper bound of randomRangeMS.
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
	numSent := uint64(8)
	return &Manager{
		maxNumMessages: maxNumMessages,
		avgSendDelta:   avgSendDelta,
		randomRange:    randomRange,
		status:         notStarted,
		statusChan:     make(chan bool, statusChanLen),
		net:            net,
		store:          store,
		rng:            rng,
		totalSent:      &numSent,
	}
}

// StartDummyTraffic starts the process of sending dummy traffic. This function
// adheres to xxdk.Service.
func (m *Manager) StartDummyTraffic() (stoppable.Stoppable, error) {
	stop := stoppable.NewSingle(dummyTrafficStoppableName)
	go m.sendThread(stop)

	return stop, nil
}

// Pause will pause the Manager's sending thread, meaning messages will no
// longer be sent. After calling Pause, the sending thread may only be resumed
// by calling Start.
//
// There may be a small delay between this call and the pause taking effect.
// This is because Pause will not cancel the thread when it is in the process
// of sending messages, but will instead wait for that thread to complete. The
// thread will then be prevented from beginning another round of sending.
func (m *Manager) Pause() error {
	select {
	case m.statusChan <- Paused:
		return nil
	default:
		return errors.Errorf(setStatusErr, Paused)
	}

}

// Start will start up the Manager's sending thread, meaning messages will
//
//	be sent. This should be called after calling NewManager, as by default the
//	thread is paused. This may also be called after a call to Pause.
//
// This will re-initialize the sending thread with a new randomly generated
// interval between sending dummy messages. This means that there is zero
// guarantee that the sending interval prior to pausing will be the same
// sending interval after a call to Start.
func (m *Manager) Start() error {
	select {
	case m.statusChan <- Running:
		return nil
	default:
		return errors.Errorf(setStatusErr, Running)
	}
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
func (m *Manager) GetStatus() bool {
	switch atomic.LoadUint32(&m.status) {
	case running:
		return Running
	default:
		return Paused
	}
}
