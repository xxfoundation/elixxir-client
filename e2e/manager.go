package e2e

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/parse"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/network/message"
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
	myDefaultID *id.ID
	rng         *fastRNG.StreamGenerator
	events      event.Manager
	grp         *cyclic.Group
	crit        *critical
	rekeyParams rekey.Params
}

const e2eRekeyParamsKey = "e2eRekeyParams"
const e2eRekeyParamsVer = 0

// Init Creates stores. After calling, use load
// Passes a default ID and public key which is used for relationship with
// partners when no default ID is selected
func Init(kv *versioned.KV, myDefaultID *id.ID, privKey *cyclic.Int,
	grp *cyclic.Group, rekeyParams rekey.Params) error {
	rekeyParamsData, err := json.Marshal(rekeyParams)
	if err != nil {
		return errors.WithMessage(err, "Failed to marshal rekeyParams")
	}
	err = kv.Set(e2eRekeyParamsKey, e2eRekeyParamsVer, &versioned.Object{
		Version:   e2eRekeyParamsVer,
		Timestamp: time.Now(),
		Data:      rekeyParamsData,
	})
	if err != nil {
		return errors.WithMessage(err, "Failed to save rekeyParams")
	}
	return ratchet.New(kv, myDefaultID, privKey, grp)
}

// Load returns an e2e manager from storage
// Passes a default ID which is used for relationship with
// partners when no default ID is selected
func Load(kv *versioned.KV, net network.Manager, myDefaultID *id.ID,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator, events event.Manager) (Handler, error) {

	//build the manager
	m := &manager{
		Switchboard: receive.New(),
		partitioner: parse.NewPartitioner(kv, net.GetMaxMessageLength()),
		net:         net,
		myDefaultID: myDefaultID,
		events:      events,
		grp:         grp,
		rekeyParams: rekey.Params{},
	}
	var err error

	//load the ratchets
	m.Ratchet, err = ratchet.Load(kv, myDefaultID, grp,
		&fpGenerator{m}, net, rng)
	if err != nil {
		return nil, err
	}

	//load rekey params here
	rekeyParams, err := kv.Get(e2eRekeyParamsKey, e2eRekeyParamsVer)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load rekeyParams")
	}
	err = json.Unmarshal(rekeyParams.Data, &m.rekeyParams)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal rekeyParams data")
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

// EnableUnsafeReception enables the reception of unsafe message by registering
// bespoke services for reception. For debugging only!
func (m *manager) EnableUnsafeReception() {
	m.net.AddService(m.myDefaultID, message.Service{
		Identifier: m.myDefaultID[:],
		Tag:        ratchet.Silent,
	}, &UnsafeProcessor{
		m:   m,
		tag: ratchet.Silent,
	})
	m.net.AddService(m.myDefaultID, message.Service{
		Identifier: m.myDefaultID[:],
		Tag:        ratchet.E2e,
	}, &UnsafeProcessor{
		m:   m,
		tag: ratchet.E2e,
	})
}
