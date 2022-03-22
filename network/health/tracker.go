///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains functionality related to the event model driven network health
// tracker.

package health

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"sync"
	"time"
)

type Tracker struct {
	timeout time.Duration

	heartbeat chan network.Heartbeat

	channels   map[uint64]chan bool
	funcs      map[uint64]func(isHealthy bool)
	channelsID uint64
	funcsID    uint64

	running bool

	// Determines the current health status
	isHealthy bool

	// Denotes that the past health status wasHealthy is true if isHealthy has
	// ever been true
	wasHealthy bool
	mux        sync.RWMutex
}

// Init creates a single HealthTracker thread, starts it, and returns a tracker
// and a stoppable.
func Init(instance *network.Instance, timeout time.Duration) *Tracker {
	tracker := newTracker(timeout)
	instance.SetNetworkHealthChan(tracker.heartbeat)

	return tracker
}

// newTracker builds and returns a new Tracker object given a Context.
func newTracker(timeout time.Duration) *Tracker {
	return &Tracker{
		timeout:   timeout,
		channels:  map[uint64]chan bool{},
		funcs:     map[uint64]func(isHealthy bool){},
		heartbeat: make(chan network.Heartbeat, 100),
		isHealthy: false,
		running:   false,
	}
}

// AddChannel adds a channel to the list of Tracker channels such that each
// channel can be notified of network changes.  Returns a unique ID for the
// channel.
func (t *Tracker) AddChannel(c chan bool) uint64 {
	var currentID uint64

	t.mux.Lock()
	t.channels[t.channelsID] = c
	currentID = t.channelsID
	t.channelsID++
	t.mux.Unlock()

	select {
	case c <- t.IsHealthy():
	default:
	}

	return currentID
}

// RemoveChannel removes the channel with the given ID from the list of Tracker
// channels so that it will not longer be notified of network changes.
func (t *Tracker) RemoveChannel(chanID uint64) {
	t.mux.Lock()
	delete(t.channels, chanID)
	t.mux.Unlock()
}

// AddFunc adds a function to the list of Tracker functions such that each
// function can be run after network changes. Returns a unique ID for the
// function.
func (t *Tracker) AddFunc(f func(isHealthy bool)) uint64 {
	var currentID uint64

	t.mux.Lock()
	t.funcs[t.funcsID] = f
	currentID = t.funcsID
	t.funcsID++
	t.mux.Unlock()

	go f(t.IsHealthy())

	return currentID
}

// RemoveFunc removes the function with the given ID from the list of Tracker
// functions so that it will not longer be run.
func (t *Tracker) RemoveFunc(chanID uint64) {
	t.mux.Lock()
	delete(t.channels, chanID)
	t.mux.Unlock()
}

func (t *Tracker) IsHealthy() bool {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.isHealthy
}

// WasHealthy returns true if isHealthy has ever been true.
func (t *Tracker) WasHealthy() bool {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.wasHealthy
}

func (t *Tracker) setHealth(h bool) {
	t.mux.Lock()
	// Only set wasHealthy to true if either
	//  wasHealthy is true or
	//  wasHealthy is false but h value is true
	t.wasHealthy = t.wasHealthy || h
	t.isHealthy = h
	t.mux.Unlock()

	t.transmit(h)
}

func (t *Tracker) Start() (stoppable.Stoppable, error) {
	t.mux.Lock()
	if t.running {
		t.mux.Unlock()
		return nil, errors.New("cannot start health tracker threads, " +
			"they are already running")
	}
	t.running = true

	t.isHealthy = false
	t.mux.Unlock()

	stop := stoppable.NewSingle("health Tracker")

	go t.start(stop)

	return stop, nil
}

// start starts a long-running thread used to monitor and report on network
// health.
func (t *Tracker) start(stop *stoppable.Single) {
	for {
		var heartbeat network.Heartbeat
		select {
		case <-stop.Quit():
			t.mux.Lock()
			t.isHealthy = false
			t.running = false
			t.mux.Unlock()

			t.transmit(false)
			stop.ToStopped()

			return
		case heartbeat = <-t.heartbeat:
			// FIXME: There's no transition to unhealthy here
			// and there needs to be after some number of bad
			// polls
			if healthy(heartbeat) {
				t.setHealth(true)
			}
		case <-time.After(t.timeout):
			if !t.isHealthy {
				jww.WARN.Printf("Network health tracker timed out, network is no longer healthy...")
			}
			t.setHealth(false)
		}
	}
}

func (t *Tracker) transmit(health bool) {
	for _, c := range t.channels {
		select {
		case c <- health:
		default:
			jww.DEBUG.Printf("Unable to send health event")
		}
	}

	// Run all listening functions
	for _, f := range t.funcs {
		go f(health)
	}
}

func healthy(a network.Heartbeat) bool {
	return a.IsRoundComplete
}
