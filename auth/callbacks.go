package auth

import (
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type callbackMap struct {
	generalCallback  []interface{}
	specificCallback map[id.ID]interface{}
	overrideCallback []interface{}
	mux              sync.RWMutex
}

func newCallbackMap() *callbackMap {
	return &callbackMap{
		generalCallback:  make([]interface{}, 0),
		specificCallback: make(map[id.ID]interface{}),
		overrideCallback: make([]interface{}, 0),
	}
}

//adds a general callback. This will be preempted by any specific callback
func (cm *callbackMap) AddGeneral(cb interface{}) {
	cm.mux.Lock()
	cm.generalCallback = append(cm.generalCallback, cb)
	cm.mux.Unlock()
}

//adds an override callback. This will NOT be preempted by any callback
func (cm *callbackMap) AddOverride(cb interface{}) {
	cm.mux.Lock()
	cm.overrideCallback = append(cm.overrideCallback, cb)
	cm.mux.Unlock()
}

// adds a callback for a specific user ID. Only only callback can exist for a
// user ID. False will be returned if a callback already exists and the new
// one was not added
func (cm *callbackMap) AddSpecific(id *id.ID, cb interface{}) bool {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	if _, ok := cm.specificCallback[*id]; ok {
		return false
	}
	cm.specificCallback[*id] = cb
	return true
}

// removes a callback for a specific user ID if it exists.
func (cm *callbackMap) RemoveSpecific(id *id.ID) {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	delete(cm.specificCallback, *id)
}

//get all callback which fit with the passed id
func (cm *callbackMap) Get(id *id.ID) []interface{} {
	cm.mux.RLock()
	defer cm.mux.RUnlock()
	cbList := cm.overrideCallback

	if specific, ok := cm.specificCallback[*id]; ok {
		cbList = append(cbList, specific)
	} else {
		cbList = append(cbList, cm.generalCallback)
	}

	return cbList
}
