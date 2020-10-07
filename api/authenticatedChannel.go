package api

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/storage/e2e"
)

// CreateAuthenticatedChannel creates a 1-way authenticated channel
// so this user can send messages to the desired recipient Contact.
// To receive confirmation from the remote user, clients must
// register a listener to do that.
func (c *Client) CreateAuthenticatedChannel(recipient contact.Contact,
	payload []byte) error {
	jww.INFO.Printf("CreateAuthenticatedChannel(%v, %v)",
		recipient, payload)
	return nil
}

// RegisterAuthConfirmationCb registers a callback for channel
// authentication confirmation events.
func (c *Client) RegisterAuthConfirmationCb(cb func(contact contact.Contact,
	payload []byte)) {
	jww.INFO.Printf("RegisterAuthConfirmationCb(...)")
}

// RegisterAuthRequestCb registers a callback for channel
// authentication request events.
func (c *Client) RegisterAuthRequestCb(cb func(contact contact.Contact,
	payload []byte)) {
	jww.INFO.Printf("RegisterAuthRequestCb(...)")
}

// Create an insecure e2e relationship with a precanned user
func (c *Client) MakePrecannedAuthenticatedChannel(precannedID uint) contact.Contact {

	precan := c.MakePrecannedContact(precannedID)

	//add the precanned user as a e2e contact
	sesParam := e2e.GetDefaultSessionParams()
	c.storage.E2e().AddPartner(precan.ID, precan.DhPubKey, sesParam, sesParam)

	return precan
}

// Create an insecure e2e contact object for a precanned user
func (c *Client) MakePrecannedContact(precannedID uint) contact.Contact {

	e2eGrp := c.storage.E2e().GetGroup()

	//get the user definition
	precanned := createPrecannedUser(precannedID, c.rng.GetStream(),
		c.storage.Cmix().GetGroup(), e2eGrp)

	//compute their public e2e key
	partnerPubKey := e2eGrp.ExpG(precanned.E2eDhPrivateKey, e2eGrp.NewInt(1))

	return contact.Contact{
		ID:             precanned.ID,
		DhPubKey:       partnerPubKey,
		OwnershipProof: nil,
		Facts:          make([]contact.Fact, 0),
	}
}