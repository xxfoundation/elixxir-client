package key

import (
	"bytes"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestKey_EncryptDecrypt_Key(t *testing.T) {

	const numTests = 100

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	prng := rand.New(rand.NewSource(42))

	for i := 0; i < numTests; i++ {
		//generate the baseKey and session
		privateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
		publicKey := dh.GeneratePublicKey(privateKey, grp)
		baseKey := dh.GenerateSessionKey(privateKey, publicKey, grp)

		s := &Session{
			baseKey: baseKey,
		}

		//create the keys
		k := newKey(s, parse.E2E, prng.Uint32())

		//make the message to be encrypted
		msg := format.NewMessage()

		//set the contents
		contents := make([]byte, format.ContentsLen-format.PadMinLen)
		prng.Read(contents)
		msg.Contents.SetRightAligned(contents)

		//set the timestamp
		now := time.Now()
		nowBytes, _ := now.MarshalBinary()
		extendedNowBytes := append(nowBytes, 0)
		msg.SetTimestamp(extendedNowBytes)

		//Encrypt
		ecrMsg := k.Encrypt(*msg)

		if !reflect.DeepEqual(k.Fingerprint(), ecrMsg.GetKeyFP()) {
			t.Errorf("Fingerprint in the ecrypted payload is wrong: "+
				"Expected: %+v, Recieved: %+v", k.Fingerprint(), ecrMsg.GetKeyFP())
		}

		//Decrypt
		resultMsg, _ := k.Decrypt(ecrMsg)

		if !bytes.Equal(resultMsg.Contents.Get(), msg.Contents.Get()) {
			t.Errorf("contents in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.Contents.Get(), resultMsg.Contents.Get())
		}

		if !bytes.Equal(resultMsg.GetTimestamp(), msg.GetTimestamp()) {
			t.Errorf("timestamp in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.GetTimestamp(), resultMsg.GetTimestamp())
		}
	}

}

func TestKey_EncryptDecrypt_ReKey(t *testing.T) {

	const numTests = 100

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	prng := rand.New(rand.NewSource(42))

	for i := 0; i < numTests; i++ {
		//generate the baseKey and session
		privateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
		publicKey := dh.GeneratePublicKey(privateKey, grp)
		baseKey := dh.GenerateSessionKey(privateKey, publicKey, grp)

		s := &Session{
			baseKey: baseKey,
		}

		//create the keys
		k := newKey(s, parse.Rekey, prng.Uint32())

		//make the message to be encrypted
		msg := format.NewMessage()

		//set the contents
		contents := make([]byte, format.ContentsLen-format.PadMinLen)
		prng.Read(contents)
		msg.Contents.SetRightAligned(contents)

		//set the timestamp
		now := time.Now()
		nowBytes, _ := now.MarshalBinary()
		extendedNowBytes := append(nowBytes, 0)
		msg.SetTimestamp(extendedNowBytes)

		//Encrypt
		ecrMsg := k.Encrypt(*msg)

		if !reflect.DeepEqual(k.Fingerprint(), ecrMsg.GetKeyFP()) {
			t.Errorf("Fingerprint in the ecrypted payload is wrong: "+
				"Expected: %+v, Recieved: %+v", k.Fingerprint(), ecrMsg.GetKeyFP())
		}

		//Decrypt
		resultMsg, _ := k.Decrypt(ecrMsg)

		if !bytes.Equal(resultMsg.Contents.Get(), msg.Contents.Get()) {
			t.Errorf("contents in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.Contents.Get(), resultMsg.Contents.Get())
		}

		if !bytes.Equal(resultMsg.GetTimestamp(), msg.GetTimestamp()) {
			t.Errorf("timestamp in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.GetTimestamp(), resultMsg.GetTimestamp())
		}
	}

}

func TestKey_EncryptDecrypt_Key_Unsafe(t *testing.T) {

	const numTests = 100

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	prng := rand.New(rand.NewSource(42))

	for i := 0; i < numTests; i++ {
		//generate the baseKey and session
		privateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
		publicKey := dh.GeneratePublicKey(privateKey, grp)
		baseKey := dh.GenerateSessionKey(privateKey, publicKey, grp)

		s := &Session{
			baseKey: baseKey,
		}

		//create the keys
		k := newKey(s, parse.E2E, 1)

		//make the message to be encrypted
		msg := format.NewMessage()

		//set the contents
		contents := make([]byte, format.ContentsLen)
		prng.Read(contents)
		msg.Contents.Set(contents)

		//set the timestamp
		now := time.Now()
		nowBytes, _ := now.MarshalBinary()
		extendedNowBytes := append(nowBytes, 0)
		msg.SetTimestamp(extendedNowBytes)

		//Encrypt
		ecrMsg := k.EncryptUnsafe(*msg)

		if !reflect.DeepEqual(k.Fingerprint(), ecrMsg.GetKeyFP()) {
			t.Errorf("Fingerprint in the ecrypted payload is wrong: "+
				"Expected: %+v, Recieved: %+v", k.Fingerprint(), ecrMsg.GetKeyFP())
		}

		//Decrypt
		resultMsg, _ := k.DecryptUnsafe(ecrMsg)

		if !bytes.Equal(resultMsg.Contents.Get(), msg.Contents.Get()) {
			t.Errorf("contents in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.Contents.Get(), resultMsg.Contents.Get())
		}

		if !bytes.Equal(resultMsg.GetTimestamp(), msg.GetTimestamp()) {
			t.Errorf("timestamp in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.GetTimestamp(), resultMsg.GetTimestamp())
		}
	}
}

func TestKey_EncryptDecrypt_ReKey_Unsafe(t *testing.T) {

	const numTests = 100

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	prng := rand.New(rand.NewSource(42))

	for i := 0; i < numTests; i++ {
		//generate the baseKey and session
		privateKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
		publicKey := dh.GeneratePublicKey(privateKey, grp)
		baseKey := dh.GenerateSessionKey(privateKey, publicKey, grp)

		s := &Session{
			baseKey: baseKey,
		}

		//create the keys
		k := newKey(s, parse.E2E, 1)

		//make the message to be encrypted
		msg := format.NewMessage()

		//set the contents
		contents := make([]byte, format.ContentsLen)
		prng.Read(contents)
		msg.Contents.Set(contents)

		//set the timestamp
		now := time.Now()
		nowBytes, _ := now.MarshalBinary()
		extendedNowBytes := append(nowBytes, 0)
		msg.SetTimestamp(extendedNowBytes)

		//Encrypt
		ecrMsg := k.EncryptUnsafe(*msg)

		if !reflect.DeepEqual(k.Fingerprint(), ecrMsg.GetKeyFP()) {
			t.Errorf("Fingerprint in the ecrypted payload is wrong: "+
				"Expected: %+v, Recieved: %+v", k.Fingerprint(), ecrMsg.GetKeyFP())
		}

		//Decrypt
		resultMsg, _ := k.DecryptUnsafe(ecrMsg)

		if !bytes.Equal(resultMsg.Contents.Get(), msg.Contents.Get()) {
			t.Errorf("contents in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.Contents.Get(), resultMsg.Contents.Get())
		}

		if !bytes.Equal(resultMsg.GetTimestamp(), msg.GetTimestamp()) {
			t.Errorf("timestamp in the decrypted payload does not match: "+
				"Expected: %v, Recieved: %v", msg.GetTimestamp(), resultMsg.GetTimestamp())
		}
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
