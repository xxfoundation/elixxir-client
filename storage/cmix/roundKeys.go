package cmix

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
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
		keys[i] = k.Get()
	}

	ecrMsg := cmix.ClientEncrypt(rk.g, msg, salt, keys)

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
