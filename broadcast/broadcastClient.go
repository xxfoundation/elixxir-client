package broadcast

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
)

type Method uint8

const (
	Symmetric Method = iota
	Asymmetric
)

func (m Method) String() string {
	switch m {
	case Symmetric:
		return "Symmetric"
	case Asymmetric:
		return "Asymmetric"
	default:
		return "Unknown"
	}
}

type Param struct {
	Method Method
}

type broadcastClient struct {
	channel crypto.Channel
	net     Client
	rng     *fastRNG.StreamGenerator
	param   Param
}

// NewBroadcastChannel creates a channel interface based on crypto.Channel, accepts net client connection & callback for received messages
func NewBroadcastChannel(channel crypto.Channel, listenerCb ListenerFunc, net Client, rng *fastRNG.StreamGenerator, param Param) (Channel, error) {
	// Add channel's identity
	net.AddIdentity(channel.ReceptionID, identity.Forever, true)

	p := &processor{
		c:      &channel,
		cb:     listenerCb,
		method: param.Method,
	}
	var tag string
	switch param.Method {
	case Symmetric:
		tag = symmetricBroadcastServiceTag
	case Asymmetric:
		tag = asymmetricBroadcastServiceTag
	default:
		return nil, errors.Errorf("Cannot make broadcast client for unknown broadcast method %s", param.Method)
	}
	service := message.Service{
		Identifier: channel.ReceptionID.Bytes(),
		Tag:        tag,
	}

	net.AddService(channel.ReceptionID, service, p)

	jww.INFO.Printf("New broadcast client created for channel %q (%s)",
		channel.Name, channel.ReceptionID)

	return &broadcastClient{
		channel: channel,
		net:     net,
		rng:     rng,
		param:   param,
	}, nil
}

// Stop unregisters the listener callback and stops the channel's identity
// from being tracked.
func (bc *broadcastClient) Stop() {
	// Removes currently tracked identity
	bc.net.RemoveIdentity(bc.channel.ReceptionID)

	// Delete all registered services
	bc.net.DeleteClientService(bc.channel.ReceptionID)
}

func (bc *broadcastClient) Get() crypto.Channel {
	return bc.channel
}
