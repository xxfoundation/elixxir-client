////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"time"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// ListenerFunc is registered when creating a new broadcasting channel and
// receives all new broadcast messages for the channel.
type ListenerFunc func(payload, encryptedPayload []byte, tags []string,
	metadata [2]byte, receptionID receptionID.EphemeralIdentity,
	round rounds.Round)

// Channel is the public-facing interface to interact with broadcast channels.
type Channel interface {
	// MaxPayloadSize returns the maximum size for a symmetric broadcast
	// payload.
	MaxPayloadSize() int

	// MaxRSAToPublicPayloadSize returns the maximum size for an asymmetric
	// broadcast payload.
	MaxRSAToPublicPayloadSize() int

	// Get returns the underlying broadcast.Channel object.
	Get() *crypto.Channel

	// Broadcast broadcasts a payload to the channel. The payload must be of the
	// size Channel.MaxPayloadSize or smaller.
	//
	// The network must be healthy to send.
	Broadcast(payload []byte, tags []string, metadata [2]byte,
		cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error)

	// BroadcastWithAssembler broadcasts a payload over a channel with a payload
	// assembled after the round is selected, allowing the round info to be
	// included in the payload.
	//
	// The payload must be of the size Channel.MaxPayloadSize or smaller.
	//
	// The network must be healthy to send.
	BroadcastWithAssembler(assembler Assembler, tags []string, metadata [2]byte,
		cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error)

	// BroadcastRSAtoPublic broadcasts the payload to the channel.
	//
	// The payload must be of the size Channel.MaxRSAToPublicPayloadSize or
	// smaller and the channel rsa.PrivateKey must be passed in.
	//
	// The network must be healthy to send.
	BroadcastRSAtoPublic(pk rsa.PrivateKey, payload []byte, tags []string,
		metadata [2]byte, cMixParams cmix.CMIXParams) (
		[]byte, rounds.Round, ephemeral.Id, error)

	// BroadcastRSAToPublicWithAssembler broadcasts the payload to the channel
	// with a function that builds the payload based upon the ID of the selected
	// round.
	//
	// The payload must be of the size Channel.MaxRSAToPublicPayloadSize or
	// smaller and the channel rsa.PrivateKey must be passed in.
	//
	// The network must be healthy to send.
	BroadcastRSAToPublicWithAssembler(pk rsa.PrivateKey, assembler Assembler,
		tags []string, metadata [2]byte, cMixParams cmix.CMIXParams) (
		[]byte, rounds.Round, ephemeral.Id, error)

	// RegisterRSAtoPublicListener registers a listener for asymmetric broadcast messages.
	// Note: only one Asymmetric Listener can be registered at a time.
	// Registering a new one will overwrite the old one
	RegisterRSAtoPublicListener(listenerCb ListenerFunc, tags []string) (
		Processor, error)

	// RegisterSymmetricListener registers a listener for asymmetric broadcast messages.
	// Note: only one Asymmetric Listener can be registered at a time.
	// Registering a new one will overwrite the old one
	RegisterSymmetricListener(listenerCb ListenerFunc, tags []string) (
		Processor, error)

	// Stop unregisters the listener callback and stops the channel's identity
	// from being tracked.
	Stop()

	// AsymmetricIdentifier returns a copy of the asymmetric identifier.
	AsymmetricIdentifier() []byte

	// SymmetricIdentifier returns a copy of the symmetric identifier.
	SymmetricIdentifier() []byte
}

// Processor handles channel message decryption and handling.
type Processor interface {
	message.Processor

	// ProcessAdminMessage decrypts an admin message and sends the results on
	// the callback.
	ProcessAdminMessage(innerCiphertext []byte, tags []string, metadata [2]byte,
		receptionID receptionID.EphemeralIdentity, round rounds.Round)
}

// Assembler assembles the message to send using the provided round ID.
type Assembler func(rid id.Round) (payload []byte, err error)

// Client contains the methods from [cmix.Client] that are required by the
// broadcastClient.
type Client interface {
	SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
		cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error)
	IsHealthy() bool
	AddIdentityWithHistory(
		id *id.ID, validUntil, beginning time.Time, persistent bool,
		fallthroughProcessor message.Processor)
	UpsertCompressedService(clientID *id.ID, newService message.CompressedService,
		response message.Processor)
	DeleteClientService(clientID *id.ID)
	RemoveIdentity(id *id.ID)
	GetMaxMessageLength() int
}
