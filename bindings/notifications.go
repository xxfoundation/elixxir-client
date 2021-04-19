///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

// Register for notifications, accepts firebase messaging token
func (c *Client) RegisterForNotifications(token []byte) error {
	return c.api.RegisterForNotifications(token)
}

// Unregister for notifications
func (c *Client) UnregisterForNotifications() error {
	return c.api.UnregisterForNotifications()
}
