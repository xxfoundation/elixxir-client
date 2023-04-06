////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/connect"
	"gitlab.com/elixxir/client/v4/restlike"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/xx_network/primitives/id"
)

// Server implements the RestServer interface using connect.Connection
type Server struct {
	receptionId   *id.ID
	endpoints     *restlike.Endpoints
	ConnectServer *connect.ConnectionServer
}

// NewServer builds a RestServer with connect.Connection and
// the provided arguments, then registers necessary external services
func NewServer(identity xxdk.ReceptionIdentity, net *xxdk.Cmix,
	p xxdk.E2EParams, clParams connect.ConnectionListParams) (*Server, error) {
	var err error
	newServer := &Server{
		receptionId: identity.ID,
		endpoints:   restlike.NewEndpoints(),
	}

	// Callback for connection requests
	cb := func(conn connect.Connection) {
		if conn == nil {
			jww.ERROR.Printf("nill connection")
		}
		if newServer == nil {
			jww.ERROR.Printf("nil server!")
		}
		handler := receiver{endpoints: newServer.endpoints, conn: conn}
		conn.RegisterListener(catalog.XxMessage, handler)
	}

	// Build the connection listener
	newServer.ConnectServer, err = connect.StartServer(identity, cb, net, p, clParams)
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
