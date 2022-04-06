///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

type State struct {
	requestCallbacks *callbackMap
	confirmCallbacks *callbackMap
	resetCallbacks   *callbackMap

	net network.Manager
	e2e e2e.Handler
	rng *fastRNG.StreamGenerator

	store *store.Store
	event event.Manager

	registeredIDs map[id.ID]keypair

	replayRequests bool
}

type keypair struct {
	privkey *cyclic.Int
	//generated from pubkey on instantiation
	pubkey *cyclic.Int
}

func NewManager(sw interfaces.Switchboard, storage *storage.Session,
	net interfaces.NetworkManager, rng *fastRNG.StreamGenerator,
	backupTrigger interfaces.TriggerBackup, replayRequests bool) *State {
	m := &State{
		requestCallbacks: newCallbackMap(),
		confirmCallbacks: newCallbackMap(),
		resetCallbacks:   newCallbackMap(),
		rawMessages:      make(chan message.Receive, 1000),
		storage:          storage,
		net:              net,
		rng:              rng,
		backupTrigger:    backupTrigger,
		replayRequests:   replayRequests,
	}

	sw.RegisterChannel("Auth", switchboard.AnyUser(), message.Raw, m.rawMessages)

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
