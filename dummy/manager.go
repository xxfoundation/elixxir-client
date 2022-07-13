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
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/fastRNG"
	"sync/atomic"
	"time"
)

const (
	dummyTrafficStoppableName = "DummyTraffic"
	statusChanLen             = 100
)

// Thread status.
const (
	notStarted uint32 = iota
	running
	paused
	stopped
)

// Error messages.
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

	// Cmix interfaces
	net            *xxdk.Cmix
	store          *storage.Session
	networkManager interfaces.NetworkManager
	rng            *fastRNG.StreamGenerator
}

// NewManager creates a new dummy Manager with the specified average send delta
// and the range used for generating random durations.
func NewManager(maxNumMessages int, avgSendDelta, randomRange time.Duration,
	net *xxdk.Cmix, manager interfaces.NetworkManager) *Manager {
	clientStorage := net.GetStorage()
	return newManager(maxNumMessages, avgSendDelta, randomRange, net,
		&clientStorage, manager, net.GetRng())
}

// newManager builds a new dummy Manager from fields explicitly passed in. This
// function is a helper function for NewManager to make it easier to test.
func newManager(maxNumMessages int, avgSendDelta, randomRange time.Duration,
	net *xxdk.Cmix, store *storage.Session, networkManager interfaces.NetworkManager,
	rng *fastRNG.StreamGenerator) *Manager {
	return &Manager{
		maxNumMessages: maxNumMessages,
		avgSendDelta:   avgSendDelta,
		randomRange:    randomRange,
		status:         notStarted,
		statusChan:     make(chan bool, statusChanLen),
		net:            net,
		store:          store,
		networkManager: networkManager,
		rng:            rng,
	}
}

// StartDummyTraffic starts the process of sending dummy traffic. This function
// matches the xxdk.Service type.
func (m *Manager) StartDummyTraffic() (stoppable.Stoppable, error) {
	stop := stoppable.NewSingle(dummyTrafficStoppableName)
	go m.sendThread(stop)

	return stop, nil
}

// SetStatus sets the state of the dummy traffic send thread, which determines
// if the thread is running or paused. The possible statuses are:
//  true  = send thread is sending dummy messages
//  false = send thread is paused/stopped and not sending dummy messages
// Returns an error if the channel is full.
// Note that this function cannot change the status of the send thread if it has
// yet to be started via StartDummyTraffic or if it has been stopped.
func (m *Manager) SetStatus(status bool) error {
	select {
	case m.statusChan <- status:
		return nil
	default:
		return errors.Errorf(setStatusErr, status)
	}
}

// GetStatus returns the current state of the dummy traffic send thread. It has
// the following return values:
//  true  = send thread is sending dummy messages
//  false = send thread is paused/stopped and not sending dummy messages
// Note that this function does not return the status set by SetStatus directly;
// it returns the current status of the send thread, which means any call to
// SetStatus will have a small delay before it is returned by GetStatus.
func (m *Manager) GetStatus() bool {
	switch atomic.LoadUint32(&m.status) {
	case running:
		return true
	default:
		return false
	}
}
