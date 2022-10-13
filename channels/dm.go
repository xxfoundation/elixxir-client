////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"crypto/sha512"

	"filippo.io/edwards25519"

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

	net Client
	rng *fastRNG.StreamGenerator
}

func fromPublicEdwards(publicKey ed25519.PublicKey) *ecdh.PublicKey {
	ed_pub, _ := new(edwards25519.Point).SetBytes(publicKey)
	r := new(ecdh.PublicKey)
	r.FromBytes(ed_pub.BytesMontgomery())
	return r
}

func fromPrivateEdwards(privateKey ed25519.PrivateKey) *ecdh.PrivateKey {
	dhBytes := sha512.Sum512(privateKey[:32])
	dhBytes[0] &= 248
	dhBytes[31] &= 127
	dhBytes[31] |= 64
	r := new(ecdh.PrivateKey)
	r.FromBytes(dhBytes[:32])
	return r
}

func NewDMClient(privateEdwardsKey *ed25519.PrivateKey) {

}

// RegisterListener registers a listener for broadcast messages.
func (dc *dmClient) RegisterListener(listenerCb ListenerFunc) error {
	p := &processor{
		c:      dc,
		cb:     listenerCb,
		method: method,
	}

	service := message.Service{
		Identifier: dc.ReceptionID.Bytes(),
		Tag:        directMessageServiceTag,
	}

	dc.net.AddService(dc.ReceptionID, service, p)
	return nil
}
