///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	version = 0
	prefix  = "udStorePrefix"
	key     = "udStoreKey"
)

// Store is the storage object for the higher level ud.Manager object.
// This storage implementation is written for client side.
type Store struct {
	registeredFacts map[fact.Fact]struct{}
	kv              *versioned.KV
	mux             sync.RWMutex
}

// NewStore creates a new, empty Store object.
func NewStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(prefix)

	s := &Store{
		registeredFacts: make(map[fact.Fact]struct{}),
		kv:              kv,
	}

	return s, s.save()
}

// LoadStore loads the Store object from the provided versioned.KV.
func LoadStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(prefix)

	obj, err := kv.Get(key, version)
	if err != nil {
		return nil, err
	}

	s := &Store{
		registeredFacts: make(map[fact.Fact]struct{}),
		kv:              kv,
	}

	err = s.unmarshal(obj.Data)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal store")
	}

	return s, nil

}

// StoreFact adds a registered fact to the Store object. Both the Store's
// registeredFacts map in memory and the kv's storage are updated.
func (s *Store) StoreFact(f fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.registeredFacts[f] = struct{}{}
	return s.save()
}

// GetStringifiedFacts returns a list of stringified facts from the Store's
// registeredFacts map.
func (s *Store) GetStringifiedFacts() []string {
	s.mux.RLock()
	defer s.mux.RUnlock()

	return s.serializeFacts()
}

// GetFacts returns a list of fact.Fact objects that exist within the
// Store's registeredFacts map.
func (s *Store) GetFacts() []fact.Fact {
	s.mux.RLock()
	defer s.mux.RUnlock()

	facts := make([]fact.Fact, 0, len(s.registeredFacts))
	for f := range s.registeredFacts {
		facts = append(facts, f)
	}

	return facts
}

// save serializes the state within Store into byte data and stores
// that data into storage via the EKV.
func (s *Store) save() error {
	now := netTime.Now()

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   version,
		Timestamp: now,
		Data:      data,
	}

	return s.kv.Set(key, version, &obj)
}

// marshal serializes the state within Store into byte data.
func (s *Store) marshal() ([]byte, error) {
	return json.Marshal(s.serializeFacts())
}

// unmarshal deserializes byte data into Store's state.
func (s *Store) unmarshal(data []byte) error {
	var fStrings []string

	err := json.Unmarshal(data, &fStrings)
	if err != nil {
		return err
	}

	return s.deserializeFacts(fStrings)
}

// serializeFacts is a helper function which serializes Store's registeredFacts
// map into a list of strings. Each string in the list represents
// a fact.Fact that has been Stringified.
func (s *Store) serializeFacts() []string {
	fStrings := make([]string, 0, len(s.registeredFacts))
	for f := range s.registeredFacts {
		fStrings = append(fStrings, f.Stringify())
	}

	return fStrings
}

// deserializeFacts takes a list of stringified fact.Fact's and un-stringifies
// them into fact.Fact objects. These objects are them placed into Store's
// registeredFacts map.
func (s *Store) deserializeFacts(fStrings []string) error {
	for _, fStr := range fStrings {
		f, err := fact.UnstringifyFact(fStr)
		if err != nil {
			return errors.WithMessage(err, "Failed to add due to "+
				"malformed fact")
		}

		s.registeredFacts[f] = struct{}{}
	}
	return nil
}
