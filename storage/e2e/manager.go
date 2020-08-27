package e2e

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

type Manager struct {
	ctx *context

	partner *id.ID

	receive *sessionBuff
	send    *sessionBuff
}

// create the manager and its first send and receive sessions
func newManager(ctx *context, partnerID *id.ID, myPrivKey *cyclic.Int,
	partnerPubKey *cyclic.Int, sendParams, receiveParams SessionParams) (*Manager, error) {
	m := &Manager{
		ctx:     ctx,
		partner: partnerID,
	}

	m.send = NewSessionBuff(m, "send")
	m.receive = NewSessionBuff(m, "receive")

	sendSession, err := newSession(m, myPrivKey, partnerPubKey, sendParams, Send)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to create the send session")
	}

	err = m.send.AddSession(sendSession)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to add the send session to buffer")
	}

	receiveSession, err := newSession(m, myPrivKey, partnerPubKey, receiveParams, Receive)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to create the receive session")
	}

	err = m.receive.AddSession(receiveSession)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to add the receive session to buffer")
	}

	return m, nil
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
func (m *Manager) NewReceiveSession(partnerPubKey *cyclic.Int, params SessionParams) error {
	//find your last confirmed private key
	myPrivKey := m.send.GetNewestConfirmed().GetMyPrivKey()

	//create the session
	session, err := newSession(m, myPrivKey, partnerPubKey, params, Receive)

	if err != nil {
		return err
	}

	//add the session to the buffer
	err = m.receive.AddSession(session)
	if err != nil {
		//delete the session if it failed to add to the buffer
		session.Delete()
	}

	return err
}

// creates a new receive session using the latest public key received from the
// partner and a mew private key for the user
// passing in a private key is optional. a private key will be generated if
// none is passed
func (m *Manager) NewSendSession(myPrivKey *cyclic.Int, params SessionParams) error {
	//find the latest public key from the other party
	partnerPubKey := m.receive.GetNewestConfirmed().partnerPubKey

	session, err := newSession(m, myPrivKey, partnerPubKey, params, Send)
	if err != nil {
		return err
	}

	//add the session to the buffer
	err = m.send.AddSession(session)
	if err != nil {
		//delete the session if it failed to add to the buffer
		session.Delete()
	}

	return err
}

// gets the session buffer for message reception
func (m *Manager) GetSendingSession() *Session {
	return m.send.GetSessionForSending()
}

// Confirms a send session is known about by the partner
func (m *Manager) Confirm(sid SessionID) error {
	return m.send.Confirm(sid)
}
