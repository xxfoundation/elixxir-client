////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

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

	isHealthy bool
	mux       sync.RWMutex
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

func (t *Tracker) setHealth(h bool) {
	t.mux.Lock()
	t.isHealthy = h
	t.mux.Unlock()
	t.transmit(h)
}

func (t *Tracker) Start() (stoppable.Stoppable, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	if t.running {
		return nil, errors.New("cannot start Health tracker threads, " +
			"they are already running")
	}
	t.running = true

	t.isHealthy = false

	stop := stoppable.NewSingle("Health Tracker")
	stopCleanup := stoppable.NewCleanup(stop, func(duration time.Duration) error {
		t.mux.Lock()
		defer t.mux.Unlock()
		t.isHealthy = false
		t.transmit(false)
		t.running = false
		return nil
	})

	go t.start(stop.Quit())

	return stopCleanup, nil
}

// Long-running thread used to monitor and report on network health
func (t *Tracker) start(quitCh <-chan struct{}) {

	var timerChan <-chan time.Time
	timerChan = make(chan time.Time)

	for {
		var heartbeat network.Heartbeat
		select {
		case <-quitCh:
			// Handle thread kill
			break
		case heartbeat = <-t.heartbeat:
			jww.INFO.Printf("heartbeat: %+v", heartbeat)
			if healthy(heartbeat) {
				timerChan = time.NewTimer(t.timeout).C
				t.setHealth(true)
			}
		case <-timerChan:
			t.setHealth(false)
		}
	}
}

func (t *Tracker) transmit(health bool) {
	for _, c := range t.channels {
		select {
		case c <- health:
		default:
			jww.WARN.Printf("Unable to send Health event")
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
