////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"gitlab.com/elixxir/client/v4/cmix/identity"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

func NewServer(net cMixClient,
	rng *fastRNG.StreamGenerator,
	receptionID *id.ID,
	privateKey nike.PrivateKey) Server {
	rpc := &rpcServer{
		me:         receptionID,
		privateKey: privateKey,
		publicKey:  privateKey.Scheme().DerivePublicKey(privateKey),
		net:        net,
		rng:        rng,
		listener:   make(chan Request, listenChSize),
		cbs:        make(map[int]Callback),
		ctr:        0,
	}

	return rpc
}

func (r *rpcServer) Start() {
	r.net.AddIdentityWithHistory(r.me, identity.Forever,
		beginningOfTime, true, r)
}

func (r *rpcServer) Stop() {
	r.net.RemoveIdentity(r.me)
}

type rpcServer struct {
	me         *id.ID
	privateKey nike.PrivateKey
	publicKey  nike.PublicKey

	net      cMixClient
	rng      *fastRNG.StreamGenerator
	listener chan Request
	cbs      map[int]Callback
	ctr      int
}

func (r *rpcServer) AddCallback(cbFn Callback) int {
	i := r.ctr
	r.cbs[i] = cbFn
	r.ctr += 1
	return i
}

func (r *rpcServer) DeleteCallback(i int) bool {
	_, ok := r.cbs[i]
	return ok
}

////
// [message.Processor] Interface
////

func (r *rpcServer) String() string {
	return "rpcServer"
}

func (r *rpcServer) Process(msg format.Message, _ []string, _ []byte,
	ephID receptionID.EphemeralIdentity, round rounds.Round) {
	ciphertext := reconstructCiphertext(msg)
	// Decrypt with r.privateKey
	// Construct request
	// Iterate over callbacks, creating a goroutine for each.
	// When all goroutines exit, then exit this function.
}

// This helper does the opposite of "createCMIXFields" in send.go
func reconstructCiphertext(msg format.Message) []byte {
	var res []byte
	fp := msg.GetKeyFP()
	res = append(res, fp[1:]...)
	res = append(res, msg.GetMac()[1:]...)
	res = append(res, msg.GetContents()...)
	return res
}
