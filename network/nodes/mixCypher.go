///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package nodes

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
)

type MixCypher interface {
	Encrypt(msg format.Message, salt []byte, roundID id.Round) (format.Message, [][]byte)
	MakeClientGatewayAuthMAC(salt, digest []byte) []byte
}

type mixCypher struct {
	keys []*key
	g    *cyclic.Group
}

// Encrypt encrypts the given message for CMIX.
// Panics if the passed message is not sized correctly for the group.
func (mc *mixCypher) Encrypt(msg format.Message,
	salt []byte, roundID id.Round) (format.Message, [][]byte) {

	if msg.GetPrimeByteLen() != mc.g.GetP().ByteLen() {
		jww.FATAL.Panicf("Cannot encrypt message whose size does not " +
			"align with the size of the prime")
	}

	keys := make([]*cyclic.Int, len(mc.keys))

	for i, k := range mc.keys {
		jww.TRACE.Printf("CMIXKEY: num: %d, key: %s", i, k.get().Text(16))
		keys[i] = k.get()
	}

	ecrMsg := clientEncrypt(mc.g, msg, salt, roundID, keys)

	h, err := hash.NewCMixHash()
	if err != nil {
		jww.FATAL.Panicf("Could not get hash for KMAC generation: %+v", h)
	}

	KMAC := cmix.GenerateKMACs(salt, keys, roundID, h)

	return ecrMsg, KMAC
}

func (mc *mixCypher) MakeClientGatewayAuthMAC(salt, digest []byte) []byte {
	clientGatewayKey := cmix.GenerateClientGatewayKey(mc.keys[0].k)
	h, _ := hash.NewCMixHash()
	h.Write(clientGatewayKey)
	h.Write(salt)
	hashed := h.Sum(nil)
	h.Reset()
	h.Write(hashed)
	h.Write(digest)
	return h.Sum(nil)
}

func clientEncrypt(grp *cyclic.Group, msg format.Message,
	salt []byte, roundID id.Round, baseKeys []*cyclic.Int) format.Message {

	// get the salt for associated data
	h, err := blake2b.New256(nil)
	if err != nil {
		panic("E2E Client Encrypt could not get blake2b Hash")
	}
	h.Reset()
	h.Write(salt)
	salt2 := h.Sum(nil)

	// get encryption keys
	keyEcrA := cmix.ClientKeyGen(grp, salt, roundID, baseKeys)
	keyEcrB := cmix.ClientKeyGen(grp, salt2, roundID, baseKeys)

	// get message payloads as cyclic integers
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
