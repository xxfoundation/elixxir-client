///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/xx_network/primitives/id"
)

type Manager struct {
	requestCallbacks *callbackMap
	confirmCallbacks *callbackMap

	rawMessages chan message.Receive

	storage *storage.Session
	net     interfaces.NetworkManager

	replayRequests bool
}

func NewManager(sw interfaces.Switchboard, storage *storage.Session,
	net interfaces.NetworkManager, replayRequests bool) *Manager {
	m := &Manager{
		requestCallbacks: newCallbackMap(),
		confirmCallbacks: newCallbackMap(),
		rawMessages:      make(chan message.Receive, 1000),
		storage:          storage,
		net:              net,
		replayRequests:   replayRequests,
	}

	sw.RegisterChannel("Auth", switchboard.AnyUser(), message.Raw, m.rawMessages)

	return m
}

// Adds a general callback to be used on auth requests. This will be preempted
// by any specific callback
func (m *Manager) AddGeneralRequestCallback(cb interfaces.RequestCallback) {
	m.requestCallbacks.AddGeneral(cb)
}

// Adds a general callback to be used on auth requests. This will not be
// preempted by any specific callback. It is recommended that the specific
// callbacks are used, this is primarily for debugging.
func (m *Manager) AddOverrideRequestCallback(cb interfaces.RequestCallback) {
	m.requestCallbacks.AddOverride(cb)
}

// Adds a specific callback to be used on auth requests. This will preempt a
// general callback, meaning the request will be heard on this callback and not
// the general. Request will still be heard on override callbacks.
func (m *Manager) AddSpecificRequestCallback(id *id.ID, cb interfaces.RequestCallback) {
	m.requestCallbacks.AddSpecific(id, cb)
}

// Removes a specific callback to be used on auth requests.
func (m *Manager) RemoveSpecificRequestCallback(id *id.ID) {
	m.requestCallbacks.RemoveSpecific(id)
}

// Adds a general callback to be used on auth confirms. This will be preempted
// by any specific callback
func (m *Manager) AddGeneralConfirmCallback(cb interfaces.ConfirmCallback) {
	m.confirmCallbacks.AddGeneral(cb)
}

// Adds a general callback to be used on auth confirms. This will not be
// preempted by any specific callback. It is recommended that the specific
// callbacks are used, this is primarily for debugging.
func (m *Manager) AddOverrideConfirmCallback(cb interfaces.ConfirmCallback) {
	m.confirmCallbacks.AddOverride(cb)
}

// Adds a specific callback to be used on auth confirms. This will preempt a
// general callback, meaning the request will be heard on this callback and not
// the general. Request will still be heard on override callbacks.
func (m *Manager) AddSpecificConfirmCallback(id *id.ID, cb interfaces.ConfirmCallback) {
	m.confirmCallbacks.AddSpecific(id, cb)
}

// Removes a specific callback to be used on auth confirm.
func (m *Manager) RemoveSpecificConfirmCallback(id *id.ID) {
	m.confirmCallbacks.RemoveSpecific(id)
}

func (m *Manager)ReplayRequests(){
	cList := m.storage.Auth().GetAllReceived()
	for i := range cList{
		c := cList[i]
		cbList := m.requestCallbacks.Get(c.ID)
		for _, cb := range cbList {
			rcb := cb.(interfaces.RequestCallback)
			go rcb(c)
		}
	}
}
