////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const prefix = "cmix"
const currentStoreVersion = 0
const storeKey = "KeyStore"
const pubKeyKey = "DhPubKey"
const privKeyKey = "DhPrivKey"
const grpKey = "GroupKey"

type Store struct {
	nodes        map[id.ID]*key
	dhPrivateKey *cyclic.Int
	dhPublicKey  *cyclic.Int

	grp *cyclic.Group

	kv *versioned.KV

	mux sync.RWMutex
}

// returns a new cmix storage object
func NewStore(grp *cyclic.Group, kv *versioned.KV, priv *cyclic.Int) (*Store, error) {
	//generate public key
	pub := diffieHellman.GeneratePublicKey(priv, grp)
	kv = kv.Prefix(prefix)

	s := &Store{
		nodes:        make(map[id.ID]*key),
		dhPrivateKey: priv,
		dhPublicKey:  pub,
		grp:          grp,
		kv:           kv,
	}

	err := utility.StoreCyclicKey(kv, pub, pubKeyKey)
	if err != nil {
		return nil,
			errors.WithMessage(err,
				"Failed to store cmix DH public key")
	}

	err = utility.StoreCyclicKey(kv, priv, privKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to store cmix DH private key")
	}

	err = utility.StoreGroup(kv, grp, grpKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to store cmix group")
	}

	return s, s.save()
}

// loads the cmix storage object
func LoadStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(prefix)
	s := &Store{
		nodes: make(map[id.ID]*key),
		kv:    kv,
	}

	obj, err := kv.Get(storeKey)
	if err != nil {
		return nil, err
	}

	err = s.unmarshal(obj.Data)

	if err != nil {
		return nil, err
	}

	return s, nil
}

// adds the key for a round to the cmix storage object. Saves the updated list
// of nodes and the key to disk
func (s *Store) Add(nid *id.ID, k *cyclic.Int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	nodekey := newKey(s.kv, k, nid)

	s.nodes[*nid] = nodekey
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save nodeKey list for %s: %s", nid, err)
	}
}

// Done a Node key from the nodes map and save
func (s *Store) Remove(nid *id.ID) {
	s.mux.Lock()
	defer s.mux.Unlock()

	nodekey, ok := s.nodes[*nid]
	if !ok {
		return
	}

	nodekey.delete(s.kv, nid)

	delete(s.nodes, *nid)

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed    make nodeKey for %s: %s", nid, err)
	}
}

//Returns a RoundKeys for the topology and a list of nodes it did not have a key for
// If there are missing keys, returns nil RoundKeys
func (s *Store) GetRoundKeys(topology *connect.Circuit) (*RoundKeys, []*id.ID) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	var missingNodes []*id.ID

	keys := make([]*key, topology.Len())

	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		k, ok := s.nodes[*nid]
		if !ok {
			missingNodes = append(missingNodes, nid)
		} else {
			keys[i] = k
		}
	}

	// Handle missing keys case
	if len(missingNodes) > 0 {
		return nil, missingNodes
	}

	rk := &RoundKeys{
		keys: keys,
		g:    s.grp,
	}

	return rk, missingNodes
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

func (s *Store) IsRegistered(nid *id.ID) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	_, ok := s.nodes[*nid]
	return ok
}

// stores the cmix store
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

// builds a byte representation of the store
func (s *Store) marshal() ([]byte, error) {
	nodes := make([]id.ID, len(s.nodes))

	index := 0
	for nid := range s.nodes {
		nodes[index] = nid
	}

	return json.Marshal(&nodes)
}

// restores the data for a store from the byte representation of the store
func (s *Store) unmarshal(b []byte) error {
	var nodes []id.ID

	err := json.Unmarshal(b, &nodes)
	if err != nil {
		return err
	}

	for _, nid := range nodes {
		k, err := loadKey(s.kv, &nid)
		if err != nil {
			return errors.WithMessagef(err, "could not load node key for %s", &nid)
		}
		s.nodes[nid] = k
	}

	s.dhPrivateKey, err = utility.LoadCyclicKey(s.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load cmix DH private key")
	}

	s.dhPublicKey, err = utility.LoadCyclicKey(s.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load cmix DH public key")
	}

	s.grp, err = utility.LoadGroup(s.kv, grpKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load cmix group")
	}

	return nil
}
