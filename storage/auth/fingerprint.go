package auth

import "gitlab.com/elixxir/crypto/cyclic"

type FingerprintType uint

const (
	General  FingerprintType = 1
	Specific FingerprintType = 2
)

type fingerprint struct {
	Type FingerprintType
	//only populated if ti is general
	PrivKey *cyclic.Int
	//only populated if it is specific
	Request *request
}
