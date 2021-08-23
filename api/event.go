///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"sync"
)

// EventCallbackFunction defines the callback functions for client event reports
type EventCallbackFunction func(priority int, category, evtType, details string)

// ReportableEvent is used to surface events to client users.
type reportableEvent struct {
	Priority  int
	Category  string
	EventType string
	Details   string
}

// Holds state for the event reporting system
type eventManager struct {
	eventCh  chan reportableEvent
	eventCbs []EventCallbackFunction
	eventLck sync.Mutex
}

func newEventManager() eventManager {
	return eventManager{
		eventCh:  make(chan reportableEvent, 1000),
		eventCbs: make([]EventCallbackFunction, 0),
	}
}

// ReportEvent reports an event from the client to api users, providing a
// priority, category, eventType, and details
func (e eventManager) ReportEvent(priority int, category, evtType,
	details string) {
	re := reportableEvent{
		Priority:  priority,
		Category:  category,
		EventType: evtType,
		Details:   details,
	}
	select {
	case e.eventCh <- re:
		jww.TRACE.Printf("Event reported: %s", re)
	default:
		jww.ERROR.Printf("Event Queue full, unable to report: %s", re)
	}
}

// RegisterEventCallback records the given function to receive
// ReportableEvent objects. It returns the internal index
// of the callback so that it can be deleted later.
func (e eventManager) RegisterEventCallback(myFunc EventCallbackFunction) int {
	e.eventLck.Lock()
	defer e.eventLck.Unlock()
	e.eventCbs = append(e.eventCbs, myFunc)
	return len(e.eventCbs) - 1
}

// UnregisterEventCallback deletes the callback identified by the
// index. It returns an error if it fails.
func (e eventManager) UnregisterEventCallback(index int) error {
	e.eventLck.Lock()
	defer e.eventLck.Unlock()
	if index > 0 && index < len(e.eventCbs) {
		e.eventCbs = append(e.eventCbs[:index], e.eventCbs[index+1:]...)
	} else {
		return errors.Errorf("Index %d out of bounds: %d -> %d",
			index, 0, len(e.eventCbs))
	}
	return nil
}

func (e eventManager) eventService() (stoppable.Stoppable, error) {
	stop := stoppable.NewSingle("EventReporting")
	go e.reportEventsHandler(stop)
	return stop, nil
}

// reportEventsHandler reports events to every registered event callback
func (e eventManager) reportEventsHandler(stop *stoppable.Single) {
	jww.DEBUG.Print("reportEventsHandler routine started")
	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Printf("Stopping reportEventsHandler")
			stop.ToStopped()
			return
		case evt := <-e.eventCh:
			jww.DEBUG.Printf("Received event: %s", evt)
			// NOTE: We could call each in a routine but decided
			// against it. It's the users responsibility not to let
			// the event queue explode. The API will report errors
			// in the logging any time the event queue gets full.
			for i := 0; i < len(e.eventCbs); i++ {
				e.eventCbs[i](evt.Priority, evt.Category,
					evt.EventType, evt.Details)
			}
		}
	}
}

// ReportEvent reports an event from the client to api users, providing a
// priority, category, eventType, and details
func (c *Client) ReportEvent(priority int, category, evtType, details string) {
	c.events.ReportEvent(priority, category, evtType, details)
}

// RegisterEventCallback records the given function to receive
// ReportableEvent objects. It returns the internal index
// of the callback so that it can be deleted later.
func (c *Client) RegisterEventCallback(myFunc EventCallbackFunction) int {
	return c.events.RegisterEventCallback(myFunc)
}

// UnregisterEventCallback deletes the callback identified by the
// index. It returns an error if it fails.
func (c *Client) UnregisterEventCallback(index int) error {
	return c.events.UnregisterEventCallback(index)
}

func (e reportableEvent) String() string {
	return fmt.Sprintf("Event(%d, %s, %s, %s)", e.Priority, e.Category,
		e.EventType, e.Details)
}
