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
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const currentStoreVersion = 0
const storeKey = "e2eKeyStore"
const pubKeyKey = "e2eDhPubKey"
const privKeyKey = "e2eDhPrivKey"
const grpKey = "e2eGroupKey"

type Store struct {
	managers map[id.ID]*Manager
	mux      sync.RWMutex

	dhPrivateKey *cyclic.Int
	dhPublicKey  *cyclic.Int
	grp          *cyclic.Group

	fingerprints

	context
}

func NewStore(grp *cyclic.Group, kv *versioned.KV, priv *cyclic.Int) (*Store, error) {
	//generate public key
	pub := diffieHellman.GeneratePublicKey(priv, grp)

	fingerprints := newFingerprints()
	s := &Store{
		managers:     make(map[id.ID]*Manager),
		fingerprints: fingerprints,

		dhPrivateKey: priv,
		dhPublicKey:  pub,
		grp:          grp,

		context: context{
			fa:  &fingerprints,
			grp: grp,
			kv:  kv,
		},
	}

	err := utility.StoreCyclicKey(kv, pub, pubKeyKey)
	if err != nil {
		return nil,
			errors.WithMessage(err,
				"Failed to store e2e DH public key")
	}

	err = utility.StoreCyclicKey(kv, priv, privKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to store e2e DH private key")
	}

	err = utility.StoreGroup(kv, grp, grpKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to store e2e group")
	}

	return s, s.save()
}

func LoadStore(kv *versioned.KV) (*Store, error) {
	fingerprints := newFingerprints()
	s := &Store{
		managers:     make(map[id.ID]*Manager),
		fingerprints: fingerprints,

		context: context{
			fa: &fingerprints,
			kv: kv,
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

func (s *Store) AddPartner(partnerID *id.ID, myPrivKey *cyclic.Int,
	partnerPubKey *cyclic.Int, sendParams, receiveParams SessionParams) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	m, err := newManager(&s.context, partnerID, myPrivKey, partnerPubKey, sendParams, receiveParams)

	if err != nil {
		return err
	}

	s.managers[*partnerID] = m

	return s.save()
}

func (s *Store) GetPartner(partnerID *id.ID) (*Manager, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	m, ok := s.managers[*partnerID]

	if !ok {
		return nil, errors.New("Cound not find manager for partner")
	}

	return m, nil
}

//Pops a key for use based upon its fingerprint
func (s *Store) PopKey(f format.Fingerprint) (*Key, error) {
	return s.fingerprints.Pop(f)
}

//Returns the diffie hellman private key
func (s *Store) GetDHPrivateKey() *cyclic.Int {
	return s.dhPrivateKey
}

//Returns the diffie hellman public key
func (s *Store) GetDHPublicKey() *cyclic.Int {
	return s.dhPublicKey
}

//Returns the cyclic group used for cmix
func (s *Store) GetGroup() *cyclic.Group {
	return s.grp
}

//ekv functions
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
		// load the manager. Manager handles adding the fingerprints via the
		// context object
		manager, err := loadManager(&s.context, &partnerID)
		if err != nil {
			jww.FATAL.Panicf("Failed to load manager for partner %s: %s", &partnerID, err.Error())
		}

		s.managers[partnerID] = manager
	}

	s.dhPrivateKey, err = utility.LoadCyclicKey(s.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH private key")
	}

	s.dhPublicKey, err = utility.LoadCyclicKey(s.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH public key")
	}

	s.grp, err = utility.LoadGroup(s.kv, grpKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e group")
	}

	return nil
}

type fingerprints struct {
	toKey map[format.Fingerprint]*Key
	mux   sync.RWMutex
}

func newFingerprints() fingerprints {
	return fingerprints{
		toKey: make(map[format.Fingerprint]*Key),
	}
}

//fingerprint adhere to the fingerprintAccess interface
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

func (f *fingerprints) Pop(fingerprint format.Fingerprint) (*Key, error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	key, ok := f.toKey[fingerprint]

	if !ok {
		return nil, errors.New("Key could not be found")
	}

	delete(f.toKey, fingerprint)

	err := key.denoteUse()

	if err != nil {
		return nil, err
	}

	key.fp = &fingerprint

	return key, nil
}
