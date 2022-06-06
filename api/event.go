package api

import (
	"gitlab.com/elixxir/client/event"
)

// ReportEvent reports an event from the client to api users, providing a
// priority, category, eventType, and details
func (c *Client) ReportEvent(priority int, category, evtType, details string) {
	c.events.Report(priority, category, evtType, details)
}

// RegisterEventCallback records the given function to receive
// ReportableEvent objects. It returns the internal index
// of the callback so that it can be deleted later.
func (c *Client) RegisterEventCallback(name string,
	myFunc event.Callback) error {
	return c.events.RegisterEventCallback(name, myFunc)
}

// UnregisterEventCallback deletes the callback identified by the
// index. It returns an error if it fails.
func (c *Client) UnregisterEventCallback(name string) {
	c.events.UnregisterEventCallback(name)
}
