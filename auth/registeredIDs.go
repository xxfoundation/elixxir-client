package auth

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type registeredIDs struct {
	r   map[id.ID]keypair
	mux sync.RWMutex
}

type keypair struct {
	privkey *cyclic.Int
	//generated from pubkey on instantiation
	pubkey *cyclic.Int
}

func newRegisteredIDs() *registeredIDs {
	return &registeredIDs{
		r:   make(map[id.ID]keypair),
		mux: sync.RWMutex{},
	}
}

func (rids *registeredIDs) addRegisteredIDs(id *id.ID, privkey, pubkey *cyclic.Int) {
	rids.mux.Lock()
	defer rids.mux.Unlock()
	rids.r[*id] = keypair{
		privkey: privkey,
		pubkey:  pubkey,
	}
}

func (rids *registeredIDs) getRegisteredIDs(id *id.ID) (keypair, bool) {
	rids.mux.Lock()
	defer rids.mux.Unlock()

	kp, exists := rids.r[*id]
	return kp, exists
}
