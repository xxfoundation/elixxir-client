////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package health

import (
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/elixxir/comms/network"
)

// Happy path smoke test.
func Test_newTracker(t *testing.T) {
	// Initialize required variables
	timeout := 800 * time.Millisecond
	trkr := newTracker(timeout)
	counter := int64(2) // First signal is "false/unhealthy"
	positiveHb := network.Heartbeat{
		HasWaitingRound: true,
		IsRoundComplete: true,
	}

	// Build listening channel and listening function
	listenChan := make(chan bool, 10)
	listenFunc := func(isHealthy bool) {
		if isHealthy {
			atomic.AddInt64(&counter, 1)
		} else {
			atomic.AddInt64(&counter, -1)
		}
	}
	trkr.AddHealthCallback(listenFunc)
	trkr.AddHealthCallback(listenFunc)
	go func() {
		for isHealthy := range listenChan {
			if isHealthy {
				atomic.AddInt64(&counter, 1)
			} else {
				atomic.AddInt64(&counter, -1)
			}
		}
	}()

	// Begin the health tracker
	_, err := trkr.StartProcesses()
	if err != nil {
		t.Fatalf("Unable to start tracker: %+v", err)
	}

	// Send a positive health heartbeat
	expectedCount := int64(2)
	trkr.heartbeat <- positiveHb

	// Wait for the heartbeat to register
	for i := 0; i < 4; i++ {
		if trkr.IsHealthy() && atomic.LoadInt64(&counter) == expectedCount {
			break
		} else {
			time.Sleep(100 * time.Millisecond)
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
	if atomic.LoadInt64(&counter) != expectedCount {
		t.Errorf("Expected counter to be %d, got %d",
			expectedCount, atomic.LoadInt64(&counter))
	}

	expectedCount = 0
	counterChan := make(chan int64)
	go func() {
		c := atomic.LoadInt64(&counter)
		for ; c != expectedCount; c = atomic.LoadInt64(&counter) {
			time.Sleep(50 * time.Millisecond)
		}
		counterChan <- c
	}()

	// Wait out the timeout
	select {
	case c := <-counterChan:
		// Verify the network was marked as NOT healthy
		if trkr.IsHealthy() {
			t.Fatal("tracker should not report healthy.")
		}

		// Check if the tracker was ever healthy, after setting healthy to false
		if !trkr.WasHealthy() {
			t.Fatal("tracker was healthy previously but not reported healthy.")
		}

		// Verify the timeout triggered the listening chan/func
		if c != expectedCount {
			t.Errorf("Expected counter to be %d, got %d", expectedCount, c)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("Timed out waiting for counter to be expected value."+
			"\nexpected: %d\nreceived: %d",
			expectedCount, atomic.LoadInt64(&counter))
	}
}
