////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	unknownRoundsStorageKey     = "UnknownRoundsKey"
	unknownRoundsStorageVersion = 0
	unknownRoundPrefix          = "UnknownRoundPrefix"
	defaultMaxCheck             = 3
)

// UnknownRoundsParams allows configuration of UnknownRounds parameters.
type UnknownRoundsParams struct {
	// MaxChecks is the maximum amount of checks of a round before that round
	// gets discarded
	MaxChecks uint64

	// Stored determines if the unknown rounds is stored to disk
	Stored bool
}

// unknownRoundsParamsDisk will be the marshal-able and umarshal-able object.
type unknownRoundsParamsDisk struct {
	MaxChecks uint64
	Stored    bool
}

// DefaultUnknownRoundsParams returns a default set of UnknownRoundsParams.
func DefaultUnknownRoundsParams() UnknownRoundsParams {
	return UnknownRoundsParams{
		MaxChecks: defaultMaxCheck,
		Stored:    true,
	}
}

// GetParameters returns the default UnknownRoundsParams,
// or override with given parameters, if set.
func GetParameters(params string) (UnknownRoundsParams, error) {
	p := DefaultUnknownRoundsParams()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return UnknownRoundsParams{}, err
		}
	}
	return p, nil
}

// MarshalJSON adheres to the json.Marshaler interface.
func (urp UnknownRoundsParams) MarshalJSON() ([]byte, error) {
	urpDisk := unknownRoundsParamsDisk{
		MaxChecks: urp.MaxChecks,
		Stored:    urp.Stored,
	}

	return json.Marshal(&urpDisk)
}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (urp *UnknownRoundsParams) UnmarshalJSON(data []byte) error {
	urpDisk := unknownRoundsParamsDisk{}
	err := json.Unmarshal(data, &urpDisk)
	if err != nil {
		return err
	}

	*urp = UnknownRoundsParams{
		MaxChecks: urpDisk.MaxChecks,
		Stored:    urpDisk.Stored,
	}

	return nil
}

// UnknownRounds tracks data for unknown rounds. Should adhere to UnknownRounds
// interface.
type UnknownRounds struct {
	// Maps an unknown round to how many times the round has been checked
	rounds map[id.Round]*uint64

	// Configurations of UnknownRounds
	params UnknownRoundsParams

	// Key Value store to save data to disk
	kv versioned.KV

	mux sync.Mutex
}

// NewUnknownRounds builds and returns a new UnknownRounds object.
func NewUnknownRounds(kv versioned.KV,
	params UnknownRoundsParams) *UnknownRounds {

	urs, err := newUnknownRounds(kv, params)
	if err != nil {
		jww.FATAL.Panicf("Failed to create UnknownRoundStore: %+v", err)
	}

	if err := urs.save(); err != nil {
		jww.FATAL.Printf("Failed to store new UnknownRounds: %+v", err)
	}

	return urs
}

func newUnknownRounds(kv versioned.KV, params UnknownRoundsParams) (*UnknownRounds, error) {
	kv, err := kv.Prefix(unknownRoundPrefix)
	if err != nil {
		return nil, errors.Errorf("failed to add prefix %s to KV: %+v", unknownRoundPrefix, err)
	}

	urs := &UnknownRounds{
		rounds: make(map[id.Round]*uint64),
		params: params,
		kv:     kv,
	}

	return urs, nil
}

// LoadUnknownRounds loads the data for a UnknownRounds from disk into an
// object.
func LoadUnknownRounds(kv versioned.KV,
	params UnknownRoundsParams) *UnknownRounds {
	kv, err := kv.Prefix(unknownRoundPrefix)
	if err != nil {
		jww.FATAL.Panicf("Failed to add prefix %s to KV: %+v", unknownRoundPrefix, err)
	}

	urs, err := newUnknownRounds(kv, params)
	if err != nil {
		jww.FATAL.Panicf("Failed to create UnknownRoundStore: %+v", err)
	}

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

// Iterate iterates over all rounds. First it runs the checker function on the
// stored rounds:
// If true, it removes from the map and adds to the return slice.
// If false, it increments the counter and if it has passed the maxChecks in
// params, it removes from the map.
// Afterwards it adds the roundToAdd to the map if an entry isn't present.
// Finally, it saves the modified map to disk.
// The abandon function can be used to pass the abandoned round somewhere else.
func (urs *UnknownRounds) Iterate(checker func(rid id.Round) bool,
	roundsToAdd []id.Round, abandon func(round id.Round)) []id.Round {
	returnSlice := make([]id.Round, 0)
	urs.mux.Lock()
	defer urs.mux.Unlock()

	// Check the rounds stored
	for rnd := range urs.rounds {
		ok := checker(rnd)
		if ok {
			// If true, append to the return list and remove from the map
			returnSlice = append(returnSlice, rnd)
			delete(urs.rounds, rnd)
		} else {
			// If false, we increment the check counter for that round
			totalChecks := atomic.AddUint64(urs.rounds[rnd], 1)

			// If the round has been checked the maximum amount, then the rond
			// is removed from the map
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
		jww.FATAL.Panicf("Failed to save unknown rounds after edit: %+v", err)
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
	obj := &versioned.Object{
		Version:   unknownRoundsStorageVersion,
		Timestamp: now,
		Data:      data,
	}

	// Save to disk
	return urs.kv.Set(unknownRoundsStorageKey, obj)
}

func (urs *UnknownRounds) Delete() {
	urs.mux.Lock()
	defer urs.mux.Unlock()
	if urs.params.Stored {
		err := urs.kv.Delete(unknownRoundPrefix, unknownRoundsStorageVersion)
		if err != nil {
			jww.FATAL.Panicf("Failed to delete unknown rounds: %+v", err)
		}
	}

	urs.kv = nil
	urs.rounds = nil
}

// unmarshal loads the serialized round data into the UnknownRounds map.
func (urs *UnknownRounds) unmarshal(b []byte) error {
	return json.Unmarshal(b, &urs.rounds)
}

func (urs *UnknownRounds) Get(round id.Round) (bool, uint64) {
	urs.mux.Lock()
	defer urs.mux.Unlock()
	numCheck, exist := urs.rounds[round]
	if !exist {
		return false, 0
	}
	return exist, *numCheck

}
