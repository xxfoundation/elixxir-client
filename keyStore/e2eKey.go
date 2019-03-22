package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
)

type E2EKey struct {
	// Link to source key
	Source *KeyLifecycle

	// Key to be used
	Key *cyclic.Int

	// Designation of outer type
	Outer format.OuterType
}
