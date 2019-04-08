package keyStore

import (
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

func cryptoTypePrint(typ format.CryptoType) string {
	var ret string
	switch typ {
	case format.None:
		ret = "None"
	case format.Unencrypted:
		ret = "Unencrypted"
	case format.E2E:
		ret = "E2E"
	case format.Garbled:
		ret = "Garbled"
	case format.Error:
		ret = "Error"
	case format.Rekey:
		ret = "Rekey"
	}
	return ret
}

// initGroup sets up the cryptographic constants for cMix
func initGroup() *cyclic.Group {

	base := 16

	pString := "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
		"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
		"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
		"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
		"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
		"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
		"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
		"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"

	gString := "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
		"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
		"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
		"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
		"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
		"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
		"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
		"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"

	qString := "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F"

	p := large.NewIntFromString(pString, base)
	g := large.NewIntFromString(gString, base)
	q := large.NewIntFromString(qString, base)

	grp := cyclic.NewGroup(p, g, q)

	return grp
}

// Test RegisterPartner correctly creates keys and adds them to maps
func TestRegisterPartner(t *testing.T) {
	grp := initGroup()
	userID := id.NewUserFromUint(18, t)
	partner := id.NewUserFromUint(14, t)
	params := signature.CustomDSAParams(
		grp.GetP(),
		grp.GetG(),
		grp.GetQ())
	rng := csprng.NewSystemRNG()
	myPrivKey := params.PrivateKeyGen(rng)
	myPrivKeyCyclic := grp.NewIntFromLargeInt(myPrivKey.GetKey())
	myPubKey := myPrivKey.PublicKeyGen()
	myPubKeyCyclic := grp.NewIntFromLargeInt(myPubKey.GetKey())
	partnerPrivKey := params.PrivateKeyGen(rng)
	partnerPubKey := partnerPrivKey.PublicKeyGen()

	myUser := &user.User{User: userID, Nick: "test"}
	session := user.NewSession(myUser, "", []user.NodeKeys{},
		myPrivKeyCyclic, myPubKeyCyclic, grp)

	user.TheSession = session

	RegisterPartner(partner, partnerPubKey)

	// Confirm we can get all types of keys
	key, action := TransmissionKeys.Pop(partner)
	if key == nil {
		t.Errorf("TransmissionKeys map returned nil")
	} else if key.GetOuterType() != format.E2E {
		t.Errorf("Key type expected 'E2E', got %s",
			cryptoTypePrint(key.GetOuterType()))
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	key, action = TransmissionReKeys.Pop(partner)
	if key == nil {
		t.Errorf("TransmissionReKeys map returned nil")
	} else if key.GetOuterType() != format.Rekey {
		t.Errorf("Key type expected 'Rekey', got %s",
			cryptoTypePrint(key.GetOuterType()))
	} else if action != None {
		t.Errorf("Expected 'None' action, got %s instead",
			actionPrint(action))
	}

	km := key.GetManager()

	key = ReceptionKeys.Pop(km.receiveKeysFP[0])
	if key == nil {
		t.Errorf("ReceptionKeys map returned nil for Key")
	} else if key.GetOuterType() != format.E2E {
		t.Errorf("Key type expected 'E2E', got %s",
			cryptoTypePrint(key.GetOuterType()))
	}

	key = ReceptionKeys.Pop(km.receiveReKeysFP[0])
	if key == nil {
		t.Errorf("ReceptionKeys map returned nil for ReKey")
	} else if key.GetOuterType() != format.Rekey {
		t.Errorf("Key type expected 'Rekey', got %s",
			cryptoTypePrint(key.GetOuterType()))
	}
}

// Test all keys created with RegisterPartner match what is expected
func TestRegisterPartner_CheckAllKeys(t *testing.T) {
	grp := initGroup()
	userID := id.NewUserFromUint(18, t)
	partner := id.NewUserFromUint(14, t)
	params := signature.CustomDSAParams(
		grp.GetP(),
		grp.GetG(),
		grp.GetQ())
	rng := csprng.NewSystemRNG()
	myPrivKey := params.PrivateKeyGen(rng)
	myPrivKeyCyclic := grp.NewIntFromLargeInt(myPrivKey.GetKey())
	myPubKey := myPrivKey.PublicKeyGen()
	myPubKeyCyclic := grp.NewIntFromLargeInt(myPubKey.GetKey())
	partnerPrivKey := params.PrivateKeyGen(rng)
	partnerPubKey := partnerPrivKey.PublicKeyGen()
	partnerPubKeyCyclic := grp.NewIntFromLargeInt(partnerPubKey.GetKey())

	myUser := &user.User{User: userID, Nick: "test"}
	session := user.NewSession(myUser, "", []user.NodeKeys{},
		myPrivKeyCyclic, myPubKeyCyclic, grp)

	user.TheSession = session

	RegisterPartner(partner, partnerPubKey)

	// Generate all keys and confirm they all match
	baseKey, _ := diffieHellman.CreateDHSessionKey(partnerPubKeyCyclic, myPrivKeyCyclic, grp)
	keyTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(), minKeys, maxKeys,
		e2e.TTLParams{TTLScalar: ttlScalar, MinNumKeys: threshold})

	sendKeys := e2e.DeriveKeys(grp, baseKey, userID, uint(numKeys))
	sendReKeys := e2e.DeriveEmergencyKeys(grp, baseKey, userID, uint(numReKeys))
	recvKeys := e2e.DeriveKeys(grp, baseKey, partner, uint(numKeys))
	recvReKeys := e2e.DeriveEmergencyKeys(grp, baseKey, partner, uint(numReKeys))

	// Confirm all keys
	for i := 0; i < int(numKeys); i++ {
		key, action := TransmissionKeys.Pop(partner)
		if key == nil {
			t.Errorf("TransmissionKeys map returned nil")
		} else if key.GetOuterType() != format.E2E {
			t.Errorf("Key type expected 'E2E', got %s",
				cryptoTypePrint(key.GetOuterType()))
		}

		if i < int(keyTTL-1) {
			if action != None {
				t.Errorf("Expected 'None' action, got %s instead",
					actionPrint(action))
			}
		} else {
			if action != Rekey {
				t.Errorf("Expected 'Rekey' action, got %s instead",
					actionPrint(action))
			}
		}

		if key.GetKey().Cmp(sendKeys[int(numKeys)-1-i]) != 0 {
			t.Errorf("Key value expected %s, got %s",
				sendKeys[int(numKeys)-1-i].Text(10),
				key.GetKey().Text(10))
		}
	}

	for i := 0; i < int(numReKeys); i++ {
		key, action := TransmissionReKeys.Pop(partner)
		if key == nil {
			t.Errorf("TransmissionReKeys map returned nil")
		} else if key.GetOuterType() != format.Rekey {
			t.Errorf("Key type expected 'Rekey', got %s",
				cryptoTypePrint(key.GetOuterType()))
		}

		if i < int(numReKeys-1) {
			if action != None {
				t.Errorf("Expected 'None' action, got %s instead",
					actionPrint(action))
			}
		} else {
			if action != Purge {
				t.Errorf("Expected 'Purge' action, got %s instead",
					actionPrint(action))
			}
		}

		if key.GetKey().Cmp(sendReKeys[int(numReKeys)-1-i]) != 0 {
			t.Errorf("Key value expected %s, got %s",
				sendReKeys[int(numReKeys)-1-i].Text(10),
				key.GetKey().Text(10))
		}
	}

	for i := 0; i < int(numKeys); i++ {
		e2ekey := new(E2EKey)
		e2ekey.key = recvKeys[i]
		fp := e2ekey.KeyFingerprint()
		key := ReceptionKeys.Pop(fp)
		if key == nil {
			t.Errorf("ReceptionKeys map returned nil for Key")
		} else if key.GetOuterType() != format.E2E {
			t.Errorf("Key type expected 'E2E', got %s",
				cryptoTypePrint(key.GetOuterType()))
		}

		if key.GetKey().Cmp(recvKeys[i]) != 0 {
			t.Errorf("Key value expected %s, got %s",
				recvKeys[i].Text(10),
				key.GetKey().Text(10))
		}
	}

	for i := 0; i < int(numReKeys); i++ {
		e2ekey := new(E2EKey)
		e2ekey.key = recvReKeys[i]
		fp := e2ekey.KeyFingerprint()
		key := ReceptionKeys.Pop(fp)
		if key == nil {
			t.Errorf("ReceptionKeys map returned nil for Rekey")
		} else if key.GetOuterType() != format.Rekey {
			t.Errorf("Key type expected 'Rekey', got %s",
				cryptoTypePrint(key.GetOuterType()))
		}

		if key.GetKey().Cmp(recvReKeys[i]) != 0 {
			t.Errorf("Key value expected %s, got %s",
				recvReKeys[i].Text(10),
				key.GetKey().Text(10))
		}
	}
}
