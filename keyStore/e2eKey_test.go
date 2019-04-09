package keyStore

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Test key fingerprint for consistency
func TestE2EKey_KeyFingerprint(t *testing.T) {
	grp := initGroup()
	key := new(E2EKey)
	key.key = grp.NewInt(42)
	keyFP := key.KeyFingerprint()
	expectedFP, _ := hex.DecodeString(
		"395a122eb1402bf256d86e3fa44764cf" +
			"9acc559017a00b2b9ee12498e73ef2b5")

	if !bytes.Equal(keyFP[:], expectedFP) {
		t.Errorf("Key Fingerprint value is wrong. Expected %x" +
			", got %x", expectedFP, keyFP[:])
	}
}
