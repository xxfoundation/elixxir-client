package bindings

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/xxdk"
	"sync"
)

// cmixTracker is a singleton used to keep track of extant Cmix objects,
// preventing race conditions created by passing it over the bindings
type cmixTracker struct {
	clients map[int]*Cmix
	count   int
	mux     sync.RWMutex
}

// make a Cmix from an xxdk.Cmix, assigns it a unique ID,
// and adds it to the cmixTracker
func (ct *cmixTracker) make(c *xxdk.Cmix) *Cmix {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	id := ct.count
	ct.count++

	ct.clients[id] = &Cmix{
		api: c,
		id:  id,
	}

	return ct.clients[id]
}

// get a Cmix from the cmixTracker given its ID
func (ct *cmixTracker) get(id int) (*Cmix, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.clients[id]
	if !exist {
		return nil, errors.Errorf("Cannot get client for id %d, client "+
			"does not exist", id)
	}

	return c, nil
}

// delete a Cmix if it exists in the cmixTracker
func (ct *cmixTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.clients, id)
}
