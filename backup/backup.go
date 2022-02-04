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
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
	"sync"
)

// Error messages.
const (
	// initializeBackup
	errSaveKey = "failed to save user's backup key to storage: %+v"

	// resumeBackup
	errLoadKey = "cannot resume backup without user key: %+v"
)

// Backup stores the user's key and backup callback used to encrypt and transmit
// the backup data.
type Backup struct {

	// User provided key used to encrypt the backup
	key []byte
	// Callback that is called with the encrypted backup
	cb GetBackup

	mux sync.RWMutex

	// Client structures
	client *api.Client
	store  *storage.Session
	rng    *fastRNG.StreamGenerator
}

// GetBackup is the callback that encrypted backup data is returned on
type GetBackup func(encryptedBackup []byte)

// InitializeBackup creates a new Backup object with the user's key and
// initializes the callback to return backups when triggered. Call this to turn
// on backups for the first time or to replace the user's key.
func InitializeBackup(key []byte, cb GetBackup, c *api.Client) (*Backup, error) {
	return initializeBackup(key, cb, c, c.GetStorage(), c.GetRng())
}

// initializeBackup is a helper function that takes in all the fields for Backup
// as parameters for easier testing.
func initializeBackup(key []byte, cb GetBackup, c *api.Client,
	store *storage.Session, rng *fastRNG.StreamGenerator) (*Backup, error) {
	b := &Backup{
		key:    make([]byte, len(key)),
		cb:     cb,
		client: c,
		store:  store,
		rng:    rng,
	}

	// Copy key
	copy(b.key, key)

	// Save key to storage
	err := storeKey(b.key, b.store.GetKV())
	if err != nil {
		return nil, errors.Errorf(errSaveKey, err)
	}

	jww.INFO.Print("Initializing backup with new user key.")

	return b, nil
}

// ResumeBackup resumes a backup by restoring the Backup object and registering
// a new callback. Call this to resume backups that have already been
// initialized. Returns an error if backups have not already been initialized.
func ResumeBackup(cb GetBackup, c *api.Client) (*Backup, error) {
	return resumeBackup(cb, c, c.GetStorage(), c.GetRng())
}

// resumeBackup is a helper function that takes in all the fields for Backup as
// parameters for easier testing.
func resumeBackup(cb GetBackup, c *api.Client, store *storage.Session,
	rng *fastRNG.StreamGenerator) (*Backup, error) {
	key, err := loadKey(store.GetKV())
	if err != nil {
		return nil, errors.Errorf(errLoadKey, err)
	}

	b := &Backup{
		key:    key,
		cb:     cb,
		client: c,
		store:  store,
		rng:    rng,
	}

	jww.INFO.Print("Resuming backup with loaded user key.")

	return b, nil
}

// TriggerBackup collates the backup and calls it on the registered backup
// callback. Does nothing if no encryption key or backup callback is registered.
func (b *Backup) TriggerBackup() {
	b.mux.RLock()
	defer b.mux.RUnlock()

	// Skip triggering backup if there is no callback or key registered
	if b.cb == nil {
		jww.TRACE.Print("TriggerBackup: skipping backup, no callback registered.")
		return
	} else if len(b.key) == 0 {
		jww.TRACE.Print("TriggerBackup: skipping backup, no key registered.")
		return
	}

	// Grab backup data
	collatedBackup := b.collateBackup()

	fmt.Printf("%+v\n", collatedBackup)

	// Encrypt backup data with user key
	rand := b.rng.GetStream()
	encryptedBackup, err := collatedBackup.Encrypt(rand, b.key)
	if err != nil {
		jww.FATAL.Panicf("Failed to encrypt backup: %+v", err)
	}
	rand.Close()

	jww.INFO.Print("Triggering backup.")

	// Send backup on callback
	go b.cb(encryptedBackup)
}

// StopBackup stops the backup processes and deletes the user's key.
func (b *Backup) StopBackup() error {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.cb = nil
	b.key = nil
	return deleteKey(b.store.GetKV())
}

// collateBackup gathers all the contents of the backup and stores them in a
// backup.Backup. This backup contains:
//  1. Cryptographic information for the transmission identity
//  2. Cryptographic information for the reception identity
//  3. User's UD facts (username, email, phone number)
//  4. Contact list
func (b *Backup) collateBackup() backup.Backup {
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
	facts := b.store.GetUd().GetFacts()
	for _, userFact := range facts {
		switch userFact.T {
		case fact.Username:
			bu.UserDiscoveryRegistration.Username = &userFact
		case fact.Email:
			bu.UserDiscoveryRegistration.Email = &userFact
		case fact.Phone:
			bu.UserDiscoveryRegistration.Phone = &userFact
		}
	}

	// Get contacts
	bu.Contacts.Identities = b.store.E2e().GetPartners()

	return bu
}
