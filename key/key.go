package key

import (
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

type Key struct {
	// Links
	session *Session

	// Key to be used
	key *cyclic.Int

	// Designation of crypto type
	outer parse.CryptoType

	// keyNum is the index of the key by order of creation
	// it is used to identify the key in the key.Session
	keyNum uint32
}

func Operate(func(s *Session, )) error

// return pointers to higher level management structures
func (k *Key) GetSession() *Session { return k.session }

// Get key value (cyclic.Int)
func (k *Key) GetKey() *cyclic.Int { return k.key }

// Get key type, E2E or Rekey
func (k *Key) GetOuterType() parse.CryptoType { return k.outer }

// Generate key fingerprint
// NOTE: This function is not a getter,
// it returns a new byte array on each call
func (k *Key) KeyFingerprint() format.Fingerprint {
	h, _ := hash.NewCMixHash()
	h.Write(k.key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))
	return fp
}
