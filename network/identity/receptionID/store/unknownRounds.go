///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"sync/atomic"
)

const (
	unknownRoundsStorageKey     = "UnknownRoundsKey"
	unknownRoundsStorageVersion = 0
	unknownRoundPrefix          = "UnknownRoundPrefix"
	defaultMaxCheck             = 3
)

// UnknownRounds tracks data for unknown rounds
// Should adhere to UnknownRounds interface
type UnknownRounds struct {
	// Maps an unknown round to how many times the round
	// has been checked
	rounds map[id.Round]*uint64
	// Configurations of UnknownRoundStore
	params UnknownRoundsParams

	// Key Value store to save data to disk
	kv *versioned.KV

	mux sync.Mutex
}

// Allows configuration of UnknownRounds parameters
type UnknownRoundsParams struct {
	// Maximum amount of checks of a round
	// before that round gets discarded
	MaxChecks uint64
	//Determines if the unknown rounds is stored to disk
	Stored bool
}

// Returns a default set of UnknownRoundsParams
func DefaultUnknownRoundsParams() UnknownRoundsParams {
	return UnknownRoundsParams{
		MaxChecks: defaultMaxCheck,
		Stored:    true,
	}
}

// Build and return new UnknownRounds object
func NewUnknownRounds(kv *versioned.KV,
	params UnknownRoundsParams) *UnknownRounds {

	urs := newUnknownRounds(kv, params)

	if err := urs.save(); err != nil {
		jww.FATAL.Printf("Failed to store New Unknown Rounds: %+v", err)
	}

	return urs
}

func newUnknownRounds(kv *versioned.KV,
	params UnknownRoundsParams) *UnknownRounds {
	// Build the UnmixedMessagesMap
	// Modify the prefix of the KV
	kv = kv.Prefix(unknownRoundPrefix)

	urs := &UnknownRounds{
		rounds: make(map[id.Round]*uint64),
		params: params,
		kv:     kv,
	}

	return urs
}

// LoadUnknownRounds loads the data for a UnknownRoundStore from disk into an object
func LoadUnknownRounds(kv *versioned.KV,
	params UnknownRoundsParams) *UnknownRounds {
	kv = kv.Prefix(unknownRoundPrefix)

	urs := newUnknownRounds(kv, params)

	// get the versioned data from the kv
	obj, err := kv.Get(unknownRoundsStorageKey, unknownRoundsStorageVersion)
	if err != nil {
		jww.FATAL.Panicf("Failed to load UnknownRounds: %+v", err)
	}

	// Process the data into the object
	err = urs.unmarshal(obj.Data)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal UnknownRounds: %+v", err)
	}

	return urs
}

// Iterate iterates over all rounds. First it runs the
// checker function on the stored rounds:
// If true, it removes from the map and adds to the return slice
// If false, it increments the counter and if it has passed the maxChecks
// in params, it removes from the map
// Afterwards it adds the roundToAdd to the map if an entry isn't present
// Finally it saves the modified map to disk.
// The abandon function can be used to pass the abandoned round somewhere else
func (urs *UnknownRounds) Iterate(checker func(rid id.Round) bool,
	roundsToAdd []id.Round, abandon func(round id.Round)) []id.Round {
	returnSlice := make([]id.Round, 0)
	urs.mux.Lock()
	defer urs.mux.Unlock()
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
				localRnd := rnd
				go abandon(localRnd)
				delete(urs.rounds, rnd)
			}
		}

	}

	// Iterate over all rounds passed in
	for _, rnd := range roundsToAdd {
		// Process non-tracked rounds into map
		if _, ok := urs.rounds[rnd]; !ok {
			newCheck := uint64(0)
			urs.rounds[rnd] = &newCheck
		}
	}

	if err := urs.save(); err != nil {
		jww.FATAL.Panicf("Failed to save unknown reounds after "+
			"edit: %+v", err)
	}

	return returnSlice
}

// save stores the unknown rounds store.
func (urs *UnknownRounds) save() error {
	if !urs.params.Stored {
		return nil
	}
	now := netTime.Now()

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

// save stores the unknown rounds store.
func (urs *UnknownRounds) Delete() {
	urs.mux.Lock()
	defer urs.mux.Unlock()
	if urs.params.Stored {
		if err := urs.kv.Delete(unknownRoundPrefix, unknownRoundsStorageVersion); err != nil {
			jww.FATAL.Panicf("Failed to delete unknown rounds: %+v", err)
		}
	}

	urs.kv = nil
	urs.rounds = nil
}

// unmarshal loads the serialized round data into the UnknownRounds map
func (urs *UnknownRounds) unmarshal(b []byte) error {
	return json.Unmarshal(b, &urs.rounds)
}

func (urs *UnknownRounds) Get(round id.Round) (present bool, numchecked uint64) {
	urs.mux.Lock()
	defer urs.mux.Unlock()
	numcheck, exist := urs.rounds[round]
	if !exist {
		return false, 0
	}
	return exist, *numcheck

}
