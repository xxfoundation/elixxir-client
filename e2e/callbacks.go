////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// partnerCallbacks is a thread-safe wrapper for Callbacks specific to partner
// IDs. For E2E operations with a specific partner, these Callbacks will be used
// instead.
type partnerCallbacks struct {
	callbacks map[id.ID]Callbacks
	sync.RWMutex
}

// newPartnerCallbacks initializes an empty partnerCallbacks.
func newPartnerCallbacks() *partnerCallbacks {
	return &partnerCallbacks{
		callbacks: make(map[id.ID]Callbacks),
	}
}

// add registers Callbacks that override the generic E2E callback for the given
// partner ID.
func (pcb *partnerCallbacks) add(partnerID *id.ID, cbs Callbacks) {
	pcb.Lock()
	defer pcb.Unlock()
	pcb.callbacks[*partnerID] = cbs
}

// delete deletes the callbacks for the given partner ID.
func (pcb *partnerCallbacks) delete(partnerID *id.ID) {
	pcb.Lock()
	defer pcb.Unlock()
	delete(pcb.callbacks, *partnerID)
}

// get returns the Callbacks for the given partner ID.
func (pcb *partnerCallbacks) get(partnerID *id.ID) Callbacks {
	pcb.RLock()
	defer pcb.RUnlock()

	return pcb.callbacks[*partnerID]
}

// DefaultCallbacks is a simple structure for providing a default Callbacks
// implementation. It should generally not be used.
type DefaultCallbacks struct{}

func (d *DefaultCallbacks) ConnectionClosed(*id.ID, rounds.Round) {
	jww.ERROR.Printf("No valid e2e callback assigned!")
}
