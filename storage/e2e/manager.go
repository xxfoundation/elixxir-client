package e2e

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	jww "github.com/spf13/jwalterweatherman"
)

type Manager struct {
	ctx *context

	partner *id.ID

	receive *sessionBuff
	send    *sessionBuff
}

// create the manager and its first send and receive sessions
func newManager(ctx *context, partnerID *id.ID, myPrivKey *cyclic.Int,
	partnerPubKey *cyclic.Int, sendParams, receiveParams SessionParams) *Manager {
	m := &Manager{
		ctx:     ctx,
		partner: partnerID,
	}

	m.send = NewSessionBuff(m, "send")
	m.receive = NewSessionBuff(m, "receive")

	sendSession := newSession(m, myPrivKey, partnerPubKey, sendParams, Send)

	m.send.AddSession(sendSession)

	receiveSession := newSession(m, myPrivKey, partnerPubKey, receiveParams, Receive)

	m.receive.AddSession(receiveSession)

	return m
}

//loads a manager and all buffers and sessions from disk
func loadManager(ctx *context, partnerID *id.ID) (*Manager, error) {
	m := &Manager{
		ctx:     ctx,
		partner: partnerID,
	}

	var err error

	m.send, err = LoadSessionBuff(m, "send", partnerID)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load partner key manager due to failure to "+
				"load the send session buffer")
	}

	m.receive, err = LoadSessionBuff(m, "receive", partnerID)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load partner key manager due to failure to "+
				"load the receive session buffer")
	}

	return m, nil
}

//gets a copy of the ID of the partner
func (m *Manager) GetPartnerID() *id.ID {
	p := m.partner
	return p
}

// creates a new receive session using the latest private key this user has sent
// and the new public key received from the partner.
func (m *Manager) NewReceiveSession(partnerPubKey *cyclic.Int, params SessionParams) *Session {
	//find your last confirmed private key
	myPrivKey := m.send.GetNewestRekeyableSession().GetMyPrivKey()

	//create the session
	session := newSession(m, myPrivKey, partnerPubKey, params, Receive)

	//add the session to the buffer
	m.receive.AddSession(session)

	return session
}

// creates a new receive session using the latest public key received from the
// partner and a mew private key for the user
// passing in a private key is optional. a private key will be generated if
// none is passed
func (m *Manager) NewSendSession(myPrivKey *cyclic.Int, params SessionParams) *Session {
	//find the latest public key from the other party
	partnerPubKey := m.receive.GetNewestRekeyableSession().partnerPubKey

	//create the session
	session := newSession(m, myPrivKey, partnerPubKey, params, Send)

	//add the session to the send session buffer and return
	m.send.AddSession(session)

	return session
}

// gets the correct session to send with depending on the type of send
func (m *Manager) GetSendingSession(st params.SendType) *Session {
	switch st {
	case params.Standard:
		return m.send.GetSessionForSending()
	case params.KeyExchange:
		return m.send.GetNewestRekeyableSession()
	default:
		jww.ERROR.Printf("Cannot get session for invalid Send Type: %s",
			st)
	}

	return nil
}

// Confirms a send session is known about by the partner
func (m *Manager) Confirm(sid SessionID) error {
	return m.send.Confirm(sid)
}

// returns a list of key exchange operations if any are necessary
func (m *Manager) TriggerNegotiations() []*Session {
	return m.send.TriggerNegotiation()
}
