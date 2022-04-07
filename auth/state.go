///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

type State struct {
	requestCallbacks *callbackMap
	confirmCallbacks *callbackMap
	resetCallbacks   *callbackMap

	net cmix.Client
	e2e e2e.Handler
	rng *fastRNG.StreamGenerator

	store *store.Store
	event event.Manager

	registeredIDs

	params Param
}

type identity struct {
	identity                *id.ID
	pubkey, privkey         *cyclic.Int
	request, confirm, reset Callback
}

func NewManager(kv *versioned.KV, net cmix.Client, e2e e2e.Handler,
	rng *fastRNG.StreamGenerator, event event.Manager, params Param,
	defaultID []identity) *State {
	m := &State{
		requestCallbacks: newCallbackMap(),
		confirmCallbacks: newCallbackMap(),
		resetCallbacks:   newCallbackMap(),

		net: net,
		e2e: e2e,
		rng: rng,

		params: params,
		event:  event,

		//created lazily in add identity, see add identity for more details
		store: nil,
	}

	return m
}

// ReplayRequests will iterate through all pending contact requests and resend them
// to the desired contact.
func (s *State) ReplayRequests() {
	cList := s.storage.Auth().GetAllReceived()
	for i := range cList {
		c := cList[i]
		cbList := s.requestCallbacks.Get(c.ID)
		for _, cb := range cbList {
			rcb := cb.(interfaces.RequestCallback)
			go rcb(c)
		}
	}
}

// AddIdentity adds an identity and its callbacks to receive requests.
// This auto registers the appropriate services
// Note: the internal storage for auth is loaded on the first added identity,
// with that identity as the default identity. This is to allow v2.0
// instantiations of this library (pre April 2022) to load, before requests were
// keyed on both parties IDs
func (s *State) AddIdentity(identity *id.ID, pubkey, privkey *cyclic.Int,
	request, confirm, reset Callback) {
	if s.store == nil {
		//load store
	}

}

func (s *State) AddDefaultIdentity(identity *id.ID, pubkey, privkey *cyclic.Int,
	request, confirm, reset Callback) {
	if s.store == nil {
		//load store
	}

}
