package ud

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
)

const (
	registrationDiskKey                   = "registrationDiskKey"
	registrationDiskVersion               = 0
	graceDuration           time.Duration = time.Hour
)

var startChannelNameServiceOnce sync.Once
var ErrChannelLeaseSignature = errors.New("failure to validate lease signature")

// NameService is an interface which encapsulates
// the user identity channel tracking service.
type NameService interface {

	// GetUsername returns the username.
	GetUsername() string

	// GetChannelValidationSignature returns the validation
	// signature and the time it was signed.
	GetChannelValidationSignature() (signature []byte, lease time.Time)

	// GetChannelPubkey returns the user's public key.
	GetChannelPubkey() ed25519.PublicKey

	// SignChannelMessage returns the signature of the
	// given message.
	SignChannelMessage(message []byte) (signature []byte, err error)

	// ValidateChannelMessage
	ValidateChannelMessage(username string, lease time.Time, pubKey ed25519.PublicKey, authorIDSignature ed25519.PublicKey) bool
}

func loadRegistrationDisk(kv *versioned.KV) (registrationDisk, error) {
	obj, err := kv.Get(registrationDiskKey, registrationDiskVersion)
	if err != nil {
		return registrationDisk{}, err
	}
	return UnmarshallRegistrationDisk(obj.Data)
}

func saveRegistrationDisk(kv *versioned.KV, reg registrationDisk) error {
	regBytes, err := reg.Marshall()
	if err != nil {
		return err
	}
	obj := versioned.Object{
		Version:   registrationDiskVersion,
		Timestamp: time.Now(),
		Data:      regBytes,
	}
	kv.Set(registrationDiskKey, registrationDiskVersion, &obj)
	return nil
}

type registrationDisk struct {
	rwmutex sync.RWMutex

	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	Lease      int64
	Signature  []byte
}

func newRegistrationDisk(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey,
	lease time.Time, signature []byte) registrationDisk {
	return registrationDisk{
		Lease:      lease.Unix(),
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Signature:  signature,
	}
}

func (r registrationDisk) Update(lease int64, signature []byte) {
	r.rwmutex.Lock()
	defer r.rwmutex.Unlock()

	r.Lease = lease
	r.Signature = signature
}

func (r registrationDisk) Marshall() ([]byte, error) {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return json.Marshal(&r)
}

func UnmarshallRegistrationDisk(data []byte) (registrationDisk, error) {
	var r registrationDisk
	err := json.Unmarshal(data, &r)
	if err != nil {
		return registrationDisk{}, err
	}
	return r, nil
}

func (r registrationDisk) GetLease() time.Time {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return time.Unix(0, r.Lease)
}

func (r registrationDisk) GetPublicKey() ed25519.PublicKey {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return r.PublicKey
}

func (r registrationDisk) GetLeaseSignature() ([]byte, time.Time) {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return r.Signature, time.Unix(0, r.Lease)
}

type clientIDTracker struct {
	kv *versioned.KV

	username string

	registrationDisk  *registrationDisk
	receptionIdentity *xxdk.ReceptionIdentity

	rngSource *fastRNG.StreamGenerator

	host     *connect.Host
	comms    channelLeaseComms
	udPubKey ed25519.PublicKey
}

var _ NameService = (*clientIDTracker)(nil)

func newclientIDTracker(comms channelLeaseComms, host *connect.Host, username string, kv *versioned.KV,
	receptionIdentity xxdk.ReceptionIdentity, udPubKey ed25519.PublicKey, rngSource *fastRNG.StreamGenerator) *clientIDTracker {

	var err error

	reg, err := loadRegistrationDisk(kv)
	if err != nil {
		rng := rngSource.GetStream()
		defer rng.Close()

		publicKey, privateKey, err := ed25519.GenerateKey(rng)
		if err != nil {
			jww.FATAL.Panic(err)
		}

		reg = registrationDisk{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
			Lease:      0,
		}
		err = saveRegistrationDisk(kv, reg)
		if err != nil {
			jww.FATAL.Panic(err)
		}
	}

	return &clientIDTracker{
		registrationDisk:  &reg,
		receptionIdentity: &receptionIdentity,
		username:          username,
		comms:             comms,
		host:              host,
		udPubKey:          udPubKey,
	}
}

// Start starts the registration worker.
func (c *clientIDTracker) Start() (stoppable.Stoppable, error) {
	stopper := stoppable.NewSingle("ud.ClientIDTracker")
	go c.registrationWorker(stopper)
	return stopper, nil
}

func (c *clientIDTracker) registrationWorker(stopper *stoppable.Single) {

	for {
		if time.Now().After(c.registrationDisk.GetLease().Add(-graceDuration)) {
			c.register()
		}

		select {
		case <-stopper.Quit():
			return
		case <-time.After(c.registrationDisk.GetLease().Add(-graceDuration).Sub(time.Now())):
		}

		// Avoid spamming the server in the event that it's service is down.
		select {
		case <-stopper.Quit():
			return
		case <-time.After(time.Second):
		}

	}
}

// GetUsername returns the username.
func (c *clientIDTracker) GetUsername() string {
	return c.username
}

// GetChannelValidationSignature returns the validation
// signature and the time it was signed.
func (c *clientIDTracker) GetChannelValidationSignature() ([]byte, time.Time) {
	return c.registrationDisk.GetLeaseSignature()
}

// GetChannelPubkey returns the user's public key.
func (c *clientIDTracker) GetChannelPubkey() ed25519.PublicKey {
	return c.registrationDisk.GetPublicKey()
}

// SignChannelMessage returns the signature of the
// given message.
func (c *clientIDTracker) SignChannelMessage(message []byte) ([]byte, error) {
	return nil, nil // XXX FIXME
}

// ValidateoChannelMessage
func (c *clientIDTracker) ValidateChannelMessage(username string, lease time.Time, pubKey ed25519.PublicKey, authorIDSignature ed25519.PublicKey) bool {
	// XXX FIXME

	return false
}

func (c *clientIDTracker) register() error {

	lease, signature, err := c.requestChannelLease()
	if err != nil {
		return err
	}

	c.registrationDisk.Update(lease, signature)

	return nil
}

func (c *clientIDTracker) requestChannelLease() (int64, []byte, error) {

	ts := time.Now().UnixNano()
	privKey, err := c.receptionIdentity.GetRSAPrivatePem()
	if err != nil {
		return 0, nil, err
	}
	rng := c.rngSource.GetStream()
	userPubKey := c.registrationDisk.GetPublicKey()
	fSig, err := channel.SignChannelIdentityRequest(userPubKey, time.Unix(0, ts), privKey, rng)
	if err != nil {
		return 0, nil, err
	}
	rng.Close()

	msg := &mixmessages.ChannelLeaseRequest{
		UserID:                 c.receptionIdentity.ID.Marshal(),
		UserEd25519PubKey:      userPubKey,
		Timestamp:              ts,
		UserPubKeyRSASignature: fSig,
	}

	resp, err := c.comms.SendChannelLeaseRequest(c.host, msg)
	if err != nil {
		return 0, nil, err
	}

	ok := channel.VerifyChannelLease(resp.UDLeaseEd25519Signature,
		userPubKey, c.username, time.Unix(0, resp.Lease), c.udPubKey)
	if !ok {
		return 0, nil, ErrChannelLeaseSignature
	}

	return resp.Lease, resp.UDLeaseEd25519Signature, err
}

func (m *Manager) StartChannelNameService() NameService {
	udPubKeyBytes := m.user.GetCmix().GetInstance().GetPartialNdf().Get().UDB.DhPubKey
	var service NameService
	startChannelNameServiceOnce.Do(func() {
		service = newclientIDTracker(
			m.comms,
			m.ud.host,
			m.user.GetUsername(),
			m.getKv(),
			m.user.GetReceptionIdentity(),
			ed25519.PublicKey(udPubKeyBytes),
			m.getRng())
	})
	return service
}
