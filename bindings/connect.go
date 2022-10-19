////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"sync"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/connect"
	xxdk "gitlab.com/elixxir/client/xxdk2"
)

// connectionTrackerSingleton is used to track connections so that they can be
// referenced by ID back over the bindings.
var connectionTrackerSingleton = &connectionTracker{
	connections: make(map[int]*Connection),
	count:       0,
}

// Connection is the bindings' representation of a connect.Connection object
// that can be tracked by ID.
type Connection struct {
	connection connect.Connection
	id         int
	params     xxdk.E2EParams
}

// GetId returns the Connection ID.
func (c *Connection) GetId() int {
	return c.id
}

// connectionTracker is a singleton used to keep track of extant connections,
// allowing for race condition-free passing over the bindings.
type connectionTracker struct {
	connections map[int]*Connection
	count       int
	mux         sync.RWMutex
}

// make makes a Connection, assigning it a unique ID.
func (ct *connectionTracker) make(
	c connect.Connection, params xxdk.E2EParams) *Connection {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	id := ct.count
	ct.count++

	ct.connections[id] = &Connection{
		connection: c,
		id:         id,
		params:     params,
	}

	return ct.connections[id]
}

// get returns a Connection given its ID.
func (ct *connectionTracker) get(id int) (*Connection, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.connections[id]
	if !exist {
		return nil, errors.Errorf("Cannot get Connection for ID %d, "+
			"does not exist", id)
	}

	return c, nil
}

// delete deletes a Connection.
func (ct *connectionTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.connections, id)
}
