///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/primitives/fact"
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
	jww.INFO.Printf("RequestAuthenticatedChannel(%s)", recipient.ID)

	if !c.network.GetHealthTracker().IsHealthy() {
		return errors.New("Cannot request authenticated channel " +
			"creation when the network is not healthy")
	}

	return auth.RequestAuth(recipient, me, message, c.rng.GetStream(),
		c.storage, c.network)
}

// GetAuthRegistrar gets the object which allows the registration of auth
// callbacks
func (c *Client) GetAuthRegistrar() interfaces.Auth {
	jww.INFO.Printf("GetAuthRegistrar(...)")

	return c.auth
}

// GetAuthenticatedChannelRequest returns the contact received in a request if
// one exists for the given userID.  Returns an error if no contact is found.
func (c *Client) GetAuthenticatedChannelRequest(partner *id.ID) (contact.Contact, error) {
	jww.INFO.Printf("GetAuthenticatedChannelRequest(%s)", partner)

	return c.storage.Auth().GetReceivedRequestData(partner)
}

// ConfirmAuthenticatedChannel creates an authenticated channel out of a valid
// received request and sends a message to the requestor that the request has
// been confirmed
// It will not run if the network status is not healthy
// An error will be returned if a channel already exists, if a request doest
// exist, or if the passed in contact does not exactly match the received
// request
func (c *Client) ConfirmAuthenticatedChannel(recipient contact.Contact) error {
	jww.INFO.Printf("ConfirmAuthenticatedChannel(%s)", recipient.ID)

	if !c.network.GetHealthTracker().IsHealthy() {
		return errors.New("Cannot request authenticated channel " +
			"creation when the network is not healthy")
	}

	return auth.ConfirmRequestAuth(recipient, c.rng.GetStream(),
		c.storage, c.network)
}

// VerifyOwnership checks if the ownership proof on a passed contact matches the
// identity in a verified contact
func (c *Client) VerifyOwnership(received, verified contact.Contact) bool {
	jww.INFO.Printf("VerifyOwnership(%s)", received.ID)

	return auth.VerifyOwnership(received, verified, c.storage)
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

	// add the precanned user as a e2e contact
	sesParam := c.parameters.E2EParams
	err := c.storage.E2e().AddPartner(precan.ID, precan.DhPubKey,
		c.storage.E2e().GetDHPrivateKey(), sesParam, sesParam)

	return precan, err
}

// Create an insecure e2e contact object for a precanned user
func (c *Client) MakePrecannedContact(precannedID uint) contact.Contact {

	e2eGrp := c.storage.E2e().GetGroup()

	// get the user definition
	precanned := createPrecannedUser(precannedID, c.rng.GetStream(),
		c.storage.Cmix().GetGroup(), e2eGrp)

	// compute their public e2e key
	partnerPubKey := e2eGrp.ExpG(precanned.E2eDhPrivateKey, e2eGrp.NewInt(1))

	return contact.Contact{
		ID:             precanned.ReceptionID,
		DhPubKey:       partnerPubKey,
		OwnershipProof: nil,
		Facts:          make([]fact.Fact, 0),
	}
}
