///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/binary"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/primitives/fact"
	"math/rand"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// CreatePrecannedUser creates a precanned user
func CreatePrecannedUser(precannedID uint, rng csprng.Source) user.Info {

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
		E2eDhPrivateKey:  nil,
		E2eDhPublicKey:   nil,
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
	password []byte) (ReceptionIdentity, error) {
	jww.INFO.Printf("NewPrecannedClient()")
	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024,
		csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()

	def, err := ParseNDF(defJSON)
	if err != nil {
		return ReceptionIdentity{}, err
	}
	cmixGrp, e2eGrp := DecodeGroups(def)

	dhPrivKey := generatePrecanDHKeypair(precannedID, e2eGrp)

	userInfo := CreatePrecannedUser(precannedID, rngStream)
	identity, err := buildReceptionIdentity(userInfo.ReceptionID, userInfo.ReceptionSalt,
		userInfo.ReceptionRSA, e2eGrp, dhPrivKey)
	if err != nil {
		return ReceptionIdentity{}, err
	}

	store, err := CheckVersionAndSetupStorage(def, storageDir, password,
		userInfo, cmixGrp, e2eGrp, "")
	if err != nil {
		return ReceptionIdentity{}, err
	}

	// Mark the precanned user as finished with permissioning and registered
	// with the network.
	err = store.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return ReceptionIdentity{}, err
	}

	return identity, err
}

func generatePrecanDHKeypair(precannedID uint, e2eGrp *cyclic.Group) *cyclic.Int {
	// DH Keygen
	prng := rand.New(rand.NewSource(int64(precannedID)))
	prime := e2eGrp.GetPBytes()
	keyLen := len(prime)
	priv := diffieHellman.GeneratePrivateKey(keyLen, e2eGrp, prng)
	return priv
}

// Create an insecure e2e relationship with a precanned user
func (m *E2e) MakePrecannedAuthenticatedChannel(precannedID uint) (
	contact.Contact, error) {

	precan := m.MakePrecannedContact(precannedID)

	myID := binary.BigEndian.Uint64(m.GetStorage().GetReceptionID()[:])
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
	_, err := m.e2e.AddPartner(precan.ID, precan.DhPubKey,
		m.e2e.GetHistoricalDHPrivkey(), theirSIDHPubKey,
		mySIDHPrivKey, sesParam, sesParam)

	// check garbled messages in case any messages arrived before creating
	// the channel
	m.GetCmix().CheckInProgressMessages()

	return precan, err
}

// Create an insecure e2e contact object for a precanned user
func (m *E2e) MakePrecannedContact(precannedID uint) contact.Contact {

	e2eGrp := m.GetStorage().GetE2EGroup()

	rng := m.GetRng().GetStream()
	precanned := CreatePrecannedUser(precannedID, rng)
	rng.Close()

	precanned.E2eDhPrivateKey = generatePrecanDHKeypair(precannedID,
		m.GetStorage().GetE2EGroup())

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
