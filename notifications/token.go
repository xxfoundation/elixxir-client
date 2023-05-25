package notifications

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	notifCrypto "gitlab.com/elixxir/crypto/notifications"
	"gitlab.com/xx_network/primitives/netTime"
)

var (
	ErrNoTokenRegistered = errors.New("cannot do operation, no token is " +
		"registered with the remote")
)

// AddToken registers the Token with the remote server if this manager is
// in set to register, otherwise it will return ErrRemoteRegistrationDisabled
// This will add the token to the list of tokens which are forwarded the messages
// for connected IDs.
// the App will tell the server what App to forward the notifications to.
func (m *manager) AddToken(token, app string) error {

	m.mux.Lock()
	defer m.mux.Unlock()
	ts := netTime.Now().UTC()

	stream := m.rng.GetStream()
	tokenSig, err := notifCrypto.SignToken(m.transmissionRSA, token, app, ts,
		notifCrypto.RegisterTokenTag, stream)
	if err != nil {
		return err
	}

	_, err = m.comms.RegisterToken(m.notificationHost, &pb.RegisterTokenRequest{
		App:                         app,
		Token:                       token,
		TransmissionRsaPem:          m.transmissionRSAPubPem,
		TransmissionSalt:            m.transmissionSalt,
		RegistrationTimestamp:       m.registrationTimestampNs,
		TransmissionRsaRegistrarSig: m.transmissionRegistrationValidationSignature,
		RequestTimestamp:            ts.UnixNano(),
		TokenSignature:              tokenSig,
	})

	if err != nil {
		return err
	}

	m.setTokenUnsafe(token, app)
	return nil
}

// RemoveToken unregistered the currently registered Token with the remote
// server. Only can be called if this manager is set to register, otherwise
// it will return ErrRemoteRegistrationDisabled. If no token is registered,
// ErrNoTokenRegistered will be returned.
// This will remove all registered identities if it is the last token for the
// given identity,
func (m *manager) RemoveToken() error {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.token.Token == "" {
		return errors.WithStack(ErrNoTokenRegistered)
	}

	ts := netTime.Now().UTC()

	stream := m.rng.GetStream()
	tokenSig, err := notifCrypto.SignToken(m.transmissionRSA, m.token.Token,
		m.token.App, ts, notifCrypto.UnregisterTokenTag, stream)
	if err != nil {
		return err
	}

	_, err = m.comms.UnregisterToken(m.notificationHost, &pb.UnregisterTokenRequest{
		App:                m.token.App,
		Token:              m.token.Token,
		TransmissionRsaPem: m.transmissionRSAPubPem,
		RequestTimestamp:   ts.UnixNano(),
		TokenSignature:     tokenSig,
	})

	if err != nil {
		return err
	}

	m.deleteTokenUnsafe()
	return nil
}
