package bindings

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/connect"
	"sync"
)

// connectionTracker is a singleton used to keep track of extant clients, allowing
// for race condition free passing over the bindings

type connectionTracker struct {
	connections map[int]*Connection
	count       int
	mux         sync.RWMutex
}

// make makes a client from an API client, assigning it a unique ID
func (ct *connectionTracker) make(c connect.Connection) *Connection {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	id := ct.count
	ct.count++

	ct.connections[id] = &Connection{
		connection: c,
		id:         id,
	}

	return ct.connections[id]
}

//get returns a client given its ID
func (ct *connectionTracker) get(id int) (*Connection, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.connections[id]
	if !exist {
		return nil, errors.Errorf("Cannot get client for id %d, client "+
			"does not exist", id)
	}

	return c, nil
}

//deletes a client if it exists
func (ct *connectionTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.connections, id)
}
