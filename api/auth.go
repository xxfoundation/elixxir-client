///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"encoding/binary"
	"math/rand"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

// RequestAuthenticatedChannel sends a request to another party to establish an
// authenticated channel
// It will not run if the network state is not healthy
// An error will be returned if a channel already exists or if a request was
// already received
// When a confirmation occurs, the channel will be created and the callback
// will be called
// Can be retried.
func (c *Client) RequestAuthenticatedChannel(recipient, me contact.Contact,
	message string) (id.Round, error) {
	jww.INFO.Printf("RequestAuthenticatedChannel(%s)", recipient.ID)

	if !c.network.IsHealthy() {
		return 0, errors.New("Cannot request authenticated channel " +
			"creation when the network is not healthy")
	}

	return c.auth.Request(recipient, c.GetUser().GetContact().Facts)
}

// ResetSession resets an authenticate channel that already exists
func (c *Client) ResetSession(recipient, me contact.Contact,
	message string) (id.Round, error) {
	jww.INFO.Printf("ResetSession(%s)", recipient.ID)

	if !c.network.IsHealthy() {
		return 0, errors.New("Cannot request authenticated channel " +
			"creation when the network is not healthy")
	}

	return c.auth.Reset(recipient)
}

// GetAuthRegistrar gets the object which allows the registration of auth
// callbacks
func (c *Client) GetAuthRegistrar() auth.State {
	jww.INFO.Printf("GetAuthRegistrar(...)")

	return c.auth
}

// GetAuthenticatedChannelRequest returns the contact received in a request if
// one exists for the given userID.  Returns an error if no contact is found.
func (c *Client) GetAuthenticatedChannelRequest(partner *id.ID) (contact.Contact, error) {
	jww.INFO.Printf("GetAuthenticatedChannelRequest(%s)", partner)

	return c.auth.GetReceivedRequest(partner)
}

// ConfirmAuthenticatedChannel creates an authenticated channel out of a valid
// received request and sends a message to the requestor that the request has
// been confirmed
// It will not run if the network state is not healthy
// An error will be returned if a channel already exists, if a request doest
// exist, or if the passed in contact does not exactly match the received
// request
// Can be retried.
func (c *Client) ConfirmAuthenticatedChannel(recipient contact.Contact) (id.Round, error) {
	jww.INFO.Printf("ConfirmAuthenticatedChannel(%s)", recipient.ID)

	if !c.network.IsHealthy() {
		return 0, errors.New("Cannot request authenticated channel " +
			"creation when the network is not healthy")
	}

	return c.auth.Confirm(recipient)
}

// VerifyOwnership checks if the ownership proof on a passed contact matches the
// identity in a verified contact
func (c *Client) VerifyOwnership(received, verified contact.Contact) bool {
	jww.INFO.Printf("VerifyOwnership(%s)", received.ID)

	return c.auth.VerifyOwnership(received, verified, c.e2e)
}

// HasAuthenticatedChannel returns true if an authenticated channel exists for
// the partner
func (c *Client) HasAuthenticatedChannel(partner *id.ID) bool {
	m, err := c.e2e.GetPartner(partner)
	return m != nil && err == nil
}

// Create an insecure e2e relationship with a precanned user
func (c *Client) MakePrecannedAuthenticatedChannel(precannedID uint) (
	contact.Contact, error) {

	precan := c.MakePrecannedContact(precannedID)

	myID := binary.BigEndian.Uint64(c.GetUser().GetContact().ID[:])
	// Pick a variant based on if their ID is bigger than mine.
	myVariant := sidh.KeyVariantSidhA
	theirVariant := sidh.KeyVariant(sidh.KeyVariantSidhB)
	if myID > uint64(precannedID) {
		myVariant = sidh.KeyVariantSidhB
		theirVariant = sidh.KeyVariantSidhA
	}
	prng1 := rand.New(rand.NewSource(int64(precannedID)))
	theirSIDHPrivKey := util.NewSIDHPrivateKey(theirVariant)
	theirSIDHPubKey := util.NewSIDHPublicKey(theirVariant)
	theirSIDHPrivKey.Generate(prng1)
	theirSIDHPrivKey.GeneratePublicKey(theirSIDHPubKey)

	prng2 := rand.New(rand.NewSource(int64(myID)))
	mySIDHPrivKey := util.NewSIDHPrivateKey(myVariant)
	mySIDHPubKey := util.NewSIDHPublicKey(myVariant)
	mySIDHPrivKey.Generate(prng2)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	// add the precanned user as a e2e contact
	// FIXME: these params need to be threaded through...
	sesParam := session.GetDefaultParams()
	_, err := c.e2e.AddPartner(precan.ID, precan.DhPubKey,
		c.e2e.GetHistoricalDHPrivkey(), theirSIDHPubKey,
		mySIDHPrivKey, sesParam, sesParam)

	// check garbled messages in case any messages arrived before creating
	// the channel
	c.network.CheckInProgressMessages()

	return precan, err
}

// Create an insecure e2e contact object for a precanned user
func (c *Client) MakePrecannedContact(precannedID uint) contact.Contact {

	e2eGrp := c.storage.GetE2EGroup()

	precanned := createPrecannedUser(precannedID, c.rng.GetStream(),
		c.storage.GetCmixGroup(), e2eGrp)

	// compute their public e2e key
	partnerPubKey := e2eGrp.ExpG(precanned.E2eDhPrivateKey,
		e2eGrp.NewInt(1))

	return contact.Contact{
		ID:             precanned.ReceptionID,
		DhPubKey:       partnerPubKey,
		OwnershipProof: nil,
		Facts:          make([]fact.Fact, 0),
	}
}

// GetRelationshipFingerprint returns a unique 15 character fingerprint for an
// E2E relationship. An error is returned if no relationship with the partner
// is found.
func (c *Client) GetRelationshipFingerprint(partner *id.ID) (string, error) {
	m, err := c.e2e.GetPartner(partner)
	if err != nil {
		return "", errors.Errorf("could not get partner %s: %+v",
			partner, err)
	} else if m == nil {
		return "", errors.Errorf("manager for partner %s is nil.",
			partner)
	}

	return m.GetConnectionFingerprint(), nil
}
