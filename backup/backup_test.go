////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"bytes"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/interfaces/params"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
)

// Tests that Backup.initializeBackup returns a new Backup with a copy of the
// key and the callback.
func Test_initializeBackup(t *testing.T) {
	cbChan := make(chan []byte, 2)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	expectedPassword := "MySuperSecurePassword"
	b, err := initializeBackup(expectedPassword, cb, nil,
		storage.InitTestingSession(t), &interfaces.BackupContainer{},
		fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG))
	if err != nil {
		t.Errorf("initializeBackup returned an error: %+v", err)
	}

	select {
	case <-cbChan:
	case <-time.After(10 * time.Millisecond):
		t.Error("Timed out waiting for callback.")
	}

	// Check that the correct password is in storage
	loadedPassword, err := loadPassword(b.store.GetKV())
	if err != nil {
		t.Errorf("Failed to load password: %+v", err)
	}
	if expectedPassword != loadedPassword {
		t.Errorf("Loaded invalid key.\nexpected: %q\nreceived: %q",
			expectedPassword, loadedPassword)
	}

	// Check that the key, salt, and params were saved to storage
	key, salt, p, err := loadBackup(b.store.GetKV())
	if err != nil {
		t.Errorf("Failed to load key, salt, and params: %+v", err)
	}
	if len(key) != keyLen || bytes.Equal(key, make([]byte, keyLen)) {
		t.Errorf("Invalid key: %v", key)
	}
	if len(salt) != saltLen || bytes.Equal(salt, make([]byte, saltLen)) {
		t.Errorf("Invalid salt: %v", salt)
	}
	if !reflect.DeepEqual(p, backup.DefaultParams()) {
		t.Errorf("Invalid params.\nexpected: %+v\nreceived: %+v",
			backup.DefaultParams(), p)
	}

	encryptedBackup := []byte("encryptedBackup")
	go b.updateBackupCb(encryptedBackup)

	select {
	case r := <-cbChan:
		if !bytes.Equal(encryptedBackup, r) {
			t.Errorf("Callback has unexepected data."+
				"\nexpected: %q\nreceived: %q", encryptedBackup, r)
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("Timed out waiting for callback.")
	}
}

// Initialises a new backup and then tests that Backup.resumeBackup overwrites
// the callback but keeps the password.
func Test_resumeBackup(t *testing.T) {
	// Start the first backup
	cbChan1 := make(chan []byte)
	cb1 := func(encryptedBackup []byte) { cbChan1 <- encryptedBackup }
	s := storage.InitTestingSession(t)
	expectedPassword := "MySuperSecurePassword"
	b, err := initializeBackup(expectedPassword, cb1, nil, s,
		&interfaces.BackupContainer{},
		fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG))
	if err != nil {
		t.Errorf("Failed to initialize new Backup: %+v", err)
	}

	select {
	case <-cbChan1:
	case <-time.After(10 * time.Millisecond):
		t.Error("Timed out waiting for callback.")
	}

	// Get key and salt to compare to later
	key1, salt1, _, err := loadBackup(b.store.GetKV())
	if err != nil {
		t.Errorf("Failed to load key, salt, and params from newly "+
			"initialized backup: %+v", err)
	}

	// Resume the backup with a new callback
	cbChan2 := make(chan []byte)
	cb2 := func(encryptedBackup []byte) { cbChan2 <- encryptedBackup }
	b2, err := resumeBackup(cb2, nil, s, &interfaces.BackupContainer{},
		fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG))
	if err != nil {
		t.Errorf("resumeBackup returned an error: %+v", err)
	}

	// Check that the correct password is in storage
	loadedPassword, err := loadPassword(b.store.GetKV())
	if err != nil {
		t.Errorf("Failed to load password: %+v", err)
	}
	if expectedPassword != loadedPassword {
		t.Errorf("Loaded invalid key.\nexpected: %q\nreceived: %q",
			expectedPassword, loadedPassword)
	}

	// Get key, salt, and parameters of resumed backup
	key2, salt2, _, err := loadBackup(b.store.GetKV())
	if err != nil {
		t.Errorf("Failed to load key, salt, and params from resumed "+
			"backup: %+v", err)
	}

	// Check that the loaded key and salt are the same
	if !bytes.Equal(key1, key2) {
		t.Errorf("New key does not match old key.\nold: %v\nnew: %v", key1, key2)
	}
	if !bytes.Equal(salt1, salt2) {
		t.Errorf("New salt does not match old salt.\nold: %v\nnew: %v", salt1, salt2)
	}

	encryptedBackup := []byte("encryptedBackup")
	go b2.updateBackupCb(encryptedBackup)

	select {
	case r := <-cbChan1:
		t.Errorf("Callback of first Backup called: %q", r)
	case r := <-cbChan2:
		if !bytes.Equal(encryptedBackup, r) {
			t.Errorf("Callback has unexepected data."+
				"\nexpected: %q\nreceived: %q", encryptedBackup, r)
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("Timed out waiting for callback.")
	}
}

// Error path: Tests that Backup.resumeBackup returns an error if no password is
// present in storage.
func Test_resumeBackup_NoKeyError(t *testing.T) {
	expectedErr := strings.Split(errLoadPassword, "%")[0]
	s := storage.InitTestingSession(t)
	_, err := resumeBackup(nil, nil, s, &interfaces.BackupContainer{}, nil)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("resumeBackup did not return the expected error when no "+
			"password is present.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Backup.TriggerBackup triggers the callback and that the data
// received can be decrypted.
func TestBackup_TriggerBackup(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	b := newTestBackup("MySuperSecurePassword", cb, t)

	// Get password
	password, err := loadPassword(b.store.GetKV())
	if err != nil {
		t.Errorf("Failed to load password from storage: %+v", err)
	}

	collatedBackup := b.assembleBackup()

	b.TriggerBackup("")

	select {
	case r := <-cbChan:
		receivedCollatedBackup := backup.Backup{}
		err := receivedCollatedBackup.Decrypt(password, r)
		if err != nil {
			t.Errorf("Failed to decrypt collated backup: %+v", err)
		} else if !reflect.DeepEqual(collatedBackup, receivedCollatedBackup) {
			t.Errorf("Unexpected decrypted collated backup."+
				"\nexpected: %#v\nreceived: %#v",
				collatedBackup, receivedCollatedBackup)
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("Timed out waiting for callback.")
	}
}

// Tests that Backup.TriggerBackup does not call the callback if there is no
// key, salt, and params in storage.
func TestBackup_TriggerBackup_NoKey(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	b := newTestBackup("MySuperSecurePassword", cb, t)
	select {
	case <-cbChan:
	case <-time.After(10 * time.Millisecond):
		t.Errorf("backup not called")
	}

	err := deleteBackup(b.store.GetKV())
	if err != nil {
		t.Errorf("Failed to delete key, salt, and params: %+v", err)
	}

	b.TriggerBackup("")

	select {
	case r := <-cbChan:
		t.Errorf("Callback received when it should not have been called: %q", r)
	case <-time.After(10 * time.Millisecond):
	}

}

// Tests that Backup.StopBackup prevents the callback from triggering and that
// the password, key, salt, and parameters were deleted.
func TestBackup_StopBackup(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	b := newTestBackup("MySuperSecurePassword", cb, t)
	select {
	case <-cbChan:
	case <-time.After(1000 * time.Millisecond):
		t.Errorf("backup not called")
	}

	err := b.StopBackup()
	if err != nil {
		t.Errorf("StopBackup returned an error: %+v", err)
	}

	if b.updateBackupCb != nil {
		t.Error("Callback not cleared.")
	}

	b.TriggerBackup("")

	select {
	case r := <-cbChan:
		t.Errorf("Callback received when it should not have been called: %q", r)
	case <-time.After(10 * time.Millisecond):
	}

	// Make sure password is deleted
	password, err := loadPassword(b.store.GetKV())
	if err == nil || len(password) != 0 {
		t.Errorf("Loaded password that should be deleted: %q", password)
	}

	// Make sure key, salt, and params are deleted
	key, salt, p, err := loadBackup(b.store.GetKV())
	if err == nil || len(key) != 0 || len(salt) != 0 || p != (backup.Params{}) {
		t.Errorf("Loaded key, salt, and params that should be deleted.")
	}
}

func TestBackup_IsBackupRunning(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	b := newTestBackup("MySuperSecurePassword", cb, t)

	// Check that the backup is running after being initialized
	if !b.IsBackupRunning() {
		t.Error("Backup is not running after initialization.")
	}

	// Stop the backup
	err := b.StopBackup()
	if err != nil {
		t.Errorf("Failed to stop backup: %+v", err)
	}

	// Check that the backup is stopped
	if b.IsBackupRunning() {
		t.Error("Backup is running after being stopped.")
	}
}

func TestBackup_AddJson(t *testing.T) {
	b := newTestBackup("MySuperSecurePassword", nil, t)
	s := b.store
	json := "{'data': {'one': 1}}"

	expectedCollatedBackup := backup.Backup{
		RegistrationTimestamp: s.GetUser().RegistrationTimestamp,
		TransmissionIdentity: backup.TransmissionIdentity{
			RSASigningPrivateKey: s.GetUser().TransmissionRSA,
			RegistrarSignature:   s.User().GetTransmissionRegistrationValidationSignature(),
			Salt:                 s.GetUser().TransmissionSalt,
			ComputedID:           s.GetUser().TransmissionID,
		},
		ReceptionIdentity: backup.ReceptionIdentity{
			RSASigningPrivateKey: s.GetUser().ReceptionRSA,
			RegistrarSignature:   s.User().GetReceptionRegistrationValidationSignature(),
			Salt:                 s.GetUser().ReceptionSalt,
			ComputedID:           s.GetUser().ReceptionID,
			DHPrivateKey:         s.GetUser().E2eDhPrivateKey,
			DHPublicKey:          s.GetUser().E2eDhPublicKey,
		},
		UserDiscoveryRegistration: backup.UserDiscoveryRegistration{
			FactList: s.GetUd().GetFacts(),
		},
		Contacts:   backup.Contacts{Identities: s.E2e().GetPartners()},
		JSONParams: json,
	}

	b.AddJson(json)

	collatedBackup := b.assembleBackup()
	if !reflect.DeepEqual(expectedCollatedBackup, collatedBackup) {
		t.Errorf("Collated backup does not match expected."+
			"\nexpected: %+v\nreceived: %+v",
			expectedCollatedBackup, collatedBackup)
	}
}

func TestBackup_AddJson_badJson(t *testing.T) {
	b := newTestBackup("MySuperSecurePassword", nil, t)
	s := b.store
	json := "abc{'i'm a bad json: 'one': 1'''}}"

	expectedCollatedBackup := backup.Backup{
		RegistrationTimestamp: s.GetUser().RegistrationTimestamp,
		TransmissionIdentity: backup.TransmissionIdentity{
			RSASigningPrivateKey: s.GetUser().TransmissionRSA,
			RegistrarSignature:   s.User().GetTransmissionRegistrationValidationSignature(),
			Salt:                 s.GetUser().TransmissionSalt,
			ComputedID:           s.GetUser().TransmissionID,
		},
		ReceptionIdentity: backup.ReceptionIdentity{
			RSASigningPrivateKey: s.GetUser().ReceptionRSA,
			RegistrarSignature:   s.User().GetReceptionRegistrationValidationSignature(),
			Salt:                 s.GetUser().ReceptionSalt,
			ComputedID:           s.GetUser().ReceptionID,
			DHPrivateKey:         s.GetUser().E2eDhPrivateKey,
			DHPublicKey:          s.GetUser().E2eDhPublicKey,
		},
		UserDiscoveryRegistration: backup.UserDiscoveryRegistration{
			FactList: s.GetUd().GetFacts(),
		},
		Contacts:   backup.Contacts{Identities: s.E2e().GetPartners()},
		JSONParams: json,
	}

	b.AddJson(json)

	collatedBackup := b.assembleBackup()
	if !reflect.DeepEqual(expectedCollatedBackup, collatedBackup) {
		t.Errorf("Collated backup does not match expected."+
			"\nexpected: %+v\nreceived: %+v",
			expectedCollatedBackup, collatedBackup)
	}
}

// Tests that Backup.assembleBackup returns the backup.Backup with the expected
// results.
func TestBackup_assembleBackup(t *testing.T) {
	b := newTestBackup("MySuperSecurePassword", nil, t)
	s := b.store

	rng := csprng.NewSystemRNG()
	for i := 0; i < 10; i++ {
		recipient, _ := id.NewRandomID(rng, id.User)
		dhKey := s.E2e().GetGroup().NewInt(int64(i + 10))
		pubKey := diffieHellman.GeneratePublicKey(dhKey, s.E2e().GetGroup())
		_, mySidhPriv := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, rng)
		theirSidhPub, _ := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhB, rng)
		p := params.GetDefaultE2ESessionParams()

		err := s.E2e().AddPartner(
			recipient, pubKey, dhKey, mySidhPriv, theirSidhPub, p, p)
		if err != nil {
			t.Errorf("Failed to add partner %s: %+v", recipient, err)
		}
	}

	expectedCollatedBackup := backup.Backup{
		RegistrationTimestamp: s.GetUser().RegistrationTimestamp,
		TransmissionIdentity: backup.TransmissionIdentity{
			RSASigningPrivateKey: s.GetUser().TransmissionRSA,
			RegistrarSignature:   s.User().GetTransmissionRegistrationValidationSignature(),
			Salt:                 s.GetUser().TransmissionSalt,
			ComputedID:           s.GetUser().TransmissionID,
		},
		ReceptionIdentity: backup.ReceptionIdentity{
			RSASigningPrivateKey: s.GetUser().ReceptionRSA,
			RegistrarSignature:   s.User().GetReceptionRegistrationValidationSignature(),
			Salt:                 s.GetUser().ReceptionSalt,
			ComputedID:           s.GetUser().ReceptionID,
			DHPrivateKey:         s.GetUser().E2eDhPrivateKey,
			DHPublicKey:          s.GetUser().E2eDhPublicKey,
		},
		UserDiscoveryRegistration: backup.UserDiscoveryRegistration{
			FactList: s.GetUd().GetFacts(),
		},
		Contacts: backup.Contacts{Identities: s.E2e().GetPartners()},
	}

	collatedBackup := b.assembleBackup()

	sort.Slice(expectedCollatedBackup.Contacts.Identities, func(i, j int) bool {
		return bytes.Compare(expectedCollatedBackup.Contacts.Identities[i].Bytes(),
			expectedCollatedBackup.Contacts.Identities[j].Bytes()) == -1
	})

	sort.Slice(collatedBackup.Contacts.Identities, func(i, j int) bool {
		return bytes.Compare(collatedBackup.Contacts.Identities[i].Bytes(),
			collatedBackup.Contacts.Identities[j].Bytes()) == -1
	})

	if !reflect.DeepEqual(expectedCollatedBackup, collatedBackup) {
		t.Errorf("Collated backup does not match expected."+
			"\nexpected: %+v\nreceived: %+v",
			expectedCollatedBackup, collatedBackup)
	}
}

// newTestBackup creates a new Backup for testing.
func newTestBackup(password string, cb UpdateBackupFn, t *testing.T) *Backup {
	b, err := initializeBackup(
		password,
		cb,
		nil,
		storage.InitTestingSession(t),
		&interfaces.BackupContainer{},
		fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
	)
	if err != nil {
		t.Fatalf("Failed to initialize backup: %+v", err)
	}

	return b
}
