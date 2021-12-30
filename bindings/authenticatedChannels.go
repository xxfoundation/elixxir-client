///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"errors"
	"fmt"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

// Create an insecure e2e relationship with a precanned user
func (c *Client) MakePrecannedAuthenticatedChannel(precannedID int) (*Contact, error) {
	precannedContact, err := c.api.MakePrecannedAuthenticatedChannel(uint(precannedID))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to "+
			"MakePrecannedAuthenticatedChannel: %+v", err))
	}
	return &Contact{c: &precannedContact}, nil
}

// RequestAuthenticatedChannel sends a request to another party to establish an
// authenticated channel
// It will not run if the network status is not healthy
// An error will be returned if a channel already exists, if a request was
// already received.
// When a confirmation occurs, the channel will be created and the callback
// will be called
// This can be called many times and retried.
//
// This function takes the marshaled send report to ensure a memory leak does
// not occur as a result of both sides of the bindings holding a refrence to
// the same pointer.
func (c *Client) RequestAuthenticatedChannel(recipientMarshaled,
	meMarshaled []byte, message string) (int, error) {
	recipent, err := contact.Unmarshal(recipientMarshaled)

	if err != nil {
		return 0, errors.New(fmt.Sprintf("Failed to "+
			"RequestAuthenticatedChannel: Failed to Unmarshal Recipent: "+
			"%+v", err))
	}

	me, err := contact.Unmarshal(meMarshaled)

	if err != nil {
		return 0, errors.New(fmt.Sprintf("Failed to "+
			"RequestAuthenticatedChannel: Failed to Unmarshal Me: %+v", err))
	}

	rid, err := c.api.RequestAuthenticatedChannel(recipent, me, message)

	return int(rid), err
}

// RegisterAuthCallbacks registers both callbacks for authenticated channels.
// This can only be called once
func (c *Client) RegisterAuthCallbacks(request AuthRequestCallback,
	confirm AuthConfirmCallback) {

	requestFunc := func(requestor contact.Contact, message string) {
		requestorBind := &Contact{c: &requestor}
		request.Callback(requestorBind)
	}

	confirmFunc := func(partner contact.Contact) {
		partnerBind := &Contact{c: &partner}
		confirm.Callback(partnerBind)
	}

	c.api.GetAuthRegistrar().AddGeneralConfirmCallback(confirmFunc)
	c.api.GetAuthRegistrar().AddGeneralRequestCallback(requestFunc)

	return
}

// ConfirmAuthenticatedChannel creates an authenticated channel out of a valid
// received request and sends a message to the requestor that the request has
// been confirmed
// It will not run if the network status is not healthy
// An error will be returned if a request doest
// exist, or if the passed in contact does not exactly match the received
// request.
// This can be called many times and retried.
//
// This function takes the marshaled send report to ensure a memory leak does
// not occur as a result of both sides of the bindings holding a refrence to
// the same pointer.
func (c *Client) ConfirmAuthenticatedChannel(recipientMarshaled []byte) (int, error) {
	recipent, err := contact.Unmarshal(recipientMarshaled)

	if err != nil {
		return 0, errors.New(fmt.Sprintf("Failed to "+
			"ConfirmAuthenticatedChannel: Failed to Unmarshal Recipient: "+
			"%+v", err))
	}

	rid, err := c.api.ConfirmAuthenticatedChannel(recipent)

	return int(rid), err
}

// VerifyOwnership checks if the ownership proof on a passed contact matches the
// identity in a verified contact
func (c *Client) VerifyOwnership(receivedMarshaled, verifiedMarshaled []byte) (bool, error) {
	received, err := contact.Unmarshal(receivedMarshaled)

	if err != nil {
		return false, errors.New(fmt.Sprintf("Failed to "+
			"VerifyOwnership: Failed to Unmarshal Received: %+v", err))
	}

	verified, err := contact.Unmarshal(verifiedMarshaled)

	if err != nil {
		return false, errors.New(fmt.Sprintf("Failed to "+
			"VerifyOwnership: Failed to Unmarshal Verified: %+v", err))
	}

	return c.api.VerifyOwnership(received, verified), nil
}

// GetRelationshipFingerprint returns a unique 15 character fingerprint for an
// E2E relationship. An error is returned if no relationship with the partner
// is found.
func (c *Client) GetRelationshipFingerprint(partnerID []byte) (string, error) {
	partner, err := id.Unmarshal(partnerID)
	if err != nil {
		return "", err
	}

	return c.api.GetRelationshipFingerprint(partner)
}
