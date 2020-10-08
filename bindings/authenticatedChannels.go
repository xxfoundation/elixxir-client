package bindings

import "gitlab.com/elixxir/client/interfaces/bind"

// Create an insecure e2e relationship with a precanned user
func (c *Client) MakePrecannedAuthenticatedChannel(precannedID int) bind.Contact {
	return c.api.MakePrecannedContact(uint(precannedID))
}
