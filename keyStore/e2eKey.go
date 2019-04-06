package keyStore

import (
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
	outer format.CryptoType
}

func (e2ekey *E2EKey) GetManager() *KeyManager {
	return e2ekey.manager
}

func (e2ekey *E2EKey) GetKey() *cyclic.Int {
	return e2ekey.key
}

func (e2ekey *E2EKey) GetOuterType() format.CryptoType {
	return e2ekey.outer
}

func (e2ekey *E2EKey) KeyFingerprint() format.Fingerprint {
	h, _ := hash.NewCMixHash()
	h.Write(e2ekey.key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))
	return fp
}
