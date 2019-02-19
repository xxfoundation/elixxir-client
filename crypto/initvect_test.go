////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"testing"
)

// Smoke test for MakeInitVect
func TestMakeInitVect(t *testing.T) {
	InitCrypto()
	tests := 100
	min := cyclic.NewInt(2)
	max := cyclic.NewIntFromString("7FFFFFFFFFFFFFFFFF", 16)
	for i := 0; i < tests; i++ {
		rand := cyclic.NewIntFromBytes(MakeInitVect(make([]byte, format.RIV_LEN)))
		if rand.Cmp(min) < 0 || rand.Cmp(max) >= 0 {
			t.Error("MakeInitVector is out of range.")
		}
	}
}
