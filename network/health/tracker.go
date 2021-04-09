///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains functionality related to the event model driven network health tracker

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

	channels []chan bool
	funcs    []func(isHealthy bool)

	running bool

	// Determines the current health status
	isHealthy bool
	// Denotes the past health status
	// wasHealthy is true if isHealthy has ever been true
	wasHealthy bool
	mux        sync.RWMutex
}

// Creates a single HealthTracker thread, starts it, and returns a tracker and a stoppable
func Init(instance *network.Instance, timeout time.Duration) *Tracker {

	tracker := newTracker(timeout)
	instance.SetNetworkHealthChan(tracker.heartbeat)

	return tracker
}

// Builds and returns a new Tracker object given a Context
func newTracker(timeout time.Duration) *Tracker {
	return &Tracker{
		timeout:   timeout,
		channels:  make([]chan bool, 0),
		heartbeat: make(chan network.Heartbeat, 100),
		isHealthy: false,
		running:   false,
	}
}

// Add a channel to the list of Tracker channels
// such that each channel can be notified of network changes
func (t *Tracker) AddChannel(c chan bool) {
	t.mux.Lock()
	t.channels = append(t.channels, c)
	t.mux.Unlock()
	select {
	case c <- t.IsHealthy():
	default:
	}
}

// Add a function to the list of Tracker function
// such that each function can be run after network changes
func (t *Tracker) AddFunc(f func(isHealthy bool)) {
	t.mux.Lock()
	t.funcs = append(t.funcs, f)
	t.mux.Unlock()
	go f(t.IsHealthy())
}

func (t *Tracker) IsHealthy() bool {
	t.mux.RLock()
	defer t.mux.RUnlock()
	return t.isHealthy
}

// Returns true if isHealthy has ever been true
func (t *Tracker) WasHealthy() bool {
	t.mux.RLock()
	defer t.mux.RUnlock()
	return t.wasHealthy
}

func (t *Tracker) setHealth(h bool) {
	t.mux.Lock()
	// Only set wasHealthy to true if either
	//  wasHealthy is true or
	//  wasHealthy false but h value is true
	t.wasHealthy = t.wasHealthy || h
	t.isHealthy = h
	t.mux.Unlock()
	t.transmit(h)
}

func (t *Tracker) Start() (stoppable.Stoppable, error) {
	t.mux.Lock()
	if t.running {
		return nil, errors.New("cannot start Health tracker threads, " +
			"they are already running")
	}
	t.running = true

	t.isHealthy = false
	t.mux.Unlock()

	stop := stoppable.NewSingle("Health Tracker")

	go t.start(stop.Quit())

	return stop, nil
}

// Long-running thread used to monitor and report on network health
func (t *Tracker) start(quitCh <-chan struct{}) {
	timer := time.NewTimer(t.timeout)

	for {
		var heartbeat network.Heartbeat
		select {
		case <-quitCh:
			t.mux.Lock()
			t.isHealthy = false
			t.running = false
			t.mux.Unlock()
			t.transmit(false)
			break
		case heartbeat = <-t.heartbeat:
			if healthy(heartbeat) {
				// Stop and reset timer
				if !timer.Stop() {
					select {
					// per docs explicitly drain
					case <-timer.C:
					default:
					}
				}
				timer.Reset(t.timeout)
				t.setHealth(true)
			}
			break
		case <-timer.C:
			t.setHealth(false)
			break
		}
	}
}

func (t *Tracker) transmit(health bool) {
	for _, c := range t.channels {
		select {
		case c <- health:
		default:
			jww.DEBUG.Printf("Unable to send Health event")
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
