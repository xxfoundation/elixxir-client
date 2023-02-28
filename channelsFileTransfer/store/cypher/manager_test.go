////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cypher

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/xx_network/crypto/csprng"

	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
)

// Tests that NewManager returns a new Manager that matches the expected
// manager.
func TestNewManager(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	numFps := uint16(64)
	fpv, _ := utility.NewStateVector(uint32(numFps), false,
		cypherManagerFpVectorKey, kv.Prefix(cypherManagerPrefix))
	expected := &Manager{
		key:      &ftCrypto.TransferKey{1, 2, 3},
		fpVector: fpv,
		kv:       kv.Prefix(cypherManagerPrefix),
	}

	manager, err := NewManager(expected.key, numFps, false, kv)
	if err != nil {
		t.Errorf("NewManager returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, manager) {
		t.Errorf("New manager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, manager)
	}
}

// Tests that Manager.PopCypher returns cyphers with correct fingerprint numbers
// and that trying to pop after the last pop results in an error.
func TestManager_PopCypher(t *testing.T) {
	m, _ := newTestManager(64, t)

	for i := uint16(0); i < uint16(m.fpVector.GetNumKeys()); i++ {
		c, err := m.PopCypher()
		if err != nil {
			t.Errorf("Failed to pop cypher #%d: %+v", i, err)
		}

		if c.fpNum != i {
			t.Errorf("Fingerprint number does not match expected."+
				"\nexpected: %d\nreceived: %d", i, c.fpNum)
		}

		if c.Manager != m {
			t.Errorf("Cypher has wrong manager.\nexpected: %v\nreceived: %v",
				m, c.Manager)
		}
	}

	// Test that an error is returned when popping a cypher after all
	// fingerprints have been used
	expectedErr := fmt.Sprintf(errGetNextFp, m.fpVector.GetNumKeys())
	_, err := m.PopCypher()
	if err == nil || (err.Error() != expectedErr) {
		t.Errorf("PopCypher did not return the expected error when all "+
			"fingerprints should be used.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests Manager.GetUnusedCyphers
func TestManager_GetUnusedCyphers(t *testing.T) {
	m, _ := newTestManager(64, t)

	// Use every other key
	for i := uint32(0); i < m.fpVector.GetNumKeys(); i += 2 {
		m.fpVector.Use(i)
	}

	// Check that every other key is in the list
	for i, c := range m.GetUnusedCyphers() {
		if c.fpNum != uint16(2*i)+1 {
			t.Errorf("Fingerprint number #%d incorrect."+
				"\nexpected: %d\nreceived: %d", i, 2*i+1, c.fpNum)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that a Manager loaded via LoadManager matches the original.
func TestLoadManager(t *testing.T) {
	m, kv := newTestManager(64, t)

	// Use every other key
	for i := uint32(0); i < m.fpVector.GetNumKeys(); i += 2 {
		m.fpVector.Use(i)
	}

	newManager, err := LoadManager(kv)
	if err != nil {
		t.Errorf("Failed to load manager: %+v", err)
	}

	if !reflect.DeepEqual(m, newManager) {
		t.Errorf("Loaded manager does not match original."+
			"\nexpected: %+v\nreceived: %+v", m, newManager)
	}
}

// Tests that LoadManager returns the expected error when the key cannot be
// loaded from storage
func TestLoadManager_LoadKeyError(t *testing.T) {
	m, kv := newTestManager(64, t)
	_ = m.kv.Delete(cypherManagerKeyStoreKey, cypherManagerKeyStoreVersion)

	expectedErr := strings.Split(errLoadKey, ":")[0]
	_, err := LoadManager(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Unexpected error.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests that LoadManager returns the expected error when the state vector
// cannot be loaded from storage
func TestLoadManager_LoadStateVectorError(t *testing.T) {
	m, kv := newTestManager(64, t)
	_ = m.fpVector.Delete()

	expectedErr := strings.Split(errLoadFpVector, ":")[0]
	_, err := LoadManager(kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Unexpected error.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests that Manager.Delete deletes the storage by trying to load the manager.
func TestManager_Delete(t *testing.T) {
	m, _ := newTestManager(64, t)

	err := m.Delete()
	if err != nil {
		t.Errorf("Failed to delete manager: %+v", err)
	}

	_, err = LoadManager(m.kv)
	if err == nil {
		t.Error("Failed to receive error when loading manager that was deleted.")
	}
}

// Tests that a transfer key saved via saveKey can be loaded via loadKey.
func Test_saveKey_loadKey(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	key := &ftCrypto.TransferKey{42}

	err := saveKey(key, kv)
	if err != nil {
		t.Errorf("Error when saving key: %+v", err)
	}

	loadedKey, err := loadKey(kv)
	if err != nil {
		t.Errorf("Error when loading key: %+v", err)
	}

	if *key != *loadedKey {
		t.Errorf("Loaded key does not match original."+
			"\nexpected: %s\nreceived: %s", key, loadedKey)
	}
}

// newTestManager creates a new Manager for testing.
func newTestManager(numFps uint16, t *testing.T) (*Manager, *versioned.KV) {
	key, err := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	if err != nil {
		t.Errorf("Failed to generate transfer key: %+v", err)
	}

	kv := versioned.NewKV(ekv.MakeMemstore())
	m, err := NewManager(&key, numFps, false, kv)
	if err != nil {
		t.Errorf("Failed to make new Manager: %+v", err)
	}

	return m, kv
}
