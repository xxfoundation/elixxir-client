////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

// Package channels provides a channels implementation on top of broadcast
// which is capable of handing the user facing features of channels, including
// replies, reactions, and eventually admin commands.
package channels

import (
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"time"
)

type manager struct {
	// List of all channels
	channels map[id.ID]*joinedChannel
	mux      sync.RWMutex

	// External references
	kv   *versioned.KV
	net  Client
	rng  *fastRNG.StreamGenerator
	name NameService

	// Events model
	*events

	//send tracker
	st *sendTracker

	// Makes the function that is used to create broadcasts be a pointer so that
	// it can be replaced in tests
	broadcastMaker broadcast.NewBroadcastChannelFunc
}

// Client contains the methods from cmix.Client that are required by
// symmetricClient.
type Client interface {
	GetMaxMessageLength() int
	SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
		cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error)
	IsHealthy() bool
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteClientService(clientID *id.ID)
	RemoveIdentity(id *id.ID)
	GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback,
		roundList ...id.Round)
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
}

// NewManager creates a new channel.Manager. It prefixes the KV with the
// username so that multiple instances for multiple users will not error.
func NewManager(kv *versioned.KV, net Client,
	rng *fastRNG.StreamGenerator, name NameService, model EventModel) Manager {

	// Prefix the kv with the username so multiple can be run
	kv = kv.Prefix(name.GetUsername())

	m := manager{
		kv:             kv,
		net:            net,
		rng:            rng,
		name:           name,
		broadcastMaker: broadcast.NewBroadcastChannel,
	}

	m.events = initEvents(model)

	m.st = loadSendTracker(net, kv, m.events.triggerEvent,
		m.events.triggerAdminEvent, model.UpdateSentStatus)

	m.loadChannels()

	return &m
}

// JoinChannel joins the given channel. It will fail if the channel has already
// been joined.
func (m *manager) JoinChannel(channel *cryptoBroadcast.Channel) error {
	err := m.addChannel(channel)
	if err != nil {
		return err
	}

	go m.events.model.JoinChannel(channel)

	return nil
}

// LeaveChannel leaves the given channel. It will return an error if the channel
// was not previously joined.
func (m *manager) LeaveChannel(channelID *id.ID) error {
	err := m.removeChannel(channelID)
	if err != nil {
		return err
	}

	go m.events.model.LeaveChannel(channelID)

	return nil
}

// GetChannels returns the IDs of all channels that have been joined. Use
// getChannelsUnsafe if you already have taken the mux.
func (m *manager) GetChannels() []*id.ID {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.getChannelsUnsafe()
}

// GetChannel returns the underlying cryptographic structure for a given channel.
func (m *manager) GetChannel(chID *id.ID) (*cryptoBroadcast.Channel, error) {
	jc, err := m.getChannel(chID)
	if err != nil {
		return nil, err
	}
	return jc.broadcast.Get(), nil
}

// ReplayChannel replays all messages from the channel within the network's
// memory (~3 weeks) over the event model. It does this by wiping the
// underlying state tracking for message pickup for the channel, causing all
// messages to be re-retrieved from the network
func (m *manager) ReplayChannel(chID *id.ID) error {
	m.mux.RLock()
	defer m.mux.RUnlock()

	jc, exists := m.channels[*chID]
	if !exists {
		return ChannelDoesNotExistsErr
	}

	c := jc.broadcast.Get()

	// stop the broadcast which will completely wipe it from the underlying
	// cmix object
	jc.broadcast.Stop()

	//re-instantiate the broadcast, re-registering it from scratch
	b, err := initBroadcast(c, m.name, m.events, m.net, m.broadcastMaker, m.rng,
		m.st.MessageReceive)
	if err != nil {
		return err
	}
	jc.broadcast = b

	return nil

}
