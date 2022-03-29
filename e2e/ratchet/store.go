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
	storeKey            = "Store"
	pubKeyKey           = "DhPubKey"
	privKeyKey          = "DhPrivKey"
	sidhPubKeyKey       = "SidhPubKey"
	sidhPrivKeyKey      = "SidhPrivKey"
)

var NoPartnerErrorStr = "No relationship with partner found"

type Store struct {
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

func NewStore(kv *versioned.KV, privKey *cyclic.Int,
	myID *id.ID, grp *cyclic.Group, cyHandler session.CypherHandler,
	rng *fastRNG.StreamGenerator) (*Store, error) {
	// Generate public key
	pubKey := diffieHellman.GeneratePublicKey(privKey, grp)

	// Modify the prefix of the KV
	kv = kv.Prefix(packagePrefix)

	s := &Store{
		managers: make(map[id.ID]*partner.Manager),

		myID:         myID,
		dhPrivateKey: privKey,
		dhPublicKey:  pubKey,

		kv: kv,

		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
	}

	err := util.StoreCyclicKey(kv, pubKey, pubKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to store e2e DH public key")
	}

	err = util.StoreCyclicKey(kv, privKey, privKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to store e2e DH private key")
	}

	return s, s.save()
}

func LoadStore(kv *versioned.KV, myID *id.ID, grp *cyclic.Group,
	cyHandler session.CypherHandler, rng *fastRNG.StreamGenerator) (
	*Store, error) {
	kv = kv.Prefix(packagePrefix)

	s := &Store{
		managers: make(map[id.ID]*partner.Manager),

		myID: myID,

		kv: kv,

		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
	}

	obj, err := kv.Get(storeKey, currentStoreVersion)
	if err != nil {
		return nil, err
	}

	err = s.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) save() error {
	now := netTime.Now()

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentStoreVersion,
		Timestamp: now,
		Data:      data,
	}

	return s.kv.Set(storeKey, currentStoreVersion, &obj)
}

func (s *Store) AddPartner(partnerID *id.ID, partnerPubKey,
	myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey,
	mySIDHPrivKey *sidh.PrivateKey, sendParams,
	receiveParams session.Params) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	jww.INFO.Printf("Adding Partner %s:\n\tMy Private Key: %s"+
		"\n\tPartner Public Key: %s",
		partnerID,
		myPrivKey.TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0))

	if _, ok := s.managers[*partnerID]; ok {
		return errors.New("Cannot overwrite existing partner")
	}

	m := partner.NewManager(s.kv, s.myID, partnerID, myPrivKey, partnerPubKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sendParams, receiveParams, s.cyHandler, s.grp, s.rng)

	s.managers[*partnerID] = m
	if err := s.save(); err != nil {
		jww.FATAL.Printf("Failed to add Partner %s: Save of store failed: %s",
			partnerID, err)
	}

	return nil
}

// DeletePartner removes the associated contact from the E2E store
func (s *Store) DeletePartner(partnerId *id.ID) error {
	m, ok := s.managers[*partnerId]
	if !ok {
		return errors.New(NoPartnerErrorStr)
	}

	if err := partner.ClearManager(m, s.kv); err != nil {
		return errors.WithMessagef(err, "Could not remove partner %s from store", partnerId)
	}

	delete(s.managers, *partnerId)
	return s.save()
}

func (s *Store) GetPartner(partnerID *id.ID) (*partner.Manager, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	m, ok := s.managers[*partnerID]

	if !ok {
		return nil, errors.New(NoPartnerErrorStr)
	}

	return m, nil
}

// GetAllPartnerIDs returns a list of all partner IDs that the user has
// an E2E relationship with.
func (s *Store) GetAllPartnerIDs() []*id.ID {
	s.mux.RLock()
	defer s.mux.RUnlock()

	partnerIds := make([]*id.ID, 0, len(s.managers))

	for partnerId := range s.managers {
		partnerIds = append(partnerIds, &partnerId)
	}

	return partnerIds
}

// GetDHPrivateKey returns the diffie hellman private key.
func (s *Store) GetDHPrivateKey() *cyclic.Int {
	return s.dhPrivateKey
}

// GetDHPublicKey returns the diffie hellman public key.
func (s *Store) GetDHPublicKey() *cyclic.Int {
	return s.dhPublicKey
}

// ekv functions
func (s *Store) marshal() ([]byte, error) {
	contacts := make([]id.ID, len(s.managers))

	index := 0
	for partnerID := range s.managers {
		contacts[index] = partnerID
		index++
	}

	return json.Marshal(&contacts)
}

func (s *Store) unmarshal(b []byte) error {

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
		manager, err := partner.LoadManager(s.kv, s.myID, partnerID,
			s.cyHandler, s.grp, s.rng)
		if err != nil {
			jww.FATAL.Panicf("Failed to load relationship for partner %s: %s",
				partnerID, err.Error())
		}

		if !manager.GetPartnerID().Cmp(partnerID) {
			jww.FATAL.Panicf("Loaded a manager with the wrong partner "+
				"ID: \n\t loaded: %s \n\t present: %s",
				partnerID, manager.GetPartnerID())
		}

		s.managers[*partnerID] = manager
	}

	s.dhPrivateKey, err = util.LoadCyclicKey(s.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH private key")
	}

	s.dhPublicKey, err = util.LoadCyclicKey(s.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH public key")
	}

	return nil
}
