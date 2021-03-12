///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package health

import (
	"gitlab.com/elixxir/comms/network"
	//	"gitlab.com/elixxir/comms/network"
	"testing"
	"time"
)

// Happy path smoke test
func TestNewTracker(t *testing.T) {
	// Initialize required variables
	timeout := 250 * time.Millisecond
	tracker := newTracker(timeout)
	counter := 2 // First signal is "false/unhealthy"
	positiveHb := network.Heartbeat{
		HasWaitingRound: true,
		IsRoundComplete: true,
	}

	// Build listening channel and listening function
	listenChan := make(chan bool, 10)
	listenFunc := func(isHealthy bool) {
		if isHealthy {
			counter++
		} else {
			counter--
		}
	}
	tracker.AddChannel(listenChan)
	tracker.AddFunc(listenFunc)
	go func() {
		for isHealthy := range listenChan {
			if isHealthy {
				counter++
			} else {
				counter--
			}
		}
	}()

	// Begin the health tracker
	_, err := tracker.Start()
	if err != nil {
		t.Errorf("Unable to start tracker: %+v", err)
		return
	}

	// Send a positive health heartbeat
	expectedCount := 2
	tracker.heartbeat <- positiveHb

	// Wait for the heartbeat to register
	for i := 0; i < 4; i++ {
		if tracker.IsHealthy() && counter == expectedCount {
			break
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Verify the network was marked as healthy
	if !tracker.IsHealthy() {
		t.Errorf("Tracker did not become healthy")
		return
	}

	// Check if the tracker was ever healthy
	if !tracker.WasHealthy() {
		t.Errorf("Tracker did not become healthy")
		return
	}

	// Verify the heartbeat triggered the listening chan/func
	if counter != expectedCount {
		t.Errorf("Expected counter to be %d, got %d", expectedCount, counter)
	}

	// Wait out the timeout
	expectedCount = 0
	time.Sleep(timeout)

	// Verify the network was marked as NOT healthy
	if tracker.IsHealthy() {
		t.Errorf("Tracker should not report healthy")
		return
	}

	// Check if the tracker was ever healthy,
	// after setting healthy to false
	if !tracker.WasHealthy() {
		t.Errorf("Tracker was healthy previously but not reported healthy")
		return
	}

	// Verify the timeout triggered the listening chan/func
	if counter != expectedCount {
		t.Errorf("Expected counter to be %d, got %d", expectedCount, counter)
	}
}
