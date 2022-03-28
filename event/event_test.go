///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

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

	evtMgr := newEventManager()
	stop, _ := evtMgr.eventService()
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

	if len(evts) != 4 {
		t.Errorf("TestEventReporting: Got %d events, expected 4",
			len(evts))
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
	if len(evts) != 4 {
		t.Errorf("TestEventReporting: Got %d events, expected 4",
			len(evts))
	}
	stop.Close()
}
