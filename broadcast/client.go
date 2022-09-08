////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"golang.org/x/crypto/blake2b"

	"gitlab.com/xx_network/crypto/multicastRSA"

	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
)

// broadcastClient implements the Channel interface for sending/receiving asymmetric or symmetric broadcast messages
type broadcastClient struct {
	channel *crypto.Channel
	net     Client
	rng     *fastRNG.StreamGenerator
}

type NewBroadcastChannelFunc func(channel *crypto.Channel, net Client, rng *fastRNG.StreamGenerator) (Channel, error)

// NewBroadcastChannel creates a channel interface based on crypto.Channel, accepts net client connection & callback for received messages
func NewBroadcastChannel(channel *crypto.Channel, net Client, rng *fastRNG.StreamGenerator) (Channel, error) {
	bc := &broadcastClient{
		channel: channel,
		net:     net,
		rng:     rng,
	}

	if !bc.verifyID() {
		return nil, errors.New("Failed ID verification for broadcast channel")
	}

	// Add channel's identity
	net.AddIdentity(channel.ReceptionID, identity.Forever, true)

	jww.INFO.Printf("New broadcast channel client created for channel %q (%s)",
		channel.Name, channel.ReceptionID)

	return bc, nil
}

// RegisterListener adds a service to hear broadcast messages of a given type via the passed in callback
func (bc *broadcastClient) RegisterListener(listenerCb ListenerFunc, method Method) error {
	var tag string
	switch method {
	case Symmetric:
		tag = symmetricBroadcastServiceTag
	case Asymmetric:
		tag = asymmetricBroadcastServiceTag
	default:
		return errors.Errorf("Cannot register listener for broadcast method %s", method)
	}

	p := &processor{
		c:      bc.channel,
		cb:     listenerCb,
		method: method,
	}

	service := message.Service{
		Identifier: bc.channel.ReceptionID.Bytes(),
		Tag:        tag,
	}

	bc.net.AddService(bc.channel.ReceptionID, service, p)
	return nil
}

// Stop unregisters the listener callback and stops the channel's identity
// from being tracked.
func (bc *broadcastClient) Stop() {
	// Removes currently tracked identity
	bc.net.RemoveIdentity(bc.channel.ReceptionID)

	// Delete all registered services
	bc.net.DeleteClientService(bc.channel.ReceptionID)
}

// Get returns the underlying crypto.Channel object.
func (bc *broadcastClient) Get() *crypto.Channel {
	return bc.channel
}

// verifyID generates a symmetric ID based on the info in the channel and
// compares it to the one passed in.
func (bc *broadcastClient) verifyID() bool {
	hashedSecret := blake2b.Sum256(bc.channel.Secret)
	gen, err := crypto.NewChannelID(bc.channel.Name, bc.channel.Description,
		bc.channel.Salt, bc.channel.RsaPubKeyHash, hashedSecret[:])
	if err != nil {
		jww.FATAL.Panicf("[verifyID] Failed to generate verified channel ID")
		return false
	}
	return bc.channel.ReceptionID.Cmp(gen)
}

func (bc *broadcastClient) MaxPayloadSize() int {
	return bc.maxSymmetricPayload()
}

func (bc *broadcastClient) MaxAsymmetricPayloadSize(pk multicastRSA.PublicKey) int {
	return bc.maxAsymmetricPayloadSizeRaw(pk) - internalPayloadSizeLength
}

func (bc *broadcastClient) maxAsymmetricPayloadSizeRaw(pk multicastRSA.PublicKey) int {
	return bc.channel.MaxAsymmetricPayloadSize(pk)
}
