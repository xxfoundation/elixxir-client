package api

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/xx_network/primitives/id"
)


// RequestAuthenticatedChannel sends a request to another party to establish an
// authenticated channel
// It will not run if the network status is not healthy
// An error will be returned if a channel already exists, if a request was
// already received, or if a request was already sent
// When a confirmation occurs, the channel will be created and the callback
// will be called
func (c *Client) RequestAuthenticatedChannel(recipient, me contact.Contact,
	message string) error {
	jww.INFO.Printf("RequestAuthenticatedChannel(%v)", recipient)

	if !c.network.GetHealthTracker().IsHealthy() {
		return errors.New("Cannot request authenticated channel " +
			"creation when the network is not healthy")
	}

	return auth.RequestAuth(recipient, me, message, c.rng.GetStream(),
		c.storage, c.network)
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

// HasAuthenticatedChannel returns true if an authenticated channel exists for
// the partner
func (c *Client) HasAuthenticatedChannel(partner *id.ID) bool {
	m, err := c.storage.E2e().GetPartner(partner)
	return m != nil && err == nil
}

// Create an insecure e2e relationship with a precanned user
func (c *Client) MakePrecannedAuthenticatedChannel(precannedID uint) (contact.Contact, error) {

	precan := c.MakePrecannedContact(precannedID)

	//add the precanned user as a e2e contact
	sesParam := e2e.GetDefaultSessionParams()
	err := c.storage.E2e().AddPartner(precan.ID, precan.DhPubKey,
		c.storage.E2e().GetDHPrivateKey(), sesParam, sesParam)

	return precan, err
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