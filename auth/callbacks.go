///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/cmix/historical"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type Callback func(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round historical.Round)

type callbackMap struct {
	callbacks map[id.ID]Callback
	mux       sync.RWMutex
}

func newCallbackMap() *callbackMap {
	return &callbackMap{
		callbacks: make(map[id.ID]Callback),
	}
}

//adds a general callback. This will be preempted by any specific callback
func (cm *callbackMap) AddCallback(recipeint *id.ID, cb Callback) {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	cm.callbacks[*recipeint] = cb
}

// removes a callback for a specific user ID if it exists.
func (cm *callbackMap) RemoveCallback(id *id.ID) {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	delete(cm.callbacks, *id)
}

//get all callback which fit with the passed id
func (cm *callbackMap) Get(id *id.ID) (Callback, bool) {
	cm.mux.RLock()
	defer cm.mux.RUnlock()
	cb, exist := cm.callbacks[*id]
	return cb, exist
}
