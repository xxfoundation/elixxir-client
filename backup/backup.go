////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/fastRNG"
	"sync"
)

// Error messages.
const (
	// initializeBackup
	errSavePassword      = "failed to save password: %+v"
	errSaveKeySaltParams = "failed to save key, salt, and params: %+v"

	// resumeBackup
	errLoadPassword = "backup not initialized: load user password failed: %+v"

	// Backup.StopBackup
	errDeletePassword = "failed to delete password: %+v"
	errDeleteCrypto   = "failed to delete key, salt, and parameters: %+v"
)

// Backup stores the user's key and backup callback used to encrypt and transmit
// the backup data.
type Backup struct {
	// Callback that is called with the encrypted backup when triggered
	cb UpdateBackup

	mux sync.RWMutex

	// Client structures
	client          *api.Client
	store           *storage.Session
	backupContainer *interfaces.BackupContainer
	rng             *fastRNG.StreamGenerator
}

// UpdateBackup is the callback that encrypted backup data is returned on
type UpdateBackup func(encryptedBackup []byte)

// InitializeBackup creates a new Backup object with the callback to return
// backups when triggered. On initialization, 32-bit key is derived from the
// user's password via Argon2 and a 16-bit salt is generated. Both are saved to
// storage along with the parameters used in Argon2 to be used when encrypting
// new backups.
// Call this to turn on backups for the first time or to replace the user's
// password.
func InitializeBackup(password string, cb UpdateBackup, c *api.Client) (*Backup, error) {
	return initializeBackup(
		password, cb, c, c.GetStorage(), c.GetBackup(), c.GetRng())
}

// initializeBackup is a helper function that takes in all the fields for Backup
// as parameters for easier testing.
func initializeBackup(password string, cb UpdateBackup, c *api.Client,
	store *storage.Session, backupContainer *interfaces.BackupContainer,
	rng *fastRNG.StreamGenerator) (*Backup, error) {
	b := &Backup{
		cb:              cb,
		client:          c,
		store:           store,
		backupContainer: backupContainer,
		rng:             rng,
	}

	// Save password to storage
	err := savePassword(password, b.store.GetKV())
	if err != nil {
		return nil, errors.Errorf(errSavePassword, err)
	}

	// Derive key and get generated salt and parameters
	key, salt, p, err := b.getKeySaltParams(password)
	if err != nil {
		return nil, err
	}

	// Save key, salt, and parameters to storage
	err = saveBackup(key, salt, p, b.store.GetKV())
	if err != nil {
		return nil, errors.Errorf(errSaveKeySaltParams, err)
	}

	// Setting backup trigger in client
	b.backupContainer.SetBackup(b.TriggerBackup)

	jww.INFO.Print("Initialized backup with new user key.")

	return b, nil
}

// ResumeBackup resumes a backup by restoring the Backup object and registering
// a new callback. Call this to resume backups that have already been
// initialized. Returns an error if backups have not already been initialized.
func ResumeBackup(cb UpdateBackup, c *api.Client) (*Backup, error) {
	return resumeBackup(cb, c, c.GetStorage(), c.GetBackup(), c.GetRng())
}

// resumeBackup is a helper function that takes in all the fields for Backup as
// parameters for easier testing.
func resumeBackup(cb UpdateBackup, c *api.Client, store *storage.Session,
	backupContainer *interfaces.BackupContainer, rng *fastRNG.StreamGenerator) (
	*Backup, error) {
	_, err := loadPassword(store.GetKV())
	if err != nil {
		return nil, errors.Errorf(errLoadPassword, err)
	}

	b := &Backup{
		cb:              cb,
		client:          c,
		store:           store,
		backupContainer: backupContainer,
		rng:             rng,
	}

	// Setting backup trigger in client
	b.backupContainer.SetBackup(b.TriggerBackup)

	jww.INFO.Print("Resumed backup with password loaded from storage.")

	return b, nil
}

// getKeySaltParams derives a key from the user's password, a generated salt,
// and the default parameters and return all three.
func (b *Backup) getKeySaltParams(password string) (
	key, salt []byte, p backup.Params, err error) {
	rand := b.rng.GetStream()
	salt, err = backup.MakeSalt(rand)
	if err != nil {
		return
	}
	rand.Close()

	p = backup.DefaultParams()
	key = backup.DeriveKey(password, salt, p)

	return
}

// TriggerBackup collates the backup and calls it on the registered backup
// callback. Does nothing if no encryption key or backup callback is registered.
// The passed in reason will be printed to the log when the backup is sent. It
// should be in the paste tense. For example, if a contact is deleted, the
// reason can be "contact deleted" and the log will show:
//	Triggering backup: contact deleted
func (b *Backup) TriggerBackup(reason string) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	key, salt, p, err := loadBackup(b.store.GetKV())
	if err != nil {
		jww.ERROR.Printf("Backup Failed: could not load key, salt, and "+
			"parameters for encrypting backup from storage: %+v", err)
		return
	}

	// Grab backup data
	collatedBackup := b.assembleBackup()

	// Encrypt backup data with user key
	rand := b.rng.GetStream()
	encryptedBackup, err := collatedBackup.Encrypt(rand, key, salt, p)
	if err != nil {
		jww.FATAL.Panicf("Failed to encrypt backup: %+v", err)
	}
	rand.Close()

	jww.INFO.Printf("Backup triggered: %s", reason)

	// Send backup on callback
	go b.cb(encryptedBackup)
}

// StopBackup stops the backup processes and deletes the user's password, key,
// salt, and parameters from storage.
func (b *Backup) StopBackup() error {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.cb = nil

	err := deletePassword(b.store.GetKV())
	if err != nil {
		return errors.Errorf(errDeletePassword, err)
	}

	err = deleteBackup(b.store.GetKV())
	if err != nil {
		return errors.Errorf(errDeleteCrypto, err)
	}

	jww.INFO.Print("Stopped backups.")

	return nil
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

	// Get user and storage user
	u := b.store.GetUser()
	su := b.store.User()

	// Get registration timestamp
	bu.RegistrationTimestamp = u.RegistrationTimestamp

	// Get registration code; ignore the error because if there is no
	// registration, then an empty string is returned
	bu.RegistrationCode, _ = b.store.GetRegCode()

	// Get transmission identity
	bu.TransmissionIdentity = backup.TransmissionIdentity{
		RSASigningPrivateKey: u.TransmissionRSA,
		RegistrarSignature:   su.GetTransmissionRegistrationValidationSignature(),
		Salt:                 u.TransmissionSalt,
		ComputedID:           u.TransmissionID,
	}

	// Get reception identity
	bu.ReceptionIdentity = backup.ReceptionIdentity{
		RSASigningPrivateKey: u.ReceptionRSA,
		RegistrarSignature:   su.GetReceptionRegistrationValidationSignature(),
		Salt:                 u.ReceptionSalt,
		ComputedID:           u.ReceptionID,
		DHPrivateKey:         u.E2eDhPrivateKey,
		DHPublicKey:          u.E2eDhPublicKey,
	}

	// Get facts
	bu.UserDiscoveryRegistration.FactList = b.store.GetUd().GetFacts()

	// Get contacts
	bu.Contacts.Identities = b.store.E2e().GetPartners()

	return bu
}
