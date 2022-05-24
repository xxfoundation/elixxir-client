package bindings

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/connect"
	"sync"
)

// connectionTracker is a singleton used to keep track of extant clients, allowing
// for race condition free passing over the bindings

type authenticatedConnectionTracker struct {
	connections map[int]*AuthenticatedConnection
	count       int
	mux         sync.RWMutex
}

// make makes a client from an API client, assigning it a unique ID
func (act *authenticatedConnectionTracker) make(c connect.AuthenticatedConnection) *AuthenticatedConnection {
	act.mux.Lock()
	defer act.mux.Unlock()

	id := act.count
	act.count++

	act.connections[id] = &AuthenticatedConnection{
		Connection: Connection{
			connection: c,
			id:         id,
		},
	}

	return act.connections[id]
}

//get returns a client given its ID
func (act *authenticatedConnectionTracker) get(id int) (*AuthenticatedConnection, error) {
	act.mux.RLock()
	defer act.mux.RUnlock()

	c, exist := act.connections[id]
	if !exist {
		return nil, errors.Errorf("Cannot get client for id %d, client "+
			"does not exist", id)
	}

	return c, nil
}

//deletes a client if it exists
func (act *authenticatedConnectionTracker) delete(id int) {
	act.mux.Lock()
	defer act.mux.Unlock()

	delete(act.connections, id)
}
