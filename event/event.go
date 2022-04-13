///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package event

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
)

// ReportableEvent is used to surface events to client users.
type reportableEvent struct {
	Priority  int
	Category  string
	EventType string
	Details   string
}

// String stringer interace implementation
func (e reportableEvent) String() string {
	return fmt.Sprintf("Event(%d, %s, %s, %s)", e.Priority, e.Category,
		e.EventType, e.Details)
}

// Holds state for the event reporting system
type eventManager struct {
	eventCh  chan reportableEvent
	eventCbs sync.Map
}

func NewEventManager() Manager {
	return &eventManager{
		eventCh: make(chan reportableEvent, 1000),
	}
}

// Report reports an event from the client to api users, providing a
// priority, category, eventType, and details
func (e *eventManager) Report(priority int, category, evtType, details string) {
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
func (e *eventManager) RegisterEventCallback(name string,
	myFunc Callback) error {
	_, existsAlready := e.eventCbs.LoadOrStore(name, myFunc)
	if existsAlready {
		return errors.Errorf("Key %s already exists as event callback",
			name)
	}
	return nil
}

// UnregisterEventCallback deletes the callback identified by the
// index. It returns an error if it fails.
func (e *eventManager) UnregisterEventCallback(name string) {
	e.eventCbs.Delete(name)
}

func (e *eventManager) EventService() (stoppable.Stoppable, error) {
	stop := stoppable.NewSingle("EventReporting")
	go e.reportEventsHandler(stop)
	return stop, nil
}

// reportEventsHandler reports events to every registered event callback
func (e *eventManager) reportEventsHandler(stop *stoppable.Single) {
	jww.DEBUG.Print("reportEventsHandler routine started")
	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Printf("Stopping reportEventsHandler")
			stop.ToStopped()
			return
		case evt := <-e.eventCh:
			jww.TRACE.Printf("Received event: %s", evt)
			// NOTE: We could call each in a routine but decided
			// against it. It's the users responsibility not to let
			// the event queue explode. The API will report errors
			// in the logging any time the event queue gets full.
			e.eventCbs.Range(func(name, myFunc interface{}) bool {
				f := myFunc.(Callback)
				f(evt.Priority, evt.Category, evt.EventType,
					evt.Details)
				return true
			})
		}
	}
}
