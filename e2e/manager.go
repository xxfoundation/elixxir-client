////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/e2e"
	"sync"

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

	// Generic Callbacks for all E2E operations; by default this is nil and
	// ignored until set via RegisterCallbacks
	callbacks Callbacks
	cbMux     sync.Mutex

	// Partner-specific Callbacks
	partnerCallbacks *partnerCallbacks
}

const legacyE2EKey = "legacyE2ESystem"
const e2eRekeyParamsKey = "e2eRekeyParams"
const e2eRekeyParamsVer = 0

// Init Creates stores. After calling, use load
// Passes the ID public key which is used for the relationship
// uses the passed ID to modify the kv prefix for a unique storage path
func Init(kv *versioned.KV, myID *id.ID, privKey *cyclic.Int,
	grp *cyclic.Group, rekeyParams rekey.Params) error {
	jww.INFO.Printf("Initializing new e2e.Handler for %s", myID.String())
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
	if _, err := kv.Get(e2eRekeyParamsKey, e2eRekeyParamsVer); err == nil {
		if _, err = kv.Get(legacyE2EKey, e2eRekeyParamsVer); err != nil {
			return nil, errors.New("E2E rekey params" +
				" are already on disk, " +
				"LoadLegacy should not be called")
		}
	}

	// Store the rekey params to disk/memory
	err = kv.Set(e2eRekeyParamsKey, e2eRekeyParamsVer, &versioned.Object{
		Version:   e2eRekeyParamsVer,
		Timestamp: netTime.Now(),
		Data:      rekeyParamsData,
	})
	if err != nil {
		return nil, err
	}
	err = kv.Set(legacyE2EKey, e2eRekeyParamsVer, &versioned.Object{
		Version:   e2eRekeyParamsVer,
		Timestamp: netTime.Now(),
		Data:      []byte{1},
	})
	if err != nil {
		return nil, err
	}

	// Load the legacy data
	return loadE2E(kv, net, myID, grp, rng, events)

}

func loadE2E(kv *versioned.KV, net cmix.Client, myDefaultID *id.ID,
	grp *cyclic.Group, rng *fastRNG.StreamGenerator,
	events event.Reporter) (Handler, error) {

	m := &manager{
		Switchboard:      receive.New(),
		net:              net,
		myID:             myDefaultID,
		events:           events,
		grp:              grp,
		rekeyParams:      rekey.Params{},
		kv:               kv,
		callbacks:        nil,
		partnerCallbacks: newPartnerCallbacks(),
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

	// Register listener that calls the ConnectionClosed callback when a
	// catalog.E2eClose message is received
	m.Switchboard.RegisterFunc(
		"connectionClosing", &id.ZeroUser, catalog.E2eClose, m.closeE2eListener)

	return m, nil
}

// RegisterCallbacks registers the Callbacks to E2E. This function overwrite any
// previously saved Callbacks.
func (m *manager) RegisterCallbacks(callbacks Callbacks) {
	m.cbMux.Lock()
	defer m.cbMux.Unlock()
	m.callbacks = callbacks
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
		cmixParams cmix.CMIXParams) (e2e.SendReport, error) {
		// FIXME: we should have access to the e2e params here...
		par := GetDefaultParams()
		par.CMIXParams = cmixParams
		return m.SendE2E(mt, recipient, payload, par)
	}
	rekeyStopper, err := rekey.Start(m.Switchboard, m.Ratchet,
		rekeySendFunc, m.net, m.grp, m.rekeyParams)
	if err != nil {
		return nil, err
	}

	multi.Add(rekeyStopper)

	return multi, nil
}

// DeletePartner removes the contact associated with the partnerId from the E2E
// store.
func (m *manager) DeletePartner(partnerId *id.ID) error {
	err := m.Ratchet.DeletePartner(partnerId)
	if err != nil {
		return err
	}

	m.DeletePartnerCallbacks(partnerId)
	return nil
}

// DeletePartnerNotify removes the contact associated with the partnerId
// from the E2E store. It then sends a critical E2E message to the partner
// informing them that the E2E connection is closed.
func (m *manager) DeletePartnerNotify(partnerId *id.ID, params Params) error {
	// Check if the partner exists
	p, err := m.GetPartner(partnerId)
	if err != nil {
		return err
	}

	// Enable critical message sending
	params.Critical = true

	// Setting the connection fingerprint as the payload allows the receiver
	// to verify that this catalog.E2eClose message is from this specific
	// E2E relationship. However, this is not strictly necessary since this
	// message should not be received by any future E2E connection with the
	// same partner. This is done as a sanity check and to plainly show
	// which relationship this message belongs to.
	payload := p.ConnectionFingerprint().Bytes()

	// Prepare an E2E message informing the partner that you are closing the E2E
	// connection. The send is prepared before deleting the partner because the
	// partner needs to be available to build the E2E message. The message is
	// not sent first to avoid sending the partner an erroneous message of
	// deletion fails.
	sendFunc, err := m.prepareSendE2E(
		catalog.E2eClose, partnerId, payload, params)
	if err != nil {
		return err
	}

	err = m.Ratchet.DeletePartner(partnerId)
	if err != nil {
		return err
	}

	m.DeletePartnerCallbacks(partnerId)

	// Send closing E2E message

	sendReport, err := sendFunc()
	if err != nil {
		jww.ERROR.Printf("Failed to send %s E2E message to %s: %+v",
			catalog.E2eClose, partnerId, err)
	} else {
		jww.INFO.Printf(
			"Sent %s E2E message to %s on rounds %v with message ID %s at %s",
			catalog.E2eClose, partnerId, sendReport.RoundList, sendReport.MessageId, sendReport.SentTime)
	}

	return nil
}

// closeE2eListener calls the ConnectionClose callback when a catalog.E2eClose
// message is received from a partner.
func (m *manager) closeE2eListener(item receive.Message) {
	p, err := m.GetPartner(item.Sender)
	if err != nil {
		jww.ERROR.Printf("Could not find sender %s of %s message: %+v",
			item.Sender, catalog.E2eClose, err)
		return
	}

	// Check the connection fingerprint to verify that the message is
	// from the expected E2E relationship (refer to the comment in
	// DeletePartner for more details)
	if !bytes.Equal(p.ConnectionFingerprint().Bytes(), item.Payload) {
		jww.ERROR.Printf("Received %s message from %s with incorrect "+
			"connection fingerprint %s.", catalog.E2eClose, item.Sender,
			base64.StdEncoding.EncodeToString(item.Payload))
		return
	}

	jww.INFO.Printf("Received %s message from %s for relationship %s. "+
		"Calling ConnectionClosed callback.",
		catalog.E2eClose, item.Sender, p.ConnectionFingerprint())

	if cb := m.partnerCallbacks.get(item.Sender); cb != nil {
		cb.ConnectionClosed(item.Sender, item.Round)
	} else if m.callbacks != nil {
		m.cbMux.Lock()
		m.callbacks.ConnectionClosed(item.Sender, item.Round)
		m.cbMux.Unlock()
	} else {
		jww.INFO.Printf("No ConnectionClosed callback found.")
	}
}

// AddPartnerCallbacks registers a new Callbacks that overrides the generic
// e2e callbacks for the given partner ID.
func (m *manager) AddPartnerCallbacks(partnerID *id.ID, cb Callbacks) {
	m.partnerCallbacks.add(partnerID, cb)
}

// DeletePartnerCallbacks deletes the Callbacks that override the generic
// e2e callback for the given partner ID. Deleting these callbacks will
// result in the generic e2e callbacks being used.
func (m *manager) DeletePartnerCallbacks(partnerID *id.ID) {
	m.partnerCallbacks.delete(partnerID)
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
