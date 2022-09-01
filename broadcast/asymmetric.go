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
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/xx_network/crypto/multicastRSA"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

const (
	asymmetricBroadcastServiceTag = "AsymmBcast"
	asymmCMixSendTag              = "AsymmetricBroadcast"
	internalPayloadSizeLength     = 2
)

// BroadcastAsymmetric broadcasts the payload to the channel. Requires a healthy network state to send
// Payload must be equal to bc.MaxAsymmetricPayloadSize, and the channel PrivateKey must be passed in
func (bc *broadcastClient) BroadcastAsymmetric(pk multicastRSA.PrivateKey, payload []byte, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	// Confirm network health
	if !bc.net.IsHealthy() {
		return 0, ephemeral.Id{}, errors.New(errNetworkHealth)
	}

	// Check payload size
	if len(payload) > bc.MaxAsymmetricPayloadSize() {
		return 0, ephemeral.Id{},
			errors.Errorf(errPayloadSize, len(payload), bc.MaxAsymmetricPayloadSize())
	}
	payloadLength := uint16(len(payload))

	finalPayload := make([]byte, bc.maxAsymmetricPayloadSizeRaw())
	binary.BigEndian.PutUint16(finalPayload[:internalPayloadSizeLength], payloadLength)
	copy(finalPayload[internalPayloadSizeLength:], payload)

	// Encrypt payload
	encryptedPayload, mac, fp, err := bc.channel.EncryptAsymmetric(finalPayload, pk, bc.rng.GetStream())
	if err != nil {
		return 0, ephemeral.Id{}, errors.WithMessage(err, "Failed to encrypt asymmetric broadcast message")
	}

	// Create service using asymmetric broadcast service tag & channel reception ID
	// Allows anybody with this info to listen for messages on this channel
	service := message.Service{
		Identifier: bc.channel.ReceptionID.Bytes(),
		Tag:        asymmetricBroadcastServiceTag,
	}

	if cMixParams.DebugTag == cmix.DefaultDebugTag {
		cMixParams.DebugTag = asymmCMixSendTag
	}

	// Create payload sized for sending over cmix
	sizedPayload := make([]byte, bc.net.GetMaxMessageLength())
	// Read random data into sized payload
	_, err = bc.rng.GetStream().Read(sizedPayload)
	if err != nil {
		return 0, ephemeral.Id{}, errors.WithMessage(err, "Failed to add random data to sized broadcast")
	}
	copy(sizedPayload[:len(encryptedPayload)], encryptedPayload)

	return bc.net.Send(
		bc.channel.ReceptionID, fp, service, sizedPayload, mac, cMixParams)
}
