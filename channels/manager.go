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

// on sending, data propagates as follows:
// Send function (Example: SendMessage) - > SendGeneric ->
//     Broadcast.BroadcastWithAssembler -> cmix.SendWithAssembler

// on receiving messages propagate as follows:
// cmix message pickup (by service)- > broadcast.Processor ->
//     userListener ->  events.triggerEvent ->
//     messageTypeHandler (example: Text) ->
//     eventModel (example: ReceiveMessage)

// on sendingAdmin, data propagates as follows:
// Send function - > SendAdminGeneric ->
//     Broadcast.BroadcastAsymmetricWithAssembler -> cmix.SendWithAssembler

// on receiving admin messages propagate as follows:
// cmix message pickup (by service)- > broadcast.Processor -> adminListener ->
//     events.triggerAdminEvent -> messageTypeHandler (example: Text) ->
//     eventModel (example: ReceiveMessage)

import (
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type manager struct {
	// List of all channels
	channels map[id.ID]*joinedChannel
	mux      sync.RWMutex

	// External references
	kv     *versioned.KV
	client broadcast.Client
	rng    *fastRNG.StreamGenerator
	name   NameService

	// Events model
	*events

	// Makes the function that is used to create broadcasts be a pointer so that
	// it can be replaced in tests
	broadcastMaker broadcast.NewBroadcastChannelFunc
}

// NewManager creates a new channel.Manager. It prefixes the KV with the
// username so that multiple instances for multiple users will not error.
func NewManager(kv *versioned.KV, client broadcast.Client,
	rng *fastRNG.StreamGenerator, name NameService, model EventModel) Manager {

	// Prefix the kv with the username so multiple can be run
	kv = kv.Prefix(name.GetUsername())

	m := manager{
		kv:             kv,
		client:         client,
		rng:            rng,
		name:           name,
		broadcastMaker: broadcast.NewBroadcastChannel,
	}

	m.events = initEvents(model)

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
