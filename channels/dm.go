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

	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/crypto/dm"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
)

const (
	DirectMessage           = "direct_message"
	directMessageServiceTag = "direct_message"
)

type dmClient struct {
	receptionID   *id.ID
	privateKey    nike.PrivateKey
	partnerPubKey nike.PublicKey

	net Client
	rng *fastRNG.StreamGenerator
}

func NewDMClient(privateEdwardsKey ed25519.PrivateKey, partnerPublicKey ed25519.PublicKey, net Client, rng *fastRNG.StreamGenerator) *dmClient {
	privateKey := ecdh.ECDHNIKE.NewEmptyPrivateKey()
	privateKey.(*ecdh.PrivateKey).FromEdwards(privateEdwardsKey)
	publicKey := ecdh.ECDHNIKE.DerivePublicKey(privateKey)

	partnerPubKey := ecdh.ECDHNIKE.NewEmptyPublicKey()
	partnerPubKey.(*ecdh.PublicKey).FromEdwards(partnerPublicKey)

	hash := blake2b.Sum256(publicKey.Bytes())
	receptionID, err := id.Unmarshal(hash[:])
	if err != nil {
		jww.FATAL.Panic(err)
	}

	return &dmClient{
		receptionID:   receptionID,
		privateKey:    privateKey,
		partnerPubKey: partnerPubKey,
		net:           net,
		rng:           rng,
	}
}

// RegisterListener registers a listener for broadcast messages.
func (dc *dmClient) RegisterListener(listenerCb ListenerFunc) error {
	p := &processor{
		c:  dc,
		cb: listenerCb,
	}

	service := message.Service{
		Identifier: dc.receptionID.Bytes(),
		Tag:        directMessageServiceTag,
	}

	dc.net.AddService(dc.receptionID, service, p)
	return nil
}

// ListenerFunc is registered when creating a new broadcasting channel and
// receives all new broadcast messages for the channel.
type ListenerFunc func(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round)

// processor struct for message handling
type processor struct {
	c  *dmClient
	cb ListenerFunc
}

// String returns a string identifying the symmetricProcessor for debugging purposes.
func (p *processor) String() string {
	return "directMessage-"
}

// Process decrypts the broadcast message and sends the results on the callback.
func (p *processor) Process(msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	payload, err := dm.Cipher.Decrypt(msg.GetContents(), p.c.privateKey, p.c.partnerPubKey)
	if err != nil {
		jww.ERROR.Printf("failed to decrypt direct message: %s", err)
		return
	}

	p.cb(payload, receptionID, round)
}
