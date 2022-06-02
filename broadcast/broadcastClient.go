////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

// Param encapsulates configuration options for a broadcastClient
type Param struct {
	Method Method
}

// broadcastClient implements the Channel interface for sending/receiving asymmetric or symmetric broadcast messages
type broadcastClient struct {
	channel crypto.Channel
	net     Client
	rng     *fastRNG.StreamGenerator
	param   Param
}

// NewBroadcastChannel creates a channel interface based on crypto.Channel, accepts net client connection & callback for received messages
func NewBroadcastChannel(channel crypto.Channel, listenerCb ListenerFunc, net Client, rng *fastRNG.StreamGenerator, param Param) (Channel, error) {
	bc := &broadcastClient{
		channel: channel,
		net:     net,
		rng:     rng,
		param:   param,
	}

	if !bc.verifyID() {
		jww.FATAL.Panicf("Failed ID verification for broadcast channel")
	}

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

	jww.INFO.Printf("New %s broadcast client created for channel %q (%s)",
		param.Method, channel.Name, channel.ReceptionID)

	return bc, nil
}

// Stop unregisters the listener callback and stops the channel's identity
// from being tracked.
func (bc *broadcastClient) Stop() {
	// Removes currently tracked identity
	bc.net.RemoveIdentity(bc.channel.ReceptionID)

	// Delete all registered services
	bc.net.DeleteClientService(bc.channel.ReceptionID)
}

// Get returns the underlying crypto.Channel object
func (bc *broadcastClient) Get() crypto.Channel {
	return bc.channel
}

// verifyID generates a symmetric ID based on the info in the channel & compares it to the one passed in
// TODO: it seems very odd to me that we do this, rather than just making the ID a private/ephemeral component like the key
func (bc *broadcastClient) verifyID() bool {
	gen, err := crypto.NewChannelID(bc.channel.Name, bc.channel.Description, bc.channel.Salt, rsa.CreatePublicKeyPem(bc.channel.RsaPubKey))
	if err != nil {
		jww.FATAL.Panicf("[verifyID] Failed to generate verified channel ID")
		return false
	}
	return bc.channel.ReceptionID.Cmp(gen)
}

func (bc *broadcastClient) MaxPayloadSize() int {
	switch bc.param.Method {
	case Symmetric:
		return bc.maxSymmetricPayload()
	case Asymmetric:
		return bc.maxAsymmetricPayload()
	default:
		return -1
	}
}
