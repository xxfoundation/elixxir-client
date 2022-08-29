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
	broadcastMaker broadcast.NewBroadcastChannelFunc
}

func NewManager() {

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
