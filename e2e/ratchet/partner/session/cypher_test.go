///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package session

import (
	"bytes"
	"github.com/cloudflare/circl/dh/sidh"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"math/rand"
	"reflect"
	"testing"
)

// TestGenerateE2ESessionBaseKey smoke tests the GenerateE2ESessionBaseKey
// function to ensure that it produces the correct key on both sides of the
// connection.
func TestGenerateE2ESessionBaseKey(t *testing.T) {
	rng := fastRNG.NewStreamGenerator(1, 3, csprng.NewSystemRNG)
	myRng := rng.GetStream()

	// DH Keys
	grp := getGroup()
	dhPrivateKeyA := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp,
		myRng)
	dhPublicKeyA := dh.GeneratePublicKey(dhPrivateKeyA, grp)
	dhPrivateKeyB := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp,
		myRng)
	dhPublicKeyB := dh.GeneratePublicKey(dhPrivateKeyB, grp)

	// SIDH keys
	pubA := sidh.NewPublicKey(sidh.Fp434, sidh.KeyVariantSidhA)
	privA := sidh.NewPrivateKey(sidh.Fp434, sidh.KeyVariantSidhA)
	privA.Generate(myRng)
	privA.GeneratePublicKey(pubA)
	pubB := sidh.NewPublicKey(sidh.Fp434, sidh.KeyVariantSidhB)
	privB := sidh.NewPrivateKey(sidh.Fp434, sidh.KeyVariantSidhB)
	privB.Generate(myRng)
	privB.GeneratePublicKey(pubB)

	myRng.Close()

	baseKey1 := GenerateE2ESessionBaseKey(dhPrivateKeyA, dhPublicKeyB,
		grp, privA, pubB)
	baseKey2 := GenerateE2ESessionBaseKey(dhPrivateKeyB, dhPublicKeyA,
		grp, privB, pubA)

	if !reflect.DeepEqual(baseKey1, baseKey2) {
		t.Errorf("Cannot produce the same session key:\n%v\n%v",
			baseKey1, baseKey2)
	}

}

// Happy path of newKey().
func Test_newKey(t *testing.T) {
	s, _ := makeTestSession()

	expectedKey := &Cypher{
		session: s,
		keyNum:  rand.Uint32(),
	}

	testKey := newKey(expectedKey.session, expectedKey.keyNum)

	if !reflect.DeepEqual(expectedKey, testKey) {
		t.Errorf("newKey() did not produce the expected Key."+
			"\n\texpected: %v\n\treceived: %v",
			expectedKey, testKey)
	}
}

// Happy path of Key.GetSession().
func TestKey_GetSession(t *testing.T) {
	s, _ := makeTestSession()

	k := newKey(s, rand.Uint32())

	testSession := k.GetSession()

	if !reflect.DeepEqual(k.session, testSession) {

		if !reflect.DeepEqual(k.session, testSession) {
			t.Errorf("GetSession() did not produce the expected Session."+
				"\n\texpected: %v\n\treceived: %v",
				k.session, testSession)
		}
	}
}

// Happy path of Key.Fingerprint().
func TestKey_Fingerprint(t *testing.T) {
	s, _ := makeTestSession()

	k := newKey(s, rand.Uint32())

	// Generate test and expected fingerprints
	testFingerprint := getFingerprint()
	testData := []struct {
		testFP     *format.Fingerprint
		expectedFP format.Fingerprint
	}{
		{testFingerprint, *testFingerprint},
		{nil, e2e.DeriveKeyFingerprint(k.session.baseKey, k.keyNum)},
	}

	// Test cases
	for _, data := range testData {
		k.fp = data.testFP
		testFP := k.Fingerprint()

		if !reflect.DeepEqual(data.expectedFP, testFP) {
			t.Errorf("Fingerprint() did not produce the expected Fingerprint."+
				"\n\texpected: %v\n\treceived: %v",
				data.expectedFP, testFP)
		}
	}
}

func TestKey_EncryptDecrypt(t *testing.T) {

	const numTests = 100

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	prng := rand.New(rand.NewSource(42))

	for i := 0; i < numTests; i++ {
		// finalizeKeyNegotation the baseKey and session
		privateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
		publicKey := dh.GeneratePublicKey(privateKey, grp)
		baseKey := dh.GenerateSessionKey(privateKey, publicKey, grp)

		s := &Session{
			baseKey: baseKey,
		}

		//create the keys
		k := newKey(s, prng.Uint32())

		//make the message to be encrypted
		msg := format.NewMessage(grp.GetP().ByteLen())

		//set the contents
		contents := make([]byte, msg.ContentsSize())
		prng.Read(contents)
		msg.SetContents(contents)

		// Encrypt
		contentsEnc, mac := k.Encrypt(msg.GetContents())

		//make the encrypted message
		ecrMsg := format.NewMessage(grp.GetP().ByteLen())
		ecrMsg.SetKeyFP(k.Fingerprint())
		ecrMsg.SetContents(contentsEnc)
		ecrMsg.SetMac(mac)

		// Decrypt
		contentsDecr, err := k.Decrypt(ecrMsg)
		if err != nil {
			t.Fatalf("Decrypt error: %v", err)
		}

		if !bytes.Equal(contentsDecr, msg.GetContents()) {
			t.Errorf("contents in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.GetContents(), contentsDecr)
		}
	}
}

// Happy path of Key.Use()
func TestKey_denoteUse(t *testing.T) {
	s, _ := makeTestSession()

	keyNum := uint32(rand.Int31n(31))

	k := newKey(s, keyNum)

	k.Use()

	if !k.session.keyState.Used(keyNum) {
		t.Errorf("Use() did not use the key")
	}
}

// Happy path of generateKey().
func TestKey_generateKey(t *testing.T) {
	s, _ := makeTestSession()

	k := newKey(s, rand.Uint32())

	// Generate test CryptoType values and expected keys
	expectedKey := e2e.DeriveKey(k.session.baseKey, k.keyNum)
	testKey := k.generateKey()

	if !reflect.DeepEqual(expectedKey, testKey) {
		t.Errorf("generateKey() did not produce the expected e2e key."+
			"\n\texpected: %v\n\treceived: %v",
			expectedKey, testKey)
	}

}
