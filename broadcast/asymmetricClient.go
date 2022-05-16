package broadcast

import (
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
)

// TODO: what happens if this is called using a symmetric broadcast client (& vice versa)

// BroadcastAsymmetric broadcasts the payload to the channel. Requires a healthy network state to send
// Payload must be equal to ac.MaxPayloadSize, and the channel PrivateKey must be passed in
func (bc *broadcastClient) BroadcastAsymmetric(pk multicastRSA.PrivateKey, payload []byte, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	if !bc.net.IsHealthy() {
		return 0, ephemeral.Id{}, errors.New(errNetworkHealth)
	}

	if len(payload) != bc.MaxAsymmetricPayloadSize() {
		return 0, ephemeral.Id{},
			errors.Errorf(errPayloadSize, len(payload), bc.MaxAsymmetricPayloadSize())
	}
	// Encrypt payload to send using asymmetric channel
	encryptedPayload, mac, fp, err := bc.channel.EncryptAsymmetric(payload, pk, bc.rng.GetStream())
	if err != nil {
		return 0, ephemeral.Id{}, errors.WithMessage(err, "Failed to encrypt asymmetric broadcast message")
	}

	// Create service object to send message
	service := message.Service{
		Identifier: bc.channel.ReceptionID.Bytes(),
		Tag:        asymmetricBroadcastServiceTag,
	}

	if cMixParams.DebugTag == cmix.DefaultDebugTag {
		cMixParams.DebugTag = asymmCMixSendTag
	}

	sizedPayload, err := NewSizedBroadcast(bc.net.GetMaxMessageLength(), encryptedPayload)
	if err != nil {
		return id.Round(0), ephemeral.Id{}, err
	}

	return bc.net.Send(
		bc.channel.ReceptionID, fp, service, sizedPayload, mac, cMixParams)
}

// MaxAsymmetricPayloadSize returns the maximum size for an asymmetric broadcast payload.
func (bc *broadcastClient) MaxAsymmetricPayloadSize() int {
	return bc.channel.MaxAsymmetricPayloadSize()
}
