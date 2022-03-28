///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package health

import (
	"gitlab.com/elixxir/comms/network"
	"testing"
	"time"
)

// Happy path smoke test.
func TestNewTracker(t *testing.T) {
	// Initialize required variables
	timeout := 250 * time.Millisecond
	trkr := newTracker(timeout)
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
	trkr.AddHealthCallback(listenFunc)
	trkr.AddHealthCallback(listenFunc)
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
	_, err := trkr.StartProcessies()
	if err != nil {
		t.Fatalf("Unable to start tracker: %+v", err)
	}

	// Send a positive health heartbeat
	expectedCount := 2
	trkr.heartbeat <- positiveHb

	// Wait for the heartbeat to register
	for i := 0; i < 4; i++ {
		if trkr.IsHealthy() && counter == expectedCount {
			break
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Verify the network was marked as healthy
	if !trkr.IsHealthy() {
		t.Fatal("tracker did not become healthy.")
	}

	// Check if the tracker was ever healthy
	if !trkr.WasHealthy() {
		t.Fatal("tracker did not become healthy.")
	}

	// Verify the heartbeat triggered the listening chan/func
	if counter != expectedCount {
		t.Errorf("Expected counter to be %d, got %d", expectedCount, counter)
	}

	// Wait out the timeout
	expectedCount = 0
	time.Sleep(timeout)

	// Verify the network was marked as NOT healthy
	if trkr.IsHealthy() {
		t.Fatal("tracker should not report healthy.")
	}

	// Check if the tracker was ever healthy, after setting healthy to false
	if !trkr.WasHealthy() {
		t.Fatal("tracker was healthy previously but not reported healthy.")
	}

	// Verify the timeout triggered the listening chan/func
	if counter != expectedCount {
		t.Errorf("Expected counter to be %d, got %d", expectedCount, counter)
	}
}
