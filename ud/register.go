package ud

import (
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
)

func (m *Manager)Register(myID *id.ID, username string)error{

	msg := &messages.AuthenticatedMessage{
		ID:                   myID.Bytes(),
		Signature:            nil,
		Token:                nil,
		Client:               nil,
		Message:              nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}

	m.comms.SendRegisterUser(m.host)


}