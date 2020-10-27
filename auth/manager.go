package auth

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/xx_network/primitives/id"
)

type Manager struct {
	requestCallbacks *callbackMap
	confirmCallbacks *callbackMap

	rawMessages chan message.Receive

	storage *storage.Session
	net     interfaces.NetworkManager
}

func NewManager(sw interfaces.Switchboard, storage *storage.Session,
	net interfaces.NetworkManager) *Manager {
	m := &Manager{
		requestCallbacks: newCallbackMap(),
		confirmCallbacks: newCallbackMap(),
		rawMessages:      make(chan message.Receive, 1000),
		storage:          storage,
		net:              net,
	}

	sw.RegisterChannel("Auth", &id.ID{}, message.Raw, m.rawMessages)

	return m
}

// Adds a general callback to be used on auth requests. This will be preempted
// by any specific callback
func (m *Manager) AddGeneralRequestCallback(cb RequestCallback) {
	m.requestCallbacks.AddGeneral(cb)
}

// Adds a general callback to be used on auth requests. This will not be
// preempted by any specific callback. It is recommended that the specific
// callbacks are used, this is primarily for debugging.
func (m *Manager) AddOverrideRequestCallback(cb RequestCallback) {
	m.requestCallbacks.AddOverride(cb)
}

// Adds a specific callback to be used on auth requests. This will preempt a
// general callback, meaning the request will be heard on this callback and not
// the general. Request will still be heard on override callbacks.
func (m *Manager) AddSpecificRequestCallback(id *id.ID, cb RequestCallback) {
	m.requestCallbacks.AddSpecific(id, cb)
}

// Removes a specific callback to be used on auth requests.
func (m *Manager) RemoveSpecificRequestCallback(id *id.ID) {
	m.requestCallbacks.RemoveSpecific(id)
}

// Adds a general callback to be used on auth confirms. This will be preempted
// by any specific callback
func (m *Manager) AddGeneralConfirmCallback(cb ConfirmCallback) {
	m.confirmCallbacks.AddGeneral(cb)
}

// Adds a general callback to be used on auth confirms. This will not be
// preempted by any specific callback. It is recommended that the specific
// callbacks are used, this is primarily for debugging.
func (m *Manager) AddOverrideConfirmCallback(cb ConfirmCallback) {
	m.confirmCallbacks.AddOverride(cb)
}

// Adds a specific callback to be used on auth confirms. This will preempt a
// general callback, meaning the request will be heard on this callback and not
// the general. Request will still be heard on override callbacks.
func (m *Manager) AddSpecificConfirmCallback(id *id.ID, cb ConfirmCallback) {
	m.confirmCallbacks.AddSpecific(id, cb)
}

// Removes a specific callback to be used on auth confirm.
func (m *Manager) RemoveSpecificConfirmCallback(id *id.ID) {
	m.confirmCallbacks.RemoveSpecific(id)
}
