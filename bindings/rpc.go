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

type RPCResponse interface {
	Callback(responseFn func(response []byte),
		errorFn func(errorStr string))
	Wait() []byte
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

func (r *rpcResponse) Callback(responseFn func(response []byte),
	errorFn func(errorStr string)) {
	if r.err != nil {
		errorFn(fmt.Sprintf("%+v", r.err))
		return
	}
	r.response.Callback(responseFn,
		func(err error) {
			errorFn(fmt.Sprintf("%+v", err))
		})
}

func (r *rpcResponse) Wait() []byte {
	return r.response.Wait()
}
