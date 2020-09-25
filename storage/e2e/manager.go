////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/primitives/id"
)

const managerPrefix = "Manager{partner:%s}"

type Manager struct {
	ctx *context
	kv  *versioned.KV

	partner *id.ID

	receive *sessionBuff
	send    *sessionBuff
}

// newManager creates the manager and its first send and receive sessions.
func newManager(ctx *context, kv *versioned.KV, partnerID *id.ID, myPrivKey,
	partnerPubKey *cyclic.Int, sendParams, receiveParams SessionParams) *Manager {

	kv = kv.Prefix(fmt.Sprintf(managerPrefix, partnerID))

	m := &Manager{
		ctx:     ctx,
		kv:      kv,
		partner: partnerID,
	}

	m.send = NewSessionBuff(m, "send")
	m.receive = NewSessionBuff(m, "receive")

	sendSession := newSession(m, myPrivKey, partnerPubKey, nil,
		sendParams, Send, SessionID{})

	m.send.AddSession(sendSession)

	receiveSession := newSession(m, myPrivKey, partnerPubKey, nil,
		receiveParams, Receive, SessionID{})

	m.receive.AddSession(receiveSession)

	return m
}

// loadManager loads a manager and all buffers and sessions from disk.
func loadManager(ctx *context, kv *versioned.KV, partnerID *id.ID) (*Manager, error) {

	kv = kv.Prefix(fmt.Sprintf(managerPrefix, partnerID))

	m := &Manager{
		ctx:     ctx,
		kv:      kv,
		partner: partnerID,
	}

	var err error

	m.send, err = LoadSessionBuff(m, "send")
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load partner key manager due to failure to "+
				"load the send session buffer")
	}

	m.receive, err = LoadSessionBuff(m, "receive")
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load partner key manager due to failure to "+
				"load the receive session buffer")
	}

	return m, nil
}

// NewReceiveSession creates a new receive session using the latest private key
// this user has sent and the new public key received from the partner. If the
// session already exists, then it will not be overwritten and the extant
// session will be returned with the bool set to true denoting a duplicate. This
// allows for support of duplicate key exchange triggering.
func (m *Manager) NewReceiveSession(partnerPubKey *cyclic.Int, params SessionParams,
	source *Session) (*Session, bool) {

	// Check if the session already exists
	baseKey := dh.GenerateSessionKey(source.myPrivKey, partnerPubKey, m.ctx.grp)
	sessionID := getSessionIDFromBaseKey(baseKey)

	if s := m.receive.GetByID(sessionID); s != nil {
		return s, true
	}

	// Create the session but do not save
	session := newSession(m, source.myPrivKey, partnerPubKey, baseKey, params,
		Receive, source.GetID())

	// Add the session to the buffer
	m.receive.AddSession(session)

	return session, false
}

// NewSendSession creates a new receive session using the latest public key
// received from the partner and a new private key for the user. Passing in a
// private key is optional. A private key will be generated if none is passed.
func (m *Manager) NewSendSession(myPrivKey *cyclic.Int, params SessionParams) *Session {
	// Find the latest public key from the other party
	sourceSession := m.receive.getNewestRekeyableSession()

	// Create the session
	session := newSession(m, myPrivKey, sourceSession.partnerPubKey, nil,
		params, Send, sourceSession.GetID())

	// Add the session to the send session buffer and return
	m.send.AddSession(session)

	return session
}

// GetKeyForSending gets the correct session to send with depending on the type
// of send.
func (m *Manager) GetKeyForSending(st params.SendType) (*Key, error) {
	switch st {
	case params.Standard:
		return m.send.getKeyForSending()
	case params.KeyExchange:
		return m.send.getKeyForRekey()
	default:
	}

	return nil, errors.Errorf("Cannot get session for invalid Send Type: %s", st)
}

// GetPartnerID returns a copy of the ID of the partner.
func (m *Manager) GetPartnerID() *id.ID {
	p := m.partner
	return p
}

// GetSendSession gets the send session of the passed ID. Returns nil if no
// session is found.
func (m *Manager) GetSendSession(sid SessionID) *Session {
	return m.send.GetByID(sid)
}

// GetReceiveSession gets the receive session of the passed ID. Returns nil if
// no session is found.
func (m *Manager) GetReceiveSession(sid SessionID) *Session {
	return m.receive.GetByID(sid)
}

// Confirm confirms a send session is known about by the partner.
func (m *Manager) Confirm(sid SessionID) error {
	return m.send.Confirm(sid)
}

// TriggerNegotiations returns a list of key exchange operations if any are
// necessary.
func (m *Manager) TriggerNegotiations() []*Session {
	return m.send.TriggerNegotiation()
}
