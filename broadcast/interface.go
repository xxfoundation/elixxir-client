////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/xx_network/crypto/multicastRSA"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// ListenerFunc is registered when creating a new broadcasting channel
// and receives all new broadcast messages for the channel.
type ListenerFunc func(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round)

// Symmetric manages the listening and broadcasting of a symmetric broadcast
// channel.
type Symmetric interface {
	// MaxPayloadSize returns the maximum size for a broadcasted payload.
	MaxPayloadSize() int

	// Get returns the crypto.Symmetric object containing the cryptographic and
	// identifying information about the channel.
	Get() crypto.Symmetric

	// Broadcast broadcasts the payload to the channel. The payload size must be
	// equal to MaxPayloadSize.
	Broadcast(payload []byte, cMixParams cmix.CMIXParams) (
		id.Round, ephemeral.Id, error)

	// Stop unregisters the listener callback and stops the channel's identity
	// from being tracked.
	Stop()
}

// Asymmetric manages the listening and broadcasting of an asymmetric broadcast
// channel.
type Asymmetric interface {
	// MaxPayloadSize returns the maximum size for a broadcasted payload.
	MaxPayloadSize() int

	// Get returns the crypto.Asymmetric object containing the cryptographic and
	// identifying information about the channel.
	Get() crypto.Asymmetric

	// Broadcast broadcasts the payload to the channel. The payload size must be
	// equal to MaxPayloadSize.
	Broadcast(pk multicastRSA.PrivateKey, payload []byte, cMixParams cmix.CMIXParams) (
		id.Round, ephemeral.Id, error)

	// Stop unregisters the listener callback and stops the channel's identity
	// from being tracked.
	Stop()
}
