////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"bytes"
	"gitlab.com/elixxir/client/xxdk"
	"reflect"
	"testing"
	"time"

	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"

	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
)

// Tests that Backup.InitializeBackup returns a new Backup with a copy of the
// key and the callback.
func Test_InitializeBackup(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	cbChan := make(chan []byte, 2)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	expectedPassword := "MySuperSecurePassword"
	b, err := InitializeBackup(expectedPassword, cb, &xxdk.Container{},
		newMockE2e(t),
		newMockSession(t), newMockUserDiscovery(), kv, rngGen)
	if err != nil {
		t.Errorf("InitializeBackup returned an error: %+v", err)
	}

	select {
	case <-cbChan:
	case <-time.After(10 * time.Millisecond):
		t.Error("Timed out waiting for callback.")
	}

	// Check that the key, salt, and params were saved to storage
	key, salt, _, err := loadBackup(b.kv)
	if err != nil {
		t.Errorf("Failed to load key, salt, and params: %+v", err)
	}
	if len(key) != keyLen || bytes.Equal(key, make([]byte, keyLen)) {
		t.Errorf("Invalid key: %v", key)
	}
	if len(salt) != saltLen || bytes.Equal(salt, make([]byte, saltLen)) {
		t.Errorf("Invalid salt: %v", salt)
	}
	// if !reflect.DeepEqual(p, backup.DefaultParams()) {
	// 	t.Errorf("Invalid params.\nexpected: %+v\nreceived: %+v",
	// 		backup.DefaultParams(), p)
	// }

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

// Initialises a new backup and then tests that ResumeBackup overwrites the
// callback but keeps the password.
func Test_ResumeBackup(t *testing.T) {
	// Start the first backup
	kv := versioned.NewKV(ekv.MakeMemstore())
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	cbChan1 := make(chan []byte)
	cb1 := func(encryptedBackup []byte) { cbChan1 <- encryptedBackup }
	expectedPassword := "MySuperSecurePassword"
	b, err := InitializeBackup(expectedPassword, cb1, &xxdk.Container{},
		newMockE2e(t), newMockSession(t), newMockUserDiscovery(), kv, rngGen)
	if err != nil {
		t.Errorf("Failed to initialize new Backup: %+v", err)
	}

	select {
	case <-cbChan1:
	case <-time.After(10 * time.Millisecond):
		t.Error("Timed out waiting for callback.")
	}

	// get key and salt to compare to later
	key1, salt1, _, err := loadBackup(b.kv)
	if err != nil {
		t.Errorf("Failed to load key, salt, and params from newly "+
			"initialized backup: %+v", err)
	}

	// Resume the backup with a new callback
	cbChan2 := make(chan []byte)
	cb2 := func(encryptedBackup []byte) { cbChan2 <- encryptedBackup }
	b2, err := ResumeBackup(cb2, &xxdk.Container{}, newMockE2e(t), newMockSession(t),
		newMockUserDiscovery(), kv, rngGen)
	if err != nil {
		t.Errorf("ResumeBackup returned an error: %+v", err)
	}

	// Get key, salt, and parameters of resumed backup
	key2, salt2, _, err := loadBackup(b.kv)
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

// Error path: Tests that ResumeBackup returns an error if no password is
// present in storage.
func Test_resumeBackup_NoKeyError(t *testing.T) {
	expectedErr := "object not found"
	s := storage.InitTestingSession(t)
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	_, err := ResumeBackup(nil, &xxdk.Container{}, newMockE2e(t), newMockSession(t),
		newMockUserDiscovery(), s.GetKV(), rngGen)
	if err == nil || s.GetKV().Exists(err) {
		t.Errorf("ResumeBackup did not return the expected error when no "+
			"password is present.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Backup.TriggerBackup triggers the callback and that the data
// received can be decrypted.
func TestBackup_TriggerBackup(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	password := "MySuperSecurePassword"
	b := newTestBackup(password, cb, t)

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

	err := deleteBackup(b.kv)
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

	// Make sure key, salt, and params are deleted
	key, salt, p, err := loadBackup(b.kv)
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
	s := b.session.(*mockSession)
	e2e := b.e2e.(*mockE2e)
	json := "{'data': {'one': 1}}"

	expected := backup.Backup{
		RegistrationCode:      s.regCode,
		RegistrationTimestamp: s.registrationTimestamp.UnixNano(),
		TransmissionIdentity: backup.TransmissionIdentity{
			RSASigningPrivateKey: s.transmissionRSA,
			RegistrarSignature:   s.transmissionRegistrationValidationSignature,
			Salt:                 s.transmissionSalt,
			ComputedID:           s.transmissionID,
		},
		ReceptionIdentity: backup.ReceptionIdentity{
			RSASigningPrivateKey: s.receptionRSA,
			RegistrarSignature:   s.receptionRegistrationValidationSignature,
			Salt:                 s.receptionSalt,
			ComputedID:           s.receptionID,
			DHPrivateKey:         e2e.historicalDHPrivkey,
			DHPublicKey:          e2e.historicalDHPubkey,
		},
		UserDiscoveryRegistration: backup.UserDiscoveryRegistration{
			FactList: b.ud.(*mockUserDiscovery).facts,
		},
		Contacts:   backup.Contacts{Identities: e2e.partnerIDs},
		JSONParams: json,
	}

	b.AddJson(json)

	collatedBackup := b.assembleBackup()
	if !reflect.DeepEqual(expected, collatedBackup) {
		t.Errorf("Collated backup does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, collatedBackup)
	}
}

func TestBackup_AddJson_badJson(t *testing.T) {
	b := newTestBackup("MySuperSecurePassword", nil, t)
	s := b.session.(*mockSession)
	e2e := b.e2e.(*mockE2e)
	json := "abc{'i'm a bad json: 'one': 1'''}}"

	expected := backup.Backup{
		RegistrationCode:      s.regCode,
		RegistrationTimestamp: s.registrationTimestamp.UnixNano(),
		TransmissionIdentity: backup.TransmissionIdentity{
			RSASigningPrivateKey: s.transmissionRSA,
			RegistrarSignature:   s.transmissionRegistrationValidationSignature,
			Salt:                 s.transmissionSalt,
			ComputedID:           s.transmissionID,
		},
		ReceptionIdentity: backup.ReceptionIdentity{
			RSASigningPrivateKey: s.receptionRSA,
			RegistrarSignature:   s.receptionRegistrationValidationSignature,
			Salt:                 s.receptionSalt,
			ComputedID:           s.receptionID,
			DHPrivateKey:         e2e.historicalDHPrivkey,
			DHPublicKey:          e2e.historicalDHPubkey,
		},
		UserDiscoveryRegistration: backup.UserDiscoveryRegistration{
			FactList: b.ud.(*mockUserDiscovery).facts,
		},
		Contacts:   backup.Contacts{Identities: e2e.partnerIDs},
		JSONParams: json,
	}

	b.AddJson(json)

	collatedBackup := b.assembleBackup()
	if !reflect.DeepEqual(expected, collatedBackup) {
		t.Errorf("Collated backup does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, collatedBackup)
	}
}

// Tests that Backup.assembleBackup returns the backup.Backup with the expected
// results.
func TestBackup_assembleBackup(t *testing.T) {
	b := newTestBackup("MySuperSecurePassword", nil, t)
	s := b.session.(*mockSession)
	e2e := b.e2e.(*mockE2e)

	expected := backup.Backup{
		RegistrationCode:      s.regCode,
		RegistrationTimestamp: s.registrationTimestamp.UnixNano(),
		TransmissionIdentity: backup.TransmissionIdentity{
			RSASigningPrivateKey: s.transmissionRSA,
			RegistrarSignature:   s.transmissionRegistrationValidationSignature,
			Salt:                 s.transmissionSalt,
			ComputedID:           s.transmissionID,
		},
		ReceptionIdentity: backup.ReceptionIdentity{
			RSASigningPrivateKey: s.receptionRSA,
			RegistrarSignature:   s.receptionRegistrationValidationSignature,
			Salt:                 s.receptionSalt,
			ComputedID:           s.receptionID,
			DHPrivateKey:         e2e.historicalDHPrivkey,
			DHPublicKey:          e2e.historicalDHPubkey,
		},
		UserDiscoveryRegistration: backup.UserDiscoveryRegistration{
			FactList: b.ud.(*mockUserDiscovery).facts,
		},
		Contacts: backup.Contacts{Identities: e2e.partnerIDs},
	}

	collatedBackup := b.assembleBackup()

	if !reflect.DeepEqual(expected, collatedBackup) {
		t.Errorf("Collated backup does not match expected."+
			"\nexpected: %+v\nreceived: %+v",
			expected, collatedBackup)
	}
}

// newTestBackup creates a new Backup for testing.
func newTestBackup(password string, cb UpdateBackupFn, t *testing.T) *Backup {
	b, err := InitializeBackup(
		password,
		cb,
		&xxdk.Container{},
		newMockE2e(t),
		newMockSession(t),
		newMockUserDiscovery(),
		versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
	)
	if err != nil {
		t.Fatalf("Failed to initialize backup: %+v", err)
	}

	return b
}

// Tests that Backup.InitializeBackup returns a new Backup with a copy of the
// key and the callback.
func Benchmark_InitializeBackup(t *testing.B) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	cbChan := make(chan []byte, 2)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	expectedPassword := "MySuperSecurePassword"
	for i := 0; i < t.N; i++ {
		_, err := InitializeBackup(expectedPassword, cb,
			&xxdk.Container{},
			newMockE2e(t),
			newMockSession(t), newMockUserDiscovery(), kv, rngGen)
		if err != nil {
			t.Errorf("InitializeBackup returned an error: %+v", err)
		}
	}
}
