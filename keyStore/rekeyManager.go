package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

type RekeyContext struct {
	BaseKey *cyclic.Int
	PrivKey *cyclic.Int
	PubKey  *cyclic.Int
}

type RekeyKeys struct {
	CurrPrivKey *cyclic.Int
	CurrPubKey  *cyclic.Int
	NewPrivKey  *cyclic.Int
	NewPubKey   *cyclic.Int
}

func (k *RekeyKeys) RotateKeysIfReady() {
	if k.NewPrivKey != nil && k.NewPubKey != nil {
		k.CurrPrivKey = k.NewPrivKey
		k.CurrPubKey = k.NewPubKey
		k.NewPrivKey = nil
		k.NewPubKey = nil
	}
}

type RekeyManager struct {
	Ctxs map[id.User]*RekeyContext
	Keys map[id.User]*RekeyKeys
	lock sync.Mutex
}

func NewRekeyManager() *RekeyManager {
	return &RekeyManager{
		Ctxs: make(map[id.User]*RekeyContext),
		Keys: make(map[id.User]*RekeyKeys),
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

func (rkm *RekeyManager) AddKeys(partner *id.User,
	keys *RekeyKeys) {
	rkm.lock.Lock()
	defer rkm.lock.Unlock()
	rkm.Keys[*partner] = keys
}

func (rkm *RekeyManager) GetKeys(partner *id.User) *RekeyKeys {
	rkm.lock.Lock()
	defer rkm.lock.Unlock()
	return rkm.Keys[*partner]
}

func (rkm *RekeyManager) DeleteKeys(partner *id.User) {
	rkm.lock.Lock()
	defer rkm.lock.Unlock()
	delete(rkm.Keys, *partner)
}
