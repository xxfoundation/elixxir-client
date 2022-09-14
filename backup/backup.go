////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"gitlab.com/elixxir/client/xxdk"
	"sync"
	"time"

	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

// Error messages.
const (
	// InitializeBackup
	errSavePassword      = "failed to save password: %+v"
	errSaveKeySaltParams = "failed to save key, salt, and params: %+v"

	// Backup.StopBackup
	errDeletePassword = "failed to delete password: %+v"
	errDeleteCrypto   = "failed to delete key, salt, and parameters: %+v"
)

// Backup stores the user's key and backup callback used to encrypt and transmit
// the backup data.
type Backup struct {
	// Callback that is called with the encrypted backup when triggered
	updateBackupCb UpdateBackupFn

	container *xxdk.Container

	jsonParams string

	// E2e structures
	e2e     E2e
	session Session
	ud      UserDiscovery
	kv      *versioned.KV
	rng     *fastRNG.StreamGenerator

	mux sync.RWMutex
}

// E2e is a subset of functions from the interface e2e.Handler.
type E2e interface {
	GetAllPartnerIDs() []*id.ID
	GetHistoricalDHPubkey() *cyclic.Int
	GetHistoricalDHPrivkey() *cyclic.Int
}

// Session is a subset of functions from the interface storage.Session.
type Session interface {
	GetRegCode() (string, error)
	GetTransmissionID() *id.ID
	GetTransmissionSalt() []byte
	GetReceptionID() *id.ID
	GetReceptionSalt() []byte
	GetReceptionRSA() *rsa.PrivateKey
	GetTransmissionRSA() *rsa.PrivateKey
	GetTransmissionRegistrationValidationSignature() []byte
	GetReceptionRegistrationValidationSignature() []byte
	GetRegistrationTimestamp() time.Time
}

type UserDiscovery interface {
	GetFacts() fact.FactList
}

// UpdateBackupFn is the callback that encrypted backup data is returned on
type UpdateBackupFn func(encryptedBackup []byte)

// InitializeBackup creates a new Backup object with the callback to return
// backups when triggered. On initialization, 32-bit key is derived from the
// user's password via Argon2 and a 16-bit salt is generated. Both are saved to
// storage along with the parameters used in Argon2 to be used when encrypting
// new backups.
// Call this to turn on backups for the first time or to replace the user's
// password.
func InitializeBackup(backupPassphrase string, updateBackupCb UpdateBackupFn,
	container *xxdk.Container, e2e E2e, session Session, ud UserDiscovery,
	kv *versioned.KV, rng *fastRNG.StreamGenerator) (*Backup, error) {
	b := &Backup{
		updateBackupCb: updateBackupCb,
		container:      container,
		e2e:            e2e,
		session:        session,
		ud:             ud,
		kv:             kv,
		rng:            rng,
	}

	// Derive key and get generated salt and parameters
	rand := b.rng.GetStream()
	salt, err := backup.MakeSalt(rand)
	if err != nil {
		return nil, err
	}
	rand.Close()

	params := backup.DefaultParams()
	params.Memory = 64 * 1024 // 64 MiB
	params.Threads = 1
	params.Time = 5
	key := backup.DeriveKey(backupPassphrase, salt, params)

	// Save key, salt, and parameters to storage
	err = saveBackup(key, salt, params, b.kv)
	if err != nil {
		return nil, errors.Errorf(errSaveKeySaltParams, err)
	}

	// Setting backup trigger in client
	b.container.SetBackup(b.TriggerBackup)

	b.TriggerBackup("InitializeBackup")
	jww.INFO.Print("Initialized backup with new user key.")

	return b, nil
}

// ResumeBackup resumes a backup by restoring the Backup object and registering
// a new callback. Call this to resume backups that have already been
// initialized. Returns an error if backups have not already been initialized.
func ResumeBackup(updateBackupCb UpdateBackupFn, container *xxdk.Container,
	e2e E2e, session Session, ud UserDiscovery, kv *versioned.KV,
	rng *fastRNG.StreamGenerator) (*Backup, error) {
	_, _, _, err := loadBackup(kv)
	if err != nil {
		return nil, err
	}

	b := &Backup{
		updateBackupCb: updateBackupCb,
		container:      container,
		jsonParams:     loadJson(kv),
		e2e:            e2e,
		session:        session,
		ud:             ud,
		kv:             kv,
		rng:            rng,
	}

	// Setting backup trigger in client
	b.container.SetBackup(b.TriggerBackup)

	jww.INFO.Print("Resumed backup with password loaded from storage.")

	return b, nil
}

// getKeySaltParams derives a key from the user's password, a generated salt,
// and the default parameters and return all three.
func (b *Backup) getKeySaltParams(password string) (
	key, salt []byte, params backup.Params, err error) {
	rand := b.rng.GetStream()
	salt, err = backup.MakeSalt(rand)
	if err != nil {
		return
	}
	rand.Close()

	params = backup.DefaultParams()
	key = backup.DeriveKey(password, salt, params)

	return
}

// TriggerBackup assembles the backup and calls it on the registered backup
// callback. Does nothing if no encryption key or backup callback is registered.
// The passed in reason will be printed to the log when the backup is sent. It
// should be in the past tense. For example, if a contact is deleted, the
// reason can be "contact deleted" and the log will show:
//	Triggering backup: contact deleted
func (b *Backup) TriggerBackup(reason string) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	if b == nil || b.kv == nil {
		jww.ERROR.Printf("TriggerBackup called on unitialized object")
		return
	}

	key, salt, params, err := loadBackup(b.kv)
	if err != nil {
		jww.ERROR.Printf("Backup Failed: could not load key, salt, and "+
			"parameters for encrypting backup from storage: %+v", err)
		return
	}

	// Grab backup data
	collatedBackup := b.assembleBackup()

	// Encrypt backup data with user key
	rand := b.rng.GetStream()
	encryptedBackup, err := collatedBackup.Encrypt(rand, key, salt, params)
	if err != nil {
		jww.FATAL.Panicf("Failed to encrypt backup: %+v", err)
	}
	rand.Close()

	jww.INFO.Printf("Backup triggered: %s", reason)

	// Send backup on callback
	b.mux.RLock()
	defer b.mux.RUnlock()
	if b.updateBackupCb != nil {
		go b.updateBackupCb(encryptedBackup)
	} else {
		jww.WARN.Printf("could not call backup callback, stopped...")
	}
}

func (b *Backup) AddJson(newJson string) {
	b.mux.Lock()
	defer b.mux.Unlock()

	if newJson != b.jsonParams {
		b.jsonParams = newJson
		if err := storeJson(newJson, b.kv); err != nil {
			jww.FATAL.Panicf("Failed to store json: %+v", err)
		}
		go b.TriggerBackup("New Json")
	}
}

// StopBackup stops the backup processes and deletes the user's password, key,
// salt, and parameters from storage.
func (b *Backup) StopBackup() error {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.updateBackupCb = nil

	err := deleteBackup(b.kv)
	if err != nil {
		return errors.Errorf(errDeleteCrypto, err)
	}

	jww.INFO.Print("Stopped backups.")

	return nil
}

// IsBackupRunning returns true if the backup has been initialized and is
// running. Returns false if it has been stopped.
func (b *Backup) IsBackupRunning() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.updateBackupCb != nil
}

// assembleBackup gathers all the contents of the backup and stores them in a
// backup.Backup. This backup contains:
//  1. Cryptographic information for the transmission identity
//  2. Cryptographic information for the reception identity
//  3. User's UD facts (username, email, phone number)
//  4. Contact list
func (b *Backup) assembleBackup() backup.Backup {
	bu := backup.Backup{
		TransmissionIdentity:      backup.TransmissionIdentity{},
		ReceptionIdentity:         backup.ReceptionIdentity{},
		UserDiscoveryRegistration: backup.UserDiscoveryRegistration{},
		Contacts:                  backup.Contacts{},
	}

	// Get registration timestamp
	bu.RegistrationTimestamp = b.session.GetRegistrationTimestamp().UnixNano()

	// Get registration code; ignore the error because if there is no
	// registration, then an empty string is returned
	bu.RegistrationCode, _ = b.session.GetRegCode()

	// Get transmission identity
	bu.TransmissionIdentity = backup.TransmissionIdentity{
		RSASigningPrivateKey: b.session.GetTransmissionRSA(),
		RegistrarSignature:   b.session.GetTransmissionRegistrationValidationSignature(),
		Salt:                 b.session.GetTransmissionSalt(),
		ComputedID:           b.session.GetTransmissionID(),
	}

	// Get reception identity
	bu.ReceptionIdentity = backup.ReceptionIdentity{
		RSASigningPrivateKey: b.session.GetReceptionRSA(),
		RegistrarSignature:   b.session.GetReceptionRegistrationValidationSignature(),
		Salt:                 b.session.GetReceptionSalt(),
		ComputedID:           b.session.GetReceptionID(),
		DHPrivateKey:         b.e2e.GetHistoricalDHPrivkey(),
		DHPublicKey:          b.e2e.GetHistoricalDHPubkey(),
	}

	// Get facts
	if b.ud != nil {
		bu.UserDiscoveryRegistration.FactList = b.ud.GetFacts()
	} else {
		bu.UserDiscoveryRegistration.FactList = fact.FactList{}
	}

	// Get contacts
	bu.Contacts.Identities = b.e2e.GetAllPartnerIDs()

	// Add the memoized json params
	bu.JSONParams = b.jsonParams

	return bu
}
