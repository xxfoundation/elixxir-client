////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/crypto/contact"
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

// DefaultAuthCallbacks is a simple structure for providing a default Callbacks implementation
// It should generally not be used.
type DefaultAuthCallbacks struct{}

// Confirm will be called when an auth Confirm message is processed.
func (a DefaultAuthCallbacks) Confirm(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}

// Request will be called when an auth Request message is processed.
func (a DefaultAuthCallbacks) Request(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}

// Reset will be called when an auth Reset operation occurs.
func (a DefaultAuthCallbacks) Reset(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}
