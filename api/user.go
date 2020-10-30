package api

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
)

const (
	// SaltSize size of user salts
	SaltSize = 32
)

// createNewUser generates an identity for cMix
func createNewUser(rng csprng.Source, cmix, e2e *cyclic.Group) user.User {
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

	return user.User{
		ID:               userID.DeepCopy(),
		Salt:             salt,
		RSA:              rsaKey,
		Precanned:        false,
		CmixDhPrivateKey: cmix.NewIntFromBytes(cMixKeyBytes),
		E2eDhPrivateKey:  e2e.NewIntFromBytes(e2eKeyBytes),
	}
}

// TODO: Add precanned user code structures here.
// creates a precanned user
func createPrecannedUser(precannedID uint, rng csprng.Source, cmix, e2e *cyclic.Group) user.User {
	// DH Keygen
	// FIXME: Why 256 bits? -- this is spec but not explained, it has
	// to do with optimizing operations on one side and still preserves
	// decent security -- cite this. Why valid for BOTH e2e and cmix?
	prng := rand.New(rand.NewSource(int64(precannedID)))
	e2eKeyBytes, err := csprng.GenerateInGroup(e2e.GetPBytes(), 256, prng)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	// Salt, UID, etc gen
	salt := make([]byte, SaltSize)

	userID := id.ID{}
	binary.BigEndian.PutUint64(userID[:], uint64(precannedID))
	userID.SetType(id.User)

	// NOTE: not used... RSA Keygen (4096 bit defaults)
	rsaKey, err := rsa.GenerateKey(rng, rsa.DefaultRSABitLen)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	return user.User{
		ID:              userID.DeepCopy(),
		Salt:            salt,
		Precanned:       true,
		E2eDhPrivateKey: e2e.NewIntFromBytes(e2eKeyBytes),
		// NOTE: These are dummy/not used
		CmixDhPrivateKey: cmix.NewInt(1),
		RSA:              rsaKey,
	}
}
