package e2e

import (
	"gitlab.com/elixxir/client/e2e/parse"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

type Manager struct {
	*ratchet.Ratchet
	*receive.Switchboard
	partitioner parse.Partitioner
	net         network.Manager
	myID        *id.ID
	rng         *fastRNG.StreamGenerator
	events      event.Manager
	grp         *cyclic.Group
}

//Init Creates stores. After calling, use load
func Init(kv *versioned.KV, myID *id.ID, privKey *cyclic.Int, grp *cyclic.Group) error {
	return ratchet.New(kv, myID, privKey, grp)
}

// Load returns an e2e manager from storage
func Load(kv *versioned.KV, net network.Manager, myID *id.ID,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator, events event.Manager) (*Manager, error) {
	m := &Manager{
		Switchboard: receive.New(),
		partitioner: parse.NewPartitioner(kv, net.GetMaxMessageLength()),
		net:         net,
		myID:        myID,
		events:      events,
		grp:         grp,
	}
	var err error
	m.Ratchet, err = ratchet.Load(kv, myID, grp,
		&fpGenerator{m}, net, rng)
	if err != nil {
		return nil, err
	}
	return m, nil
}
