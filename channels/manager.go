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
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"time"
)

const storageTagFormat = "channelManagerStorageTag-%s"

type manager struct {
	// Sender Identity
	me cryptoChannel.PrivateIdentity

	// List of all channels
	channels map[id.ID]*joinedChannel
	mux      sync.RWMutex

	// External references
	kv  *versioned.KV
	net Client
	rng *fastRNG.StreamGenerator

	// Events model
	*events

	// Nicknames
	*nicknameManager

	// Send tracker
	st *sendTracker

	// Makes the function that is used to create broadcasts be a pointer so that
	// it can be replaced in tests
	broadcastMaker broadcast.NewBroadcastChannelFunc
}

// Client contains the methods from cmix.Client that are required by the
// [Manager].
type Client interface {
	GetMaxMessageLength() int
	SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
		cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error)
	IsHealthy() bool
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool)
	AddIdentityWithHistory(
		id *id.ID, validUntil, beginning time.Time, persistent bool)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteClientService(clientID *id.ID)
	RemoveIdentity(id *id.ID)
	GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback,
		roundList ...id.Round)
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
}

// EventModelBuilder initialises the event model using the given path.
type EventModelBuilder func(path string) (EventModel, error)

// NewManager creates a new channel Manager from a [channel.PrivateIdentity]. It
// prefixes the KV with a tag derived from the public key that can be retried
// for reloading using [Manager.GetStorageTag].
func NewManager(identity cryptoChannel.PrivateIdentity, kv *versioned.KV,
	net Client, rng *fastRNG.StreamGenerator, modelBuilder EventModelBuilder) (
	Manager, error) {

	// Prefix the kv with the username so multiple can be run
	storageTag := getStorageTag(identity.PubKey)
	jww.INFO.Printf("NewManager(ID:%s-%s, tag:%s)", identity.Codename,
		identity.PubKey, storageTag)
	kv = kv.Prefix(storageTag)

	if err := storeIdentity(kv, identity); err != nil {
		return nil, err
	}

	model, err := modelBuilder(storageTag)
	if err != nil {
		return nil, errors.Errorf("Failed to build event model: %+v", err)
	}

	m := setupManager(identity, kv, net, rng, model)

	return m, nil
}

// LoadManager restores a channel Manager from disk stored at the given storage
// tag.
func LoadManager(storageTag string, kv *versioned.KV, net Client,
	rng *fastRNG.StreamGenerator, modelBuilder EventModelBuilder) (Manager, error) {

	jww.INFO.Printf("LoadManager(tag:%s)", storageTag)

	// Prefix the kv with the username so multiple can be run
	kv = kv.Prefix(storageTag)

	// Load the identity
	identity, err := loadIdentity(kv)
	if err != nil {
		return nil, err
	}

	model, err := modelBuilder(storageTag)
	if err != nil {
		return nil, errors.Errorf("Failed to build event model: %+v", err)
	}

	m := setupManager(identity, kv, net, rng, model)

	return m, nil
}

func setupManager(identity cryptoChannel.PrivateIdentity, kv *versioned.KV,
	net Client, rng *fastRNG.StreamGenerator, model EventModel) *manager {

	m := manager{
		me:             identity,
		kv:             kv,
		net:            net,
		rng:            rng,
		broadcastMaker: broadcast.NewBroadcastChannel,
	}

	m.events = initEvents(model)

	m.st = loadSendTracker(net, kv, m.events.triggerEvent,
		m.events.triggerAdminEvent, model.UpdateSentStatus, rng)

	m.loadChannels()

	m.nicknameManager = loadOrNewNicknameManager(kv)

	return &m
}

// JoinChannel joins the given channel. It will fail if the channel has already
// been joined.
func (m *manager) JoinChannel(channel *cryptoBroadcast.Channel) error {
	jww.INFO.Printf("JoinChannel(%s[%s])", channel.Name, channel.ReceptionID)
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
	jww.INFO.Printf("LeaveChannel(%s)", channelID)
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
	jww.INFO.Printf("GetChannels")
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.getChannelsUnsafe()
}

// GetChannel returns the underlying cryptographic structure for a given
// channel.
func (m *manager) GetChannel(chID *id.ID) (*cryptoBroadcast.Channel, error) {
	jww.INFO.Printf("GetChannel(%s)", chID)
	jc, err := m.getChannel(chID)
	if err != nil {
		return nil, err
	} else if jc.broadcast == nil {
		return nil, errors.New("broadcast.Channel on joinedChannel is nil")
	}
	return jc.broadcast.Get(), nil
}

// ReplayChannel replays all messages from the channel within the network's
// memory (~3 weeks) over the event model. It does this by wiping the underlying
// state tracking for message pickup for the channel, causing all messages to be
// re-retrieved from the network.
func (m *manager) ReplayChannel(chID *id.ID) error {
	jww.INFO.Printf("ReplayChannel(%s)", chID)
	m.mux.RLock()
	defer m.mux.RUnlock()

	jc, exists := m.channels[*chID]
	if !exists {
		return ChannelDoesNotExistsErr
	}

	c := jc.broadcast.Get()

	// Stop the broadcast that will completely wipe it from the underlying cmix
	// object
	jc.broadcast.Stop()

	// Re-instantiate the broadcast, re-registering it from scratch
	b, err := initBroadcast(c, m.events, m.net, m.broadcastMaker, m.rng,
		m.st.MessageReceive)
	if err != nil {
		return err
	}
	jc.broadcast = b

	return nil

}

// GetIdentity returns the public identity associated with this channel manager.
func (m *manager) GetIdentity() cryptoChannel.Identity {
	return m.me.Identity
}

// ExportPrivateIdentity encrypts and exports the private identity to a portable
// string.
func (m *manager) ExportPrivateIdentity(password string) ([]byte, error) {
	jww.INFO.Printf("ExportPrivateIdentity()")
	rng := m.rng.GetStream()
	defer rng.Close()
	return m.me.Export(password, rng)
}

// GetStorageTag returns the tag at where this manager is stored. To be used
// when loading the manager. The storage tag is derived from the public key.
func (m *manager) GetStorageTag() string {
	return getStorageTag(m.me.PubKey)
}

// getStorageTag generates a storage tag from an Ed25519 public key.
func getStorageTag(pub ed25519.PublicKey) string {
	return fmt.Sprintf(storageTagFormat, base64.StdEncoding.EncodeToString(pub))
}
