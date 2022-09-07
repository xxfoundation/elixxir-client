////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/xx_network/crypto/multicastRSA"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

// ListenerFunc is registered when creating a new broadcasting channel and
// receives all new broadcast messages for the channel.
type ListenerFunc func(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round)

// Channel is the public-facing interface to interact with broadcast channels
type Channel interface {
	// MaxPayloadSize returns the maximum size for a symmetric broadcast payload
	MaxPayloadSize() int

	// MaxAsymmetricPayloadSize returns the maximum size for an asymmetric
	// broadcast payload
	MaxAsymmetricPayloadSize() int

	// Get returns the underlying crypto.Channel
	Get() *crypto.Channel

	// Broadcast broadcasts the payload to the channel. The payload size must be
	// equal to MaxPayloadSize.
	Broadcast(payload []byte, cMixParams cmix.CMIXParams) (
		id.Round, ephemeral.Id, error)

	// BroadcastWithAssembler broadcasts a payload over a symmetric channel.
	// With a payload assembled after the round is selected, allowing the round
	// info to be included in the payload. Network must be healthy to send.
	// Requires a payload of size bc.MaxSymmetricPayloadSize()
	BroadcastWithAssembler(assembler Assembler, cMixParams cmix.CMIXParams) (
		id.Round, ephemeral.Id, error)

	// BroadcastAsymmetric broadcasts the payload to the channel. Requires a
	// healthy network state to send. Payload length must be equal to
	// bc.MaxAsymmetricPayloadSize and the channel PrivateKey must be passed in
	BroadcastAsymmetric(pk multicastRSA.PrivateKey, payload []byte,
		cMixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error)

	// BroadcastAsymmetricWithAssembler broadcasts the payload to the channel.
	// Requires a healthy network state to send. Payload length must be equal to
	// bc.MaxAsymmetricPayloadSize and the channel PrivateKey must be passed in.
	// The assembler will run once a round is selected and will receive the
	// round ID
	BroadcastAsymmetricWithAssembler(
		pk multicastRSA.PrivateKey, assembler Assembler,
		cMixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error)

	// RegisterListener registers a listener for broadcast messages
	RegisterListener(listenerCb ListenerFunc, method Method) error

	// Stop unregisters the listener callback and stops the channel's identity
	// from being tracked.
	Stop()
}

// Assembler is a function which allows a bre
type Assembler func(rid id.Round) (payload []byte, err error)

// Client contains the methods from cmix.Client that are required by
// symmetricClient.
type Client interface {
	GetMaxMessageLength() int
	SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
		cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error)
	IsHealthy() bool
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteClientService(clientID *id.ID)
	RemoveIdentity(id *id.ID)
}
