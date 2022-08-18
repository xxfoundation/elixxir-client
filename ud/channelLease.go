package ud

import (
	"crypto/ed25519"
	"errors"
	"time"

	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
)

func requestChannelLease(userPubKey ed25519.PublicKey,
	username string,
	comms channelLeaseComms,
	host *connect.Host,
	receptionIdentity xxdk.ReceptionIdentity,
	rngGenerator *fastRNG.StreamGenerator,
	signerPubKey ed25519.PublicKey) (int64, []byte, error) {

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

	msg := &mixmessages.ChannelLeaseRequest{
		UserID:                 receptionIdentity.ID.Marshal(),
		UserEd25519PubKey:      userPubKey,
		Timestamp:              ts,
		UserPubKeyRSASignature: fSig,
	}

	resp, err := comms.SendChannelLeaseRequest(host, msg)
	if err != nil {
		return 0, nil, err
	}

	ok := channel.VerifyChannelLease(resp.UDLeaseEd25519Signature,
		userPubKey, username, time.Unix(0, resp.Lease), signerPubKey)
	if !ok {
		return 0, nil, errors.New("error could not verify signature returned with channel lease")
	}

	return resp.Lease, resp.UDLeaseEd25519Signature, err
}
