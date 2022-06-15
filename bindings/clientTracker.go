package bindings

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/xxdk"
	"sync"
)

// clientTracker is a singleton used to keep track of extant clients, allowing
// for race condition free passing over the bindings

type clientTracker struct {
	clients map[int]*Client
	count   int
	mux     sync.RWMutex
}

// make makes a client from an API client, assigning it a unique ID
func (ct *clientTracker) make(c *xxdk.Cmix) *Client {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	id := ct.count
	ct.count++

	ct.clients[id] = &Client{
		api: c,
		id:  id,
	}

	return ct.clients[id]
}

//get returns a client given its ID
func (ct *clientTracker) get(id int) (*Client, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.clients[id]
	if !exist {
		return nil, errors.Errorf("Cannot get client for id %d, client "+
			"does not exist", id)
	}

	return c, nil
}

//deletes a client if it exists
func (ct *clientTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.clients, id)
}
