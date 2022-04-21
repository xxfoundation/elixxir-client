///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"encoding/binary"
	"math/rand"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// creates a precanned user
func createPrecannedUser(precannedID uint, rng csprng.Source, cmix,
	e2e *cyclic.Group) user.Info {
	// DH Keygen
	prng := rand.New(rand.NewSource(int64(precannedID)))
	prime := e2e.GetPBytes()
	keyLen := len(prime)
	e2eKeyBytes, err := csprng.GenerateInGroup(prime, keyLen, prng)
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

	return user.Info{
		TransmissionID:   &userID,
		TransmissionSalt: salt,
		ReceptionID:      &userID,
		ReceptionSalt:    salt,
		Precanned:        true,
		E2eDhPrivateKey:  e2e.NewIntFromBytes(e2eKeyBytes),
		TransmissionRSA:  rsaKey,
		ReceptionRSA:     rsaKey,
	}
}

// NewPrecannedClient creates an insecure user with predetermined keys
// with nodes It creates client storage, generates keys, connects, and
// registers with the network. Note that this does not register a
// username/identity, but merely creates a new cryptographic identity
// for adding such information at a later date.
func NewPrecannedClient(precannedID uint, defJSON, storageDir string,
	password []byte) error {
	jww.INFO.Printf("NewPrecannedClient()")
	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024,
		csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()

	def, err := parseNDF(defJSON)
	if err != nil {
		return err
	}
	cmixGrp, e2eGrp := decodeGroups(def)

	protoUser := createPrecannedUser(precannedID, rngStream,
		cmixGrp, e2eGrp)

	store, err := checkVersionAndSetupStorage(def, storageDir, password,
		protoUser, cmixGrp, e2eGrp, rngStreamGen, "")
	if err != nil {
		return err
	}

	// Mark the precanned user as finished with permissioning and registered
	// with the network.
	err = store.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return err
	}

	return nil
}
