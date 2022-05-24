///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

// EventCallbackFunctionObject bindings interface which contains function
// that implements the EventCallbackFunction
type EventCallbackFunctionObject interface {
	ReportEvent(priority int, category, evtType, details string)
}

// RegisterEventCallback records the given function to receive
// ReportableEvent objects. It returns the internal index
// of the callback so that it can be deleted later.
func (c *Client) RegisterEventCallback(name string,
	myObj EventCallbackFunctionObject) error {
	return c.api.RegisterEventCallback(name, myObj.ReportEvent)
}

// UnregisterEventCallback deletes the callback identified by the
// index. It returns an error if it fails.
func (c *Client) UnregisterEventCallback(name string) {
	c.api.UnregisterEventCallback(name)
}
