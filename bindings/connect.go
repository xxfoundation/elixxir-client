///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
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

// Connect performs auth key negotiation with the given recipient and returns a
// Connection object for the newly created partner.Manager.
//
// This function is to be used sender-side and will block until the
// partner.Manager is confirmed.
//
// Parameters:
//  - e2eId - ID of the E2E object in the e2e tracker
//  - recipientContact - marshalled contact.Contact object
//  - myIdentity - marshalled ReceptionIdentity object
func (c *Cmix) Connect(e2eId int, recipientContact, e2eParamsJSON []byte) (
	*Connection, error) {
	if len(e2eParamsJSON) == 0 {
		jww.WARN.Printf("e2e params not specified, using defaults...")
		e2eParamsJSON = GetDefaultE2EParams()
	}
	cont, err := contact.Unmarshal(recipientContact)
	if err != nil {
		return nil, err
	}

	user, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return nil, err
	}

	p, err := parseE2EParams(e2eParamsJSON)
	if err != nil {
		return nil, err
	}

	connection, err := connect.Connect(cont, user.api, p)
	if err != nil {
		return nil, err
	}

	return connectionTrackerSingleton.make(connection, p), nil
}

// SendE2E is a wrapper for sending specifically to the Connection's
// partner.Manager.
//
// Returns:
//  - []byte - the JSON marshalled bytes of the E2ESendReport object, which can
//    be passed into WaitForRoundResult to see if the send succeeded.
func (c *Connection) SendE2E(mt int, payload []byte) ([]byte, error) {
	rounds, mid, ts, err := c.connection.SendE2E(catalog.MessageType(mt), payload,
		c.params.Base)

	if err != nil {
		return nil, err
	}

	sr := E2ESendReport{
		RoundsList: makeRoundsList(rounds...),
		MessageID:  mid.Marshal(),
		Timestamp:  ts.UnixNano(),
	}

	return json.Marshal(&sr)
}

// Close deletes this Connection's partner.Manager and releases resources.
func (c *Connection) Close() error {
	return c.connection.Close()
}

// GetPartner returns the partner.Manager for this Connection.
func (c *Connection) GetPartner() []byte {
	return c.connection.GetPartner().PartnerId().Marshal()
}

// RegisterListener is used for E2E reception and allows for reading data sent
// from the partner.Manager.
func (c *Connection) RegisterListener(messageType int, newListener Listener) error {
	_, err := c.connection.RegisterListener(
		catalog.MessageType(messageType), listener{l: newListener})
	return err
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
