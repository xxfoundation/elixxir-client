package userRegistry

import (
	"encoding/base32"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
)

const RegCodeLen = 5

func RegistrationCode(id *id.ID) string {
	return base32.StdEncoding.EncodeToString(userHash(id))
}

func userHash(id *id.ID) []byte {
	h, _ := blake2b.New256(nil)
	h.Write(id.Marshal())
	huid := h.Sum(nil)
	huid = huid[len(huid)-RegCodeLen:]
	return huid
}
