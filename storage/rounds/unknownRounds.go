///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"sync/atomic"
	"time"
)

const unknownRoundsStorageKey = "UnknownRoundsKey"
const unknownRoundsStorageVersion = 0
const unknownRoundPrefix = "UnknownRoundPrefix"

type UnknownRounds interface {
	Iterate(checker func(rid id.Round) bool, roundsToAdd ...[]id.Round) ([]id.Round, error)
}

// UnknownRoundsStore tracks data for unknown rounds
// Should adhere to UnknownRounds interface
type UnknownRoundsStore struct {
	// Maps an unknown round to how many times the round
	// has been checked
	Round map[id.Round]*uint64
	// Configurations of UnknownRoundStore
	Params UnknownRoundsParams

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
		MaxChecks: 3,
	}
}

// Build and return new UnknownRounds object
func NewUnknownRoundsStore(kv *versioned.KV, params UnknownRoundsParams) *UnknownRoundsStore {
	// Build the UnmixedMessagesMap
	kv.Prefix(unknownRoundPrefix)
	return &UnknownRoundsStore{
		Round:  make(map[id.Round]*uint64),
		Params: params,
		kv:     kv,
	}
}

// LoadUnknownRoundsStore loads the data for a UnknownRoundStore from disk into an object
func LoadUnknownRoundsStore(kv *versioned.KV, params UnknownRoundsParams) (*UnknownRoundsStore, error) {

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

// Iterate iterates over all rounds. First it adds the round
// rounds to the map if an entry isn't present. Then  it
// runs the checker function on them:
// If true, it removes from the map and adds to the return slice
// If false, it increments the counter and if it has passed the maxChecks
// in params, it removes from the map
// Finally it saves the map to disk.
func (urs *UnknownRoundsStore) Iterate(checker func(rid id.Round) bool,
	roundsToAdd ...[]id.Round) ([]id.Round, error) {

	returnSlice := make([]id.Round, 0)
	// Iterate over all rounds passed in
	for _, roundSlice := range roundsToAdd {
		for _, rnd := range roundSlice {
			// Process non-tracked rounds into map
			if _, ok := urs.Round[rnd]; !ok {
				newCheck := uint64(0)
				urs.Round[rnd] = &newCheck
			}

			// Check the round
			ok := checker(rnd)
			if ok {
				// If true, Append to the return list and remove from the map
				returnSlice = append(returnSlice, rnd)
				delete(urs.Round, rnd)
			} else {
				// If false, we increment the check counter for that round
				totalChecks := atomic.AddUint64(urs.Round[rnd], 1)

				// If the round has been checked the maximum amount,
				// the rond is removed from the map
				if totalChecks > urs.Params.MaxChecks {
					delete(urs.Round, rnd)
				}
			}
		}
	}

	return returnSlice, urs.save()
}

// save stores the unknown rounds store.
func (urs *UnknownRoundsStore) save() error {
	now := time.Now()

	data, err := urs.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   unknownRoundsStorageVersion,
		Timestamp: now,
		Data:      data,
	}

	return urs.kv.Set(unknownRoundsStorageKey, unknownRoundsStorageVersion, &obj)
}

// marshal serializes the round map for disk storage
func (urs *UnknownRoundsStore) marshal() ([]byte, error) {
	rounds := make([]roundDisk, len(urs.Round))
	index := 0
	for rnd, checks := range urs.Round {
		rounds[index] = roundDisk{
			Checks:  atomic.LoadUint64(checks),
			RoundId: uint64(rnd),
		}
		index++
	}

	return json.Marshal(&rounds)
}

// unmarshal loads the serialized round data into the
// UnknownRoundsStore map
func (urs *UnknownRoundsStore) unmarshal(b []byte) error {

	var roundsDisk []roundDisk
	if err := json.Unmarshal(b, &roundsDisk); err != nil {
		return errors.WithMessagef(err, "Failed to "+
			"unmarshal UnknownRoundStore")
	}

	for _, rndDisk := range roundsDisk {
		urs.Round[id.Round(rndDisk.RoundId)] = &rndDisk.Checks

	}

	return nil
}
