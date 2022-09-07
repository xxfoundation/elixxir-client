package ud

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/channels"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
)

const (
	registrationDiskKey     = "registrationDiskKey"
	registrationDiskVersion = 0
	graceDuration           = time.Hour
)

var startChannelNameServiceOnce sync.Once
var ErrChannelLeaseSignature = errors.New("failure to validate lease signature")

// loadRegistrationDisk loads a registrationDisk from the kv
// and returns the registrationDisk.
func loadRegistrationDisk(kv *versioned.KV) (registrationDisk, error) {
	obj, err := kv.Get(registrationDiskKey, registrationDiskVersion)
	if err != nil {
		return registrationDisk{}, err
	}
	return UnmarshallRegistrationDisk(obj.Data)
}

// saveRegistrationDisk saves the given saveRegistrationDisk to
// the given kv.
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
	return kv.Set(registrationDiskKey, &obj)
}

// registrationDisk is used to encapsulate the channel user's key pair,
// lease and lease signature.
type registrationDisk struct {
	rwmutex sync.RWMutex

	Registered bool
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	Lease      int64
	Signature  []byte
}

// newRegistrationDisk creates a new newRegistrationDisk.
func newRegistrationDisk(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey,
	lease time.Time, signature []byte) registrationDisk {
	return registrationDisk{
		Lease:      lease.UnixNano(),
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Signature:  signature,
	}
}

func (r registrationDisk) IsRegistered() bool {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return r.Registered
}

// Update updates the registrationDisk that is currently
// stored on the kv with a new lease and lease signature.
func (r registrationDisk) Update(lease int64, signature []byte) {
	r.rwmutex.Lock()
	defer r.rwmutex.Unlock()

	r.Registered = true
	r.Lease = lease
	r.Signature = signature
}

// Marshall marshalls the registrationDisk.
func (r registrationDisk) Marshall() ([]byte, error) {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return json.Marshal(&r)
}

// UnmarshallRegistrationDisk unmarshalls a registrationDisk
func UnmarshallRegistrationDisk(data []byte) (registrationDisk, error) {
	var r registrationDisk
	err := json.Unmarshal(data, &r)
	if err != nil {
		return registrationDisk{}, err
	}
	return r, nil
}

// GetLease returns the current registrationDisk lease.
func (r registrationDisk) GetLease() time.Time {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return time.Unix(0, r.Lease)
}

// GetPublicKey returns the current public key.
func (r registrationDisk) GetPublicKey() ed25519.PublicKey {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	pubkey := make([]byte, ed25519.PublicKeySize)
	copy(pubkey, r.PublicKey)
	return pubkey
}

// GetPrivateKey returns the current private key.
func (r registrationDisk) getPrivateKey() ed25519.PrivateKey {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return r.PrivateKey
}

// GetLeaseSignature returns the currentl signature and lease time.
func (r registrationDisk) GetLeaseSignature() ([]byte, time.Time) {
	r.rwmutex.RLock()
	defer r.rwmutex.RUnlock()

	return r.Signature, time.Unix(0, r.Lease)
}

// clientIDTracker encapsulates the client channel lease and the
// repetitive scheduling of new lease registrations when the
// current lease expires.
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

// clientIDTracker implements the NameService interface.
var _ channels.NameService = (*clientIDTracker)(nil)

// newclientIDTracker creates a new clientIDTracker.
func newclientIDTracker(comms channelLeaseComms, host *connect.Host, username string, kv *versioned.KV,
	receptionIdentity xxdk.ReceptionIdentity, udPubKey ed25519.PublicKey, rngSource *fastRNG.StreamGenerator) *clientIDTracker {

	reg, err := loadRegistrationDisk(kv)
	if !kv.Exists(err) {
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
	} else if err != nil {
		jww.FATAL.Panic(err)
	}

	c := &clientIDTracker{
		kv:                kv,
		rngSource:         rngSource,
		registrationDisk:  &reg,
		receptionIdentity: &receptionIdentity,
		username:          username,
		comms:             comms,
		host:              host,
		udPubKey:          udPubKey,
	}

	if !reg.IsRegistered() {
		err = c.register()
		if err != nil {
			jww.FATAL.Panic(err)
		}
	}

	return c
}

// Start starts the registration worker.
func (c *clientIDTracker) Start() (stoppable.Stoppable, error) {
	stopper := stoppable.NewSingle("ud.ClientIDTracker")
	go c.registrationWorker(stopper)
	return stopper, nil
}

func pow(base, exponent int) int {
	if exponent == 0 {
		return 1
	}
	result := base
	for i := 2; i <= exponent; i++ {
		result *= base
	}
	return result
}

// registrationWorker is meant to run in it's own goroutine
// periodically registering, getting a new lease.
func (c *clientIDTracker) registrationWorker(stopper *stoppable.Single) {
	// start backoff at 32 seconds
	base := 2
	exponent := 5
	waitTime := time.Second
	maxBackoff := 300
	for {
		if time.Now().After(c.registrationDisk.GetLease().Add(-graceDuration)) {
			err := c.register()
			if err != nil {
				backoffSeconds := pow(base, exponent)
				if backoffSeconds > maxBackoff {
					backoffSeconds = maxBackoff
				} else {
					exponent += 1
				}
				waitTime = time.Second * time.Duration(backoffSeconds)
			} else {
				waitTime = time.Second
			}
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
		case <-time.After(waitTime):
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

// SignChannelMessage returns the signature of the given
// message. The ed25519 private key stored in the registrationDisk on the
// kv is used for signing.
func (c *clientIDTracker) SignChannelMessage(message []byte) ([]byte, error) {
	privateKey := c.registrationDisk.getPrivateKey()
	return ed25519.Sign(privateKey, message), nil
}

// ValidateoChannelMessage
func (c *clientIDTracker) ValidateChannelMessage(username string, lease time.Time, pubKey ed25519.PublicKey, authorIDSignature []byte) bool {
	return channel.VerifyChannelLease(authorIDSignature, pubKey, username, lease, c.udPubKey)
}

// register causes a request for a new channel lease to be sent to
// the user discovery server. If successful in procuration of a new lease
// then it is written to the registrationDisk on the kv.
func (c *clientIDTracker) register() error {
	lease, signature, err := c.requestChannelLease()
	if err != nil {
		return err
	}

	c.registrationDisk.Update(lease, signature)

	return nil
}

// requestChannelLease requests a new channel lease
// from the user discovery server.
func (c *clientIDTracker) requestChannelLease() (int64, []byte, error) {
	ts := time.Now().UnixNano()
	privKey, err := c.receptionIdentity.GetRSAPrivateKey()
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

// StartChannelNameService creates a new clientIDTracker
// and returns a reference to it's type as the NameService interface.
// However it's scheduler thread isn't started until it's Start
// method is called.
func (m *Manager) StartChannelNameService() channels.NameService {
	udPubKeyBytes := m.user.GetCmix().GetInstance().GetPartialNdf().Get().UDB.DhPubKey
	var service channels.NameService
	username, err := m.store.GetUsername()
	if err != nil {
		jww.FATAL.Panic(err)
	}
	startChannelNameServiceOnce.Do(func() {
		service = newclientIDTracker(
			m.comms,
			m.ud.host,
			username,
			m.getKv(),
			m.user.GetReceptionIdentity(),
			ed25519.PublicKey(udPubKeyBytes),
			m.getRng())
	})
	return service
}
