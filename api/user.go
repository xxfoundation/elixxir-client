///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

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
	"regexp"
	"runtime"
	"strings"
	"sync"
)

const (
	// SaltSize size of user salts
	SaltSize = 32
)

// createNewUser generates an identity for cMix
func createNewUser(rng csprng.Source, cmix, e2e *cyclic.Group) user.User {
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

	// RSA Keygen (4096 bit defaults)
	transmissionRsaKey, err := rsa.GenerateKey(rng, rsa.DefaultRSABitLen)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
	receptionRsaKey, err := rsa.GenerateKey(rng, rsa.DefaultRSABitLen)
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
	transmissionID, err := xx.NewID(transmissionRsaKey.GetPublic(), transmissionSalt, id.User)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	// Salt, UID, etc gen
	receptionSalt := make([]byte, SaltSize)
	n, err = csprng.NewSystemRNG().Read(receptionSalt)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
	if n != SaltSize {
		jww.FATAL.Panicf("receptionSalt size too small: %d", n)
	}
	receptionID, err := xx.NewID(receptionRsaKey.GetPublic(), receptionSalt, id.User)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	return user.User{
		TransmissionID:   transmissionID.DeepCopy(),
		TransmissionSalt: transmissionSalt,
		TransmissionRSA:  transmissionRsaKey,
		ReceptionID:      receptionID.DeepCopy(),
		ReceptionSalt:    receptionSalt,
		ReceptionRSA:     receptionRsaKey,
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
		TransmissionID:   &userID,
		TransmissionSalt: salt,
		ReceptionID:      &userID,
		ReceptionSalt:    salt,
		Precanned:        true,
		E2eDhPrivateKey:  e2e.NewIntFromBytes(e2eKeyBytes),
		// NOTE: These are dummy/not used
		CmixDhPrivateKey: cmix.NewInt(1),
		TransmissionRSA:  rsaKey,
		ReceptionRSA:     rsaKey,
	}
}

// createNewVanityUser generates an identity for cMix
// The identity's ReceptionID is not random but starts with the supplied prefix 
func createNewVanityUser(rng csprng.Source, cmix, e2e *cyclic.Group, prefix string) user.User {
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
	transmissionID, err := xx.NewID(transmissionRsaKey.GetPublic(), transmissionSalt, id.User)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	receptionRsaKey, err := rsa.GenerateKey(rng, rsa.DefaultRSABitLen)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	var mu sync.Mutex // just in case more than one go routine tries to access receptionSalt and receptionID
	done := make(chan struct{}) 
	found:= make(chan bool)
	wg:= &sync.WaitGroup{}
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
	jww.INFO.Printf("Vanity userID generation started. Prefix: %s Ignore-Case: %v NumCPU: %d", pref, ignoreCase, cores)
	for w := 0; w < cores; w++{
		wg.Add(1)
		go func() {
			rSalt := make([]byte, SaltSize)
			for {	
				select {
				case <- done:
					defer wg.Done()
					return
				default:
					n, err = csprng.NewSystemRNG().Read(rSalt)
					if err != nil {
						jww.FATAL.Panicf(err.Error())
					}
					if n != SaltSize {
						jww.FATAL.Panicf("receptionSalt size too small: %d", n)
					}
					rID, err := xx.NewID(receptionRsaKey.GetPublic(), rSalt, id.User)
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
	// wait for a solution then close the done channel to signal the workers to exit
	<- found
	close(done)
	wg.Wait()
	return user.User{
		TransmissionID:   transmissionID.DeepCopy(),
		TransmissionSalt: transmissionSalt,
		TransmissionRSA:  transmissionRsaKey,
		ReceptionID:      receptionID.DeepCopy(),
		ReceptionSalt:    receptionSalt,
		ReceptionRSA:     receptionRsaKey,
		Precanned:        false,
		CmixDhPrivateKey: cmix.NewIntFromBytes(cMixKeyBytes),
		E2eDhPrivateKey:  e2e.NewIntFromBytes(e2eKeyBytes),
	}
}
