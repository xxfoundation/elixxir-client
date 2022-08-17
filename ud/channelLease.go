package ud

import (
	"errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/channel"
	"time"
)

func (m *Manager) RequestChannelLease(userEd25519PubKey []byte) (int64, []byte, error) {
	return m.requestChannelLease(userEd25519PubKey, m.comms)
}

func (m *Manager) requestChannelLease(userEd25519PubKey []byte, comms channelLeaseComms) (int64, []byte, error) {
	ts := time.Now().UnixNano()
	privKey, err := m.user.GetReceptionIdentity().GetRSAPrivatePem()
	if err != nil {
		return 0, nil, err
	}
	stream := m.getRng().GetStream()
	fSig, err := channel.SignChannelIdentityRequest(userEd25519PubKey, ts, privKey, stream)
	if err != nil {
		return 0, nil, err
	}
	stream.Close()

	msg := &mixmessages.ChannelAuthenticationRequest{
		UserID:             m.user.GetReceptionIdentity().ID.Marshal(),
		UserEd25519PubKey:  userEd25519PubKey,
		Timestamp:          ts,
		UserSignedEdPubKey: fSig,
	}

	resp, err := comms.SendChannelAuthRequest(m.ud.host, msg)
	if err != nil {
		return 0, nil, err
	}

	ok := channel.VerifyChannelLease(resp.UDSignedEdPubKey, userEd25519PubKey, uint64(resp.Lease), nil)
	if !ok {
		return 0, nil, errors.New("error could not verify signature returned with channel lease")
	}

	return resp.Lease, resp.UDSignedEdPubKey, err
}
