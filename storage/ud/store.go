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

// Error constants
const (
	factTypeExistsErr          = "Fact %v cannot be added as fact type %s has already been stored. Cancelling backup operation!"
	unrecognizedFactErr        = "Fact %v is not of expected type (%s). Cancelling backup operation!"
	unrecognizedFactInStoreErr = "Fact %s with type %s loaded from memory is invalid"
)

// Store is the storage object for the higher level ud.Manager object.
// This storage implementation is written for client side.
type Store struct {
	// registeredFacts contains only 2 registered facts: an email and a phone number.
	// These are definitely indexed, as defined above.
	registeredFacts map[fact.Fact]struct{}
	kv              *versioned.KV
	mux             sync.RWMutex
}

// NewStore creates a new, empty Store object.
func NewStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(prefix)

	s := &Store{
		registeredFacts: make(map[fact.Fact]struct{}, 0),
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
		registeredFacts: make(map[fact.Fact]struct{}, 0),
		kv:              kv,
	}

	err = s.unmarshal(obj.Data)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal store")
	}

	return s, nil

}

// BackUpMissingFacts adds a registered fact to the Store object. It can take in both an
// email and a phone number. One or the other may be an empty string, however both is considered
// an error. It checks for each whether that fact type already exists in the structure. If a fact
// type already exists, an error is returned.
// ************************************************************************
// NOTE: This is done since BackUpMissingFacts is exposed to the
// bindings layer. This prevents front end from using this as the method
// to store facts on their end, which is not its intended use case. It's intended use
// case is to store already registered facts, prior to the creation of this function.
// We handle storage of newly registered internally using Store.StoreFact.
// ************************************************************************
// Any other fact.FactType is not accepted and returns an error and nothing is backed up.
// If you attempt to back up a fact type that has already been backed up,
// an error will be returned and nothing will be backed up.
// Otherwise, it adds the fact and returns whether the Store saved successfully.
func (s *Store) BackUpMissingFacts(email, phone fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if isFactZero(email) && isFactZero(phone) {
		return errors.New("Cannot backup missing facts: Both email and phone facts are empty!")
	}

	modifiedEmail, modifiedPhone := false, false

	// Handle email if it is not zero (empty string)
	if !isFactZero(email) {
		// check if fact is expected type
		if email.T != fact.Email {
			return errors.New(fmt.Sprintf("BackUpMissingFacts expects input in the order (email, phone). "+
				"Email (%s) is non-empty but not an email. Cancelling backup operation", email.Fact))
		}

		// Check if fact type is already in map. See docstring NOTE for explanation
		if isFactTypeInMap(fact.Email, s.registeredFacts) {
			// If an email exists in memory, return an error
			return errors.Errorf(factTypeExistsErr, email, fact.Email)
		} else {
			modifiedEmail = true
		}
	}

	if !isFactZero(phone) {
		// check if fact is expected type
		if phone.T != fact.Phone {
			return errors.New(fmt.Sprintf("BackUpMissingFacts expects input in the order (email, phone). "+
				"Phone (%s) is non-empty but not an phone. Cancelling backup operation", phone.Fact))
		}

		// Check if fact type is already in map. See docstring NOTE for explanation
		if isFactTypeInMap(fact.Phone, s.registeredFacts) {
			// If a phone exists in memory, return an error
			return errors.Errorf(factTypeExistsErr, phone, fact.Phone)
		} else {
			modifiedPhone = true
		}
	}

	if modifiedPhone || modifiedEmail {
		if modifiedEmail {
			s.registeredFacts[email] = struct{}{}
		}

		if modifiedPhone {
			s.registeredFacts[phone] = struct{}{}
		}

		return s.save()
	}

	return nil

}

// StoreFact is our internal use function which will add the fact to
// memory and save to storage. THIS IS FOR REGISTERED FACTS ONLY.
func (s *Store) StoreFact(f fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.registeredFacts[f] = struct{}{}
	return s.save()
}

// DeleteFact is our internal use function which will delete the fact to
// memory and save to storage. An error is returned if the fact does not exist in memory.
func (s *Store) DeleteFact(f fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, exists := s.registeredFacts[f]; !exists {
		return errors.Errorf("Fact %v does not exist in store", f)
	}

	delete(s.registeredFacts, f)
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

	// Flatten the facts into a slice
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
	facts := make(map[fact.Fact]struct{}, 0)
	for _, fStr := range fStrings {
		f, err := fact.UnstringifyFact(fStr)
		if err != nil {
			return errors.WithMessage(err, "Failed to load due to "+
				"malformed fact")
		}

		facts[f] = struct{}{}
	}

	s.registeredFacts = facts

	return nil
}

// fixme: consider this being a method on the fact.Fact object?
// isFactZero tests whether a fact has been uninitialized.
func isFactZero(f fact.Fact) bool {
	return f.T == fact.Username && f.Fact == ""
}

// isFactTypeInMap is a helper function which determines whether a fact type exists within
// the data structure.
func isFactTypeInMap(factType fact.FactType, facts map[fact.Fact]struct{}) bool {
	for f := range facts {
		if f.T == factType {
			return true
		}
	}

	return false
}
