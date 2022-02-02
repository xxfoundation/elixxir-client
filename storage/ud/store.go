///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

// Storage constants
const (
	version = 0
	prefix  = "udStorePrefix"
	key     = "udStoreKey"
)

// Fact indexing constants
const (
	emailIndex = 0
	phoneIndex = 1
)

// Error constants
const (
	factTypeExistsErr          = "Fact %s cannot be added as fact type %s has already been stored"
	unrecognizedFactErr        = "Fact %s with type %s cannot be added to store"
	unrecognizedFactInStoreErr = "Fact %s with type %s loaded from memory is invalid"
)

// Store is the storage object for the higher level ud.Manager object.
// This storage implementation is written for client side.
type Store struct {
	// registeredFacts contains only 2 registered facts: an email and a phone number.
	// These are definitely indexed, as defined above.
	registeredFacts [2]fact.Fact
	kv              *versioned.KV
	mux             sync.RWMutex
}

// NewStore creates a new, empty Store object.
func NewStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(prefix)

	s := &Store{
		registeredFacts: [2]fact.Fact{},
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
		registeredFacts: [2]fact.Fact{},
		kv:              kv,
	}

	err = s.unmarshal(obj.Data)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal store")
	}

	return s, nil

}

// StoreFact adds a registered fact to the Store object.
// It checks for the fact type, and accepts only fact.Email and fact.Phone.
// Any other fact.FactType is not accepted and returns an error. If trying to add a
// fact.Fact with a fact.FactType that has already been added, an error will be returned.
// Otherwise, it adds the fact and returns whether the Store saved successfully.
func (s *Store) StoreFact(f fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	switch f.T { // Check fact type
	case fact.Email:
		if s.registeredFacts[emailIndex].Fact == "" {
			s.registeredFacts[emailIndex] = f
			return s.save()
		}
	case fact.Phone:
		if s.registeredFacts[phoneIndex].Fact == "" {
			s.registeredFacts[phoneIndex] = f
			return s.save()
		}

	default:
		return errors.New(fmt.Sprintf(unrecognizedFactErr, f.Fact, f.T))
	}

	return errors.New(fmt.Sprintf(factTypeExistsErr, f.Fact, f.T))
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

	return s.registeredFacts[:]
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
	for _, f := range s.registeredFacts {
		fStrings = append(fStrings, f.Stringify())
	}

	return fStrings
}

// deserializeFacts takes a list of stringified fact.Fact's and un-stringifies
// them into fact.Fact objects. These objects are them placed into Store's
// registeredFacts map.
func (s *Store) deserializeFacts(fStrings []string) error {
	for _, fStr := range fStrings {
		// Since the length of s.registeredFacts is predefined,
		// indices wil be initialized with zero values, which
		// are not valid facts.
		//Skip by this initial value if this is the case
		if len(fStr) < 2 {
			continue
		}

		f, err := fact.UnstringifyFact(fStr)
		if err != nil {
			return errors.WithMessage(err, "Failed to load due to "+
				"malformed fact")
		}

		switch f.T {
		case fact.Email:
			s.registeredFacts[emailIndex] = f
		case fact.Phone:
			s.registeredFacts[phoneIndex] = f
		default:
			return errors.New(fmt.Sprintf(unrecognizedFactInStoreErr, f.Fact, f.T))
		}

	}
	return nil
}
