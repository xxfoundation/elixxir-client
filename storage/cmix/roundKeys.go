package cmix

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

type RoundKeys []*cyclic.Int

// Encrypts the given message for CMIX
// Panics if the passed message is not sized correctly for the group
func (rk RoundKeys) Encrypt(grp *cyclic.Group, msg format.Message,
	salt []byte) (format.Message, [][]byte) {

	if msg.GetPrimeByteLen() != grp.GetP().ByteLen() {
		jww.FATAL.Panicf("Cannot encrypt message whose size does not " +
			"align with the size of the prime")
	}

	ecrMsg := cmix.ClientEncrypt(grp, msg, salt, rk)

	h, err := hash.NewCMixHash()
	if err != nil {
		globals.Log.ERROR.Printf("Cound not get hash for KMAC generation: %+v", h)
	}

	KMAC := cmix.GenerateKMACs(salt, rk, h)

	return ecrMsg, KMAC
}
