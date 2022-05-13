////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/restlike"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

// Server implements the RestServer interface using connect.Connection
type Server struct {
	receptionId *id.ID
	endpoints   *restlike.Endpoints
}

// NewServer builds a RestServer with connect.Connection and
// the provided arguments, then registers necessary external services
func NewServer(receptionId *id.ID, privKey *cyclic.Int,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, net cmix.Client, p connect.Params) (*Server, error) {
	newServer := &Server{
		receptionId: receptionId,
		endpoints:   restlike.NewEndpoints(),
	}

	// Callback for connection requests
	cb := func(conn connect.Connection) {
		handler := connectReceiver{endpoints: newServer.endpoints, conn: conn}
		conn.RegisterListener(catalog.XxMessage, handler)
	}

	// Build the connection listener
	err := connect.StartServer(cb, receptionId, privKey, rng, grp, net, p)
	if err != nil {
		return nil, err
	}
	return newServer, nil
}

// GetEndpoints returns the association of a Callback with
// a specific URI and a variety of different REST Method
func (c *Server) GetEndpoints() *restlike.Endpoints {
	return c.endpoints
}

// Close the internal RestServer endpoints and external services
func (c *Server) Close() {
	// Clear all internal endpoints
	c.endpoints = nil
	// TODO: Destroy external services
}
