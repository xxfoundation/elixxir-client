////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"gitlab.com/elixxir/client/v4/event"
)

// ReportEvent reports an event from the client to api users, providing a
// priority, category, eventType, and details.
func (c *Cmix) ReportEvent(priority int, category, evtType, details string) {
	c.events.Report(priority, category, evtType, details)
}

// RegisterEventCallback records the given function to receive ReportableEvent
// objects.
func (c *Cmix) RegisterEventCallback(name string, myFunc event.Callback) error {
	return c.events.RegisterEventCallback(name, myFunc)
}

// UnregisterEventCallback deletes the callback identified by the name.
func (c *Cmix) UnregisterEventCallback(name string) {
	c.events.UnregisterEventCallback(name)
}
