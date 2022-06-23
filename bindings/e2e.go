////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
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

	newIdentity, err := unmarshalIdentity(identity, cmix.api.GetStorage().GetE2EGroup())
	if err != nil {
		return nil, err
	}

	var authCallbacks auth.Callbacks
	if callbacks == nil {
		authCallbacks = auth.DefaultAuthCallbacks{}
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

	newIdentity, err := unmarshalIdentity(identity, cmix.api.GetStorage().GetE2EGroup())
	if err != nil {
		return nil, err
	}

	var authCallbacks auth.Callbacks
	if callbacks == nil {
		authCallbacks = auth.DefaultAuthCallbacks{}
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

	var authCallbacks auth.Callbacks
	if callbacks == nil {
		authCallbacks = auth.DefaultAuthCallbacks{}
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
	return e.api.GetReceptionIdentity().GetContact(e.api.GetStorage().GetE2EGroup()).Marshal()
}

// unmarshalIdentity is a helper function for taking in a marshalled xxdk.ReceptionIdentity and making it an object
func unmarshalIdentity(marshaled []byte, e2eGrp *cyclic.Group) (xxdk.ReceptionIdentity, error) {
	newIdentity := xxdk.ReceptionIdentity{}

	// Unmarshal given identity into ReceptionIdentity object
	givenIdentity := ReceptionIdentity{}
	err := json.Unmarshal(marshaled, &givenIdentity)
	if err != nil {
		return xxdk.ReceptionIdentity{}, err
	}

	newIdentity.ID, err = id.Unmarshal(givenIdentity.ID)
	if err != nil {
		return xxdk.ReceptionIdentity{}, err
	}

	newIdentity.DHKeyPrivate = e2eGrp.NewInt(1)
	err = newIdentity.DHKeyPrivate.UnmarshalJSON(givenIdentity.DHKeyPrivate)
	if err != nil {
		return xxdk.ReceptionIdentity{}, err
	}

	newIdentity.RSAPrivatePem, err = rsa.LoadPrivateKeyFromPem(givenIdentity.RSAPrivatePem)
	if err != nil {
		return xxdk.ReceptionIdentity{}, err
	}

	newIdentity.Salt = givenIdentity.Salt
	return newIdentity, nil
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
func (a *authCallback) Confirm(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	a.bindingsCbs.Confirm(convertAuthCallbacks(requestor, receptionID, round))
}

// Request will be called when an auth Request message is processed.
func (a *authCallback) Request(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	a.bindingsCbs.Request(convertAuthCallbacks(requestor, receptionID, round))
}

// Reset will be called when an auth Reset operation occurs.
func (a *authCallback) Reset(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	a.bindingsCbs.Reset(convertAuthCallbacks(requestor, receptionID, round))
}
