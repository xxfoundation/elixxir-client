///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"encoding/base64"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

// state is an implementation of the State interface.
type state struct {
	callbacks Callbacks

	net cmixClient
	e2e e2eHandler
	rng *fastRNG.StreamGenerator

	store *store.Store
	event event.Reporter

	params Param

	backupTrigger func(reason string)
}

// NewState loads the auth state or creates new auth state if one cannot be
// found.
// Bases its reception identity and keys off of what is found in e2e.
// Uses this ID to modify the kv prefix for a unique storage path
// Parameters:
//   The params object passed in determines the services that will be used
//   to pick up requests and signal notifications. These are unique to an
//   identity, so multiple auth states with the same service tags with
//   different identities can run simultaneously.
//   Default parameters can be retrieved via GetDefaultParameters()
// Temporary:
//   In some cases, for example client <-> server communications, connections
//   are treated as ephemeral. To do so in auth, pass in an ephemeral e2e (made
//   with a memory only versioned.KV) as well as a memory only versioned.KV for
//   NewState and use GetDefaultTemporaryParams() for the parameters
func NewState(kv *versioned.KV, net cmix.Client, e2e e2e.Handler,
	rng *fastRNG.StreamGenerator, event event.Reporter, params Param,
	callbacks Callbacks, backupTrigger func(reason string)) (State, error) {
	kv = kv.Prefix(makeStorePrefix(e2e.GetReceptionID()))
	return NewStateLegacy(
		kv, net, e2e, rng, event, params, callbacks, backupTrigger)
}

// NewStateLegacy loads the auth state or creates new auth state if one cannot
// be found. Bases its reception identity and keys off of what is found in e2e.
// Does not modify the kv prefix for backwards compatibility.
// Otherwise, acts the same as NewState
func NewStateLegacy(kv *versioned.KV, net cmix.Client, e2e e2e.Handler,
	rng *fastRNG.StreamGenerator, event event.Reporter, params Param,
	callbacks Callbacks, backupTrigger func(reason string)) (State, error) {

	s := &state{
		callbacks:     callbacks,
		net:           net,
		e2e:           e2e,
		rng:           rng,
		event:         event,
		params:        params,
		backupTrigger: backupTrigger,
	}

	// create the store
	var err error
	s.store, err = store.NewOrLoadStore(kv, e2e.GetGroup(),
		&sentRequestHandler{s: s})

	// register services
	net.AddService(e2e.GetReceptionID(), message.Service{
		Identifier: e2e.GetReceptionID()[:],
		Tag:        params.RequestTag,
		Metadata:   nil,
	}, &receivedRequestService{s: s, reset: false})

	net.AddService(e2e.GetReceptionID(), message.Service{
		Identifier: e2e.GetReceptionID()[:],
		Tag:        params.ResetRequestTag,
		Metadata:   nil,
	}, &receivedRequestService{s: s, reset: true})

	if err != nil {
		return nil, errors.Errorf("Failed to make Auth State manager")
	}

	return s, nil
}

// CallAllReceivedRequests will iterate through all pending contact requests
// and replay them on the callbacks.
func (s *state) CallAllReceivedRequests() {
	rrList := s.store.GetAllReceivedRequests()
	for i := range rrList {
		rr := rrList[i]
		eph := receptionID.BuildIdentityFromRound(rr.GetContact().ID,
			rr.GetRound())
		s.callbacks.Request(rr.GetContact(), eph, rr.GetRound())
	}
}

func makeStorePrefix(partner *id.ID) string {
	return "authStore:" + base64.StdEncoding.EncodeToString(partner.Marshal())
}
