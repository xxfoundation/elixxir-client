////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/fact"
)

// Error constants to return up the stack.
const (
	factTypeExistsErr               = "Fact %v cannot be added as fact type %s has already been stored. Cancelling backup operation!"
	backupMissingInvalidFactTypeErr = "BackUpMissingFacts expects input in the order (email, phone). " +
		"%s (%s) is non-empty but not an email. Cancelling backup operation"
	factNotInStoreErr = "Fact %v does not exist in store"
	statefulStoreErr  = "cannot overwrite ud store with existing data"
)

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
		if !isFactZero(f) {
			s.confirmedFacts[f] = struct{}{}
		}
	}

	return s.save()
}

// StoreUsername forces the storage of a username fact.Fact into the
// Store's confirmedFacts map. The passed in fact.Fact must be of
// type fact.Username or this will not store the username.
func (s *Store) StoreUsername(f fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if f.T != fact.Username {
		return errors.Errorf("Fact (%s) is not of type username", f.Stringify())
	}

	s.confirmedFacts[f] = struct{}{}

	return s.saveUnconfirmedFacts()
}

// GetUsername retrieves the username from the Store object.
// If it is not directly in the Store's username field, it is
// searched for in the map.
func (s *Store) GetUsername() (string, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// todo: refactor this in the future so that
	//  it's an O(1) lookup (place this object in another map
	//  or have it's own field)
	for f := range s.confirmedFacts {
		if f.T == fact.Username {
			return f.Fact, nil
		}
	}

	return "", errors.New("Could not find username in store")

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
func (s *Store) BackUpMissingFacts(username, email, phone fact.Fact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	modified := false

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
			s.confirmedFacts[email] = struct{}{}
			modified = true
		}
	}

	// Handle phone if it is not an empty string
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
			s.confirmedFacts[phone] = struct{}{}
			modified = true
		}
	}

	if !isFactZero(username) {
		// Check if fact type is already in map. You should not be able to
		// overwrite your username.
		if isFactTypeInMap(fact.Username, s.confirmedFacts) {
			// If a username exists in memory, return an error
			return errors.Errorf(factTypeExistsErr, username, fact.Username)
		} else {
			s.confirmedFacts[username] = struct{}{}
			modified = true
		}
	}

	if modified {
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
