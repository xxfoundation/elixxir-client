////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package event

import (
	"testing"
	"time"
)

func TestEventReporting(t *testing.T) {
	evts := make([]reportableEvent, 0) // used for convenience...
	myCb := func(priority int, cat, ty, det string) {
		evt := reportableEvent{
			Priority:  priority,
			Category:  cat,
			EventType: ty,
			Details:   det,
		}
		t.Logf("EVENT: %s", evt)
		evts = append(evts, evt)
	}

	evtMgr := NewEventManager()
	stop, _ := evtMgr.EventService()
	// Register a callback
	err := evtMgr.RegisterEventCallback("test", myCb)
	if err != nil {
		t.Errorf("TestEventReporting unexpected error: %+v", err)
	}

	// Send a few events
	evtMgr.Report(10, "Hi", "TypityType", "I'm an event")
	evtMgr.Report(1, "Hi", "TypeII", "Tag II errors are the worst")
	evtMgr.Report(20, "Hi", "TypityType3", "eventy details")
	evtMgr.Report(22, "Hi", "TypityType4", "I'm an event 2")

	time.Sleep(100 * time.Millisecond)

	c := make(chan struct{})
	go func() {
		for len(evts) != 4 {
			time.Sleep(20 * time.Millisecond)
		}
		c <- struct{}{}
	}()

	select {
	case <-c:
	case <-time.After(3 * time.Second):
		t.Errorf("TestEventReporting: Got %d events, expected %d",
			len(evts), 4)
	}

	// Verify events are received
	if evts[0].Priority != 10 {
		t.Errorf("TestEventReporting: Expected priority 10, got: %s",
			evts[0])
	}
	if evts[1].Category != "Hi" {
		t.Errorf("TestEventReporting: Expected cat Hi, got: %s",
			evts[1])
	}
	if evts[2].EventType != "TypityType3" {
		t.Errorf("TestEventReporting: Expected TypeityType3, got: %s",
			evts[2])
	}
	if evts[3].Details != "I'm an event 2" {
		t.Errorf("TestEventReporting: Expected event 2, got: %s",
			evts[3])
	}

	// Delete callback
	evtMgr.UnregisterEventCallback("test")
	// Send more events
	evtMgr.Report(10, "Hi", "TypityType", "I'm an event")
	evtMgr.Report(1, "Hi", "TypeII", "Tag II errors are the worst")
	evtMgr.Report(20, "Hi", "TypityType3", "eventy details")
	evtMgr.Report(22, "Hi", "TypityType4", "I'm an event 2")

	time.Sleep(100 * time.Millisecond)

	// Verify events are not received
	c = make(chan struct{})
	go func() {
		for len(evts) != 4 {
			time.Sleep(20 * time.Millisecond)
		}
		c <- struct{}{}
	}()

	select {
	case <-c:
	case <-time.After(3 * time.Second):
		t.Errorf("TestEventReporting: Got %d events, expected %d",
			len(evts), 4)
	}

	if err = stop.Close(); err != nil {
		t.Errorf("Failed to close stoppable: %+v", err)
	}
}
