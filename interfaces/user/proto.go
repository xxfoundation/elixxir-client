package user

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type Proto struct {
	//General Identity
	TransmissionID   *id.ID
	TransmissionSalt []byte
	TransmissionRSA  *rsa.PrivateKey
	ReceptionID      *id.ID
	ReceptionSalt    []byte
	ReceptionRSA     *rsa.PrivateKey
	Precanned        bool
	// Timestamp in which user has registered with the network
	RegistrationTimestamp int64

	RegCode string

	TransmissionRegValidationSig []byte
	ReceptionRegValidationSig    []byte

	//e2e Identity
	E2eDhPrivateKey *cyclic.Int
	E2eDhPublicKey  *cyclic.Int
}
