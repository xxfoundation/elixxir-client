////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

// This file handles the storage operations on facts.

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

// Storage constants
const (
	version            = 0
	prefix             = "udStorePrefix"
	unconfirmedFactKey = "unconfirmedFactKey"
	confirmedFactKey   = "confirmedFactKey"
)

// Error constants
const (
	malformedFactErr       = "Failed to load due to malformed fact %s"
	loadConfirmedFactErr   = "Failed to load confirmed facts"
	loadUnconfirmedFactErr = "Failed to load unconfirmed facts"
	saveUnconfirmedFactErr = "Failed to save unconfirmed facts"
	saveConfirmedFactErr   = "Failed to save confirmed facts"
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

// newStore creates a new, empty Store object.
func newStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(prefix)

	s := &Store{
		confirmedFacts:   make(map[fact.Fact]struct{}),
		unconfirmedFacts: make(map[string]fact.Fact),
		kv:               kv,
	}

	return s, s.save()
}

/////////////////////////////////////////////////////////////////
// SAVE FUNCTIONS
/////////////////////////////////////////////////////////////////

// save serializes the state within Store into byte data and stores
// that data into storage via the EKV.
func (s *Store) save() error {

	err := s.saveUnconfirmedFacts()
	if err != nil {
		return errors.WithMessage(err, saveUnconfirmedFactErr)
	}

	err = s.saveConfirmedFacts()
	if err != nil {
		return errors.WithMessage(err, saveConfirmedFactErr)
	}

	return nil
}

// saveConfirmedFacts saves all the data within Store.confirmedFacts into storage.
func (s *Store) saveConfirmedFacts() error {

	data, err := s.marshalConfirmedFacts()
	if err != nil {
		return err
	}

	// Construct versioned object
	now := netTime.Now()
	obj := versioned.Object{
		Version:   version,
		Timestamp: now,
		Data:      data,
	}

	// Save to storage
	return s.kv.Set(confirmedFactKey, &obj)
}

// saveUnconfirmedFacts saves all data within Store.unconfirmedFacts into storage.
func (s *Store) saveUnconfirmedFacts() error {
	data, err := s.marshalUnconfirmedFacts()
	if err != nil {
		return err
	}

	// Construct versioned object
	now := netTime.Now()
	obj := versioned.Object{
		Version:   version,
		Timestamp: now,
		Data:      data,
	}

	// Save to storage
	return s.kv.Set(unconfirmedFactKey, &obj)

}

/////////////////////////////////////////////////////////////////
// LOAD FUNCTIONS
/////////////////////////////////////////////////////////////////

// NewOrLoadStore loads the Store object from the provided versioned.KV.
func NewOrLoadStore(kv *versioned.KV) (*Store, error) {

	s := &Store{
		kv: kv.Prefix(prefix),
	}
	if err := s.load(); err != nil {
		if !s.kv.Exists(err) {
			return newStore(kv)
		} else {
			return nil, err
		}
	}

	return s, nil

}

// load is a helper function which loads all data stored in storage from
// the save operation.
func (s *Store) load() error {

	err := s.loadUnconfirmedFacts()
	if err != nil {
		return errors.WithMessage(err, loadUnconfirmedFactErr)
	}

	err = s.loadConfirmedFacts()
	if err != nil {
		return errors.WithMessage(err, loadConfirmedFactErr)
	}

	return nil
}

// loadConfirmedFacts loads all confirmed facts from storage.
// It is the inverse operation of saveConfirmedFacts.
func (s *Store) loadConfirmedFacts() error {
	// Pull data from storage
	obj, err := s.kv.Get(confirmedFactKey, version)
	if err != nil {
		return err
	}

	// Place the map in memory
	s.confirmedFacts, err = s.unmarshalConfirmedFacts(obj.Data)
	if err != nil {
		return err
	}

	return nil
}

// loadUnconfirmedFacts loads all unconfirmed facts from storage.
// It is the inverse operation of saveUnconfirmedFacts.
func (s *Store) loadUnconfirmedFacts() error {
	// Pull data from storage
	obj, err := s.kv.Get(unconfirmedFactKey, version)
	if err != nil {
		return err
	}

	// Place the map in memory
	s.unconfirmedFacts, err = s.unmarshalUnconfirmedFacts(obj.Data)
	if err != nil {
		return err
	}

	return nil
}

/////////////////////////////////////////////////////////////////
// MARSHAL/UNMARSHAL FUNCTIONS
/////////////////////////////////////////////////////////////////

// unconfirmedFactDisk is an object used to store the data of an unconfirmed fact.
// It combines the key (confirmationId) and fact data (stringifiedFact) into a
// single JSON-able object.
type unconfirmedFactDisk struct {
	confirmationId  string
	stringifiedFact string
}

// marshalConfirmedFacts is a marshaller which serializes the data
//// in the confirmedFacts map into a JSON.
func (s *Store) marshalConfirmedFacts() ([]byte, error) {
	// Flatten confirmed facts to a list
	fStrings := s.serializeConfirmedFacts()

	// Marshal to JSON
	return json.Marshal(&fStrings)
}

// marshalUnconfirmedFacts is a marshaller which serializes the data
// in the unconfirmedFacts map into a JSON.
func (s *Store) marshalUnconfirmedFacts() ([]byte, error) {
	// Flatten unconfirmed facts to a list
	ufdList := make([]unconfirmedFactDisk, 0, len(s.unconfirmedFacts))
	for confirmationId, f := range s.unconfirmedFacts {
		ufd := unconfirmedFactDisk{
			confirmationId:  confirmationId,
			stringifiedFact: f.Stringify(),
		}
		ufdList = append(ufdList, ufd)
	}

	return json.Marshal(&ufdList)
}

// unmarshalConfirmedFacts is a function which deserializes the data from storage
// into a structure matching the confirmedFacts map.
func (s *Store) unmarshalConfirmedFacts(data []byte) (map[fact.Fact]struct{}, error) {
	// Unmarshal into list
	var fStrings []string
	err := json.Unmarshal(data, &fStrings)
	if err != nil {
		return nil, err
	}

	// Deserialize the list into a map
	confirmedFacts := make(map[fact.Fact]struct{}, 0)
	for i := range fStrings {
		fStr := fStrings[i]
		f, err := fact.UnstringifyFact(fStr)
		if err != nil {
			return confirmedFacts, errors.WithMessagef(err,
				malformedFactErr, string(data))
		}

		confirmedFacts[f] = struct{}{}
	}

	return confirmedFacts, nil
}

// unmarshalUnconfirmedFacts is a function which deserializes the data from storage
// into a structure matching the unconfirmedFacts map.
func (s *Store) unmarshalUnconfirmedFacts(data []byte) (map[string]fact.Fact, error) {
	// Unmarshal into list
	var ufdList []unconfirmedFactDisk
	err := json.Unmarshal(data, &ufdList)
	if err != nil {
		return nil, err
	}

	// Deserialize the list into a map
	unconfirmedFacts := make(map[string]fact.Fact, 0)
	for i := range ufdList {
		ufd := ufdList[i]
		f, err := fact.UnstringifyFact(ufd.stringifiedFact)
		if err != nil {
			return unconfirmedFacts, errors.WithMessagef(err,
				malformedFactErr, string(data))
		}

		unconfirmedFacts[ufd.confirmationId] = f
	}

	return unconfirmedFacts, nil
}
