////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/xx_network/primitives/id"
)

const (
	joinedChannelsMapVersion = 0
	joinedChannelsMap        = "JoinedChannelsMap"
)

// loadChannels loads all currently joined channels from disk and registers them
// for message reception.
func (m *manager) loadChannels() {
	m.mux.Lock()
	m.channels = make(map[id.ID]*joinedChannel)
	m.mux.Unlock()
	err := m.remote.ListenOnRemoteMap(joinedChannelsMap, joinedChannelsMapVersion,
		m.mapUpdate, false)

	if err != nil {
		jww.FATAL.Panicf("Failed to set up listener on remote for "+
			"channels: %+v", err)
	}

}

func (m *manager) mapUpdate(edits map[string]versioned.ElementEdit) {
	m.mux.Lock()
	defer m.mux.Unlock()

	jww.DEBUG.Printf("[CH] Applying mapUpdate: %d", len(edits))

	// Keep count of Channel changes
	channelsChanged := 0

	for elementName, edit := range edits {
		channelID := &id.ID{}
		elementBytes, err := base64.StdEncoding.DecodeString(
			elementName)
		if err != nil {
			jww.WARN.Printf("[CH] Failed to unmarshal channel ID "+
				"in remote channel %s, skipping: %+v",
				elementName, err)
			continue
		}
		copy(channelID[:], elementBytes)

		if edit.Operation == versioned.Deleted {
			if err := m.removeChannelUnsafe(channelID); err != nil {
				jww.WARN.Printf("[CH] Failed to remove "+
					"channel on instruction from remote "+
					"%s: %+v", channelID, err)
			}
			m.events.model.LeaveChannel(channelID)
			channelsChanged += 1
			continue
		} else if edit.Operation == versioned.Updated {
			jc, err := m.getChannelUnsafe(channelID)
			if err != nil {
				jww.WARN.Printf("[CH] Failed to update "+
					"channel on instruction from remote "+
					"%s: %+v", channelID,
					err)
				continue
			}
			jcd := &joinedChannelDisk{}
			err = json.Unmarshal(edit.NewElement.Data, jcd)
			if err != nil {
				jww.WARN.Printf("[CH] Failed to update "+
					"channel on instruction from remote "+
					"%s: %+v", channelID, err)
				continue
			}
			jc.dmEnabled = jcd.DmEnabled
			go m.dmCallback(channelID, jc.dmEnabled)
			channelsChanged += 1
		}

		jc, err := m.setUpJoinedChannel(edit.NewElement.Data)
		if err != nil {
			jww.WARN.Printf("[CH] Failed to set up channel %s "+
				"passed by remote, skipping: %+v",
				channelID, err)
			continue
		}
		m.events.model.JoinChannel(jc.broadcast.Get())
		channelsChanged += 1
	}

	if channelsChanged == 0 {
		jww.WARN.Printf("[CH] Received empty update from remote in " +
			"join channels")
	}
}

// addChannel adds a channel.
func (m *manager) addChannel(channel *cryptoBroadcast.Channel, dmEnabled bool) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	jc, err := m.addChannelInternal(channel, dmEnabled)
	if err != nil {
		return err
	}

	err = m.saveChannel(jc)
	if err != nil {
		return err
	}
	m.notifications.addChannel(channel.ReceptionID)
	return nil
}

func (m *manager) saveChannel(jc *joinedChannel) error {
	elementName := base64.StdEncoding.EncodeToString(jc.broadcast.Get().ReceptionID[:])

	jcBytes, err := jc.Marshal()
	if err != nil {
		return err
	}

	return m.remote.StoreMapElement(joinedChannelsMap, elementName, &versioned.Object{
		Version:   joinedChannelsMapVersion,
		Timestamp: time.Time{},
		Data:      jcBytes,
	}, joinedChannelsMapVersion)
}

// addChannel adds a channel.
func (m *manager) addChannelInternal(channel *cryptoBroadcast.Channel,
	dmEnabled bool) (*joinedChannel, error) {
	if _, exists := m.channels[*channel.ReceptionID]; exists {
		return nil, errors.WithStack(ChannelAlreadyExistsErr)
	}

	b, err := m.broadcastMaker(channel, m.net, m.rng)
	if err != nil {
		return nil, err
	}

	jc := &joinedChannel{broadcast: b, dmEnabled: dmEnabled}

	m.channels[*jc.broadcast.Get().ReceptionID] = jc

	// Connect to listeners
	_, err = m.registerListeners(b, channel)

	return jc, nil
}

// removeChannel deletes the channel with the given ID from the channel list and
// stops it from broadcasting. Returns ChannelDoesNotExistsErr error if the
// channel does not exist.
func (m *manager) removeChannel(channelID *id.ID) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if err := m.removeChannelUnsafe(channelID); err != nil {
		return err
	}
	m.notifications.removeChannel(channelID)
	return nil
}

func (m *manager) removeChannelUnsafe(channelID *id.ID) error {
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

	_, err = m.remote.DeleteMapElement(joinedChannelsMap,
		base64.StdEncoding.EncodeToString(channelID[:]), joinedChannelsMapVersion)

	return err
}

// getChannel returns the given channel. Returns ChannelDoesNotExistsErr error
// if the channel does not exist.
func (m *manager) getChannel(channelID *id.ID) (*joinedChannel, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.getChannelUnsafe(channelID)
}

// getChannelUnsafe returns the given channel. Returns ChannelDoesNotExistsErr error
// if the channel does not exist. Does not take the lock
func (m *manager) getChannelUnsafe(channelID *id.ID) (*joinedChannel, error) {
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
	dmEnabled bool
}

// joinedChannelDisk is the representation of joinedChannel for storage.
type joinedChannelDisk struct {
	Broadcast *cryptoBroadcast.Channel `json:"broadcast"`
	DmEnabled bool                     `json:"dmEnabled"`
}

// Marshal marshals a given channel to bytes.
func (jc *joinedChannel) Marshal() ([]byte, error) {
	jcd := joinedChannelDisk{
		Broadcast: jc.broadcast.Get(),
		DmEnabled: jc.dmEnabled,
	}
	return json.Marshal(&jcd)
}

// Unmarshal loads a given channel from ekv storage.
func (m *manager) setUpJoinedChannel(b []byte) (*joinedChannel, error) {
	jcd := &joinedChannelDisk{}
	err := json.Unmarshal(b, jcd)
	if err != nil {
		return nil, err
	}

	bc, err := m.initBroadcast(jcd.Broadcast)
	if err != nil {
		return nil, err
	}

	jc := &joinedChannel{broadcast: bc, dmEnabled: jcd.DmEnabled}

	m.channels[*jc.broadcast.Get().ReceptionID] = jc

	return jc, nil
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
