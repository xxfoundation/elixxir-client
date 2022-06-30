////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
)

// e2eTrackerSingleton is used to track E2e objects so that
// they can be referenced by id back over the bindings
var e2eTrackerSingleton = &e2eTracker{
	clients: make(map[int]*E2e),
	count:   0,
}

// E2e BindingsClient wraps the xxdk.E2e, implementing additional functions
// to support the gomobile E2e interface
type E2e struct {
	api *xxdk.E2e
	id  int
}

// GetID returns the e2eTracker ID for the E2e object
func (e *E2e) GetID() int {
	return e.id
}

// LoginE2e creates and returns a new E2e object and adds it to the e2eTrackerSingleton
// identity should be created via MakeIdentity() and passed in here
// If callbacks is left nil, a default auth.Callbacks will be used
func LoginE2e(cmixId int, callbacks AuthCallbacks, identity []byte) (*E2e, error) {
	cmix, err := cmixTrackerSingleton.get(cmixId)
	if err != nil {
		return nil, err
	}

	newIdentity, err := xxdk.UnmarshalReceptionIdentity(identity)
	if err != nil {
		return nil, err
	}

	var authCallbacks xxdk.AuthCallbacks
	if callbacks == nil {
		authCallbacks = xxdk.DefaultAuthCallbacks{}
	} else {
		authCallbacks = &authCallback{bindingsCbs: callbacks}
	}

	newE2e, err := xxdk.Login(cmix.api, authCallbacks, newIdentity)
	if err != nil {
		return nil, err
	}

	return e2eTrackerSingleton.make(newE2e), nil
}

// LoginE2eEphemeral creates and returns a new ephemeral E2e object and adds it to the e2eTrackerSingleton
// identity should be created via MakeIdentity() and passed in here
// If callbacks is left nil, a default auth.Callbacks will be used
func LoginE2eEphemeral(cmixId int, callbacks AuthCallbacks, identity []byte) (*E2e, error) {
	cmix, err := cmixTrackerSingleton.get(cmixId)
	if err != nil {
		return nil, err
	}

	newIdentity, err := xxdk.UnmarshalReceptionIdentity(identity)
	if err != nil {
		return nil, err
	}

	var authCallbacks xxdk.AuthCallbacks
	if callbacks == nil {
		authCallbacks = xxdk.DefaultAuthCallbacks{}
	} else {
		authCallbacks = &authCallback{bindingsCbs: callbacks}
	}

	newE2e, err := xxdk.LoginEphemeral(cmix.api, authCallbacks, newIdentity)
	if err != nil {
		return nil, err
	}
	return e2eTrackerSingleton.make(newE2e), nil
}

// LoginE2eLegacy creates a new E2e backed by the xxdk.Cmix persistent versioned.KV
// Uses the pre-generated transmission ID used by xxdk.Cmix
// If callbacks is left nil, a default auth.Callbacks will be used
// This function is designed to maintain backwards compatibility with previous xx messenger designs
// and should not be used for other purposes
func LoginE2eLegacy(cmixId int, callbacks AuthCallbacks) (*E2e, error) {
	cmix, err := cmixTrackerSingleton.get(cmixId)
	if err != nil {
		return nil, err
	}

	var authCallbacks xxdk.AuthCallbacks
	if callbacks == nil {
		authCallbacks = xxdk.DefaultAuthCallbacks{}
	} else {
		authCallbacks = &authCallback{bindingsCbs: callbacks}
	}

	newE2e, err := xxdk.LoginLegacy(cmix.api, authCallbacks)
	if err != nil {
		return nil, err
	}
	return e2eTrackerSingleton.make(newE2e), nil
}

// GetContact returns a marshalled contact.Contact object for the E2e ReceptionIdentity
func (e *E2e) GetContact() []byte {
	return e.api.GetReceptionIdentity().GetContact().Marshal()
}

// AuthCallbacks is the bindings-specific interface for auth.Callbacks methods.
type AuthCallbacks interface {
	Request(contact, receptionId []byte, ephemeralId, roundId int64)
	Confirm(contact, receptionId []byte, ephemeralId, roundId int64)
	Reset(contact, receptionId []byte, ephemeralId, roundId int64)
}

// authCallback implements AuthCallbacks as a way of obtaining
// an auth.Callbacks over the bindings
type authCallback struct {
	bindingsCbs AuthCallbacks
}

// convertAuthCallbacks turns an auth.Callbacks into an AuthCallbacks
func convertAuthCallbacks(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round) (contact []byte, receptionId []byte, ephemeralId int64, roundId int64) {

	contact = requestor.Marshal()
	receptionId = receptionID.Source.Marshal()
	ephemeralId = int64(receptionID.EphId.UInt64())
	roundId = int64(round.ID)
	return
}

// Confirm will be called when an auth Confirm message is processed.
func (a *authCallback) Confirm(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round, _ *xxdk.E2e) {
	a.bindingsCbs.Confirm(convertAuthCallbacks(partner, receptionID, round))
}

// Request will be called when an auth Request message is processed.
func (a *authCallback) Request(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round, _ *xxdk.E2e) {
	a.bindingsCbs.Request(convertAuthCallbacks(partner, receptionID, round))
}

// Reset will be called when an auth Reset operation occurs.
func (a *authCallback) Reset(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round, _ *xxdk.E2e) {
	a.bindingsCbs.Reset(convertAuthCallbacks(partner, receptionID, round))
}
