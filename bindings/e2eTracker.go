////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/xxdk"
	"sync"
)

// e2eTracker is a singleton used to keep track of extant E2e objects,
// preventing race conditions created by passing it over the bindings
type e2eTracker struct {
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
