package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
)

func (m *Manager) Register(myID *id.ID, username string) error {

	msg := &pb.UDBUserRegistration{
		PermissioningSignature: nil,
		RSAPublicPem:           "",
		IdentityRegistration:   nil,
		IdentitySignature:      nil,
		Frs:                    nil,
		UID:                    myID.Bytes(),
		XXX_NoUnkeyedLiteral:   struct{}{},
		XXX_unrecognized:       nil,
		XXX_sizecache:          0,
	}

	_, _ = m.comms.SendRegisterUser(m.host, msg)

	return nil
}
