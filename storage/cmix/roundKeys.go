package cmix

import (
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

	// Get encryption keys
	keyEcrA := cmix.ClientKeyGen(grp, salt, baseKeys)
	keyEcrB := cmix.ClientKeyGen(grp, salt2, baseKeys)

	// Get message payloads as cyclic integers
	payloadA := grp.NewIntFromBytes(msg.GetPayloadA())
	payloadB := grp.NewIntFromBytes(msg.GetPayloadB())

	// Encrypt payload A with the key
	EcrPayloadA := grp.Mul(keyEcrA, payloadA, grp.NewInt(1))
	EcrPayloadB := grp.Mul(keyEcrB, payloadB, grp.NewInt(1))

	primeLen := grp.GetP().ByteLen()

	// Create the encrypted message
	encryptedMsg := format.NewMessage(primeLen)

	encryptedMsg.SetPayloadA(EcrPayloadA.LeftpadBytes(uint64(primeLen)))
	encryptedMsg.SetPayloadB(EcrPayloadB.LeftpadBytes(uint64(primeLen)))

	return encryptedMsg

}
