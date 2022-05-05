////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

// Error messages.
const (
	// symmetricClient.Broadcast
	errNetworkHealth = "cannot send broadcast when the network is not healthy"
	errPayloadSize   = "size of payload %d must be %d"
)

// Tags.
const (
	cMixSendTag                  = "SymmBcast"
	symmetricBroadcastServiceTag = "SymmetricBroadcast"
)

// symmetricClient manages the sending and receiving of symmetric broadcast
// messages on a given symmetric broadcast channel. Adheres to the Symmetric
// interface.
type symmetricClient struct {
	channel crypto.Symmetric
	net     Client
	rng     *fastRNG.StreamGenerator
}

// Client contains the methods from cmix.Client that are required by
// symmetricClient.
type Client interface {
	GetMaxMessageLength() int
	Send(recipient *id.ID, fingerprint format.Fingerprint,
		service message.Service, payload, mac []byte,
		cMixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error)
	IsHealthy() bool
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteClientService(clientID *id.ID)
	RemoveIdentity(id *id.ID)
}

// NewSymmetricClient generates a new Symmetric for the given channel. It starts
// listening for new messages on the callback immediately.
func NewSymmetricClient(channel crypto.Symmetric, listenerCb ListenerFunc,
	net Client, rng *fastRNG.StreamGenerator) Symmetric {
	sc := &symmetricClient{
		channel: channel,
		net:     net,
		rng:     rng,
	}
	if !sc.verifyID() {
		jww.FATAL.Panicf("Failed ID verification for symmetric channel")
	}

	// Add channel's identity
	net.AddIdentity(channel.ReceptionID, identity.Forever, true)

	// Create new service
	service := message.Service{
		Identifier: channel.ReceptionID.Bytes(),
		Tag:        symmetricBroadcastServiceTag,
	}

	// Create new message processor
	p := &processor{
		s:  &channel,
		cb: listenerCb,
	}

	// Add service
	net.AddService(channel.ReceptionID, service, p)

	jww.INFO.Printf("New symmetric broadcast client created for channel %q (%s)",
		channel.Name, channel.ReceptionID)

	return sc
}

// MaxPayloadSize returns the maximum size for a broadcasted payload.
func (s *symmetricClient) MaxPayloadSize() int {
	return s.net.GetMaxMessageLength()
}

// Get returns the crypto.Symmetric object containing the cryptographic and
// identifying information about the channel.
func (s *symmetricClient) Get() crypto.Symmetric {
	return s.channel
}

// Broadcast broadcasts the payload to the channel.
func (s *symmetricClient) Broadcast(payload []byte, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	if !s.net.IsHealthy() {
		return 0, ephemeral.Id{}, errors.New(errNetworkHealth)
	}

	if len(payload) != s.MaxPayloadSize() {
		return 0, ephemeral.Id{},
			errors.Errorf(errPayloadSize, len(payload), s.MaxPayloadSize())
	}

	// Encrypt payload
	rng := s.rng.GetStream()
	encryptedPayload, mac, fp := s.channel.Encrypt(payload, rng)
	rng.Close()

	// Create service
	service := message.Service{
		Identifier: s.channel.ReceptionID.Bytes(),
		Tag:        symmetricBroadcastServiceTag,
	}

	if cMixParams.DebugTag == cmix.DefaultDebugTag {
		cMixParams.DebugTag = cMixSendTag
	}

	return s.net.Send(
		s.channel.ReceptionID, fp, service, encryptedPayload, mac, cMixParams)
}

// Stop unregisters the listener callback and stops the channel's identity
// from being tracked.
func (s *symmetricClient) Stop() {
	// Removes currently tracked identity
	s.net.RemoveIdentity(s.channel.ReceptionID)

	// Delete all registered services
	s.net.DeleteClientService(s.channel.ReceptionID)
}

func (s *symmetricClient) verifyID() bool {
	h, err := hash.NewCMixHash()
	if err != nil {
		jww.FATAL.Panicf("[verifyID] Failed to create cmix hash")
		return false
	}
	h.Write([]byte(s.channel.Name))
	h.Write([]byte(s.channel.Description))
	h.Write(s.channel.Salt)
	h.Write(rsa.CreatePublicKeyPem(s.channel.RsaPubKey))
	ridBytes := h.Sum(nil)
	gen := &id.ID{}
	copy(gen[:], ridBytes)
	gen.SetType(id.User)
	return s.channel.ReceptionID.Cmp(gen)
}
