////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"

	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/v4/cmix/message"
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
func NewDMClient(privateEdwardsKey ed25519.PrivateKey,
	myIDToken []byte, nickMgr nickNameManager,
	net cMixClient,
	rng *fastRNG.StreamGenerator) *dmClient {

	privateKey := ecdh.Edwards2ECDHNIKEPrivateKey(&privateEdwardsKey)
	publicKey := ecdh.ECDHNIKE.DerivePublicKey(privateKey)

	receptionID := deriveReceptionID(publicKey, myIDToken)

	// TODO: do we do the reception registration here or do we do
	// it outside of this function?

	return &dmClient{
		receptionID: receptionID,
		privateKey:  privateKey,
		publicKey:   publicKey,
		myToken:     myIDToken,
		nm:          nickMgr,
		net:         net,
		rng:         rng,
	}
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
