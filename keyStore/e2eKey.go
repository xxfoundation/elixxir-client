package keyStore

import (
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

type E2EKey struct {
	// Link to Manager
	manager *KeyManager

	// Key to be used
	key *cyclic.Int

	// Designation of crypto type
	outer parse.CryptoType

	// keyNum is needed by Key Manager
	// to keep track of which receiving keys
	// have been used
	keyNum uint32
}

// Get key manager
func (e2ekey *E2EKey) GetManager() *KeyManager {
	return e2ekey.manager
}

// Get key value (cyclic.Int)
func (e2ekey *E2EKey) GetKey() *cyclic.Int {
	return e2ekey.key
}

// Get key type, E2E or Rekey
func (e2ekey *E2EKey) GetOuterType() parse.CryptoType {
	return e2ekey.outer
}

// Generate key fingerprint
// NOTE: This function is not a getter,
// it returns a new byte array on each call
func (e2ekey *E2EKey) KeyFingerprint() format.Fingerprint {
	h, _ := hash.NewCMixHash()
	h.Write(e2ekey.key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))
	return fp
}
