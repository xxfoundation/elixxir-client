////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
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
		jww.FATAL.Panicf("Failed to load channels: %+v", err)
	}

	chList := make([]*id.ID, 0, len(m.channels))
	if err = json.Unmarshal(obj.Data, &chList); err != nil {
		jww.FATAL.Panicf("Failed to load channels: %+v", err)
	}

	chMap := make(map[id.ID]*joinedChannel)

	for i := range chList {
		jc, err := loadJoinedChannel(
			chList[i], m.kv, m.net, m.rng, m.events, m.broadcastMaker,
			m.st.MessageReceive)
		if err != nil {
			jww.FATAL.Panicf("Failed to load channel %s: %+v", chList[i], err)
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

	// Connect to listeners
	err = b.RegisterListener((&userListener{
		chID:      channel.ReceptionID,
		trigger:   m.events.triggerEvent,
		checkSent: m.st.MessageReceive,
	}).Listen, broadcast.Symmetric)
	if err != nil {
		return err
	}

	err = b.RegisterListener((&adminListener{
		chID:      channel.ReceptionID,
		trigger:   m.events.triggerAdminEvent,
		checkSent: m.st.MessageReceive,
	}).Listen, broadcast.RSAToPublic)
	if err != nil {
		return err
	}

	return nil
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

	delete(m.channels, *channelID)

	err := m.storeUnsafe()
	if err != nil {
		return err
	}

	return ch.delete(m.kv)
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
func loadJoinedChannel(chId *id.ID, kv *versioned.KV, net broadcast.Client,
	rngGen *fastRNG.StreamGenerator, e *events,
	broadcastMaker broadcast.NewBroadcastChannelFunc, mr messageReceiveFunc) (
	*joinedChannel, error) {
	obj, err := kv.Get(makeJoinedChannelKey(chId), joinedChannelVersion)
	if err != nil {
		return nil, err
	}

	jcd := &joinedChannelDisk{}

	err = json.Unmarshal(obj.Data, jcd)
	if err != nil {
		return nil, err
	}

	b, err := initBroadcast(jcd.Broadcast, e, net, broadcastMaker, rngGen, mr)

	jc := &joinedChannel{broadcast: b}
	return jc, nil
}

// delete removes the channel from the kv.
func (jc *joinedChannel) delete(kv *versioned.KV) error {
	return kv.Delete(makeJoinedChannelKey(jc.broadcast.Get().ReceptionID),
		joinedChannelVersion)
}

func makeJoinedChannelKey(chId *id.ID) string {
	return joinedChannelKey + chId.HexEncode()
}

func initBroadcast(c *cryptoBroadcast.Channel, e *events, net broadcast.Client,
	broadcastMaker broadcast.NewBroadcastChannelFunc,
	rngGen *fastRNG.StreamGenerator, mr messageReceiveFunc) (
	broadcast.Channel, error) {
	b, err := broadcastMaker(c, net, rngGen)
	if err != nil {
		return nil, err
	}

	err = b.RegisterListener((&userListener{
		chID:      c.ReceptionID,
		trigger:   e.triggerEvent,
		checkSent: mr,
	}).Listen, broadcast.Symmetric)
	if err != nil {
		return nil, err
	}

	err = b.RegisterListener((&adminListener{
		chID:      c.ReceptionID,
		trigger:   e.triggerAdminEvent,
		checkSent: mr,
	}).Listen, broadcast.RSAToPublic)
	if err != nil {
		return nil, err
	}

	return b, nil
}
