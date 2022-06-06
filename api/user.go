///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
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
func createNewUser(rng *fastRNG.StreamGenerator) user.Info {
	// CMIX Keygen
	var transmissionRsaKey, receptionRsaKey *rsa.PrivateKey
	var transmissionSalt, receptionSalt []byte

	transmissionSalt, receptionSalt,
		transmissionRsaKey, receptionRsaKey = createKeys(rng)

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
		E2eDhPrivateKey:  nil,
		E2eDhPublicKey:   nil,
	}
}

func createKeys(rng *fastRNG.StreamGenerator) (
	transmissionSalt, receptionSalt []byte,
	transmissionRsaKey, receptionRsaKey *rsa.PrivateKey) {
	wg := sync.WaitGroup{}

	wg.Add(2)

	// RSA Keygen (4096 bit defaults)
	go func() {
		defer wg.Done()
		var err error
		stream := rng.GetStream()
		transmissionRsaKey, err = rsa.GenerateKey(stream,
			rsa.DefaultRSABitLen)
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}
		transmissionSalt = make([]byte, 32)
		_, err = stream.Read(transmissionSalt)
		stream.Close()
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		stream := rng.GetStream()
		receptionRsaKey, err = rsa.GenerateKey(stream,
			rsa.DefaultRSABitLen)
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}
		receptionSalt = make([]byte, 32)
		_, err = stream.Read(receptionSalt)
		stream.Close()
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}
	}()
	wg.Wait()

	isZero := func(data []byte) bool {
		if len(data) == 0 {
			return true
		}
		for i := len(data) - 1; i != 0; i-- {
			if data[i] != 0 {
				return false
			}
		}
		return true
	}

	if isZero(receptionSalt) || isZero(transmissionSalt) {
		jww.FATAL.Panicf("empty salt generation detected")
	}
	return

}

// createNewVanityUser generates an identity for cMix
// The identity's ReceptionID is not random but starts with the supplied prefix
func createNewVanityUser(rng csprng.Source, cmix,
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
