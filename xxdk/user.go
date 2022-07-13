///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/binary"
	"math/rand"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"gitlab.com/elixxir/crypto/diffieHellman"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

const (
	// SaltSize size of user salts
	SaltSize = 32
)

// createNewUser generates an identity for cMix
func createNewUser(rng *fastRNG.StreamGenerator, e2eGroup *cyclic.Group) user.Info {
	// CMIX Keygen
	var transmissionRsaKey, receptionRsaKey *rsa.PrivateKey
	var transmissionSalt, receptionSalt []byte

	e2eKeyBytes, transmissionSalt, receptionSalt,
		transmissionRsaKey, receptionRsaKey := createDhKeys(rng, e2eGroup)

	transmissionID, err := xx.NewID(transmissionRsaKey.GetPublic(),
		transmissionSalt, id.User)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	receptionID, err := xx.NewID(receptionRsaKey.GetPublic(),
		receptionSalt, id.User)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	return user.Info{
		TransmissionID:   transmissionID.DeepCopy(),
		TransmissionSalt: transmissionSalt,
		TransmissionRSA:  transmissionRsaKey,
		ReceptionID:      receptionID.DeepCopy(),
		ReceptionSalt:    receptionSalt,
		ReceptionRSA:     receptionRsaKey,
		Precanned:        false,
		E2eDhPrivateKey:  e2eGroup.NewIntFromBytes(e2eKeyBytes),
		E2eDhPublicKey:   nil,
	}
}

func createDhKeys(rng *fastRNG.StreamGenerator,
	e2e *cyclic.Group) (e2eKeyBytes,
	transmissionSalt, receptionSalt []byte,
	transmissionRsaKey, receptionRsaKey *rsa.PrivateKey) {
	wg := sync.WaitGroup{}

	wg.Add(3)

	go func() {
		defer wg.Done()
		var err error
		// DH Keygen
		// FIXME: Why 256 bits? -- this is spec but not explained, it has
		// to do with optimizing operations on one side and still preserves
		// decent security -- cite this. Why valid for BOTH e2e and cmix?
		stream := rng.GetStream()
		e2eKeyBytes, err = csprng.GenerateInGroup(e2e.GetPBytes(), 256, stream)
		stream.Close()
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}
	}()

	// RSA Keygen (4096 bit defaults)
	go func() {
		defer wg.Done()
		var err error
		stream := rng.GetStream()
		transmissionRsaKey, err = rsa.GenerateKey(stream, rsa.DefaultRSABitLen)
		stream.Close()
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		stream := rng.GetStream()
		receptionRsaKey, err = rsa.GenerateKey(stream, rsa.DefaultRSABitLen)
		stream.Close()
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}
	}()
	wg.Wait()

	return

}

// createNewVanityUser generates an identity for cMix
// The identity's ReceptionID is not random but starts with the supplied prefix
func createNewVanityUser(rng csprng.Source,
	e2e *cyclic.Group, prefix string) user.Info {
	// DH Keygen
	prime := e2e.GetPBytes()
	keyLen := len(prime)

	e2eKey := diffieHellman.GeneratePrivateKey(keyLen, e2e, rng)

	// RSA Keygen (4096 bit defaults)
	transmissionRsaKey, err := rsa.GenerateKey(rng, rsa.DefaultRSABitLen)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	// Salt, UID, etc gen
	transmissionSalt := make([]byte, SaltSize)
	n, err := csprng.NewSystemRNG().Read(transmissionSalt)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
	if n != SaltSize {
		jww.FATAL.Panicf("transmissionSalt size too small: %d", n)
	}
	transmissionID, err := xx.NewID(transmissionRsaKey.GetPublic(),
		transmissionSalt, id.User)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	receptionRsaKey, err := rsa.GenerateKey(rng, rsa.DefaultRSABitLen)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	// just in case more than one go routine tries to access
	// receptionSalt and receptionID
	var mu sync.Mutex
	done := make(chan struct{})
	found := make(chan bool)
	wg := &sync.WaitGroup{}
	cores := runtime.NumCPU()

	var receptionSalt []byte
	var receptionID *id.ID

	pref := prefix
	ignoreCase := false
	// check if case-insensitivity is enabled
	if strings.HasPrefix(prefix, "(?i)") {
		pref = strings.ToLower(pref[4:])
		ignoreCase = true
	}
	// Check if prefix contains valid Base64 characters
	match, _ := regexp.MatchString("^[A-Za-z0-9+/]+$", pref)
	if match == false {
		jww.FATAL.Panicf("Prefix contains non-Base64 characters")
	}
	jww.INFO.Printf("Vanity userID generation started. Prefix: %s "+
		"Ignore-Case: %v NumCPU: %d", pref, ignoreCase, cores)
	for w := 0; w < cores; w++ {
		wg.Add(1)
		go func() {
			rSalt := make([]byte, SaltSize)
			for {
				select {
				case <-done:
					defer wg.Done()
					return
				default:
					n, err = csprng.NewSystemRNG().Read(
						rSalt)
					if err != nil {
						jww.FATAL.Panicf(err.Error())
					}
					if n != SaltSize {
						jww.FATAL.Panicf(
							"receptionSalt size "+
								"too small: %d",
							n)
					}
					rID, err := xx.NewID(
						receptionRsaKey.GetPublic(),
						rSalt, id.User)
					if err != nil {
						jww.FATAL.Panicf(err.Error())
					}
					id := rID.String()
					if ignoreCase {
						id = strings.ToLower(id)
					}
					if strings.HasPrefix(id, pref) {
						mu.Lock()
						receptionID = rID
						receptionSalt = rSalt
						mu.Unlock()
						found <- true
						defer wg.Done()
						return
					}
				}
			}
		}()
	}
	// wait for a solution then close the done channel to signal
	// the workers to exit
	<-found
	close(done)
	wg.Wait()
	return user.Info{
		TransmissionID:   transmissionID.DeepCopy(),
		TransmissionSalt: transmissionSalt,
		TransmissionRSA:  transmissionRsaKey,
		ReceptionID:      receptionID.DeepCopy(),
		ReceptionSalt:    receptionSalt,
		ReceptionRSA:     receptionRsaKey,
		Precanned:        false,
		E2eDhPrivateKey:  e2eKey,
		E2eDhPublicKey:   diffieHellman.GeneratePublicKey(e2eKey, e2e),
	}
}

// createPrecannedUser
func createPrecannedUser(precannedID uint, rng csprng.Source, e2e *cyclic.Group) user.Info {
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

	prime := e2e.GetPBytes()
	keyLen := len(prime)
	dhPrivKey := diffieHellman.GeneratePrivateKey(keyLen, e2e, prng)
	return user.Info{
		TransmissionID:   &userID,
		TransmissionSalt: salt,
		ReceptionID:      &userID,
		ReceptionSalt:    salt,
		Precanned:        true,
		E2eDhPrivateKey:  dhPrivKey,
		E2eDhPublicKey:   e2e.ExpG(dhPrivKey, e2e.NewInt(1)),
		TransmissionRSA:  rsaKey,
		ReceptionRSA:     rsaKey,
	}
}
