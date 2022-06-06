///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const prefix = "cmix"
const currentStoreVersion = 0
const (
	storeKey = "KeyStore"
	grpKey   = "GroupKey"
)

type Store struct {
	nodes      map[id.ID]*key
	validUntil uint64
	keyId      []byte
	grp        *cyclic.Group
	kv         *versioned.KV
	mux        sync.RWMutex
}

// NewStore returns a new cMix storage object.
func NewStore(grp *cyclic.Group, kv *versioned.KV) (*Store, error) {
	// Generate public key
	kv = kv.Prefix(prefix)

	s := &Store{
		nodes: make(map[id.ID]*key),
		grp:   grp,
		kv:    kv,
	}

	err := utility.StoreGroup(kv, grp, grpKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to store cMix group")
	}

	return s, s.save()
}

// LoadStore loads the cMix storage object.
func LoadStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(prefix)
	s := &Store{
		nodes: make(map[id.ID]*key),
		kv:    kv,
	}

	obj, err := kv.Get(storeKey, currentKeyVersion)
	if err != nil {
		return nil, err
	}

	err = s.unmarshal(obj.Data)

	if err != nil {
		return nil, err
	}

	return s, nil
}

// Add adds the key for a round to the cMix storage object. Saves the updated
// list of nodes and the key to disk.
func (s *Store) Add(nid *id.ID, k *cyclic.Int,
	validUntil uint64, keyId []byte) {
	s.mux.Lock()
	defer s.mux.Unlock()

	nodeKey := newKey(s.kv, k, nid, validUntil, keyId)

	s.nodes[*nid] = nodeKey
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save nodeKey list for %s: %+v", nid, err)
	}
}

// Returns if the store has the node
func (s *Store) Has(nid *id.ID) bool {
	s.mux.RLock()
	_, exists := s.nodes[*nid]
	s.mux.RUnlock()
	return exists
}

// Remove removes a node key from the nodes map and saves.
func (s *Store) Remove(nid *id.ID) {
	s.mux.Lock()
	defer s.mux.Unlock()

	nodeKey, ok := s.nodes[*nid]
	if !ok {
		return
	}

	nodeKey.delete(s.kv, nid)

	delete(s.nodes, *nid)

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to make nodeKey for %s: %+v", nid, err)
	}
}

// GetRoundKeys returns a RoundKeys for the topology and a list of nodes it did
// not have a key for. If there are missing keys, then returns nil RoundKeys.
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

// GetGroup returns the cyclic group used for cMix.
func (s *Store) GetGroup() *cyclic.Group {
	return s.grp
}

func (s *Store) IsRegistered(nid *id.ID) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	_, ok := s.nodes[*nid]
	return ok
}

// Count returns the number of registered nodes.
func (s *Store) Count() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return len(s.nodes)
}

// save stores the cMix store.
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

	return s.kv.Set(storeKey, currentKeyVersion, &obj)
}

// marshal builds a byte representation of the Store.
func (s *Store) marshal() ([]byte, error) {
	nodes := make([]id.ID, len(s.nodes))

	index := 0
	for nid := range s.nodes {
		nodes[index] = nid
		index++
	}

	return json.Marshal(&nodes)
}

// unmarshal restores the data for a Store from the byte representation of the
// Store
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

	s.grp, err = utility.LoadGroup(s.kv, grpKey)
	if err != nil {
		return errors.WithMessage(err, "Failed to load cMix group")
	}

	return nil
}
