package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

type RekeyContext struct {
	BaseKey  *cyclic.Int
	PrivKey  *cyclic.Int
	PubKey   *cyclic.Int
}

type RekeyManager struct {
	Ctxs map[id.User]*RekeyContext
	lock sync.Mutex
}

func NewRekeyManager() *RekeyManager {
	return &RekeyManager{
		Ctxs: make(map[id.User]*RekeyContext),
	}
}

func (rkm *RekeyManager) AddCtx(partner *id.User,
	ctx *RekeyContext) {
	rkm.lock.Lock()
	defer rkm.lock.Unlock()
	rkm.Ctxs[*partner] = ctx
}

func (rkm *RekeyManager) GetCtx(partner *id.User) *RekeyContext {
	rkm.lock.Lock()
	defer rkm.lock.Unlock()
	return rkm.Ctxs[*partner]
}

func (rkm *RekeyManager) DeleteCtx(partner *id.User) {
	rkm.lock.Lock()
	defer rkm.lock.Unlock()
	delete(rkm.Ctxs, *partner)
}
