////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/multicastRSA"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

const (
	asymmetricBroadcastServiceTag = "AsymmBcast"
	asymmCMixSendTag              = "AsymmetricBroadcast"
)

// MaxAsymmetricPayloadSize returns the maximum size for an asymmetric broadcast payload
func (bc *broadcastClient) MaxAsymmetricPayloadSize() int {
	return bc.maxParts() * bc.channel.MaxAsymmetricPayloadSize()
}

// BroadcastAsymmetric broadcasts the payload to the channel. Requires a healthy network state to send
// Payload must be equal to bc.MaxAsymmetricPayloadSize, and the channel PrivateKey must be passed in
// Broadcast method must be set to asymmetric
func (bc *broadcastClient) BroadcastAsymmetric(pk multicastRSA.PrivateKey, payload []byte, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	if bc.param.Method != Asymmetric {
		return 0, ephemeral.Id{}, errors.Errorf(errBroadcastMethodType, Asymmetric, bc.param.Method)
	}

	if !bc.net.IsHealthy() {
		return 0, ephemeral.Id{}, errors.New(errNetworkHealth)
	}

	if len(payload) != bc.MaxAsymmetricPayloadSize() {
		return 0, ephemeral.Id{},
			errors.Errorf(errPayloadSize, len(payload), bc.MaxAsymmetricPayloadSize())
	}

	numParts := bc.maxParts()
	size := bc.channel.MaxAsymmetricPayloadSize()
	var mac []byte
	var fp format.Fingerprint
	var sequential []byte
	for i := 0; i < numParts; i++ {
		// Encrypt payload to send using asymmetric channel
		var encryptedPayload []byte
		var err error
		encryptedPayload, mac, fp, err = bc.channel.EncryptAsymmetric(payload[:size], pk, bc.rng.GetStream())
		if err != nil {
			return 0, ephemeral.Id{}, errors.WithMessage(err, "Failed to encrypt asymmetric broadcast message")
		}
		payload = payload[size:]
		sequential = append(sequential, encryptedPayload...)
	}

	// Create service object to send message
	service := message.Service{
		Identifier: bc.channel.ReceptionID.Bytes(),
		Tag:        asymmetricBroadcastServiceTag,
	}

	if cMixParams.DebugTag == cmix.DefaultDebugTag {
		cMixParams.DebugTag = asymmCMixSendTag
	}

	sizedPayload, err := NewSizedBroadcast(bc.net.GetMaxMessageLength(), sequential)
	if err != nil {
		return id.Round(0), ephemeral.Id{}, err
	}

	return bc.net.Send(
		bc.channel.ReceptionID, fp, service, sizedPayload, mac, cMixParams)
}

// Helper function for maximum number of encrypted message parts
func (bc *broadcastClient) maxParts() int {
	encPartSize := bc.channel.RsaPubKey.Size()
	maxSend := bc.net.GetMaxMessageLength()
	return maxSend / encPartSize
}
