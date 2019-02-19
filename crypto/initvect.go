////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"gitlab.com/elixxir/crypto/cyclic"
)

var vectGen *cyclic.Random

// MakeInitVect creates a random string for the initialization vector with lazy
// creation of the underlying random number generator
func MakeInitVect(v []byte) []byte {
	if vectGen == nil {
		min := cyclic.NewInt(2)
		max := cyclic.NewInt(0).Exp(cyclic.NewInt(2), cyclic.NewInt(71),
			Grp.GetP(cyclic.NewInt(0)))
		max = max.Sub(max, cyclic.NewInt(1))

		v := cyclic.NewRandom(min, max)

		vectGen = &v
	}
	initVect := vectGen.Rand(cyclic.NewInt(0))
	copy(v, initVect.LeftpadBytes(uint64(len(v))))

	return v
}
