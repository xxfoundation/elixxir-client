////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/json"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	joinedChannelsVersion = 0
	joinedChannelsKey     = "JoinedChannelsKey"
	joinedChannelVersion  = 0
	joinedChannelKey      = "JoinedChannelKey-"
)

// store stores the list of joined channels to disk while taking the read lock.
func (m *manager) store() error {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.storeUnsafe()
}

// storeUnsafe stores the list of joined channels to disk without taking the
// read lock. It must be used by another function that has already taken the
// read lock.
func (m *manager) storeUnsafe() error {
	channelsList := m.getChannelsUnsafe()

	data, err := json.Marshal(&channelsList)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   joinedChannelsVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return m.kv.Set(joinedChannelsKey, obj)
}

// loadChannels loads all currently joined channels from disk and registers them
// for message reception.
func (m *manager) loadChannels() {
	obj, err := m.kv.Get(joinedChannelsKey, joinedChannelsVersion)
	if !m.kv.Exists(err) {
		m.channels = make(map[id.ID]*joinedChannel)
		return
	} else if err != nil {
		jww.FATAL.Panicf("[CH] Failed to load channels: %+v", err)
	}

	chList := make([]*id.ID, 0, len(m.channels))
	if err = json.Unmarshal(obj.Data, &chList); err != nil {
		jww.FATAL.Panicf("[CH] Failed to load channels: %+v", err)
	}

	chMap := make(map[id.ID]*joinedChannel)

	for i := range chList {
		jc, err2 := m.loadJoinedChannel(chList[i])
		if err2 != nil {
			jww.FATAL.Panicf("[CH] Failed to load channel %s (%d of %d): %+v",
				chList[i], i, len(chList), err2)
		}
		chMap[*chList[i]] = jc
	}

	m.channels = chMap
}

// addChannel adds a channel.
func (m *manager) addChannel(channel *cryptoBroadcast.Channel) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	if _, exists := m.channels[*channel.ReceptionID]; exists {
		return ChannelAlreadyExistsErr
	}

	b, err := m.broadcastMaker(channel, m.net, m.rng)
	if err != nil {
		return err
	}

	jc := &joinedChannel{b}
	if err = jc.Store(m.kv); err != nil {
		go b.Stop()
		return err
	}

	m.channels[*jc.broadcast.Get().ReceptionID] = jc

	if err = m.storeUnsafe(); err != nil {
		go b.Stop()
		return err
	}

	// Enable notifications
	err = m.notif.addChannel(channel.ReceptionID)
	if err != nil {
		return errors.WithMessage(err,
			"failed to add channel to notification manager")
	}

	// Connect to listeners
	_, err = m.registerListeners(b, channel)

	return err
}

// removeChannel deletes the channel with the given ID from the channel list and
// stops it from broadcasting. Returns ChannelDoesNotExistsErr error if the
// channel does not exist.
func (m *manager) removeChannel(channelID *id.ID) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	ch, exists := m.channels[*channelID]
	if !exists {
		return ChannelDoesNotExistsErr
	}

	ch.broadcast.Stop()

	err := m.mutedUsers.removeChannel(channelID)
	if err != nil {
		return err
	}

	err = m.leases.deleteLeaseMessages(channelID)
	if err != nil {
		return err
	}

	m.broadcast.removeProcessors(channelID)

	m.events.leases.RemoveChannel(channelID)

	delete(m.channels, *channelID)

	// Delete channel from channel list
	err = m.storeUnsafe()
	if err != nil {
		return err
	}

	// Delete channel from storage
	ch.delete(m.kv)

	// Disable notifications
	m.notif.removeChannel(channelID)

	return nil
}

// getChannel returns the given channel. Returns ChannelDoesNotExistsErr error
// if the channel does not exist.
func (m *manager) getChannel(channelID *id.ID) (*joinedChannel, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	jc, exists := m.channels[*channelID]
	if !exists {
		return nil, ChannelDoesNotExistsErr
	}

	return jc, nil
}

// getChannelsUnsafe returns the IDs of all channels that have been joined. This
// function is unsafe because it does not take the mux; only use this function
// when under a lock.
func (m *manager) getChannelsUnsafe() []*id.ID {
	list := make([]*id.ID, 0, len(m.channels))
	for chID := range m.channels {
		list = append(list, chID.DeepCopy())
	}
	return list
}

// joinedChannel holds channel info. It will expand to include admin data, so it
// will be treated as a struct for now.
type joinedChannel struct {
	broadcast broadcast.Channel
}

// joinedChannelDisk is the representation of joinedChannel for storage.
type joinedChannelDisk struct {
	Broadcast *cryptoBroadcast.Channel
}

// Store writes the given channel to a unique storage location within the EKV.
func (jc *joinedChannel) Store(kv *versioned.KV) error {
	jcd := joinedChannelDisk{jc.broadcast.Get()}
	data, err := json.Marshal(&jcd)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   joinedChannelVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return kv.Set(makeJoinedChannelKey(jc.broadcast.Get().ReceptionID), obj)
}

// loadJoinedChannel loads a given channel from ekv storage.
func (m *manager) loadJoinedChannel(channelID *id.ID) (*joinedChannel, error) {
	obj, err := m.kv.Get(makeJoinedChannelKey(channelID), joinedChannelVersion)
	if err != nil {
		return nil, err
	}

	jcd := &joinedChannelDisk{}
	err = json.Unmarshal(obj.Data, jcd)
	if err != nil {
		return nil, err
	}

	b, err := m.initBroadcast(jcd.Broadcast)
	if err != nil {
		return nil, err
	}

	jc := &joinedChannel{broadcast: b}
	return jc, nil
}

// delete removes the channel from the kv.
func (jc *joinedChannel) delete(kv *versioned.KV) {
	err := kv.Delete(makeJoinedChannelKey(jc.broadcast.Get().ReceptionID),
		joinedChannelVersion)
	if err != nil {
		// Print an error instead of returning/panicking because the worst case
		// scenario is a storage leak
		jww.ERROR.Printf("[CH] Failed to delete channel %s from KV: %+v",
			jc.broadcast.Get().ReceptionID, err)
	}
}

func makeJoinedChannelKey(channelID *id.ID) string {
	return joinedChannelKey + channelID.HexEncode()
}

func (m *manager) initBroadcast(
	channel *cryptoBroadcast.Channel) (broadcast.Channel, error) {
	broadcastChan, err := m.broadcastMaker(channel, m.net, m.rng)
	if err != nil {
		return nil, err
	}

	return m.registerListeners(broadcastChan, channel)
}

// registerListeners registers all the listeners on the broadcast channel.
func (m *manager) registerListeners(broadcastChan broadcast.Channel,
	channel *cryptoBroadcast.Channel) (broadcast.Channel, error) {
	// User message listener
	p, err := broadcastChan.RegisterSymmetricListener((&userListener{
		chID:      channel.ReceptionID,
		trigger:   m.events.triggerEvent,
		checkSent: m.st.MessageReceive,
	}).Listen, nil)
	if err != nil {
		return nil, err
	}
	m.broadcast.addProcessor(channel.ReceptionID, userProcessor, p)

	// Admin message listener
	p, err = broadcastChan.RegisterRSAtoPublicListener((&adminListener{
		chID:    channel.ReceptionID,
		trigger: m.events.triggerAdminEvent,
	}).Listen, nil)
	if err != nil {
		return nil, err
	}
	m.broadcast.addProcessor(channel.ReceptionID, adminProcessor, p)

	return broadcastChan, nil
}
