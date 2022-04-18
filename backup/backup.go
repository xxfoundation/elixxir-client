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
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	// initializeBackup
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

	mux sync.RWMutex

	// Client structures
	client          *api.Client
	store           *storage.Session
	backupContainer *interfaces.BackupContainer
	rng             *fastRNG.StreamGenerator

	jsonParams string
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
func InitializeBackup(password string, updateBackupCb UpdateBackupFn,
	c *api.Client) (*Backup, error) {
	return initializeBackup(
		password, updateBackupCb, c, c.GetStorage(), c.GetBackup(), c.GetRng())
}

// initializeBackup is a helper function that takes in all the fields for Backup
// as parameters for easier testing.
func initializeBackup(password string, updateBackupCb UpdateBackupFn,
	c *api.Client, store *storage.Session,
	backupContainer *interfaces.BackupContainer, rng *fastRNG.StreamGenerator) (
	*Backup, error) {
	b := &Backup{
		updateBackupCb:  updateBackupCb,
		client:          c,
		store:           store,
		backupContainer: backupContainer,
		rng:             rng,
	}

	// Derive key and get generated salt and parameters
	rand := b.rng.GetStream()
	salt, err := backup.MakeSalt(rand)
	if err != nil {
		return nil, err
	}
	rand.Close()

	params := backup.DefaultParams()
	params.Memory = 256 * 1024 // 256 MiB
	params.Threads = 4
	params.Time = 100
	key := backup.DeriveKey(password, salt, params)

	// Save key, salt, and parameters to storage
	err = saveBackup(key, salt, params, b.store.GetKV())
	if err != nil {
		return nil, errors.Errorf(errSaveKeySaltParams, err)
	}

	// Setting backup trigger in client
	b.backupContainer.SetBackup(b.TriggerBackup)

	b.TriggerBackup("initializeBackup")
	jww.INFO.Print("Initialized backup with new user key.")

	return b, nil
}

// ResumeBackup resumes a backup by restoring the Backup object and registering
// a new callback. Call this to resume backups that have already been
// initialized. Returns an error if backups have not already been initialized.
func ResumeBackup(updateBackupCb UpdateBackupFn, c *api.Client) (*Backup, error) {
	return resumeBackup(
		updateBackupCb, c, c.GetStorage(), c.GetBackup(), c.GetRng())
}

// resumeBackup is a helper function that takes in all the fields for Backup as
// parameters for easier testing.
func resumeBackup(updateBackupCb UpdateBackupFn, c *api.Client,
	store *storage.Session, backupContainer *interfaces.BackupContainer,
	rng *fastRNG.StreamGenerator) (*Backup, error) {
	_, _, _, err := loadBackup(store.GetKV())
	if err != nil {
		return nil, err
	}

	b := &Backup{
		updateBackupCb:  updateBackupCb,
		client:          c,
		store:           store,
		backupContainer: backupContainer,
		rng:             rng,
		jsonParams:      loadJson(store.GetKV()),
	}

	// Setting backup trigger in client
	b.backupContainer.SetBackup(b.TriggerBackup)

	jww.INFO.Print("resumed backup with password loaded from storage.")

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

	key, salt, params, err := loadBackup(b.store.GetKV())
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
		if err := storeJson(newJson, b.store.GetKV()); err != nil {
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

	err := deleteBackup(b.store.GetKV())
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
	// Get pending auth requests
	// NOTE: Received requests don't matter here, as those are either
	// not yet noticed by user OR explicitly rejected.
	bu.Contacts.Identities = append(bu.Contacts.Identities,
		b.store.Auth().GetAllSentIDs()...)
	jww.INFO.Printf("backup saw %d contacts", len(bu.Contacts.Identities))
	jww.DEBUG.Printf("contacts in backup list: %+v", bu.Contacts.Identities)
	//deduplicate list
	bu.Contacts.Identities = deduplicate(bu.Contacts.Identities)

	jww.INFO.Printf("backup saved %d contacts after deduplication",
		len(bu.Contacts.Identities))

	// Add the memoized JSON params
	bu.JSONParams = b.jsonParams

	return bu
}

func deduplicate(list []*id.ID) []*id.ID {
	entryMap := make(map[id.ID]bool)
	newList := make([]*id.ID, 0)
	for i, _ := range list {
		if _, value := entryMap[*list[i]]; !value {
			entryMap[*list[i]] = true
			newList = append(newList, list[i])
		}
	}
	return newList
}
