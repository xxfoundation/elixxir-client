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

// Login creates and returns a new E2e object
// and adds it to the e2eTrackerSingleton
// identity can be left nil such that a new
// TransmissionIdentity will be created automatically
func (e *E2e) Login(cmixId int, callbacks AuthCallbacks, identity []byte) (*E2e, error) {
	cmix, err := cmixTrackerSingleton.get(cmixId)
	if err != nil {
		return nil, err
	}

	newIdentity := &xxdk.TransmissionIdentity{}
	if identity == nil {
		newIdentity = nil
	} else {
		newIdentity, err = cmix.unmarshalIdentity(identity)
		if err != nil {
			return nil, err
		}
	}

	authCallbacks := authCallback{bindingsCbs: callbacks}
	newE2e, err := xxdk.Login(cmix.api, authCallbacks, newIdentity)
	if err != nil {
		return nil, err
	}
	return e2eTrackerSingleton.make(newE2e), nil
}

// AuthCallbacks is the bindings-specific interface for auth.Callbacks methods.
type AuthCallbacks interface {
	Request(contact, receptionId []byte, ephemeralId, roundId uint64)
	Confirm(contact, receptionId []byte, ephemeralId, roundId uint64)
	Reset(contact, receptionId []byte, ephemeralId, roundId uint64)
}

// authCallback implements AuthCallbacks
type authCallback struct {
	bindingsCbs AuthCallbacks
}

// convertAuthCallbacks turns an auth.Callbacks into an AuthCallbacks
func convertAuthCallbacks(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round) (contact []byte, receptionId []byte, ephemeralId uint64, roundId uint64) {

	contact = requestor.Marshal()
	receptionId = receptionID.Source.Marshal()
	ephemeralId = receptionID.EphId.UInt64()
	roundId = uint64(round.ID)
	return
}

// Confirm will be called when an auth Confirm message is processed.
func (a authCallback) Confirm(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	a.bindingsCbs.Confirm(convertAuthCallbacks(requestor, receptionID, round))
}

// Request will be called when an auth Request message is processed.
func (a authCallback) Request(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	a.bindingsCbs.Request(convertAuthCallbacks(requestor, receptionID, round))
}

// Reset will be called when an auth Reset operation occurs.
func (a authCallback) Reset(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	a.bindingsCbs.Reset(convertAuthCallbacks(requestor, receptionID, round))
}
