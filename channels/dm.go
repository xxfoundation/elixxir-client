////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"

	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike/ecdh"
)

const (
	DirectMessage           = "direct_message"
	directMessageServiceTag = "direct_message"
)

type dmClient struct {
	receptionID *id.ID
	privateKey  *ed25519.PrivateKey

	net Client
	rng *fastRNG.StreamGenerator
}

func NewDMClient(privateEdwardsKey ed25519.PrivateKey) *dmClient {
	privateKey := ecdh.ECDHNIKE.NewEmptyPrivateKey()
	privateKey.FromEdwards(privateEdwardsKey)
	publicKey := privateKey.DerivePublicKey()

	hash := blake2b.Sum256(publicKey.Bytes())
	receptionID, err := id.Unmarshal(hash[:])
	if err != nil {
		jww.FATAL.Panic(err)
	}

	return &dmClient{
		receptionID: receptionID,
		privateKey:  alicePrivateKey,
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
