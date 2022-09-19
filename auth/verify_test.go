////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

// // Unit test.
// func TestVerifyOwnership(t *testing.T) {
// 	const numTests = 100

// 	grp := getGroup()
// 	prng := rand.New(rand.NewSource(69))

// 	for i := 0; i < numTests; i++ {
// 		// Generate mock keys
// 		myPrivKey := diffieHellman.GeneratePrivateKey(
// 			diffieHellman.DefaultPrivateKeyLength, grp, prng)
// 		partnerPubKey := diffieHellman.GeneratePublicKey(
// 			diffieHellman.GeneratePrivateKey(512, grp, prng), grp)

// 		// Init mock e2e Handler
// 		mockHandler := mockE2eHandler{
// 			privKey: myPrivKey,
// 		}

// 		proof := cAuth.MakeOwnershipProof(myPrivKey, partnerPubKey, grp)

// 		// Generate mock contact objects to pass in
// 		received := contact.Contact{
// 			OwnershipProof: proof,
// 		}
// 		verified := contact.Contact{
// 			DhPubKey: partnerPubKey,
// 		}

// 		// Call VerifyOwnership
// 		if !VerifyOwnership(received, verified, mockHandler) {
// 			t.Errorf("Proof could not be verified at index %v", i)
// 		}
// 	}
// }

// // Tests that bad proofs are not verified
// func TestVerifyOwnershipProof_Bad(t *testing.T) {

// 	const numTests = 100

// 	grp := getGroup()
// 	prng := rand.New(rand.NewSource(420))

// 	for i := 0; i < numTests; i++ {
// 		myPrivKey := diffieHellman.GeneratePrivateKey(
// 			diffieHellman.DefaultPrivateKeyLength, grp, prng)
// 		partnerPubKey := diffieHellman.GeneratePublicKey(
// 			diffieHellman.GeneratePrivateKey(512, grp, prng), grp)
// 		proof := make([]byte, 32)
// 		prng.Read(proof)

// 		// Generate mock contact objects to pass in
// 		received := contact.Contact{
// 			OwnershipProof: proof,
// 		}
// 		verified := contact.Contact{
// 			DhPubKey: partnerPubKey,
// 		}

// 		// Init mock e2e Handler
// 		mockHandler := mockE2eHandler{
// 			privKey: myPrivKey,
// 		}

// 		// Call VerifyOwnership
// 		if VerifyOwnership(received, verified, mockHandler) {
// 			t.Errorf("Proof was verified at index %v when it is bad", i)
// 		}

// 	}
// }
