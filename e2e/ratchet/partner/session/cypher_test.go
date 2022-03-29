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
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/netTime"
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
	expectedKey := &Cypher{
		session: getSession(t),
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
	k := newKey(getSession(t), rand.Uint32())

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
	k := newKey(getSession(t), rand.Uint32())

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
		ecrMsg := k.Encrypt(msg)

		if !reflect.DeepEqual(k.Fingerprint(), ecrMsg.GetKeyFP()) {
			t.Errorf("Fingerprint in the ecrypted payload is wrong: "+
				"Expected: %+v, Recieved: %+v", k.Fingerprint(), ecrMsg.GetKeyFP())
		}

		// Decrypt
		resultMsg, _ := k.Decrypt(ecrMsg)

		if !bytes.Equal(resultMsg.GetContents(), msg.GetContents()) {
			t.Errorf("contents in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.GetContents(), resultMsg.GetContents())
		}
	}
}

// Happy path of Key.denoteUse()
func TestKey_denoteUse(t *testing.T) {
	keyNum := uint32(rand.Int31n(31))

	k := newKey(getSession(t), keyNum)

	k.denoteUse()

	if !k.session.keyState.Used(keyNum) {
		t.Errorf("denoteUse() did not use the key")
	}
}

// Happy path of generateKey().
func TestKey_generateKey(t *testing.T) {
	k := newKey(getSession(t), rand.Uint32())

	// Generate test CryptoType values and expected keys
	expectedKey := e2e.DeriveKey(k.session.baseKey, k.keyNum)
	testKey := k.generateKey()

	if !reflect.DeepEqual(expectedKey, testKey) {
		t.Errorf("generateKey() did not produce the expected e2e key."+
			"\n\texpected: %v\n\treceived: %v",
			expectedKey, testKey)
	}

}

func getGroup() *cyclic.Group {
	e2eGrp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))

	return e2eGrp

}

func getSession(t *testing.T) *Session {
	if t == nil {
		panic("getSession is a testing function and should be called from a test")
	}
	grp := getGroup()
	rng := csprng.NewSystemRNG()

	// finalizeKeyNegotation the baseKey and session
	privateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	publicKey := dh.GeneratePublicKey(privateKey, grp)

	// SIDH keys
	pubA := sidh.NewPublicKey(sidh.Fp434, sidh.KeyVariantSidhA)
	privA := sidh.NewPrivateKey(sidh.Fp434, sidh.KeyVariantSidhA)
	privA.Generate(rng)
	privA.GeneratePublicKey(pubA)
	pubB := sidh.NewPublicKey(sidh.Fp434, sidh.KeyVariantSidhB)
	privB := sidh.NewPrivateKey(sidh.Fp434, sidh.KeyVariantSidhB)
	privB.Generate(rng)
	privB.GeneratePublicKey(pubB)

	baseKey := GenerateE2ESessionBaseKey(privateKey, publicKey, grp, privA,
		pubB)

	keyState, err := utility.NewStateVector(versioned.NewKV(make(ekv.Memstore)), "keyState", rand.Uint32())
	if err != nil {
		panic(err)
	}

	return &Session{
		baseKey:  baseKey,
		keyState: keyState,
	}
}

func getFingerprint() *format.Fingerprint {
	rand.Seed(netTime.Now().UnixNano())
	fp := format.Fingerprint{}
	rand.Read(fp[:])

	return &fp
}
