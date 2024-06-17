///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"fmt"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/rpc"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/primitives/id"
)

// RPCResponseCallbacks implements the callback functions for an RPCResponse
// This is required because of gomobile restrictions.
type RPCResponseCallbacks interface {
	Response(response []byte)
	Error(errorStr string)
}

// RPCServerCallback implements the server callback function
// This is required due to gomobile restrictions.
type RPCServerCallback interface {
	Callback(sender, request []byte) []byte
}

// Response interface for RPC responses from the [Send]
// function. Provides for the ability to call a callback on error or
// success or to listen on the channels directly.
type RPCResponse interface {
	// Callback calls success or error function if/when the
	// response is received. Meant to be very similar to Promise/async
	// found in other languages.
	// RPC will call respFn 3 times: sent, round finished, and
	// response received.
	Callback(cbs RPCResponseCallbacks) RPCResponse
	// Wait waits until the response is complete returns the final
	// response bytes
	Wait() []byte
}

// RPCServer is the interface for the RPC server, users should
// add a callback then start listening for RPC requests.
type RPCServer interface {
	// Start listening on the server's cMix Identity
	Start()
	// Stop listening on the server's cMix Identity
	// Note: if you want to stop all network traffic, do
	// so with the cMix network client.
	Stop()
}

// RPCSend sends an RPC `request` to the provided server `recipient` and
// `pubkey`. It returns a `Response` Object which
func RPCSend(cMixID int, recipient, pubkey, request []byte) RPCResponse {
	net, err := cmixTrackerSingleton.get(cMixID)
	if err != nil {
		return errResponse(err)
	}

	serverID, err := id.Unmarshal(recipient)
	if err != nil {
		return errResponse(err)
	}

	serverKey := ecdh.ECDHNIKE.NewEmptyPublicKey()
	err = serverKey.FromBytes(pubkey)
	if err != nil {
		return errResponse(err)
	}

	params := cmix.GetDefaultCMIXParams()

	res := rpc.Send(net.api.GetCmix(), serverID, serverKey, request, params)
	return &rpcResponse{response: res}
}

func errResponse(err error) *rpcResponse {
	return &rpcResponse{err: err}
}

type rpcResponse struct {
	response rpc.Response
	err      error
}

func (r *rpcResponse) Callback(cbs RPCResponseCallbacks) RPCResponse {
	if r.err != nil {
		cbs.Error(fmt.Sprintf("%+v", r.err))
		return r
	}
	r.response.Callback(cbs.Response,
		func(err error) {
			cbs.Error(fmt.Sprintf("%+v", err))
		})
	return r
}

func (r *rpcResponse) Wait() []byte {
	return r.response.Wait()
}

// NewRPCServer returns a new RPC server with the specified
// reception ID and private Key
func NewRPCServer(cMixID int, callback RPCServerCallback,
	receptionID, privateKey []byte) (RPCServer, error) {
	net, err := cmixTrackerSingleton.get(cMixID)
	if err != nil {
		return nil, err
	}

	serverID, err := id.Unmarshal(receptionID)
	if err != nil {
		return nil, err
	}

	serverKey := ecdh.ECDHNIKE.NewEmptyPrivateKey()
	err = serverKey.FromBytes(privateKey)
	if err != nil {
		return nil, err
	}

	cbFn := func(id *id.ID, request []byte) []byte {
		idBytes := id.Marshal()
		return callback.Callback(idBytes, request)
	}

	net.EKVSet("rpcServerID", receptionID)
	net.EKVSet("rpcServerKey", privateKey)

	server := rpc.NewServer(net.api.GetCmix(), cbFn, serverID, serverKey)
	return server, nil
}

// GenerateRandomReceptionID generates a random well formed reception id
func GenerateRandomReceptionID(cMixID int) ([]byte, error) {
	net, err := cmixTrackerSingleton.get(cMixID)
	if err != nil {
		return nil, err
	}
	newID, err := rpc.GenerateRandomID(net.api.GetCmix())
	if err != nil {
		return nil, err
	}
	return newID.Marshal(), nil
}

// GenerateRandomRPCKey generates a random private key for a server
func GenerateRandomRPCKey(cMixID int) ([]byte, error) {
	net, err := cmixTrackerSingleton.get(cMixID)
	if err != nil {
		return nil, err
	}

	rng := net.api.GetRng().GetStream()
	defer rng.Close()
	prk, _ := ecdh.ECDHNIKE.NewKeypair(rng)
	return prk.Bytes(), nil
}

// LoadRPCServer load key and id from disk and return an RPC server
func LoadRPCServer(cMixID int, callback RPCServerCallback) (
	RPCServer, error) {
	net, err := cmixTrackerSingleton.get(cMixID)
	if err != nil {
		return nil, err
	}

	serverID, err := net.EKVGet("rpcServerID")
	if err != nil {
		return nil, err
	}
	serverKey, err := net.EKVGet("rpcServerKey")
	if err != nil {
		return nil, err
	}

	return NewRPCServer(cMixID, callback, serverID, serverKey)
}
