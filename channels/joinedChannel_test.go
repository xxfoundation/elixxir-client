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
	"encoding/base64"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"

	clientCmix "gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/collective"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
)

// Tests that manager.store stores the channel list in the ekv.
func Test_manager_setGet(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)
	mFace, err := NewManagerBuilder(pi, kv,
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
	}

	m := mFace.(*manager)

	numtests := 5

	idsList := make([]*id.ID, 0, numtests)

	stream := m.rng.GetStream()
	for i := 0; i < numtests; i++ {
		ch, _, err := newTestChannel(
			"name_"+strconv.Itoa(i), "description_"+strconv.Itoa(i),
			stream, cryptoBroadcast.Public)
		if err != nil {
			t.Errorf("Failed to create new channel %d: %+v", i, err)
		}

		if err = m.addChannel(ch, true); err != nil {
			t.Errorf("Failed to make new broadcast channel: %+v", err)
		}

		idsList = append(idsList, ch.ReceptionID)

	}
	stream.Close()

	chMap, err := m.remote.GetMap(joinedChannelsMap, joinedChannelsMapVersion)
	if err != nil {
		t.Fatalf("failed to get the map")
	}

	for _, chID := range idsList {
		if _, exists := chMap[base64.StdEncoding.EncodeToString(chID[:])]; !exists {
			t.Errorf("channel %s not found in KV: %+v", chID, err)
		}
	}

}

// Tests that the manager.loadChannels loads all the expected channels from the
// ekv.
func Test_manager_loadChannels(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)

	mFace, err := NewManagerBuilder(pi, kv, new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
	}

	m := mFace.(*manager)

	expected := make([]*joinedChannel, 10)

	for i := range expected {
		ch, _, err := newTestChannel(
			"name_"+strconv.Itoa(i), "description_"+strconv.Itoa(i),
			m.rng.GetStream(), cryptoBroadcast.Public)
		if err != nil {
			t.Errorf("Failed to create new channel %d: %+v", i, err)
		}

		err = m.addChannel(ch, true)
		if err != nil {
			t.Errorf("Failed to add new channel %d: %+v", i, err)
		}
	}

	if err != nil {
		t.Errorf("Error storing channels: %+v", err)
	}

	cbs := &dummyUICallback{}

	newManager := &manager{
		channels: make(map[id.ID]*joinedChannel),
		local:    m.local,
		remote:   m.remote,
		net:      m.net,
		rng:      m.rng,
		events: &events{broadcast: newProcessorList(),
			model: &mockEventModel{},
		},
		broadcastMaker: m.broadcastMaker,
		dmCallback:     cbs.DmTokenUpdate,
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
// the local.
func Test_manager_addChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)

	mFace, err := NewManagerBuilder(pi, kv, new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch, true)
	if err != nil {
		t.Errorf("Failed to add new channel: %+v", err)
	}

	if _, exists := m.channels[*ch.ReceptionID]; !exists {
		t.Errorf("Channel %s not added to channel map.", ch.Name)
	}

	_, err = m.remote.GetMapElement(joinedChannelsMap,
		base64.StdEncoding.EncodeToString(ch.ReceptionID[:]), joinedChannelsMapVersion)
	if err != nil {
		t.Errorf("Failed to get joinedChannel from local: %+v", err)
	}

	_, err = m.remote.GetMap(joinedChannelsMap, joinedChannelsMapVersion)
	if err != nil {
		t.Errorf("Failed to get channels from local: %+v", err)
	}
}

// Error path: tests that manager.addChannel returns ChannelAlreadyExistsErr
// when the channel was already added.
func Test_manager_addChannel_ChannelAlreadyExistsErr(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)

	mFace, err := NewManagerBuilder(pi, kv, new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch, true)
	if err != nil {
		t.Errorf("Failed to add new channel: %+v", err)
	}

	err = m.addChannel(ch, true)
	if !errors.Is(err, ChannelAlreadyExistsErr) {
		t.Errorf("Received incorrect error when adding a channel that already "+
			"exists.\nexpected: %s\nreceived: %+v", ChannelAlreadyExistsErr, err)
	}
}

// Tests the manager.removeChannel deletes the channel from the map.
func Test_manager_removeChannel(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)

	mFace, err := NewManagerBuilder(pi, kv, new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch, true)
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

	_, err = m.remote.GetMapElement(joinedChannelsMap,
		base64.StdEncoding.EncodeToString(ch.ReceptionID[:]), joinedChannelsMapVersion)
	if ekv.Exists(err) {
		t.Errorf("joinedChannel not removed from local: %+v", err)
	}
}

// Error path: tests that manager.removeChannel returns ChannelDoesNotExistsErr
// when the channel was never added.
func Test_manager_removeChannel_ChannelDoesNotExistsErr(t *testing.T) {
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)

	mFace, err := NewManagerBuilder(pi, kv, new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
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
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)

	mFace, err := NewManagerBuilder(pi, kv, new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
	}

	m := mFace.(*manager)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), cryptoBroadcast.Public)
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.addChannel(ch, true)
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
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)

	mFace, err := NewManagerBuilder(pi, kv, new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
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
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)

	mFace, err := NewManagerBuilder(pi, kv, new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder, nil, mockAddServiceFn, newMockNM(),
		&dummyUICallback{})
	if err != nil {
		t.Errorf("NewManager error: %+v", err)
	}

	m := mFace.(*manager)

	expected := make([]*id.ID, 10)

	for i := range expected {
		ch, _, err := newTestChannel(
			"name_"+strconv.Itoa(i), "description_"+strconv.Itoa(i),
			m.rng.GetStream(), cryptoBroadcast.Public)
		if err != nil {
			t.Errorf("Failed to create new channel %d: %+v", i, err)
		}
		expected[i] = ch.ReceptionID

		err = m.addChannel(ch, true)
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

// newTestChannel creates a new cryptoBroadcast.Channel in the same way that
// cryptoBroadcast.NewChannel does but with a smaller RSA key and salt to make
// tests run quicker.
func newTestChannel(name, description string, rng csprng.Source,
	level cryptoBroadcast.PrivacyLevel) (
	*cryptoBroadcast.Channel, rsa.PrivateKey, error) {
	c, pk, err := cryptoBroadcast.NewChannelVariableKeyUnsafe(
		name, description, level, netTime.Now(), false, 1000, rng)
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

func (m *mockBroadcastClient) IsHealthy() bool                                        { return true }
func (m *mockBroadcastClient) AddIdentity(*id.ID, time.Time, bool, message.Processor) {}
func (m *mockBroadcastClient) AddIdentityWithHistory(*id.ID, time.Time, time.Time, bool, message.Processor) {
}
func (m *mockBroadcastClient) AddService(*id.ID, message.Service, message.Processor) {}
func (m *mockBroadcastClient) DeleteClientService(*id.ID)                            {}
func (m *mockBroadcastClient) RemoveIdentity(*id.ID)                                 {}
func (m *mockBroadcastClient) GetRoundResults(time.Duration, clientCmix.RoundEventCallback, ...id.Round) {
}
func (m *mockBroadcastClient) AddHealthCallback(func(bool)) uint64 { return 0 }
func (m *mockBroadcastClient) RemoveHealthCallback(uint64)         {}
func (m *mockBroadcastClient) UpsertCompressedService(clientID *id.ID, newService message.CompressedService,
	response message.Processor) {
}

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

	// Used to prevent fix race condition
	sync.Mutex
}

func (m *mockEventModel) getJoinedCh() *cryptoBroadcast.Channel {
	m.Lock()
	defer m.Unlock()
	return m.joinedCh
}

func (m *mockEventModel) getLeftCh() *id.ID {
	m.Lock()
	defer m.Unlock()
	return m.leftCh
}

func (m *mockEventModel) JoinChannel(c *cryptoBroadcast.Channel) {
	m.Lock()
	defer m.Unlock()
	m.joinedCh = c
}

func (m *mockEventModel) LeaveChannel(c *id.ID) {
	m.Lock()
	defer m.Unlock()
	m.leftCh = c
}

func (m *mockEventModel) ReceiveMessage(*id.ID, cryptoMessage.ID,
	string, string, ed25519.PublicKey, uint32, uint8, time.Time, time.Duration,
	rounds.Round, MessageType, SentStatus, bool) uint64 {
	return 0
}

func (m *mockEventModel) ReceiveReply(*id.ID, cryptoMessage.ID,
	cryptoMessage.ID, string, string, ed25519.PublicKey, uint32, uint8,
	time.Time, time.Duration, rounds.Round, MessageType, SentStatus, bool) uint64 {
	return 0
}

func (m *mockEventModel) ReceiveReaction(*id.ID, cryptoMessage.ID,
	cryptoMessage.ID, string, string, ed25519.PublicKey, uint32, uint8,
	time.Time, time.Duration, rounds.Round, MessageType, SentStatus, bool) uint64 {
	return 0
}

func (m *mockEventModel) UpdateFromUUID(uint64, *cryptoMessage.ID, *time.Time,
	*rounds.Round, *bool, *bool, *SentStatus) error {
	panic("implement me")
}

func (m *mockEventModel) UpdateFromMessageID(cryptoMessage.ID, *time.Time,
	*rounds.Round, *bool, *bool, *SentStatus) (uint64, error) {
	panic("implement me")
}

func (m *mockEventModel) GetMessage(cryptoMessage.ID) (ModelMessage, error) {
	panic("implement me")
}
func (m *mockEventModel) DeleteMessage(cryptoMessage.ID) error {
	panic("implement me")
}

func (m *mockEventModel) MuteUser(*id.ID, ed25519.PublicKey, bool) {
	panic("implement me")
}
