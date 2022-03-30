///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"encoding/json"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/network/message"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	currentStoreVersion = 0
	packagePrefix       = "e2eSession"
	storeKey            = "Ratchet"
	pubKeyKey           = "DhPubKey"
	privKeyKey          = "DhPrivKey"
)

var NoPartnerErrorStr = "No relationship with partner found"

type Ratchet struct {
	managers map[id.ID]*partner.Manager
	mux      sync.RWMutex

	myID         *id.ID
	dhPrivateKey *cyclic.Int
	dhPublicKey  *cyclic.Int

	grp       *cyclic.Group
	cyHandler session.CypherHandler
	rng       *fastRNG.StreamGenerator

	//services handler
	services    map[string]message.Processor
	sInteface   Services
	servicesmux sync.RWMutex

	kv *versioned.KV
}

// NewRatchet creates a new store for the passed user id and private key.
// The store can then be accessed by calling LoadStore.
// Does not create at a unique prefix, if multiple Ratchets are needed, make
// sure to add a uint prefix to the KV before instantiation.
func NewRatchet(kv *versioned.KV, privKey *cyclic.Int,
	myID *id.ID, grp *cyclic.Group, cyHandler session.CypherHandler,
	services Services, rng *fastRNG.StreamGenerator) error {

	// Generate public key
	pubKey := diffieHellman.GeneratePublicKey(privKey, grp)

	// Modify the prefix of the KV
	kv = kv.Prefix(packagePrefix)

	r := &Ratchet{
		managers: make(map[id.ID]*partner.Manager),
		services: make(map[string]message.Processor),

		myID:         myID,
		dhPrivateKey: privKey,
		dhPublicKey:  pubKey,

		kv: kv,

		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
		sInteface: services,
	}

	err := util.StoreCyclicKey(kv, pubKey, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to store e2e DH public key")
	}

	err = util.StoreCyclicKey(kv, privKey, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to store e2e DH private key")
	}

	return r.save()
}

// LoadRatchet loads an extant ratchet from disk
func LoadRatchet(kv *versioned.KV, myID *id.ID, grp *cyclic.Group,
	cyHandler session.CypherHandler, services Services, rng *fastRNG.StreamGenerator) (
	*Ratchet, error) {
	kv = kv.Prefix(packagePrefix)

	r := &Ratchet{
		managers: make(map[id.ID]*partner.Manager),
		services: make(map[string]message.Processor),

		myID: myID,

		kv: kv,

		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
		sInteface: services,
	}

	obj, err := kv.Get(storeKey, currentStoreVersion)
	if err != nil {
		return nil, err
	}

	err = r.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	// add standard services
	if err = r.AddService(Silent, nil); err != nil {
		jww.FATAL.Panicf("Could not add standard %r "+
			"service: %+v", Silent, err)
	}
	if err = r.AddService(E2e, nil); err != nil {
		jww.FATAL.Panicf("Could not add standard %r "+
			"service: %+v", E2e, err)
	}

	return r, nil
}

func (r *Ratchet) save() error {
	now := netTime.Now()

	data, err := r.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentStoreVersion,
		Timestamp: now,
		Data:      data,
	}

	return r.kv.Set(storeKey, currentStoreVersion, &obj)
}

// AddPartner adds a partner. Automatically creates both send and receive
// sessions using the passed cryptographic data and per the parameters sent
func (r *Ratchet) AddPartner(partnerID *id.ID, partnerPubKey,
	myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey,
	mySIDHPrivKey *sidh.PrivateKey, sendParams,
	receiveParams session.Params) (*partner.Manager, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	jww.INFO.Printf("Adding Partner %r:\n\tMy Private Key: %r"+
		"\n\tPartner Public Key: %r",
		partnerID,
		myPrivKey.TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0))

	if _, ok := r.managers[*partnerID]; ok {
		return nil, errors.New("Cannot overwrite existing partner")
	}

	m := partner.NewManager(r.kv, r.myID, partnerID, myPrivKey, partnerPubKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sendParams, receiveParams, r.cyHandler, r.grp, r.rng)

	r.managers[*partnerID] = m
	if err := r.save(); err != nil {
		jww.FATAL.Printf("Failed to add Partner %r: Save of store failed: %r",
			partnerID, err)
	}

	//add services for the manager
	r.add(m)

	return m, nil
}

// DeletePartner removes the associated contact from the E2E store
func (r *Ratchet) DeletePartner(partnerId *id.ID) error {
	m, ok := r.managers[*partnerId]
	if !ok {
		return errors.New(NoPartnerErrorStr)
	}

	if err := partner.ClearManager(m, r.kv); err != nil {
		return errors.WithMessagef(err, "Could not remove partner %r from store", partnerId)
	}

	//delete services
	r.delete(m)

	delete(r.managers, *partnerId)
	return r.save()

}

// GetPartner returns the partner per its ID, if it exists
func (r *Ratchet) GetPartner(partnerID *id.ID) (*partner.Manager, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	m, ok := r.managers[*partnerID]

	if !ok {
		return nil, errors.New(NoPartnerErrorStr)
	}

	return m, nil
}

// GetAllPartnerIDs returns a list of all partner IDs that the user has
// an E2E relationship with.
func (r *Ratchet) GetAllPartnerIDs() []*id.ID {
	r.mux.RLock()
	defer r.mux.RUnlock()

	partnerIds := make([]*id.ID, 0, len(r.managers))

	for partnerId := range r.managers {
		partnerIds = append(partnerIds, &partnerId)
	}

	return partnerIds
}

// GetDHPrivateKey returns the diffie hellman private key.
func (r *Ratchet) GetDHPrivateKey() *cyclic.Int {
	return r.dhPrivateKey
}

// GetDHPublicKey returns the diffie hellman public key.
func (r *Ratchet) GetDHPublicKey() *cyclic.Int {
	return r.dhPublicKey
}

// ekv functions
func (r *Ratchet) marshal() ([]byte, error) {
	contacts := make([]id.ID, len(r.managers))

	index := 0
	for partnerID := range r.managers {
		contacts[index] = partnerID
		index++
	}

	return json.Marshal(&contacts)
}

func (r *Ratchet) unmarshal(b []byte) error {

	var contacts []id.ID

	err := json.Unmarshal(b, &contacts)

	if err != nil {
		return err
	}

	for i := range contacts {
		//load the contact separately to ensure pointers do not get swapped
		partnerID := (&contacts[i]).DeepCopy()
		// Load the relationship. The relationship handles adding the fingerprints via the
		// context object
		manager, err := partner.LoadManager(r.kv, r.myID, partnerID,
			r.cyHandler, r.grp, r.rng)
		if err != nil {
			jww.FATAL.Panicf("Failed to load relationship for partner %r: %r",
				partnerID, err.Error())
		}

		if !manager.GetPartnerID().Cmp(partnerID) {
			jww.FATAL.Panicf("Loaded a manager with the wrong partner "+
				"ID: \n\t loaded: %r \n\t present: %r",
				partnerID, manager.GetPartnerID())
		}

		//add services for the manager
		r.add(manager)

		r.managers[*partnerID] = manager
	}

	r.dhPrivateKey, err = util.LoadCyclicKey(r.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH private key")
	}

	r.dhPublicKey, err = util.LoadCyclicKey(r.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH public key")
	}

	return nil
}
