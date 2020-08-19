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

	vect []uint64

	firstAvailable uint32
	numkeys        uint32
	numAvalible    uint32

	mux sync.RWMutex
}

type stateVectorDisk struct {
	vect           []uint64
	firstAvailable uint32
	numkeys        uint32
}

func newStateVector(ctx *context, key string, numkeys uint32) *stateVector {
	numBlocks := (numkeys + 63) / 64

	sv := &stateVector{
		ctx:            ctx,
		vect:           make([]uint64, numBlocks),
		key:            key,
		firstAvailable: 0,
		numAvalible:    numkeys,
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

	sv.numAvalible--

	return sv.save()
}

func (sv *stateVector) GetNumAvailable() uint32 {
	sv.mux.RLock()
	defer sv.mux.RUnlock()
	return sv.numAvalible
}

func (sv *stateVector) Used(keynum uint32) bool {
	sv.mux.RLock()
	defer sv.mux.RUnlock()

	block := keynum / 64
	pos := keynum % 64

	sv.vect[block] |= 1 << pos

	return (sv.vect[block]>>pos)&1 == 1
}

func (sv *stateVector) Next() (uint32, error) {
	sv.mux.Lock()
	defer sv.mux.Lock()

	if sv.firstAvailable >= sv.numkeys {
		return sv.numkeys, errors.New("No keys remaining")
	}

	next := sv.firstAvailable

	sv.nextAvailable()
	sv.numAvalible--

	return next, sv.save()

}

func (sv *stateVector) GetNumKeys() uint32 {
	return sv.numkeys
}

// finds the next used state and sets that as firstAvailable. This does not
// execute a store and a store must be executed after.
func (sv *stateVector) nextAvailable() {

	block := (sv.firstAvailable + 1) / 64
	pos := (sv.firstAvailable + 1) % 64

	for ; block < uint32(len(sv.vect)) && sv.vect[block]>>pos&1 == 1; pos++ {
		if pos == 64 {
			pos = 0
			block++
		}
	}

	sv.firstAvailable = pos
}

//ekv functions
func (sv *stateVector) marshal() ([]byte, error) {
	svd := stateVectorDisk{}

	svd.firstAvailable = sv.firstAvailable
	svd.numkeys = sv.numkeys
	svd.vect = sv.vect

	return json.Marshal(&svd)
}

func (sv *stateVector) unmarshal(b []byte) error {

	svd := stateVectorDisk{}

	err := json.Unmarshal(b, &svd)

	if err != nil {
		return err
	}

	sv.firstAvailable = svd.firstAvailable
	sv.numkeys = svd.numkeys
	sv.vect = svd.vect

	return nil
}

func makeStateVectorKey(prefix string, sid SessionID) string {
	return sid.String() + prefix
}
