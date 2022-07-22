///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/crypto/contact"
)

// authenticatedConnectionTrackerSingleton is used to track connections so that
// they can be referenced by ID back over the bindings.
var authenticatedConnectionTrackerSingleton = &authenticatedConnectionTracker{
	connections: make(map[int]*AuthenticatedConnection),
	count:       0,
}

type AuthenticatedConnection struct {
	Connection
}

func (_ *AuthenticatedConnection) IsAuthenticated() bool {
	return true
}

// ConnectWithAuthentication is called by the client (i.e., the one establishing
// connection with the server). Once a connect.Connection has been established
// with the server and then authenticate their identity to the server.
// accepts a marshalled ReceptionIdentity and contact.Contact object
func (c *Cmix) ConnectWithAuthentication(e2eId int, recipientContact,
	e2eParamsJSON []byte) (*AuthenticatedConnection, error) {
	if len(e2eParamsJSON) == 0 {
		jww.WARN.Printf("e2e params not specified, using defaults...")
		e2eParamsJSON = GetDefaultE2EParams()
	}

	cont, err := contact.Unmarshal(recipientContact)
	if err != nil {
		return nil, err
	}

	e2eClient, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return nil, err
	}

	params, err := parseE2EParams(e2eParamsJSON)
	if err != nil {
		return nil, err
	}

	connection, err := connect.ConnectWithAuthentication(cont,
		e2eClient.api, params)
	return authenticatedConnectionTrackerSingleton.make(connection), err
}

// authenticatedConnectionTracker is a singleton used to keep track of extant
// clients, allowing for race condition-free passing over the bindings.
type authenticatedConnectionTracker struct {
	connections map[int]*AuthenticatedConnection
	count       int
	mux         sync.RWMutex
}

// make makes a client from an API client, assigning it a unique ID
func (act *authenticatedConnectionTracker) make(
	c connect.AuthenticatedConnection) *AuthenticatedConnection {
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

// get returns a client given its ID.
func (act *authenticatedConnectionTracker) get(id int) (
	*AuthenticatedConnection, error) {
	act.mux.RLock()
	defer act.mux.RUnlock()

	c, exist := act.connections[id]
	if !exist {
		return nil, errors.Errorf("Cannot get client for ID %d, client "+
			"does not exist", id)
	}

	return c, nil
}

// delete deletes a client, if it exists.
func (act *authenticatedConnectionTracker) delete(id int) {
	act.mux.Lock()
	defer act.mux.Unlock()

	delete(act.connections, id)
}
