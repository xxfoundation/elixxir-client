package channels

import (
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type manager struct {
	//List of all channels
	channels map[*id.ID]*joinedChannel
	mux      sync.RWMutex

	//External references
	kv     *versioned.KV
	client broadcast.Client
	rng    *fastRNG.StreamGenerator
	name   NameService

	//Events model
	events

	// make the function used to create broadcasts be a pointer so it
	// can be replaced in tests
	broadcastMaker broadcast.NewBroadcastChannelFunc
}

func NewManager(kv *versioned.KV, client broadcast.Client,
	rng *fastRNG.StreamGenerator, name NameService) Manager {

	//prefix the kv with the username so multiple can be run
	kv = kv.Prefix(name.GetUsername())

	m := manager{
		kv:             kv,
		client:         client,
		rng:            rng,
		name:           name,
		events:         events{},
		broadcastMaker: broadcast.NewBroadcastChannel,
	}

	m.loadChannels()

	return &m
}

func (m *manager) JoinChannel(channel cryptoBroadcast.Channel) error {
	err := m.addChannel(channel)
	if err != nil {
		return err
	}
	go m.events.model.JoinChannel(channel)
	return nil
}

func (m *manager) LeaveChannel(channelId *id.ID) error {
	err := m.removeChannel(channelId)
	if err != nil {
		return err
	}
	go m.events.model.LeaveChannel(channelId)
	return nil
}
