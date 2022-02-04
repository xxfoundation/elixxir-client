////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"bytes"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"golang.org/x/crypto/chacha20poly1305"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Tests that Backup.initializeBackup returns a new Backup with a copy of the
// key and the callback.
func Test_initializeBackup(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	expectedKey := []byte("MyTestKey")
	b, err := initializeBackup(
		expectedKey, cb, nil, storage.InitTestingSession(t), nil)
	if err != nil {
		t.Errorf("initializeBackup returned an error: %+v", err)
	}

	// Check that Backup has a copy of the correct key
	if !bytes.Equal(expectedKey, b.key) {
		t.Errorf("Backup has invalid key.\nexpected: %q\nreceived: %q",
			expectedKey, b.key)
	} else if &expectedKey[0] == &b.key[0] {
		t.Errorf("Backup does not have copy of key."+
			"\noriginal: %p\nreceived: %p", expectedKey, b.key)
	}

	// Check that the correct key is in storage
	loadedKey, err := loadKey(b.store.GetKV())
	if err != nil {
		t.Errorf("Failed to load key: %+v", err)
	}
	if !bytes.Equal(expectedKey, loadedKey) {
		t.Errorf("Loaded invalid key.\nexpected: %q\nreceived: %q",
			expectedKey, loadedKey)
	}

	encryptedBackup := []byte("encryptedBackup")
	go b.cb(encryptedBackup)

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
// the callback but keeps the key.
func Test_resumeBackup(t *testing.T) {
	// Start the first backup
	cbChan1 := make(chan []byte)
	cb1 := func(encryptedBackup []byte) { cbChan1 <- encryptedBackup }
	s := storage.InitTestingSession(t)
	expectedKey := []byte("MyTestKey")
	_, err := initializeBackup(expectedKey, cb1, nil, s, nil)
	if err != nil {
		t.Errorf("Failed to initialize new Backup: %+v", err)
	}

	// Resume the backup with a new callback
	cbChan2 := make(chan []byte)
	cb2 := func(encryptedBackup []byte) { cbChan2 <- encryptedBackup }
	b2, err := resumeBackup(cb2, nil, s, nil)
	if err != nil {
		t.Errorf("resumeBackup returned an error: %+v", err)
	}

	// Check that Backup has a copy of the correct key
	if !bytes.Equal(expectedKey, b2.key) {
		t.Errorf("Backup has invalid key.\nexpected: %q\nreceived: %q",
			expectedKey, b2.key)
	} else if &expectedKey[0] == &b2.key[0] {
		t.Errorf("Backup does not have copy of key."+
			"\noriginal: %p\nreceived: %p", expectedKey, b2.key)
	}

	// Check that the correct key is in storage
	loadedKey, err := loadKey(b2.store.GetKV())
	if err != nil {
		t.Errorf("Failed to load key: %+v", err)
	}
	if !bytes.Equal(expectedKey, loadedKey) {
		t.Errorf("Loaded invalid key.\nexpected: %q\nreceived: %q",
			expectedKey, loadedKey)
	}

	encryptedBackup := []byte("encryptedBackup")
	go b2.cb(encryptedBackup)

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

// Error path: Tests that Backup.resumeBackup returns an error if no key is
// present in storage.
func Test_resumeBackup_NoKeyError(t *testing.T) {
	expectedErr := strings.Split(errLoadKey, "%")[0]
	s := storage.InitTestingSession(t)
	_, err := resumeBackup(nil, nil, s, nil)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("resumeBackup did not return the expected error when no key"+
			"is present.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Backup.TriggerBackup triggers the callback and that the data
// received can be decrypted.
func TestBackup_TriggerBackup(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	b := newTestBackup(cb, t)

	collatedBackup := b.collateBackup()

	b.TriggerBackup()

	select {
	case r := <-cbChan:
		receivedCollatedBackup := backup.Backup{}
		err := receivedCollatedBackup.Decrypt(b.key, r)
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

// Tests that Backup.TriggerBackup does not panic if there is no callback.
func TestBackup_TriggerBackup_NoCallback(t *testing.T) {
	b := newTestBackup(nil, t)

	b.TriggerBackup()
}

// Tests that Backup.TriggerBackup does not call the callback if there is no
// key.
func TestBackup_TriggerBackup_NoKey(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	b := newTestBackup(cb, t)
	b.key = nil

	b.TriggerBackup()

	select {
	case r := <-cbChan:
		t.Errorf("Callback received when it should not have been called: %q", r)
	case <-time.After(10 * time.Millisecond):
	}
}

// Tests that Backup.StopBackup prevents the callback from triggering and that
// the key was deleted.
func TestBackup_StopBackup(t *testing.T) {
	cbChan := make(chan []byte)
	cb := func(encryptedBackup []byte) { cbChan <- encryptedBackup }
	b := newTestBackup(cb, t)

	err := b.StopBackup()
	if err != nil {
		t.Errorf("StopBackup returned an error: %+v", err)
	}

	if b.cb != nil {
		t.Error("Callback not cleared.")
	}
	if b.key != nil {
		t.Errorf("Key not cleared: %v", b.key)
	}

	b.TriggerBackup()

	select {
	case r := <-cbChan:
		t.Errorf("Callback received when it should not have been called: %q", r)
	case <-time.After(10 * time.Millisecond):
	}

	key, err := loadKey(b.store.GetKV())
	if err == nil || len(key) != 0 {
		t.Errorf("Loaded key that should be deleted: %q", key)
	}
}

// Tests that Backup.collateBackup returns the backup.Backup with the expected
// results.
func TestBackup_collateBackup(t *testing.T) {
	b := newTestBackup(nil, t)
	s := b.store

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
			Username: &s.GetUd().GetFacts()[0],
		},
		Contacts: backup.Contacts{Identities: s.E2e().GetPartners()},
	}

	collatedBackup := b.collateBackup()

	if !reflect.DeepEqual(expectedCollatedBackup, collatedBackup) {
		t.Errorf("Collated backup does not match expected."+
			"\nexpected: %+v\nreceived: %+v",
			expectedCollatedBackup, collatedBackup)
	}
}

// newTestBackup creates a new Backup for testing.
func newTestBackup(cb GetBackup, t *testing.T) *Backup {
	key := []byte("MyTestKey")
	key = bytes.Repeat(key, chacha20poly1305.KeySize/len(key)+1)[:chacha20poly1305.KeySize]
	b, err := initializeBackup(
		key,
		cb,
		nil,
		storage.InitTestingSession(t),
		fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
	)
	if err != nil {
		t.Fatalf("Failed to initialize backup: %+v", err)
	}

	return b
}
