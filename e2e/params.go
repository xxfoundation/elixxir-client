package e2e

import "gitlab.com/elixxir/crypto/e2e"

// DEFAULT KEY GENERATION PARAMETERS
// Hardcoded limits for keys
// With 16 receiving states we can hold
// 16*64=1024 dirty bits for receiving keys
// With that limit, and setting maxKeys to 800,
// we need a Threshold of 224, and a scalar
// smaller than 1.28 to ensure we never generate
// more than 1024 keys
// With 1 receiving states for ReKeys we can hold
// 64 Rekeys
const (
	minKeys   uint16  = 500
	maxKeys   uint16  = 800
	ttlScalar float64 = 1.2 // generate 20% extra keys
	threshold uint16  = 224
	numReKeys uint16  = 64
)

type SessionParams struct {
	MinKeys   uint16
	MaxKeys   uint16
	NumRekeys uint16
	e2e.TTLParams
}

func GetDefaultSessionParams() SessionParams {
	return SessionParams{
		MinKeys:   minKeys,
		MaxKeys:   maxKeys,
		NumRekeys: numReKeys,
		TTLParams: e2e.TTLParams{
			TTLScalar:  ttlScalar,
			MinNumKeys: threshold,
		},
	}
}