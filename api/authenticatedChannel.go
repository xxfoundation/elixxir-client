package api

import jww "github.com/spf13/jwalterweatherman"

// CreateAuthenticatedChannel creates a 1-way authenticated channel
// so this user can send messages to the desired recipient Contact.
// To receive confirmation from the remote user, clients must
// register a listener to do that.
func (c *Client) CreateAuthenticatedChannel(recipient Contact,
	payload []byte) error {
	jww.INFO.Printf("CreateAuthenticatedChannel(%v, %v)",
		recipient, payload)
	return nil
}

// RegisterAuthConfirmationCb registers a callback for channel
// authentication confirmation events.
func (c *Client) RegisterAuthConfirmationCb(cb func(contact Contact,
	payload []byte)) {
	jww.INFO.Printf("RegisterAuthConfirmationCb(...)")
}

// RegisterAuthRequestCb registers a callback for channel
// authentication request events.
func (c *Client) RegisterAuthRequestCb(cb func(contact Contact,
	payload []byte)) {
	jww.INFO.Printf("RegisterAuthRequestCb(...)")
}
