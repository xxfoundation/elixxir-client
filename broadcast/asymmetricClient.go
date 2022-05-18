package broadcast

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/multicastRSA"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

const (
	asymmetricBroadcastServiceTag = "AsymmBcast"
	asymmCMixSendTag              = "AsymmetricBroadcast"
)

type asymmetricClient struct {
	channel crypto.Asymmetric
	net     Client
	rng     *fastRNG.StreamGenerator
}

// Creates a
func NewAsymmetricClient(channel crypto.Asymmetric, listenerCb ListenerFunc, net Client, rng *fastRNG.StreamGenerator) Asymmetric {
	// Add channel's identity
	net.AddIdentity(channel.ReceptionID, identity.Forever, true)

	p := &asymmetricProcessor{
		ac: &channel,
		cb: listenerCb,
	}

	service := message.Service{
		Identifier: channel.ReceptionID.Bytes(),
		Tag:        asymmetricBroadcastServiceTag,
	}

	net.AddService(channel.ReceptionID, service, p)

	jww.INFO.Printf("New asymmetric broadcast client created for channel %q (%s)",
		channel.Name, channel.ReceptionID)

	return &asymmetricClient{
		channel: channel,
		net:     net,
		rng:     rng,
	}
}

// Broadcast broadcasts the payload to the channel. Requires a healthy network state to send
// Payload must be equal to ac.MaxPayloadSize, and the channel PrivateKey must be passed in
func (ac *asymmetricClient) Broadcast(pk multicastRSA.PrivateKey, payload []byte, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	if !ac.net.IsHealthy() {
		return 0, ephemeral.Id{}, errors.New(errNetworkHealth)
	}

	if len(payload) != ac.MaxPayloadSize() {
		return 0, ephemeral.Id{},
			errors.Errorf(errPayloadSize, len(payload), ac.MaxPayloadSize())
	}
	// Encrypt payload to send using asymmetric channel
	encryptedPayload, mac, fp, err := ac.channel.Encrypt(payload, pk, ac.rng.GetStream())
	if err != nil {
		return 0, ephemeral.Id{}, errors.WithMessage(err, "Failed to encrypt asymmetric broadcast message")
	}

	// Create service object to send message
	service := message.Service{
		Identifier: ac.channel.ReceptionID.Bytes(),
		Tag:        asymmetricBroadcastServiceTag,
	}

	if cMixParams.DebugTag == cmix.DefaultDebugTag {
		cMixParams.DebugTag = asymmCMixSendTag
	}

	return ac.net.Send(
		ac.channel.ReceptionID, fp, service, encryptedPayload, mac, cMixParams)
}

// MaxPayloadSize returns the maximum size for a broadcasted payload.
func (ac *asymmetricClient) MaxPayloadSize() int {
	return ac.net.GetMaxMessageLength()
}

// Stop unregisters the listener callback and stops the channel's identity
// from being tracked.
func (ac *asymmetricClient) Stop() {
	// Removes currently tracked identity
	ac.net.RemoveIdentity(ac.channel.ReceptionID)

	// Delete all registered services
	ac.net.DeleteClientService(ac.channel.ReceptionID)
}

func (ac *asymmetricClient) Get() crypto.Asymmetric {
	return ac.channel
}
