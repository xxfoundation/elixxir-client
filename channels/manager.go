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
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

const storageTagFormat = "channelManagerStorageTag-%s"

type manager struct {
	// Sender Identity
	me cryptoChannel.PrivateIdentity

	// List of all channels
	channels map[id.ID]*joinedChannel
	// List of dmTokens for each channel
	dmTokens map[id.ID]uint32
	mux      sync.RWMutex

	// External references
	kv  *versioned.KV
	net Client
	nm  NotificationsManager
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

	// Notification manager
	*notifications
}

// Client contains the methods from [cmix.Client] that are required by the
// [Manager].
type Client interface {
	GetMaxMessageLength() int
	SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
		cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error)
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool,
		fallthroughProcessor message.Processor)
	AddIdentityWithHistory(
		id *id.ID, validUntil, beginning time.Time,
		persistent bool, fallthroughProcessor message.Processor)
	RemoveIdentity(id *id.ID)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	UpsertCompressedService(clientID *id.ID, newService message.CompressedService,
		response message.Processor)
	DeleteClientService(clientID *id.ID)
	IsHealthy() bool
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
	GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback,
		roundList ...id.Round)
}

// NotificationsManager contains the methods from [notifications.Manager] that
// are required by the [Manager].
// TODO: update doc link real API
type NotificationsManager interface {
	Set(toBeNotifiedOn *id.ID, group string, metadata []byte,
		status clientNotif.NotificationStatus) error
	Get(toBeNotifiedOn *id.ID) (status clientNotif.NotificationStatus,
		metadata []byte, group string, exists bool)
	Delete(toBeNotifiedOn *id.ID)
	GetGroup(group string) (clientNotif.Group, bool)
	AddToken(newToken, app string) error
	RemoveToken() error
	RegisterUpdateCallback(group string, nu clientNotif.Update)
}

// NotificationInfo contains notification information for each identity.
type NotificationInfo struct {
	Status   bool   `json:"status"`
	Metadata []byte `json:"metadata"`
}

// NewManagerBuilder creates a new channel Manager using an EventModelBuilder.
func NewManagerBuilder(identity cryptoChannel.PrivateIdentity, kv *versioned.KV,
	net Client, rng *fastRNG.StreamGenerator, modelBuilder EventModelBuilder,
	extensions []ExtensionBuilder, addService AddServiceFn,
	nm NotificationsManager, cb FilterCallback) (Manager, error) {
	model, err := modelBuilder(getStorageTag(identity.PubKey))
	if err != nil {
		return nil, errors.Errorf("Failed to build event model: %+v", err)
	}

	return NewManager(
		identity, kv, net, rng, model, extensions, addService, nm, cb)
}

// NewManager creates a new channel [Manager] from a
// [cryptoChannel.PrivateIdentity]. It prefixes the KV with a tag derived from
// the public key that can be retried for reloading using
// [Manager.GetStorageTag].
//
// The [FilterCallback] returns services that are compared with notification
// data to determine which channel notifications belong to you using
// [GetNotificationReportsForMe]. It must be registered even if notifications
// are not used.
func NewManager(identity cryptoChannel.PrivateIdentity, kv *versioned.KV,
	net Client, rng *fastRNG.StreamGenerator, model EventModel,
	extensions []ExtensionBuilder, addService AddServiceFn,
	nm NotificationsManager, cb FilterCallback) (Manager, error) {

	// Make a copy of the public key to prevent outside edits
	// TODO: Convert this to DeepCopy() method
	pubKey := make([]byte, len(identity.PubKey))
	copy(pubKey, identity.PubKey)
	identity.PubKey = pubKey

	// Prefix the kv with the username so multiple can be run
	storageTag := getStorageTag(identity.PubKey)
	jww.INFO.Printf("[CH] NewManager for %s (pubKey:%x tag:%s)",
		identity.Codename, identity.PubKey, storageTag)
	kv = kv.Prefix(storageTag)

	if err := storeIdentity(kv, identity); err != nil {
		return nil, err
	}

	m := setupManager(identity, kv, net, rng, model, extensions, nm, cb)
	m.dmTokens = make(map[id.ID]uint32)

	return m, addService(m.leases.StartProcesses)
}

// LoadManager restores a channel Manager from disk stored at the given storage
// tag.
func LoadManager(storageTag string, kv *versioned.KV, net Client,
	rng *fastRNG.StreamGenerator, model EventModel,
	extensions []ExtensionBuilder, nm NotificationsManager, cb FilterCallback) (
	Manager, error) {
	jww.INFO.Printf("[CH] LoadManager for tag %s", storageTag)

	// Prefix the kv with the username so multiple can be run
	kv = kv.Prefix(storageTag)

	// Load the identity
	identity, err := loadIdentity(kv)
	if err != nil {
		return nil, err
	}

	m := setupManager(identity, kv, net, rng, model, extensions, nm, cb)
	m.loadDMTokens()

	return m, nil
}

// LoadManagerBuilder restores a channel Manager from disk stored at the given storage
// tag.
func LoadManagerBuilder(storageTag string, kv *versioned.KV, net Client,
	rng *fastRNG.StreamGenerator, modelBuilder EventModelBuilder,
	extensions []ExtensionBuilder, nm NotificationsManager, cb FilterCallback) (
	Manager, error) {
	model, err := modelBuilder(storageTag)
	if err != nil {
		return nil, errors.Errorf("Failed to build event model: %+v", err)
	}

	return LoadManager(storageTag, kv, net, rng, model, extensions, nm, cb)
}

func setupManager(identity cryptoChannel.PrivateIdentity, kv *versioned.KV,
	net Client, rng *fastRNG.StreamGenerator, model EventModel,
	extensionBuilders []ExtensionBuilder, nm NotificationsManager,
	cb FilterCallback) *manager {

	// Build the manager
	m := &manager{
		me:             identity,
		kv:             kv,
		net:            net,
		nm:             nm,
		rng:            rng,
		events:         initEvents(model, 512, kv, rng),
		broadcastMaker: broadcast.NewBroadcastChannel,
	}

	m.events.leases.RegisterReplayFn(m.adminReplayHandler)

	m.st = loadSendTracker(net, kv, m.events.triggerEvent,
		m.events.triggerAdminEvent, model.UpdateFromUUID, rng)

	m.loadChannels()

	m.nicknameManager = LoadOrNewNicknameManager(kv)

	m.notifications = newNotifications(identity.PubKey, cb, m, nm)

	// Activate all extensions
	var extensions []ExtensionMessageHandler
	for i := range extensionBuilders {
		ext, err := extensionBuilders[i](model, m, m.me)
		if err != nil {
			jww.FATAL.Panicf("[CH] Failed to initialize extension %d of %d: %+v",
				i, len(extensionBuilders), err)
		}
		extensions = append(extensions, ext...)
	}

	// Register all extensions
	for i := range extensions {
		ext := extensions[i]
		name, userSpace, adminSpace, mutedSpace := ext.GetProperties()
		err := m.events.RegisterReceiveHandler(ext.GetType(),
			&ReceiveMessageHandler{
				name, ext.Handle, userSpace, adminSpace, mutedSpace})
		if err != nil {
			jww.FATAL.Panicf("[CH] Extension message handle %s (%d of %d) "+
				"failed to register: %+v", name, i, len(extensions), err)
		}
	}

	return m
}

// adminReplayHandler registers a ReplayActionFunc with the lease system.
func (m *manager) adminReplayHandler(channelID *id.ID, encryptedPayload []byte) {
	messageID, r, _, err := m.replayAdminMessage(
		channelID, encryptedPayload, cmix.GetDefaultCMIXParams())
	if err != nil {
		jww.ERROR.Printf("[CH] Failed to replay admin message: %+v", err)
		return
	}

	jww.INFO.Printf("[CH] Replayed admin message on message %s in round %d",
		messageID, r.ID)
}

// GenerateChannel creates a new channel with the user as the admin and returns
// the broadcast.Channel object. This function only create a channel and does
// not join it.
//
// The private key is saved to storage and can be accessed with
// ExportChannelAdminKey.
func (m *manager) GenerateChannel(
	name, description string, privacyLevel cryptoBroadcast.PrivacyLevel) (
	*cryptoBroadcast.Channel, error) {
	jww.INFO.Printf("[CH] GenerateChannel %q with description %q and privacy "+
		"level %s", name, description, privacyLevel)
	ch, _, err := m.generateChannel(
		name, description, privacyLevel, m.net.GetMaxMessageLength())
	return ch, err
}

// generateChannel generates a new channel with a custom packet payload length.
func (m *manager) generateChannel(name, description string,
	privacyLevel cryptoBroadcast.PrivacyLevel, packetPayloadLength int) (
	*cryptoBroadcast.Channel, rsa.PrivateKey, error) {

	// Generate channel
	stream := m.rng.GetStream()
	ch, pk, err := cryptoBroadcast.NewChannel(
		name, description, privacyLevel, packetPayloadLength, stream)
	stream.Close()
	if err != nil {
		return nil, nil, err
	}

	// Save private key to storage
	err = saveChannelPrivateKey(ch.ReceptionID, pk, m.kv)
	if err != nil {
		return nil, nil, err
	}

	return ch, pk, nil
}

// JoinChannel joins the given channel. It will return the error
// ChannelAlreadyExistsErr if the channel has already been joined. This function
// will block until the event model returns from joining the channel.
func (m *manager) JoinChannel(channel *cryptoBroadcast.Channel) error {
	jww.INFO.Printf(
		"[CH] JoinChannel %q with ID %s", channel.Name, channel.ReceptionID)
	err := m.addChannel(channel)
	if err != nil {
		return err
	}

	err = m.EnableDirectMessages(channel.ReceptionID)
	if err != nil {
		return err
	}

	// Report joined channel to the event model
	m.events.model.JoinChannel(channel)

	return nil
}

// LeaveChannel leaves the given channel. It will return the error
// ChannelDoesNotExistsErr if the channel was not previously joined. This
// function will block until the event model returns from leaving the channel.
func (m *manager) LeaveChannel(channelID *id.ID) error {
	jww.INFO.Printf("[CH] LeaveChannel %s", channelID)
	err := m.removeChannel(channelID)
	if err != nil {
		return err
	}

	m.events.model.LeaveChannel(channelID)

	return nil
}

// EnableDirectMessages enables the token for direct messaging for this
// channel.
func (m *manager) EnableDirectMessages(chId *id.ID) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.enableDirectMessageToken(chId)
}

// DisableDirectMessages removes the token for direct messaging for a given
// channel.
func (m *manager) DisableDirectMessages(chId *id.ID) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.disableDirectMessageToken(chId)
}

// AreDMsEnabled returns status of DMs for a given channel ID (true if enabled)
func (m *manager) AreDMsEnabled(chId *id.ID) bool {
	m.mux.RLock()
	defer m.mux.RUnlock()
	_, ok := m.dmTokens[*chId]
	return ok
}

// ReplayChannel replays all messages from the channel within the network's
// memory (~3 weeks) over the event model. It does this by wiping the underlying
// state tracking for message pickup for the channel, causing all messages to be
// re-retrieved from the network.
//
// Returns the error ChannelDoesNotExistsErr if the channel was not previously
// joined.
func (m *manager) ReplayChannel(channelID *id.ID) error {
	jww.INFO.Printf("[CH] ReplayChannel %s", channelID)
	m.mux.RLock()
	defer m.mux.RUnlock()

	jc, exists := m.channels[*channelID]
	if !exists {
		return ChannelDoesNotExistsErr
	}

	c := jc.broadcast.Get()

	// Stop the broadcast that will completely wipe it from the underlying cmix
	// object
	jc.broadcast.Stop()

	// Re-instantiate the broadcast, re-registering it from scratch
	b, err := m.initBroadcast(c)
	if err != nil {
		return err
	}
	jc.broadcast = b

	return nil
}

// GetChannels returns the IDs of all channels that have been joined.
//
// Use manager.getChannelsUnsafe if you already have taken the mux.
func (m *manager) GetChannels() []*id.ID {
	jww.INFO.Print("[CH] GetChannels")
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.getChannelsUnsafe()
}

// GetChannel returns the underlying cryptographic structure for a given
// channel.
//
// Returns the error ChannelDoesNotExistsErr if the channel was not previously
// joined.
func (m *manager) GetChannel(channelID *id.ID) (*cryptoBroadcast.Channel, error) {
	jww.INFO.Printf("[CH] GetChannel %s", channelID)
	jc, err := m.getChannel(channelID)
	if err != nil {
		return nil, err
	} else if jc.broadcast == nil {
		return nil, errors.New("broadcast.Channel on joinedChannel is nil")
	}
	return jc.broadcast.Get(), nil
}

////////////////////////////////////////////////////////////////////////////////
// Other Channel Actions                                                      //
////////////////////////////////////////////////////////////////////////////////

// GetIdentity returns the public identity of the user associated with this
// channel manager.
func (m *manager) GetIdentity() cryptoChannel.Identity {
	return m.me.GetIdentity()
}

// ExportPrivateIdentity encrypts the private identity using the password and
// exports it to a portable string.
func (m *manager) ExportPrivateIdentity(password string) ([]byte, error) {
	jww.INFO.Print("[CH] ExportPrivateIdentity")
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

// Muted returns true if the user is currently muted in the given channel.
func (m *manager) Muted(channelID *id.ID) bool {
	jww.INFO.Printf("[CH] Muted in channel %s", channelID)
	return m.events.mutedUsers.isMuted(channelID, m.me.PubKey)
}

// GetMutedUsers returns the list of the public keys for each muted user in
// the channel. If there are no muted user or if the channel does not exist,
// an empty list is returned.
func (m *manager) GetMutedUsers(channelID *id.ID) []ed25519.PublicKey {
	jww.INFO.Printf("[CH] GetMutedUsers in channel %s", channelID)
	return m.mutedUsers.getMutedUsers(channelID)
}
