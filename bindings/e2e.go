///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"sync"

	"github.com/pkg/errors"
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
	paramsJSON := GetDefaultE2EParams()
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

	params, err := parseE2EParams(paramsJSON)
	if err != nil {
		return nil, err
	}

	newE2e, err := xxdk.Login(cmix.api, authCallbacks, newIdentity, params)
	if err != nil {
		return nil, err
	}

	return e2eTrackerSingleton.make(newE2e), nil
}

// LoginE2eEphemeral creates and returns a new ephemeral E2e object and adds it to the e2eTrackerSingleton
// identity should be created via MakeIdentity() and passed in here
// If callbacks is left nil, a default auth.Callbacks will be used
func LoginE2eEphemeral(cmixId int, callbacks AuthCallbacks, identity []byte) (*E2e, error) {
	paramsJSON := GetDefaultE2EParams()
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

	params, err := parseE2EParams(paramsJSON)
	if err != nil {
		return nil, err
	}

	newE2e, err := xxdk.LoginEphemeral(cmix.api, authCallbacks,
		newIdentity, params)
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
	paramsJSON := GetDefaultE2EParams()
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

	params, err := parseE2EParams(paramsJSON)
	if err != nil {
		return nil, err
	}

	newE2e, err := xxdk.LoginLegacy(cmix.api, params, authCallbacks)
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

// e2eTracker is a singleton used to keep track of extant E2e objects,
// preventing race conditions created by passing it over the bindings
type e2eTracker struct {
	// TODO: Key on Identity.ID to prevent duplication
	clients map[int]*E2e
	count   int
	mux     sync.RWMutex
}

// make a E2e from an xxdk.E2e, assigns it a unique ID,
// and adds it to the e2eTracker
func (ct *e2eTracker) make(c *xxdk.E2e) *E2e {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	id := ct.count
	ct.count++

	ct.clients[id] = &E2e{
		api: c,
		id:  id,
	}

	return ct.clients[id]
}

// get an E2e from the e2eTracker given its ID
func (ct *e2eTracker) get(id int) (*E2e, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.clients[id]
	if !exist {
		return nil, errors.Errorf("Cannot get client for id %d, client "+
			"does not exist", id)
	}

	return c, nil
}

// delete an E2e if it exists in the e2eTracker
func (ct *e2eTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.clients, id)
}
