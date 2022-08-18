package ud

import (
	"crypto/ed25519"
	"errors"
	"time"

	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
)

func requestChannelLease(userPubKey ed25519.PublicKey, username string, comms channelLeaseComms, ud *userDiscovery, receptionIdentity xxdk.ReceptionIdentity, rngGenerator *fastRNG.StreamGenerator) (int64, []byte, error) {
	ts := time.Now().UnixNano()
	privKey, err := receptionIdentity.GetRSAPrivatePem()
	if err != nil {
		return 0, nil, err
	}
	rng := rngGenerator.GetStream()
	fSig, err := channel.SignChannelIdentityRequest(userPubKey, time.Unix(0, ts), privKey, rng)
	if err != nil {
		return 0, nil, err
	}
	rng.Close()

	msg := &mixmessages.ChannelAuthenticationRequest{
		UserID:             receptionIdentity.ID.Marshal(),
		UserEd25519PubKey:  userPubKey,
		Timestamp:          ts,
		UserSignedEdPubKey: fSig,
	}

	resp, err := comms.SendChannelAuthRequest(ud.host, msg)
	if err != nil {
		return 0, nil, err
	}

	ok := channel.VerifyChannelLease(resp.UDLeaseEd25519Signature, resp.UDSignedEdPubKey, userPubKey, uint64(resp.Lease), nil)
	if !ok {
		return 0, nil, errors.New("error could not verify signature returned with channel lease")
	}

	return resp.Lease, resp.UDSignedEdPubKey, err
}
