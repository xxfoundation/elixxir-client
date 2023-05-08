package notifications

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
)

// registerNotificationUnsafe registers to receive notifications on the given
// id from remote. Only can be called if this manager is set to register, otherwise
// it will return ErrRemoteRegistrationDisabled.
// Must be called under the lock
func (m *manager) registerNotificationUnsafe(nid *id.ID) error {
	if m.notificationHost == nil {
		return errors.WithStack(ErrRemoteRegistrationDisabled)
	}
	m.comms.RegisterTrackedID(m.notificationHost, &pb.TrackedIntermediaryIDRequest{
		Token:                 "",
		IntermediaryId:        nil,
		TransmissionRsa:       nil,
		TransmissionSalt:      nil,
		TransmissionRsaSig:    nil,
		IIDTransmissionRsaSig: nil,
		RegistrationTimestamp: 0,
	}})
}
