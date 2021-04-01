package rounds

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const unknownRoundStorageKey = "unknownRoundStorage"
const unknownRoundStorageVersion = 0

type EarliestRound struct {
	stored bool
	kv     *versioned.KV
	rid    id.Round
	mux    sync.Mutex
}

func NewEarliestRound(stored bool, kv *versioned.KV) *EarliestRound {
	ur := &EarliestRound{
		stored: stored,
		kv:     kv,
		rid:    0,
	}
	ur.save()
	return ur
}

func LoadEarliestRound(kv *versioned.KV) *EarliestRound {
	ur := &EarliestRound{
		stored: true,
		kv:     kv,
		rid:    0,
	}

	obj, err := kv.Get(unknownRoundStorageKey, unknownRoundStorageVersion)
	if err != nil {
		jww.FATAL.Panicf("Failed to get the unknown round: %+v", err)
	}

	err = json.Unmarshal(obj.Data, &ur.rid)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal the unknown round: %+v", err)
	}
	return ur
}

func (ur *EarliestRound) save() {
	if ur.stored {
		urStr, err := json.Marshal(&ur.rid)
		if err != nil {
			jww.FATAL.Panicf("Failed to marshal the unknown round: %+v", err)
		}

		// Create versioned object with data
		obj := &versioned.Object{
			Version:   unknownRoundStorageVersion,
			Timestamp: time.Now(),
			Data:      urStr,
		}

		err = ur.kv.Set(unknownRoundStorageKey,
			unknownRoundStorageVersion, obj)
		if err != nil {
			jww.FATAL.Panicf("Failed to store the unknown round: %+v", err)
		}

	}
}

func (ur *EarliestRound) Set(rid id.Round) (id.Round, bool) {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	changed := false
	if rid > ur.rid {
		changed = true
		ur.rid = rid
		ur.save()
	}
	return ur.rid, changed
}

func (ur *EarliestRound) Get() id.Round {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	return ur.rid
}

func (ur *EarliestRound) delete() {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	err := ur.kv.Delete(unknownRoundStorageKey, unknownRoundStorageVersion)
	if err != nil {
		jww.FATAL.Panicf("Failed to delete unknownRound storage: %+v", err)
	}
}
