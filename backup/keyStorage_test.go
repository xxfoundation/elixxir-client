////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"testing"

	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
)

// Tests that savePassword saves the password to storage by loading it and
// comparing it to the original.
func Test_savePassword(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expectedPassword := "MySuperSecurePassword"

	// Save the password
	err := savePassword(expectedPassword, kv)
	if err != nil {
		t.Errorf("savePassword returned an error: %+v", err)
	}

	// Attempt to load the password
	obj, err := kv.Get(passwordStorageKey, passwordStorageVersion)
	if err != nil {
		t.Errorf("Failed to get password from storage: %+v", err)
	}

	// Check that the password matches the original
	if expectedPassword != string(obj.Data) {
		t.Errorf("Loaded password does not match original."+
			"\nexpected: %q\nreceived: %q", expectedPassword, obj.Data)
	}
}

// Tests that loadPassword restores the original password saved to stage and
// compares it to the original.
func Test_loadPassword(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expectedPassword := "MySuperSecurePassword"

	// Save the password
	err := kv.Set(passwordStorageKey, passwordStorageVersion, &versioned.Object{
		Version:   passwordStorageVersion,
		Timestamp: netTime.Now(),
		Data:      []byte(expectedPassword),
	})
	if err != nil {
		t.Errorf("Failed to save password to storage: %+v", err)
	}

	// Attempt to load the password
	loadedPassword, err := loadPassword(kv)
	if err != nil {
		t.Errorf("loadPassword returned an error: %+v", err)
	}

	// Check that the password matches the original
	if expectedPassword != loadedPassword {
		t.Errorf("Loaded password does not match original."+
			"\nexpected: %q\nreceived: %q", expectedPassword, loadedPassword)
	}
}

// Tests that deletePassword deletes the password from storage by trying to recover a
// deleted password.
func Test_deletePassword(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expectedPassword := "MySuperSecurePassword"

	// Save the password
	err := savePassword(expectedPassword, kv)
	if err != nil {
		t.Errorf("Failed to save password to storage: %+v", err)
	}

	// Delete the password
	err = deletePassword(kv)
	if err != nil {
		t.Errorf("deletePassword returned an error: %+v", err)
	}

	// Attempt to load the password
	obj, err := loadPassword(kv)
	if err == nil || obj != "" {
		t.Errorf("Loaded object from storage when it should be deleted: %+v", obj)
	}
}
