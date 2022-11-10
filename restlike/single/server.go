////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/restlike"
	"gitlab.com/elixxir/client/v4/single"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// Server implements the RestServer interface using single-use
type Server struct {
	receptionId *id.ID
	listener    single.Listener
	endpoints   *restlike.Endpoints
}

// NewServer builds a RestServer with single-use and
// the provided arguments, then registers necessary external services
func NewServer(receptionId *id.ID, privKey *cyclic.Int, grp *cyclic.Group, net single.ListenCmix) *Server {
	newServer := &Server{
		receptionId: receptionId,
		endpoints:   restlike.NewEndpoints(),
	}
	newServer.listener = single.Listen(catalog.RestLike, receptionId, privKey,
		net, grp, &receiver{newServer.endpoints})
	return newServer
}

// GetEndpoints returns the association of a Callback with
// a specific URI and a variety of different REST Method
func (r *Server) GetEndpoints() *restlike.Endpoints {
	return r.endpoints
}

// Close the internal RestServer endpoints and external services
func (r *Server) Close() {
	// Clear all internal endpoints
	r.endpoints = nil
	// Destroy external services
	r.listener.Stop()
}
