////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// partnerCallbacks is a thread-safe wrapper for Callbacks specific to partnerIds
// For auth operations with a specific partner, these Callbacks will be used instead
type partnerCallbacks struct {
	callbacks map[id.ID]Callbacks
	sync.RWMutex
}

// AddPartnerCallback that overrides the generic auth callback for the given partnerId
func (p *partnerCallbacks) AddPartnerCallback(partnerId *id.ID, cb Callbacks) {
	p.Lock()
	defer p.Unlock()
	if _, ok := p.callbacks[*partnerId]; !ok {
		p.callbacks[*partnerId] = cb
	}
}

// DeletePartnerCallback that overrides the generic auth callback for the given partnerId
func (p *partnerCallbacks) DeletePartnerCallback(partnerId *id.ID) {
	p.Lock()
	defer p.Unlock()
	if _, ok := p.callbacks[*partnerId]; ok {
		delete(p.callbacks, *partnerId)
	}
}

// getPartnerCallback returns the Callbacks for the given partnerId
func (p *partnerCallbacks) getPartnerCallback(partnerId *id.ID) Callbacks {
	return p.callbacks[*partnerId]
}
