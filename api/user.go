////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/xx"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type user struct {
	UID         *id.ID
	Salt        []byte
	RSAKey      *rsa.PrivateKey
	CMixKey     *cyclic.Int
	E2EKey      *cyclic.Int
	IsPrecanned bool
}

const (
	// SaltSize size of user salts
	SaltSize = 32
)

// createNewUser generates an identity for cMix
func createNewUser(rng csprng.Source, cmix, e2e *cyclic.Group) user {
	// RSA Keygen (4096 bit defaults)
	rsaKey, err := rsa.GenerateKey(rng, rsa.DefaultRSABitLen)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	// CMIX Keygen
	// FIXME: Why 256 bits? -- this is spec but not explained, it has
	// to do with optimizing operations on one side and still preserves
	// decent security -- cite this.
	cMixKeyBytes, err := csprng.GenerateInGroup(cmix.GetPBytes(), 256, rng)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	// DH Keygen
	// FIXME: Why 256 bits? -- this is spec but not explained, it has
	// to do with optimizing operations on one side and still preserves
	// decent security -- cite this. Why valid for BOTH e2e and cmix?
	e2eKeyBytes, err := csprng.GenerateInGroup(e2e.GetPBytes(), 256, rng)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	// Salt, UID, etc gen
	salt := make([]byte, SaltSize)
	n, err := csprng.NewSystemRNG().Read(salt)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
	if n != SaltSize {
		jww.FATAL.Panicf("salt size too small: %d", n)
	}
	userID, err := xx.NewID(rsaKey.GetPublic(), salt, id.User)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	return user{
		UID:         userID,
		Salt:        salt,
		RSAKey:      rsaKey,
		CMixKey:     cmix.NewIntFromBytes(cMixKeyBytes),
		E2EKey:      e2e.NewIntFromBytes(e2eKeyBytes),
		IsPrecanned: false,
	}
}

// TODO: Add precanned user code structures here.
