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

// store Stores the list of joined channels to disk while taking the read lock
func (m *manager) store() error {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.storeUnsafe()
}

// storeUnsafe Stores the list of joined channels to disk without taking the
// read lock. Must be used by another function which has already taken the read
// lock
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

	return m.kv.Set(joinedChannelsKey,
		joinedChannelsVersion, obj)
}

// loadChannels loads all currently joined channels from disk and registers
// them for message reception
func (m *manager) loadChannels() map[*id.ID]*joinedChannel {

	obj, err := m.kv.Get(joinedChannelsKey,
		joinedChannelsVersion)
	if err != nil {
		jww.FATAL.Panicf("Failed to load channels %+v", err)
	}

	chList := make([]*id.ID, 0, len(m.channels))
	if err = json.Unmarshal(obj.Data, &chList); err != nil {
		jww.FATAL.Panicf("Failed to load channels %+v", err)
	}

	chMap := make(map[*id.ID]*joinedChannel)

	for i := range chList {
		jc, err := loadJoinedChannel(chList[i], m.kv, m.client, m.rng, m.name,
			&m.events, m.broadcastMaker)
		if err != nil {
			jww.FATAL.Panicf("Failed to load channel %s:  %+v",
				chList[i], err)
		}
		chMap[chList[i]] = jc
	}
	return chMap
}

//addChannel Adds a channel
func (m *manager) addChannel(channel cryptoBroadcast.Channel) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	if _, exists := m.channels[channel.ReceptionID]; exists {
		return ChannelAlreadyExistsErr
	}

	b, err := m.broadcastMaker(channel, m.client, m.rng)
	if err != nil {
		return err
	}

	//Connect to listeners
	err = b.RegisterListener((&userListener{
		name:   m.name,
		events: &m.events,
		chID:   channel.ReceptionID,
	}).Listen, broadcast.Symmetric)
	if err != nil {
		return err
	}

	err = b.RegisterListener((&adminListener{
		name:   m.name,
		events: &m.events,
		chID:   channel.ReceptionID,
	}).Listen, broadcast.Asymmetric)
	if err != nil {
		return err
	}

	jc := &joinedChannel{
		broadcast: b,
	}

	if err = jc.Store(m.kv); err != nil {
		go b.Stop()
		return err
	}

	if err = m.storeUnsafe(); err != nil {
		go b.Stop()
		return err
	}
	return nil
}

func (m *manager) removeChannel(channelId *id.ID) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	ch, exists := m.channels[channelId]
	if !exists {
		return ChannelDoesNotExistsErr
	}

	ch.broadcast.Stop()

	delete(m.channels, channelId)

	return nil
}

//getChannel returns the given channel, if it exists
func (m *manager) getChannel(channelId *id.ID) (*joinedChannel, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	jc, exists := m.channels[channelId]
	if !exists {
		return nil, ChannelDoesNotExistsErr
	}

	return jc, nil
}

//getChannels returns the ids of all channels that have been joined
//use getChannelsUnsafe if you already have taken the mux
func (m *manager) getChannels() []*id.ID {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.getChannelsUnsafe()
}

//getChannelsUnsafe returns the ids of all channels that have been joined
//is unsafe because it does not take the mux, only use when under a lock.
func (m *manager) getChannelsUnsafe() []*id.ID {
	list := make([]*id.ID, 0, len(m.channels))
	for chID := range m.channels {
		list = append(list, chID)
	}
	return list
}

// joinedChannel which holds channel info. Will expand to include admin data,
// so will be treated as a struct for now
type joinedChannel struct {
	broadcast broadcast.Channel
}

// joinedChannelDisk is the representation for storage
type joinedChannelDisk struct {
	broadcast cryptoBroadcast.Channel
}

//Store writes the given channel to a unique storage location within the EKV
func (jc *joinedChannel) Store(kv *versioned.KV) error {
	jcd := joinedChannelDisk{broadcast: jc.broadcast.Get()}
	data, err := json.Marshal(&jcd)
	if err != nil {
		return err
	}
	obj := &versioned.Object{
		Version:   joinedChannelVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return kv.Set(makeJoinedChannelKey(jc.broadcast.Get().ReceptionID),
		joinedChannelVersion, obj)
}

//loadJoinedChannel loads a given channel from ekv storage
func loadJoinedChannel(chId *id.ID, kv *versioned.KV, net broadcast.Client,
	rngGen *fastRNG.StreamGenerator, name NameService, e *events,
	broadcastMaker broadcast.NewBroadcastChannelFunc) (*joinedChannel, error) {
	obj, err := kv.Get(makeJoinedChannelKey(chId), joinedChannelVersion)
	if err != nil {
		return nil, err
	}

	jcd := &joinedChannelDisk{}

	err = json.Unmarshal(obj.Data, jcd)
	if err != nil {
		return nil, err
	}
	b, err := broadcastMaker(jcd.broadcast, net, rngGen)
	if err != nil {
		return nil, err
	}

	err = b.RegisterListener((&userListener{
		name:   name,
		events: e,
		chID:   jcd.broadcast.ReceptionID,
	}).Listen, broadcast.Symmetric)
	if err != nil {
		return nil, err
	}

	err = b.RegisterListener((&adminListener{
		name:   name,
		events: e,
		chID:   jcd.broadcast.ReceptionID,
	}).Listen, broadcast.Asymmetric)
	if err != nil {
		return nil, err
	}

	jc := &joinedChannel{broadcast: b}
	return jc, nil
}

func makeJoinedChannelKey(chId *id.ID) string {
	return joinedChannelKey + chId.HexEncode()
}
