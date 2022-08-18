package ud

import (
	"crypto/ed25519"
	"encoding/json"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/fastRNG"
)

const (
	registrationDiskKey     = "registrationDiskKey"
	registrationDiskVersion = 0
)

// NameService is an interface which encapsulates
// the user identity channel tracking service.
type NameService interface {

	// GetUsername returns the username.
	GetUsername() string

	// GetChannelValidationSignature returns the validation
	// signature and the time it was signed.
	GetChannelValidationSignature() ([]byte, time.Time)

	// GetChannelPubkey returns the user's public key.
	GetChannelPubkey() ed25519.PublicKey

	// SignChannelMessage returns the signature of the
	// given message.
	SignChannelMessage(message []byte) ([]byte, error)

	// ValidateChannelMessage
	ValidateChannelMessage(message []byte, lease time.Time, pubKey ed25519.PublicKey, signature []byte)

	// Stop stops the NameService.
	Stop()
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
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	Lease      int64
}

func newRegistrationDisk(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey,
	lease time.Time) registrationDisk {
	return registrationDisk{
		Lease:      lease.Unix(),
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}

func (r registrationDisk) Marshall() ([]byte, error) {
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
	return time.Unix(0, r.Lease)
}

type clientIDTracker struct {
	wg        sync.WaitGroup
	haltCh    chan interface{}
	closeOnce sync.Once

	kv *versioned.KV

	registrationDuration time.Duration
	username             string

	registrationDisk  *registrationDisk
	receptionIdentity *xxdk.ReceptionIdentity

	pubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey

	rngSource *fastRNG.StreamGenerator
}

var _ NameService = (*clientIDTracker)(nil)

func newclientIDTracker(comms channelLeaseComms, username string, kv *versioned.KV,
	receptionIdentity xxdk.ReceptionIdentity, rngSource *fastRNG.StreamGenerator,
	registrationDuration time.Duration) *clientIDTracker {

	var err error
	var reg registrationDisk

	reg, err = loadRegistrationDisk(kv)
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
		haltCh:               make(chan interface{}),
		registrationDuration: registrationDuration,
		receptionIdentity:    &receptionIdentity,
		username:             username,
	}

}

// Start starts the registration worker.
func (c *clientIDTracker) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.registrationWorker()
	}()
}

// Stop stops the registration worker.
func (c *clientIDTracker) Stop() {
	c.closeOnce.Do(func() {
		close(c.haltCh)
	})
	c.wg.Wait()
}

func (c *clientIDTracker) registrationWorker() {
	for {
		select {
		case <-c.haltCh:
			return
		case <-time.After(c.registrationDuration):
			c.register()
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
	return nil, time.Time{} // XXX FIXME
}

// GetChannelPubkey returns the user's public key.
func (c *clientIDTracker) GetChannelPubkey() ed25519.PublicKey {
	return c.pubKey
}

// SignChannelMessage returns the signature of the
// given message.
func (c *clientIDTracker) SignChannelMessage(message []byte) ([]byte, error) {
	return nil, nil // XXX FIXME
}

// ValidateChannelMessage
func (c *clientIDTracker) ValidateChannelMessage(message []byte, lease time.Time, pubKey ed25519.PublicKey, signature []byte) {
	// XXX FIXME
}

func (c *clientIDTracker) register() {
	// XXX FIXME
}
