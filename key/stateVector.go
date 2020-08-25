package key

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage"
	"sync"
	"time"
)

type stateVector struct {
	ctx *context
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

func newStateVector(ctx *context, key string, numkeys uint32) *stateVector {
	numBlocks := (numkeys + 63) / 64

	sv := &stateVector{
		ctx:            ctx,
		vect:           make([]uint64, numBlocks),
		key:            key,
		firstAvailable: 0,
		numAvailable:   numkeys,
		numkeys:        numkeys,
	}

	return sv
}

func loadStateVector(ctx *context, key string) (*stateVector, error) {
	sv := &stateVector{
		ctx: ctx,
		key: key,
	}

	obj, err := ctx.kv.Get(key)
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
	now, err := time.Now().MarshalText()
	if err != nil {
		return err
	}

	data, err := sv.marshal()
	if err != nil {
		return err
	}

	obj := storage.VersionedObject{
		Version:   currentSessionVersion,
		Timestamp: now,
		Data:      data,
	}

	return sv.ctx.kv.Set(sv.key, &obj)
}

func (sv *stateVector) Use(keynum uint32) error {
	sv.mux.Lock()
	defer sv.mux.Unlock()

	block := keynum / 64
	pos := keynum % 64

	sv.vect[block] |= 1 << pos

	if keynum == sv.firstAvailable {
		sv.nextAvailable()
	}

	sv.numAvailable--

	return sv.save()
}

func (sv *stateVector) GetNumAvailable() uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()
	return sv.numAvailable
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

	return next, sv.save()

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

func makeStateVectorKey(prefix string, sid SessionID) string {
	return sid.String() + prefix
}
