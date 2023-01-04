package message

import (
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type FallthroughManager struct {
	lookup map[id.ID]Processor
	mux    sync.Mutex
}

func newFallthroughManager() FallthroughManager {
	return FallthroughManager{
		lookup: make(map[id.ID]Processor),
	}
}

func (f *FallthroughManager) AddFallthrough(c *id.ID, p Processor) {
	f.mux.Lock()
	defer f.mux.Unlock()
	f.lookup[*c] = p
}

func (f *FallthroughManager) RemoveFallthrough(c *id.ID) {
	f.mux.Lock()
	defer f.mux.Unlock()
	delete(f.lookup, *c)
}

func (f *FallthroughManager) getFallthrough(c *id.ID) (Processor, bool) {
	f.mux.Lock()
	defer f.mux.Unlock()
	p, exists := f.lookup[*c]
	return p, exists
}
