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
	OutgoingCtxs map[id.User]*RekeyContext
	IncomingCtxs map[id.User]*RekeyContext
	outLock sync.Mutex
	inLock sync.Mutex
}

func NewRekeyManager() *RekeyManager {
	return &RekeyManager{
		OutgoingCtxs: make(map[id.User]*RekeyContext),
		IncomingCtxs: make(map[id.User]*RekeyContext),
	}
}

func (rkm *RekeyManager) AddOutCtx(partner *id.User,
	ctx *RekeyContext) {
	rkm.outLock.Lock()
	defer rkm.outLock.Unlock()
	rkm.OutgoingCtxs[*partner] = ctx
}

func (rkm *RekeyManager) GetOutCtx(partner *id.User) *RekeyContext {
	rkm.outLock.Lock()
	defer rkm.outLock.Unlock()
	return rkm.OutgoingCtxs[*partner]
}

func (rkm *RekeyManager) DeleteOutCtx(partner *id.User) {
	rkm.outLock.Lock()
	defer rkm.outLock.Unlock()
	delete(rkm.OutgoingCtxs, *partner)
}

func (rkm *RekeyManager) AddInCtx(partner *id.User,
	ctx *RekeyContext) {
	rkm.inLock.Lock()
	defer rkm.inLock.Unlock()
	rkm.IncomingCtxs[*partner] = ctx
}

func (rkm *RekeyManager) GetInCtx(partner *id.User) *RekeyContext {
	rkm.inLock.Lock()
	defer rkm.inLock.Unlock()
	return rkm.IncomingCtxs[*partner]
}

func (rkm *RekeyManager) DeleteInCtx(partner *id.User) {
	rkm.inLock.Lock()
	defer rkm.inLock.Unlock()
	delete(rkm.IncomingCtxs, *partner)
}
