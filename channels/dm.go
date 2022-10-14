////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"golang.org/x/crypto/blake2b"

	"gitlab.com/elixxir/client/cmix"
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

func DeriveReceptionID(publicKey nike.PublicKey, idToken []byte) *id.ID {
	h, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	h.Write(publicKey.Bytes())
	h.Write(idToken)
	idBytes := h.Sum(nil)
	receptionID, err := id.Unmarshal(idBytes)
	if err != nil {
		panic(err)
	}
	return receptionID
}

type dmClient struct {
	receptionID        *id.ID
	partnerReceptionID *id.ID
	privateKey         nike.PrivateKey
	partnerPubKey      nike.PublicKey

	net Client
	rng *fastRNG.StreamGenerator
}

func NewDMClient(privateEdwardsKey ed25519.PrivateKey,
	partnerPublicKey ed25519.PublicKey,
	myIdToken, partnerIdToken []byte,
	net Client,
	rng *fastRNG.StreamGenerator) *dmClient {

	privateKey := ecdh.ECDHNIKE.NewEmptyPrivateKey()
	privateKey.(*ecdh.PrivateKey).FromEdwards(privateEdwardsKey)
	publicKey := ecdh.ECDHNIKE.DerivePublicKey(privateKey)

	partnerPubKey := ecdh.ECDHNIKE.NewEmptyPublicKey()
	partnerPubKey.(*ecdh.PublicKey).FromEdwards(partnerPublicKey)

	partnerReceptionID := DeriveReceptionID(partnerPubKey, partnerIdToken)
	receptionID := DeriveReceptionID(publicKey, myIdToken)

	return &dmClient{
		receptionID:        receptionID,
		partnerReceptionID: partnerReceptionID,
		privateKey:         privateKey,
		partnerPubKey:      partnerPubKey,
		net:                net,
		rng:                rng,
	}
}

func (dc *dmClient) SendMessage(plaintext []byte, cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {

	assembler := func(rid id.Round) ([]byte, error) {
		return plaintext, nil
	}

	assemble := func(rid id.Round) (fp format.Fingerprint,
		service message.Service, encryptedPayload, mac []byte, err error) {
		payload, err := assembler(rid)
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, err
		}
		service = message.Service{
			Identifier: dc.receptionID.Bytes(),
			Tag:        directMessageServiceTag,
		}

		if cMixParams.DebugTag == cmix.DefaultDebugTag {
			cMixParams.DebugTag = directMessageServiceTag
		}

		// Create payload sized for sending over cmix
		sizedPayload := make([]byte, dc.net.GetMaxMessageLength())
		// Read random data into sized payload
		_, err = dc.rng.GetStream().Read(sizedPayload)
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, errors.WithMessage(err, "Failed to add "+
					"random data to sized broadcast")
		}
		encryptedPayload = dm.Cipher.Encrypt(payload, dc.privateKey, dc.partnerPubKey)
		copy(sizedPayload[:len(encryptedPayload)], encryptedPayload)

		return
	}

	return dc.net.SendWithAssembler(dc.partnerReceptionID,
		assemble,
		cMixParams)
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
