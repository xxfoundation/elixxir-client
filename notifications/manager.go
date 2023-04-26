package notifications

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	kvSync "gitlab.com/elixxir/client/v4/sync"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"golang.org/x/crypto/blake2b"
	"sync"
	"time"
)

const (
	prefixConst = "notificationsManager:%x"
)

type registration struct {
	Following *id.ID
	Group     string
	Metadata  []byte
	Status    bool
}

type manager struct {
	notifications map[*id.ID]*registration
	group         map[string][]*registration

	transmissionRSA                             rsa.PrivateKey
	transmissionRegistrationValidationSignature []byte
	registrationTimestampNs                     int64
	registrationSalt                            []byte

	comms *client.Comms
	rng   *fastRNG.StreamGenerator

	notificationHost *connect.Host

	mux sync.Mutex

	kvLocal  *kvSync.VersionedKV
	kvRemote *kvSync.VersionedKV
}

func NewOrLoadManager(identity xxdk.TransmissionIdentity, regSig []byte, kv *kvSync.VersionedKV,
	comms *client.Comms, rng *fastRNG.StreamGenerator) Manger {

	nd, exists := comms.GetHost(&id.NotificationBot)
	if !exists {
		jww.FATAL.Panicf("Notification bot not registered, " +
			"notifications cannot be startedL")
	}

	kvLocal, err := kv.Prefix(prefix(identity.RSAPrivate.Public()))
	if err != nil {
		jww.FATAL.Panicf("Notifications failed to prefix kv")
	}

	kvRemote, err := kvLocal.Prefix("remote")
	if err != nil {
		jww.FATAL.Panicf("Notifications failed to prefix kv")
	}

	m := &manager{
		transmissionRSA: identity.RSAPrivate,
		transmissionRegistrationValidationSignature: regSig,
		registrationTimestampNs:                     identity.RegistrationTimestamp,
		registrationSalt:                            identity.Salt,
		comms:                                       comms,
		rng:                                         rng,
		notificationHost:                            nd,
		kvLocal:                                     kvLocal,
	}

}

func prefix(pubkey rsa.PublicKey) string {
	h, _ := blake2b.New256(nil)
	h.Write(pubkey.MarshalPem())
	return fmt.Sprintf(prefixConst, h.Sum(nil))
}

func (m *manager) getIidAndSig(toBeNotified *id.ID, timestamp time.Time, operation string) (
	intermediaryReceptionID, sig []byte, err error) {
	intermediaryReceptionID, err = ephemeral.GetIntermediaryId(toBeNotified)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to form intermediary ID")
	}
	h, err := hash.NewCMixHash()
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to create cMix hash")
	}
	_, err = h.Write(intermediaryReceptionID)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to write intermediary ID to hash")
	}
	stream := m.rng.GetStream()
	defer stream.Close()
	sig, err = m.transmissionRSA.SignPSS(stream, hash.CMixHash, h.Sum(nil), nil)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to sign intermediary ID")
	}
	return intermediaryReceptionID, sig, nil
}
