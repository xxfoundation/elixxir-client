///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const currentStateVectorVersion = 0
const stateVectorKey = "stateVector"

type stateVector struct {
	kv  *versioned.KV
	key string

	// Bitfield for key states
	// If a key is clean, its bit will be 0
	// Otherwise, it's dirty/used/not available, and its bit will be 1
	vect []uint64

	firstAvailable uint32
	numkeys        uint32
	numAvailable   uint32

	mux sync.RWMutex
}

// Fields must be exported for json marshal to serialize them
type stateVectorDisk struct {
	Vect           []uint64
	FirstAvailable uint32
	NumAvailable   uint32
	Numkeys        uint32
}

func newStateVector(kv *versioned.KV, key string, numkeys uint32) (*stateVector, error) {
	numBlocks := (numkeys + 63) / 64

	sv := &stateVector{
		kv:             kv,
		vect:           make([]uint64, numBlocks),
		key:            stateVectorKey + key,
		firstAvailable: 0,
		numAvailable:   numkeys,
		numkeys:        numkeys,
	}

	return sv, sv.save()
}

func loadStateVector(kv *versioned.KV, key string) (*stateVector, error) {
	sv := &stateVector{
		kv:  kv,
		key: stateVectorKey + key,
	}

	obj, err := kv.Get(sv.key, currentStateVectorVersion)
	if err != nil {
		return nil, err
	}

	err = sv.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	return sv, nil
}

func (sv *stateVector) save() error {
	now := netTime.Now()

	data, err := sv.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentStateVectorVersion,
		Timestamp: now,
		Data:      data,
	}

	return sv.kv.Set(sv.key, currentStateVectorVersion, &obj)
}

func (sv *stateVector) Use(keynum uint32) {
	sv.mux.Lock()
	defer sv.mux.Unlock()

	block := keynum / 64
	pos := keynum % 64

	sv.vect[block] |= 1 << pos

	if keynum == sv.firstAvailable {
		sv.nextAvailable()
	}

	sv.numAvailable--

	if err := sv.save(); err != nil {
		jww.FATAL.Printf("Failed to save %s on Use(): %s", sv, err)
	}
}

func (sv *stateVector) GetNumAvailable() uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()
	return sv.numAvailable
}

func (sv *stateVector) GetNumUsed() uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()
	return sv.numkeys - sv.numAvailable
}

func (sv *stateVector) Used(keynum uint32) bool {
	sv.mux.RLock()
	defer sv.mux.RUnlock()

	return sv.used(keynum)
}

func (sv *stateVector) used(keynum uint32) bool {
	block := keynum / 64
	pos := keynum % 64

	return (sv.vect[block]>>pos)&1 == 1
}

func (sv *stateVector) Next() (uint32, error) {
	sv.mux.Lock()
	defer sv.mux.Unlock()

	if sv.firstAvailable >= sv.numkeys {
		return sv.numkeys, errors.New("No keys remaining")
	}

	next := sv.firstAvailable

	sv.nextAvailable()
	sv.numAvailable--

	if err := sv.save(); err != nil {
		jww.FATAL.Printf("Failed to save %s on Next(): %s", sv, err)
	}

	return next, nil

}

func (sv *stateVector) GetNumKeys() uint32 {
	return sv.numkeys
}

//returns a list of unused keys
func (sv *stateVector) GetUnusedKeyNums() []uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()

	keyNums := make([]uint32, 0, sv.numAvailable)

	for keyNum := sv.firstAvailable; keyNum < sv.numkeys; keyNum++ {
		if !sv.used(keyNum) {
			keyNums = append(keyNums, keyNum)
		}
	}

	return keyNums
}

//returns a list of used keys
func (sv *stateVector) GetUsedKeyNums() []uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()

	keyNums := make([]uint32, 0, sv.numkeys-sv.numAvailable)

	for keyNum := sv.firstAvailable; keyNum < sv.numkeys; keyNum++ {
		if sv.used(keyNum) {
			keyNums = append(keyNums, keyNum)
		}
	}

	return keyNums
}

//Adheres to the stringer interface
func (sv *stateVector) String() string {
	return fmt.Sprintf("stateVector: %s", sv.key)
}

//Deletes the state vector from storage
func (sv *stateVector) Delete() error {
	return sv.kv.Delete(sv.key, currentStateVectorVersion)
}

// finds the next used state and sets that as firstAvailable. This does not
// execute a store and a store must be executed after.
func (sv *stateVector) nextAvailable() {

	block := (sv.firstAvailable + 1) / 64
	pos := (sv.firstAvailable + 1) % 64

	for ; block < uint32(len(sv.vect)) && (sv.vect[block]>>pos)&1 == 1; pos++ {
		if pos == 63 {
			pos = 0
			block++
		}
	}

	sv.firstAvailable = block*64 + pos
}

//ekv functions
func (sv *stateVector) marshal() ([]byte, error) {
	svd := stateVectorDisk{}

	svd.FirstAvailable = sv.firstAvailable
	svd.Numkeys = sv.numkeys
	svd.NumAvailable = sv.numAvailable
	svd.Vect = sv.vect

	return json.Marshal(&svd)
}

func (sv *stateVector) unmarshal(b []byte) error {

	svd := stateVectorDisk{}

	err := json.Unmarshal(b, &svd)

	if err != nil {
		return err
	}

	sv.firstAvailable = svd.FirstAvailable
	sv.numkeys = svd.Numkeys
	sv.numAvailable = svd.NumAvailable
	sv.vect = svd.Vect

	return nil
}
