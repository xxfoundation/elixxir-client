package edge

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	fingerprint2 "gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// This stores Preimages which can be used with the identity fingerprint system.

const (
	edgeStorePrefix  = "edgeStore"
	edgeStoreKey     = "edgeStoreKey"
	edgeStoreVersion = 0
)

type ListUpdateCallBack func(identity *id.ID, deleted bool)

type Store struct {
	kv        *versioned.KV
	edge      map[id.ID]Preimages
	callbacks map[id.ID][]ListUpdateCallBack
	mux       sync.RWMutex
}

// NewStore creates a new edge store object and inserts the default Preimages
// for the base identity.
func NewStore(kv *versioned.KV, baseIdentity *id.ID) (*Store, error) {
	kv = kv.Prefix(edgeStorePrefix)

	s := &Store{
		kv:        kv,
		edge:      make(map[id.ID]Preimages),
		callbacks: make(map[id.ID][]ListUpdateCallBack),
	}

	defaultPreimages := newPreimages(baseIdentity)
	err := defaultPreimages.save(kv, baseIdentity)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create preimage store, "+
			"failed to create default Preimages")
	}

	s.edge[*baseIdentity] = defaultPreimages

	return s, s.save()
}

// Add adds the Preimage to the list of the given identity and calls any
// associated callbacks.
func (s *Store) Add(preimage Preimage, identity *id.ID) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Get the list to update, create if needed
	preimages, exists := s.edge[*identity]
	if !exists {
		preimages = newPreimages(identity)
	}

	// Add to the list
	if !preimages.add(preimage) {
		jww.ERROR.Printf("Preimage already exists for id %s: %v",
			identity, preimage)
		return
	}

	// Store the updated list
	if err := preimages.save(s.kv, identity); err != nil {
		jww.FATAL.Panicf("Failed to store preimages list after adding "+
			"preimage %v to identity %s: %+v", preimage.Data, identity, err)
	}

	// Update the map
	s.edge[*identity] = preimages
	if !exists {
		err := s.save()
		if err != nil {
			jww.FATAL.Panicf("Failed to store edge store after adding "+
				"preimage %v to identity %s: %+v", preimage.Data, identity, err)
		}
	}

	// Call any callbacks to notify
	for _, cb := range s.callbacks[*identity] {
		go cb(identity, false)
	}

	return
}

// Remove deletes the preimage for the given identity and triggers the
// associated callback. If the given preimage is the last in the Preimages list,
// then the entire list is removed and the associated callback will be triggered
// with the boolean indicating the list was deleted.
func (s *Store) Remove(preimage Preimage, identity *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	preimages, exists := s.edge[*identity]
	if !exists {
		return errors.Errorf("cannot delete preimage %v from identity %s; "+
			"identity cannot be found", preimage.Data, identity)
	}

	preimages.remove(preimage.Data)

	if len(preimages) == 0 {
		delete(s.edge, *identity)
		if err := s.save(); err != nil {
			jww.FATAL.Panicf("Failed to store edge store after removing "+
				"preimage %v to identity %s: %+v", preimage.Data, identity, err)
		}

		if err := preimages.delete(s.kv, identity); err != nil {
			jww.FATAL.Panicf("Failed to delete preimage list store after "+
				"removing preimage %v to identity %s: %+v", preimage.Data,
				identity, err)
		}

		// Call any callbacks to notify
		for i := range s.callbacks[*identity] {
			cb := s.callbacks[*identity][i]
			go cb(identity, true)
		}

		return nil
	}

	if err := preimages.save(s.kv, identity); err != nil {
		jww.FATAL.Panicf("Failed to store preimage list store after removing "+
			"preimage %v to identity %s: %+v", preimage.Data, identity, err)
	}

	s.edge[*identity] = preimages

	// Call any callbacks to notify
	for i := range s.callbacks[*identity] {
		cb := s.callbacks[*identity][i]
		go cb(identity, false)
	}

	return nil
}

// Get returns the Preimages list for the given identity.
func (s *Store) Get(identity *id.ID) ([]Preimage, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	preimages, exists := s.edge[*identity]
	if !exists {
		return nil, false
	}

	preiamgesSlice := make([]Preimage, 0, len(preimages))

	for _, preimage := range preimages {
		preiamgesSlice = append(preiamgesSlice, preimage)
	}
	return preiamgesSlice, exists
}

// Check looks checks if the identity fingerprint matches for any of
// the stored preimages. It returns the preimage it hit with if it
// finds one.
func (s *Store) Check(identity *id.ID, identityFP []byte, messageContents []byte) (bool, bool, Preimage) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	preimages, exists := s.edge[*identity]
	if !exists {
		return false, false, Preimage{}
	}

	for _, preimage := range preimages {
		if fingerprint2.CheckIdentityFP(identityFP, messageContents, preimage.Data) {
			return true, true, preimage
		}
	}

	return true, false, Preimage{}
}

// AddUpdateCallback adds the callback to be called for changes to the identity.
func (s *Store) AddUpdateCallback(identity *id.ID, luCB ListUpdateCallBack) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.callbacks[*identity] = append(s.callbacks[*identity], luCB)
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

func LoadStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(edgeStorePrefix)

	// Load the list of identities with preimage lists
	obj, err := kv.Get(edgeStoreKey, preimageStoreVersion)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load edge store")
	}

	identities := make([]id.ID, 0)

	err = json.Unmarshal(obj.Data, &identities)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal edge store")
	}

	s := &Store{
		kv:        kv,
		edge:      make(map[id.ID]Preimages),
		callbacks: make(map[id.ID][]ListUpdateCallBack),
	}

	// Load the preimage lists for all identities
	for i := range identities {
		eid := &identities[i]

		preimages, err := loadPreimages(kv, eid)
		if err != nil {
			return nil, err
		}

		s.edge[*eid] = preimages
	}

	return s, nil
}

func (s *Store) save() error {
	identities := make([]id.ID, 0, len(s.edge))

	for eid := range s.edge {
		identities = append(identities, eid)
	}

	// JSON marshal
	data, err := json.Marshal(&identities)
	if err != nil {
		return errors.WithMessagef(err, "Failed to marshal edge list for "+
			"storage")
	}

	// Construct versioning object
	obj := versioned.Object{
		Version:   edgeStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	// Save to storage
	err = s.kv.Set(edgeStoreKey, preimageStoreVersion, &obj)
	if err != nil {
		return errors.WithMessagef(err, "Failed to store edge list")
	}

	return nil
}
