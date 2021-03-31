///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"encoding/json"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"sync/atomic"
	"time"
)

const (
	unknownRoundsStorageKey     = "UnknownRoundsKey"
	unknownRoundsStorageVersion = 0
	unknownRoundPrefix          = "UnknownRoundPrefix"
	defaultMaxCheck = 3
)

// UnknownRoundsStore tracks data for unknown rounds
// Should adhere to UnknownRounds interface
type UnknownRoundsStore struct {
	// Maps an unknown round to how many times the round
	// has been checked
	rounds map[id.Round]*uint64
	// Configurations of UnknownRoundStore
	params UnknownRoundsParams

	// Key Value store to save data to disk
	kv *versioned.KV
}

// Allows configuration of UnknownRoundsStore parameters
type UnknownRoundsParams struct {
	// Maximum amount of checks of a round
	// before that round gets discarded
	MaxChecks uint64
}

// Returns a default set of UnknownRoundsParams
func DefaultUnknownRoundsParams() UnknownRoundsParams {
	return UnknownRoundsParams{
		MaxChecks: defaultMaxCheck,
	}
}

// Build and return new UnknownRounds object
func NewUnknownRoundsStore(kv *versioned.KV,
	params UnknownRoundsParams) *UnknownRoundsStore {
	// Build the UnmixedMessagesMap
	// Modify the prefix of the KV
	kv = kv.Prefix(unknownRoundPrefix)

	return &UnknownRoundsStore{
		rounds: make(map[id.Round]*uint64),
		params: params,
		kv:     kv,
	}
}

// LoadUnknownRoundsStore loads the data for a UnknownRoundStore from disk into an object
func LoadUnknownRoundsStore(kv *versioned.KV, params UnknownRoundsParams) (*UnknownRoundsStore, error) {
	kv = kv.Prefix(unknownRoundPrefix)

	urs := NewUnknownRoundsStore(kv, params)

	// Get the versioned data from the kv
	obj, err := kv.Get(unknownRoundsStorageKey, unknownRoundsStorageVersion)
	if err != nil {
		return nil, err
	}

	// Process the data into the object
	err = urs.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	return urs, nil
}

// Iterate iterates over all rounds. First it runs the
// checker function on the stored rounds:
// If true, it removes from the map and adds to the return slice
// If false, it increments the counter and if it has passed the maxChecks
// in params, it removes from the map
// Afterwards it adds the roundToAdd to the map if an entry isn't present
// Finally it saves the modified map to disk.
func (urs *UnknownRoundsStore) Iterate(checker func(rid id.Round) bool,
	roundsToAdd ...[]id.Round) ([]id.Round, error) {

	returnSlice := make([]id.Round, 0)
	// Check the rounds stored
	for rnd := range urs.rounds {
		ok := checker(rnd)
		if ok {
			// If true, Append to the return list and remove from the map
			returnSlice = append(returnSlice, rnd)
			delete(urs.rounds, rnd)
		} else {
			// If false, we increment the check counter for that round
			totalChecks := atomic.AddUint64(urs.rounds[rnd], 1)

			// If the round has been checked the maximum amount,
			// the rond is removed from the map
			if totalChecks > urs.params.MaxChecks {
				delete(urs.rounds, rnd)
			}
		}

	}

	// Iterate over all rounds passed in
	for _, roundSlice := range roundsToAdd {
		for _, rnd := range roundSlice {
			// Process non-tracked rounds into map
			if _, ok := urs.rounds[rnd]; !ok {
				newCheck := uint64(0)
				urs.rounds[rnd] = &newCheck
			}

		}
	}

	return returnSlice, urs.save()
}

// save stores the unknown rounds store.
func (urs *UnknownRoundsStore) save() error {
	now := time.Now()

	// Serialize the map
	data, err := json.Marshal(urs.rounds)
	if err != nil {
		return err
	}

	// Construct versioning object
	obj := versioned.Object{
		Version:   unknownRoundsStorageVersion,
		Timestamp: now,
		Data:      data,
	}

	// Save to disk
	return urs.kv.Set(unknownRoundsStorageKey, unknownRoundsStorageVersion, &obj)
}

// unmarshal loads the serialized round data into the UnknownRoundsStore map
func (urs *UnknownRoundsStore) unmarshal(b []byte) error {
	return json.Unmarshal(b, &urs.rounds)
}
