////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	earliestRoundStorageKey     = "unknownRoundStorage"
	earliestRoundStorageVersion = 0
)

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

	obj, err := kv.Get(earliestRoundStorageKey, earliestRoundStorageVersion)
	if err != nil {
		jww.FATAL.Panicf("Failed to get the earliest round: %+v", err)
	}

	err = json.Unmarshal(obj.Data, &ur.rid)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal the earliest round: %+v", err)
	}
	return ur
}

func (ur *EarliestRound) save() {
	if ur.stored {
		urStr, err := json.Marshal(&ur.rid)
		if err != nil {
			jww.FATAL.Panicf("Failed to marshal the earliest round: %+v", err)
		}

		// Create versioned object with data
		obj := &versioned.Object{
			Version:   earliestRoundStorageVersion,
			Timestamp: netTime.Now(),
			Data:      urStr,
		}

		err = ur.kv.Set(earliestRoundStorageKey, obj)
		if err != nil {
			jww.FATAL.Panicf("Failed to store the earliest round: %+v", err)
		}

	}
}

// Set returns the updated earliest round, the old earliest round, and if they
// are changed. Updates the earliest round if it is newer than stored one.
func (ur *EarliestRound) Set(rid id.Round) (id.Round, id.Round, bool) {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	changed := false
	old := ur.rid
	if rid > ur.rid {
		changed = true
		ur.rid = rid
		ur.save()
	}
	return ur.rid, old, changed
}

func (ur *EarliestRound) Get() id.Round {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	return ur.rid
}

func (ur *EarliestRound) delete() {
	ur.mux.Lock()
	defer ur.mux.Unlock()
	err := ur.kv.Delete(earliestRoundStorageKey, earliestRoundStorageVersion)
	if err != nil {
		jww.FATAL.Panicf("Failed to delete earliest storage: %+v", err)
	}
}
