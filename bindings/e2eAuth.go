////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

// Request sends a contact request from the user identity in the imported E2e
// structure to the passed contact, as well as the passed facts (it will error
// if they are too long).
//
// The other party must accept the request by calling Confirm to be able to send
// messages using E2e.SendE2E. When the other party does so, the "confirm"
// callback will get called.
//
// The round the request is initially sent on will be returned, but the request
// will be listed as a critical message, so the underlying cMix client will auto
// resend it in the event of failure.
//
// A request cannot be sent for a contact who has already received a request or
// who is already a partner.
//
// The request sends as a critical message, if the round it sends on fails, it
// will be auto resent by the cMix client.
//
// Parameters:
//  - partnerContact - the marshalled bytes of the contact.Contact object.
//  - factsListJson - the JSON marshalled bytes of [fact.FactList].
//
// Returns:
//  - int64 - ID of the round (convert to uint64)
func (e *E2e) Request(partnerContact, factsListJson []byte) (int64, error) {
	var factsList fact.FactList
	err := json.Unmarshal(factsListJson, &factsList)
	if err != nil {
		return 0, err
	}

	partner, err := contact.Unmarshal(partnerContact)
	if err != nil {
		return 0, err
	}

	roundID, err := e.api.GetAuth().Request(partner, factsList)

	return int64(roundID), err
}

// Confirm sends a confirmation for a received request. It can only be called
// once. This both sends keying material to the other party and creates a
// channel in the e2e handler, after which e2e messages can be sent to the
// partner using E2e.SendE2E.
//
// The round the request is initially sent on will be returned, but the request
// will be listed as a critical message, so the underlying cMix client will auto
// resend it in the event of failure.
//
// A confirmation cannot be sent for a contact who has not sent a request or who
// is already a partner. This can only be called once for a specific contact.
// The confirmation sends as a critical message; if the round it sends on fails,
// it will be auto resent by the cMix client.
//
// If the confirmation must be resent, use ReplayConfirm.
//
// Parameters:
//  - partnerContact - the marshalled bytes of the contact.Contact object.
//
// Returns:
//  - int64 - ID of the round (convert to uint64)
func (e *E2e) Confirm(partnerContact []byte) (int64, error) {
	partner, err := contact.Unmarshal(partnerContact)
	if err != nil {
		return 0, err
	}

	roundID, err := e.api.GetAuth().Confirm(partner)

	return int64(roundID), err
}

// Reset sends a contact reset request from the user identity in the imported
// e2e structure to the passed contact, as well as the passed facts (it will
// error if they are too long).
//
// This deletes all traces of the relationship with the partner from e2e and
// create a new relationship from scratch.
//
// The round the reset is initially sent on will be returned, but the request
// will be listed as a critical message, so the underlying cMix client will auto
// resend it in the event of failure.
//
// A request cannot be sent for a contact who has already received a request or
// who is already a partner.
//
// Parameters:
//  - partnerContact - the marshalled bytes of the contact.Contact object.
//
// Returns:
//  - int64 - ID of the round (convert to uint64)
func (e *E2e) Reset(partnerContact []byte) (int64, error) {
	partner, err := contact.Unmarshal(partnerContact)
	if err != nil {
		return 0, err
	}

	roundID, err := e.api.GetAuth().Reset(partner)

	return int64(roundID), err
}

// ReplayConfirm resends a confirmation to the partner. It will fail to send if
// the send relationship with the partner has already ratcheted.
//
// The confirmation sends as a critical message; if the round it sends on fails,
// it will be auto resent by the cMix client.
//
// This will not be useful if either side has ratcheted.
//
// Parameters:
//  - partnerID - the marshalled bytes of the id.ID object.
//
// Returns:
//  - int64 - ID of the round (convert to uint64)
func (e *E2e) ReplayConfirm(partnerID []byte) (int64, error) {
	partner, err := id.Unmarshal(partnerID)
	if err != nil {
		return 0, err
	}

	roundID, err := e.api.GetAuth().ReplayConfirm(partner)

	return int64(roundID), err
}

// CallAllReceivedRequests will iterate through all pending contact requests and
// replay them on the callbacks.
func (e *E2e) CallAllReceivedRequests() {
	e.api.GetAuth().CallAllReceivedRequests()
}

// DeleteRequest deletes sent or received requests for a specific partner ID.
//
// Parameters:
//  - partnerID - the marshalled bytes of the id.ID object.
func (e *E2e) DeleteRequest(partnerID []byte) error {
	partner, err := id.Unmarshal(partnerID)
	if err != nil {
		return err
	}

	return e.api.GetAuth().DeleteRequest(partner)
}

// DeleteAllRequests clears all requests from auth storage.
func (e *E2e) DeleteAllRequests() error {
	return e.api.GetAuth().DeleteAllRequests()
}

// DeleteSentRequests clears all sent requests from auth storage.
func (e *E2e) DeleteSentRequests() error {
	return e.api.GetAuth().DeleteSentRequests()
}

// DeleteReceiveRequests clears all received requests from auth
// storage.
func (e *E2e) DeleteReceiveRequests() error {
	return e.api.GetAuth().DeleteReceiveRequests()
}

// GetReceivedRequest returns a contact if there is a received request for it.
//
// Parameters:
//  - partnerID - the marshalled bytes of the id.ID object.
//
// Returns:
//  - []byte - the marshalled bytes of the contact.Contact object.
func (e *E2e) GetReceivedRequest(partnerID []byte) ([]byte, error) {
	partner, err := id.Unmarshal(partnerID)
	if err != nil {
		return nil, err
	}

	c, err := e.api.GetAuth().GetReceivedRequest(partner)
	if err != nil {
		return nil, err
	}

	return c.Marshal(), nil
}

// VerifyOwnership checks if the received ownership proof is valid.
//
// Parameters:
//  - receivedContact, verifiedContact - the marshalled bytes of the
//      contact.Contact object.
//  - e2eId - ID of the e2e handler
func (e *E2e) VerifyOwnership(
	receivedContact, verifiedContact []byte, e2eId int) (bool, error) {
	received, err := contact.Unmarshal(receivedContact)
	if err != nil {
		return false, err
	}

	verified, err := contact.Unmarshal(verifiedContact)
	if err != nil {
		return false, err
	}

	user, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return false, err
	}

	return e.api.GetAuth().VerifyOwnership(
		received, verified, user.api.GetE2E()), nil
}

// AddPartnerCallback adds a new callback that overrides the generic auth
// callback for the given partner ID.
//
// Parameters:
//  - partnerID - the marshalled bytes of the id.ID object.
func (e *E2e) AddPartnerCallback(partnerID []byte, cb AuthCallbacks) error {
	partnerId, err := id.Unmarshal(partnerID)
	if err != nil {
		return err
	}

	e.api.GetAuth().AddPartnerCallback(partnerId,
		xxdk.MakeAuthCallbacksAdapter(&authCallback{bindingsCbs: cb}, e.api))
	return nil
}

// DeletePartnerCallback deletes the callback that overrides the generic
// auth callback for the given partner ID.
//
// Parameters:
//  - partnerID - the marshalled bytes of the id.ID object.
func (e *E2e) DeletePartnerCallback(partnerID []byte) error {
	partnerId, err := id.Unmarshal(partnerID)
	if err != nil {
		return err
	}

	e.api.GetAuth().DeletePartnerCallback(partnerId)

	return nil
}
