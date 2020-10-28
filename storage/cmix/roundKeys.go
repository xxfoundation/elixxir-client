package cmix

import (
	"crypto/sha256"
	"crypto/sha512"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
	"golang.org/x/crypto/blake2b"
)

type RoundKeys struct {
	keys []*key
	g    *cyclic.Group
}

// Encrypts the given message for CMIX
// Panics if the passed message is not sized correctly for the group
func (rk *RoundKeys) Encrypt(msg format.Message,
	salt []byte) (format.Message, [][]byte) {

	if msg.GetPrimeByteLen() != rk.g.GetP().ByteLen() {
		jww.FATAL.Panicf("Cannot encrypt message whose size does not " +
			"align with the size of the prime")
	}

	keys := make([]*cyclic.Int, len(rk.keys))

	for i, k := range rk.keys {
		jww.INFO.Printf("CMIXKEY: num: %d, key: %s", i, k.Get().Text(16))
		keys[i] = k.Get()
	}

	ecrMsg := ClientEncrypt(rk.g, msg, salt, keys)

	h, err := hash.NewCMixHash()
	if err != nil {
		jww.FATAL.Panicf("Cound not get hash for KMAC generation: %+v", h)
	}

	KMAC := cmix.GenerateKMACs(salt, keys, h)

	return ecrMsg, KMAC
}

func (rk *RoundKeys) MakeClientGatewayKey(salt, digest []byte) []byte {
	clientGatewayKey := cmix.GenerateClientGatewayKey(rk.keys[0].k)
	h, _ := hash.NewCMixHash()
	h.Write(clientGatewayKey)
	h.Write(salt)
	hashed := h.Sum(nil)
	h.Reset()
	h.Write(hashed)
	h.Write(digest)
	return h.Sum(nil)
}

func ClientEncrypt(grp *cyclic.Group, msg format.Message,
	salt []byte, baseKeys []*cyclic.Int) format.Message {

	// Get the salt for associated data
	hash, err := blake2b.New256(nil)
	if err != nil {
		panic("E2E Client Encrypt could not get blake2b Hash")
	}
	hash.Reset()
	hash.Write(salt)
	salt2 := hash.Sum(nil)

	jww.INFO.Printf("SALT_A: %v", salt)
	jww.INFO.Printf("SALT_B: %v", salt2)

	// Get encryption keys
	keyEcrA := ClientKeyGen(grp, salt, baseKeys)
	jww.INFO.Printf("Key A: %s", keyEcrA.Text(16))
	keyEcrB := ClientKeyGen(grp, salt2, baseKeys)
	jww.INFO.Printf("Key B: %s", keyEcrA.Text(16))

	// Get message payloads as cyclic integers
	payloadA := grp.NewIntFromBytes(msg.GetPayloadA())
	payloadB := grp.NewIntFromBytes(msg.GetPayloadB())

	jww.INFO.Printf("Payload A: %s", payloadA.Text(16))
	jww.INFO.Printf("Payload B: %s", payloadB.Text(16))

	// Encrypt payload A with the key
	EcrPayloadA := grp.Mul(keyEcrA, payloadA, grp.NewInt(1))
	jww.INFO.Printf("Encrypted Payload A: %s", EcrPayloadA.Text(16))
	EcrPayloadB := grp.Mul(keyEcrB, payloadB, grp.NewInt(1))
	jww.INFO.Printf("Encrypted Payload B: %s", EcrPayloadB.Text(16))

	primeLen := grp.GetP().ByteLen()

	// Create the encrypted message
	encryptedMsg := format.NewMessage(primeLen)

	encryptedMsg.SetPayloadA(EcrPayloadA.LeftpadBytes(uint64(primeLen)))
	encryptedMsg.SetPayloadB(EcrPayloadB.LeftpadBytes(uint64(primeLen)))

	jww.INFO.Printf("Encrypted message: %v", encryptedMsg.Marshal())

	return encryptedMsg

}

func ClientKeyGen(grp *cyclic.Group, salt []byte, baseKeys []*cyclic.Int) *cyclic.Int {
	output := grp.NewInt(1)
	tmpKey := grp.NewInt(1)

	// Multiply all the generated keys together as they are generated.
	for i, baseKey := range baseKeys {
		jww.INFO.Printf("Input to Gen Key: num: %v, baseKey: %s", i, baseKey.Text(16))
		keyGen(grp, salt, baseKey, tmpKey)
		jww.INFO.Printf("Gen Key: num: %v, key: %s", i, tmpKey.Text(16))
		grp.Mul(tmpKey, output, output)
		jww.INFO.Printf("Partial full Key: num: %v, key: %s", i, output.Text(16))
	}

	jww.INFO.Printf("final full Key: %s", output.Text(16))

	grp.Inverse(output, output)

	jww.INFO.Printf("inverted Key: %s", output.Text(16))

	return output
}

// keyGen combines the salt with the baseKey to generate a new key inside the group.
func keyGen(grp *cyclic.Group, salt []byte, baseKey, output *cyclic.Int) *cyclic.Int {
	h1, _ := hash.NewCMixHash()
	h2 := sha256.New()

	a := baseKey.Bytes()

	// Blake2b Hash of the result of previous stage (base key + salt)
	h1.Reset()
	h1.Write(a)
	h1.Write(salt)
	x := h1.Sum(nil)
	jww.INFO.Printf("keygen x: %v", x)

	// Different Hash (SHA256) of the previous result to add entropy
	h2.Reset()
	h2.Write(x)
	y := h2.Sum(nil)
	jww.INFO.Printf("keygen y: %v", x)

	// Expand Key using SHA512
	k := hash.ExpandKey(sha512.New(), grp, y, output)
	jww.INFO.Printf("keygen expandedKey: %s", k.Text(16))
	return k
}
