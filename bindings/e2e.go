///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/xx_network/primitives/id"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
)

// e2eTrackerSingleton is used to track E2e objects so that they can be
// referenced by ID back over the bindings.
var e2eTrackerSingleton = &e2eTracker{
	tracked: make(map[int]*E2e),
	count:   0,
}

// E2e wraps the xxdk.E2e, implementing additional functions
// to support the bindings E2e interface.
type E2e struct {
	api *xxdk.E2e
	id  int
}

// GetID returns the e2eTracker ID for the E2e object.
func (e *E2e) GetID() int {
	return e.id
}

// Login creates and returns a new E2e object and adds it to the
// e2eTrackerSingleton. identity should be created via
// Cmix.MakeReceptionIdentity and passed in here. If callbacks is left nil, a
// default auth.Callbacks will be used.
func Login(cmixId int, callbacks AuthCallbacks, identity,
	e2eParamsJSON []byte) (*E2e, error) {
	if len(e2eParamsJSON) == 0 {
		jww.WARN.Printf("e2e params not specified, using defaults...")
		e2eParamsJSON = GetDefaultE2EParams()
	}

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

	params, err := parseE2EParams(e2eParamsJSON)
	if err != nil {
		return nil, err
	}

	newE2e, err := xxdk.Login(cmix.api, authCallbacks, newIdentity, params)
	if err != nil {
		return nil, err
	}

	return e2eTrackerSingleton.make(newE2e), nil
}

// LoginEphemeral creates and returns a new ephemeral E2e object and adds it to
// the e2eTrackerSingleton. identity should be created via
// Cmix.MakeReceptionIdentity or Cmix.MakeLegacyReceptionIdentity and passed in
// here. If callbacks is left nil, a default auth.Callbacks will be used.
func LoginEphemeral(cmixId int, callbacks AuthCallbacks, identity,
	e2eParamsJSON []byte) (*E2e, error) {
	if len(e2eParamsJSON) == 0 {
		jww.WARN.Printf("e2e params not specified, using defaults...")
		e2eParamsJSON = GetDefaultE2EParams()
	}

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

	params, err := parseE2EParams(e2eParamsJSON)
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

// GetContact returns a marshalled contact.Contact object for the E2e
// ReceptionIdentity.
func (e *E2e) GetContact() []byte {
	return e.api.GetReceptionIdentity().GetContact().Marshal()
}

// GetUdAddressFromNdf retrieve the User Discovery's network address fom the NDF.
func (e *E2e) GetUdAddressFromNdf() string {
	return e.api.GetCmix().GetInstance().GetPartialNdf().
		Get().UDB.Address
}

// GetUdCertFromNdf retrieves the User Discovery's TLS certificate from the NDF.
func (e *E2e) GetUdCertFromNdf() []byte {
	return []byte(e.api.GetCmix().GetInstance().GetPartialNdf().Get().UDB.Cert)
}

// GetUdContactFromNdf assembles the User Discovery's contact file from the data
// within the NDF.
//
// Returns
//  - []byte - A byte marshalled contact.Contact.
func (e *E2e) GetUdContactFromNdf() ([]byte, error) {
	udIdData := e.api.GetCmix().GetInstance().GetPartialNdf().Get().UDB.ID
	udId, err := id.Unmarshal(udIdData)
	if err != nil {
		return nil, err
	}

	udDhPubKeyData := e.api.GetCmix().GetInstance().GetPartialNdf().Get().UDB.DhPubKey
	udDhPubKey := e.api.GetE2E().GetGroup().NewInt(1)
	err = udDhPubKey.UnmarshalJSON(udDhPubKeyData)
	if err != nil {
		return nil, err
	}

	udContact := contact.Contact{
		ID:       udId,
		DhPubKey: udDhPubKey,
	}

	return udContact.Marshal(), nil
}

// AuthCallbacks is the bindings-specific interface for auth.Callbacks methods.
type AuthCallbacks interface {
	Request(contact, receptionId []byte, ephemeralId, roundId int64)
	Confirm(contact, receptionId []byte, ephemeralId, roundId int64)
	Reset(contact, receptionId []byte, ephemeralId, roundId int64)
}

// authCallback implements AuthCallbacks as a way of obtaining an auth.Callbacks
// over the bindings.
type authCallback struct {
	bindingsCbs AuthCallbacks
}

// convertAuthCallbacks turns an auth.Callbacks into an AuthCallbacks.
func convertAuthCallbacks(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) (
	contact []byte, receptionId []byte, ephemeralId int64, roundId int64) {

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
// preventing race conditions created by passing it over the bindings.
type e2eTracker struct {
	// TODO: Key on Identity.ID to prevent duplication
	tracked map[int]*E2e
	count   int
	mux     sync.RWMutex
}

// make create an E2e from a xxdk.E2e, assigns it a unique ID, and adds it to
// the e2eTracker.
func (ct *e2eTracker) make(c *xxdk.E2e) *E2e {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	id := ct.count
	ct.count++

	ct.tracked[id] = &E2e{
		api: c,
		id:  id,
	}

	return ct.tracked[id]
}

// get an E2e from the e2eTracker given its ID.
func (ct *e2eTracker) get(id int) (*E2e, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.tracked[id]
	if !exist {
		return nil, errors.Errorf("Cannot get E2e for ID %d, "+
			"does not exist", id)
	}

	return c, nil
}

// delete an E2e from the e2eTracker.
func (ct *e2eTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.tracked, id)
}
