////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"gitlab.com/elixxir/client/broadcast"
	clientCmix "gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// Tests that manager.store stores the channel list in the ekv.
func Test_manager_store(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	for i := 0; i < 10; i++ {
		ch, _, err := newTestChannel(
			"name_"+strconv.Itoa(i), "description_"+strconv.Itoa(i),
			m.rng.GetStream(), cryptoBroadcast.Public)
		if err != nil {
			t.Errorf("Failed to create new channel %d: %+v", i, err)
		}

		b, err := broadcast.NewBroadcastChannel(ch, m.net, m.rng)
		if err != nil {
			t.Errorf("Failed to make new broadcast channel: %+v", err)
		}

		m.channels[*ch.ReceptionID] = &joinedChannel{b}
	}

	err = m.store()
	if err != nil {
		t.Errorf("Error storing channels: %+v", err)
	}

	_, err = m.kv.Get(joinedChannelsKey, joinedChannelsVersion)
	if !ekv.Exists(err) {
		t.Errorf("channel list not found in KV: %+v", err)
	}
}

// Tests that the manager.loadChannels loads all the expected channels from the
// ekv.
func Test_manager_loadChannels(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	expected := make([]*joinedChannel, 10)

	for i := range expected {
		ch, _, err := newTestChannel(
			"name_"+strconv.Itoa(i), "description_"+strconv.Itoa(i), m.rng.GetStream(), cryptoBroadcast.Public)
		if err != nil {
			t.Errorf("Failed to create new channel %d: %+v", i, err)
		}

		b, err := broadcast.NewBroadcastChannel(ch, m.net, m.rng)
		if err != nil {
			t.Errorf("Failed to make new broadcast channel: %+v", err)
		}

		jc := &joinedChannel{b}
		if err = jc.Store(m.kv); err != nil {
			t.Errorf("Failed to store joinedChannel %d: %+v", i, err)
		}

		chID := *ch.ReceptionID
		m.channels[chID] = jc
		expected[i] = jc
	}

	err = m.store()
	if err != nil {
		t.Errorf("Error storing channels: %+v", err)
	}

	newManager := &manager{
		channels:       make(map[id.ID]*joinedChannel),
		kv:             m.kv,
		net:            m.net,
		rng:            m.rng,
		broadcastMaker: m.broadcastMaker,
	}

	newManager.loadChannels()

	for chID, loadedCh := range newManager.channels {
		ch, exists := m.channels[chID]
		if !exists {
			t.Errorf("Channel %s does not exist.", &chID)
		}

		expected := ch.broadcast.Get()
		received := loadedCh.broadcast.Get()

		// NOTE: Times don't compare properly after
		// marshalling due to the monotonic counter
		if expected.Created.Equal(received.Created) {
			expected.Created = received.Created
		}

		if !reflect.DeepEqual(expected, received) {
			t.Errorf("Channel %s does not match loaded channel."+
				"\nexpected: %+v\nreceived: %+v", &chID,
				expected, received)
		}
	}
}

// Tests that manager.addChannel adds the channel to the map and stores it in
// the kv.
func Test_manager_addChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch)
	if err != nil {
		t.Errorf("Failed to add new channel: %+v", err)
	}

	if _, exists := m.channels[*ch.ReceptionID]; !exists {
		t.Errorf("Channel %s not added to channel map.", ch.Name)
	}

	_, err = m.kv.Get(makeJoinedChannelKey(ch.ReceptionID), joinedChannelVersion)
	if err != nil {
		t.Errorf("Failed to get joinedChannel from kv: %+v", err)
	}

	_, err = m.kv.Get(joinedChannelsKey, joinedChannelsVersion)
	if err != nil {
		t.Errorf("Failed to get channels from kv: %+v", err)
	}
}

// Error path: tests that manager.addChannel returns ChannelAlreadyExistsErr
// when the channel was already added.
func Test_manager_addChannel_ChannelAlreadyExistsErr(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch)
	if err != nil {
		t.Errorf("Failed to add new channel: %+v", err)
	}

	err = m.addChannel(ch)
	if err == nil || err != ChannelAlreadyExistsErr {
		t.Errorf("Received incorrect error when adding a channel that already "+
			"exists.\nexpected: %s\nreceived: %+v", ChannelAlreadyExistsErr, err)
	}
}

// Tests the manager.removeChannel deletes the channel from the map.
func Test_manager_removeChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch)
	if err != nil {
		t.Errorf("Failed to add new channel: %+v", err)
	}

	err = m.removeChannel(ch.ReceptionID)
	if err != nil {
		t.Errorf("Error removing channel: %+v", err)
	}

	if _, exists := m.channels[*ch.ReceptionID]; exists {
		t.Errorf("Channel %s was not remove from the channel map.", ch.Name)
	}

	_, err = m.kv.Get(makeJoinedChannelKey(ch.ReceptionID), joinedChannelVersion)
	if ekv.Exists(err) {
		t.Errorf("joinedChannel not removed from kv: %+v", err)
	}
}

// Error path: tests that manager.removeChannel returns ChannelDoesNotExistsErr
// when the channel was never added.
func Test_manager_removeChannel_ChannelDoesNotExistsErr(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.removeChannel(ch.ReceptionID)
	if err == nil || err != ChannelDoesNotExistsErr {
		t.Errorf("Received incorrect error when removing a channel that does "+
			"not exists.\nexpected: %s\nreceived: %+v",
			ChannelDoesNotExistsErr, err)
	}
}

// Tests the manager.getChannel returns the expected channel.
func Test_manager_getChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch)
	if err != nil {
		t.Errorf("Failed to add new channel: %+v", err)
	}

	jc, err := m.getChannel(ch.ReceptionID)
	if err != nil {
		t.Errorf("Error getting channel: %+v", err)
	}

	if !reflect.DeepEqual(ch, jc.broadcast.Get()) {
		t.Errorf("Received unexpected channel.\nexpected: %+v\nreceived: %+v",
			ch, jc.broadcast.Get())
	}
}

// Error path: tests that manager.getChannel returns ChannelDoesNotExistsErr
// when the channel was never added.
func Test_manager_getChannel_ChannelDoesNotExistsErr(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	_, err = m.getChannel(ch.ReceptionID)
	if err == nil || err != ChannelDoesNotExistsErr {
		t.Errorf("Received incorrect error when getting a channel that does "+
			"not exists.\nexpected: %s\nreceived: %+v",
			ChannelDoesNotExistsErr, err)
	}
}

// Tests that manager.getChannels returns all the channels that were added to
// the map.
func Test_manager_getChannels(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	expected := make([]*id.ID, 10)

	for i := range expected {
		ch, _, err := newTestChannel(
			"name_"+strconv.Itoa(i), "description_"+strconv.Itoa(i), m.rng.GetStream(), cryptoBroadcast.Public)
		if err != nil {
			t.Errorf("Failed to create new channel %d: %+v", i, err)
		}
		expected[i] = ch.ReceptionID

		err = m.addChannel(ch)
		if err != nil {
			t.Errorf("Failed to add new channel %d: %+v", i, err)
		}
	}

	channelIDs := m.getChannelsUnsafe()

	sort.SliceStable(expected, func(i, j int) bool {
		return bytes.Compare(expected[i][:], expected[j][:]) == -1
	})
	sort.SliceStable(channelIDs, func(i, j int) bool {
		return bytes.Compare(channelIDs[i][:], channelIDs[j][:]) == -1
	})

	if !reflect.DeepEqual(expected, channelIDs) {
		t.Errorf("ID list does not match expected.\nexpected: %v\nreceived: %v",
			expected, channelIDs)
	}
}

// Tests that joinedChannel.Store saves the joinedChannel to the expected place
// in the ekv.
func Test_joinedChannel_Store(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	ch, _, err := newTestChannel(
		"name", "description", rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	b, err := broadcast.NewBroadcastChannel(ch, new(mockBroadcastClient), rng)
	if err != nil {
		t.Errorf("Failed to create new broadcast channel: %+v", err)
	}

	jc := &joinedChannel{b}

	err = jc.Store(kv)
	if err != nil {
		t.Errorf("Error storing joinedChannel: %+v", err)
	}

	_, err = kv.Get(makeJoinedChannelKey(ch.ReceptionID), joinedChannelVersion)
	if !ekv.Exists(err) {
		t.Errorf("joinedChannel not found in KV: %+v", err)
	}
}

// Tests that loadJoinedChannel returns a joinedChannel from storage that
// matches the original.
func Test_loadJoinedChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch)
	if err != nil {
		t.Errorf("Failed to add channel: %+v", err)
	}

	loadedJc, err := loadJoinedChannel(ch.ReceptionID, m.kv, m.net, m.rng,
		m.events, m.broadcastMaker, func(messageID cryptoChannel.MessageID, r rounds.Round) bool {
			return false
		})
	if err != nil {
		t.Errorf("Failed to load joinedChannel: %+v", err)
	}

	expected := *ch
	received := *loadedJc.broadcast.Get()
	// NOTE: Times don't compare properly after marshalling due to the
	// monotonic counter
	if expected.Created.Equal(received.Created) {
		expected.Created = received.Created
	}

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("Loaded joinedChannel does not match original."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that joinedChannel.delete deletes the stored joinedChannel from the
// ekv.
func Test_joinedChannel_delete(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	ch, _, err := newTestChannel(
		"name", "description", rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	b, err := broadcast.NewBroadcastChannel(ch, new(mockBroadcastClient), rng)
	if err != nil {
		t.Errorf("Failed to create new broadcast channel: %+v", err)
	}

	jc := &joinedChannel{b}

	err = jc.Store(kv)
	if err != nil {
		t.Errorf("Error storing joinedChannel: %+v", err)
	}

	err = jc.delete(kv)
	if err != nil {
		t.Errorf("Error deleting joinedChannel: %+v", err)
	}

	_, err = kv.Get(makeJoinedChannelKey(ch.ReceptionID), joinedChannelVersion)
	if ekv.Exists(err) {
		t.Errorf("joinedChannel found in KV: %+v", err)
	}
}

// Consistency test of makeJoinedChannelKey.
func Test_makeJoinedChannelKey_Consistency(t *testing.T) {
	values := map[*id.ID]string{
		id.NewIdFromUInt(0, id.User, t): "JoinedChannelKey-0x0000000000000000000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(1, id.User, t): "JoinedChannelKey-0x0000000000000001000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(2, id.User, t): "JoinedChannelKey-0x0000000000000002000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(3, id.User, t): "JoinedChannelKey-0x0000000000000003000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(4, id.User, t): "JoinedChannelKey-0x0000000000000004000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(5, id.User, t): "JoinedChannelKey-0x0000000000000005000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(6, id.User, t): "JoinedChannelKey-0x0000000000000006000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(7, id.User, t): "JoinedChannelKey-0x0000000000000007000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(8, id.User, t): "JoinedChannelKey-0x0000000000000008000000000000000000000000000000000000000000000000",
		id.NewIdFromUInt(9, id.User, t): "JoinedChannelKey-0x0000000000000009000000000000000000000000000000000000000000000000",
	}

	for chID, expected := range values {
		key := makeJoinedChannelKey(chID)

		if expected != key {
			t.Errorf("Unexpected key for ID %d.\nexpected: %s\nreceived: %s",
				binary.BigEndian.Uint64(chID[:8]), expected, key)
		}
	}

}

// newTestChannel creates a new cryptoBroadcast.Channel in the same way that
// cryptoBroadcast.NewChannel does but with a smaller RSA key and salt to make
// tests run quicker.
func newTestChannel(name, description string, rng csprng.Source,
	level cryptoBroadcast.PrivacyLevel) (
	*cryptoBroadcast.Channel, rsa.PrivateKey, error) {
	c, pk, err := cryptoBroadcast.NewChannelVariableKeyUnsafe(
		name, description, level, time.Now(), 1000, 512, rng)
	return c, pk, err
}

////////////////////////////////////////////////////////////////////////////////
// Mock Broadcast Client                                                      //
////////////////////////////////////////////////////////////////////////////////

// mockBroadcastClient adheres to the broadcast.Client interface.
type mockBroadcastClient struct{}

func (m *mockBroadcastClient) GetMaxMessageLength() int { return 123 }

func (m *mockBroadcastClient) SendWithAssembler(*id.ID,
	clientCmix.MessageAssembler, clientCmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	return rounds.Round{ID: id.Round(567)}, ephemeral.Id{}, nil
}

func (m *mockBroadcastClient) IsHealthy() bool                     { return true }
func (m *mockBroadcastClient) AddIdentity(*id.ID, time.Time, bool) {}
func (m *mockBroadcastClient) AddIdentityWithHistory(id *id.ID, validUntil, beginning time.Time, persistent bool) {
}
func (m *mockBroadcastClient) AddService(*id.ID, message.Service, message.Processor) {}
func (m *mockBroadcastClient) DeleteClientService(*id.ID)                            {}
func (m *mockBroadcastClient) RemoveIdentity(*id.ID)                                 {}
func (m *mockBroadcastClient) GetRoundResults(time.Duration, clientCmix.RoundEventCallback, ...id.Round) {
}
func (m *mockBroadcastClient) AddHealthCallback(func(bool)) uint64 { return 0 }
func (m *mockBroadcastClient) RemoveHealthCallback(uint64)         {}

////////////////////////////////////////////////////////////////////////////////
// Mock EventModel                                                            //
////////////////////////////////////////////////////////////////////////////////

func mockEventModelBuilder(string) (EventModel, error) {
	return &mockEventModel{}, nil
}

// mockEventModel adheres to the EventModel interface.
type mockEventModel struct {
	joinedCh *cryptoBroadcast.Channel
	leftCh   *id.ID
}

func (m *mockEventModel) JoinChannel(c *cryptoBroadcast.Channel) { m.joinedCh = c }
func (m *mockEventModel) LeaveChannel(c *id.ID)                  { m.leftCh = c }

func (m *mockEventModel) ReceiveMessage(*id.ID, cryptoChannel.MessageID, string,
	string, ed25519.PublicKey, uint8, time.Time, time.Duration, rounds.Round,
	MessageType, SentStatus) uint64 {
	return 0
}

func (m *mockEventModel) ReceiveReply(*id.ID, cryptoChannel.MessageID,
	cryptoChannel.MessageID, string, string, ed25519.PublicKey, uint8,
	time.Time, time.Duration, rounds.Round, MessageType, SentStatus) uint64 {
	return 0
}

func (m *mockEventModel) ReceiveReaction(*id.ID, cryptoChannel.MessageID,
	cryptoChannel.MessageID, string, string, ed25519.PublicKey, uint8,
	time.Time, time.Duration, rounds.Round, MessageType, SentStatus) uint64 {
	return 0
}

func (m *mockEventModel) UpdateSentStatus(uint64, cryptoChannel.MessageID,
	time.Time, rounds.Round, SentStatus) {
	// TODO implement me
	panic("implement me")
}
