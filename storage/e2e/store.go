////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const (
	currentStoreVersion = 0
	packagePrefix       = "e2eSession"
	storeKey            = "Store"
	pubKeyKey           = "DhPubKey"
	privKeyKey          = "DhPrivKey"
	grpKey              = "Group"
)

var NoPartnerErrorStr = "No relationship with partner found"


type Store struct {
	managers map[id.ID]*Manager
	mux      sync.RWMutex

	dhPrivateKey *cyclic.Int
	dhPublicKey  *cyclic.Int
	grp          *cyclic.Group

	kv *versioned.KV

	*fingerprints

	*context
}

func NewStore(grp *cyclic.Group, kv *versioned.KV, privKey *cyclic.Int,
	myID *id.ID, rng *fastRNG.StreamGenerator) (*Store, error) {
	// Generate public key
	pubKey := diffieHellman.GeneratePublicKey(privKey, grp)

	// Modify the prefix of the KV
	kv = kv.Prefix(packagePrefix)

	// Create new fingerprint map
	fingerprints := newFingerprints()

	s := &Store{
		managers: make(map[id.ID]*Manager),

		dhPrivateKey: privKey,
		dhPublicKey:  pubKey,
		grp:          grp,

		fingerprints: &fingerprints,

		kv: kv,

		context: &context{
			fa:   &fingerprints,
			grp:  grp,
			rng:  rng,
			myID: myID,
		},
	}

	err := utility.StoreCyclicKey(kv, pubKey, pubKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to store e2e DH public key")
	}

	err = utility.StoreCyclicKey(kv, privKey, privKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to store e2e DH private key")
	}

	err = utility.StoreGroup(kv, grp, grpKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to store e2e group")
	}

	return s, s.save()
}

func LoadStore(kv *versioned.KV, myID *id.ID, rng *fastRNG.StreamGenerator) (*Store, error) {
	fingerprints := newFingerprints()
	kv = kv.Prefix(packagePrefix)

	grp, err := utility.LoadGroup(kv, grpKey)
	if err != nil {
		return nil, err
	}

	s := &Store{
		managers: make(map[id.ID]*Manager),

		fingerprints: &fingerprints,

		kv: kv,

		context: &context{
			fa:   &fingerprints,
			rng:  rng,
			myID: myID,
			grp:  grp,
		},
	}

	obj, err := kv.Get(storeKey)
	if err != nil {
		return nil, err
	}

	err = s.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	s.context.grp = s.grp

	return s, nil
}

func (s *Store) save() error {
	now := time.Now()

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentStoreVersion,
		Timestamp: now,
		Data:      data,
	}

	return s.kv.Set(storeKey, &obj)
}

func (s *Store) AddPartner(partnerID *id.ID, partnerPubKey, myPrivKey *cyclic.Int,
	sendParams, receiveParams SessionParams) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.managers[*partnerID]; ok {
		return errors.New("Cannot overwrite existing partner")
	}

	m := newManager(s.context, s.kv, partnerID, s.dhPrivateKey, partnerPubKey,
		sendParams, receiveParams)

	s.managers[*partnerID] = m
	if err := s.save(); err != nil {
		jww.FATAL.Printf("Failed to add Parter %s: Save of store failed: %s",
			partnerID, err)
	}

	return nil
}

func (s *Store) GetPartner(partnerID *id.ID) (*Manager, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	m, ok := s.managers[*partnerID]

	if !ok {
		return nil, errors.New(NoPartnerErrorStr)
	}

	return m, nil
}

// PopKey pops a key for use based upon its fingerprint.
func (s *Store) PopKey(f format.Fingerprint) (*Key, bool) {
	return s.fingerprints.Pop(f)
}

// CheckKey checks that a key exists for the key fingerprint.
func (s *Store) CheckKey(f format.Fingerprint) bool {
	return s.fingerprints.Check(f)
}

// GetDHPrivateKey returns the diffie hellman private key.
func (s *Store) GetDHPrivateKey() *cyclic.Int {
	return s.dhPrivateKey
}

// GetDHPublicKey returns the diffie hellman public key.
func (s *Store) GetDHPublicKey() *cyclic.Int {
	return s.dhPublicKey
}

// GetGroup returns the cyclic group used for cMix.
func (s *Store) GetGroup() *cyclic.Group {
	return s.grp
}

// ekv functions

func (s *Store) marshal() ([]byte, error) {
	contacts := make([]id.ID, len(s.managers))

	index := 0
	for partnerID := range s.managers {
		contacts[index] = partnerID
	}

	return json.Marshal(&contacts)
}

func (s *Store) unmarshal(b []byte) error {

	var contacts []id.ID

	err := json.Unmarshal(b, &contacts)

	if err != nil {
		return err
	}

	for _, partnerID := range contacts {
		// Load the relationship. The relationship handles adding the fingerprints via the
		// context object
		manager, err := loadManager(s.context, s.kv, &partnerID)
		if err != nil {
			jww.FATAL.Panicf("Failed to load relationship for partner %s: %s",
				&partnerID, err.Error())
		}

		s.managers[partnerID] = manager
	}

	s.dhPrivateKey, err = utility.LoadCyclicKey(s.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err, "Failed to load e2e DH private key")
	}

	s.dhPublicKey, err = utility.LoadCyclicKey(s.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err, "Failed to load e2e DH public key")
	}

	return nil
}

type fingerprints struct {
	toKey map[format.Fingerprint]*Key
	mux   sync.RWMutex
}

// newFingerprints creates a new fingerprints with an empty map.
func newFingerprints() fingerprints {
	return fingerprints{
		toKey: make(map[format.Fingerprint]*Key),
	}
}

// fingerprints adheres to the fingerprintAccess interface.

func (f *fingerprints) add(keys []*Key) {
	f.mux.Lock()
	defer f.mux.Unlock()

	for _, k := range keys {
		f.toKey[k.Fingerprint()] = k
	}
}

func (f *fingerprints) remove(keys []*Key) {
	f.mux.Lock()
	defer f.mux.Unlock()

	for _, k := range keys {
		delete(f.toKey, k.Fingerprint())
	}
}

func (f *fingerprints) Check(fingerprint format.Fingerprint) bool {
	f.mux.RLock()
	defer f.mux.RUnlock()

	_, ok := f.toKey[fingerprint]
	return ok
}

func (f *fingerprints) Pop(fingerprint format.Fingerprint) (*Key, bool) {
	f.mux.Lock()
	defer f.mux.Unlock()

	key, ok := f.toKey[fingerprint]

	if !ok {
		return nil, false
	}

	delete(f.toKey, fingerprint)

	key.denoteUse()

	key.fp = &fingerprint

	return key, true
}
