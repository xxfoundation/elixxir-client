package e2e

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/parse"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type manager struct {
	*ratchet.Ratchet
	*receive.Switchboard
	partitioner parse.Partitioner
	net         network.Manager
	myID        *id.ID
	rng         *fastRNG.StreamGenerator
	events      event.Manager
	grp         *cyclic.Group
	crit        *critical
}

//Init Creates stores. After calling, use load
func Init(kv *versioned.KV, myID *id.ID, privKey *cyclic.Int, grp *cyclic.Group) error {
	return ratchet.New(kv, myID, privKey, grp)
}

// Load returns an e2e manager from storage
func Load(kv *versioned.KV, net network.Manager, myID *id.ID,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator, events event.Manager) (Handler, error) {

	//build the manager
	m := &manager{
		Switchboard: receive.New(),
		partitioner: parse.NewPartitioner(kv, net.GetMaxMessageLength()),
		net:         net,
		myID:        myID,
		events:      events,
		grp:         grp,
	}
	var err error

	//load the ratchets
	m.Ratchet, err = ratchet.Load(kv, myID, grp,
		&fpGenerator{m}, net, rng)
	if err != nil {
		return nil, err
	}

	//attach critical messages
	m.crit = newCritical(kv, net.AddHealthCallback,
		net.GetInstance().GetRoundEvents(), m.SendE2E)

	return m, nil
}

func (m *manager) StartProcesses() (stoppable.Stoppable, error) {
	multi := stoppable.NewMulti("e2eManager")

	critcalNetworkStopper := stoppable.NewSingle("e2eCriticalMessagesStopper")
	m.crit.runCriticalMessages(critcalNetworkStopper)
	multi.Add(critcalNetworkStopper)

	rekeySendFunc := func(mt catalog.MessageType, recipient *id.ID, payload []byte,
		cmixParams network.CMIXParams) (
		[]id.Round, e2e.MessageID, time.Time, error) {
		par := GetDefaultParams()
		par.CMIX = cmixParams
		return m.SendE2E(mt, recipient, payload, par)
	}
	rekeyStopper, err := rekey.Start(m.Switchboard, m.Ratchet, rekeySendFunc, m.net, m.grp,
		rekey.GetDefaultParams())
	if err != nil {
		return nil, err
	}

	multi.Add(rekeyStopper)

	return multi, nil
}
