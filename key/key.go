package key

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
	jww "github.com/spf13/jwalterweatherman"
)

type Key struct {
	// Links
	session *Session

	fp *format.Fingerprint

	// Designation of crypto type
	outer parse.CryptoType

	// keyNum is the index of the key by order of creation
	// it is used to identify the key in the key.Session
	keyNum uint32
}

func newKey(session *Session, key *cyclic.Int, outer parse.CryptoType, keynum uint32) *Key {
	return &Key{
		session: session,
		key:     key,
		outer:   outer,
		keyNum:  keynum,
	}
}

// return pointers to higher level management structures
func (k *Key) GetSession() *Session { return k.session }

// Get key value (cyclic.Int)
func (k *Key) GetKey() *cyclic.Int { return k.key }

// Get key type, E2E or Rekey
func (k *Key) GetOuterType() parse.CryptoType { return k.outer }

// Generate key fingerprint
// NOTE: This function is not a getter,
// it returns a new byte array on each call
func (k *Key) Fingerprint() format.Fingerprint {
	h, _ := hash.NewCMixHash()
	h.Write(k.key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))
	return fp
}

// Sets the key as used
func (k *Key) denoteUse() error {
	switch k.outer {
	case parse.E2E:
		err := k.session.useKey(k.keyNum)
		if err != nil {
			return errors.WithMessage(err, "Could not use e2e key")
		}

	case parse.Rekey:
		err := k.session.useReKey(k.keyNum)
		if err != nil {
			return errors.WithMessage(err, "Could not use e2e rekey")
		}
	default:
		jww.FATAL.Panicf("Key has invalid cryptotype: %s", k.outer)
	}
	return nil
}