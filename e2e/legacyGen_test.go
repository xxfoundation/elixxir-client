////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

// legacyGen_test.go contains the code for generating e2e relationships
// for the pre-April 2022 client. This is left here, commented for
// posterity and documentation purposes. This code was used to generate
// the legacyDataEkv directory. This data is tested in TestRatchet_unmarshalOld()

//
//func GenerateLegacyData() {
//	prng := rand.New(rand.NewSource(42))
//	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
//	privKey := grp.NewInt(57)
//	fs, err := ekv.NewFilestore("/home/josh/src/clientRelease/storage/e2e/legacyEkv", "hello")
//	if err != nil {
//		panic(
//			"Failed to create storage session")
//	}
//
//	kv := versioned.NewKV(fs)
//	//prng := rand.New(rand.NewSource(42))
//	myId := id.NewIdFromString("me", id.User, t)
//	s, err := NewStore(grp, kv, privKey, myId, prng)
//	if err != nil {
//		panic("NewStore() produced an error: " + err.Error())
//	}
//
//	partnerIds := make([]*id.ID, 0)
//	for i := 0; i < 5; i++ {
//		// Add 1 here cause 0 case: 0 is not within the group
//		partnerPubKey := diffieHellman.GeneratePublicKey(s.grp.NewInt(int64(i+1)), s.grp)
//
//		partnerID := id.NewIdFromUInt(uint64(i), id.User, t)
//		partnerIds = append(partnerIds, partnerID)
//		p := params.GetDefaultE2ESessionParams()
//		// NOTE: e2e store doesn't contain a private SIDH key, that's
//		// because they're completely ephemeral as part of the
//		// initiation of the connection.
//		_, pubSIDHKey := genSidhKeys(prng, sidh.KeyVariantSidhA)
//		privSIDHKey, _ := genSidhKeys(prng, sidh.KeyVariantSidhB)
//
//		err := s.AddPartner(partnerID, partnerPubKey, s.dhPrivateKey, pubSIDHKey,
//			privSIDHKey, p, p)
//		if err != nil {
//			panic("AddPartner returned an error: %v", err)
//		}
//	}
//
//}
//func genSidhKeys(rng io.Reader, variant sidh.KeyVariant) (*sidh.PrivateKey, *sidh.PublicKey) {
//	sidHPrivKey := util.NewSIDHPrivateKey(variant)
//	sidHPubKey := util.NewSIDHPublicKey(variant)
//
//	if err := sidHPrivKey.Generate(rng); err != nil {
//		panic("failure to generate SidH A private key")
//	}
//	sidHPrivKey.GeneratePublicKey(sidHPubKey)
//
//	return sidHPrivKey, sidHPubKey
//}
