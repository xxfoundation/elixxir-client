package e2eApi

import (
	"encoding/binary"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/primitives/fact"
	"math/rand"
)

func generatePrecanDHKeypair(precannedID uint, e2eGrp *cyclic.Group) *cyclic.Int {
	// DH Keygen
	prng := rand.New(rand.NewSource(int64(precannedID)))
	prime := e2eGrp.GetPBytes()
	keyLen := len(prime)
	priv := diffieHellman.GeneratePrivateKey(keyLen, e2eGrp, prng)
	return priv
}

// Create an insecure e2e relationship with a precanned user
func (m *Client) MakePrecannedAuthenticatedChannel(precannedID uint) (
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
func (m *Client) MakePrecannedContact(precannedID uint) contact.Contact {

	e2eGrp := m.GetStorage().GetE2EGroup()

	rng := m.GetRng().GetStream()
	precanned := api.CreatePrecannedUser(precannedID, rng)
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
