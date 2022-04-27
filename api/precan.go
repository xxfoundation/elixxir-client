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

	"github.com/cloudflare/circl/dh/sidh"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
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

// Create an insecure e2e relationship with a precanned user
func (c *Client) MakePrecannedAuthenticatedChannel(precannedID uint) (
	contact.Contact, error) {

	precan := c.MakePrecannedContact(precannedID)

	myID := binary.BigEndian.Uint64(c.GetUser().GetContact().ID[:])
	// Pick a variant based on if their ID is bigger than mine.
	myVariant := sidh.KeyVariantSidhA
	theirVariant := sidh.KeyVariant(sidh.KeyVariantSidhB)
	if myID > uint64(precannedID) {
		myVariant = sidh.KeyVariantSidhB
		theirVariant = sidh.KeyVariantSidhA
	}
	prng1 := rand.New(rand.NewSource(int64(precannedID)))
	theirSIDHPrivKey := util.NewSIDHPrivateKey(theirVariant)
	theirSIDHPubKey := util.NewSIDHPublicKey(theirVariant)
	theirSIDHPrivKey.Generate(prng1)
	theirSIDHPrivKey.GeneratePublicKey(theirSIDHPubKey)

	prng2 := rand.New(rand.NewSource(int64(myID)))
	mySIDHPrivKey := util.NewSIDHPrivateKey(myVariant)
	mySIDHPubKey := util.NewSIDHPublicKey(myVariant)
	mySIDHPrivKey.Generate(prng2)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	// add the precanned user as a e2e contact
	// FIXME: these params need to be threaded through...
	sesParam := session.GetDefaultParams()
	_, err := c.e2e.AddPartner(precan.ID, precan.DhPubKey,
		c.e2e.GetHistoricalDHPrivkey(), theirSIDHPubKey,
		mySIDHPrivKey, sesParam, sesParam)

	// check garbled messages in case any messages arrived before creating
	// the channel
	c.network.CheckInProgressMessages()

	return precan, err
}

// Create an insecure e2e contact object for a precanned user
func (c *Client) MakePrecannedContact(precannedID uint) contact.Contact {

	e2eGrp := c.storage.GetE2EGroup()

	precanned := createPrecannedUser(precannedID, c.rng.GetStream(),
		c.storage.GetCmixGroup(), e2eGrp)

	// compute their public e2e key
	partnerPubKey := e2eGrp.ExpG(precanned.E2eDhPrivateKey,
		e2eGrp.NewInt(1))

	return contact.Contact{
		ID:             precanned.ReceptionID,
		DhPubKey:       partnerPubKey,
		OwnershipProof: nil,
		Facts:          make([]fact.Fact, 0),
	}
}
