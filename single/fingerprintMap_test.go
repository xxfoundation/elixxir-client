package single

import (
	"bytes"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"reflect"
	"testing"
)

// Happy path.
func Test_newFingerprintMap(t *testing.T) {
	baseKey := getGroup().NewInt(42)
	messageCount := 3
	expected := &fingerprintMap{
		fps: map[format.Fingerprint]uint64{
			singleUse.NewResponseFingerprint(baseKey, 0): 0,
			singleUse.NewResponseFingerprint(baseKey, 1): 1,
			singleUse.NewResponseFingerprint(baseKey, 2): 2,
		},
	}

	fpm := newFingerprintMap(baseKey, uint64(messageCount))

	if !reflect.DeepEqual(expected, fpm) {
		t.Errorf("newFingerprintMap() did not generate the expected map."+
			"\nexpected: %+v\nreceived: %+v", expected, fpm)
	}
}

// Happy path.
func TestFingerprintMap_getKey(t *testing.T) {
	baseKey := getGroup().NewInt(42)
	fpm := newFingerprintMap(baseKey, 5)
	fp := singleUse.NewResponseFingerprint(baseKey, 3)
	expectedKey := singleUse.NewResponseKey(baseKey, 3)

	testKey, exists := fpm.getKey(baseKey, fp)
	if !exists {
		t.Errorf("getKey() failed to find key that exists in map."+
			"\nfingerprint: %+v", fp)
	}

	if !bytes.Equal(expectedKey, testKey) {
		t.Errorf("getKey() returned the wrong key.\nexpected: %+v\nreceived: %+v",
			expectedKey, testKey)
	}

	testFP, exists := fpm.fps[fp]
	if exists {
		t.Errorf("getKey() failed to delete the found fingerprint."+
			"\nfingerprint: %+v", testFP)
	}
}

// Error path: fingerprint does not exist in map.
func TestFingerprintMap_getKey_FingerprintNotInMap(t *testing.T) {
	baseKey := getGroup().NewInt(42)
	fpm := newFingerprintMap(baseKey, 5)
	fp := singleUse.NewResponseFingerprint(baseKey, 30)

	key, exists := fpm.getKey(baseKey, fp)
	if exists {
		t.Errorf("getKey() found a fingerprint in the map that should not exist."+
			"\nfingerprint: %+v\nkey:         %+v", fp, key)
	}

	// Ensure no fingerprints were deleted
	if len(fpm.fps) != 5 {
		t.Errorf("getKey() deleted fingerprint."+
			"\nexpected size: %d\nreceived size: %d", 5, len(fpm.fps))
	}
}
