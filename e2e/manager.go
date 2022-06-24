package e2e

import (
	"encoding/json"
	"strings"
	"time"

	"gitlab.com/xx_network/primitives/netTime"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/parse"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

type manager struct {
	*ratchet.Ratchet
	*receive.Switchboard
	partitioner *parse.Partitioner
	net         cmix.Client
	myID        *id.ID
	rng         *fastRNG.StreamGenerator
	events      event.Reporter
	grp         *cyclic.Group
	crit        *critical
	rekeyParams rekey.Params
	kv          *versioned.KV
}

const e2eRekeyParamsKey = "e2eRekeyParams"
const e2eRekeyParamsVer = 0

// Init Creates stores. After calling, use load
// Passes the ID public key which is used for the relationship
// uses the passed ID to modify the kv prefix for a unique storage path
func Init(kv *versioned.KV, myID *id.ID, privKey *cyclic.Int,
	grp *cyclic.Group, rekeyParams rekey.Params) error {
	kv = kv.Prefix(makeE2ePrefix(myID))
	return initE2E(kv, myID, privKey, grp, rekeyParams)
}

func initE2E(kv *versioned.KV, myID *id.ID, privKey *cyclic.Int,
	grp *cyclic.Group, rekeyParams rekey.Params) error {
	rekeyParamsData, err := json.Marshal(rekeyParams)
	if err != nil {
		return errors.WithMessage(err, "Failed to marshal rekeyParams")
	}
	err = kv.Set(e2eRekeyParamsKey, e2eRekeyParamsVer, &versioned.Object{
		Version:   e2eRekeyParamsVer,
		Timestamp: netTime.Now(),
		Data:      rekeyParamsData,
	})
	if err != nil {
		return errors.WithMessage(err, "Failed to save rekeyParams")
	}
	return ratchet.New(kv, myID, privKey, grp)
}

// Load returns an e2e manager from storage. It uses an ID to prefix the kv
// and is used for partner relationships.
// You can use a memkv for an ephemeral e2e id
// Can be initialized with a nil cmix.Client, but will crash on start - use when
// prebuilding e2e identity to be used later
func Load(kv *versioned.KV, net cmix.Client, myID *id.ID,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator,
	events event.Reporter) (Handler, error) {
	kv = kv.Prefix(makeE2ePrefix(myID))
	return loadE2E(kv, net, myID, grp, rng, events)
}

// LoadLegacy returns an e2e manager from storage
// Passes an ID which is used for relationship with
// partners.
// Does not modify the kv prefix in any way to maintain backwards compatibility
// before multiple IDs were supported
// You can use a memkv for an ephemeral e2e id
// Can be initialized with a nil cmix.Client, but will crash on start - use when
// prebuilding e2e identity to be used later
func LoadLegacy(kv *versioned.KV, net cmix.Client, myID *id.ID,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator,
	events event.Reporter, params rekey.Params) (Handler, error) {

	// Marshal the passed params data
	rekeyParamsData, err := json.Marshal(params)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to marshal rekeyParams")
	}

	// Check if values are already written. If they exist on disk/memory already,
	// this would be a case where LoadLegacy is most likely not the correct
	// code-path the caller should be following.
	if _, err := kv.Get(e2eRekeyParamsKey, e2eRekeyParamsVer); err != nil && !strings.Contains(err.Error(), "object not found") {
		return nil, errors.New("E2E rekey params are already on disk, " +
			"LoadLegacy should not be called")
	}

	// Store the rekey params to disk/memory
	err = kv.Set(e2eRekeyParamsKey, e2eRekeyParamsVer, &versioned.Object{
		Version:   e2eRekeyParamsVer,
		Timestamp: netTime.Now(),
		Data:      rekeyParamsData,
	})

	// Load the legacy data
	return loadE2E(kv, net, myID, grp, rng, events)

}

func loadE2E(kv *versioned.KV, net cmix.Client, myDefaultID *id.ID,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator,
	events event.Reporter) (Handler, error) {

	m := &manager{
		Switchboard: receive.New(),
		net:         net,
		myID:        myDefaultID,
		events:      events,
		grp:         grp,
		rekeyParams: rekey.Params{},
		kv:          kv,
	}
	var err error

	m.Ratchet, err = ratchet.Load(kv, myDefaultID, grp,
		&fpGenerator{m}, net, rng)
	if err != nil {
		return nil, err
	}

	rekeyParams, err := kv.Get(e2eRekeyParamsKey, e2eRekeyParamsVer)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load rekeyParams")
	}
	err = json.Unmarshal(rekeyParams.Data, &m.rekeyParams)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to unmarshal rekeyParams data")
	}

	return m, nil
}

func (m *manager) StartProcesses() (stoppable.Stoppable, error) {
	multi := stoppable.NewMulti("e2eManager")

	if m.partitioner == nil {
		m.partitioner = parse.NewPartitioner(m.kv, m.net.GetMaxMessageLength())
	}

	if m.crit == nil {
		m.crit = newCritical(m.kv, m.net.AddHealthCallback, m.SendE2E)
	}

	critcalNetworkStopper := stoppable.NewSingle(
		"e2eCriticalMessagesStopper")
	go m.crit.runCriticalMessages(critcalNetworkStopper,
		m.net.GetInstance().GetRoundEvents())
	multi.Add(critcalNetworkStopper)

	rekeySendFunc := func(mt catalog.MessageType,
		recipient *id.ID, payload []byte,
		cmixParams cmix.CMIXParams) (
		[]id.Round, e2e.MessageID, time.Time, error) {
		par := GetDefaultParams()
		par.CMIXParams = cmixParams
		return m.SendE2E(mt, recipient, payload, par)
	}
	rekeyStopper, err := rekey.Start(m.Switchboard, m.Ratchet,
		rekeySendFunc, m.net, m.grp, rekey.GetDefaultParams())
	if err != nil {
		return nil, err
	}

	multi.Add(rekeyStopper)

	return multi, nil
}

// EnableUnsafeReception enables the reception of unsafe message by registering
// bespoke services for reception. For debugging only!
func (m *manager) EnableUnsafeReception() {
	m.net.AddService(m.myID, message.Service{
		Identifier: m.myID[:],
		Tag:        ratchet.Silent,
	}, &UnsafeProcessor{
		m:   m,
		tag: ratchet.Silent,
	})
	m.net.AddService(m.myID, message.Service{
		Identifier: m.myID[:],
		Tag:        ratchet.E2e,
	}, &UnsafeProcessor{
		m:   m,
		tag: ratchet.E2e,
	})
}

// HasAuthenticatedChannel returns true if an authenticated channel with the
// partner exists, otherwise returns false
func (m *manager) HasAuthenticatedChannel(partner *id.ID) bool {
	p, err := m.GetPartner(partner)
	return p != nil && err == nil
}

func makeE2ePrefix(myid *id.ID) string {
	return "e2eStore:" + myid.String()
}
