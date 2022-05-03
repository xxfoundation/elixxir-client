////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// RestServer allows for clients to make REST-like requests this client
type RestServer interface {
	// RegisterEndpoint allows the association of a Callback with
	// a specific URI and a variety of different REST Method
	RegisterEndpoint(path URI, method Method, cb Callback) error

	// UnregisterEndpoint removes the Callback associated with
	// a specific URI and REST Method
	UnregisterEndpoint(path URI, method Method) error

	// Close the internal RestServer endpoints and external services
	Close()
}

// singleServer implements the RestServer interface using single-use
type singleServer struct {
	receptionId *id.ID
	listener    single.Listener
	endpoints   Endpoints
}

// NewSingleServer builds a RestServer with single-use and
// the provided arguments, then registers necessary external services
func NewSingleServer(receptionId *id.ID, privKey *cyclic.Int, net single.ListenCmix, e2eGrp *cyclic.Group) RestServer {
	newServer := &singleServer{
		receptionId: receptionId,
		endpoints:   make(map[URI]map[Method]Callback),
	}
	newServer.listener = single.Listen(catalog.RestLike, receptionId, privKey,
		net, e2eGrp, &singleReceiver{newServer.endpoints})
	return newServer
}

// RegisterEndpoint allows the association of a Callback with
// a specific URI and a variety of different REST Method
func (r *singleServer) RegisterEndpoint(path URI, method Method, cb Callback) error {
	return r.endpoints.Add(path, method, cb)
}

// UnregisterEndpoint removes the Callback associated with
// a specific URI and REST Method
func (r *singleServer) UnregisterEndpoint(path URI, method Method) error {
	return r.endpoints.Remove(path, method)
}

// Close the internal RestServer endpoints and external services
func (r *singleServer) Close() {
	// Clear all internal endpoints
	r.endpoints = make(map[URI]map[Method]Callback)
	// Destroy external services
	r.listener.Stop()
}
