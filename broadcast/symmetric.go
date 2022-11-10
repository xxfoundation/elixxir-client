////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v5/cmix"
	"gitlab.com/elixxir/client/v5/cmix/message"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// Error messages.
const (
	// broadcastClient.Broadcast
	errNetworkHealth = "cannot send broadcast when the network is not healthy"
	errPayloadSize   = "size of payload %d must be less than %d"
)

// Tags.
const (
	symmCMixSendTag              = "SymmBcast"
	symmetricBroadcastServiceTag = "SymmetricBroadcast"
)

// Broadcast broadcasts a payload to a symmetric channel. The payload must be of
// size [broadcastClient.MaxPayloadSize] or smaller.
//
// The network must be healthy to send.
func (bc *broadcastClient) Broadcast(payload []byte, cMixParams cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	assemble := func(rid id.Round) ([]byte, error) {
		return payload, nil
	}
	return bc.BroadcastWithAssembler(assemble, cMixParams)
}

// BroadcastWithAssembler broadcasts a payload over a symmetric channel with a
// payload assembled after the round is selected, allowing the round info to be
// included in the payload.
//
// The payload must be of the size [Channel.MaxPayloadSize] or smaller.
//
// The network must be healthy to send.
func (bc *broadcastClient) BroadcastWithAssembler(
	assembler Assembler, cMixParams cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	if !bc.net.IsHealthy() {
		return rounds.Round{}, ephemeral.Id{}, errors.New(errNetworkHealth)
	}

	assemble := func(rid id.Round) (fp format.Fingerprint,
		service message.Service, encryptedPayload, mac []byte, err error) {

		// Assemble the passed payload
		payload, err := assembler(rid)
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil, nil, err
		}

		if len(payload) > bc.maxSymmetricPayload() {
			return format.Fingerprint{}, message.Service{}, nil, nil,
				errors.Errorf(errPayloadSize, len(payload), bc.maxSymmetricPayload())
		}

		// Encrypt payload
		rng := bc.rng.GetStream()
		defer rng.Close()
		encryptedPayload, mac, fp, err = bc.channel.EncryptSymmetric(payload,
			bc.net.GetMaxMessageLength(), rng)
		if err != nil {
			return format.Fingerprint{}, message.Service{},
				nil, nil, err
		}

		// Create service using symmetric broadcast service tag & channel reception ID
		// Allows anybody with this info to listen for messages on this channel
		service = message.Service{
			Identifier: bc.channel.ReceptionID.Bytes(),
			Tag:        symmetricBroadcastServiceTag,
		}

		if cMixParams.DebugTag == cmix.DefaultDebugTag {
			cMixParams.DebugTag = symmCMixSendTag
		}
		return
	}

	return bc.net.SendWithAssembler(
		bc.channel.ReceptionID, assemble, cMixParams)
}
