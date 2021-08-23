///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

// RegisterEventCallback records the given function to receive
// ReportableEvent objects. It returns the internal index
// of the callback so that it can be deleted later.
func (c *Client) RegisterEventCallback(myFunc EventCallbackFunction) int {
	return c.api.RegisterEventCallback(myFunc)
}

// UnregisterEventCallback deletes the callback identified by the
// index. It returns an error if it fails.
func (c *Client) UnregisterEventCallback(index int) error {
	return c.api.UnregisterEventCallback(index)
}
