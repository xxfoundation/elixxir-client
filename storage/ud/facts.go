///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/fact"
	"sync"
)

const (
	factTypeExistsErr               = "Fact %v cannot be added as fact type %s has already been stored. Cancelling backup operation!"
	backupMissingInvalidFactTypeErr = "BackUpMissingFacts expects input in the order (email, phone). " +
		"%s (%s) is non-empty but not an email. Cancelling backup operation"
	backupMissingAllZeroesFactErr = "Cannot backup missing facts: Both email and phone facts are empty!"
	factNotInStoreErr             = "Fact %v does not exist in store"
	statefulStoreErr              = "cannot overwrite ud store with existing data"
)

// Store is the storage object for the higher level ud.Manager object.
// This storage implementation is written for client side.
type Store struct {
	// confirmedFacts contains facts that have been confirmed
	confirmedFacts map[fact.Fact]struct{}
	// Stores facts that have been added by UDB but unconfirmed facts.
	// Maps confirmID to fact
	unconfirmedFacts map[string]fact.Fact
	kv               *versioned.KV
	mux              sync.RWMutex
}

// NewStore creates a new Store object. If we are initializing from a backup,
// the backupFacts fact.FactList will be non-nil and initialize the state
// with the backed up data.
func NewStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(prefix)
	s := &Store{
		confirmedFacts:   make(map[fact.Fact]struct{}, 0),
		unconfirmedFacts: make(map[string]fact.Fact, 0),
		kv:               kv,
	}

	return s, s.save()
}

// RestoreFromBackUp initializes the confirmedFacts map
// with the backed up fact data. This will error if
// the store is already stateful.
func (s *Store) RestoreFromBackUp(backupData fact.FactList) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if len(s.confirmedFacts) != 0 || len(s.unconfirmedFacts) != 0 {
		return errors.New(statefulStoreErr)
	}

	for _, f := range backupData {
		s.confirmedFacts[f] = struct{}{}
	}

	return s.save()
}

// StoreUnconfirmedFact stores a fact that has been added to UD but has not been
// confirmed by the user. It is keyed on the confirmation ID given by UD.
func (s *Store) StoreUnconfirmedFact(confirmationId string, f fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.unconfirmedFacts[confirmationId] = f
	return s.saveUnconfirmedFacts()
}

// ConfirmFact will delete the fact from the unconfirmed store and
// add it to the confirmed fact store. The Store will then be saved
func (s *Store) ConfirmFact(confirmationId string) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	f, exists := s.unconfirmedFacts[confirmationId]
	if !exists {
		return errors.New(fmt.Sprintf("No fact exists in store "+
			"with confirmation ID %q", confirmationId))
	}

	delete(s.unconfirmedFacts, confirmationId)
	s.confirmedFacts[f] = struct{}{}
	return s.save()
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
// We handle storage of newly registered internally using Store.ConfirmFact.
// ************************************************************************
// Any other fact.FactType is not accepted and returns an error and nothing is backed up.
// If you attempt to back up a fact type that has already been backed up,
// an error will be returned and nothing will be backed up.
// Otherwise, it adds the fact and returns whether the Store saved successfully.
func (s *Store) BackUpMissingFacts(email, phone fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if isFactZero(email) && isFactZero(phone) {
		return errors.New(backupMissingAllZeroesFactErr)
	}

	modifiedEmail, modifiedPhone := false, false

	// Handle email if it is not zero (empty string)
	if !isFactZero(email) {
		// check if fact is expected type
		if email.T != fact.Email {
			return errors.New(fmt.Sprintf(backupMissingInvalidFactTypeErr, fact.Email, email.Fact))
		}

		// Check if fact type is already in map. See docstring NOTE for explanation
		if isFactTypeInMap(fact.Email, s.confirmedFacts) {
			// If an email exists in memory, return an error
			return errors.Errorf(factTypeExistsErr, email, fact.Email)
		} else {
			modifiedEmail = true
		}
	}

	if !isFactZero(phone) {
		// check if fact is expected type
		if phone.T != fact.Phone {
			return errors.New(fmt.Sprintf(backupMissingInvalidFactTypeErr, fact.Phone, phone.Fact))
		}

		// Check if fact type is already in map. See docstring NOTE for explanation
		if isFactTypeInMap(fact.Phone, s.confirmedFacts) {
			// If a phone exists in memory, return an error
			return errors.Errorf(factTypeExistsErr, phone, fact.Phone)
		} else {
			modifiedPhone = true
		}
	}

	if modifiedPhone || modifiedEmail {
		if modifiedEmail {
			s.confirmedFacts[email] = struct{}{}
		}

		if modifiedPhone {
			s.confirmedFacts[phone] = struct{}{}
		}

		return s.saveConfirmedFacts()
	}

	return nil

}

// DeleteFact is our internal use function which will delete the registered fact
// from memory and storage. An error is returned if the fact does not exist in
// memory.
func (s *Store) DeleteFact(f fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, exists := s.confirmedFacts[f]; !exists {
		return errors.Errorf(factNotInStoreErr, f)
	}

	delete(s.confirmedFacts, f)
	return s.saveConfirmedFacts()
}

// GetStringifiedFacts returns a list of stringified facts from the Store's
// confirmedFacts map.
func (s *Store) GetStringifiedFacts() []string {
	s.mux.RLock()
	defer s.mux.RUnlock()

	return s.serializeConfirmedFacts()
}

// GetFacts returns a list of fact.Fact objects that exist within the
// Store's confirmedFacts map.
func (s *Store) GetFacts() []fact.Fact {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// Flatten the facts into a slice
	facts := make([]fact.Fact, 0, len(s.confirmedFacts))
	for f := range s.confirmedFacts {
		facts = append(facts, f)
	}

	return facts
}

// serializeConfirmedFacts is a helper function which serializes Store's confirmedFacts
// map into a list of strings. Each string in the list represents
// a fact.Fact that has been Stringified.
func (s *Store) serializeConfirmedFacts() []string {
	fStrings := make([]string, 0, len(s.confirmedFacts))
	for f := range s.confirmedFacts {
		fStrings = append(fStrings, f.Stringify())
	}

	return fStrings
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
