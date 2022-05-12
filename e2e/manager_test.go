package e2e

import (
	"bytes"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/e2e/rekey"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// todo: come up with better name and add docstring
type legacyPartnerData struct {
	partnerId         *id.ID
	myDhPubKey        *cyclic.Int
	mySidhPrivKey     *sidh.PrivateKey
	partnerSidhPubKey *sidh.PublicKey
	sendFP            []byte
	receiveFp         []byte
}

// TestRatchet_unmarshalOld tests the loading of legacy data
// following an EKV storage structure prior to the April 2022 client
// restructure. It tests that this data is loaded and transferred to the
// current design appropriately. For this test, there are some
// hardcoded base64 encoded data in this file that represents
// data marshalled according to the previous design.
func TestLoadLegacy(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	myPrivKey := grp.NewInt(57)
	myPubKey := diffieHellman.GeneratePublicKey(myPrivKey, grp)
	myId := id.NewIdFromString("me", id.User, t)

	prng := rand.New(rand.NewSource(42))
	numTest := 5
	legacyData := make([]legacyPartnerData, 0, numTest)
	// Expected data generation. This mocks up how partner information was
	// generated using the old design (refer to legacyGen_test.go)
	for i := 0; i < numTest; i++ {
		partnerID := id.NewIdFromUInt(uint64(i), id.User, t)
		partnerPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(int64(i+1)), grp)

		// Note this sidh key generation code comes from a testing helper
		// function genSidhKeys which at the time of writing exists in both
		// the pre-April 2022 refactor and the post-refactor. It has been
		// hardcoded here to preserve the original implementation. It is
		// possible a refactor occurs on the existing helper functions
		// which breaks what is attempted to be preserved in this test.

		// Generate public key (we do not care about the private key, as that
		// will not be in storage as it represents the partner's private key,
		// which we would not know)
		variant := sidh.KeyVariantSidhA
		partnerPrivKey := util.NewSIDHPrivateKey(variant)
		partnerSidHPubKey := util.NewSIDHPublicKey(variant)

		if err := partnerPrivKey.Generate(prng); err != nil {
			t.Fatalf("failure to generate SidH A private key")
		}
		partnerPrivKey.GeneratePublicKey(partnerSidHPubKey)

		// Generate a separate private key. This represents out private key,
		// which we do know.
		variant = sidh.KeyVariantSidhB

		mySidHPrivKey := util.NewSIDHPrivateKey(variant)
		if err := partnerPrivKey.Generate(prng); err != nil {
			t.Fatalf("failure to generate SidH B private key")
		}

		d := legacyPartnerData{
			partnerId:         partnerID,
			myDhPubKey:        myPubKey,
			mySidhPrivKey:     mySidHPrivKey,
			partnerSidhPubKey: partnerSidHPubKey,
			// Fixme: if the underlying crypto implementation ever changes, this will break
			//  the legacy loading tests
			sendFP: e2e.MakeRelationshipFingerprint(myPubKey, partnerPubKey,
				myId, partnerID),
			receiveFp: e2e.MakeRelationshipFingerprint(myPubKey, partnerPubKey,
				partnerID, myId),
		}

		legacyData = append(legacyData, d)

	}

	// Construct kv with legacy data
	// fs, err := ekv.NewFilestore("/home/josh/src/client/e2e/legacyEkv", "hello")
	// if err != nil {
	//	t.Fatalf(
	//		"Failed to create storage session: %+v", err)
	// }
	kv := versioned.NewKV(ekv.MakeMemstore())

	err := ratchet.New(kv, myId, myPrivKey, grp)
	if err != nil {
		t.Errorf("Failed to init ratchet: %+v", err)
	}

	rng := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)

	// Load legacy data
	h, err := LoadLegacy(kv, newMockCmix(nil, nil, t), myId,
		grp, rng, mockEventsManager{}, rekey.GetDefaultParams())
	if err != nil {
		t.Fatalf("LoadLegacy error: %v", err)
	}

	// Parse handler for expected partners
	for _, legacyPartner := range legacyData {
		partnerManager, err := h.GetPartner(legacyPartner.partnerId)
		if err != nil {
			t.Errorf("Partner %s does not exist in handler.", legacyPartner.partnerId)
		} else {
			if !bytes.Equal(partnerManager.SendRelationshipFingerprint(), legacyPartner.sendFP) {
				t.Fatalf("Send relationship fingerprint pulled from legacy does not match expected data."+
					"\nExpected: %v"+
					"\nReceived: %v", legacyPartner.sendFP, partnerManager.SendRelationshipFingerprint())
			}

			if !bytes.Equal(partnerManager.ReceiveRelationshipFingerprint(), legacyPartner.receiveFp) {
				t.Fatalf("Receive relationship fingerprint pulled from legacy does not match expected data."+
					"\nExpected: %v"+
					"\nReceived: %v", legacyPartner.sendFP, partnerManager.SendRelationshipFingerprint())
			}
		}
	}
}
