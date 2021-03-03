package reception

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

type UnknownRound struct {
	stored bool
	kv     *versioned.KV
	rid    id.Round
	mux    sync.Mutex
}

func NewUnknownRound(stored bool, kv *versioned.KV) *UnknownRound {
	ur := &UnknownRound{
		stored: stored,
		kv:     kv,
		rid:    0,
	}
	ur.save()
	return ur
}

func LoadUnknownRound(kv *versioned.KV) *UnknownRound {
	ur := &UnknownRound{
		stored: true,
		kv:     kv,
		rid:    0,
	}

	obj, err := kv.Get(unknownRoundStorageKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to get the unknown round: %+v", err)
	}

	err = json.Unmarshal(obj.Data, &ur.rid)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal the unknown round: %+v", err)
	}
	return ur
}

func (ur *UnknownRound) save() {
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

		err = ur.kv.Set(unknownRoundStorageKey, obj)
		if err != nil {
			jww.FATAL.Panicf("Failed to store the unknown round: %+v", err)
		}

	}
}

func (ur *UnknownRound) Set(rid id.Round) id.Round {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	if rid > ur.rid {
		ur.rid = rid
		ur.save()
	}
	return ur.rid
}

func (ur *UnknownRound) Get() id.Round {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	return ur.rid
}

func (ur *UnknownRound) delete() {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	err := ur.kv.Delete(unknownRoundStorageKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to delete unknownRound storage: %+v", err)
	}
}
