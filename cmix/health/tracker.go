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
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
)

type Monitor interface {
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
	IsHealthy() bool
	WasHealthy() bool
	StartProcesses() (stoppable.Stoppable, error)
}

type tracker struct {
	timeout time.Duration

	heartbeat chan network.Heartbeat

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
func Init(instance *network.Instance, timeout time.Duration) Monitor {
	tracker := newTracker(timeout)
	instance.SetNetworkHealthChan(tracker.heartbeat)

	return tracker
}

// newTracker builds and returns a new tracker object given a Context.
func newTracker(timeout time.Duration) *tracker {
	return &tracker{
		timeout:   timeout,
		funcs:     map[uint64]func(isHealthy bool){},
		heartbeat: make(chan network.Heartbeat, 100),
		isHealthy: false,
		running:   false,
	}
}

// AddHealthCallback adds a function to the list of tracker functions such that
// each function can be run after network changes. Returns a unique ID for the
// function.
func (t *tracker) AddHealthCallback(f func(isHealthy bool)) uint64 {
	var currentID uint64

	t.mux.Lock()
	t.funcs[t.funcsID] = f
	currentID = t.funcsID
	t.funcsID++
	t.mux.Unlock()

	go f(t.IsHealthy())

	return currentID
}

// RemoveHealthCallback removes the function with the given ID from the list of
// tracker functions so that it will no longer be run.
func (t *tracker) RemoveHealthCallback(chanID uint64) {
	t.mux.Lock()
	delete(t.funcs, chanID)
	t.mux.Unlock()
}

func (t *tracker) IsHealthy() bool {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.isHealthy
}

// WasHealthy returns true if isHealthy has ever been true.
func (t *tracker) WasHealthy() bool {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.wasHealthy
}

func (t *tracker) setHealth(h bool) {
	t.mux.Lock()
	// Only set wasHealthy to true if either
	//  wasHealthy is true or
	//  wasHealthy is false but h value is true
	t.wasHealthy = t.wasHealthy || h
	t.isHealthy = h
	t.mux.Unlock()

	t.transmit(h)
}

func (t *tracker) StartProcesses() (stoppable.Stoppable, error) {
	t.mux.Lock()
	if t.running {
		t.mux.Unlock()
		return nil, errors.New(
			"cannot start health tracker threads, they are already running")
	}
	t.running = true

	t.isHealthy = false
	t.mux.Unlock()

	stop := stoppable.NewSingle("health tracker")

	go t.start(stop)

	return stop, nil
}

// start starts a long-running thread used to monitor and report on network
// health.
func (t *tracker) start(stop *stoppable.Single) {
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
			// FIXME: There's no transition to unhealthy here and there needs to
			//  be after some number of bad polls
			if healthy(heartbeat) {
				t.setHealth(true)
			}
		case <-time.After(t.timeout):
			if !t.isHealthy {
				jww.WARN.Printf("Network health tracker timed out, network " +
					"is no longer healthy...")
			}
			t.setHealth(false)
		}
	}
}

func (t *tracker) transmit(health bool) {
	// Run all listening functions
	for _, f := range t.funcs {
		go f(health)
	}
}

func healthy(a network.Heartbeat) bool {
	return a.IsRoundComplete
}
