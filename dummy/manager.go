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
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/fastRNG"
	"time"
)

const (
	dummyTrafficStoppableName = "DummyTraffic"
)

// Manager manages the sending of dummy messages.
type Manager struct {
	// The maximum number of messages to send each send
	maxNumMessages int

	// Average duration to wait between message sends
	avgSendDelta time.Duration

	// Upper limit for random duration that modified avgSendDelta
	randomRange time.Duration

	// Client interfaces
	client *api.Client
	store  *storage.Session
	net    interfaces.NetworkManager
	rng    *fastRNG.StreamGenerator
}

// NewManager creates a new dummy Manager with the specified average send delta
// and the range used for generating random durations.
func NewManager(maxNumMessages int, avgSendDelta, randomRange time.Duration,
	client *api.Client) *Manager {
	return newManager(maxNumMessages, avgSendDelta, randomRange, client,
		client.GetStorage(), client.GetNetworkInterface(), client.GetRng())
}

// newManager builds a new dummy Manager from fields explicitly passed in. This
// function is a helper function for NewManager to make it easier to test.
func newManager(maxNumMessages int, avgSendDelta, randomRange time.Duration,
	client *api.Client, store *storage.Session, net interfaces.NetworkManager,
	rng *fastRNG.StreamGenerator) *Manager {
	return &Manager{
		maxNumMessages: maxNumMessages,
		avgSendDelta:   avgSendDelta,
		randomRange:    randomRange,
		client:         client,
		store:          store,
		net:            net,
		rng:            rng,
	}
}

// StartDummyTraffic starts the process of sending dummy traffic. This function
// matches the api.Service type.
func (m *Manager) StartDummyTraffic() (stoppable.Stoppable, error) {
	stop := stoppable.NewSingle(dummyTrafficStoppableName)
	go m.sendThread(stop)

	return stop, nil
}
