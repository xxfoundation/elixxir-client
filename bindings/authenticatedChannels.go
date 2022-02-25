///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"fmt"

	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

// Create an insecure e2e relationship with a precanned user
func (c *Client) MakePrecannedAuthenticatedChannel(precannedID int) (*Contact, error) {
	precannedContact, err := c.api.MakePrecannedAuthenticatedChannel(uint(precannedID))
	if err != nil {
		return nil, fmt.Errorf("Failed to "+
			"MakePrecannedAuthenticatedChannel: %+v", err)
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
		return 0, fmt.Errorf("Failed to "+
			"RequestAuthenticatedChannel: Failed to Unmarshal Recipent: "+
			"%+v", err)
	}

	me, err := contact.Unmarshal(meMarshaled)

	if err != nil {
		return 0, fmt.Errorf("Failed to "+
			"RequestAuthenticatedChannel: Failed to Unmarshal Me: %+v", err)
	}

	rid, err := c.api.RequestAuthenticatedChannel(recipent, me, message)

	return int(rid), err
}

// ResetSession resets an authenticated channel that already exists
func (c *Client) ResetSession(recipientMarshaled,
	meMarshaled []byte, message string) (int, error) {
	recipent, err := contact.Unmarshal(recipientMarshaled)

	if err != nil {
		return 0, fmt.Errorf("failed to "+
			"ResetSession: failed to Unmarshal Recipent: "+
			"%+v", err)
	}

	me, err := contact.Unmarshal(meMarshaled)

	if err != nil {
		return 0, fmt.Errorf("failed to "+
			"ResetSession: Failed to Unmarshal Me: %+v", err)
	}

	rid, err := c.api.ResetSession(recipent, me, message)

	return int(rid), err
}

// RegisterAuthCallbacks registers all callbacks for authenticated channels.
// This can only be called once
func (c *Client) RegisterAuthCallbacks(request AuthRequestCallback,
	confirm AuthConfirmCallback, reset AuthResetCallback) {

	requestFunc := func(requestor contact.Contact) {
		requestorBind := &Contact{c: &requestor}
		request.Callback(requestorBind)
	}

	resetFunc := func(resetor contact.Contact) {
		resetorBind := &Contact{c: &resetor}
		reset.Callback(resetorBind)
	}

	confirmFunc := func(partner contact.Contact) {
		partnerBind := &Contact{c: &partner}
		confirm.Callback(partnerBind)
	}

	c.api.GetAuthRegistrar().AddGeneralConfirmCallback(confirmFunc)
	c.api.GetAuthRegistrar().AddGeneralRequestCallback(requestFunc)
	c.api.GetAuthRegistrar().AddResetCallback(resetFunc)
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
		return 0, fmt.Errorf("Failed to "+
			"ConfirmAuthenticatedChannel: Failed to Unmarshal Recipient: "+
			"%+v", err)
	}

	rid, err := c.api.ConfirmAuthenticatedChannel(recipent)

	return int(rid), err
}

// VerifyOwnership checks if the ownership proof on a passed contact matches the
// identity in a verified contact
func (c *Client) VerifyOwnership(receivedMarshaled, verifiedMarshaled []byte) (bool, error) {
	received, err := contact.Unmarshal(receivedMarshaled)

	if err != nil {
		return false, fmt.Errorf("Failed to "+
			"VerifyOwnership: Failed to Unmarshal Received: %+v", err)
	}

	verified, err := contact.Unmarshal(verifiedMarshaled)

	if err != nil {
		return false, fmt.Errorf("Failed to "+
			"VerifyOwnership: Failed to Unmarshal Verified: %+v", err)
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

// ReplayRequests Resends all pending requests over the normal callbacks
func (c *Client) ReplayRequests() {
	c.api.GetAuthRegistrar().ReplayRequests()
}
