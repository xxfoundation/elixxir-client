////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v5/cmix"
	"gitlab.com/elixxir/client/v5/cmix/message"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

const (
	asymmetricRSAToPublicBroadcastServiceTag = "AsymmToPublicBcast"
	asymmCMixSendTag                         = "AsymmetricBroadcast"
	internalPayloadSizeLength                = 2
)

// BroadcastRSAtoPublic broadcasts the payload to the channel. Requires a
// healthy network state to send Payload length less than or equal to
// bc.MaxRSAToPublicPayloadSize, and the channel PrivateKey must be passed in
//
// BroadcastRSAtoPublic broadcasts the payload to the channel.
//
// The payload must be of the size [broadcastClient.MaxRSAToPublicPayloadSize]
// or smaller and the channel [rsa.PrivateKey] must be passed in.
//
// The network must be healthy to send.
func (bc *broadcastClient) BroadcastRSAtoPublic(
	pk rsa.PrivateKey, payload []byte, cMixParams cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	assemble := func(rid id.Round) ([]byte, error) { return payload, nil }
	return bc.BroadcastRSAToPublicWithAssembler(pk, assemble, cMixParams)
}

// BroadcastRSAToPublicWithAssembler broadcasts the payload to the channel
// with a function that builds the payload based upon the ID of the selected
// round.
//
// The payload must be of the size [broadcastClient.MaxRSAToPublicPayloadSize]
// or smaller and the channel [rsa.PrivateKey] must be passed in.
//
// The network must be healthy to send.
func (bc *broadcastClient) BroadcastRSAToPublicWithAssembler(
	pk rsa.PrivateKey, assembler Assembler,
	cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	// Confirm network health
	if !bc.net.IsHealthy() {
		return rounds.Round{}, ephemeral.Id{}, errors.New(errNetworkHealth)
	}

	assemble := func(rid id.Round) (fp format.Fingerprint,
		service message.Service, encryptedPayload, mac []byte, err error) {
		payload, err := assembler(rid)
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, err
		}
		// Check payload size
		if len(payload) > bc.MaxRSAToPublicPayloadSize() {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, errors.Errorf(errPayloadSize, len(payload),
					bc.MaxRSAToPublicPayloadSize())
		}
		payloadLength := uint16(len(payload))

		finalPayload := make([]byte, bc.maxRSAToPublicPayloadSizeRaw())
		binary.BigEndian.PutUint16(finalPayload[:internalPayloadSizeLength],
			payloadLength)
		copy(finalPayload[internalPayloadSizeLength:], payload)

		// Encrypt payload
		encryptedPayload, mac, fp, err =
			bc.channel.EncryptRSAToPublic(finalPayload, pk, bc.net.GetMaxMessageLength(),
				bc.rng.GetStream())
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, errors.WithMessage(err, "Failed to encrypt "+
					"asymmetric broadcast message")
		}

		// Create service using asymmetric broadcast service tag and channel
		// reception ID allows anybody with this info to listen for messages on
		// this channel
		service = message.Service{
			Identifier: bc.channel.ReceptionID.Bytes(),
			Tag:        asymmetricRSAToPublicBroadcastServiceTag,
		}

		if cMixParams.DebugTag == cmix.DefaultDebugTag {
			cMixParams.DebugTag = asymmCMixSendTag
		}

		// Create payload sized for sending over cmix
		sizedPayload := make([]byte, bc.net.GetMaxMessageLength())
		// Read random data into sized payload
		_, err = bc.rng.GetStream().Read(sizedPayload)
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, errors.WithMessage(err, "Failed to add "+
					"random data to sized broadcast")
		}
		copy(sizedPayload[:len(encryptedPayload)], encryptedPayload)

		return
	}

	return bc.net.SendWithAssembler(bc.channel.ReceptionID, assemble, cMixParams)
}
