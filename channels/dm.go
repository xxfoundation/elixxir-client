////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"

	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike/ecdh"
)

const (
	DirectMessage           = "direct_message"
	directMessageServiceTag = "direct_message"
)

type dmClient struct {
	ReceptionID *id.ID
	privateKey  *ed25519.PrivateKey

	net Client
	rng *fastRNG.StreamGenerator
}

func NewDMClient(privateEdwardsKey ed25519.PrivateKey) *dmClient {
	alicePrivateKey := ecdh.ECDHNIKE.NewEmptyPrivateKey()
	alicePrivateKey.FromEdwards(privateEdwardsKey)

	return &dmClient{
		privateKey: alicePrivateKey,
	}
}

// RegisterListener registers a listener for broadcast messages.
func (dc *dmClient) RegisterListener(listenerCb ListenerFunc) error {
	p := &processor{
		c:  dc,
		cb: listenerCb,
	}

	service := message.Service{
		Identifier: dc.ReceptionID.Bytes(),
		Tag:        directMessageServiceTag,
	}

	dc.net.AddService(dc.ReceptionID, service, p)
	return nil
}
