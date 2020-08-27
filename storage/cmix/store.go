package cmix

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const currentStoreVersion = 0
const storeKey = "cmixKeyStore"

type Store struct {
	nodes map[id.ID]*key

	kv *versioned.KV

	mux sync.RWMutex
}

// returns a new cmix storage object
func NewStore(kv *versioned.KV) *Store {
	return &Store{
		nodes: make(map[id.ID]*key),
		kv:    kv,
	}
}

// loads the cmix storage object
func LoadStore(kv *versioned.KV) (*Store, error) {
	s := NewStore(kv)

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
func (s *Store) Add(nid *id.ID, k *cyclic.Int) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	nodekey := &key{k: k}
	err := nodekey.save(s.kv, nid)
	if err != nil {
		return err
	}

	s.nodes[*nid] = nodekey
	return s.save()
}

// removes the key from the cmix storage object. Saves an updates node list to
//
func (s *Store) Remove(nid *id.ID, k *cyclic.Int) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	nodekey, ok := s.nodes[*nid]
	if !ok {
		return errors.New("Cannot remove, no key with given ID found")
	}

	err := nodekey.delete(s.kv, nid)
	if err != nil {
		return err
	}

	delete(s.nodes, *nid)

	return nil
}

//Returns a RoundKeys for the topology and a list of nodes it did not have a key for
func (s *Store) GetRoundKeys(topology *connect.Circuit) (RoundKeys, []*id.ID) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	var missingNodes []*id.ID

	rk := RoundKeys(make([]*cyclic.Int, topology.Len()))

	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		k, ok := s.nodes[*nid]
		if !ok {
			missingNodes = append(missingNodes, nid)
		} else {
			rk[i] = k.k
		}
	}

	return rk, missingNodes
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
	for nid, _ := range s.nodes {
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
		k, err := loadKey(s.s, &nid)
		if err != nil {
			return errors.WithMessagef(err, "could not load node key for %s", &nid)
		}
		s.nodes[nid] = k
	}

	return nil
}
