////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
)

type dmClient struct {
	receptionID *id.ID
	privateKey  nike.PrivateKey
	publicKey   nike.PublicKey
	myToken     []byte

	nm  nickNameManager
	net cMixClient
	rng *fastRNG.StreamGenerator
}

// NewDMClient creates a new client for direct messaging. This should
// be called when the channels manager is created/loaded. It has no
// associated state, so it does not have a corresponding Load
// function.
//
// The DMClient implements both the Sender and ListenerRegistrar interface.
// See send.go for implementation of the Sender interface.
func NewDMClient(myID codename.PrivateIdentity, receiver Receiver,
	nickMgr nickNameManager,
	net cMixClient,
	rng *fastRNG.StreamGenerator) Client {

	privateEdwardsKey := myID.Privkey
	myIDToken := myID.GetDMToken()

	privateKey := ecdh.Edwards2ECDHNIKEPrivateKey(privateEdwardsKey)
	publicKey := ecdh.ECDHNIKE.DerivePublicKey(privateKey)

	receptionID := deriveReceptionID(publicKey, myIDToken)

	// TODO: do we do the reception registration here or do we do
	// it outside of this function?

	dmc := &dmClient{
		receptionID: receptionID,
		privateKey:  privateKey,
		publicKey:   publicKey,
		myToken:     myIDToken,
		nm:          nickMgr,
		net:         net,
		rng:         rng,
	}

	// Register the listener
	// TODO: For now we are not doing send tracking. Add it when
	// hitting WASM.
	dmc.Register(receiver, func(
		messageID MessageID, r rounds.Round) bool {
		return false
	})

	return dmc
}

// Register registers a listener for direct messages.
func (dc *dmClient) Register(apiReceiver Receiver,
	checkSent messageReceiveFunc) error {
	p := &receiver{
		c:         dc,
		api:       apiReceiver,
		checkSent: checkSent,
	}

	service := message.Service{
		Identifier: dc.receptionID.Bytes(),
		Tag:        directMessageServiceTag,
	}

	dc.net.AddService(dc.receptionID, service, p)
	return nil
}
