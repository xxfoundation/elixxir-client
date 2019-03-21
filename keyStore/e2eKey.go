package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
)

type E2EKey struct {
	// Link to source key
	source *KeyLifecycle

	// Key to be used
	key *cyclic.Int

	// Designation of outer type
	outer format.OuterType
}

func (e2ekey *E2EKey) GetSource() *KeyLifecycle {
	return e2ekey.source
}

func (E2EKey *E2EKey) GetKey() *cyclic.Int {
	return cyclic.NewIntFromBytes(E2EKey.key.Bytes())
}

func (E2EKey *E2EKey) GetOuterType() *format.OuterType {
	return E2EKey.outer
}
