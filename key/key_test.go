package key

import (
	"bytes"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Happy path of newKey().
func Test_newKey(t *testing.T) {
	expectedKey := &Key{
		session: getSession(t),
		outer:   parse.CryptoType(rand.Int31n(int32(parse.E2E))),
		keyNum:  rand.Uint32(),
	}

	testKey := newKey(expectedKey.session, expectedKey.outer, expectedKey.keyNum)

	if !reflect.DeepEqual(expectedKey, testKey) {
		t.Errorf("newKey() did not produce the expected Key."+
			"\n\texpected: %v\n\treceived: %v",
			expectedKey, testKey)
	}
}

// Happy path of Key.GetSession().
func TestKey_GetSession(t *testing.T) {
	k := newKey(getSession(t), parse.CryptoType(rand.Int31n(int32(parse.E2E))),
		rand.Uint32())

	testSession := k.GetSession()

	if !reflect.DeepEqual(k.session, testSession) {

		if !reflect.DeepEqual(k.session, testSession) {
			t.Errorf("GetSession() did not produce the expected Session."+
				"\n\texpected: %v\n\treceived: %v",
				k.session, testSession)
		}
	}
}

// Happy path of Key.GetCryptoType().
func TestKey_GetCryptoType(t *testing.T) {
	k := newKey(getSession(t), parse.CryptoType(rand.Int31n(int32(parse.E2E))),
		rand.Uint32())

	testCryptoType := k.GetCryptoType()

	if !reflect.DeepEqual(k.outer, testCryptoType) {

		if !reflect.DeepEqual(k.outer, testCryptoType) {
			t.Errorf("GetCryptoType() did not produce the expected CryptoType."+
				"\n\texpected: %v\n\treceived: %v",
				k.outer, testCryptoType)
		}
	}
}

// Happy path of Key.Fingerprint().
func TestKey_Fingerprint(t *testing.T) {
	k := newKey(getSession(t), 0, rand.Uint32())

	// Generate test and expected fingerprints
	testFingerprint := getFingerprint()
	testData := []struct {
		outer      parse.CryptoType
		testFP     *format.Fingerprint
		expectedFP format.Fingerprint
	}{
		{0, testFingerprint, *testFingerprint},
		{parse.E2E, nil, e2e.DeriveKeyFingerprint(k.session.baseKey, k.keyNum)},
		{parse.Rekey, nil, e2e.DeriveReKeyFingerprint(k.session.baseKey, k.keyNum)},
	}

	// Test cases
	for _, data := range testData {
		k.outer = data.outer
		k.fp = data.testFP
		testFP := k.Fingerprint()

		if !reflect.DeepEqual(data.expectedFP, testFP) {
			t.Errorf("Fingerprint() did not produce the expected Fingerprint."+
				"\n\texpected: %v\n\treceived: %v",
				data.expectedFP, testFP)
		}
	}
}

// Tests that Key.Fingerprint() panics when Key.outer is invalid.
func TestKey_Fingerprint_Panic(t *testing.T) {

	k := &Key{getSession(t), nil, parse.Unencrypted, rand.Uint32()}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Fingerprint() did not panic when key.outer (value of %s) "+
				"is invalid.", k.outer)
		}
	}()
	_ = k.Fingerprint()
}

func TestKey_EncryptDecrypt(t *testing.T) {

	const numTests = 100

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	prng := rand.New(rand.NewSource(42))

	for i := 0; i < numTests; i++ {
		// generate the baseKey and session
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

		// set the timestamp
		now := time.Now()
		nowBytes, _ := now.MarshalBinary()
		extendedNowBytes := append(nowBytes, 0)
		msg.SetTimestamp(extendedNowBytes)

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

		if !bytes.Equal(resultMsg.GetTimestamp(), msg.GetTimestamp()) {
			t.Errorf("timestamp in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.GetTimestamp(), resultMsg.GetTimestamp())
		}
	}
}


// Happy path of Key.denoteUse().
func TestKey_denoteUse(t *testing.T) {
	k := newKey(getSession(t), 0, uint32(rand.Int31n(31)))

	// Generate test CryptoType values
	testData := []parse.CryptoType{parse.E2E, parse.Rekey}

	// Test cases
	for _, outer := range testData {
		k.outer = outer
		err := k.denoteUse()
		if err != nil {
			t.Errorf("denoteUse() produced an unexpected error."+
				"\n\texpected: %v\n\treceived: %v", nil, err)
		}
	}
}

// Tests that Key.denoteUse() panics for invalid values of Key.outer.
func TestKey_denoteUse_Panic(t *testing.T) {
	k := newKey(getSession(t), parse.Unencrypted, uint32(rand.Int31n(31)))

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("denoteUse() did not panic when key.outer (value of %s) "+
				"is invalid.", k.outer)
		}
	}()

	_ = k.denoteUse()
}

// Happy path of generateKey().
func TestKey_generateKey(t *testing.T) {
	k := newKey(getSession(t), 0, rand.Uint32())

	// Generate test CryptoType values and expected keys
	testData := []struct {
		outer       parse.CryptoType
		expectedKey e2e.Key
	}{
		{parse.E2E, e2e.DeriveKey(k.session.baseKey, k.keyNum)},
		{parse.Rekey, e2e.DeriveReKey(k.session.baseKey, k.keyNum)},
	}

	// Test cases
	for _, data := range testData {
		k.outer = data.outer
		testKey := k.generateKey()

		if !reflect.DeepEqual(data.expectedKey, testKey) {
			t.Errorf("generateKey() did not produce the expected e2e key."+
				"\n\texpected: %v\n\treceived: %v",
				data.expectedKey, testKey)
		}
	}
}

// Tests that generateKey() panics for invalid values of Key.outer.
func TestKey_generateKey_Panic(t *testing.T) {
	k := newKey(getSession(t), parse.Unencrypted, rand.Uint32())

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("generateKey() did not panic when key.outer (value of %s) "+
				"is invalid.", k.outer)
		}
	}()

	_ = k.generateKey()
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
	grp := getGroup()
	rng := csprng.NewSystemRNG()

	// generate the baseKey and session
	privateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	publicKey := dh.GeneratePublicKey(privateKey, grp)
	baseKey := dh.GenerateSessionKey(privateKey, publicKey, grp)

	fps := newFingerprints()
	ctx := &context{
		fa:  &fps,
		grp: grp,
		kv:  storage.InitMem(t),
	}

	keyState := newStateVector(ctx, "keyState", rand.Uint32())
	reKeyState := newStateVector(ctx, "reKeyState", rand.Uint32())

	return &Session{
		manager: &Manager{
			ctx: ctx,
		},
		baseKey:    baseKey,
		keyState:   keyState,
		reKeyState: reKeyState,
	}
}

func getFingerprint() *format.Fingerprint {
	rand.Seed(time.Now().UnixNano())
	fp := make([]byte, format.KeyFPLen)
	rand.Read(fp)

	return format.NewFingerprint(fp)
}
